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

	"golang.org/x/sys/windows"
	"runtime"
)

type EventEmitter func(name string, data any)

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
	childPid int // 实际子进程 PID（如 node/python），0 表示未知
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
	emit      EventEmitter
	processes map[uint]*ManagedProcess
	statuses  map[uint]processStatus
	logs      map[uint][]LogEntry // 每个 script 的日志缓冲
	logMax    int                 // 每个 script 最大日志条数
}

// NewManager 创建新的进程管理器
func NewManager(ctx context.Context, emit EventEmitter) *Manager {
	return &Manager{
		ctx:       ctx,
		emit:      emit,
		processes: make(map[uint]*ManagedProcess),
		statuses:  make(map[uint]processStatus),
		logs:      make(map[uint][]LogEntry),
		logMax:    2000,
	}
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
		m.emitStatus(scriptID, "running", 0, pid, 0)
		go m.waitForElevatedExit(scriptID, handle)
		return nil
	}

	// 在 Windows 上使用 cmd /C 来执行命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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

	// 发送状态变更事件（childPid 暂为 0，后台探测）
	m.emitStatus(scriptID, "running", 0, cmd.Process.Pid, 0)

	// 后台异步探测子进程 PID
	go m.detectChildPid(scriptID, cmd.Process.Pid)

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
			m.emitStatus(scriptID, status, exitCode, 0, 0)
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
		m.emitStatus(scriptID, "stopped", 0, 0, 0)
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

	m.emitStatus(scriptID, "stopped", 0, 0, 0)
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
func (m *Manager) GetStatus(scriptID uint) (status string, exitCode int, pid int, childPid int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.processes[scriptID]; ok {
		s := m.statuses[scriptID]
		return "running", 0, p.PID, s.childPid
	}
	if s, ok := m.statuses[scriptID]; ok {
		return s.status, s.exitCode, s.pid, s.childPid
	}
	return "stopped", 0, 0, 0
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

// detectChildPid 异步探测 root PID 的直接子进程 PID 并更新状态
func (m *Manager) detectChildPid(scriptID uint, rootPID int) {
	// 等待一小段时间让子进程启动
	time.Sleep(800 * time.Millisecond)

	// 检查进程是否仍在运行
	m.mu.RLock()
	_, ok := m.processes[scriptID]
	m.mu.RUnlock()
	if !ok {
		return
	}

	// 用 wmic 获取 rootPID 的直接子进程
	cmd := exec.Command("wmic", "process", "get", "ParentProcessId,ProcessId", "/format:csv")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return
	}

	childPIDs := parseDirectChildren(string(out), rootPID)
	if len(childPIDs) == 0 {
		return
	}

	// 取第一个子进程作为代表性 childPid
	child := childPIDs[0]

	m.mu.Lock()
	// 再次确认进程仍在运行且 PID 匹配
	p, ok := m.processes[scriptID]
	if !ok {
		m.mu.Unlock()
		return
	}
	if p.PID != rootPID {
		m.mu.Unlock()
		return
	}
	s := m.statuses[scriptID]
	if s.status != "running" {
		m.mu.Unlock()
		return
	}
	m.statuses[scriptID] = processStatus{
		status:   s.status,
		exitCode: s.exitCode,
		pid:      s.pid,
		childPid: child,
	}
	m.mu.Unlock()

	m.emitStatus(scriptID, "running", 0, rootPID, child)
}

// parseDirectChildren 从 wmic 输出中解析指定 parentPID 的直接子进程
func parseDirectChildren(output string, parentPID int) []int {
	var children []int
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") || !strings.HasPrefix(line, ",") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		ppid := atoiSafe(strings.TrimSpace(parts[1]))
		cpid := atoiSafe(strings.TrimSpace(parts[2]))
		if ppid == parentPID && cpid > 0 {
			children = append(children, cpid)
		}
	}
	return children
}

// GetPorts 获取指定脚本进程及其后代进程监听的端口
func (m *Manager) GetPorts(scriptID uint) []string {
	m.mu.RLock()
	p, ok := m.processes[scriptID]
	m.mu.RUnlock()

	if !ok || p.PID <= 0 {
		return nil
	}

	// 收集该脚本进程及其所有后代进程的 PID
	pids := collectDescendantPIDs(p.PID)
	if len(pids) == 0 {
		return nil
	}

	// 用 netstat 获取所有 LISTENING 端口
	ports := findListeningPorts(pids)
	return ports
}

// collectDescendantPIDs 收集 rootPID 及其所有后代进程 PID
func collectDescendantPIDs(rootPID int) []int {
	// 使用 wmic 递归获取进程树: wmic process get ParentProcessId,ProcessId
	cmd := exec.Command("wmic", "process", "get", "ParentProcessId,ProcessId", "/format:csv")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		// fallback: 至少返回 rootPID 自己
		return []int{rootPID}
	}

	// 解析输出构建 parent->children 映射
	parentChildren := make(map[int][]int)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") || strings.HasPrefix(line, ",") == false {
			continue
		}
		// CSV 格式: ,ParentProcessId,ProcessId
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		parentStr := strings.TrimSpace(parts[1])
		childStr := strings.TrimSpace(parts[2])
		parentID := atoiSafe(parentStr)
		childID := atoiSafe(childStr)
		if parentID > 0 && childID > 0 {
			parentChildren[parentID] = append(parentChildren[parentID], childID)
		}
	}

	// BFS 收集所有后代
	seen := map[int]bool{rootPID: true}
	queue := []int{rootPID}
	var result []int

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		for _, child := range parentChildren[cur] {
			if !seen[child] {
				seen[child] = true
				queue = append(queue, child)
			}
		}
	}

	return result
}

// findListeningPorts 查找属于给定 PID 集合的 LISTENING 端口
func findListeningPorts(pids []int) []string {
	pidSet := make(map[int]bool, len(pids))
	for _, pid := range pids {
		pidSet[pid] = true
	}

	// netstat -aon -p TCP 获取所有 TCP 连接
	cmd := exec.Command("cmd", "/C", "netstat -aon -p TCP")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var ports []string

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// LISTENING 行格式:  TCP    0.0.0.0:3000    0.0.0.0:0    LISTENING    1234
		if !strings.Contains(line, "LISTENING") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// fields[4] 是 PID
		pid := atoiSafe(fields[4])
		if !pidSet[pid] {
			continue
		}

		// fields[1] 是本地地址，如 0.0.0.0:3000 或 [::]:3000
		localAddr := fields[1]
		colonIdx := strings.LastIndex(localAddr, ":")
		if colonIdx < 0 {
			continue
		}
		port := localAddr[colonIdx+1:]
		if port != "" && !seen[port] {
			seen[port] = true
			ports = append(ports, port)
		}
	}

	return ports
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
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

	if m.logs[scriptID] == nil {
		m.logs[scriptID] = make([]LogEntry, 0, 100)
	}

	// 限制日志条数
	if len(m.logs[scriptID]) >= m.logMax {
		m.logs[scriptID] = m.logs[scriptID][len(m.logs[scriptID])-m.logMax+100:]
	}

	m.logs[scriptID] = append(m.logs[scriptID], entry)
	m.mu.Unlock()

	// 锁外发送事件到前端
	if m.emit != nil {
		m.emit("script:log", map[string]any{
			"id":        scriptID,
			"text":      entry.Text,
			"source":    entry.Source,
			"timestamp": entry.Timestamp,
		})
	}
}

// emitStatus 发送状态变更事件
func (m *Manager) emitStatus(scriptID uint, status string, exitCode int, pid int, childPid int) {
	if m.emit != nil {
		m.emit("script:status", map[string]any{
			"id":       scriptID,
			"status":   status,
			"exitCode": exitCode,
			"pid":      pid,
			"childPid": childPid,
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
		m.emitStatus(scriptID, "exited", 0, 0, 0)
	}
}

func killProcessTree(pid int) error {
	cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/T", "/F")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		probe := exec.Command("cmd", "/C", fmt.Sprintf("tasklist /FI \"PID eq %d\" /NH", pid))
		if runtime.GOOS == "windows" {
			probe.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
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
