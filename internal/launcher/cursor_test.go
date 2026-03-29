package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWorkspaceFlag(t *testing.T) {
	t.Parallel()

	if got := parseWorkspaceFlag([]string{"--workspace", "/tmp/foo"}); got != "/tmp/foo" {
		t.Errorf("got %q", got)
	}
	if got := parseWorkspaceFlag([]string{"--workspace=/bar"}); got != "/bar" {
		t.Errorf("got %q", got)
	}
	if got := parseWorkspaceFlag([]string{"-p", "x"}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestEnsureSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "source.mcp.json")
	if err := os.WriteFile(src, []byte(`{"mcpServers":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, ".cursor", "mcp.json")

	var b strings.Builder
	if err := ensureSymlink(src, dst, &b); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	absSrc, _ := filepath.Abs(src)
	if filepath.Clean(link) != filepath.Clean(absSrc) {
		t.Errorf("Readlink got %q want %q", link, absSrc)
	}

	if err := ensureSymlink(src, dst, &b); err != nil {
		t.Fatal(err)
	}
}
