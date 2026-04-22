package commands

import (
	"fmt"
	"strings"
)

// Install runs `stock install [group...]`.
func Install(args []string) error {
	fs, dry := parseFlags("install", args)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, err := Setup(*dry)
	if err != nil {
		return err
	}
	return RunInstall(ctx, fs.Args())
}

// RunInstall is the reusable install core. Bootstrap reuses this instead of
// reinvoking Install(args), which would reset the dry-run runner.
func RunInstall(ctx *Context, groupArgs []string) error {
	only, err := selectGroups(ctx, groupArgs)
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
		fmt.Fprintln(ctx.Stdout, "nothing to install: no managers matched the current platform")
		return nil
	}

	for _, p := range plans {
		if len(p.Missing) == 0 {
			fmt.Fprintf(ctx.Stdout, "%s: up to date (%d package(s))\n", p.Manager.Name(), len(p.Desired))
			continue
		}
		fmt.Fprintf(ctx.Stdout, "%s: installing %d package(s): %s\n",
			p.Manager.Name(), len(p.Missing), strings.Join(p.Missing, " "))
		if err := p.Manager.Install(p.Missing); err != nil {
			return fmt.Errorf("%s: %w", p.Manager.Name(), err)
		}
	}
	return nil
}

// selectGroups validates the positional arguments to `install` against the
// group names in packages.yaml and returns a set for computePlan's filter.
// Empty args means "all groups" — returned as nil so the filter is skipped.
func selectGroups(ctx *Context, args []string) (map[string]bool, error) {
	if len(args) == 0 {
		return nil, nil
	}
	known := groupNames(ctx.Cfg)
	out := make(map[string]bool, len(args))
	var unknown []string
	for _, a := range args {
		if !known[a] {
			unknown = append(unknown, a)
			continue
		}
		out[a] = true
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown group(s): %s", strings.Join(unknown, ", "))
	}
	return out, nil
}
