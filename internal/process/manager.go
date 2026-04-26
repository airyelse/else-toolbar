package process

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"runtime"
)

// ManagedProcess 管理的进程
type ManagedProcess struct {
	Cmd      *exec.Cmd
	Cancel   context.CancelFunc
	PID      int
	Elevated bool
	Job      windows.Handle
	Handle   windows.Handle
}

type processStatus struct {
	status   string
	exitCode int
	pid      int
}

// LogEntry 日志条目
type LogEntry struct {
	Text      string `json:"text"`
	Source    string `json:"source"` // stdout or stderr
	Timestamp string `json:"timestamp"`
}

// Manager 进程管理器
type Manager struct {
	mu        sync.RWMutex
	ctx       context.Context
	processes map[uint]*ManagedProcess
	statuses  map[uint]processStatus
	logs      map[uint][]LogEntry // 每个 script 的日志缓冲
	logMax    int                 // 每个 script 最大日志条数
}

var instance *Manager
var once sync.Once

// GetManager 获取单例管理器
func GetManager(ctx context.Context) *Manager {
	once.Do(func() {
		instance = &Manager{
			ctx:       ctx,
			processes: make(map[uint]*ManagedProcess),
			statuses:  make(map[uint]processStatus),
			logs:      make(map[uint][]LogEntry),
			logMax:    2000,
		}
	})
	instance.ctx = ctx
	return instance
}

// Start 启动脚本进程
func (m *Manager) Start(scriptID uint, command string, workDir string, envVarsJSON string, elevated bool, keepWindow bool) error {
	if err := m.Stop(scriptID); err != nil {
		return err
	}
	m.mu.Lock()
	// 清空旧日志
	m.logs[scriptID] = nil
	m.statuses[scriptID] = processStatus{status: "starting", exitCode: 0, pid: 0}
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(m.ctx)
	if elevated && runtime.GOOS == "windows" {
		m.addLog(scriptID, LogEntry{Text: "[系统] 脚本将以管理员权限在独立窗口中运行，stdout/stderr 不可捕获。", Source: "system", Timestamp: time.Now().Format("15:04:05")})
		commandToRun := command
		if envPrefix := buildEnvPrefix(envVarsJSON); envPrefix != "" {
			commandToRun = envPrefix + command
		}
		handle, pid, err := shellExecuteRunAs(commandToRun, workDir, keepWindow)
		if err != nil {
			cancel()
			return err
		}
		m.mu.Lock()
		m.processes[scriptID] = &ManagedProcess{Cancel: cancel, Elevated: true, PID: pid, Handle: handle}
		m.statuses[scriptID] = processStatus{status: "running", exitCode: 0, pid: pid}
		m.mu.Unlock()
		m.emitStatus(scriptID, "running", 0, pid)
		go m.waitForElevatedExit(scriptID, handle)
		return nil
	}

	// 在 Windows 上使用 cmd /C 来执行命令
	isWindows := true
	var cmd *exec.Cmd
	if isWindows {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		// 解析命令
		parts := strings.Fields(command)
		if len(parts) == 0 {
			cancel()
			return fmt.Errorf("命令为空")
		}
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	// 设置环境变量
	if envVarsJSON != "" {
		var envList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(envVarsJSON), &envList); err == nil {
			for _, env := range envList {
				if env.Key != "" {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Key, env.Value))
				}
			}
		}
	}

	// 捕获 stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("创建 stdout 管道失败: %w", err)
	}

	// 捕获 stderr
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("创建 stderr 管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("启动进程失败: %w", err)
	}

	var job windows.Handle
	if runtime.GOOS == "windows" {
		if h, err := createKillOnCloseJob(); err == nil {
			job = h
			if err := assignProcessToJob(job, cmd.Process.Pid); err != nil {
				_ = windows.CloseHandle(job)
				job = 0
			}
		}
	}

	m.mu.Lock()
	m.processes[scriptID] = &ManagedProcess{
		Cmd:    cmd,
		Cancel: cancel,
		PID:    cmd.Process.Pid,
		Job:    job,
	}
	m.statuses[scriptID] = processStatus{status: "running", exitCode: 0, pid: cmd.Process.Pid}
	m.mu.Unlock()

	// 发送状态变更事件
	m.emitStatus(scriptID, "running", 0, cmd.Process.Pid)

	// 读取 stdout
	go m.readStream(scriptID, stdoutPipe, "stdout")

	// 读取 stderr
	go m.readStream(scriptID, stderrPipe, "stderr")

	// 等待进程结束
	go func() {
		err := cmd.Wait()
		exitCode := 0
		status := "exited"
		logText := "[系统] 脚本已正常退出。"
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
			status = "exited"
			logText = fmt.Sprintf("[系统] 脚本异常退出（exitCode=%d）。", exitCode)
		}
		m.mu.Lock()
		currentProcess, ok := m.processes[scriptID]
		if ok && currentProcess.Cmd == cmd {
			delete(m.processes, scriptID)
		}
		current := m.statuses[scriptID]
		if current.status != "stopped" {
			m.statuses[scriptID] = processStatus{status: status, exitCode: exitCode, pid: 0}
		}
		m.mu.Unlock()
		if job != 0 {
			_ = windows.CloseHandle(job)
		}
		if current.status != "stopped" {
			m.addLog(scriptID, LogEntry{Text: logText, Source: "system", Timestamp: time.Now().Format("15:04:05")})
			m.emitStatus(scriptID, status, exitCode, 0)
		}
	}()

	return nil
}

// Stop 停止脚本进程
func (m *Manager) Stop(scriptID uint) error {
	m.mu.RLock()
	p, ok := m.processes[scriptID]
	m.mu.RUnlock()

	if !ok {
		return nil // 未运行
	}

	if p.Cancel != nil {
		p.Cancel()
	}

	if p.Elevated {
		m.mu.Lock()
		delete(m.processes, scriptID)
		m.statuses[scriptID] = processStatus{status: "stopped", exitCode: 0, pid: 0}
		m.mu.Unlock()
		m.addLog(scriptID, LogEntry{
			Text:      "[系统] 管理员脚本已从面板标记为停止；如管理员窗口仍在运行，请手动关闭。",
			Source:    "system",
			Timestamp: time.Now().Format("15:04:05"),
		})
		m.emitStatus(scriptID, "stopped", 0, 0)
		return nil
	}

	if runtime.GOOS == "windows" {
		if p.Job != 0 {
			_ = terminateJob(p.Job)
			_ = windows.CloseHandle(p.Job)
			p.Job = 0
		} else if p.PID > 0 {
			_ = killProcessTree(p.PID)
			_ = waitForProcessExit(p.PID, 3*time.Second)
		}
	}

	m.mu.Lock()
	delete(m.processes, scriptID)
	m.statuses[scriptID] = processStatus{status: "stopped", exitCode: 0, pid: 0}
	m.mu.Unlock()

	// 发送停止日志
	m.addLog(scriptID, LogEntry{
		Text:      "[进程已停止]",
		Source:    "system",
		Timestamp: time.Now().Format("15:04:05"),
	})

	return nil
}

// Restart 重启脚本
func (m *Manager) Restart(scriptID uint, command string, workDir string, envVarsJSON string, elevated bool, keepWindow bool) error {
	_ = m.Stop(scriptID)
	if runtime.GOOS == "windows" {
		// Give Windows a short, deterministic grace period for handle/port release.
		time.Sleep(250 * time.Millisecond)
	}
	return m.Start(scriptID, command, workDir, envVarsJSON, elevated, keepWindow)
}

// GetStatus 获取进程状态
func (m *Manager) GetStatus(scriptID uint) (status string, exitCode int, pid int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.processes[scriptID]; ok {
		return "running", 0, p.PID
	}
	if s, ok := m.statuses[scriptID]; ok {
		return s.status, s.exitCode, s.pid
	}
	return "stopped", 0, 0
}

// GetLogs 获取日志（返回副本）
func (m *Manager) GetLogs(scriptID uint) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := m.logs[scriptID]
	if logs == nil {
		return nil
	}

	// 返回副本
	result := make([]LogEntry, len(logs))
	copy(result, logs)
	return result
}

// ClearLogs 清空日志
func (m *Manager) ClearLogs(scriptID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[scriptID] = nil
}

// IsRunning 检查是否运行中
func (m *Manager) IsRunning(scriptID uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.processes[scriptID]
	return ok
}

// readStream 读取进程输出流
func (m *Manager) readStream(scriptID uint, reader io.Reader, source string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			text := string(buf[:n])
			// 按行分割
			lines := strings.Split(text, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				m.addLog(scriptID, LogEntry{
					Text:      line,
					Source:    source,
					Timestamp: time.Now().Format("15:04:05"),
				})
			}
		}
		if err != nil {
			break
		}
	}
}

// addLog 添加日志条目
func (m *Manager) addLog(scriptID uint, entry LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.logs[scriptID] == nil {
		m.logs[scriptID] = make([]LogEntry, 0, 100)
	}

	// 限制日志条数
	if len(m.logs[scriptID]) >= m.logMax {
		m.logs[scriptID] = m.logs[scriptID][len(m.logs[scriptID])-m.logMax+100:]
	}

	m.logs[scriptID] = append(m.logs[scriptID], entry)

	// 发送事件到前端
	if m.ctx != nil {
		wailsRuntime.EventsEmit(m.ctx, "script:log", map[string]interface{}{
			"id":        scriptID,
			"text":      entry.Text,
			"source":    entry.Source,
			"timestamp": entry.Timestamp,
		})
	}
}

// emitStatus 发送状态变更事件
func (m *Manager) emitStatus(scriptID uint, status string, exitCode int, pid int) {
	if m.ctx != nil {
		wailsRuntime.EventsEmit(m.ctx, "script:status", map[string]interface{}{
			"id":       scriptID,
			"status":   status,
			"exitCode": exitCode,
			"pid":      pid,
		})
	}
}

func shellExecuteRunAs(command string, workDir string, keepWindow bool) (windows.Handle, int, error) {
	mod := syscall.NewLazyDLL("shell32.dll")
	proc := mod.NewProc("ShellExecuteExW")
	op := syscall.StringToUTF16Ptr("runas")
	file := syscall.StringToUTF16Ptr("cmd.exe")
	var params *uint16
	cmdLine := command
	if workDir != "" {
		absDir, err := filepath.Abs(workDir)
		if err != nil {
			absDir = workDir
		}
		cmdLine = fmt.Sprintf("cd /d \"%s\" && %s", absDir, command)
	}
	switch {
	case keepWindow:
		params = syscall.StringToUTF16Ptr("/K " + cmdLine)
	default:
		params = syscall.StringToUTF16Ptr("/C " + cmdLine)
	}
	type shellExecuteInfo struct {
		CbSize        uint32
		FMask         uint32
		Hwnd          uintptr
		LPVerb        *uint16
		LPCFile       *uint16
		LPCParameters *uint16
		LPCDirectory  *uint16
		NShow         int32
		HInstApp      uintptr
		LPIDList      uintptr
		LPCClass      *uint16
		HkeyClass     uintptr
		DwHotKey      uint32
		HIcon         uintptr
		HProcess      windows.Handle
	}
	const seeMaskNoCloseProcess = 0x00000040
	const seeMaskFlagNoUi = 0x00000400
	info := shellExecuteInfo{
		CbSize:        uint32(unsafe.Sizeof(shellExecuteInfo{})),
		FMask:         seeMaskNoCloseProcess | seeMaskFlagNoUi,
		LPVerb:        op,
		LPCFile:       file,
		LPCParameters: params,
		NShow:         1,
	}
	r, _, err := proc.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, 0, fmt.Errorf("启动管理员窗口失败: %v", err)
	}
	pid := 0
	if info.HProcess != 0 {
		if hpid, procErr := windows.GetProcessId(info.HProcess); procErr == nil {
			pid = int(hpid)
		}
	}
	return info.HProcess, pid, nil
}

func (m *Manager) waitForElevatedExit(scriptID uint, handle windows.Handle) {
	if handle == 0 {
		return
	}
	_, _ = windows.WaitForSingleObject(handle, windows.INFINITE)
	_ = windows.CloseHandle(handle)
	m.mu.Lock()
	current, ok := m.processes[scriptID]
	if ok && current.Handle == handle {
		delete(m.processes, scriptID)
	}
	status := m.statuses[scriptID]
	if status.status != "stopped" {
		m.statuses[scriptID] = processStatus{status: "exited", exitCode: 0, pid: 0}
	}
	m.mu.Unlock()
	if ok && current.Handle == handle && status.status != "stopped" {
		m.addLog(scriptID, LogEntry{Text: "[系统] 管理员脚本已正常退出。", Source: "system", Timestamp: time.Now().Format("15:04:05")})
		m.emitStatus(scriptID, "exited", 0, 0)
	}
}

func killProcessTree(pid int) error {
	cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/T", "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		probe := exec.Command("cmd", "/C", fmt.Sprintf("tasklist /FI \"PID eq %d\" /NH", pid))
		out, err := probe.Output()
		if err != nil {
			return nil
		}
		text := string(out)
		if strings.Contains(text, "No tasks are running") || !strings.Contains(text, fmt.Sprintf(" %d ", pid)) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("进程 %d 未在超时时间内退出", pid)
}

func buildEnvPrefix(envVarsJSON string) string {
	if envVarsJSON == "" {
		return ""
	}

	var envList []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(envVarsJSON), &envList); err != nil {
		return ""
	}

	var parts []string
	for _, env := range envList {
		if env.Key == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("set \"%s=%s\" && ", env.Key, env.Value))
	}
	return strings.Join(parts, "")
}

func createKillOnCloseJob() (windows.Handle, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(h)
		return 0, err
	}
	return h, nil
}

func assignProcessToJob(job windows.Handle, pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(job, h)
}

func terminateJob(job windows.Handle) error {
	return windows.TerminateJobObject(job, 1)
}
