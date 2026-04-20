package main

import (
	"context"
	"else-toolbox/internal/database"
	"else-toolbox/internal/envvars"
	"else-toolbox/internal/pathenv"
	"else-toolbox/internal/runtime"
	"else-toolbox/internal/opencode"
	"else-toolbox/internal/shell"
	"else-toolbox/internal/vault"
	"errors"
	"log"
	"os"
	"path/filepath"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	dataDir string
	*vault.Vault
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	homeDir, _ := os.UserHomeDir()
	a.dataDir = filepath.Join(homeDir, ".else-toolbox")
	os.MkdirAll(a.dataDir, 0700)

	if err := database.Init(a.dataDir); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	a.Vault = vault.New(a.dataDir)
	opencode.InitModelCache(a.dataDir)
}

func (a *App) shutdown(ctx context.Context) {
	database.Close()
}

// ==================== Runtime Manager ====================

func (a *App) ListSDKs() []runtime.SDKInfo {
	return runtime.ListSDKs()
}

func (a *App) InstallSDK(sdkType string, version string) error {
	opts := runtime.InstallOptions{
		SDKType: runtime.SDKType(sdkType),
		Version:  version,
		Ctx:      a.ctx,
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

	// Read current user PATH from registry
	currentPaths := pathenv.ReadUserPathRaw()

	// Merge: profile paths first, then existing non-duplicate paths
	merged := pathenv.MergeProfile(currentPaths, profilePaths)
	return pathenv.SavePathEntries(merged)
}

func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择目录",
	})
}

// ==================== OpenCode Config ====================

func (a *App) GetOpenCodeConfig() (*opencode.Config, error) {
	return opencode.ReadConfig()
}

func (a *App) SaveOpenCodeConfig(cfg *opencode.Config) error {
	return opencode.SaveConfig(cfg)
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

func (a *App) FetchAvailableModels() ([]string, error) {
	return opencode.FetchAvailableModels()
}

func (a *App) ForceRefreshModels() ([]string, error) {
	return opencode.ForceRefreshModels()
}
