package commands

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cushycush/store-core/platform"
	"github.com/cushycush/stock/internal/config"
)

func TestRunInstallHappyPath(t *testing.T) {
	brew := &fakeManager{
		name:      "test-install-brew",
		available: true,
		installed: []string{"git"}, // already there
	}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{{
		Name:     "core",
		Managers: map[string][]string{"test-install-brew": {"git", "ripgrep", "bat"}},
	}}}

	var out bytes.Buffer
	ctx := &Context{Cfg: cfg, Info: platform.Info{}, Stdout: &out, Stderr: &out}

	if err := RunInstall(ctx, nil); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if len(brew.installCalls) != 1 {
		t.Fatalf("Install call count = %d, want 1", len(brew.installCalls))
	}
	want := []string{"ripgrep", "bat"} // git was already installed
	if !reflect.DeepEqual(brew.installCalls[0], want) {
		t.Fatalf("Install called with %v, want %v", brew.installCalls[0], want)
	}

	s := out.String()
	if !strings.Contains(s, "installing 2 package(s)") {
		t.Errorf("stdout = %q, want it to report 'installing 2 package(s)'", s)
	}
}

func TestRunInstallReportsUpToDate(t *testing.T) {
	brew := &fakeManager{
		name:      "test-install-uptodate",
		available: true,
		installed: []string{"git", "ripgrep"},
	}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{{
		Name:     "core",
		Managers: map[string][]string{"test-install-uptodate": {"git", "ripgrep"}},
	}}}

	var out bytes.Buffer
	ctx := &Context{Cfg: cfg, Stdout: &out, Stderr: &out}
	if err := RunInstall(ctx, nil); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	if len(brew.installCalls) != 0 {
		t.Fatalf("Install was called with %v; should be a no-op when everything is installed", brew.installCalls)
	}
	if !strings.Contains(out.String(), "up to date") {
		t.Fatalf("stdout = %q, want it to say 'up to date'", out.String())
	}
}

func TestRunInstallSurfacesManagerError(t *testing.T) {
	brew := &fakeManager{
		name:         "test-install-fails",
		available:    true,
		installError: fmt.Errorf("boom"),
	}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{{
		Name:     "core",
		Managers: map[string][]string{"test-install-fails": {"ripgrep"}},
	}}}

	ctx := &Context{Cfg: cfg, Stdout: discard{}, Stderr: discard{}}
	err := RunInstall(ctx, nil)
	if err == nil {
		t.Fatal("RunInstall() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "test-install-fails") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want it to wrap manager name and cause", err.Error())
	}
}

func TestRunInstallNothingMatchesPlatform(t *testing.T) {
	// All groups filtered out by when: — RunInstall should print a useful
	// message, not error out. This is the common "wrong OS" case.
	brew := &fakeManager{name: "test-install-no-match", available: true}
	registerFakes(t, brew)

	cfg := &config.File{Groups: []config.Group{{
		Name:     "mac-only",
		Managers: map[string][]string{"test-install-no-match": {"ripgrep"}},
		// when: matches mac, test runs on non-mac via the info below
	}}}
	cfg.Groups[0].When = nil // leave nil to make the group apply, but no available manager
	cfg.Groups = append(cfg.Groups, config.Group{Name: "empty"})

	// Trick: make the group's only manager unavailable.
	brew.available = false

	var out bytes.Buffer
	ctx := &Context{Cfg: cfg, Stdout: &out, Stderr: &out}
	if err := RunInstall(ctx, nil); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if !strings.Contains(out.String(), "nothing to install") {
		t.Fatalf("stdout = %q, want it to include 'nothing to install'", out.String())
	}
}

func TestRunInstallRejectsUnknownGroup(t *testing.T) {
	cfg := &config.File{Groups: []config.Group{{Name: "core"}}}
	ctx := &Context{Cfg: cfg, Stdout: discard{}, Stderr: discard{}}
	err := RunInstall(ctx, []string{"typo"})
	if err == nil {
		t.Fatal("RunInstall(unknown group) returned nil error")
	}
}
