package managers

import "strings"

type cargo struct{ base }

func init() { Register(&cargo{}) }

func (c *cargo) Name() string    { return "cargo" }
func (c *cargo) Available() bool { return c.runner().Has("cargo") }
func (c *cargo) BootstrapHint() string {
	return "install rustup: https://rustup.rs — provides cargo"
}

// Installed parses `cargo install --list`, which outputs blocks like:
//
//	ripgrep v14.1.0:
//	    rg
//
// Only the lines ending in ':' (and starting flush-left) name installed crates.
func (c *cargo) Installed() ([]string, error) {
	out, err := c.runner().Output("cargo", "install", "--list")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		name, _, _ := strings.Cut(line, " ")
		name = strings.TrimSuffix(name, ":")
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func (c *cargo) Install(pkgs []string) error {
	missing, err := missingOrAll(c, pkgs)
	if err != nil {
		return err
	}
	for _, p := range missing {
		if err := c.runner().Run("cargo", "install", p); err != nil {
			return err
		}
	}
	return nil
}
