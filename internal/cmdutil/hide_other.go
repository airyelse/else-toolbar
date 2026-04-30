//go:build !windows

package cmdutil

import "os/exec"

// HideWindow 在非 Windows 平台为空操作。
func HideWindow(cmd *exec.Cmd) {}
