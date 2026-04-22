package commands

import (
	"fmt"
	"sort"

	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
)

// managerPlan holds the desired vs missing package lists for one manager.
type managerPlan struct {
	Manager managers.Manager
	Desired []string
	Missing []string
}

// computePlan collects every package across groups that apply to the current
// machine, per manager. `only` is an optional allow-list of group names; empty
// means "all groups that match the platform".
func computePlan(ctx *Context, only map[string]bool) ([]managerPlan, []error, error) {
	byManager := map[string][]string{}
	for _, g := range ctx.Cfg.Groups {
		if len(only) > 0 && !only[g.Name] {
			continue
		}
		if !g.Applies(ctx.Info) {
			continue
		}
		for mgr, pkgs := range g.Managers {
			byManager[mgr] = append(byManager[mgr], pkgs...)
		}
	}

	// Deterministic order so diff output is stable across runs.
	var mgrNames []string
	for name := range byManager {
		mgrNames = append(mgrNames, name)
	}
	sort.Strings(mgrNames)

	var plans []managerPlan
	var warnings []error
	for _, name := range mgrNames {
		m := managers.Get(name)
		if m == nil {
			// Unknown key: likely a typo in packages.yaml. Worth warning —
			// silent skip would mask real mistakes.
			warnings = append(warnings, fmt.Errorf("unknown manager %q in packages.yaml", name))
			continue
		}
		desired := dedupe(byManager[name])
		if !m.Available() {
			// A manager unavailable on this machine is expected whenever
			// packages.yaml describes cross-platform alternatives (e.g.,
			// brew/apt/pacman siblings). Doctor's unservable-groups
			// section already flags the genuinely broken cases, so skip
			// silently here.
			continue
		}
		installed, _ := m.Installed()
		missing := diff(desired, installed)
		plans = append(plans, managerPlan{Manager: m, Desired: desired, Missing: missing})
	}
	return plans, warnings, nil
}

// groupNames returns the set of groups mentioned in packages.yaml, used to
// validate `stock install <group>` arguments.
func groupNames(cfg *config.File) map[string]bool {
	out := map[string]bool{}
	for _, g := range cfg.Groups {
		out[g.Name] = true
	}
	return out
}

// dedupe preserves order while removing duplicate package names within one
// manager. A user could legitimately list `git` under `core` and `work`; we
// shouldn't ask the manager to install it twice.
func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func diff(desired, installed []string) []string {
	have := map[string]struct{}{}
	for _, p := range installed {
		have[p] = struct{}{}
	}
	var missing []string
	for _, d := range desired {
		if _, ok := have[d]; !ok {
			missing = append(missing, d)
		}
	}
	return missing
}
