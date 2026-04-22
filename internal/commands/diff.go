package commands

import (
	"fmt"
	"strings"
)

// Diff runs `stock diff [group...]` — previews what install would do.
// Always read-only; ignores --dry-run since diff never writes.
func Diff(args []string) error {
	fs, _ := parseFlags("diff", args)
	if err := fs.Parse(args); err != nil {
		return err
	}
	// Diff itself never runs anything destructive, but we still want the
	// warnings from computePlan (missing managers, unknown manager keys).
	ctx, err := Setup(false)
	if err != nil {
		return err
	}

	only, err := selectGroups(ctx, fs.Args())
	if err != nil {
		return err
	}

	plans, warnings, err := computePlan(ctx, only)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(ctx.Stderr, "warning: %s\n", w)
	}

	if len(plans) == 0 {
		fmt.Fprintln(ctx.Stdout, "no managers apply to the current platform")
		return nil
	}

	anyMissing := false
	for _, p := range plans {
		if len(p.Missing) == 0 {
			fmt.Fprintf(ctx.Stdout, "%s: up to date (%d package(s))\n", p.Manager.Name(), len(p.Desired))
			continue
		}
		anyMissing = true
		fmt.Fprintf(ctx.Stdout, "%s: would install %d package(s):\n", p.Manager.Name(), len(p.Missing))
		for _, pkg := range p.Missing {
			fmt.Fprintf(ctx.Stdout, "  + %s\n", pkg)
		}
	}
	if !anyMissing {
		fmt.Fprintln(ctx.Stdout, strings.TrimSpace("everything declared in packages.yaml is already installed"))
	}
	return nil
}
