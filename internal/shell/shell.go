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
