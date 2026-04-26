package models

import "gorm.io/gorm"

// Project 项目
type Project struct {
	gorm.Model
	Name  string `json:"name"`
	Notes string `json:"notes"`
	Order int    `json:"order"` // 排序
}

// Script 脚本配置
type Script struct {
	gorm.Model
	Name       string   `json:"name"`
	Command    string   `json:"command"`    // 完整命令行，如 "node server.js" 或 "python -m http.server"
	WorkDir    string   `json:"workDir"`    // 工作目录
	EnvVars    string   `json:"envVars"`    // 环境变量，JSON 格式 [{"key":"FOO","value":"bar"},...]
	Notes      string   `json:"notes"`      // 备注
	Elevated   bool     `json:"elevated"`   // 是否以管理员运行
	KeepWindow bool     `json:"keepWindow"` // 管理员窗口执行后保持打开
	ProjectID  *uint    `json:"projectId"`  // 所属项目
	Project    *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// ProjectDTO 前端传输对象
type ProjectDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Notes       string `json:"notes"`
	Order       int    `json:"order"`
	ScriptCount int    `json:"scriptCount"`
}

// ScriptDTO 前端传输对象
type ScriptDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	WorkDir     string `json:"workDir"`
	EnvVars     string `json:"envVars"`
	Notes       string `json:"notes"`
	Elevated    bool   `json:"elevated"`
	KeepWindow  bool   `json:"keepWindow"`
	ProjectID   *uint  `json:"projectId"`
	ProjectName string `json:"projectName"`
	CreatedAt   string `json:"createdAt"`
}

// ScriptStatusDTO 进程状态传输对象
type ScriptStatusDTO struct {
	ID       uint   `json:"id"`
	Status   string `json:"status"`   // running, stopped, exited
	ExitCode int    `json:"exitCode"` // 退出码（仅 exited 状态有值）
	PID      int    `json:"pid"`      // 进程 PID
}

// LogLineDTO 日志行传输对象
type LogLineDTO struct {
	ID        uint   `json:"id"`
	ScriptID  uint   `json:"scriptId"`
	Text      string `json:"text"`
	Source    string `json:"source"` // stdout or stderr
	Timestamp string `json:"timestamp"`
}

// ToDTO 转换 Script 为 DTO
func (s *Script) ToDTO() *ScriptDTO {
	createdAt := ""
	if !s.CreatedAt.IsZero() {
		createdAt = s.CreatedAt.Format("2006-01-02 15:04:05")
	}
	dto := &ScriptDTO{
		ID:         s.ID,
		Name:       s.Name,
		Command:    s.Command,
		WorkDir:    s.WorkDir,
		EnvVars:    s.EnvVars,
		Notes:      s.Notes,
		Elevated:   s.Elevated,
		KeepWindow: s.KeepWindow,
		ProjectID:  s.ProjectID,
		CreatedAt:  createdAt,
	}
	if s.Project != nil {
		dto.ProjectName = s.Project.Name
	}
	return dto
}

// ToDTO 转换 Project 为 DTO
func (p *Project) ToDTO(scriptCount int) *ProjectDTO {
	return &ProjectDTO{
		ID:          p.ID,
		Name:        p.Name,
		Notes:       p.Notes,
		Order:       p.Order,
		ScriptCount: scriptCount,
	}
}
