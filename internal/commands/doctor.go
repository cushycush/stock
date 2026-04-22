package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cushycush/stock/internal/managers"
)

// Doctor runs `stock doctor` — verifies package managers exist and reports
// drift between packages.yaml and what's actually installed.
func Doctor(args []string) error {
	fs, _ := parseFlags("doctor", args)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, err := Setup(false)
	if err != nil {
		return err
	}
	return RunDoctor(ctx)
}

// RunDoctor is the reusable doctor core for bootstrap.
func RunDoctor(ctx *Context) error {
	// Listing section: one line per known manager with a plain status. We
	// don't call anything "MISSING" here — a missing apt on Arch is normal
	// when the user listed apt as a Debian-family alternative inside the
	// same group as pacman. The load-bearing signal is unservable groups
	// (below), not this list.
	fmt.Fprintln(ctx.Stdout, "package managers:")
	for _, name := range managers.Names() {
		m := managers.Get(name)
		status := "available"
		if !m.Available() {
			status = "not installed"
		}
		fmt.Fprintf(ctx.Stdout, "  %-10s %s\n", name, status)
	}

	// Unservable groups: a group that applies to this platform but has no
	// available manager. This is the real error case — packages.yaml says
	// to install something and we have no way to do it.
	if unservable := unservableGroups(ctx); len(unservable) > 0 {
		fmt.Fprintln(ctx.Stdout, "\nunservable groups:")
		for _, line := range unservable {
			fmt.Fprintf(ctx.Stdout, "  %s\n", line)
		}
	}

	// Drift: among managers we can actually run, what's declared but not
	// installed? This is what `stock install` would change.
	fmt.Fprintln(ctx.Stdout, "\ndrift:")
	plans, warnings, err := computePlan(ctx, nil)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", w)
	}
	anyDrift := false
	for _, p := range plans {
		if len(p.Missing) == 0 {
			continue
		}
		anyDrift = true
		fmt.Fprintf(ctx.Stdout, "  %s missing: %s\n", p.Manager.Name(), strings.Join(p.Missing, " "))
	}
	if !anyDrift {
		fmt.Fprintln(ctx.Stdout, "  none — installed state matches packages.yaml")
	}
	return nil
}

// unservableGroups finds applicable groups where no referenced manager is
// available on the current system. These are actionable errors: the user
// has declared packages the machine cannot install.
func unservableGroups(ctx *Context) []string {
	var out []string
	for _, g := range ctx.Cfg.Groups {
		if !g.Applies(ctx.Info) || len(g.Managers) == 0 {
			continue
		}
		anyAvail := false
		var names []string
		for name := range g.Managers {
			names = append(names, name)
			if m := managers.Get(name); m != nil && m.Available() {
				anyAvail = true
				break
			}
		}
		if !anyAvail {
			sort.Strings(names)
			out = append(out, fmt.Sprintf("%s (no available manager among: %s)", g.Name, strings.Join(names, ", ")))
		}
	}
	sort.Strings(out)
	return out
}

