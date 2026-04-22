package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/cushycush/store-core/hooks"
	"github.com/cushycush/store-core/ui"
	"github.com/cushycush/stock/internal/managers"
)

// section prints a styled phase header so bootstrap's progress is scannable.
func section(ctx *Context, label string) {
	fmt.Fprintln(ctx.Stdout, ui.Bold("==> "+label))
}

// Bootstrap runs the full new-machine orchestration:
//
//	pre-bootstrap hook
//	→ install missing package managers (best-effort)
//	→ stock install (packages)
//	→ store (symlinks, if available)
//	→ post-bootstrap hook
//	→ stock doctor + store doctor
//
// Each step prints clearly so users can see where we are. Failure of a hook
// aborts; failure of `store` is surfaced but doesn't roll back installs —
// packages are already on disk at that point.
func Bootstrap(args []string) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dry := fs.Bool("dry-run", false, "print commands that would run without executing them")
	skipStore := fs.Bool("skip-store", false, "skip invoking the store binary for symlinks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, err := Setup(*dry)
	if err != nil {
		return err
	}

	section(ctx, "pre-bootstrap hook")
	if err := hooks.RunGlobal(ctx.Root, "pre-bootstrap", "bootstrap"); err != nil {
		return err
	}

	section(ctx, "package managers")
	bootstrapManagers(ctx)

	section(ctx, "stock install")
	if err := RunInstall(ctx, nil); err != nil {
		return err
	}

	if !*skipStore {
		section(ctx, "store (symlinks)")
		if ctx.Run.Has("store") {
			if err := ctx.Run.Run("store"); err != nil {
				fmt.Fprintln(ctx.Stderr, ui.Warning(fmt.Sprintf("store exited with error: %s", err)))
			}
		} else {
			fmt.Fprintln(ctx.Stderr, ui.Warning("store binary not found on PATH; skipping symlink step"))
		}
	}

	section(ctx, "post-bootstrap hook")
	if err := hooks.RunGlobal(ctx.Root, "post-bootstrap", "bootstrap"); err != nil {
		return err
	}

	section(ctx, "stock doctor")
	if err := RunDoctor(ctx); err != nil {
		return err
	}
	if !*skipStore && ctx.Run.Has("store") {
		section(ctx, "store doctor")
		if err := ctx.Run.Run("store", "doctor"); err != nil {
			// store may not yet implement doctor; warn rather than fail.
			fmt.Fprintln(ctx.Stderr, ui.Dim(fmt.Sprintf("note: store doctor returned %s", err)))
		}
	}
	return nil
}

// bootstrapManagers surfaces managers needed to service applicable groups on
// this machine. "Needed" means: there's at least one applicable group whose
// referenced managers are all unavailable. We intentionally don't auto-install
// managers — they often require interactive prompts, license acceptance, or
// shell integration that a non-interactive CLI shouldn't paper over. We print
// the bootstrap hint and let the user run it.
func bootstrapManagers(ctx *Context) {
	needed := map[string]bool{}
	for _, g := range ctx.Cfg.Groups {
		if !g.Applies(ctx.Info) || len(g.Managers) == 0 {
			continue
		}
		anyAvail := false
		for name := range g.Managers {
			if m := managers.Get(name); m != nil && m.Available() {
				anyAvail = true
				break
			}
		}
		if anyAvail {
			continue
		}
		for name := range g.Managers {
			needed[name] = true
		}
	}
	if len(needed) == 0 {
		fmt.Fprintln(ctx.Stdout, "  all applicable groups have an available manager")
		return
	}
	for _, name := range managers.Names() {
		if !needed[name] {
			continue
		}
		m := managers.Get(name)
		fmt.Fprintf(ctx.Stdout, "  %s: not installed — %s\n", name, m.BootstrapHint())
	}
}
