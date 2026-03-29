//go:build windows

package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func execCursorAgent(argv []string, env []string) error {
	_, path, err := resolveCursorBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(path, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("run cursor CLI: %w", err)
	}
	os.Exit(0)
	return nil
}
