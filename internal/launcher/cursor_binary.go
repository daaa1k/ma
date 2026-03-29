package launcher

import (
	"fmt"
	"os/exec"
)

// resolveCursorBinary finds cursor-agent, falling back to agent.
func resolveCursorBinary() (argv0 string, resolvedPath string, err error) {
	for _, name := range []string{"cursor-agent", "agent"} {
		if p, err := exec.LookPath(name); err == nil {
			return name, p, nil
		}
	}
	return "", "", fmt.Errorf("cursor CLI not found in PATH (tried cursor-agent, agent)")
}
