package managers

import (
	"reflect"
	"slices"
	"testing"

	"github.com/cushycush/stock/internal/runner"
)

// Builtin manager init() calls populate defaultRegistry. These tests use
// unique names that don't collide with those, and clean up after themselves
// so `go test -count=2` still passes.

// fakeManager is a minimal Manager used by tests in this package. It does not
// embed `base`, because the registry and Bind must also work with managers
// that choose not to accept a runner.
type fakeManager struct {
	name          string
	installed     []string
	installError  error
	lastInstalled []string
}

func (f *fakeManager) Name() string                        { return f.name }
func (f *fakeManager) Available() bool                     { return true }
func (f *fakeManager) Installed() ([]string, error)        { return f.installed, nil }
func (f *fakeManager) BootstrapHint() string               { return "install fake-manager" }
func (f *fakeManager) Install(pkgs []string) error {
	f.lastInstalled = append([]string(nil), pkgs...)
	return f.installError
}

// boundFakeManager embeds base so it participates in Bind().
type boundFakeManager struct {
	base
	name string
}

func (b *boundFakeManager) Name() string                 { return b.name }
func (b *boundFakeManager) Available() bool              { return true }
func (b *boundFakeManager) Installed() ([]string, error) { return nil, nil }
func (b *boundFakeManager) Install(pkgs []string) error  { return nil }
func (b *boundFakeManager) BootstrapHint() string        { return "" }

// registerForTest registers m and schedules its removal so multiple test runs
// don't trip Register's duplicate-name panic.
func registerForTest(t *testing.T, m Manager) {
	t.Helper()
	Register(m)
	t.Cleanup(func() { delete(defaultRegistry.managers, m.Name()) })
}

func TestRegisterAndGet(t *testing.T) {
	m := &fakeManager{name: "test-register-get"}
	registerForTest(t, m)

	if got := Get("test-register-get"); got != m {
		t.Fatalf("Get(%q) = %v, want %v", "test-register-get", got, m)
	}
	if got := Get("definitely-unknown-manager"); got != nil {
		t.Fatalf("Get(unknown) = %v, want nil", got)
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	m := &fakeManager{name: "test-duplicate-register"}
	registerForTest(t, m)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("second Register() did not panic")
		}
	}()
	Register(&fakeManager{name: "test-duplicate-register"})
}

func TestNamesSortedAndIncludesCustom(t *testing.T) {
	m := &fakeManager{name: "test-names-sorted"}
	registerForTest(t, m)

	names := Names()
	if !slices.IsSorted(names) {
		t.Fatalf("Names() = %v, want sorted", names)
	}
	if !slices.Contains(names, "test-names-sorted") {
		t.Fatalf("Names() missing %q: %v", "test-names-sorted", names)
	}
}

func TestBindInjectsRunnerIntoEmbeddedBase(t *testing.T) {
	m := &boundFakeManager{name: "test-bind-embeds-base"}
	registerForTest(t, m)

	dry := runner.NewDryRun()
	Bind(dry)

	if m.runner() != dry {
		t.Fatalf("after Bind(), manager.runner() = %v, want injected DryRun", m.runner())
	}
}

func TestBindIgnoresManagersWithoutBinder(t *testing.T) {
	// A manager without runnerBinder must not break Bind. Bind should silently
	// skip it.
	m := &fakeManager{name: "test-bind-no-binder"}
	registerForTest(t, m)

	Bind(runner.NewDryRun()) // must not panic
}

func TestBaseRunnerFallsBackToExec(t *testing.T) {
	var b base
	if b.runner() == nil {
		t.Fatal("base.runner() returned nil when unset")
	}
	// The fallback has to be a real Exec so an un-bound manager can still
	// shell out — that lets `stock` remain usable if someone forgets to Bind().
	if _, ok := b.runner().(*runner.Exec); !ok {
		t.Fatalf("base.runner() = %T, want *runner.Exec", b.runner())
	}
}

func TestDiffMissing(t *testing.T) {
	tests := []struct {
		name      string
		desired   []string
		installed []string
		want      []string
	}{
		{name: "all installed", desired: []string{"a", "b"}, installed: []string{"a", "b", "c"}, want: nil},
		{name: "none installed", desired: []string{"a", "b"}, installed: nil, want: []string{"a", "b"}},
		{name: "some installed", desired: []string{"a", "b", "c"}, installed: []string{"a", "c"}, want: []string{"b"}},
		{name: "preserves order", desired: []string{"z", "y", "x"}, installed: []string{"y"}, want: []string{"z", "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffMissing(tt.desired, tt.installed)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("diffMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuiltinManagersRegisteredAtStart(t *testing.T) {
	// Smoke test: every hard-coded manager in init() should be discoverable via
	// Get by its packages.yaml key. Prevents a rename from silently dropping
	// an entire manager.
	// dnf is registered once and internally falls back to yum, so there is
	// no separate "yum" key in packages.yaml.
	want := []string{
		"apk", "apt", "brew", "brew-cask", "cargo", "dnf", "gem", "go",
		"npm", "pacman", "pipx", "winget", "zypper",
	}
	for _, name := range want {
		if Get(name) == nil {
			t.Errorf("builtin manager %q is not registered", name)
		}
	}
}
