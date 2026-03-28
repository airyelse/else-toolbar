//go:build windows

package hello

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	helperSHA256       = "76F507459F1CD25C87C37D8EEB4AA0DFAB4A8D2EC3878EED51127DF7414CF834"
	helperFilename     = "windowshellolink.exe"
)

func helperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法定位主程序路径: %w", err)
	}
	absPath := filepath.Join(filepath.Dir(exePath), helperFilename)

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("缺少 Windows Hello 组件: %s", absPath)
		}
		return "", fmt.Errorf("无法访问 Windows Hello 组件: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("Windows Hello 组件路径无效")
	}

	hash, err := fileSHA256(absPath)
	if err != nil {
		return "", fmt.Errorf("无法校验 Windows Hello 组件: %w", err)
	}
	if !strings.EqualFold(hash, helperSHA256) {
		return "", errors.New("Windows Hello 组件校验失败，请重新构建或重新安装程序")
	}

	return absPath, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", hasher.Sum(nil)), nil
}

func runHelper(args ...string) (string, error) {
	path, err := helperPath()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("Windows Hello 组件执行失败: %s", msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func CheckAvailability() (string, error) {
	return runHelper("check")
}

func RequestVerification(message string) (string, error) {
	return runHelper("verify", message)
}

func OpenSettings() error {
	cmd := exec.Command("cmd", "/c", "start", "", "ms-settings:signinoptions")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
