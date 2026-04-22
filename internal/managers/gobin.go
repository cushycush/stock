package managers

import (
	"os"
	"path/filepath"
	"strings"
)

type goBin struct{ base }

func init() { Register(&goBin{}) }

func (g *goBin) Name() string    { return "go" }
func (g *goBin) Available() bool { return g.runner().Has("go") }
func (g *goBin) BootstrapHint() string {
	return "install Go from https://go.dev/dl/ or your system package manager"
}

// Installed scans $GOBIN (falling back to $GOPATH/bin, then $HOME/go/bin) for
// executable files. `go install` doesn't maintain a registry — the binaries
// in the bin dir are the source of truth. We return filename-only entries so
// a config using `go: [gopls]` matches without the module path.
func (g *goBin) Installed() ([]string, error) {
	dir := goBinDir(g.runner())
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Strip .exe on Windows so `go: [staticcheck]` still matches.
		name = strings.TrimSuffix(name, ".exe")
		names = append(names, name)
	}
	return names, nil
}

// Install takes package paths in the form expected by `go install`, e.g.
// "golang.org/x/tools/gopls@latest". The last path segment is treated as the
// binary name for idempotency checking, so configs without @latest still work.
func (g *goBin) Install(pkgs []string) error {
	installed, _ := g.Installed()
	have := map[string]struct{}{}
	for _, p := range installed {
		have[p] = struct{}{}
	}
	for _, p := range pkgs {
		base := binaryName(p)
		if _, ok := have[base]; ok {
			continue
		}
		target := p
		if !strings.Contains(target, "@") {
			target += "@latest"
		}
		if err := g.runner().Run("go", "install", target); err != nil {
			return err
		}
	}
	return nil
}

// binaryName returns the last path segment before "@version", which is the
// filename `go install` will write into $GOBIN.
func binaryName(pkg string) string {
	// Drop @version suffix first.
	if i := strings.Index(pkg, "@"); i >= 0 {
		pkg = pkg[:i]
	}
	return filepath.Base(pkg)
}

// goBinDir picks the directory `go install` writes into, honoring GOBIN,
// then GOPATH/bin, then the default $HOME/go/bin.
func goBinDir(r interface{ Output(string, ...string) (string, error) }) string {
	if v := os.Getenv("GOBIN"); v != "" {
		return v
	}
	if v := os.Getenv("GOPATH"); v != "" {
		return filepath.Join(v, "bin")
	}
	// Ask the go tool directly — honors user's env file too.
	if out, err := r.Output("go", "env", "GOBIN"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			return s
		}
	}
	if out, err := r.Output("go", "env", "GOPATH"); err == nil {
		if s := strings.TrimSpace(out); s != "" {
			return filepath.Join(s, "bin")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "go", "bin")
	}
	return ""
}
