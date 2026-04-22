package managers

import "strings"

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
	args := append([]string{"apt-get", "install", "-y"}, missing...)
	// apt-get needs root. Prefer sudo if we're not already root — fail with
	// a clear message rather than a cryptic permission error.
	if !isRoot() {
		if !a.runner().Has("sudo") {
			return errNoSudo("apt-get install", strings.Join(missing, " "))
		}
		args = append([]string{"sudo"}, args...)
	}
	return a.runner().Run(args[0], args[1:]...)
}
