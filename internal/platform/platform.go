// Package platform detects OS, arch, distro, hostname, shell, and WSL.
// Mirrors store's platform layer; will migrate to store-core once that module lands.
package platform

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Info struct {
	OS       string // linux, darwin, windows
	Arch     string // amd64, arm64, ...
	Distro   string // debian, ubuntu, arch, fedora, alpine, opensuse, "" on non-linux
	Hostname string
	Shell    string // basename of $SHELL
	WSL      bool
}

func Detect() Info {
	info := Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		info.Shell = filepath.Base(sh)
	}
	if info.OS == "linux" {
		info.Distro = detectLinuxDistro()
		info.WSL = detectWSL()
	}
	return info
}

func detectLinuxDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if v, ok := strings.CutPrefix(line, "ID="); ok {
			return strings.Trim(v, `"`)
		}
	}
	return ""
}

func detectWSL() bool {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	s := strings.ToLower(string(b))
	return strings.Contains(s, "microsoft") || strings.Contains(s, "wsl")
}
