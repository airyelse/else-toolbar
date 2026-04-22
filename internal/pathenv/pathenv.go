package pathenv

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type PathEntry struct {
	RawPath string `json:"rawPath"`
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	IsDir   bool   `json:"isDir"`
}

type PathResult struct {
	System []*PathEntry `json:"system"`
	User   []*PathEntry `json:"user"`
}

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procExpandEnvironmentStrings = kernel32.NewProc("ExpandEnvironmentStringsW")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procRegSetValueExW           = advapi32.NewProc("RegSetValueExW")
)

func expandEnv(s string) string {
	src, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return s
	}
	n, _, _ := procExpandEnvironmentStrings.Call(
		uintptr(unsafe.Pointer(src)),
		0, 0,
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

func GetPathResult() *PathResult {
	return &PathResult{
		System: readRegPath(true),
		User:   readRegPath(false),
	}
}

func GetPathEntries() []*PathEntry {
	pathStr := os.Getenv("PATH")
	return parsePathStr(pathStr)
}

func parsePathStr(pathStr string) []*PathEntry {
	if pathStr == "" {
		return nil
	}

	entries := strings.Split(pathStr, string(os.PathListSeparator))
	result := make([]*PathEntry, 0, len(entries))

	for _, p := range entries {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		expanded := expandEnv(p)
		entry := &PathEntry{RawPath: p, Path: expanded}
		info, err := os.Stat(expanded)
		if err == nil {
			entry.Exists = true
			entry.IsDir = info.IsDir()
		}
		result = append(result, entry)
	}

	return result
}

func readRegPath(system bool) []*PathEntry {
	var key syscall.Handle
	var topKey syscall.Handle = syscall.HKEY_LOCAL_MACHINE
	subKey := `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	if !system {
		topKey = syscall.HKEY_CURRENT_USER
		subKey = "Environment"
	}

	err := syscall.RegOpenKeyEx(topKey, syscall.StringToUTF16Ptr(subKey), 0, syscall.KEY_READ, &key)
	if err != nil {
		return nil
	}
	defer syscall.RegCloseKey(key)

	var bufLen uint32 = 0x8000
	buf := make([]uint16, bufLen)
	valName := syscall.StringToUTF16("Path")

	valType := uint32(0)
	err = syscall.RegQueryValueEx(key, &valName[0], nil, &valType, (*byte)(unsafe.Pointer(&buf[0])), &bufLen)
	if err != nil {
		return nil
	}

	pathStr := syscall.UTF16ToString(buf[:bufLen/2])
	return parsePathStr(pathStr)
}

// ValidatePathEntries checks for empty strings and duplicate paths (case-insensitive).
func ValidatePathEntries(paths []string) error {
	seen := make(map[string]int)
	for i, p := range paths {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("PATH 条目不能为空（第 %d 项）", i+1)
		}
		lower := strings.ToLower(p)
		if prev, ok := seen[lower]; ok {
			return fmt.Errorf("PATH 条目重复: %q 出现在第 %d 项和第 %d 项", paths[prev], prev+1, i+1)
		}
		seen[lower] = i
	}
	return nil
}

func SavePathEntries(paths []string) error {
	if err := ValidatePathEntries(paths); err != nil {
		return err
	}

	var key syscall.Handle
	var topKey syscall.Handle = syscall.HKEY_CURRENT_USER
	subKey := "Environment"

	err := syscall.RegOpenKeyEx(topKey, syscall.StringToUTF16Ptr(subKey), 0, syscall.KEY_SET_VALUE, &key)
	if err != nil {
		return err
	}
	defer syscall.RegCloseKey(key)

	pathStr := strings.Join(paths, string(os.PathListSeparator))
	utf16, err := syscall.UTF16FromString(pathStr)
	if err != nil {
		return err
	}

	valName := syscall.StringToUTF16("Path")
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

// CleanInvalidUserPaths removes non-existent or non-directory entries from user PATH.
// Returns the list of removed paths.
func CleanInvalidUserPaths() ([]string, error) {
	entries := readRegPath(false)
	kept := make([]string, 0, len(entries))
	removed := make([]string, 0)

	for _, e := range entries {
		if e.Exists && e.IsDir {
			kept = append(kept, e.RawPath)
		} else {
			removed = append(removed, e.RawPath)
		}
	}

	if len(removed) == 0 {
		return nil, nil
	}

	if err := SavePathEntries(kept); err != nil {
		return nil, err
	}
	return removed, nil
}

// ReadUserPathRaw returns the raw (unexpanded) user PATH entries from the registry.
func ReadUserPathRaw() []string {
	entries := readRegPath(false)
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.RawPath
	}
	return paths
}
