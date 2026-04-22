package managers

import "strings"

type zypper struct{ base }

func init() { Register(&zypper{}) }

func (z *zypper) Name() string    { return "zypper" }
func (z *zypper) Available() bool { return z.runner().Has("zypper") }
func (z *zypper) BootstrapHint() string {
	return "zypper ships with openSUSE / SUSE Linux Enterprise"
}

// Installed uses rpm -qa for enumeration. Zypper has no direct equivalent of
// apt-mark showmanual; for snapshots this is pragmatic — the user will
// hand-filter anyway.
func (z *zypper) Installed() ([]string, error) {
	out, err := z.runner().Output("rpm", "-qa", "--qf", "%{NAME}\n")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (z *zypper) Install(pkgs []string) error {
	missing, err := missingOrAll(z, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"zypper", "--non-interactive", "install"}, missing...)
	if !isRoot() {
		if !z.runner().Has("sudo") {
			return errNoSudo("zypper install", strings.Join(missing, " "))
		}
		args = append([]string{"sudo"}, args...)
	}
	return z.runner().Run(args[0], args[1:]...)
}
