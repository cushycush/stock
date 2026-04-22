package commands

import (
	"fmt"

	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/store-core/ui"
)

// Platform runs `stock platform` — prints detected platform info. Useful for
// debugging when: filters.
func Platform(args []string) error {
	fs, _ := parseFlags("platform", args)
	if err := fs.Parse(args); err != nil {
		return err
	}

	info := platform.Detect()
	row := func(label, value string) {
		fmt.Printf("%s %s\n", ui.Dim(fmt.Sprintf("%-9s", label+":")), value)
	}
	row("os", info.OS)
	row("arch", info.Arch)
	row("distro", emptyDash(info.Distro))
	row("hostname", info.Hostname)
	row("shell", emptyDash(info.Shell))
	row("wsl", fmt.Sprintf("%t", info.WSL))
	return nil
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
