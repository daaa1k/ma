//go:build !windows

package launcher

import (
	"syscall"
)

func execCursorAgent(argv []string, env []string) error {
	argv0, path, err := resolveCursorBinary()
	if err != nil {
		return err
	}
	args := append([]string{argv0}, argv...)
	return syscall.Exec(path, args, env)
}
