package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/cushycush/stock/internal/hooks"
	"github.com/cushycush/stock/internal/managers"
)

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

	fmt.Fprintln(ctx.Stdout, "==> pre-bootstrap hook")
	if err := hooks.Run(ctx.Root, "pre-bootstrap", ctx.Info); err != nil {
		return err
	}

	fmt.Fprintln(ctx.Stdout, "==> package managers")
	bootstrapManagers(ctx)

	fmt.Fprintln(ctx.Stdout, "==> stock install")
	if err := RunInstall(ctx, nil); err != nil {
		return err
	}

	if !*skipStore {
		fmt.Fprintln(ctx.Stdout, "==> store (symlinks)")
		if ctx.Run.Has("store") {
			if err := ctx.Run.Run("store"); err != nil {
				fmt.Fprintf(ctx.Stderr, "warning: store exited with error: %s\n", err)
			}
		} else {
			fmt.Fprintln(ctx.Stderr, "warning: store binary not found on PATH; skipping symlink step")
		}
	}

	fmt.Fprintln(ctx.Stdout, "==> post-bootstrap hook")
	if err := hooks.Run(ctx.Root, "post-bootstrap", ctx.Info); err != nil {
		return err
	}

	fmt.Fprintln(ctx.Stdout, "==> stock doctor")
	if err := RunDoctor(ctx); err != nil {
		return err
	}
	if !*skipStore && ctx.Run.Has("store") {
		fmt.Fprintln(ctx.Stdout, "==> store doctor")
		if err := ctx.Run.Run("store", "doctor"); err != nil {
			// store may not yet implement doctor; warn rather than fail.
			fmt.Fprintf(ctx.Stderr, "note: store doctor returned %s\n", err)
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
