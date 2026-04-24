import { ref, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import {
  ListProjects,
  CreateProject,
  UpdateProject,
  DeleteProject,
  ListScripts,
  CreateScript,
  UpdateScript,
  DeleteScript,
  StartScript,
  StopScript,
  RestartScript,
  GetScriptStatus,
  GetScriptLogs,
  ClearScriptLogs,
  SelectDirectory as SelectDirDialog,
} from '../../wailsjs/go/main/App'

// ==================== Types ====================
export interface ProjectItem {
  id: number
  name: string
  notes: string
  order: number
  scriptCount: number
}

export interface ScriptItem {
  id: number
  name: string
  command: string
  workDir: string
  envVars: string
  notes: string
  projectId?: number
  projectName: string
  createdAt: string
}

export interface LogLine {
  id: number
  scriptId: number
  text: string
  source: string
  timestamp: string
}

// ==================== State ====================
// Project state
export const projectList = ref<ProjectItem[]>([])
export const selectedProjectId = ref<number | null>(null) // null = 全部
export const projectFormVisible = ref(false)
export const projectForm = ref<{ id: number; name: string; notes: string }>({ id: 0, name: '', notes: '' })
export const projectFormIsEdit = ref(false)

// Script state
export const scriptList = ref<ScriptItem[]>([])
export const scriptLoading = ref(false)
export const scriptFormVisible = ref(false)
export const scriptForm = ref<{ id: number; name: string; command: string; workDir: string; envVars: string; notes: string; projectId: number }>({
  id: 0, name: '', command: '', workDir: '', envVars: '', notes: '', projectId: 0,
})
export const scriptFormIsEdit = ref(false)
export const scriptLogVisible = ref(false)
export const scriptLogId = ref(0)
export const scriptLogName = ref('')
export const scriptLogs = ref<LogLine[]>([])
export const scriptLogLoading = ref(false)
export const scriptLogAutoScroll = ref(true)
export const scriptLogRef = ref<HTMLDivElement>()
export const scriptStatuses = ref<Record<number, { status: string; pid: number; exitCode: number }>>({})

// Event listener cleanup functions
let unlistenScriptLog: (() => void) | null = null
let unlistenScriptStatus: (() => void) | null = null

// ==================== Computed ====================
export const filteredScriptList = computed(() => {
  if (selectedProjectId.value === null) return scriptList.value
  return scriptList.value.filter(s => s.projectId === selectedProjectId.value)
})

// ==================== Project Functions ====================
export async function loadProjects() {
  try {
    projectList.value = await ListProjects() || []
  } catch { /* ignore */ }
}

export function openAddProject() {
  projectFormIsEdit.value = false
  projectForm.value = { id: 0, name: '', notes: '' }
  projectFormVisible.value = true
}

export function openEditProject(item: ProjectItem) {
  projectFormIsEdit.value = true
  projectForm.value = { id: item.id, name: item.name, notes: item.notes }
  projectFormVisible.value = true
}

export async function handleSaveProject() {
  if (!projectForm.value.name.trim()) {
    ElMessage.warning('请输入项目名称')
    return
  }
  try {
    if (projectFormIsEdit.value) {
      await UpdateProject(projectForm.value.id, projectForm.value.name.trim(), projectForm.value.notes)
      ElMessage.success('更新成功')
    } else {
      await CreateProject(projectForm.value.name.trim(), projectForm.value.notes)
      ElMessage.success('创建成功')
    }
    projectFormVisible.value = false
    await loadProjects()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

export async function handleDeleteProject(item: ProjectItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除项目「${item.name}」？该项目下的 ${item.scriptCount} 个脚本将变为未分类。`,
      '删除项目',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch { return }
  try {
    await DeleteProject(item.id)
    if (selectedProjectId.value === item.id) selectedProjectId.value = null
    ElMessage.success('已删除')
    await loadProjects()
    await loadScripts()
  } catch (e) {
    ElMessage.error(String(e))
  }
}

// ==================== Script Functions ====================
export async function loadScripts() {
  scriptLoading.value = true
  try {
    scriptList.value = await ListScripts() || []
    // 真正加载每个脚本的初始状态
    for (const s of scriptList.value) {
      try {
        const status = await GetScriptStatus(s.id)
        scriptStatuses.value[s.id] = {
          status: status.status,
          pid: status.pid,
          exitCode: status.exitCode,
        }
      } catch { /* ignore */ }
    }
  } catch (e: any) {
    ElMessage.error(e.message || '加载脚本失败')
  }
  scriptLoading.value = false
}

export function openAddScript() {
  scriptFormIsEdit.value = false
  scriptForm.value = {
    id: 0, name: '', command: '', workDir: '', envVars: '', notes: '',
    projectId: selectedProjectId.value || 0,
  }
  scriptFormVisible.value = true
}

export function openEditScript(item: ScriptItem) {
  scriptFormIsEdit.value = true
  scriptForm.value = {
    id: item.id, name: item.name, command: item.command,
    workDir: item.workDir, envVars: item.envVars, notes: item.notes,
    projectId: item.projectId || 0,
  }
  scriptFormVisible.value = true
}

export async function handleSaveScript() {
  if (!scriptForm.value.name.trim()) {
    ElMessage.warning('请输入脚本名称')
    return
  }
  if (!scriptForm.value.command.trim()) {
    ElMessage.warning('请输入命令')
    return
  }
  try {
    if (scriptFormIsEdit.value) {
      await UpdateScript(
        scriptForm.value.id,
        scriptForm.value.name.trim(),
        scriptForm.value.command.trim(),
        scriptForm.value.workDir.trim(),
        scriptForm.value.envVars,
        scriptForm.value.notes,
        scriptForm.value.projectId,
      )
      ElMessage.success('更新成功')
    } else {
      await CreateScript(
        scriptForm.value.name.trim(),
        scriptForm.value.command.trim(),
        scriptForm.value.workDir.trim(),
        scriptForm.value.envVars,
        scriptForm.value.notes,
        scriptForm.value.projectId,
      )
      ElMessage.success('创建成功')
    }
    scriptFormVisible.value = false
    await loadScripts()
    await loadProjects()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

export async function handleDeleteScript(item: ScriptItem) {
  try {
    await ElMessageBox.confirm(`确定删除脚本「${item.name}」？${scriptStatuses.value[item.id]?.status === 'running' ? '运行中的进程也会被停止。' : ''}`, '删除脚本', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
  } catch { return }
  try {
    await DeleteScript(item.id)
    delete scriptStatuses.value[item.id]
    ElMessage.success('已删除')
    await loadScripts()
    await loadProjects()
  } catch (e) {
    ElMessage.error(String(e))
  }
}

export async function handleStartScript(id: number) {
  try {
    await StartScript(id)
    scriptStatuses.value[id] = { status: 'running', pid: 0, exitCode: 0 }
  } catch (e: any) {
    ElMessage.error(e.message || '启动失败')
  }
}

export async function handleStopScript(id: number) {
  try {
    await StopScript(id)
    scriptStatuses.value[id] = { status: 'stopped', pid: 0, exitCode: 0 }
  } catch (e: any) {
    ElMessage.error(e.message || '停止失败')
  }
}

export async function handleRestartScript(id: number) {
  try {
    await RestartScript(id)
    scriptStatuses.value[id] = { status: 'running', pid: 0, exitCode: 0 }
  } catch (e: any) {
    ElMessage.error(e.message || '重启失败')
  }
}

export async function openScriptLog(item: ScriptItem) {
  scriptLogId.value = item.id
  scriptLogName.value = item.name
  scriptLogVisible.value = true
  scriptLogLoading.value = true
  scriptLogs.value = []
  try {
    scriptLogs.value = await GetScriptLogs(item.id) || []
  } catch { /* ignore */ }
  scriptLogLoading.value = false
  await nextTick(() => {
    if (scriptLogAutoScroll.value) {
      scrollLogToBottom()
    }
  })
}

export async function handleClearLogs() {
  try {
    await ClearScriptLogs(scriptLogId.value)
    scriptLogs.value = []
    ElMessage.success('日志已清空')
  } catch (e: any) {
    ElMessage.error(e.message || '清空失败')
  }
}

export function scrollLogToBottom() {
  if (scriptLogRef.value) {
    scriptLogRef.value.scrollTop = scriptLogRef.value.scrollHeight
  }
}

export function handleLogScroll() {
  if (!scriptLogRef.value) return
  const el = scriptLogRef.value
  scriptLogAutoScroll.value = (el.scrollTop + el.clientHeight >= el.scrollHeight - 50)
}

export function scriptStatusLabel(status: string): { text: string; type: string } {
  switch (status) {
    case 'running': return { text: '运行中', type: 'success' }
    case 'exited': return { text: '已退出', type: 'danger' }
    default: return { text: '已停止', type: 'info' }
  }
}

export async function handleScriptBrowseDir() {
  try {
    const dir = await SelectDirDialog()
    if (dir) scriptForm.value.workDir = dir
  } catch { /* ignore */ }
}

// ==================== Event Listeners ====================
export function setupEventListeners() {
  unlistenScriptLog = EventsOn('script:log', (event: any) => {
    if (event.id === scriptLogId.value && scriptLogVisible.value) {
      scriptLogs.value.push({
        id: 0,
        scriptId: event.id,
        text: event.text,
        source: event.source,
        timestamp: event.timestamp,
      })
      nextTick(() => {
        if (scriptLogAutoScroll.value) scrollLogToBottom()
      })
    }
    // 同时更新状态为 running
    if (scriptStatuses.value[event.id]) {
      scriptStatuses.value[event.id].status = 'running'
    }
  })

  unlistenScriptStatus = EventsOn('script:status', (event: any) => {
    scriptStatuses.value[event.id] = {
      status: event.status,
      pid: event.pid || 0,
      exitCode: event.exitCode || 0,
    }
  })
}

export function teardownEventListeners() {
  if (typeof unlistenScriptLog === 'function') unlistenScriptLog()
  if (typeof unlistenScriptStatus === 'function') unlistenScriptStatus()
}
