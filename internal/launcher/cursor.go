package launcher

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RunCursor creates workspace/.cursor/mcp.json as a symlink to the resolved
// shared MCP config file (Claude Code JSON), then exec's the Cursor CLI.
// Cursor reads the same schema from .cursor/mcp.json as documented at
// https://cursor.com/docs/mcp
func RunCursor(sourceMCPPath string, extraArgs []string, errW io.Writer) error {
	workspace, err := resolveWorkspaceDir(extraArgs)
	if err != nil {
		return err
	}

	linkPath := filepath.Join(workspace, ".cursor", "mcp.json")
	if err := ensureSymlink(sourceMCPPath, linkPath, errW); err != nil {
		return err
	}

	env := buildEnv(nil)
	return execCursorAgent(extraArgs, env)
}

// resolveWorkspaceDir returns the directory used for .cursor/mcp.json.
// It honors --workspace in extraArgs (passed through to the Cursor CLI); otherwise os.Getwd.
func resolveWorkspaceDir(extraArgs []string) (string, error) {
	if w := parseWorkspaceFlag(extraArgs); w != "" {
		return filepath.Abs(w)
	}
	return os.Getwd()
}

func parseWorkspaceFlag(extra []string) string {
	for i, a := range extra {
		if a == "--workspace" && i+1 < len(extra) {
			return extra[i+1]
		}
		if strings.HasPrefix(a, "--workspace=") {
			return strings.TrimPrefix(a, "--workspace=")
		}
	}
	return ""
}

func ensureSymlink(src, dst string, errW io.Writer) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolve MCP config path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create .cursor directory: %w", err)
	}

	fi, statErr := os.Lstat(dst)
	if statErr == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			cur, err := os.Readlink(dst)
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", dst, err)
			}
			curAbs := cur
			if !filepath.IsAbs(cur) {
				curAbs = filepath.Join(filepath.Dir(dst), cur)
			}
			curAbs, err = filepath.Abs(curAbs)
			if err != nil {
				return err
			}
			if filepath.Clean(curAbs) == filepath.Clean(absSrc) {
				return nil
			}
			_, _ = fmt.Fprintf(errW, "warning: replacing symlink %s (was %s)\n", dst, cur)
		} else {
			_, _ = fmt.Fprintf(errW, "warning: replacing %s with symlink to %s\n", dst, absSrc)
		}
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("remove %s: %w", dst, err)
		}
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat %s: %w", dst, statErr)
	}

	if err := os.Symlink(absSrc, dst); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", dst, absSrc, err)
	}
	return nil
}
