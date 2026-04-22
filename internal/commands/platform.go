package commands

import (
	"fmt"

	"github.com/cushycush/stock/internal/platform"
)

// Platform runs `stock platform` — prints detected platform info. Useful for
// debugging when: filters.
func Platform(args []string) error {
	fs, _ := parseFlags("platform", args)
	if err := fs.Parse(args); err != nil {
		return err
	}

	info := platform.Detect()
	fmt.Printf("os:       %s\n", info.OS)
	fmt.Printf("arch:     %s\n", info.Arch)
	fmt.Printf("distro:   %s\n", emptyDash(info.Distro))
	fmt.Printf("hostname: %s\n", info.Hostname)
	fmt.Printf("shell:    %s\n", emptyDash(info.Shell))
	fmt.Printf("wsl:      %t\n", info.WSL)
	return nil
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
