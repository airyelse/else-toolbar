//go:build windows

package cmdutil

import (
	"os/exec"
	"syscall"
)

// HideWindow 设置命令在 Windows 上静默执行，不弹出控制台窗口。
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
