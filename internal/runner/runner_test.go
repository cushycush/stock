package runner

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestDryRunPrintsRunCommand(t *testing.T) {
	var buf bytes.Buffer
	d := &DryRun{Out: &buf}
	if err := d.Run("brew", "install", "ripgrep", "bat"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := "+ brew install ripgrep bat\n"
	if got := buf.String(); got != want {
		t.Fatalf("Run() output = %q, want %q", got, want)
	}
}

func TestDryRunOutputReturnsEmpty(t *testing.T) {
	var buf bytes.Buffer
	d := &DryRun{Out: &buf}
	out, err := d.Output("pacman", "-Q")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if out != "" {
		t.Fatalf("Output() = %q, want empty — state queries must look 'nothing installed' in dry-run", out)
	}
	if got := buf.String(); !strings.Contains(got, "pacman -Q") || !strings.Contains(got, "output suppressed") {
		t.Fatalf("Output() logged %q, want it to mention the command and 'output suppressed'", got)
	}
}

func TestDryRunHasMatchesExecLookPath(t *testing.T) {
	// DryRun.Has still consults $PATH because managers need real availability
	// to decide whether to produce an install plan at all.
	d := NewDryRun()

	tool := "sh"
	if runtime.GOOS == "windows" {
		tool = "cmd"
	}
	if _, err := exec.LookPath(tool); err != nil {
		t.Skipf("probe tool %q not on PATH", tool)
	}
	if !d.Has(tool) {
		t.Fatalf("Has(%q) = false; expected true since %s is on PATH", tool, tool)
	}
	if d.Has("definitely-not-a-real-binary-a7f3b9") {
		t.Fatal("Has(nonsense) = true; expected false")
	}
}

func TestExecRunCapturesError(t *testing.T) {
	// Probe Exec against a portable always-failing command. We're not
	// asserting on output here — stdout is routed to the configured writer,
	// which for this test is a buffer so the test doesn't pollute test output.
	var out, errBuf bytes.Buffer
	e := &Exec{Stdout: &out, Stderr: &errBuf}

	var name string
	var args []string
	switch runtime.GOOS {
	case "windows":
		name, args = "cmd", []string{"/C", "exit", "1"}
	default:
		name, args = "sh", []string{"-c", "exit 1"}
	}

	err := e.Run(name, args...)
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil for exit 1")
	}
}

func TestExecOutputStreamsStderrButReturnsStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}
	var errBuf bytes.Buffer
	e := &Exec{Stdout: nil, Stderr: &errBuf}

	got, err := e.Output("sh", "-c", "echo hello; echo warn 1>&2")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if strings.TrimSpace(got) != "hello" {
		t.Fatalf("Output() = %q, want 'hello'", got)
	}
	if !strings.Contains(errBuf.String(), "warn") {
		t.Fatalf("stderr buffer = %q, want it to contain 'warn'", errBuf.String())
	}
}

func TestExecEnvOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses /bin/sh")
	}
	var out bytes.Buffer
	e := &Exec{Stdout: &out, Stderr: &out, Env: []string{"STOCK_PROBE=yes"}}
	got, err := e.Output("sh", "-c", "echo $STOCK_PROBE")
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if strings.TrimSpace(got) != "yes" {
		t.Fatalf("Output() = %q, want 'yes'", got)
	}
}

func TestShellJoin(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b", "c"}, "a b c"},
	}
	for _, tt := range tests {
		if got := shellJoin(tt.in); got != tt.want {
			t.Errorf("shellJoin(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
