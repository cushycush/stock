// Package env exposes the shared STORE_* environment contract. Hook scripts
// invoked by either store or stock see the same variables, so a single hook
// works across both tools.
package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cushycush/stock/internal/platform"
)

// Vars returns the STORE_* environment additions for the given root + platform.
// Returned in KEY=VALUE form, ready to append to os.Environ().
func Vars(root string, info platform.Info) []string {
	return []string{
		kv("STORE_ROOT", root),
		kv("STORE_PLATFORM_OS", info.OS),
		kv("STORE_PLATFORM_ARCH", info.Arch),
		kv("STORE_PLATFORM_DISTRO", info.Distro),
		kv("STORE_PLATFORM_HOSTNAME", info.Hostname),
		kv("STORE_PLATFORM_SHELL", info.Shell),
		kv("STORE_PLATFORM_WSL", strconv.FormatBool(info.WSL)),
	}
}

// Apply merges Vars(...) into the current process environment. Use this before
// invoking subcommands that should see STORE_* (like hooks or `store` itself).
func Apply(root string, info platform.Info) {
	for _, kv := range Vars(root, info) {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				os.Setenv(kv[:i], kv[i+1:])
				break
			}
		}
	}
}

func kv(k, v string) string { return fmt.Sprintf("%s=%s", k, v) }
