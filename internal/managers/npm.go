package managers

import (
	"encoding/json"
	"fmt"
)

type npmGlobal struct{ base }

func init() { Register(&npmGlobal{}) }

func (n *npmGlobal) Name() string    { return "npm" }
func (n *npmGlobal) Available() bool { return n.runner().Has("npm") }
func (n *npmGlobal) BootstrapHint() string {
	return "install Node.js (https://nodejs.org) or use a version manager (fnm, nvm, volta) — provides npm"
}

// npmList is the subset of `npm ls -g --json` we care about.
type npmList struct {
	Dependencies map[string]json.RawMessage `json:"dependencies"`
}

// Installed parses `npm ls -g --depth=0 --json` so we see only user-installed
// top-level globals, not the transitive dep graph.
func (n *npmGlobal) Installed() ([]string, error) {
	out, err := n.runner().Output("npm", "ls", "-g", "--depth=0", "--json")
	// npm ls exits non-zero when there are peer-dep issues even though the
	// JSON is still valid, so we try to parse regardless of err.
	var parsed npmList
	if jerr := json.Unmarshal([]byte(out), &parsed); jerr != nil {
		if err != nil {
			return nil, fmt.Errorf("npm ls failed: %w", err)
		}
		return nil, fmt.Errorf("npm ls json: %w", jerr)
	}
	names := make([]string, 0, len(parsed.Dependencies))
	for name := range parsed.Dependencies {
		names = append(names, name)
	}
	return names, nil
}

func (n *npmGlobal) Install(pkgs []string) error {
	missing, err := missingOrAll(n, pkgs)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"install", "-g"}, missing...)
	return n.runner().Run("npm", args...)
}
