package commands

import (
	"reflect"
	"strings"
	"testing"

	corecfg "github.com/cushycush/store-core/config"
	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
)

// fakeManager lets us drive computePlan/install/doctor without shelling out.
type fakeManager struct {
	name      string
	available bool
	installed []string
	// install history for assertions
	installCalls [][]string
	installError error
}

func (f *fakeManager) Name() string                 { return f.name }
func (f *fakeManager) Available() bool              { return f.available }
func (f *fakeManager) Installed() ([]string, error) { return f.installed, nil }
func (f *fakeManager) BootstrapHint() string        { return "fake" }
func (f *fakeManager) Install(pkgs []string) error {
	f.installCalls = append(f.installCalls, append([]string(nil), pkgs...))
	return f.installError
}

// registerFakes adds the given fakes to the real managers registry and
// unregisters them on cleanup. All names must be unique to avoid stomping on
// real managers or each other.
func registerFakes(t *testing.T, fakes ...*fakeManager) {
	t.Helper()
	for _, f := range fakes {
		managers.Register(f)
		name := f.Name()
		t.Cleanup(func() { managers.Unregister(name) })
	}
}

func testCtx(cfg *config.File, info platform.Info) *Context {
	return &Context{Cfg: cfg, Info: info, Stdout: discard{}, Stderr: discard{}}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func TestDedupe(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{nil, nil},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "b", "a"}, []string{"a", "b"}},
		{[]string{"a", "a", "a"}, []string{"a"}},
		{[]string{"z", "y", "x", "y"}, []string{"z", "y", "x"}}, // order preserved
	}
	for _, tt := range tests {
		if got := dedupe(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("dedupe(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		desired, installed, want []string
	}{
		{nil, nil, nil},
		{[]string{"a", "b"}, nil, []string{"a", "b"}},
		{[]string{"a", "b"}, []string{"a", "b"}, nil},
		{[]string{"a", "b", "c"}, []string{"b"}, []string{"a", "c"}},
	}
	for _, tt := range tests {
		if got := diff(tt.desired, tt.installed); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("diff(%v, %v) = %v, want %v", tt.desired, tt.installed, got, tt.want)
		}
	}
}

func TestGroupNames(t *testing.T) {
	cfg := &config.File{Groups: []config.Group{
		{Name: "core"}, {Name: "gui"}, {Name: "work"},
	}}
	got := groupNames(cfg)
	for _, n := range []string{"core", "gui", "work"} {
		if !got[n] {
			t.Errorf("groupNames missing %q", n)
		}
	}
	if got["nonexistent"] {
		t.Error("groupNames included a name that wasn't in cfg")
	}
}

func TestSelectGroups(t *testing.T) {
	cfg := &config.File{Groups: []config.Group{{Name: "core"}, {Name: "gui"}}}
	ctx := testCtx(cfg, platform.Info{})

	t.Run("empty args means all", func(t *testing.T) {
		got, err := selectGroups(ctx, nil)
		if err != nil {
			t.Fatalf("selectGroups() error = %v", err)
		}
		if got != nil {
			t.Fatalf("selectGroups(nil) = %v, want nil map", got)
		}
	})

	t.Run("valid subset", func(t *testing.T) {
		got, err := selectGroups(ctx, []string{"core"})
		if err != nil {
			t.Fatalf("selectGroups() error = %v", err)
		}
		if !got["core"] || got["gui"] {
			t.Fatalf("selectGroups({core}) = %v, want {core:true}", got)
		}
	})

	t.Run("unknown group errors", func(t *testing.T) {
		_, err := selectGroups(ctx, []string{"core", "nope"})
		if err == nil {
			t.Fatal("selectGroups with unknown group returned nil error")
		}
	})
}

func TestComputePlanAggregatesAcrossApplicableGroups(t *testing.T) {
	// Two applicable groups both reference brew. Plan should merge + dedupe.
	brew := &fakeManager{name: "test-plan-brew", available: true, installed: []string{"git"}}
	apt := &fakeManager{name: "test-plan-apt", available: true, installed: []string{"git"}}
	registerFakes(t, brew, apt)

	cfg := &config.File{Groups: []config.Group{
		{
			Name: "core",
			Managers: map[string][]string{
				"test-plan-brew": {"git", "ripgrep"},
				"test-plan-apt":  {"git", "fd-find"},
			},
		},
		{
			Name: "extra",
			Managers: map[string][]string{
				"test-plan-brew": {"ripgrep", "bat"}, // ripgrep duplicated across groups
			},
		},
	}}

	plans, warnings, err := computePlan(testCtx(cfg, platform.Info{OS: "linux"}), nil)
	if err != nil {
		t.Fatalf("computePlan() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	got := map[string][]string{}
	for _, p := range plans {
		got[p.Manager.Name()] = p.Desired
	}
	wantBrew := []string{"git", "ripgrep", "bat"}
	wantApt := []string{"git", "fd-find"}
	if !reflect.DeepEqual(got["test-plan-brew"], wantBrew) {
		t.Errorf("brew desired = %v, want %v", got["test-plan-brew"], wantBrew)
	}
	if !reflect.DeepEqual(got["test-plan-apt"], wantApt) {
		t.Errorf("apt desired = %v, want %v", got["test-plan-apt"], wantApt)
	}

	// Manager order should be deterministic (sorted).
	var order []string
	for _, p := range plans {
		order = append(order, p.Manager.Name())
	}
	if !reflect.DeepEqual(order, []string{"test-plan-apt", "test-plan-brew"}) {
		t.Errorf("plan order = %v, want sorted [test-plan-apt test-plan-brew]", order)
	}
}

func TestComputePlanRespectsWhenFilter(t *testing.T) {
	brew := &fakeManager{name: "test-plan-when-brew", available: true}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{
		{
			Name:     "linux-only",
			Managers: map[string][]string{"test-plan-when-brew": {"pkg-linux"}},
			When:     &corecfg.WhenClause{OS: corecfg.Strings{"linux"}},
		},
		{
			Name:     "darwin-only",
			Managers: map[string][]string{"test-plan-when-brew": {"pkg-mac"}},
			When:     &corecfg.WhenClause{OS: corecfg.Strings{"darwin"}},
		},
	}}

	plans, _, _ := computePlan(testCtx(cfg, platform.Info{OS: "darwin"}), nil)
	if len(plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(plans))
	}
	if !reflect.DeepEqual(plans[0].Desired, []string{"pkg-mac"}) {
		t.Fatalf("desired = %v, want [pkg-mac]", plans[0].Desired)
	}
}

func TestComputePlanSkipsUnavailableManagersSilently(t *testing.T) {
	// A cross-platform group listing brew + apt should silently ignore the
	// unavailable one — otherwise every Linux user gets noise about brew.
	avail := &fakeManager{name: "test-plan-avail", available: true}
	unavail := &fakeManager{name: "test-plan-unavail", available: false}
	registerFakes(t, avail, unavail)

	cfg := &config.File{Groups: []config.Group{{
		Name: "core",
		Managers: map[string][]string{
			"test-plan-avail":   {"git"},
			"test-plan-unavail": {"git"},
		},
	}}}

	plans, warnings, _ := computePlan(testCtx(cfg, platform.Info{}), nil)
	if len(plans) != 1 || plans[0].Manager.Name() != "test-plan-avail" {
		t.Fatalf("plans = %+v, want only test-plan-avail", plans)
	}
	if len(warnings) != 0 {
		t.Fatalf("unavailable manager produced warnings (should be silent): %v", warnings)
	}
}

func TestComputePlanWarnsOnUnknownManagerKey(t *testing.T) {
	cfg := &config.File{Groups: []config.Group{{
		Name:     "typo",
		Managers: map[string][]string{"breww": {"git"}}, // user typo
	}}}

	_, warnings, _ := computePlan(testCtx(cfg, platform.Info{}), nil)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %d, want 1 (unknown manager)", len(warnings))
	}
	if msg := warnings[0].Error(); !strings.Contains(msg, "breww") {
		t.Fatalf("warning message = %q, want it to mention %q", msg, "breww")
	}
}

func TestComputePlanOnlyFilterHonoredWhenProvided(t *testing.T) {
	brew := &fakeManager{name: "test-only-brew", available: true}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{
		{Name: "in", Managers: map[string][]string{"test-only-brew": {"in-pkg"}}},
		{Name: "out", Managers: map[string][]string{"test-only-brew": {"out-pkg"}}},
	}}

	plans, _, _ := computePlan(testCtx(cfg, platform.Info{}), map[string]bool{"in": true})
	if len(plans) != 1 || plans[0].Manager.Name() != "test-only-brew" {
		t.Fatalf("plans = %+v", plans)
	}
	if !reflect.DeepEqual(plans[0].Desired, []string{"in-pkg"}) {
		t.Fatalf("only-filter let non-selected packages through: %v", plans[0].Desired)
	}
}

