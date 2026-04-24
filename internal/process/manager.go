package process

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ManagedProcess 管理的进程
type ManagedProcess struct {
	Cmd    *exec.Cmd
	Cancel context.CancelFunc
	PID    int
}

// LogEntry 日志条目
type LogEntry struct {
	Text      string `json:"text"`
	Source    string `json:"source"` // stdout or stderr
	Timestamp string `json:"timestamp"`
}

// Manager 进程管理器
type Manager struct {
	mu       sync.RWMutex
	ctx      context.Context
	processes map[uint]*ManagedProcess
	logs     map[uint][]LogEntry // 每个 script 的日志缓冲
	logMax   int                // 每个 script 最大日志条数
}

var instance *Manager
var once sync.Once

// GetManager 获取单例管理器
func GetManager(ctx context.Context) *Manager {
	once.Do(func() {
		instance = &Manager{
			ctx:       ctx,
			processes: make(map[uint]*ManagedProcess),
			logs:      make(map[uint][]LogEntry),
			logMax:    2000,
		}
	})
	instance.ctx = ctx
	return instance
}

// Start 启动脚本进程
func (m *Manager) Start(scriptID uint, command string, workDir string, envVarsJSON string) error {
	m.mu.Lock()
	// 如果已在运行，先停止
	if p, ok := m.processes[scriptID]; ok {
		if p.Cancel != nil {
			p.Cancel()
		}
		delete(m.processes, scriptID)
	}
	// 清空旧日志
	m.logs[scriptID] = nil
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(m.ctx)

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

	m.mu.Lock()
	m.processes[scriptID] = &ManagedProcess{
		Cmd:    cmd,
		Cancel: cancel,
		PID:    cmd.Process.Pid,
	}
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
		m.mu.Lock()
		delete(m.processes, scriptID)
		m.mu.Unlock()

		exitCode := 0
		status := "exited"
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		m.emitStatus(scriptID, status, exitCode, 0)
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

	// 发送停止日志
	m.addLog(scriptID, LogEntry{
		Text:      "[进程已停止]",
		Source:    "system",
		Timestamp: time.Now().Format("15:04:05"),
	})

	return nil
}

// Restart 重启脚本
func (m *Manager) Restart(scriptID uint, command string, workDir string, envVarsJSON string) error {
	m.Stop(scriptID)
	// 等待进程完全退出
	time.Sleep(500 * time.Millisecond)
	return m.Start(scriptID, command, workDir, envVarsJSON)
}

// GetStatus 获取进程状态
func (m *Manager) GetStatus(scriptID uint) (status string, exitCode int, pid int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, ok := m.processes[scriptID]; ok {
		return "running", 0, p.PID
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
		runtime.EventsEmit(m.ctx, "script:log", map[string]interface{}{
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
		runtime.EventsEmit(m.ctx, "script:status", map[string]interface{}{
			"id":       scriptID,
			"status":   status,
			"exitCode": exitCode,
			"pid":      pid,
		})
	}
}
