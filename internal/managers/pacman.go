package managers

import "strings"

type pacman struct{ base }

func init() { Register(&pacman{}) }

func (p *pacman) Name() string    { return "pacman" }
func (p *pacman) Available() bool { return p.runner().Has("pacman") }
func (p *pacman) BootstrapHint() string {
	return "pacman ships with Arch Linux and derivatives"
}

// Installed lists explicitly-installed packages (-Qe) to avoid polluting
// snapshots with transitive deps.
func (p *pacman) Installed() ([]string, error) {
	out, err := p.runner().Output("pacman", "-Qe", "-q")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (p *pacman) Install(pkgs []string) error {
	missing, err := missingOrAll(p, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"pacman", "-S", "--needed", "--noconfirm"}, missing...)
	if !isRoot() {
		if !p.runner().Has("sudo") {
			return errNoSudo("pacman -S", strings.Join(missing, " "))
		}
		args = append([]string{"sudo"}, args...)
	}
	return p.runner().Run(args[0], args[1:]...)
}
