package managers

import "strings"

type apk struct{ base }

func init() { Register(&apk{}) }

func (a *apk) Name() string    { return "apk" }
func (a *apk) Available() bool { return a.runner().Has("apk") }
func (a *apk) BootstrapHint() string {
	return "apk ships with Alpine Linux"
}

// Installed uses `apk info` which lists installed package names, one per line.
func (a *apk) Installed() ([]string, error) {
	out, err := a.runner().Output("apk", "info")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (a *apk) Install(pkgs []string) error {
	missing, err := missingOrAll(a, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"apk", "add"}, missing...)
	if !isRoot() {
		if !a.runner().Has("sudo") {
			return errNoSudo("apk add", strings.Join(missing, " "))
		}
		args = append([]string{"sudo"}, args...)
	}
	return a.runner().Run(args[0], args[1:]...)
}
