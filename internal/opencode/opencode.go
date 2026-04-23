package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// AgentConfig 单个 Agent 的配置
type AgentConfig struct {
	Model       interface{} `json:"model"`
	Variant     string      `json:"variant,omitempty"`
	Skills      []string    `json:"skills,omitempty"`
	Mcps        []string    `json:"mcps,omitempty"`
	Temperature *float64    `json:"temperature,omitempty"`
}

// Preset 预设配置
type Preset struct {
	Orchestrator *AgentConfig `json:"orchestrator,omitempty"`
	Oracle       *AgentConfig `json:"oracle,omitempty"`
	Librarian    *AgentConfig `json:"librarian,omitempty"`
	Explorer     *AgentConfig `json:"explorer,omitempty"`
	Designer     *AgentConfig `json:"designer,omitempty"`
	Fixer        *AgentConfig `json:"fixer,omitempty"`
}

// Config oh-my-opencode-slim 完整配置
type Config struct {
	Schema  string             `json:"$schema,omitempty"`
	Preset  string             `json:"preset"`
	Presets map[string]*Preset `json:"presets,omitempty"`
}

// AgentNames 所有 Agent 名称（用于 UI 遍历）
var AgentNames = []string{"orchestrator", "oracle", "librarian", "explorer", "designer", "fixer"}

// AgentLabels Agent 中文名称
var AgentLabels = map[string]string{
	"orchestrator": "Orchestrator (主编排)",
	"oracle":       "Oracle (架构师)",
	"librarian":    "Librarian (文档研究员)",
	"explorer":     "Explorer (代码搜索)",
	"designer":     "Designer (UI 设计)",
	"fixer":        "Fixer (快速实现)",
}

// AgentColors Agent 标识色
var AgentColors = map[string]string{
	"orchestrator": "#6366f1",
	"oracle":       "#f59e0b",
	"librarian":    "#10b981",
	"explorer":     "#06b6d4",
	"designer":     "#ec4899",
	"fixer":        "#64748b",
}

// GetPresetAgent 获取指定预设中某个 Agent 的配置
func (c *Config) GetPresetAgent(presetName, agentName string) *AgentConfig {
	preset := c.Presets[presetName]
	if preset == nil {
		return nil
	}
	switch agentName {
	case "orchestrator":
		return preset.Orchestrator
	case "oracle":
		return preset.Oracle
	case "librarian":
		return preset.Librarian
	case "explorer":
		return preset.Explorer
	case "designer":
		return preset.Designer
	case "fixer":
		return preset.Fixer
	default:
		return nil
	}
}

// SetPresetAgent 设置指定预设中某个 Agent 的配置
func (c *Config) SetPresetAgent(presetName, agentName string, cfg *AgentConfig) {
	preset := c.Presets[presetName]
	if preset == nil {
		preset = &Preset{}
		c.Presets[presetName] = preset
	}
	switch agentName {
	case "orchestrator":
		preset.Orchestrator = cfg
	case "oracle":
		preset.Oracle = cfg
	case "librarian":
		preset.Librarian = cfg
	case "explorer":
		preset.Explorer = cfg
	case "designer":
		preset.Designer = cfg
	case "fixer":
		preset.Fixer = cfg
	}
}

// ModelToString 将 Model 字段统一转为字符串（兼容 string 和 []string）
func ModelToString(model interface{}) string {
	if model == nil {
		return ""
	}
	switch v := model.(type) {
	case string:
		return v
	case []interface{}:
		result := ""
		for i, item := range v {
			if i > 0 {
				result += ", "
			}
			result += fmt.Sprintf("%v", item)
		}
		return result
	case []string:
		result := ""
		for i, item := range v {
			if i > 0 {
				result += ", "
			}
			result += item
		}
		return result
	default:
		return fmt.Sprintf("%v", v)
	}
}

// StringToModel 将字符串转为 Model 字段（包含逗号时转为数组）
func StringToModel(s string) interface{} {
	if s == "" {
		return ""
	}
	// 检查是否已经是 JSON 数组
	if len(s) > 0 && s[0] == '[' {
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return arr
		}
	}
	return s
}

// ConfigPath 返回配置文件路径
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode")
}

func ConfigPath() (string, error) {
	return filepath.Join(ConfigDir(), "oh-my-opencode-slim.json"), nil
}

// ReadConfig 读取配置文件
func ReadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 确保 Presets map 不为 nil
	if cfg.Presets == nil {
		cfg.Presets = make(map[string]*Preset)
	}

	return &cfg, nil
}

// SaveConfig 保存配置文件
func SaveConfig(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(path, append(data, '\n'), 0644)
}

// CreatePreset 创建新预设（基于现有预设的模板）
func (c *Config) CreatePreset(name string) error {
	if _, exists := c.Presets[name]; exists {
		return fmt.Errorf("预设「%s」已存在", name)
	}

	// 如果有活跃预设，复制其配置作为模板
	preset := &Preset{}
	if currentPreset, ok := c.Presets[c.Preset]; ok && currentPreset != nil {
		// 深拷贝
		if currentPreset.Orchestrator != nil {
			preset.Orchestrator = &AgentConfig{
				Model: currentPreset.Orchestrator.Model,
				Skills: append([]string{}, currentPreset.Orchestrator.Skills...),
				Mcps:   append([]string{}, currentPreset.Orchestrator.Mcps...),
			}
		}
	}

	c.Presets[name] = preset
	return nil
}

// DeletePreset 删除预设
func (c *Config) DeletePreset(name string) error {
	if name == c.Preset {
		return fmt.Errorf("不能删除当前活跃预设")
	}
	if _, exists := c.Presets[name]; !exists {
		return fmt.Errorf("预设「%s」不存在", name)
	}
	delete(c.Presets, name)
	return nil
}

// RenamePreset 重命名预设
func (c *Config) RenamePreset(oldName, newName string) error {
	if _, exists := c.Presets[oldName]; !exists {
		return fmt.Errorf("预设「%s」不存在", oldName)
	}
	if _, exists := c.Presets[newName]; exists {
		return fmt.Errorf("预设「%s」已存在", newName)
	}
	if newName == "" {
		return fmt.Errorf("预设名称不能为空")
	}
	c.Presets[newName] = c.Presets[oldName]
	delete(c.Presets, oldName)
	if c.Preset == oldName {
		c.Preset = newName
	}
	return nil
}

// ==================== Preset Store ====================
// 以程序持久存储 (opencode_presets.json) 为准，配置文件为同步目标

// presetStoreDir 由 InitPresetStore 设置
var presetStoreDir string

// InitPresetStore 初始化持久存储目录
func InitPresetStore(dataDir string) {
	presetStoreDir = dataDir
}

func presetStorePath() string {
	return filepath.Join(presetStoreDir, "opencode_presets.json")
}

// PresetStoreData 持久存储结构
type PresetStoreData struct {
	ActivePreset string             `json:"active_preset"`
	Presets      map[string]*Preset `json:"presets"`
}

func loadPresetStoreFile() (*PresetStoreData, error) {
	data, err := os.ReadFile(presetStorePath())
	if err != nil {
		return nil, err
	}
	var store PresetStoreData
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Presets == nil {
		store.Presets = make(map[string]*Preset)
	}
	return &store, nil
}

func savePresetStoreFile(store *PresetStoreData) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(presetStorePath(), append(data, '\n'), 0644)
}

// ReadPresetStore 从持久存储读取，不存在则从配置文件导入
func ReadPresetStore() (*PresetStoreData, error) {
	store, err := loadPresetStoreFile()
	if err == nil {
		return store, nil
	}

	// 持久存储不存在，从配置文件导入
	cfg, cfgErr := ReadConfig()
	if cfgErr != nil {
		return &PresetStoreData{
			ActivePreset: "default",
			Presets:      make(map[string]*Preset),
		}, nil
	}

	storeData := &PresetStoreData{
		ActivePreset: cfg.Preset,
		Presets:      cfg.Presets,
	}
	if storeData.Presets == nil {
		storeData.Presets = make(map[string]*Preset)
	}
	_ = savePresetStoreFile(storeData)
	return storeData, nil
}

// WritePresetStore 同时写入持久存储和配置文件
func WritePresetStore(store *PresetStoreData) error {
	if err := savePresetStoreFile(store); err != nil {
		return fmt.Errorf("写入持久存储失败: %w", err)
	}

	cfg := &Config{
		Preset:  store.ActivePreset,
		Presets: store.Presets,
	}
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("同步配置文件失败: %w", err)
	}
	return nil
}

// PresetDiff 预设差异
type PresetDiff struct {
	StoreActive string   `json:"store_active"`
	FileActive  string   `json:"file_active"`
	Differences []string `json:"differences"`
}

// DiffPresets 对比持久存储与配置文件的预设差异
func DiffPresets() (*PresetDiff, error) {
	store, err := loadPresetStoreFile()
	if err != nil {
		return nil, err
	}

	cfg, err := ReadConfig()
	if err != nil {
		return nil, err
	}

	diff := &PresetDiff{
		StoreActive: store.ActivePreset,
		FileActive:  cfg.Preset,
	}

	if store.ActivePreset != cfg.Preset {
		diff.Differences = append(diff.Differences,
			fmt.Sprintf("活跃预设不同: 存储「%s」vs 文件「%s」", store.ActivePreset, cfg.Preset))
	}

	// 收集所有预设名
	allNames := make(map[string]bool)
	for name := range store.Presets {
		allNames[name] = true
	}
	for name := range cfg.Presets {
		allNames[name] = true
	}

	for name := range allNames {
		_, inStore := store.Presets[name]
		_, inFile := cfg.Presets[name]
		if inStore && !inFile {
			diff.Differences = append(diff.Differences, fmt.Sprintf("预设「%s」仅存在于存储", name))
		} else if !inStore && inFile {
			diff.Differences = append(diff.Differences, fmt.Sprintf("预设「%s」仅存在于文件", name))
		} else {
			sJSON, _ := json.Marshal(store.Presets[name])
			fJSON, _ := json.Marshal(cfg.Presets[name])
			if string(sJSON) != string(fJSON) {
				diff.Differences = append(diff.Differences, fmt.Sprintf("预设「%s」内容不同", name))
			}
		}
	}

	if len(diff.Differences) == 0 {
		return nil, nil
	}
	return diff, nil
}

// SyncPresetsToConfig 将持久存储覆盖到配置文件
func SyncPresetsToConfig() error {
	store, err := loadPresetStoreFile()
	if err != nil {
		return err
	}
	cfg := &Config{
		Preset:  store.ActivePreset,
		Presets: store.Presets,
	}
	return SaveConfig(cfg)
}

// ImportPresetsFromConfig 将配置文件导入到持久存储
func ImportPresetsFromConfig() error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	store := &PresetStoreData{
		ActivePreset: cfg.Preset,
		Presets:      cfg.Presets,
	}
	if store.Presets == nil {
		store.Presets = make(map[string]*Preset)
	}
	return savePresetStoreFile(store)
}

// ==================== Append Prompt Files ====================
// 以程序持久存储 (append_prompts.json) 为准，.md 文件为运行时同步目标

// appendPromptStoreDir 由 InitAppendPromptStore 设置
var appendPromptStoreDir string

// InitAppendPromptStore 初始化持久存储目录
func InitAppendPromptStore(dataDir string) {
	appendPromptStoreDir = dataDir
}

// appendPromptStorePath 返回持久化 JSON 文件路径
func appendPromptStorePath() string {
	return filepath.Join(appendPromptStoreDir, "append_prompts.json")
}

// appendPromptStoreType 持久存储结构: agent -> content
type appendPromptStoreType map[string]string

func loadAppendPromptStore() appendPromptStoreType {
	data, err := os.ReadFile(appendPromptStorePath())
	if err != nil {
		return make(appendPromptStoreType)
	}
	var store appendPromptStoreType
	if err := json.Unmarshal(data, &store); err != nil {
		return make(appendPromptStoreType)
	}
	return store
}

func saveAppendPromptStore(store appendPromptStoreType) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(appendPromptStorePath(), append(data, '\n'), 0644)
}

// AppendPromptDir 返回 oh-my-opencode-slim 配置目录（_append.md 文件所在目录）
func AppendPromptDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户目录: %w", err)
	}
	return filepath.Join(home, ".config", "opencode", "oh-my-opencode-slim"), nil
}

// readAppendMd 从 .md 文件读取内容（不涉及持久存储）
func readAppendMd(agentName string) string {
	baseDir, err := AppendPromptDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, agentName+"_append.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

// ReadAppendPrompt 以持久存储为准读取指定 agent 的附加提示词
func ReadAppendPrompt(agentName string) (string, error) {
	store := loadAppendPromptStore()
	if content, ok := store[agentName]; ok {
		return content, nil
	}
	return "", nil
}

// WriteAppendPrompt 同时写入持久存储和 .md 文件
func WriteAppendPrompt(agentName, content string) error {
	// 1. 写入持久存储（source of truth）
	store := loadAppendPromptStore()
	store[agentName] = content
	if err := saveAppendPromptStore(store); err != nil {
		return fmt.Errorf("写入持久存储失败: %w", err)
	}

	// 2. 同步写入 .md 文件
	baseDir, err := AppendPromptDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	mdPath := filepath.Join(baseDir, agentName+"_append.md")
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 .md 文件失败: %w", err)
	}
	return nil
}

// GetAppendPromptPath 返回指定 agent 的 _append.md 文件路径（UI 展示用）
func GetAppendPromptPath(agentName string) (string, error) {
	baseDir, err := AppendPromptDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, agentName+"_append.md"), nil
}

// ReadAllAppendPrompts 从持久存储读取所有 agent 的附加提示词
func ReadAllAppendPrompts() (map[string]string, error) {
	store := loadAppendPromptStore()
	result := make(map[string]string)
	for _, name := range AgentNames {
		if content, ok := store[name]; ok {
			result[name] = content
		}
	}
	return result, nil
}

// AppendPromptDiff 单个 agent 的差异
type AppendPromptDiff struct {
	Agent string `json:"agent"`
	Store string `json:"store"` // 持久存储内容
	File  string `json:"file"`  // .md 文件内容
}

// DiffAppendPrompts 对比持久存储和 .md 文件，返回有差异的 agent 列表
func DiffAppendPrompts() []AppendPromptDiff {
	store := loadAppendPromptStore()
	var diffs []AppendPromptDiff

	for _, agentName := range AgentNames {
		storeContent := store[agentName]
		fileContent := readAppendMd(agentName)

		if storeContent != fileContent {
			diffs = append(diffs, AppendPromptDiff{
				Agent: agentName,
				Store: storeContent,
				File:  fileContent,
			})
		}
	}
	return diffs
}

// SyncAppendPromptsToFiles 将持久存储的内容同步到 .md 文件（覆盖）
func SyncAppendPromptsToFiles() (int, error) {
	store := loadAppendPromptStore()

	baseDir, err := AppendPromptDir()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, agentName := range AgentNames {
		content := store[agentName]
		path := filepath.Join(baseDir, agentName+"_append.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return count, fmt.Errorf("写入 %s 失败: %w", path, err)
		}
		count++
	}
	return count, nil
}

// ImportAppendPromptsFromFiles 将 .md 文件内容导入到持久存储（反向同步）
func ImportAppendPromptsFromFiles() (int, error) {
	store := loadAppendPromptStore()

	count := 0
	for _, agentName := range AgentNames {
		content := readAppendMd(agentName)
		store[agentName] = content
		if content != "" {
			count++
		}
	}
	return count, saveAppendPromptStore(store)
}

// GetAppendPromptStoreStats 返回持久存储的概要信息
func GetAppendPromptStoreStats() (int, error) {
	store := loadAppendPromptStore()
	count := 0
	for _, content := range store {
		if content != "" {
			count++
		}
	}
	return count, nil
}

// ==================== MCP & Skill Discovery ====================

// MCPInfo represents a parsed MCP server entry
type MCPInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // "local" or "remote"
	Command string `json:"command"` // for local: display command
	URL     string `json:"url"`     // for remote: the URL
	Source  string `json:"source"`  // "config" or "plugin"
}

// SkillInfo represents a discovered skill
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "config", "plugin", or "agent"
}

// ReadMCPConfig reads MCP servers from the resolved opencode config (includes plugin MCPs)
func ReadMCPConfig() ([]MCPInfo, error) {
	out, err := exec.Command("opencode", "debug", "config").Output()
	if err != nil {
		return nil, fmt.Errorf("执行 opencode debug config 失败: %w", err)
	}

	var cfg struct {
		MCP map[string]struct {
			Type    string   `json:"type"`
			Command []string `json:"command,omitempty"`
			URL     string   `json:"url,omitempty"`
		} `json:"mcp"`
	}

	if err := json.Unmarshal(out, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// Also read raw opencode.json to determine source
	configMCPs := make(map[string]bool)
	configDir := ConfigDir()
	if data, err := os.ReadFile(filepath.Join(configDir, "opencode.json")); err == nil {
		var raw struct {
			MCP map[string]struct{} `json:"mcp"`
		}
		if json.Unmarshal(data, &raw) == nil {
			for name := range raw.MCP {
				configMCPs[name] = true
			}
		}
	}

	var result []MCPInfo
	for name, mcp := range cfg.MCP {
		info := MCPInfo{
			Name:   name,
			Type:   mcp.Type,
			Source: "plugin",
		}
		if configMCPs[name] {
			info.Source = "config"
		}
		if mcp.Type == "local" && len(mcp.Command) > 0 {
			info.Command = strings.Join(mcp.Command, " ")
		}
		if mcp.Type == "remote" {
			info.URL = mcp.URL
		}
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// ReadSkills reads all skills from the resolved opencode debug skill output
func ReadSkills() ([]SkillInfo, error) {
	out, err := exec.Command("opencode", "debug", "skill").Output()
	if err != nil {
		return nil, fmt.Errorf("执行 opencode debug skill 失败: %w", err)
	}

	var skills []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Location    string `json:"location"`
	}

	if err := json.Unmarshal(out, &skills); err != nil {
		return nil, fmt.Errorf("解析 skills 失败: %w", err)
	}

	// Determine source from location
	home, _ := os.UserHomeDir()
	configSkillsDir := filepath.Join(home, ".config", "opencode", "skills")
	result := make([]SkillInfo, 0, len(skills))
	for _, s := range skills {
		source := "plugin"
		if configSkillsDir != "" && strings.HasPrefix(s.Location, configSkillsDir) {
			source = "config"
		} else if strings.Contains(s.Location, ".agents") {
			source = "agent"
		}
		result = append(result, SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			Source:      source,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// ==================== MCP & Skill Cache ====================

const mcpSkillCacheTTL = 5 * time.Minute

var (
	mcpCache     []MCPInfo
	skillCache   []SkillInfo
	mcpSkillTime time.Time
	mcpSkillMu   sync.Mutex
	mcpSkillDir  string
)

// InitMCPSkillCache 初始化缓存目录，从文件加载缓存，后台异步刷新
func InitMCPSkillCache(dataDir string) {
	mcpSkillDir = dataDir
	loadMCPSkillCacheFile()
	go refreshMCPSkills()
}

type mcpSkillCacheFile struct {
	MCPs   []MCPInfo  `json:"mcps"`
	Skills []SkillInfo `json:"skills"`
	Time   int64      `json:"time"`
}

func mcpSkillCacheFilePath() string {
	return filepath.Join(mcpSkillDir, "mcpskills_cache.json")
}

func loadMCPSkillCacheFile() {
	mcpSkillMu.Lock()
	defer mcpSkillMu.Unlock()

	data, err := os.ReadFile(mcpSkillCacheFilePath())
	if err != nil {
		return
	}
	var cf mcpSkillCacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return
	}
	mcpCache = cf.MCPs
	skillCache = cf.Skills
	mcpSkillTime = time.Unix(cf.Time, 0)
}

func saveMCPSkillCacheFile(mcps []MCPInfo, skills []SkillInfo) {
	cf := mcpSkillCacheFile{
		MCPs:   mcps,
		Skills: skills,
		Time:   time.Now().Unix(),
	}
	data, _ := json.Marshal(cf)
	os.WriteFile(mcpSkillCacheFilePath(), data, 0644)
}

func refreshMCPSkills() ([]MCPInfo, []SkillInfo, error) {
	var mcps []MCPInfo
	var skills []SkillInfo
	var mcpErr, skillErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		mcps, mcpErr = ReadMCPConfig()
	}()
	go func() {
		defer wg.Done()
		skills, skillErr = ReadSkills()
	}()
	wg.Wait()

	if mcpErr != nil && skillErr != nil {
		return nil, nil, fmt.Errorf("刷新 MCP 失败: %w; 刷新 Skills 失败: %w", mcpErr, skillErr)
	}

	if mcpErr != nil {
		return nil, skills, mcpErr
	}
	if skillErr != nil {
		return mcps, nil, skillErr
	}

	mcpSkillMu.Lock()
	mcpCache = mcps
	skillCache = skills
	mcpSkillTime = time.Now()
	mcpSkillMu.Unlock()

	go saveMCPSkillCacheFile(mcps, skills)

	return mcps, skills, nil
}

// FetchMCPSkills 获取 MCP 和 Skills，优先使用缓存，过期则后台静默刷新
func FetchMCPSkills() ([]MCPInfo, []SkillInfo) {
	mcpSkillMu.Lock()
	if mcpCache != nil && skillCache != nil && time.Since(mcpSkillTime) < mcpSkillCacheTTL {
		mcps := make([]MCPInfo, len(mcpCache))
		skills := make([]SkillInfo, len(skillCache))
		copy(mcps, mcpCache)
		copy(skills, skillCache)
		mcpSkillMu.Unlock()
		go refreshMCPSkills()
		return mcps, skills
	}
	stale := mcpCache
	staleSkills := skillCache
	mcpSkillMu.Unlock()

	// 过期或无缓存，后台刷新
	go refreshMCPSkills()

	// 返回过期数据（如果有）
	if stale != nil && staleSkills != nil {
		mcps := make([]MCPInfo, len(stale))
		skills := make([]SkillInfo, len(staleSkills))
		copy(mcps, stale)
		copy(skills, staleSkills)
		return mcps, skills
	}
	return nil, nil
}

// ForceRefreshMCPSkills 强制刷新 MCP 和 Skills 缓存
func ForceRefreshMCPSkills() ([]MCPInfo, []SkillInfo, error) {
	return refreshMCPSkills()
}

// ==================== Model Discovery ====================

const modelCacheTTL = 30 * time.Minute

var (
	modelCache     []string
	modelCacheTime time.Time
	modelCacheMu   sync.Mutex
	modelCacheDir  string // 由 InitModelCache 设置
)

// InitModelCache 初始化缓存目录，从文件加载缓存，后台异步刷新
func InitModelCache(dataDir string) {
	modelCacheDir = dataDir
	loadModelCacheFile()
	go refreshModels() // 启动后自动后台刷新
}

func modelCacheFilePath() string {
	return filepath.Join(modelCacheDir, "models_cache.json")
}

type modelCacheFile struct {
	Models []string `json:"models"`
	Time   int64    `json:"time"`
}

func loadModelCacheFile() {
	modelCacheMu.Lock()
	defer modelCacheMu.Unlock()

	data, err := os.ReadFile(modelCacheFilePath())
	if err != nil {
		return
	}
	var cf modelCacheFile
	if err := json.Unmarshal(data, &cf); err != nil || len(cf.Models) == 0 {
		return
	}
	modelCache = cf.Models
	modelCacheTime = time.Unix(cf.Time, 0)
}

func saveModelCacheFile(models []string) {
	cf := modelCacheFile{
		Models: models,
		Time:   time.Now().Unix(),
	}
	data, _ := json.Marshal(cf)
	os.WriteFile(modelCacheFilePath(), data, 0644)
}

// FetchAvailableModels 获取可用模型，优先使用缓存，过期则调用 opencode models --refresh 刷新
func FetchAvailableModels() ([]string, error) {
	modelCacheMu.Lock()
	if modelCache != nil && time.Since(modelCacheTime) < modelCacheTTL {
		result := make([]string, len(modelCache))
		copy(result, modelCache)
		modelCacheMu.Unlock()
		return result, nil
	}
	modelCacheMu.Unlock()

	models, err := refreshModels()
	if err != nil {
		// 刷新失败时返回过期缓存（如果有）
		modelCacheMu.Lock()
		if modelCache != nil {
			result := make([]string, len(modelCache))
			copy(result, modelCache)
			modelCacheMu.Unlock()
			return result, nil
		}
		modelCacheMu.Unlock()
		return nil, err
	}
	return models, nil
}

// ForceRefreshModels 强制刷新模型缓存
func ForceRefreshModels() ([]string, error) {
	return refreshModels()
}

func refreshModels() ([]string, error) {
	out, err := exec.Command("opencode", "models", "--refresh").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行 opencode models 失败: %w", err)
	}

	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Models cache") {
			continue
		}
		if len(line) > 0 && line[0] < 32 {
			continue
		}
		models = append(models, line)
	}
	sort.Strings(models)

	modelCacheMu.Lock()
	modelCache = models
	modelCacheTime = time.Now()
	modelCacheMu.Unlock()

	go saveModelCacheFile(models) // 异步写文件，不阻塞

	return models, nil
}
