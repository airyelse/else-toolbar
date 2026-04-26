import { ref, computed, nextTick, type CSSProperties } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { BrowserOpenURL, EventsOn } from '../../wailsjs/runtime/runtime'
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
  elevated: boolean
  keepWindow: boolean
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
  segments: LogSegment[]
}

type RawLogLine = Omit<LogLine, 'segments'>

export interface LogSegment {
  text: string
  url?: string
  style?: CSSProperties
}

const ANSI_ESCAPE_RE = /[\u001B\u009B][[\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?\u0007)|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PR-TZcf-nq-uy=><~]))/g
const ANSI_SGR_RE = /\u001b\[([\d;]*)m/g
const URL_RE = /(https?:\/\/[^\s<>'"`]+|(?:localhost|127\.0\.0\.1|0\.0\.0\.0):\d+(?:[^\s<>'"`]*)?)/g

type AnsiState = {
  color?: string
  backgroundColor?: string
  fontWeight?: CSSProperties['fontWeight']
  fontStyle?: CSSProperties['fontStyle']
  textDecoration?: string
  opacity?: string
}

const ANSI_FG_COLORS: Record<number, string> = {
  30: '#45475a',
  31: '#f38ba8',
  32: '#a6e3a1',
  33: '#f9e2af',
  34: '#89b4fa',
  35: '#f5c2e7',
  36: '#94e2d5',
  37: '#bac2de',
  90: '#585b70',
  91: '#f38ba8',
  92: '#a6e3a1',
  93: '#fab387',
  94: '#89dceb',
  95: '#cba6f7',
  96: '#74c7ec',
  97: '#f5e0dc',
}

const ANSI_BG_COLORS: Record<number, string> = {
  40: '#45475a',
  41: '#7d2038',
  42: '#1f4f2f',
  43: '#6b531f',
  44: '#1e3f6b',
  45: '#5f2e6b',
  46: '#1e5b56',
  47: '#bac2de',
  100: '#585b70',
  101: '#f38ba8',
  102: '#a6e3a1',
  103: '#fab387',
  104: '#89dceb',
  105: '#cba6f7',
  106: '#74c7ec',
  107: '#f5e0dc',
}

function resetAnsiState(): AnsiState {
  return {}
}

function cloneAnsiState(state: AnsiState): AnsiState {
  return { ...state }
}

function applyAnsiCode(state: AnsiState, code: number) {
  if (code === 0) {
    state.color = undefined
    state.backgroundColor = undefined
    state.fontWeight = undefined
    state.fontStyle = undefined
    state.textDecoration = undefined
    state.opacity = undefined
    return
  }

  if (code === 1) {
    state.fontWeight = '700'
    state.opacity = undefined
    return
  }

  if (code === 2) {
    state.opacity = '0.75'
    return
  }

  if (code === 3) {
    state.fontStyle = 'italic'
    return
  }

  if (code === 4) {
    state.textDecoration = mergeTextDecoration(state.textDecoration, 'underline')
    return
  }

  if (code === 9) {
    state.textDecoration = mergeTextDecoration(state.textDecoration, 'line-through')
    return
  }

  if (code === 22) {
    state.fontWeight = undefined
    state.opacity = undefined
    return
  }

  if (code === 23) {
    state.fontStyle = undefined
    return
  }

  if (code === 24) {
    state.textDecoration = removeTextDecoration(state.textDecoration, 'underline')
    return
  }

  if (code === 29) {
    state.textDecoration = removeTextDecoration(state.textDecoration, 'line-through')
    return
  }

  if (code === 39) {
    state.color = undefined
    return
  }

  if (code === 49) {
    state.backgroundColor = undefined
    return
  }

  if (ANSI_FG_COLORS[code]) {
    state.color = ANSI_FG_COLORS[code]
    return
  }

  if (ANSI_BG_COLORS[code]) {
    state.backgroundColor = ANSI_BG_COLORS[code]
  }
}

function mergeTextDecoration(value: string | undefined, token: string): string {
  const items = new Set((value || '').split(' ').filter(Boolean))
  items.add(token)
  return Array.from(items).join(' ')
}

function removeTextDecoration(value: string | undefined, token: string): string | undefined {
  const items = new Set((value || '').split(' ').filter(Boolean))
  items.delete(token)
  const result = Array.from(items).join(' ')
  return result || undefined
}

function sanitizeLogText(text: string): string {
  return text
    .replace(ANSI_ESCAPE_RE, '')
    .replace(/\r/g, '')
    .trimEnd()
}

function stripUnsupportedAnsi(text: string): string {
  return text.replace(ANSI_ESCAPE_RE, '').replace(/\r/g, '')
}

function segmentsFromAnsi(text: string): LogSegment[] {
  const input = text.replace(/\r/g, '')
  const state = resetAnsiState()
  const segments: LogSegment[] = []
  let lastIndex = 0

  for (const match of input.matchAll(ANSI_SGR_RE)) {
    const index = match.index ?? 0
    if (index > lastIndex) {
      const chunk = stripUnsupportedAnsi(input.slice(lastIndex, index))
      if (chunk) {
        const style = cloneAnsiState(state)
        segments.push(Object.keys(style).length > 0 ? { text: chunk, style } : { text: chunk })
      }
    }

    const codes = match[1] ? match[1].split(';').map(part => Number.parseInt(part, 10) || 0) : [0]
    for (const code of codes) applyAnsiCode(state, code)
    lastIndex = index + match[0].length
  }

  if (lastIndex < input.length) {
    const chunk = stripUnsupportedAnsi(input.slice(lastIndex))
    if (chunk) {
      const style = cloneAnsiState(state)
      segments.push(Object.keys(style).length > 0 ? { text: chunk, style } : { text: chunk })
    }
  }

  return segments
}

function trimTrailingPunctuation(text: string): { url: string; trailing: string } {
  let url = text
  let trailing = ''
  while (/[),.;!?]$/.test(url)) {
    trailing = url.slice(-1) + trailing
    url = url.slice(0, -1)
  }
  return { url, trailing }
}

function enrichSegments(segments: LogSegment[]): LogSegment[] {
  const fullText = segments.map(segment => segment.text).join('')
  const ranges = Array.from(fullText.matchAll(URL_RE)).flatMap(match => {
    const start = match.index ?? 0
    const matchedText = match[0]
    const { url, trailing } = trimTrailingPunctuation(matchedText)
    if (!url) return []
    return [{
      start,
      end: start + url.length,
      url: normalizeUrl(url),
      text: url,
      trailing,
    }]
  })

  if (ranges.length === 0) return segments

  const enriched: LogSegment[] = []
  let cursor = 0

  for (const segment of segments) {
    const segmentStart = cursor
    const segmentEnd = cursor + segment.text.length
    let localCursor = segmentStart

    for (const range of ranges) {
      if (range.end <= segmentStart || range.start >= segmentEnd) continue

      if (range.start > localCursor) {
        enriched.push({
          text: fullText.slice(localCursor, Math.min(range.start, segmentEnd)),
          style: segment.style,
        })
      }

      const overlapStart = Math.max(range.start, segmentStart)
      const overlapEnd = Math.min(range.end, segmentEnd)
      if (overlapEnd > overlapStart) {
        enriched.push({
          text: fullText.slice(overlapStart, overlapEnd),
          url: range.url,
          style: segment.style,
        })
      }

      localCursor = Math.max(localCursor, overlapEnd)
    }

    if (localCursor < segmentEnd) {
      enriched.push({
        text: fullText.slice(localCursor, segmentEnd),
        style: segment.style,
      })
    }

    cursor = segmentEnd
  }

  return enriched.filter(part => part.text)
}

function normalizeUrl(url: string): string {
  if (/^https?:\/\//i.test(url)) return url
  return `http://${url}`
}

function normalizeLogLine(line: RawLogLine): LogLine {
  const segments = enrichSegments(segmentsFromAnsi(line.text))
  return {
    ...line,
    text: sanitizeLogText(line.text),
    segments,
  }
}

export function openLogUrl(url: string) {
  BrowserOpenURL(url)
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
export const scriptForm = ref<{ id: number; name: string; command: string; workDir: string; envVars: string; notes: string; elevated: boolean; keepWindow: boolean; projectId: number }>({
  id: 0, name: '', command: '', workDir: '', envVars: '', notes: '', elevated: false, keepWindow: false, projectId: 0,
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
    id: 0, name: '', command: '', workDir: '', envVars: '', notes: '', elevated: false, keepWindow: false,
    projectId: selectedProjectId.value || 0,
  }
  scriptFormVisible.value = true
}

export function openEditScript(item: ScriptItem) {
  scriptFormIsEdit.value = true
  scriptForm.value = {
    id: item.id, name: item.name, command: item.command,
    workDir: item.workDir, envVars: item.envVars, notes: item.notes,
    elevated: item.elevated,
    keepWindow: item.keepWindow,
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
        scriptForm.value.elevated,
        scriptForm.value.keepWindow,
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
        scriptForm.value.elevated,
        scriptForm.value.keepWindow,
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
    scriptLogs.value = ((await GetScriptLogs(item.id)) || [])
      .map(normalizeLogLine)
      .filter(line => line.text)
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

export function scriptStatusLabel(status: { status: string; exitCode?: number }): { text: string; type: string } {
  switch (status.status) {
    case 'running':
      return { text: '运行中', type: 'success' }
    case 'exited': {
      const code = status.exitCode ?? 0
      return code === 0
        ? { text: `已退出（退出码 ${code}）`, type: 'success' }
        : { text: `异常退出（退出码 ${code}）`, type: 'danger' }
    }
    default:
      return { text: '已停止', type: 'info' }
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
      const line = normalizeLogLine({
        id: 0,
        scriptId: event.id,
        text: event.text,
        source: event.source,
        timestamp: event.timestamp,
      })
      if (line.text) {
        scriptLogs.value.push(line)
        nextTick(() => {
          if (scriptLogAutoScroll.value) scrollLogToBottom()
        })
      }
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
