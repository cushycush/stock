package managers

import (
	"encoding/json"
	"fmt"
)

type pipx struct{ base }

func init() { Register(&pipx{}) }

func (p *pipx) Name() string    { return "pipx" }
func (p *pipx) Available() bool { return p.runner().Has("pipx") }
func (p *pipx) BootstrapHint() string {
	return "install pipx: python3 -m pip install --user pipx && python3 -m pipx ensurepath"
}

// pipxList mirrors the shape of `pipx list --json`.
type pipxList struct {
	Venvs map[string]struct {
		Metadata struct {
			MainPackage struct {
				Package string `json:"package"`
			} `json:"main_package"`
		} `json:"metadata"`
	} `json:"venvs"`
}

func (p *pipx) Installed() ([]string, error) {
	out, err := p.runner().Output("pipx", "list", "--json")
	if err != nil {
		// Empty environments can make pipx exit non-zero on some versions;
		// treat that as "nothing installed" rather than failing hard.
		if out == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("pipx list: %w", err)
	}
	var parsed pipxList
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, fmt.Errorf("pipx list json: %w", err)
	}
	names := make([]string, 0, len(parsed.Venvs))
	for name, v := range parsed.Venvs {
		// Prefer the package name from metadata; fall back to the venv key.
		if v.Metadata.MainPackage.Package != "" {
			names = append(names, v.Metadata.MainPackage.Package)
		} else {
			names = append(names, name)
		}
	}
	return names, nil
}

func (p *pipx) Install(pkgs []string) error {
	missing, err := missingOrAll(p, pkgs)
	if err != nil {
		return err
	}
	for _, pkg := range missing {
		if err := p.runner().Run("pipx", "install", pkg); err != nil {
			return err
		}
	}
	return nil
}
