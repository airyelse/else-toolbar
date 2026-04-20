package pathenv

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ==================== Types ====================

type PathProfile struct {
	Name      string   `json:"name"`
	Paths     []string `json:"paths"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

type PathProfileDTO struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

// ==================== Storage ====================

var (
	profileMu sync.Mutex
)

func profilePath(dataDir string) string {
	return filepath.Join(dataDir, "path_profiles.json")
}

func loadProfiles(dataDir string) ([]PathProfile, error) {
	data, err := os.ReadFile(profilePath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var profiles []PathProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func saveProfiles(dataDir string, profiles []PathProfile) error {
	os.MkdirAll(dataDir, 0700)
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(profilePath(dataDir), data, 0644)
}

// ==================== CRUD ====================

func ListProfiles(dataDir string) []PathProfileDTO {
	profileMu.Lock()
	defer profileMu.Unlock()

	profiles, err := loadProfiles(dataDir)
	if err != nil || profiles == nil {
		return nil
	}

	result := make([]PathProfileDTO, len(profiles))
	for i, p := range profiles {
		result[i] = PathProfileDTO{Name: p.Name, Paths: p.Paths}
	}
	return result
}

func SaveProfile(dataDir string, dto PathProfileDTO) error {
	profileMu.Lock()
	defer profileMu.Unlock()

	if strings.TrimSpace(dto.Name) == "" {
		return errors.New("profile 名称不能为空")
	}

	profiles, err := loadProfiles(dataDir)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	for i, p := range profiles {
		if strings.EqualFold(p.Name, dto.Name) {
			profiles[i].Paths = dto.Paths
			profiles[i].UpdatedAt = now
			return saveProfiles(dataDir, profiles)
		}
	}

	profiles = append(profiles, PathProfile{
		Name:      dto.Name,
		Paths:     dto.Paths,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return saveProfiles(dataDir, profiles)
}

func DeleteProfile(dataDir string, name string) error {
	profileMu.Lock()
	defer profileMu.Unlock()

	profiles, err := loadProfiles(dataDir)
	if err != nil {
		return err
	}

	for i, p := range profiles {
		if strings.EqualFold(p.Name, name) {
			profiles = append(profiles[:i], profiles[i+1:]...)
			return saveProfiles(dataDir, profiles)
		}
	}
	return errors.New("profile 不存在")
}

func RenameProfile(dataDir string, oldName, newName string) error {
	profileMu.Lock()
	defer profileMu.Unlock()

	if strings.TrimSpace(newName) == "" {
		return errors.New("profile 名称不能为空")
	}

	profiles, err := loadProfiles(dataDir)
	if err != nil {
		return err
	}

	for i, p := range profiles {
		if strings.EqualFold(p.Name, newName) && !strings.EqualFold(oldName, newName) {
			return errors.New("已存在同名 profile")
		}
		if strings.EqualFold(p.Name, oldName) {
			profiles[i].Name = newName
			profiles[i].UpdatedAt = time.Now().Unix()
			return saveProfiles(dataDir, profiles)
		}
	}
	return errors.New("profile 不存在")
}

// GetProfilePaths returns the paths for a given profile name.
func GetProfilePaths(dataDir string, name string) ([]string, error) {
	profileMu.Lock()
	defer profileMu.Unlock()

	profiles, err := loadProfiles(dataDir)
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if strings.EqualFold(p.Name, name) {
			return p.Paths, nil
		}
	}
	return nil, errors.New("profile 不存在")
}

// ==================== Merge Logic ====================

// MergeProfile prepends profile paths to existing user PATH, deduplicating case-insensitively.
// Profile entries take priority — if a path already exists in the current PATH, it's moved to the front.
func MergeProfile(currentPaths []string, profilePaths []string) []string {
	if len(profilePaths) == 0 {
		return currentPaths
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(profilePaths)+len(currentPaths))

	// First: add profile paths (deduplicated among themselves)
	for _, p := range profilePaths {
		lower := strings.ToLower(p)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, p)
		}
	}

	// Then: append existing paths that aren't in the profile
	for _, p := range currentPaths {
		lower := strings.ToLower(p)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, p)
		}
	}

	return result
}
