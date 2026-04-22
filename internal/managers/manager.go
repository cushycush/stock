// Package managers defines the package-manager abstraction and registers an
// implementation for each supported manager key (brew, apt, cargo, ...).
package managers

import (
	"fmt"
	"sort"

	"github.com/cushycush/stock/internal/runner"
)

// Manager is the contract every package manager satisfies. Implementations
// should be safe to call when Available() is false — callers check first.
type Manager interface {
	// Name returns the key used in packages.yaml (e.g., "brew", "apt").
	Name() string
	// Available reports whether the underlying tool is on $PATH.
	Available() bool
	// Installed returns package names the manager reports as installed.
	// The returned list is what we compare against desired state; managers
	// that cannot reliably list installed packages may return an empty slice
	// with a nil error and let the installer rely on the manager's own
	// idempotency.
	Installed() ([]string, error)
	// Install installs the given packages. It should be idempotent: already
	// installed packages are skipped. Implementations may either pre-filter
	// against Installed() or delegate to the manager.
	Install(pkgs []string) error
	// BootstrapHint returns a human-readable note describing how to install
	// the manager itself, used by `stock doctor` and `stock bootstrap`.
	BootstrapHint() string
}

// Registry holds all known managers keyed by Name(). Implementations register
// themselves in their package init().
type Registry struct {
	managers map[string]Manager
}

var defaultRegistry = &Registry{managers: map[string]Manager{}}

// Register adds m to the default registry. Panics on duplicate names because
// that's a programming error at binary-build time, not runtime.
func Register(m Manager) {
	if _, exists := defaultRegistry.managers[m.Name()]; exists {
		panic(fmt.Sprintf("manager %q registered twice", m.Name()))
	}
	defaultRegistry.managers[m.Name()] = m
}

// Get returns the manager with the given name, or nil if unknown.
func Get(name string) Manager { return defaultRegistry.managers[name] }

// Names returns every registered manager name in deterministic order.
func Names() []string {
	out := make([]string, 0, len(defaultRegistry.managers))
	for n := range defaultRegistry.managers {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Bind injects a runner into every registered manager. Call this once at
// program start so every manager uses the same (possibly dry-run) runner.
func Bind(r runner.Runner) {
	for _, m := range defaultRegistry.managers {
		if b, ok := m.(runnerBinder); ok {
			b.setRunner(r)
		}
	}
}

// runnerBinder is an internal hook; all builtin managers implement it.
type runnerBinder interface {
	setRunner(r runner.Runner)
}

// base is embedded by every manager implementation to satisfy runnerBinder
// and provide shared helpers.
type base struct {
	r runner.Runner
}

func (b *base) setRunner(r runner.Runner) { b.r = r }
func (b *base) runner() runner.Runner {
	if b.r == nil {
		return runner.NewExec()
	}
	return b.r
}

// diffMissing returns items in desired that are not in installed.
func diffMissing(desired, installed []string) []string {
	have := make(map[string]struct{}, len(installed))
	for _, p := range installed {
		have[p] = struct{}{}
	}
	var missing []string
	for _, p := range desired {
		if _, ok := have[p]; !ok {
			missing = append(missing, p)
		}
	}
	return missing
}
