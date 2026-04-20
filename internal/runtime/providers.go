package runtime

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ==================== Provider Interface ====================

type SDKProvider interface {
	DownloadURL(version string) (string, error)
	Checksum(version string) string
	PostExtract(dir string, version string) error
}

func getProvider(sdkType SDKType) (SDKProvider, error) {
	reg := GetRegistry(sdkType)
	if reg == nil || reg.Provider == nil {
		return nil, fmt.Errorf("不支持的 SDK 类型: %s", sdkType)
	}
	return reg.Provider, nil
}

// ==================== Go Provider ====================

type GoProvider struct{}

func (p *GoProvider) DownloadURL(version string) (string, error) {
	v := strings.TrimPrefix(version, "go")
	osName := "linux"
	arch := "amd64"

	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "darwin":
		osName = "darwin"
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	}

	ext := ".tar.gz"
	if osName == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("https://go.dev/dl/go%s.%s-%s%s", v, osName, arch, ext), nil
}

func (p *GoProvider) Checksum(version string) string {
	return ""
}

func (p *GoProvider) PostExtract(dir string, version string) error {
	goDir := filepath.Join(dir, "go")
	if info, err := os.Stat(goDir); err == nil && info.IsDir() {
		if err := moveContents(goDir, dir); err != nil {
			return fmt.Errorf("移动 Go 文件失败: %w", err)
		}
		os.RemoveAll(goDir)
	}
	return nil
}

// ==================== Node.js Provider ====================

type NodeProvider struct{}

func (p *NodeProvider) DownloadURL(version string) (string, error) {
	v := strings.TrimPrefix(version, "v")
	osName := "win"
	arch := "x64"

	switch runtime.GOOS {
	case "linux":
		osName = "linux"
	case "darwin":
		osName = "darwin"
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	}

	ext := "zip"
	if runtime.GOOS != "windows" {
		ext = "tar.gz"
	}

	return fmt.Sprintf("https://nodejs.org/dist/v%s/node-v%s-%s-%s.%s", v, v, osName, arch, ext), nil
}

func (p *NodeProvider) Checksum(version string) string {
	return ""
}

func (p *NodeProvider) PostExtract(dir string, version string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "node-") {
			subDir := filepath.Join(dir, e.Name())
			if err := moveContents(subDir, dir); err != nil {
				return fmt.Errorf("移动 Node.js 文件失败: %w", err)
			}
			os.RemoveAll(subDir)
			break
		}
	}
	return nil
}

// ==================== Java (JDK) Provider - Adoptium/Eclipse Temurin ====================

type JavaProvider struct{}

func (p *JavaProvider) DownloadURL(version string) (string, error) {
	osName := "windows"
	arch := "x64"

	switch runtime.GOOS {
	case "linux":
		osName = "linux"
	case "darwin":
		osName = "mac"
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "aarch64"
	}

	// Extract feature version (major) from semver like "21.0.10+7"
	featureVer := version
	if idx := strings.Index(version, "."); idx > 0 {
		featureVer = version[:idx]
	}

	imageType := "jdk"
	ext := "zip"
	if runtime.GOOS != "windows" {
		ext = "tar.gz"
	}

	return fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%s/ga/%s/%s/%s/binary/%s",
		featureVer, osName, arch, imageType, ext,
	), nil
}

func (p *JavaProvider) Checksum(version string) string {
	return ""
}

func (p *JavaProvider) PostExtract(dir string, version string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && (strings.HasPrefix(e.Name(), "jdk-") || strings.HasPrefix(e.Name(), "jbr-")) {
			subDir := filepath.Join(dir, e.Name())
			if err := moveContents(subDir, dir); err != nil {
				return fmt.Errorf("移动 Java 文件失败: %w", err)
			}
			os.RemoveAll(subDir)
			break
		}
	}
	return nil
}

// ==================== Helpers ====================

func moveContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if err := os.Rename(srcPath, dstPath); err != nil {
			if e.IsDir() {
				if err := copyDir(srcPath, dstPath); err != nil {
					return fmt.Errorf("复制目录 %s 失败: %w", srcPath, err)
				}
			} else {
				if err := copyFile(srcPath, dstPath); err != nil {
					return fmt.Errorf("复制文件 %s 失败: %w", srcPath, err)
				}
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()

	_, err = io.Copy(dstF, srcF)
	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func execCommand(cmd string) error {
	c := exec.Command("cmd", "/c", cmd)
	c.Stdout = nil
	c.Stderr = nil
	if err := c.Run(); err != nil {
		return fmt.Errorf("命令执行失败: %s", cmd)
	}
	return nil
}

// ==================== Registration ====================

func init() {
	RegisterSDK(SDKGo, &SDKRegistry{
		Name:     "Go",
		Icon:     "Cyan",
		Provider: &GoProvider{},
		Fetcher:  fetchGoVersions,
	})
	RegisterSDK(SDKNode, &SDKRegistry{
		Name:     "Node.js",
		Icon:     "#68A063",
		Provider: &NodeProvider{},
		Fetcher:  fetchNodeVersions,
	})
	RegisterSDK(SDKJava, &SDKRegistry{
		Name:     "Java (JDK)",
		Icon:     "#ED8B00",
		Provider: &JavaProvider{},
		Fetcher:  fetchJavaVersions,
	})
}
