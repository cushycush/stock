package commands

import (
	"github.com/cushycush/stock/internal/tui"
)

// TUI launches the read-only stock dashboard. It reuses Setup so the user
// gets the same root-discovery + config-load errors they see from every
// other subcommand — the TUI is a view onto that same context.
//
// version is wired through from main.go; the TUI shows it alongside the
// brand in its header so users can see what they're running.
func TUI(version string) func(args []string) error {
	return func(args []string) error {
		fs, _ := parseFlags("tui", args)
		if err := fs.Parse(args); err != nil {
			return err
		}
		ctx, err := Setup(false)
		if err != nil {
			return err
		}
		return tui.New(ctx.Root, version, ctx.Cfg, ctx.Info).Run()
	}
}
