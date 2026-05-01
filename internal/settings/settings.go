package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	CloseBehavior string `json:"closeBehavior"` // "minimize" | "quit", empty means "ask"
}

func loadPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".else-toolbox", "settings.json"), nil
}

func Load() (*Settings, error) {
	path, err := loadPath()
	if err != nil {
		return nil, err
	}

	s := &Settings{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}

	return s, nil
}

func Save(s *Settings) error {
	path, err := loadPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
