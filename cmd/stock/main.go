// Command stock installs the packages, tools, and runtimes your dotfiles
// depend on. Companion to store — see https://github.com/cushycush/store.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/cushycush/stock/internal/commands"
)

// builtins maps subcommand names to their handlers. If an argument isn't here,
// main looks for a `stock-<arg>` executable on $PATH (Git-style dispatch).
var builtins = map[string]func(args []string) error{
	"install":   commands.Install,
	"diff":      commands.Diff,
	"doctor":    commands.Doctor,
	"snapshot":  commands.Snapshot,
	"platform":  commands.Platform,
	"bootstrap": commands.Bootstrap,
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-h", "--help", "help":
		usage()
		return
	case "-v", "--version", "version":
		fmt.Println("stock dev")
		return
	}

	if fn, ok := builtins[cmd]; ok {
		if err := fn(args); err != nil {
			fmt.Fprintf(os.Stderr, "stock %s: %s\n", cmd, err)
			os.Exit(1)
		}
		return
	}

	// Git-style fallback: look for stock-<cmd> on $PATH and run it with
	// the remaining args. `store` uses the same pattern to reach this
	// binary as `store-stock`; this block extends it one level deeper.
	if path, err := exec.LookPath("stock-" + cmd); err == nil {
		child := exec.Command(path, args...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Env = os.Environ()
		if err := child.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "stock: run %s: %s\n", path, err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "stock: unknown command %q\n", cmd)
	usage()
	os.Exit(2)
}

func usage() {
	fmt.Fprintln(os.Stderr, `stock — machine provisioning companion to store

usage: stock <command> [flags] [args]

commands:
  install [group...]   install everything matching platform + when:
  diff    [group...]   preview what install would change (read-only)
  doctor               verify managers, detect drift from packages.yaml
  snapshot             write currently installed packages to .store/packages.yaml
  platform             print detected platform info
  bootstrap            run the full new-machine flow (hooks, install, store)

flags:
  --dry-run            print commands that would run (install, bootstrap)
  --help               show this message

see .store/packages.yaml for config.`)
}
