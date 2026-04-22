// Package config loads .store/packages.yaml and evaluates when: filters
// against the current platform. The .store layout, when: semantics, tilde
// expansion, and root discovery all live in store-core; this package is
// just the stock-specific packages.yaml schema layered on top.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corecfg "github.com/cushycush/store-core/config"
	"github.com/cushycush/store-core/platform"
	"gopkg.in/yaml.v3"
)

// DirName is the shared directory both store and stock read from. Sourced
// from store-core so the two tools can never drift.
const DirName = corecfg.ConfigDir

// FileName is stock's config file inside .store/.
const FileName = "packages.yaml"

// Group is one named entry under `packages:` in packages.yaml. Each manager
// key (brew, apt, cargo, ...) maps to a list of package names. An optional
// `when:` filter gates whether the group applies to the current machine.
type Group struct {
	Name     string
	Managers map[string][]string
	When     *corecfg.WhenClause
}

// File is the top-level schema of packages.yaml.
type File struct {
	Groups []Group
}

// reservedKeys inside a group map to well-known fields, not package lists.
var reservedKeys = map[string]bool{"when": true}

// Load reads <root>/.store/packages.yaml and returns the parsed file.
// It is not an error for packages.yaml to be missing — an empty File is returned.
func Load(root string) (*File, error) {
	path := filepath.Join(root, DirName, FileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{}, nil
		}
		return nil, err
	}
	return parse(b)
}

// parse turns YAML bytes into a File. The YAML shape is:
//
//	packages:
//	  <group-name>:
//	    <manager>: [pkg, pkg, ...]
//	    when: { ... }
//
// Unknown manager keys pass through unchanged — the managers package owns
// the list of recognised names.
func parse(b []byte) (*File, error) {
	var raw struct {
		Packages map[string]map[string]yaml.Node `yaml:"packages"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	f := &File{}
	for name, body := range raw.Packages {
		g := Group{Name: name, Managers: map[string][]string{}}
		for key, node := range body {
			if key == "when" {
				var w corecfg.WhenClause
				if err := node.Decode(&w); err != nil {
					return nil, fmt.Errorf("group %q: invalid when: %w", name, err)
				}
				g.When = &w
				continue
			}
			if reservedKeys[key] {
				continue
			}
			var pkgs []string
			if err := node.Decode(&pkgs); err != nil {
				return nil, fmt.Errorf("group %q: manager %q: expected list of strings: %w", name, key, err)
			}
			g.Managers[key] = pkgs
		}
		f.Groups = append(f.Groups, g)
	}
	return f, nil
}

// Applies reports whether the group's when: filter matches info. A nil
// filter (group without when:) always applies.
func (g Group) Applies(info platform.Info) bool {
	return g.When.Matches(info)
}

// FindRoot walks upward from start looking for a directory containing
// .store/. Re-exported for stock's callers; the implementation lives in
// store-core.
func FindRoot(start string) (string, error) { return corecfg.FindRoot(start) }
