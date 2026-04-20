package runtime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ==================== Types ====================

// opMu serializes install/uninstall/switch operations to prevent concurrent conflicts.
var opMu sync.Mutex

type SDKType string

const (
	SDKGo     SDKType = "go"
	SDKNode   SDKType = "nodejs"
	SDKJava   SDKType = "java"
)

type SDKInfo struct {
	Type     SDKType `json:"type"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	Installed []SDKVersion `json:"installed"`
	Current  string  `json:"current"`
}

type SDKVersion struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	Active  bool   `json:"active"`
}

type ProgressEvent struct {
	Phase   string `json:"phase"`   // download, extract, switch
	Message string `json:"message"`
	Percent int    `json:"percent"`
}

// ==================== Config ====================

type Config struct {
	BaseDir string `json:"baseDir"`
}

func defaultBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".else-toolbox", "runtimes")
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".else-toolbox", "runtime.json")
}

func baseDir() string {
	cfg, err := loadConfig()
	if err != nil || cfg.BaseDir == "" {
		return defaultBaseDir()
	}
	return cfg.BaseDir
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	os.MkdirAll(filepath.Dir(configPath()), 0700)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

func GetConfig() *Config {
	cfg, _ := loadConfig()
	if cfg == nil {
		return &Config{BaseDir: defaultBaseDir()}
	}
	if cfg.BaseDir == "" {
		cfg.BaseDir = defaultBaseDir()
	}
	return cfg
}

func sdkDir(sdkType SDKType) string {
	d := filepath.Join(baseDir(), string(sdkType))
	os.MkdirAll(d, 0700)
	return d
}

func symlinkPath(sdkType SDKType) string {
	return filepath.Join(baseDir(), string(sdkType), "current")
}

func versionDir(sdkType SDKType, version string) string {
	return filepath.Join(sdkDir(sdkType), version)
}

func validateVersion(version string) error {
	if version == "" {
		return errors.New("版本号不能为空")
	}
	if strings.ContainsAny(version, `/\`) {
		return errors.New("版本号不能包含路径分隔符")
	}
	if strings.Contains(version, "..") {
		return errors.New("版本号不能包含 ..")
	}
	if filepath.Base(version) != version {
		return errors.New("版本号不能包含目录组件")
	}
	return nil
}

// ==================== Core Operations ====================

func ListSDKs() []SDKInfo {
	list := make([]SDKInfo, 0, len(registry))
	for _, t := range RegisteredSDKs() {
		list = append(list, listVersions(t))
	}
	return list
}

func listVersions(sdkType SDKType) SDKInfo {
	dir := sdkDir(sdkType)
	entries, _ := os.ReadDir(dir)

	reg := GetRegistry(sdkType)
	displayName := string(sdkType)
	icon := "#6366f1"
	if reg != nil {
		displayName = reg.Name
		icon = reg.Icon
	}

	info := SDKInfo{
		Type: sdkType,
		Name: displayName,
		Icon: icon,
	}

	current := resolveCurrent(sdkType)
	info.Current = current

	for _, e := range entries {
		if !e.IsDir() || e.Name() == "current" {
			continue
		}
		v := SDKVersion{
			Version: e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			Active:  e.Name() == current,
		}
		info.Installed = append(info.Installed, v)
	}
	sortVersions(info.Installed)
	return info
}

func resolveCurrent(sdkType SDKType) string {
	link := symlinkPath(sdkType)
	target, err := os.Readlink(link)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

func sortVersions(versions []SDKVersion) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})
}

func compareVersions(a, b string) int {
	// Simple semver-like comparison
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		ca := strings.TrimLeft(pa[i], "0123456789")[0:]
		cb := strings.TrimLeft(pb[i], "0123456789")[0:]
		if ca != cb {
			// numeric vs non-numeric
			return 0
		}
		// numeric comparison
		ia, ib := 0, 0
		fmt.Sscanf(pa[i], "%d", &ia)
		fmt.Sscanf(pb[i], "%d", &ib)
		if ia != ib {
			return ia - ib
		}
	}
	return len(pa) - len(pb)
}

func SwitchVersion(sdkType SDKType, version string) error {
	opMu.Lock()
	defer opMu.Unlock()

	if err := validateVersion(version); err != nil {
		return err
	}

	vDir := versionDir(sdkType, version)
	if _, err := os.Stat(vDir); os.IsNotExist(err) {
		return fmt.Errorf("版本 %s 未安装", version)
	}

	link := symlinkPath(sdkType)
	os.Remove(link)

	// On Windows, os.Symlink requires admin. Use junction instead.
	if runtime.GOOS == "windows" {
		return createJunction(link, vDir)
	}
	return os.Symlink(vDir, link)
}

func Uninstall(sdkType SDKType, version string) error {
	opMu.Lock()
	defer opMu.Unlock()

	if err := validateVersion(version); err != nil {
		return err
	}

	current := resolveCurrent(sdkType)
	vDir := versionDir(sdkType, version)
	if _, err := os.Stat(vDir); os.IsNotExist(err) {
		return fmt.Errorf("版本 %s 未安装", version)
	}

	if err := os.RemoveAll(vDir); err != nil {
		return err
	}

	if version == current {
		link := symlinkPath(sdkType)
		os.Remove(link)
	}
	return nil
}

// ==================== Install ====================

type InstallOptions struct {
	SDKType  SDKType
	Version  string
	Ctx      context.Context // Wails context for emitting events
}

func Install(opts InstallOptions) error {
	opMu.Lock()
	defer opMu.Unlock()

	if err := validateVersion(opts.Version); err != nil {
		return err
	}

	provider, err := getProvider(opts.SDKType)
	if err != nil {
		return err
	}

	vDir := versionDir(opts.SDKType, opts.Version)
	if _, err := os.Stat(vDir); err == nil {
		return fmt.Errorf("版本 %s 已安装", opts.Version)
	}

	emit := func(phase, msg string, pct int) {
		if opts.Ctx != nil {
			wailsRuntime.EventsEmit(opts.Ctx, "sdk:progress", ProgressEvent{
				Phase: phase, Message: msg, Percent: pct,
			})
		}
	}

	// Download
	url, err := provider.DownloadURL(opts.Version)
	if err != nil {
		return err
	}

	emit("download", "正在下载...", 0)
	archivePath, err := downloadFile(url, string(opts.SDKType), opts.Version, func(pct int) {
		emit("download", fmt.Sprintf("正在下载... %d%%", pct), pct)
	})
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(archivePath)

	// Verify checksum
	if sha := provider.Checksum(opts.Version); sha != "" {
		emit("download", "校验文件...", 99)
		if err := verifyChecksum(archivePath, sha); err != nil {
			return fmt.Errorf("校验失败: %w", err)
		}
	}

	// Extract
	emit("extract", "正在解压...", 0)
	if err := extract(archivePath, vDir, opts.SDKType, opts.Version, func(pct int) {
		emit("extract", fmt.Sprintf("正在解压... %d%%", pct), pct)
	}); err != nil {
		os.RemoveAll(vDir)
		return fmt.Errorf("解压失败: %w", err)
	}

	// Post-extract hook
	if hook := provider.PostExtract; hook != nil {
		if err := hook(vDir, opts.Version); err != nil {
			return err
		}
	}

	// Auto-switch if no current version (non-fatal)
	current := resolveCurrent(opts.SDKType)
	if current == "" {
		if err := SwitchVersion(opts.SDKType, opts.Version); err != nil {
			emit("done", "安装完成（自动切换失败，请手动切换）", 100)
		} else {
			emit("done", fmt.Sprintf("已自动切换到 %s", opts.Version), 100)
		}
	} else {
		emit("done", "安装完成", 100)
	}
	return nil
}

// ==================== Download ====================

func downloadFile(url, sdkType, version string, onProgress func(int)) (string, error) {
	cacheDir := filepath.Join(baseDir(), "cache")
	os.MkdirAll(cacheDir, 0700)

	ext := filepath.Ext(url)
	if ext == ".gz" && strings.HasSuffix(url, ".tar.gz") {
		ext = ".tar.gz"
	}
	if ext == "" {
		ext = ".zip"
	}

	fileName := fmt.Sprintf("%s-%s%s", sdkType, version, ext)
	destPath := filepath.Join(cacheDir, fileName)

	// Check cache
	if f, err := os.Stat(destPath); err == nil && f.Size() > 0 {
		return destPath, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ElseToolbox/1.0")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Write to temp file, rename on success
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	var total int64 = resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32 * 1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpPath)
				return "", fmt.Errorf("写入失败: %w", writeErr)
			}
			downloaded += int64(n)
			if total > 0 && onProgress != nil {
				onProgress(int(downloaded * 100 / total))
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			os.Remove(tmpPath)
			return "", fmt.Errorf("下载中断: %w", readErr)
		}
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("关闭文件失败: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("重命名失败: %w", err)
	}

	return destPath, nil
}

func verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("SHA256 不匹配: expected %s, got %s", expected, actual)
	}
	return nil
}

// ==================== Extract ====================

func extract(archivePath, destDir string, sdkType SDKType, version string, onProgress func(int)) error {
	os.MkdirAll(destDir, 0700)

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGz(archivePath, destDir, onProgress)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, destDir, onProgress)
	default:
		return fmt.Errorf("不支持的压缩格式: %s", archivePath)
	}
}

func extractTarGz(path, dest string, onProgress func(int)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		// Security: prevent path traversal
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}

	if onProgress != nil {
		onProgress(100)
	}
	return nil
}

func extractZip(path, dest string, onProgress func(int)) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	var totalSize int64
	for _, f := range r.File {
		totalSize += f.FileInfo().Size()
	}
	var processed int64

	for _, f := range r.File {
		target := filepath.Join(dest, filepath.Clean(f.Name))
		// Security: prevent path traversal
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		if err := out.Close(); err != nil {
			rc.Close()
			return err
		}
		rc.Close()

		processed += f.FileInfo().Size()
		if totalSize > 0 && onProgress != nil {
			onProgress(int(processed * 100 / totalSize))
		}
	}
	return nil
}

// ==================== Windows Junction ====================

func createJunction(link, target string) error {
	// os.Readlink works on junctions, but os.Symlink requires SeCreateSymbolicLinkPrivilege.
	// Use cmd /c mklink /J instead which works without admin on NTFS.
	link = filepath.Clean(link)
	target = filepath.Clean(target)

	cmd := fmt.Sprintf(`cmd /c mklink /J "%s" "%s"`, link, target)
	return runCommand(cmd)
}

func removeJunction(path string) error {
	path = filepath.Clean(path)
	cmd := fmt.Sprintf(`cmd /c rmdir "%s"`, path)
	return runCommand(cmd)
}

// ==================== Helpers ====================

func runCommand(cmd string) error {
	// Simple command execution on Windows
	return execCommand(cmd)
}
