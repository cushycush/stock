package managers

import "strings"

type dnf struct{ base }

func init() { Register(&dnf{}) }

func (d *dnf) Name() string { return "dnf" }
func (d *dnf) Available() bool {
	// Prefer dnf, fall back to yum for older RHEL-family systems.
	return d.runner().Has("dnf") || d.runner().Has("yum")
}
func (d *dnf) BootstrapHint() string {
	return "dnf/yum ships with Fedora/RHEL/CentOS"
}

// bin returns whichever of dnf/yum is installed.
func (d *dnf) bin() string {
	if d.runner().Has("dnf") {
		return "dnf"
	}
	return "yum"
}

// Installed uses `repoquery --userinstalled` to mirror apt-mark showmanual.
// Available on modern dnf; the output is <name>-<version>.<arch>, so we trim
// it back to the bare package name.
func (d *dnf) Installed() ([]string, error) {
	out, err := d.runner().Output(d.bin(), "repoquery", "--userinstalled", "--qf", "%{name}")
	if err != nil {
		// Not all dnf versions support --userinstalled; fall back to
		// rpm -qa which at least gives idempotency, at the cost of
		// polluting snapshots with deps. The install path doesn't need
		// perfect accuracy here since dnf is itself idempotent.
		out, err = d.runner().Output("rpm", "-qa", "--qf", "%{NAME}\n")
		if err != nil {
			return nil, err
		}
	}
	return splitLines(out), nil
}

func (d *dnf) Install(pkgs []string) error {
	missing, err := missingOrAll(d, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{d.bin(), "install", "-y"}, missing...)
	if !isRoot() {
		if !d.runner().Has("sudo") {
			return errNoSudo(d.bin()+" install", strings.Join(missing, " "))
		}
		args = append([]string{"sudo"}, args...)
	}
	return d.runner().Run(args[0], args[1:]...)
}
