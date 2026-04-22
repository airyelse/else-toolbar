package shell

import (
	"os/exec"
	"path/filepath"
)

func OpenExplorer(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return exec.Command("explorer", abs).Start()
}

// OpenTerminal launches Windows Terminal (wt.exe), falls back to cmd if unavailable.
func OpenTerminal(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	// Try Windows Terminal first
	if err := exec.Command("wt", "-d", abs).Start(); err == nil {
		return nil
	}
	// Fallback to cmd
	return exec.Command("cmd", "/c", "start", "cmd", "/k", "cd", "/d", abs).Start()
}
