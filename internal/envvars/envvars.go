package envvars

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// EnvVar represents a single environment variable.
type EnvVar struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsPath  bool   `json:"isPath"` // true for the PATH variable
}

// EnvResult holds user and system environment variables.
type EnvResult struct {
	System []EnvVar `json:"system"`
	User   []EnvVar `json:"user"`
}

var (
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procExpandEnvironmentStrings = kernel32.NewProc("ExpandEnvironmentStringsW")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procRegSetValueExW           = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW          = advapi32.NewProc("RegDeleteValueW")
	procRegEnumValueW            = advapi32.NewProc("RegEnumValueW")
)

const (
	ERROR_NO_MORE_ITEMS syscall.Errno = 259
)

// regKey returns the top-level key and subkey path for user or system env.
func regKey(system bool) (syscall.Handle, string) {
	if system {
		return syscall.HKEY_LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	}
	return syscall.HKEY_CURRENT_USER, "Environment"
}

// expandEnv expands %VAR% references using the Windows API.
func expandEnv(s string) string {
	src, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return s
	}
	n, _, _ := procExpandEnvironmentStrings.Call(
		uintptr(unsafe.Pointer(src)), 0, 0,
	)
	if n == 0 {
		return s
	}
	buf := make([]uint16, n)
	procExpandEnvironmentStrings.Call(
		uintptr(unsafe.Pointer(src)),
		uintptr(unsafe.Pointer(&buf[0])),
		n,
	)
	return syscall.UTF16ToString(buf)
}

func broadcastSettingChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")
	procSendMessageTimeoutW.Call(
		0xFFFF, // HWND_BROADCAST
		0x001A, // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(env)),
		0x0002, // SMTO_ABORTIFHUNG
		5000,
		0,
	)
}

// openRegKey opens the registry key for reading or writing.
func openRegKey(system bool, write bool) (syscall.Handle, error) {
	topKey, subKey := regKey(system)
	var access uint32 = syscall.KEY_READ
	if write {
		access = syscall.KEY_SET_VALUE
	}
	var key syscall.Handle
	err := syscall.RegOpenKeyEx(topKey, syscall.StringToUTF16Ptr(subKey), 0, access, &key)
	return key, err
}

// ==================== List ====================

// ListEnvVars reads all environment variables from the registry.
func ListEnvVars() *EnvResult {
	return &EnvResult{
		System: readAllEnvVars(true),
		User:   readAllEnvVars(false),
	}
}

func readAllEnvVars(system bool) []EnvVar {
	key, err := openRegKey(system, false)
	if err != nil {
		return nil
	}
	defer syscall.RegCloseKey(key)

	var vars []EnvVar
	// Enumerate values: index 0, 1, 2, ... until ERROR_NO_MORE_ITEMS
	for i := uint32(0); ; i++ {
		var nameBuf [256]uint16
		var nameLen uint32 = uint32(len(nameBuf))
		var valBuf [0x8000]uint16 // 64KB value buffer
		var valLen uint32 = uint32(len(valBuf)) * 2 // in bytes
		var valType uint32

		ret, _, _ := procRegEnumValueW.Call(
			uintptr(key),
			uintptr(i),
			uintptr(unsafe.Pointer(&nameBuf[0])),
			uintptr(unsafe.Pointer(&nameLen)),
			0,
			uintptr(unsafe.Pointer(&valType)),
			uintptr(unsafe.Pointer(&valBuf[0])),
			uintptr(unsafe.Pointer(&valLen)),
		)
		if syscall.Errno(ret) == ERROR_NO_MORE_ITEMS {
			break
		}
		if ret != 0 {
			continue
		}

		name := syscall.UTF16ToString(nameBuf[:nameLen])
		if name == "" {
			continue
		}

		var value string
		switch valType {
		case syscall.REG_SZ, syscall.REG_EXPAND_SZ:
			value = syscall.UTF16ToString(valBuf[:valLen/2])
		default:
			// Skip non-string types (binary, dword, etc.)
			continue
		}

		vars = append(vars, EnvVar{
			Name:   name,
			Value:  value,
			IsPath: strings.EqualFold(name, "PATH"),
		})
	}

	return vars
}

// ==================== Read Single ====================

// GetEnvVar reads a single environment variable from the registry.
func GetEnvVar(name string, system bool) (string, error) {
	key, err := openRegKey(system, false)
	if err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(key)

	valName := syscall.StringToUTF16(name)
	var valType uint32
	var buf [0x8000]uint16
	var bufLen uint32 = 0x8000

	err = syscall.RegQueryValueEx(key, &valName[0], nil, &valType, (*byte)(unsafe.Pointer(&buf[0])), &bufLen)
	if err != nil {
		return "", fmt.Errorf("环境变量 %q 不存在", name)
	}

	return syscall.UTF16ToString(buf[:bufLen/2]), nil
}

// ==================== Write ====================

// SetEnvVar writes a single environment variable to the registry.
func SetEnvVar(name, value string, system bool) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("变量名不能为空")
	}
	// Validate: no null bytes, no equals signs
	if strings.ContainsAny(name, "\x00=") {
		return fmt.Errorf("变量名包含非法字符")
	}

	key, err := openRegKey(system, true)
	if err != nil {
		return fmt.Errorf("无法打开注册表: %w", err)
	}
	defer syscall.RegCloseKey(key)

	valName := syscall.StringToUTF16(name)
	utf16, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}

	ret, _, _ := procRegSetValueExW.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(&valName[0])),
		0,
		uintptr(syscall.REG_EXPAND_SZ),
		uintptr(unsafe.Pointer(&utf16[0])),
		uintptr(len(utf16)*2),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}

	broadcastSettingChange()
	return nil
}

// ==================== Delete ====================

// DeleteEnvVar deletes a single environment variable from the registry.
func DeleteEnvVar(name string, system bool) error {
	key, err := openRegKey(system, true)
	if err != nil {
		return fmt.Errorf("无法打开注册表: %w", err)
	}
	defer syscall.RegCloseKey(key)

	valName := syscall.StringToUTF16(name)
	ret, _, _ := procRegDeleteValueW.Call(uintptr(key), uintptr(unsafe.Pointer(&valName[0])))
	if ret != 0 {
		return syscall.Errno(ret)
	}

	broadcastSettingChange()
	return nil
}

// ==================== Expand ====================

// ExpandValue expands %VAR% references in a value string.
func ExpandValue(value string) string {
	return expandEnv(value)
}
