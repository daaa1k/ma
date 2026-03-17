//go:build windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// execTool runs binary as a child process on Windows (syscall.Exec is not
// available). The exit code of the child is propagated via os.Exit.
func execTool(binary string, argv []string, env []string) error {
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("tool %q not found in PATH: %w", binary, err)
	}
	cmd := exec.Command(path, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
