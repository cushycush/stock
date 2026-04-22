package managers

import "strings"

type gem struct{ base }

func init() { Register(&gem{}) }

func (g *gem) Name() string    { return "gem" }
func (g *gem) Available() bool { return g.runner().Has("gem") }
func (g *gem) BootstrapHint() string {
	return "install Ruby (https://www.ruby-lang.org) — provides gem"
}

// Installed parses `gem list --local --no-versions` which emits one gem per line.
func (g *gem) Installed() ([]string, error) {
	out, err := g.runner().Output("gem", "list", "--local", "--no-versions")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "***") {
			names = append(names, line)
		}
	}
	return names, nil
}

func (g *gem) Install(pkgs []string) error {
	missing, err := missingOrAll(g, pkgs)
	if err != nil {
		return err
	}
	for _, p := range missing {
		if err := g.runner().Run("gem", "install", p); err != nil {
			return err
		}
	}
	return nil
}
