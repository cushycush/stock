package managers

import (
	"os"
	"strings"
)

type apt struct{ base }

func init() { Register(&apt{}) }

func (a *apt) Name() string    { return "apt" }
func (a *apt) Available() bool { return a.runner().Has("apt-get") }
func (a *apt) BootstrapHint() string {
	return "apt is only available on Debian/Ubuntu-family systems (ships with the OS)"
}

// Installed enumerates packages the user explicitly asked for, rather than
// every library pulled in as a dependency. apt-mark showmanual is the right
// tool — dpkg -l would drown the list in transitive packages and cause
// `stock snapshot` to emit a config no human wrote.
func (a *apt) Installed() ([]string, error) {
	out, err := a.runner().Output("apt-mark", "showmanual")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (a *apt) Install(pkgs []string) error {
	missing, err := missingOrAll(a, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	// Fresh Debian/Ubuntu images (and long-idle hosts) ship with an empty
	// /var/lib/apt/lists, so `apt-get install` fails with "Unable to locate
	// package" until `apt-get update` has run at least once. Refresh here
	// so `stock install` works end-to-end on a clean machine.
	if aptListsEmpty() {
		if err := a.runAsRoot("apt-get", "update"); err != nil {
			return err
		}
	}
	return a.runAsRoot(append([]string{"apt-get", "install", "-y"}, missing...)...)
}

// runAsRoot invokes an apt-get verb, prefixing sudo when not already root.
// Failing here with a clear sudo-missing error beats a cryptic permission
// denied out of apt itself.
func (a *apt) runAsRoot(argv ...string) error {
	if !isRoot() {
		if !a.runner().Has("sudo") {
			return errNoSudo(strings.Join(argv[:2], " "), strings.Join(argv[2:], " "))
		}
		argv = append([]string{"sudo"}, argv...)
	}
	return a.runner().Run(argv[0], argv[1:]...)
}

// aptListsEmpty reports whether /var/lib/apt/lists has no cached repo data.
// `lock` and `partial/` always exist even on a freshly-wiped cache, so we
// ignore them. A missing or unreadable directory is treated as empty so the
// caller triggers `apt-get update`, which will create it.
func aptListsEmpty() bool {
	entries, err := os.ReadDir("/var/lib/apt/lists")
	if err != nil {
		return true
	}
	for _, e := range entries {
		switch e.Name() {
		case "lock", "partial":
			continue
		}
		return false
	}
	return true
}
