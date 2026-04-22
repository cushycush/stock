package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/managers"
	"gopkg.in/yaml.v3"
)

// Snapshot runs `stock snapshot` — the install-side analog of store import.
// Captures currently installed packages into .store/packages.yaml.
//
// By default it prints the generated YAML to stdout. With --write, it writes
// it to .store/packages.yaml, failing if the file already exists unless --force
// is passed (we refuse to silently clobber hand-written config).
func Snapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	write := fs.Bool("write", false, "write .store/packages.yaml instead of printing to stdout")
	force := fs.Bool("force", false, "overwrite an existing .store/packages.yaml (use with --write)")
	group := fs.String("group", "host", "name of the group to emit")
	managerList := fs.String("managers", "", "comma-separated managers to snapshot (default: all available)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Setup also validates we're inside a store root — snapshot requires one
	// because the output is meaningless without somewhere to put it.
	ctx, err := Setup(false)
	if err != nil {
		return err
	}

	var chosen []managers.Manager
	if *managerList != "" {
		for _, name := range strings.Split(*managerList, ",") {
			name = strings.TrimSpace(name)
			m := managers.Get(name)
			if m == nil {
				return fmt.Errorf("unknown manager: %s", name)
			}
			if !m.Available() {
				fmt.Fprintf(ctx.Stderr, "warning: %s not available; skipping\n", name)
				continue
			}
			chosen = append(chosen, m)
		}
	} else {
		for _, name := range managers.Names() {
			m := managers.Get(name)
			if m.Available() {
				chosen = append(chosen, m)
			}
		}
	}

	body := map[string][]string{}
	for _, m := range chosen {
		pkgs, err := m.Installed()
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "warning: %s installed: %s\n", m.Name(), err)
			continue
		}
		if len(pkgs) == 0 {
			continue
		}
		sort.Strings(pkgs)
		body[m.Name()] = pkgs
	}

	out, err := renderSnapshot(*group, body)
	if err != nil {
		return err
	}

	if !*write {
		_, err := fmt.Fprint(ctx.Stdout, out)
		return err
	}

	target := filepath.Join(ctx.Root, config.DirName, config.FileName)
	if _, err := os.Stat(target); err == nil && !*force {
		return fmt.Errorf("%s already exists; rerun with --force to overwrite", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(out), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "wrote %s\n", target)
	return nil
}

// renderSnapshot emits yaml in the shape computePlan expects. We build it
// manually (via yaml.Marshal of a map) rather than through our own Group
// struct so the output is minimal — no empty fields, no when: block.
func renderSnapshot(group string, body map[string][]string) (string, error) {
	if len(body) == 0 {
		return "packages: {}\n", nil
	}
	doc := map[string]any{
		"packages": map[string]any{
			group: body,
		},
	}
	b, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
