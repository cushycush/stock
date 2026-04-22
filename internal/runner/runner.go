// Package runner executes shell commands. It exists as an interface so package
// managers can be unit-tested with a fake runner, and so a single --dry-run flag
// can swap the real runner for one that only prints.
package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Runner interface {
	// Run executes name with args, streaming stdout/stderr to the user's terminal.
	// Returns the command's exit error, if any.
	Run(name string, args ...string) error
	// Output executes name with args and captures stdout. Stderr is streamed
	// so users see diagnostics even when we only want stdout.
	Output(name string, args ...string) (string, error)
	// Has reports whether `name` exists on $PATH.
	Has(name string) bool
}

type Exec struct {
	Stdout io.Writer
	Stderr io.Writer
	Env    []string // if nil, inherits os.Environ()
}

func NewExec() *Exec {
	return &Exec{Stdout: os.Stdout, Stderr: os.Stderr}
}

func (e *Exec) env() []string {
	if e.Env == nil {
		return os.Environ()
	}
	return e.Env
}

func (e *Exec) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	cmd.Env = e.env()
	return cmd.Run()
}

func (e *Exec) Output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = e.Stderr
	cmd.Env = e.env()
	if err := cmd.Run(); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func (e *Exec) Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// DryRun prints commands it would have run instead of executing them. Output
// always returns empty string so callers treat state queries as "nothing installed".
type DryRun struct{ Out io.Writer }

func NewDryRun() *DryRun { return &DryRun{Out: os.Stdout} }

func (d *DryRun) Run(name string, args ...string) error {
	fmt.Fprintf(d.Out, "+ %s %s\n", name, shellJoin(args))
	return nil
}

func (d *DryRun) Output(name string, args ...string) (string, error) {
	fmt.Fprintf(d.Out, "+ %s %s  # output suppressed\n", name, shellJoin(args))
	return "", nil
}

func (d *DryRun) Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func shellJoin(args []string) string {
	var b bytes.Buffer
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(a)
	}
	return b.String()
}
