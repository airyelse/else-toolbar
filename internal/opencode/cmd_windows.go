//go:build windows

package opencode

import (
	"os/exec"
	"syscall"
)

func configureOpenCodeCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
