package main

import (
	"context"
	"else-toolbox/internal/database"
	"else-toolbox/internal/envvars"
	"else-toolbox/internal/models"
	"else-toolbox/internal/opencode"
	"else-toolbox/internal/pathenv"
	"else-toolbox/internal/process"
	"else-toolbox/internal/runtime"
	"else-toolbox/internal/settings"
	"else-toolbox/internal/shell"
	"else-toolbox/internal/vault"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type App struct {
	app            *application.App
	mainWindow     *application.WebviewWindow
	allowCloseOnce atomic.Bool
	dataDir        string
	processManager *process.Manager
	*vault.Vault
}

func NewApp() *App {
	return &App{}
}

func (a *App) SetApp(app *application.App) {
	a.app = app
}

func (a *App) SetMainWindow(window *application.WebviewWindow) {
	a.mainWindow = window
}

// ==================== Close Behavior ====================

func (a *App) ShouldBypassCloseConfirm() bool {
	return a.allowCloseOnce.Load()
}

func (a *App) GetCloseBehavior() string {
	s, err := settings.Load()
	if err != nil {
		return ""
	}
	return s.CloseBehavior
}

func (a *App) SetCloseBehavior(behavior string) error {
	s, err := settings.Load()
	if err != nil {
		return err
	}
	s.CloseBehavior = behavior
	return settings.Save(s)
}

func (a *App) QuitApp() {
	if a.app != nil {
		a.allowCloseOnce.Store(true)
		a.app.Quit()
	}
}

func (a *App) HideWindow() {
	if a.mainWindow != nil {
		a.mainWindow.Hide()
	}
}

func (a *App) emitEvent(name string, data any) {
	if a.app != nil {
		a.app.Event.Emit(name, data)
	}
}

func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	a.dataDir = filepath.Join(homeDir, ".else-toolbox")
	if err := os.MkdirAll(a.dataDir, 0700); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	if err := database.Init(a.dataDir); err != nil {
		return fmt.Errorf("数据库初始化失败: %w", err)
	}
	a.Vault = vault.New(a.dataDir)
	opencode.InitModelCache(a.dataDir)
	opencode.InitPresetStore(a.dataDir)
	opencode.InitAppendPromptStore(a.dataDir)
	opencode.InitMCPSkillCache(a.dataDir)

	a.processManager = process.NewManager(ctx, a.emitEvent)
	return nil
}

func (a *App) ServiceShutdown() error {
	database.Close()
	return nil
}

// ==================== Runtime Manager ====================

func (a *App) ListSDKs() []runtime.SDKInfo {
	return runtime.ListSDKs()
}

func (a *App) InstallSDK(sdkType string, version string) error {
	opts := runtime.InstallOptions{
		SDKType: runtime.SDKType(sdkType),
		Version: version,
		EmitProgress: func(event runtime.ProgressEvent) {
			a.emitEvent("sdk:progress", event)
		},
	}
	return runtime.Install(opts)
}

func (a *App) UninstallSDK(sdkType string, version string) error {
	return runtime.Uninstall(runtime.SDKType(sdkType), version)
}

func (a *App) SwitchSDK(sdkType string, version string) error {
	return runtime.SwitchVersion(runtime.SDKType(sdkType), version)
}

func (a *App) GetRuntimeConfig() *runtime.Config {
	return runtime.GetConfig()
}

func (a *App) SetRuntimeConfig(baseDir string) error {
	return runtime.SaveConfig(&runtime.Config{BaseDir: baseDir})
}

func (a *App) FetchAvailableVersions(sdkType string, force bool) []string {
	return runtime.FetchAvailable(runtime.SDKType(sdkType), force)
}

// ==================== Environment Variables ====================

func (a *App) ListEnvVars() *envvars.EnvResult {
	return envvars.ListEnvVars()
}

func (a *App) GetEnvVar(name string, system bool) (string, error) {
	return envvars.GetEnvVar(name, system)
}

func (a *App) SetEnvVar(name string, value string, system bool) error {
	return envvars.SetEnvVar(name, value, system)
}

func (a *App) DeleteEnvVar(name string, system bool) error {
	return envvars.DeleteEnvVar(name, system)
}

func (a *App) ExpandEnvValue(value string) string {
	return envvars.ExpandValue(value)
}

// ==================== PATH (special management) ====================

func (a *App) GetPathEntries() []*pathenv.PathEntry {
	return pathenv.GetPathEntries()
}

func (a *App) GetPathResult() *pathenv.PathResult {
	return pathenv.GetPathResult()
}

func (a *App) OpenInExplorer(path string) error {
	return shell.OpenExplorer(path)
}

func (a *App) SavePathEntries(paths []string) error {
	return pathenv.SavePathEntries(paths)
}

func (a *App) OpenTerminal(dir string) error {
	return shell.OpenTerminal(dir)
}

func (a *App) ListPathProfiles() []pathenv.PathProfileDTO {
	return pathenv.ListProfiles(a.dataDir)
}

func (a *App) SavePathProfile(dto pathenv.PathProfileDTO) error {
	return pathenv.SaveProfile(a.dataDir, dto)
}

func (a *App) DeletePathProfile(name string) error {
	return pathenv.DeleteProfile(a.dataDir, name)
}

func (a *App) RenamePathProfile(oldName string, newName string) error {
	return pathenv.RenameProfile(a.dataDir, oldName, newName)
}

func (a *App) ApplyPathProfile(profileName string) error {
	profilePaths, err := pathenv.GetProfilePaths(a.dataDir, profileName)
	if err != nil {
		return err
	}
	if len(profilePaths) == 0 {
		return errors.New("profile 为空")
	}

	currentPaths := pathenv.ReadUserPathRaw()
	merged := pathenv.MergeProfile(currentPaths, profilePaths)
	return pathenv.SavePathEntries(merged)
}

func (a *App) PreviewMergeProfile(profileName string) ([]string, error) {
	profilePaths, err := pathenv.GetProfilePaths(a.dataDir, profileName)
	if err != nil {
		return nil, err
	}
	currentPaths := pathenv.ReadUserPathRaw()
	return pathenv.MergeProfile(currentPaths, profilePaths), nil
}

func (a *App) CleanInvalidUserPaths() ([]string, error) {
	return pathenv.CleanInvalidUserPaths()
}

func (a *App) SelectDirectory() (string, error) {
	if a.app == nil {
		return "", errors.New("application 未初始化")
	}
	return a.app.Dialog.OpenFile().
		SetTitle("选择目录").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
}

func (a *App) SelectScriptFile() (string, error) {
	if a.app == nil {
		return "", errors.New("application 未初始化")
	}
	return a.app.Dialog.OpenFile().
		SetTitle("选择脚本文件").
		CanChooseDirectories(false).
		CanChooseFiles(true).
		PromptForSingleSelection()
}

// ==================== OpenCode Main Config ====================

func (a *App) ReadMainConfig() (*opencode.MainConfig, error) {
	return opencode.ReadMainConfig()
}

func (a *App) SaveMainConfig(cfg *opencode.MainConfig) error {
	return opencode.SaveMainConfig(cfg)
}

func (a *App) GetMainConfigPath() (string, error) {
	return opencode.MainConfigPath()
}

// ==================== OpenCode Config (oh-my-opencode-slim) ====================

func (a *App) GetOpenCodeConfig() (*opencode.PresetStoreData, error) {
	return opencode.ReadPresetStore()
}

func (a *App) SaveOpenCodeConfig(store *opencode.PresetStoreData) error {
	return opencode.WritePresetStore(store)
}

func (a *App) DiffPresets() (*opencode.PresetDiff, error) {
	return opencode.DiffPresets()
}

func (a *App) SyncPresetsToConfig() error {
	return opencode.SyncPresetsToConfig()
}

func (a *App) ImportPresetsFromConfig() error {
	return opencode.ImportPresetsFromConfig()
}

func (a *App) GetOpenCodeAgentNames() []string {
	return opencode.AgentNames
}

func (a *App) GetOpenCodeAgentLabels() map[string]string {
	return opencode.AgentLabels
}

func (a *App) GetOpenCodeAgentColors() map[string]string {
	return opencode.AgentColors
}

func (a *App) GetOpenCodeConfigPath() (string, error) {
	return opencode.ConfigPath()
}

func (a *App) OpenOpenCodeConfigDir() error {
	return shell.OpenExplorer(opencode.ConfigDir())
}

func (a *App) FetchAvailableModels() ([]string, error) {
	return opencode.FetchAvailableModels()
}

func (a *App) ForceRefreshModels() ([]string, error) {
	return opencode.ForceRefreshModels()
}

func (a *App) ReadAppendPrompt(agentName string) (string, error) {
	return opencode.ReadAppendPrompt(agentName)
}

func (a *App) WriteAppendPrompt(agentName, content string) error {
	return opencode.WriteAppendPrompt(agentName, content)
}

func (a *App) GetAppendPromptPath(agentName string) (string, error) {
	return opencode.GetAppendPromptPath(agentName)
}

func (a *App) ReadAllAppendPrompts() (map[string]string, error) {
	return opencode.ReadAllAppendPrompts()
}

func (a *App) RestoreAppendPrompts() (int, error) {
	return opencode.SyncAppendPromptsToFiles()
}

func (a *App) ImportAppendPromptsFromFiles() (int, error) {
	return opencode.ImportAppendPromptsFromFiles()
}

func (a *App) DiffAppendPrompts() ([]opencode.AppendPromptDiff, error) {
	return opencode.DiffAppendPrompts(), nil
}

func (a *App) GetAppendPromptStoreStats() (int, error) {
	return opencode.GetAppendPromptStoreStats()
}

func (a *App) GetAppendPromptStoreDir() (string, error) {
	return a.dataDir, nil
}

func (a *App) GetOpenCodeMCPs() ([]opencode.MCPInfo, error) {
	mcps, _, err := opencode.ForceRefreshMCPSkills()
	return mcps, err
}

func (a *App) GetOpenCodeSkills() ([]opencode.SkillInfo, error) {
	_, skills, err := opencode.ForceRefreshMCPSkills()
	return skills, err
}

type MCPSkillResult struct {
	MCPs   []opencode.MCPInfo   `json:"mcps"`
	Skills []opencode.SkillInfo `json:"skills"`
}

func (a *App) FetchMCPSkills() MCPSkillResult {
	mcps, skills := opencode.FetchMCPSkills()
	return MCPSkillResult{MCPs: mcps, Skills: skills}
}

// ==================== Script Console ====================

func (a *App) ListProjects() []models.ProjectDTO {
	var projects []models.Project
	database.DB.Order("`order` ASC, created_at ASC").Find(&projects)
	result := make([]models.ProjectDTO, len(projects))
	for i, p := range projects {
		var count int64
		database.DB.Model(&models.Script{}).Where("project_id = ?", p.ID).Count(&count)
		result[i] = *p.ToDTO(int(count))
	}
	return result
}

func (a *App) CreateProject(name string, notes string) (models.ProjectDTO, error) {
	if name == "" {
		return models.ProjectDTO{}, errors.New("项目名称不能为空")
	}
	var maxOrder int
	database.DB.Model(&models.Project{}).Select("COALESCE(MAX(`order`), -1)").Scan(&maxOrder)

	project := models.Project{
		Name:  name,
		Notes: notes,
		Order: maxOrder + 1,
	}
	if err := database.DB.Create(&project).Error; err != nil {
		return models.ProjectDTO{}, err
	}
	return *project.ToDTO(0), nil
}

func (a *App) UpdateProject(id uint, name string, notes string) error {
	return database.DB.Model(&models.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":  name,
		"notes": notes,
	}).Error
}

func (a *App) DeleteProject(id uint) error {
	var scripts []models.Script
	database.DB.Where("project_id = ?", id).Find(&scripts)
	for _, s := range scripts {
		a.processManager.Stop(s.ID)
	}
	database.DB.Model(&models.Script{}).Where("project_id = ?", id).Update("project_id", nil)
	return database.DB.Delete(&models.Project{}, id).Error
}

func (a *App) ListScripts() []models.ScriptDTO {
	var scripts []models.Script
	database.DB.Preload("Project").Order("created_at DESC").Find(&scripts)
	result := make([]models.ScriptDTO, len(scripts))
	for i, s := range scripts {
		result[i] = *s.ToDTO()
	}
	return result
}

func (a *App) CreateScript(name string, command string, workDir string, envVars string, notes string, elevated bool, keepWindow bool, projectID uint) (models.ScriptDTO, error) {
	script := models.Script{
		Name:       name,
		Command:    command,
		WorkDir:    workDir,
		EnvVars:    envVars,
		Notes:      notes,
		Elevated:   elevated,
		KeepWindow: keepWindow,
	}
	if projectID > 0 {
		script.ProjectID = &projectID
	}
	if err := database.DB.Create(&script).Error; err != nil {
		return models.ScriptDTO{}, err
	}
	return *script.ToDTO(), nil
}

func (a *App) UpdateScript(id uint, name string, command string, workDir string, envVars string, notes string, elevated bool, keepWindow bool, projectID uint) error {
	updates := map[string]interface{}{
		"name":        name,
		"command":     command,
		"work_dir":    workDir,
		"env_vars":    envVars,
		"notes":       notes,
		"elevated":    elevated,
		"keep_window": keepWindow,
	}
	if projectID > 0 {
		updates["project_id"] = projectID
	} else {
		updates["project_id"] = nil
	}
	return database.DB.Model(&models.Script{}).Where("id = ?", id).Updates(updates).Error
}

func (a *App) DeleteScript(id uint) error {
	a.processManager.Stop(id)
	return database.DB.Delete(&models.Script{}, id).Error
}

func (a *App) StartScript(id uint) error {
	var script models.Script
	if err := database.DB.First(&script, id).Error; err != nil {
		return errors.New("脚本不存在")
	}
	return a.processManager.Start(id, script.Command, script.WorkDir, script.EnvVars, script.Elevated, script.KeepWindow)
}

func (a *App) StopScript(id uint) error {
	return a.processManager.Stop(id)
}

func (a *App) RestartScript(id uint) error {
	var script models.Script
	if err := database.DB.First(&script, id).Error; err != nil {
		return errors.New("脚本不存在")
	}
	return a.processManager.Restart(id, script.Command, script.WorkDir, script.EnvVars, script.Elevated, script.KeepWindow)
}

func (a *App) GetScriptStatus(id uint) models.ScriptStatusDTO {
	status, exitCode, pid := a.processManager.GetStatus(id)
	return models.ScriptStatusDTO{
		ID:       id,
		Status:   status,
		ExitCode: exitCode,
		PID:      pid,
	}
}

func (a *App) GetScriptLogs(id uint) []models.LogLineDTO {
	entries := a.processManager.GetLogs(id)
	result := make([]models.LogLineDTO, len(entries))
	for i, e := range entries {
		result[i] = models.LogLineDTO{
			ScriptID:  id,
			Text:      e.Text,
			Source:    e.Source,
			Timestamp: e.Timestamp,
		}
	}
	return result
}

func (a *App) ClearScriptLogs(id uint) error {
	a.processManager.ClearLogs(id)
	return nil
}
