// Package hooks runs scripts placed under <root>/.store/hooks/.
// Layout mirrors store's hook layout so the same directory serves both tools.
package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cushycush/stock/internal/env"
	"github.com/cushycush/stock/internal/platform"
)

// Run invokes <root>/.store/hooks/<name> if it exists and is executable. A
// missing hook is not an error — hooks are always optional.
func Run(root, name string, info platform.Info) error {
	path := filepath.Join(root, ".store", "hooks", name)
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&0111 == 0 {
		return fmt.Errorf("hook %s exists but is not executable", path)
	}
	cmd := execCommand(path)
	cmd.Env = append(os.Environ(), env.Vars(root, info)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook %s failed: %w", name, err)
	}
	return nil
}
