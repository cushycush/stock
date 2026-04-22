package tui

import (
	"testing"

	corecfg "github.com/cushycush/store-core/config"
	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
	"github.com/cushycush/stock/internal/runner"
)

// fakeManager is a stand-in for a real package manager so Build can be tested
// without touching the host's brew/apt/etc.
type fakeManager struct {
	name      string
	available bool
	installed []string
}

func (f *fakeManager) Name() string                { return f.name }
func (f *fakeManager) Available() bool             { return f.available }
func (f *fakeManager) Installed() ([]string, error) { return f.installed, nil }
func (f *fakeManager) Install([]string) error      { return nil }
func (f *fakeManager) BootstrapHint() string       { return "" }

// registerFakes swaps in fakes for the duration of a test. Every call here
// must be matched by an Unregister in cleanup or subsequent tests see the
// fake instead of the real manager.
func registerFakes(t *testing.T, fakes ...*fakeManager) {
	t.Helper()
	for _, f := range fakes {
		if managers.Get(f.name) != nil {
			managers.Unregister(f.name)
		}
		managers.Register(f)
	}
	managers.Bind(runner.NewDryRun())
	t.Cleanup(func() {
		for _, f := range fakes {
			managers.Unregister(f.name)
		}
	})
}

func TestBuildAggregatesState(t *testing.T) {
	// Two fake managers: fakebrew is available with some pkgs installed,
	// fakeapt is not available on this "machine". They cover the cross-
	// platform alternative pattern the doctor intentionally ignores.
	registerFakes(t,
		&fakeManager{name: "fakebrew", available: true, installed: []string{"git", "ripgrep"}},
		&fakeManager{name: "fakeapt", available: false},
	)

	cfg := &config.File{
		Groups: []config.Group{
			{
				Name: "installed-group",
				Managers: map[string][]string{
					"fakebrew": {"git", "ripgrep"},
					"fakeapt":  {"git", "ripgrep"},
				},
			},
			{
				Name: "partial-group",
				Managers: map[string][]string{
					"fakebrew": {"git", "newthing"},
				},
			},
			{
				Name: "missing-group",
				Managers: map[string][]string{
					"fakebrew": {"newpkg1", "newpkg2"},
				},
			},
			{
				Name: "skipped-group",
				Managers: map[string][]string{
					"fakebrew": {"slack"},
				},
				When: &corecfg.WhenClause{OS: corecfg.Strings{"plan9"}},
			},
			{
				Name: "unservable-group",
				Managers: map[string][]string{
					"fakeapt": {"git"},
				},
			},
		},
	}

	rows := Build(cfg, platform.Info{OS: "linux", Arch: "amd64"})
	got := map[string]State{}
	for _, r := range rows {
		got[r.Name] = r.State
	}
	want := map[string]State{
		"installed-group":  StateInstalled,
		"partial-group":    StatePartial,
		"missing-group":    StateMissing,
		"skipped-group":    StateSkipped,
		"unservable-group": StateUnservable,
	}
	for name, expected := range want {
		if got[name] != expected {
			t.Errorf("%s: got state %v, want %v", name, got[name], expected)
		}
	}

	// Rows are sorted by name.
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	wantOrder := []string{"installed-group", "missing-group", "partial-group", "skipped-group", "unservable-group"}
	if len(names) != len(wantOrder) {
		t.Fatalf("got %d rows, want %d", len(names), len(wantOrder))
	}
	for i := range names {
		if names[i] != wantOrder[i] {
			t.Errorf("row %d: got %q, want %q", i, names[i], wantOrder[i])
		}
	}
}

func TestBuildEmptyConfig(t *testing.T) {
	if got := Build(nil, platform.Info{}); got != nil {
		t.Errorf("Build(nil) = %v, want nil", got)
	}
	if got := Build(&config.File{}, platform.Info{}); len(got) != 0 {
		t.Errorf("Build(empty) = %v, want empty", got)
	}
}
