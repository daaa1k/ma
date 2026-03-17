//go:build !windows

package launcher

import (
	"fmt"
	"os/exec"
	"syscall"
)

// execTool replaces the current process with binary using syscall.Exec so that
// the tool receives the same terminal, signals, and exit codes as if invoked
// directly by the user.
func execTool(binary string, argv []string, env []string) error {
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("tool %q not found in PATH: %w", binary, err)
	}
	// argv passed to syscall.Exec must include the program name as argv[0].
	args := append([]string{binary}, argv...)
	return syscall.Exec(path, args, env)
}
