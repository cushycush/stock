package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	corecfg "github.com/cushycush/store-core/config"
	"github.com/cushycush/store-core/platform"
)

func TestParseMinimal(t *testing.T) {
	in := []byte(`
packages:
  core:
    pacman: [git, ripgrep]
    apt:    [git, ripgrep]
`)
	f, err := parse(in)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if len(f.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(f.Groups))
	}
	g := f.Groups[0]
	if g.Name != "core" {
		t.Errorf("name = %q, want core", g.Name)
	}
	if !reflect.DeepEqual(g.Managers["pacman"], []string{"git", "ripgrep"}) {
		t.Errorf("pacman = %v, want [git ripgrep]", g.Managers["pacman"])
	}
	if !reflect.DeepEqual(g.Managers["apt"], []string{"git", "ripgrep"}) {
		t.Errorf("apt = %v, want [git ripgrep]", g.Managers["apt"])
	}
	if g.When != nil {
		t.Errorf("expected no when clause, got %+v", g.When)
	}
}

func TestParseWhenScalarAndList(t *testing.T) {
	in := []byte(`
packages:
  scalar:
    brew: [fd]
    when: { os: darwin }
  list:
    brew: [bat]
    when: { os: [linux, darwin] }
`)
	f, err := parse(in)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	got := map[string]corecfg.Strings{}
	for _, g := range f.Groups {
		if g.When == nil {
			t.Fatalf("group %q missing when:", g.Name)
		}
		got[g.Name] = g.When.OS
	}
	if !reflect.DeepEqual([]string(got["scalar"]), []string{"darwin"}) {
		t.Errorf("scalar.OS = %v, want [darwin]", got["scalar"])
	}
	if !reflect.DeepEqual([]string(got["list"]), []string{"linux", "darwin"}) {
		t.Errorf("list.OS = %v, want [linux darwin]", got["list"])
	}
}

func TestParsePreservesUnknownManagerKeys(t *testing.T) {
	// An unknown key is passed through unchanged — the managers package owns
	// the list of recognised names, and plan.go warns if an unknown name
	// survives to install time.
	in := []byte(`
packages:
  wacky:
    my-custom-mgr: [foo, bar]
`)
	f, err := parse(in)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if got := f.Groups[0].Managers["my-custom-mgr"]; !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		t.Fatalf("my-custom-mgr = %v, want [foo bar]", got)
	}
}

func TestParseRejectsInvalidYAML(t *testing.T) {
	if _, err := parse([]byte("packages: [unclosed")); err == nil {
		t.Fatal("expected error on invalid YAML, got nil")
	}
}

func TestParseRejectsNonListManagerValue(t *testing.T) {
	in := []byte(`
packages:
  bad:
    brew: "just-a-string"
`)
	_, err := parse(in)
	if err == nil {
		t.Fatal("expected error on scalar manager value, got nil")
	}
	if !strings.Contains(err.Error(), "expected list of strings") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "expected list of strings")
	}
}

func TestParseRejectsInvalidWhen(t *testing.T) {
	in := []byte(`
packages:
  bad:
    brew: [git]
    when: "just-a-string"
`)
	_, err := parse(in)
	if err == nil {
		t.Fatal("expected error on invalid when:, got nil")
	}
	if !strings.Contains(err.Error(), "invalid when") {
		t.Fatalf("error = %q, want substring %q", err.Error(), "invalid when")
	}
}

func TestParseEmptyFile(t *testing.T) {
	f, err := parse([]byte(""))
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if len(f.Groups) != 0 {
		t.Fatalf("groups = %d, want 0", len(f.Groups))
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	// It's not an error for packages.yaml to be missing — stock is useful
	// even on a fresh .store/ that only has dotfile config so far.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(f.Groups) != 0 {
		t.Fatalf("groups = %d, want 0", len(f.Groups))
	}
}

func TestLoadReadsFromDotStorePath(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("packages:\n  core:\n    brew: [git]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(f.Groups) != 1 || f.Groups[0].Name != "core" {
		t.Fatalf("groups = %+v, want single 'core' group", f.Groups)
	}
}

func TestLoadSurfacesIOErrors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("ok"), 0o000); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root; 0o000 permission doesn't block reads")
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("expected error on unreadable file, got nil")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got ErrNotExist, want a permission error: %v", err)
	}
}

func TestGroupApplies(t *testing.T) {
	info := platform.Info{OS: "linux", Arch: "amd64", Distro: "arch", WSL: false}

	tests := []struct {
		name  string
		group Group
		want  bool
	}{
		{
			name:  "nil when always applies",
			group: Group{},
			want:  true,
		},
		{
			name:  "scalar match",
			group: Group{When: &corecfg.WhenClause{OS: corecfg.Strings{"linux"}}},
			want:  true,
		},
		{
			name:  "scalar mismatch",
			group: Group{When: &corecfg.WhenClause{OS: corecfg.Strings{"darwin"}}},
			want:  false,
		},
		{
			name:  "list match",
			group: Group{When: &corecfg.WhenClause{OS: corecfg.Strings{"darwin", "linux"}}},
			want:  true,
		},
		{
			name:  "multi-field all match",
			group: Group{When: &corecfg.WhenClause{OS: corecfg.Strings{"linux"}, Distro: corecfg.Strings{"arch"}}},
			want:  true,
		},
		{
			name:  "multi-field one mismatch",
			group: Group{When: &corecfg.WhenClause{OS: corecfg.Strings{"linux"}, Distro: corecfg.Strings{"ubuntu"}}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.Applies(info); got != tt.want {
				t.Fatalf("Applies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindRootDelegates(t *testing.T) {
	// Thin wrapper, but worth proving it walks up the same way store-core's
	// FindRoot does.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := FindRoot(child)
	if err != nil {
		t.Fatalf("FindRoot() error = %v", err)
	}
	gotResolved, _ := filepath.EvalSymlinks(got)
	wantResolved, _ := filepath.EvalSymlinks(root)
	if gotResolved != wantResolved {
		t.Fatalf("FindRoot() = %q, want %q", got, root)
	}
}

// sortedNames extracts a sorted list of group names for stable assertions.
// Kept as a helper because map iteration order breaks every other test.
func sortedNames(f *File) []string {
	names := make([]string, 0, len(f.Groups))
	for _, g := range f.Groups {
		names = append(names, g.Name)
	}
	sort.Strings(names)
	return names
}

func TestParsePreservesMultipleGroups(t *testing.T) {
	in := []byte(`
packages:
  a:
    brew: [x]
  b:
    brew: [y]
  c:
    brew: [z]
`)
	f, err := parse(in)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if got := sortedNames(f); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("groups = %v, want [a b c]", got)
	}
}
