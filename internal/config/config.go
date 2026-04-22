// Package config loads .store/packages.yaml, finds the store root,
// and evaluates when: filters against the current platform.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cushycush/stock/internal/platform"
	"gopkg.in/yaml.v3"
)

// DirName is the shared directory both store and stock read from.
const DirName = ".store"

// FileName is stock's config file inside .store/.
const FileName = "packages.yaml"

// Group is one named entry under `packages:` in packages.yaml. Each manager key
// (brew, apt, cargo, ...) maps to a list of package names. An optional `when:`
// filter gates whether the group applies to the current machine.
type Group struct {
	Name     string
	Managers map[string][]string
	When     *When
}

// When filters a Group to specific platforms/hosts. All specified fields must
// match (AND). Within a list-valued field, any entry matches (OR). Each
// list-valued field accepts either a YAML scalar or a YAML sequence.
type When struct {
	OS       stringList `yaml:"os,omitempty"`
	Arch     stringList `yaml:"arch,omitempty"`
	Distro   stringList `yaml:"distro,omitempty"`
	Hostname stringList `yaml:"hostname,omitempty"`
	Shell    stringList `yaml:"shell,omitempty"`
	WSL      *bool      `yaml:"wsl,omitempty"`
}

// File is the top-level schema of packages.yaml.
type File struct {
	Groups []Group
}

// reservedKeys inside a group map to well-known fields, not package lists.
var reservedKeys = map[string]bool{"when": true}

// FindRoot walks upward from start looking for a directory containing .store/.
// Returns the directory that contains .store/ (not .store/ itself).
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, DirName)); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s directory found above %s", DirName, start)
		}
		dir = parent
	}
}

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
// We decode into a generic map so unknown manager keys are preserved without
// needing a hard-coded list here (the managers package owns that list).
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
				var w When
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

// Applies reports whether the group's when: filter matches info. A nil filter
// (group without when:) always applies.
func (g Group) Applies(info platform.Info) bool {
	if g.When == nil {
		return true
	}
	w := g.When
	if len(w.OS) > 0 && !contains(w.OS, info.OS) {
		return false
	}
	if len(w.Arch) > 0 && !contains(w.Arch, info.Arch) {
		return false
	}
	if len(w.Distro) > 0 && !contains(w.Distro, info.Distro) {
		return false
	}
	if len(w.Hostname) > 0 && !contains(w.Hostname, info.Hostname) {
		return false
	}
	if len(w.Shell) > 0 && !contains(w.Shell, info.Shell) {
		return false
	}
	if w.WSL != nil && *w.WSL != info.WSL {
		return false
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}

// ExpandPath expands a leading ~ to the current user's home directory.
// Mirrors store's path expansion semantics.
func ExpandPath(p string) string {
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}
