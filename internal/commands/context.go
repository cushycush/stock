// Package commands implements stock's subcommands. Each command is a small
// struct with an Exec(args []string) method; main.go dispatches to them.
package commands

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cushycush/stock/internal/config"
	"github.com/cushycush/stock/internal/env"
	"github.com/cushycush/stock/internal/managers"
	"github.com/cushycush/stock/internal/platform"
	"github.com/cushycush/stock/internal/runner"
)

// Context bundles everything subcommands need. Commands receive one via Setup.
type Context struct {
	Root   string
	Cfg    *config.File
	Info   platform.Info
	Stdout io.Writer
	Stderr io.Writer
	Run    runner.Runner
	DryRun bool
}

// Setup discovers the store root, loads packages.yaml, detects platform, and
// wires the runner into every registered manager. Called once per command.
func Setup(dryRun bool) (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := config.FindRoot(cwd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	info := platform.Detect()
	env.Apply(root, info)

	var r runner.Runner
	if dryRun {
		r = runner.NewDryRun()
	} else {
		r = runner.NewExec()
	}
	managers.Bind(r)

	return &Context{
		Root:   root,
		Cfg:    cfg,
		Info:   info,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Run:    r,
		DryRun: dryRun,
	}, nil
}

// parseFlags returns a FlagSet pre-registered with --dry-run. Every command
// accepts it so users can preview any action the same way.
func parseFlags(name string, args []string) (*flag.FlagSet, *bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dry := fs.Bool("dry-run", false, "print commands that would run without executing them")
	return fs, dry
}
