//go:build windows

package daemon

import (
	"os/exec"
)

func detachProcess(cmd *exec.Cmd) {
	// Windows-specific detachment logic if needed.
	// For now, we do nothing as the daemon handles its own backgrounding.
}
