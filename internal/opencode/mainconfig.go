package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MainConfig represents the main opencode.json
type MainConfig struct {
	Schema     string                       `json:"$schema,omitempty"`
	Model      string                       `json:"model,omitempty"`
	SmallModel string                       `json:"small_model,omitempty"`
	Plugin     []string                     `json:"plugin,omitempty"`
	Agent      map[string]MainAgentConfig   `json:"agent,omitempty"`
	Provider   map[string]MainProviderConfig `json:"provider,omitempty"`
	MCP        map[string]interface{}       `json:"mcp,omitempty"`
}

// MainAgentConfig represents an agent configuration in the main config
type MainAgentConfig struct {
	Disable bool `json:"disable,omitempty"`
}

// MainProviderConfig represents a provider configuration in the main config
type MainProviderConfig struct {
	Options map[string]interface{} `json:"options,omitempty"`
}

// MainConfigPath returns the path to the main opencode.json file
func MainConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
}

// ReadMainConfig reads and parses the main opencode.json file.
// Returns nil if the file doesn't exist (not an error).
func ReadMainConfig() (*MainConfig, error) {
	path, err := MainConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg MainConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// Ensure maps are not nil to prevent panics
	if cfg.Agent == nil {
		cfg.Agent = make(map[string]MainAgentConfig)
	}
	if cfg.Provider == nil {
		cfg.Provider = make(map[string]MainProviderConfig)
	}
	if cfg.MCP == nil {
		cfg.MCP = make(map[string]interface{})
	}
	if cfg.Plugin == nil {
		cfg.Plugin = []string{}
	}

	return &cfg, nil
}

// SaveMainConfig writes the MainConfig to the opencode.json file
func SaveMainConfig(cfg *MainConfig) error {
	path, err := MainConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// GetProviderNames returns a sorted list of provider names from the config
func GetProviderNames(cfg *MainConfig) []string {
	if cfg == nil || cfg.Provider == nil {
		return []string{}
	}

	names := make([]string, 0, len(cfg.Provider))
	for name := range cfg.Provider {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetProviderAPIKey safely retrieves the API key for a provider from its options
func GetProviderAPIKey(cfg *MainConfig, providerName string) string {
	if cfg == nil || cfg.Provider == nil {
		return ""
	}

	provider, exists := cfg.Provider[providerName]
	if !exists || provider.Options == nil {
		return ""
	}

	apiKey, ok := provider.Options["apiKey"]
	if !ok {
		return ""
	}

	if keyStr, ok := apiKey.(string); ok {
		return keyStr
	}
	return ""
}
