package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cushycush/store-core/ui"
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
	fmt.Fprintln(ctx.Stdout, ui.Bold("package managers:"))
	for _, name := range managers.Names() {
		m := managers.Get(name)
		chip := ui.DoctorOK()
		status := "available"
		if !m.Available() {
			chip = ui.Dim("[  ]")
			status = ui.Dim("not installed")
		}
		fmt.Fprintf(ctx.Stdout, "  %s %-10s %s\n", chip, name, status)
	}

	// Unservable groups: a group that applies to this platform but has no
	// available manager. This is the real error case — packages.yaml says
	// to install something and we have no way to do it.
	if unservable := unservableGroups(ctx); len(unservable) > 0 {
		fmt.Fprintln(ctx.Stdout, "\n"+ui.Bold("unservable groups:"))
		for _, line := range unservable {
			fmt.Fprintf(ctx.Stdout, "  %s %s\n", ui.DoctorError(), line)
		}
	}

	// Drift: among managers we can actually run, what's declared but not
	// installed? This is what `stock install` would change.
	fmt.Fprintln(ctx.Stdout, "\n"+ui.Bold("drift:"))
	plans, warnings, err := computePlan(ctx, nil)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintln(ctx.Stderr, ui.Warning(w.Error()))
	}
	anyDrift := false
	for _, p := range plans {
		if len(p.Missing) == 0 {
			continue
		}
		anyDrift = true
		fmt.Fprintf(ctx.Stdout, "  %s %s missing: %s\n",
			ui.DoctorWarn(), ui.Bold(p.Manager.Name()), strings.Join(p.Missing, " "))
	}
	if !anyDrift {
		fmt.Fprintf(ctx.Stdout, "  %s installed state matches packages.yaml\n", ui.DoctorOK())
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

