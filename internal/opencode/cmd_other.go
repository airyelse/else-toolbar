//go:build !windows

package opencode

import "os/exec"

func configureOpenCodeCommand(cmd *exec.Cmd) {}
