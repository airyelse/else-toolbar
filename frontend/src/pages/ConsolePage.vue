<script lang="ts" setup>
import { onMounted, onUnmounted } from 'vue'
import {
  projectList,
  selectedProjectId,
  projectFormVisible,
  projectForm,
  projectFormIsEdit,
  scriptList,
  scriptLoading,
  scriptFormVisible,
  scriptForm,
  scriptFormIsEdit,
  scriptLogVisible,
  scriptLogId,
  scriptLogName,
  scriptLogs,
  scriptLogLoading,
  scriptLogAutoScroll,
  scriptLogRef,
  scriptStatuses,
  filteredScriptList,
  loadProjects,
  openAddProject,
  openEditProject,
  handleSaveProject,
  handleDeleteProject,
  loadScripts,
  openAddScript,
  openEditScript,
  handleSaveScript,
  handleDeleteScript,
  handleStartScript,
  handleStopScript,
  handleRestartScript,
  openScriptLog,
  handleClearLogs,
  scrollLogToBottom,
  handleLogScroll,
  openLogUrl,
  scriptStatusLabel,
  handleScriptBrowseDir,
  setupEventListeners,
  teardownEventListeners,
} from '../composables/useScriptConsole'

onMounted(async () => {
  await loadProjects()
  await loadScripts()
  setupEventListeners()
})

onUnmounted(() => {
  teardownEventListeners()
})
</script>

<template>
  <div class="body" v-loading="scriptLoading">
    <!-- Project Sidebar -->
    <aside class="sidebar">
      <div class="sidebar-section">
        <div
          class="sidebar-item"
          :class="{ active: selectedProjectId === null }"
          @click="selectedProjectId = null"
        >
          <el-icon><Grid /></el-icon>
          <span class="sidebar-label">全部脚本</span>
          <span class="sidebar-count">{{ scriptList.length }}</span>
        </div>
      </div>

      <div class="sidebar-section">
        <div class="sidebar-section-header">
          <span class="sidebar-section-title">项目</span>
          <el-button text size="small" @click="openAddProject()" class="sidebar-add-btn">
            <el-icon><Plus /></el-icon>
          </el-button>
        </div>
        <div class="project-list">
          <div
            v-for="proj in projectList"
            :key="proj.id"
            class="sidebar-item"
            :class="{ active: selectedProjectId === proj.id }"
            @click="selectedProjectId = proj.id"
          >
            <el-icon size="14" style="color: var(--primary)"><Folder /></el-icon>
            <span class="sidebar-label">{{ proj.name }}</span>
            <span class="sidebar-count">{{ proj.scriptCount }}</span>
            <el-icon size="12" class="tag-edit" @click.stop="openEditProject(proj)"><Edit /></el-icon>
            <el-icon size="12" class="tag-delete" @click.stop="handleDeleteProject(proj)"><Delete /></el-icon>
          </div>
        </div>
        <div class="sidebar-empty" v-if="projectList.length === 0">
          暂无项目
        </div>
      </div>

      <div class="sidebar-section" v-if="selectedProjectId !== null">
        <div class="sidebar-item" style="color: var(--text-muted); font-size: 12px; padding: 6px 16px">
          <el-icon size="12"><InfoFilled /></el-icon>
          <span>点击「全部脚本」返回</span>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content">
      <div class="content-header">
        <div class="content-header-info">
          <h2 class="content-title">{{ selectedProjectId ? (projectList.find(p => p.id === selectedProjectId)?.name || '项目') : '全部脚本' }}</h2>
          <span class="content-count">{{ filteredScriptList.length }} 个脚本</span>
        </div>
        <el-button type="primary" size="small" @click="openAddScript">
          <el-icon><Plus /></el-icon><span>新增脚本</span>
        </el-button>
      </div>
      <div v-if="filteredScriptList.length > 0" class="console-script-list">
        <div
          v-for="item in filteredScriptList"
          :key="item.id"
          class="console-script-card"
          :class="{ 'console-script-running': scriptStatuses[item.id]?.status === 'running' }"
        >
          <div class="console-script-header">
            <div class="console-script-icon" :class="{ running: scriptStatuses[item.id]?.status === 'running' }">
              <el-icon size="18"><Cpu /></el-icon>
            </div>
            <div class="console-script-info">
              <div class="console-script-name">
                {{ item.name }}
                <el-tag v-if="item.projectName && selectedProjectId === null" size="small" effect="plain" style="margin-left: 6px; font-size: 11px">
                  {{ item.projectName }}
                </el-tag>
              </div>
              <div class="console-script-cmd" :title="item.command">{{ item.command }}</div>
            </div>
            <div class="console-script-status">
              <el-tag
                v-if="scriptStatuses[item.id]"
                :type="scriptStatusLabel(scriptStatuses[item.id]).type as any"
                size="small"
                effect="light"
              >
                {{ scriptStatusLabel(scriptStatuses[item.id]).text }}
              </el-tag>
              <el-tag v-else type="info" size="small" effect="light">已停止</el-tag>
              <span v-if="scriptStatuses[item.id]?.pid" class="console-script-pid">PID: {{ scriptStatuses[item.id].pid }}</span>
            </div>
          </div>
          <div class="console-script-meta" v-if="item.workDir || item.notes">
            <span v-if="item.workDir" class="console-script-workdir" :title="item.workDir">
              <el-icon size="12"><Folder /></el-icon>{{ item.workDir }}
            </span>
            <span v-if="item.notes" class="console-script-notes">{{ item.notes }}</span>
          </div>
          <div class="console-script-actions">
            <el-button
              v-if="scriptStatuses[item.id]?.status !== 'running'"
              type="success"
              text
              size="small"
              @click="handleStartScript(item.id)"
              class="action-btn"
            >
              <el-icon><VideoPlay /></el-icon><span>启动</span>
            </el-button>
            <el-button
              v-if="scriptStatuses[item.id]?.status === 'running'"
              type="danger"
              :disabled="item.elevated"
              text
              size="small"
              @click="handleStopScript(item.id)"
              class="action-btn"
            >
              <el-icon><VideoPause /></el-icon><span>停止</span>
            </el-button>
            <el-button
              v-if="scriptStatuses[item.id]?.status === 'running'"
              :disabled="item.elevated"
              text
              size="small"
              @click="handleRestartScript(item.id)"
              class="action-btn"
            >
              <el-icon><RefreshRight /></el-icon><span>重启</span>
            </el-button>
            <span v-if="item.elevated && scriptStatuses[item.id]?.status === 'running'" class="action-hint">管理员脚本请在外部窗口手动关闭</span>
            <el-button text size="small" @click="openScriptLog(item)" class="action-btn action-primary">
              <el-icon><Document /></el-icon><span>日志</span>
            </el-button>
            <div class="action-spacer"></div>
            <el-button text size="small" @click="openEditScript(item)" class="action-btn">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button text size="small" @click="handleDeleteScript(item)" class="action-btn action-danger">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
      </div>
      <div class="empty-state" v-else-if="!scriptLoading">
        <div class="empty-icon"><el-icon size="48"><Monitor /></el-icon></div>
        <h3 class="empty-title">暂无脚本</h3>
        <p class="empty-desc">点击「新增脚本」添加你的第一个命令或程序</p>
        <el-button type="primary" size="large" @click="openAddScript" round>
          <el-icon><Plus /></el-icon> 添加脚本
        </el-button>
      </div>
    </main>
  </div>

  <!-- Project Dialog -->
  <el-dialog
    v-model="projectFormVisible"
    :title="projectFormIsEdit ? '编辑项目' : '新增项目'"
    width="420px"
    align-center
  >
    <el-form label-position="top">
      <el-form-item label="项目名称" required>
        <el-input v-model="projectForm.name" placeholder="如：My Web App" size="large" @keyup.enter="handleSaveProject" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="projectForm.notes" placeholder="备注信息（可选）" size="large" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="projectFormVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="handleSaveProject" style="flex: 1">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- Script Edit/Create Dialog -->
  <el-dialog
    v-model="scriptFormVisible"
    :title="scriptFormIsEdit ? '编辑脚本' : '新增脚本'"
    width="560px"
    align-center
  >
    <el-form label-position="top">
      <el-form-item label="名称" required>
        <el-input v-model="scriptForm.name" placeholder="如：Dev Server" size="large" @keyup.enter="handleSaveScript" />
      </el-form-item>
      <el-form-item label="命令" required>
        <el-input
          v-model="scriptForm.command"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          placeholder="完整命令行，如：node server.js 或 python -m http.server 8000"
        />
      </el-form-item>
      <el-form-item label="所属项目">
        <el-select v-model="scriptForm.projectId" placeholder="无（未分类）" clearable size="large" style="width: 100%">
          <el-option v-for="p in projectList" :key="p.id" :label="p.name" :value="p.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="工作目录">
        <div class="config-dir-row">
          <el-input v-model="scriptForm.workDir" placeholder="留空则使用当前目录" size="large" />
          <el-button size="large" @click="handleScriptBrowseDir">
            <el-icon><FolderOpened /></el-icon>
          </el-button>
        </div>
      </el-form-item>
      <el-form-item label="环境变量">
        <el-input
          v-model="scriptForm.envVars"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          placeholder='JSON 格式：[{"key":"PORT","value":"3000"}]（可选）'
        />
        <div class="form-hint">可选，JSON 格式的键值对数组</div>
      </el-form-item>
      <el-form-item>
        <div class="switch-row">
          <el-switch v-model="scriptForm.elevated" />
          <span class="switch-label">以管理员身份运行</span>
        </div>
        <div class="form-hint">
          该脚本将以独立窗口启动，日志可能无法完整回传。
        </div>
        <div class="switch-row switch-row-secondary">
          <el-switch v-model="scriptForm.keepWindow" :disabled="!scriptForm.elevated" />
          <span class="switch-label">退出后保留管理员窗口</span>
        </div>
        <div class="form-hint">
          便于调试，脚本结束后不自动关闭窗口。
        </div>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="scriptForm.notes" placeholder="备注信息（可选）" size="large" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="scriptFormVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="handleSaveScript" style="flex: 1">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <!-- Script Log Dialog -->
  <el-dialog
    v-model="scriptLogVisible"
    :title="`日志 — ${scriptLogName}`"
    width="80%"
    top="5vh"
    :close-on-click-modal="false"
    class="script-log-dialog"
  >
    <div class="script-log-toolbar">
      <el-tag
        v-if="scriptStatuses[scriptLogId]"
        :type="scriptStatusLabel(scriptStatuses[scriptLogId]).type as any"
        size="small"
        effect="plain"
      >
        {{ scriptStatusLabel(scriptStatuses[scriptLogId]).text }}
      </el-tag>
      <span class="script-log-count">{{ scriptLogs.length }} 条日志</span>
      <div style="flex:1"></div>
      <el-button size="small" text @click="handleClearLogs">
        <el-icon><Delete /></el-icon><span>清空</span>
      </el-button>
      <el-button
        size="small"
        :type="scriptLogAutoScroll ? 'primary' : undefined"
        text
        @click="scriptLogAutoScroll = true"
      >
        <el-icon><Bottom /></el-icon><span>自动滚动</span>
      </el-button>
    </div>
    <div
      ref="scriptLogRef"
      class="script-log-viewer"
      @scroll="handleLogScroll"
      v-loading="scriptLogLoading"
    >
      <div
        v-for="(line, idx) in scriptLogs"
        :key="idx"
        class="script-log-line"
        :class="{ 'script-log-stderr': line.source === 'stderr', 'script-log-system': line.source === 'system' }"
      >
        <span class="script-log-time">{{ line.timestamp }}</span>
        <span class="script-log-text">
          <template v-for="(segment, segmentIdx) in line.segments" :key="segmentIdx">
            <a
              v-if="segment.url"
              class="script-log-link"
              :href="segment.url"
              :style="segment.style"
              @click.prevent="openLogUrl(segment.url)"
            >{{ segment.text }}</a>
            <span v-else :style="segment.style">{{ segment.text }}</span>
          </template>
        </span>
      </div>
      <div v-if="scriptLogs.length === 0 && !scriptLogLoading" class="script-log-empty">
        暂无日志
      </div>
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-tooltip v-if="scriptStatuses[scriptLogId]?.status === 'running' && scriptList.find(s => s.id === scriptLogId)?.elevated" content="管理员脚本请在外部窗口手动关闭" placement="top">
          <span style="flex: 1">
            <el-button type="danger" size="large" disabled style="width: 100%">
              <el-icon><VideoPause /></el-icon><span>停止</span>
            </el-button>
          </span>
        </el-tooltip>
        <el-button
          v-else-if="scriptStatuses[scriptLogId]?.status === 'running'"
          type="danger"
          size="large"
          @click="handleStopScript(scriptLogId)" style="flex: 1"
        >
          <el-icon><VideoPause /></el-icon><span>停止</span>
        </el-button>
        <el-button
          v-else
          type="success"
          size="large"
          @click="handleStartScript(scriptLogId)" style="flex: 1"
        >
          <el-icon><VideoPlay /></el-icon><span>启动</span>
        </el-button>
        <el-button size="large" @click="scriptLogVisible = false" style="flex: 1">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.body {
  height: 100%;
  display: flex;
  overflow: hidden;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 240px;
  flex-shrink: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.sidebar-section {
  padding: 8px 0;
}

.sidebar-section + .sidebar-section {
  border-top: 1px solid var(--border);
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 14px;
  transition: all 0.15s;
}

.sidebar-item:hover {
  background: var(--bg);
  color: var(--text);
}

.sidebar-item.active {
  background: var(--primary-bg);
  color: var(--primary);
  font-weight: 500;
}

.sidebar-label {
  flex: 1;
}

.sidebar-count {
  font-size: 12px;
  color: var(--text-muted);
  min-width: 20px;
  text-align: right;
}

.sidebar-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px 4px;
}

.sidebar-section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-muted);
}

.sidebar-add-btn {
  color: var(--text-muted) !important;
  padding: 2px !important;
}

.sidebar-add-btn:hover {
  color: var(--primary) !important;
}

.sidebar-empty {
  padding: 8px 16px;
  font-size: 13px;
  color: var(--text-muted);
}

/* Tag actions */
.tag-edit,
.tag-delete {
  display: none;
  color: var(--text-muted);
}

.sidebar-item:hover .tag-edit,
.sidebar-item:hover .tag-delete {
  display: block;
}

.tag-delete:hover {
  color: var(--danger) !important;
}

/* ===== Main Content ===== */
.main-content {
  flex: 1;
  padding: 20px 24px;
  overflow-y: auto;
}

.content-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.content-header-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.content-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
}

.content-count {
  font-size: 13px;
  color: var(--text-muted);
}

/* ===== Console Script List ===== */
.console-script-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.project-list {
  padding: 0 4px;
}

.console-script-card {
  background: var(--bg-card);
  border-radius: var(--radius);
  border: 1px solid var(--border);
  padding: 16px;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.console-script-card:hover {
  box-shadow: var(--shadow-md);
  border-color: rgba(99, 102, 241, 0.2);
}

.console-script-card.console-script-running {
  border-color: rgba(16, 185, 129, 0.4);
  background: #f0fdf4;
}

.console-script-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.console-script-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  flex-shrink: 0;
  transition: all 0.2s;
}

.console-script-icon.running {
  background: var(--success);
  color: #fff;
  animation: console-pulse 2s infinite;
}

@keyframes console-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
  50% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
}

.console-script-info {
  flex: 1;
  min-width: 0;
}

.console-script-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.console-script-cmd {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.console-script-status {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.console-script-pid {
  font-size: 11px;
  color: var(--text-muted);
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.console-script-meta {
  display: flex;
  gap: 12px;
  align-items: center;
  padding-left: 50px;
  flex-wrap: wrap;
}

.console-script-workdir {
  font-size: 12px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
  max-width: 300px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.console-script-notes {
  font-size: 12px;
  color: var(--text-muted);
  font-style: italic;
}

.console-script-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
  margin-top: auto;
}

/* ===== Actions ===== */
.action-btn {
  font-size: 12px !important;
  border-radius: 6px !important;
  color: var(--text-secondary) !important;
}

.action-btn:hover { background: var(--bg) !important; }
.action-primary { color: var(--primary) !important; }
.action-primary:hover { background: var(--primary-bg) !important; }
.action-danger:hover { color: var(--danger) !important; background: #fef2f2 !important; }
.action-spacer { flex: 1; }
.action-hint { color: var(--text-muted); font-size: 12px; align-self: center; }

/* ===== Empty State ===== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
}

.empty-icon {
  width: 88px;
  height: 88px;
  border-radius: 22px;
  background: var(--primary-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  margin-bottom: 20px;
}

.empty-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 6px;
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 20px;
}

/* ===== Config Dir Row ===== */
.config-dir-row {
  display: flex;
  gap: 8px;
}

.config-dir-row .el-input {
  flex: 1;
}

/* ===== Form Hint ===== */
.form-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
}

.switch-row {
  display: flex;
  align-items: center;
}

.switch-row-secondary {
  margin-top: 8px;
}

.switch-label {
  margin-left: 8px;
  font-size: 14px;
  color: var(--text);
}

/* ===== Script Log Dialog ===== */
.script-log-dialog :deep(.el-dialog__body) {
  padding: 0 !important;
  display: flex;
  flex-direction: column;
}

.script-log-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.script-log-count {
  font-size: 12px;
  color: var(--text-muted);
}

.script-log-viewer {
  flex: 1;
  height: 50vh;
  overflow-y: auto;
  background: #1e1e2e;
  padding: 12px 16px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.script-log-line {
  display: flex;
  gap: 8px;
  padding: 1px 0;
  color: #cdd6f4;
}

.script-log-time {
  color: #6c7086;
  flex-shrink: 0;
  font-size: 12px;
}

.script-log-text {
  white-space: pre-wrap;
  word-break: break-all;
}

.script-log-link {
  color: #74c7ec;
  text-decoration: underline;
  text-underline-offset: 2px;
  cursor: pointer;
}

.script-log-link:hover {
  color: #89dceb;
}

.script-log-line.script-log-stderr .script-log-text {
  color: #f38ba8;
}

.script-log-line.script-log-system .script-log-text {
  color: #89b4fa;
  font-style: italic;
}

.script-log-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #6c7086;
  font-size: 14px;
}

/* Script Log Dialog custom scrollbar */
.script-log-viewer::-webkit-scrollbar {
  width: 8px;
}

.script-log-viewer::-webkit-scrollbar-track {
  background: #181825;
}

.script-log-viewer::-webkit-scrollbar-thumb {
  background: #45475a;
  border-radius: 4px;
}

.script-log-viewer::-webkit-scrollbar-thumb:hover {
  background: #585b70;
}
</style>
