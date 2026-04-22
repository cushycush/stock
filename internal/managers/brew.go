package managers

import "strings"

type brew struct{ base }
type brewCask struct{ base }

func init() {
	Register(&brew{})
	Register(&brewCask{})
}

func (b *brew) Name() string      { return "brew" }
func (b *brew) Available() bool   { return b.runner().Has("brew") }
func (b *brew) BootstrapHint() string {
	return `install Homebrew: /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
}

func (b *brew) Installed() ([]string, error) {
	// --formula excludes casks; casks are handled by brewCask so the two
	// managers don't fight over the same output.
	out, err := b.runner().Output("brew", "list", "--formula", "-1")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (b *brew) Install(pkgs []string) error {
	missing, err := missingOrAll(b, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"install"}, missing...)
	return b.runner().Run("brew", args...)
}

func (b *brewCask) Name() string      { return "brew-cask" }
func (b *brewCask) Available() bool   { return b.runner().Has("brew") }
func (b *brewCask) BootstrapHint() string {
	return `brew-cask ships with Homebrew on macOS; see brew BootstrapHint`
}

func (b *brewCask) Installed() ([]string, error) {
	out, err := b.runner().Output("brew", "list", "--cask", "-1")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (b *brewCask) Install(pkgs []string) error {
	missing, err := missingOrAll(b, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"install", "--cask"}, missing...)
	return b.runner().Run("brew", args...)
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// missingOrAll returns the desired packages that aren't yet installed. If the
// manager can't enumerate installed packages (returns an error), we fall back
// to installing all desired and let the manager's own idempotency kick in.
func missingOrAll(m Manager, desired []string) ([]string, error) {
	installed, err := m.Installed()
	if err != nil {
		return desired, nil
	}
	return diffMissing(desired, installed), nil
}
