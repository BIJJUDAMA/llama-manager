//go:build windows

package runner

import (
	"os/exec"
)

func configureSysProcAttr(cmd *exec.Cmd) {
	// No setpgid on Windows
}
