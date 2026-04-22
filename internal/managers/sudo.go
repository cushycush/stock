package managers

import (
	"fmt"
	"os"
)

// isRoot reports whether the current process is running as root (uid 0).
// On Windows this always returns false — we never auto-sudo there.
func isRoot() bool {
	return os.Geteuid() == 0
}

// errNoSudo builds a clear error explaining that a manager needs root and
// sudo isn't available. Callers surface this to the user without wrapping.
func errNoSudo(action, args string) error {
	return fmt.Errorf("%s requires root and sudo was not found; rerun as root or: %s %s", action, action, args)
}
