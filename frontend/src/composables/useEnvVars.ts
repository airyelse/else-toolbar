import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  GetPathResult,
  SavePathEntries,
  OpenTerminal,
  CleanInvalidUserPaths,
  ListPathProfiles,
  SavePathProfile,
  DeletePathProfile,
  RenamePathProfile,
  PreviewMergeProfile,
  ListEnvVars,
  SetEnvVar,
  DeleteEnvVar,
  ExpandEnvValue,
  OpenInExplorer,
  GetRuntimeConfig,
} from '../../wailsjs/go/main/App'

// ==================== Types ====================
export interface PathEntryItem {
  rawPath: string
  path: string
  exists: boolean
  isDir: boolean
}

export interface EnvVarItem {
  name: string
  value: string
  expandedValue: string
  isPath: boolean
}

export interface FilteredPathItem {
  entry: PathEntryItem
  originalIndex: number
}

export interface PathProfileItem {
  name: string
  paths: string[]
}

// ==================== PATH State ====================
const systemPathEntries = ref<PathEntryItem[]>([])
const userPathEntries = ref<PathEntryItem[]>([])
const userPathStrings = ref<string[]>([])
const pathSearch = ref('')
const pathLoading = ref(false)
const pathTab = ref<'system' | 'user'>('user')
const pathDirty = ref(false)
const pathEditIdx = ref(-1)
const pathEditVal = ref('')
const pathAddVisible = ref(false)
const pathAddVal = ref('')

function filterPathList(list: PathEntryItem[]) {
  if (!pathSearch.value) return list
  const q = pathSearch.value.toLowerCase()
  return list.filter(e => e.path.toLowerCase().includes(q))
}

// Key fix: Filtered items include originalIndex to avoid index bugs
const filteredSystemPath = computed<FilteredPathItem[]>(() => {
  const list = systemPathEntries.value.map((entry, idx) => ({ entry, originalIndex: idx }))
  if (!pathSearch.value) return list
  const q = pathSearch.value.toLowerCase()
  return list.filter(e => e.entry.path.toLowerCase().includes(q))
})

const filteredUserPath = computed<FilteredPathItem[]>(() => {
  const list = userPathEntries.value.map((entry, idx) => ({ entry, originalIndex: idx }))
  if (!pathSearch.value) return list
  const q = pathSearch.value.toLowerCase()
  return list.filter(e => e.entry.path.toLowerCase().includes(q))
})

function pathListStats(list: PathEntryItem[]) {
  const total = list.length
  const valid = list.filter(e => e.exists).length
  const invalid = total - valid
  return { total, valid, invalid }
}

async function loadPathEntries() {
  pathLoading.value = true
  pathDirty.value = false
  try {
    const result = await GetPathResult()
    systemPathEntries.value = result?.system || []
    userPathEntries.value = result?.user || []
    userPathStrings.value = userPathEntries.value.map(e => e.rawPath || e.path)
  } catch (e: any) {
    ElMessage.error(e.message || '加载 PATH 失败')
  }
  pathLoading.value = false
  await loadPathProfiles()
}

function pathStartEdit(idx: number) {
  pathEditIdx.value = idx
  pathEditVal.value = userPathEntries.value[idx].rawPath || userPathEntries.value[idx].path
}

function pathCancelEdit() {
  pathEditIdx.value = -1
  pathEditVal.value = ''
}

function pathConfirmEdit() {
  const val = pathEditVal.value.trim()
  if (!val) return
  userPathStrings.value[pathEditIdx.value] = val
  userPathEntries.value[pathEditIdx.value].rawPath = val
  userPathEntries.value[pathEditIdx.value].path = val
  pathDirty.value = true
  pathEditIdx.value = -1
  pathEditVal.value = ''
}

function pathDelete(idx: number) {
  userPathStrings.value.splice(idx, 1)
  userPathEntries.value.splice(idx, 1)
  pathDirty.value = true
}

function pathMove(idx: number, dir: -1 | 1) {
  const target = idx + dir
  if (target < 0 || target >= userPathStrings.value.length) return
  const tmpStr = userPathStrings.value[idx]
  userPathStrings.value[idx] = userPathStrings.value[target]
  userPathStrings.value[target] = tmpStr
  const tmpEntry = userPathEntries.value[idx]
  userPathEntries.value[idx] = userPathEntries.value[target]
  userPathEntries.value[target] = tmpEntry
  pathDirty.value = true
}

function pathStartAdd() {
  pathAddVal.value = ''
  pathAddVisible.value = true
}

function pathConfirmAdd() {
  const p = pathAddVal.value.trim()
  if (!p) return
  userPathStrings.value.push(p)
  userPathEntries.value.push({ rawPath: p, path: p, exists: false, isDir: false })
  pathDirty.value = true
  pathAddVisible.value = false
  pathAddVal.value = ''
}

async function pathSave() {
  try {
    await SavePathEntries(userPathStrings.value)
    pathDirty.value = false
    ElMessage.success('保存成功，已通知系统更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

const invalidUserPathCount = computed(() => userPathEntries.value.filter(e => !e.exists || !e.isDir).length)

async function handleOpenTerminal() {
  try {
    await OpenTerminal('')
  } catch (e: any) {
    ElMessage.error(e.message || '打开终端失败')
  }
}

async function handleCleanInvalidPaths() {
  const count = invalidUserPathCount.value
  if (count === 0) {
    ElMessage.info('没有无效路径')
    return
  }
  try {
    await ElMessageBox.confirm(
      `检测到 ${count} 条无效路径（不存在或非目录），确定清理？`,
      '清理无效路径',
      { confirmButtonText: '清理', cancelButtonText: '取消', type: 'warning' },
    )
    try {
      const removed = await CleanInvalidUserPaths()
      if (removed && removed.length > 0) {
        ElMessage.success(`已清理 ${removed.length} 条无效路径`)
      } else {
        ElMessage.info('没有需要清理的路径')
      }
      await loadPathEntries()
    } catch (e: any) {
      ElMessage.error(e.message || '清理失败')
    }
  } catch {
    // User cancelled
  }
}

// ==================== PATH Profile State ====================
const pathProfiles = ref<PathProfileItem[]>([])
const selectedProfile = ref('')
const profileSaveDialogVisible = ref(false)
const profileSaveName = ref('')
const profileManageDialogVisible = ref(false)
const profileRenameTarget = ref('')
const profileRenameName = ref('')

// Merge preview state
const mergeDialogVisible = ref(false)
const mergePreview = ref<string[]>([])
const mergeLoading = ref(false)
const mergedSet = computed(() => new Set(mergePreview.value.map(p => p.toLowerCase())))

async function loadPathProfiles() {
  try {
    pathProfiles.value = await ListPathProfiles() || []
    // Auto-create "base" profile from current user PATH if none exist
    if (pathProfiles.value.length === 0 && userPathStrings.value.length > 0) {
      try {
        await SavePathProfile({ name: 'base', paths: [...userPathStrings.value] })
        pathProfiles.value = await ListPathProfiles() || []
      } catch {
        // Ignore error
      }
    }
  } catch {
    // Ignore error
  }
}

async function handleSaveAsProfile() {
  if (!profileSaveName.value.trim()) {
    ElMessage.warning('请输入 Profile 名称')
    return
  }
  try {
    await SavePathProfile({ name: profileSaveName.value.trim(), paths: [...userPathStrings.value] })
    profileSaveDialogVisible.value = false
    profileSaveName.value = ''
    ElMessage.success('Profile 已保存')
    await loadPathProfiles()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function handleApplyProfile(profileName: string) {
  try {
    await ElMessageBox.confirm(
      `确定将 Profile「${profileName}」合并到用户 PATH？`,
      '确认合并',
      { confirmButtonText: '合并', cancelButtonText: '取消', type: 'info' },
    )
    try {
      // Apply merge to local state only, user must click Save to write to registry
      const merged = mergePreview.value.length > 0 ? mergePreview.value : (await PreviewMergeProfile(profileName)) || []
      userPathStrings.value = [...merged]
      userPathEntries.value = merged.map(p => ({ rawPath: p, path: p, exists: false, isDir: false }))
      pathDirty.value = true
      mergeDialogVisible.value = false
      ElMessage.success(`已合并，请点击保存写入注册表`)
    } catch (e: any) {
      ElMessage.error(e.message || '合并失败')
    }
  } catch {
    // User cancelled
  }
}

async function openMergePreview() {
  if (!selectedProfile.value) return
  mergeLoading.value = true
  mergeDialogVisible.value = true
  mergePreview.value = []
  try {
    mergePreview.value = await PreviewMergeProfile(selectedProfile.value) || []
  } catch (e: any) {
    ElMessage.error(e.message || '预览失败')
    mergeDialogVisible.value = false
  }
  mergeLoading.value = false
}

async function handleDeleteProfile(name: string) {
  try {
    await ElMessageBox.confirm(`确定删除 Profile「${name}」？`, '删除 Profile', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    try {
      await DeletePathProfile(name)
      ElMessage.success('已删除')
      await loadPathProfiles()
    } catch (e: any) {
      ElMessage.error(e.message || '删除失败')
    }
  } catch {
    // User cancelled
  }
}

function openRenameProfile(name: string) {
  profileRenameTarget.value = name
  profileRenameName.value = name
}

async function handleRenameProfile() {
  if (!profileRenameName.value.trim()) {
    ElMessage.warning('请输入名称')
    return
  }
  try {
    await RenamePathProfile(profileRenameTarget.value, profileRenameName.value.trim())
    ElMessage.success('已重命名')
    await loadPathProfiles()
    profileRenameTarget.value = ''
    profileRenameName.value = ''
  } catch (e: any) {
    ElMessage.error(e.message || '重命名失败')
  }
}

async function handleUpdateProfilePaths(name: string) {
  try {
    await SavePathProfile({ name, paths: [...userPathStrings.value] })
    ElMessage.success(`已更新 Profile「${name}」`)
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
  }
}

// ==================== Environment Variables State ====================
const envList = ref<EnvVarItem[]>([])
const envLoading = ref(false)
const envSearch = ref('')
const envTab = ref<'user' | 'system'>('user')
// Key fix: Use name-based instead of index-based
const envEditingName = ref('')
const envEditName = ref('')
const envEditValue = ref('')
const envEditIsNew = ref(false)
const envAddVisible = ref(false)
const envAddName = ref('')
const envAddValue = ref('')

// PATH detail view (when user clicks on PATH)
const pathDetailView = ref(false)

// ELSE_RUNTIME_PATH hint
const showRuntimePathHint = ref(false)
const runtimePathValue = ref('')

const filteredEnvList = computed(() => {
  if (!envSearch.value) return envList.value
  const q = envSearch.value.toLowerCase()
  return envList.value.filter(e =>
    e.name.toLowerCase().includes(q) || e.value.toLowerCase().includes(q)
  )
})

async function loadEnvVars() {
  envLoading.value = true
  pathDetailView.value = false
  try {
    const result = await ListEnvVars()
    const source = envTab.value === 'user' ? (result?.user || []) : (result?.system || [])
    const items: EnvVarItem[] = []
    for (const v of source) {
      const expanded = await ExpandEnvValue(v.value)
      items.push({ name: v.name, value: v.value, expandedValue: expanded, isPath: v.isPath })
    }
    // Sort: PATH first, then alphabetical by name
    items.sort((a, b) => {
      if (a.isPath !== b.isPath) return a.isPath ? -1 : 1
      return a.name.localeCompare(b.name)
    })
    envList.value = items
  } catch (e: any) {
    ElMessage.error(e.message || '加载环境变量失败')
  }
  envLoading.value = false

  // Check ELSE_RUNTIME_PATH
  checkElseRuntimePath()
}

async function checkElseRuntimePath() {
  const has = envList.value.some(e => e.name === 'ELSE_RUNTIME_PATH')
  if (has) { showRuntimePathHint.value = false; return }

  try {
    const cfg = await GetRuntimeConfig()
    const baseDir = cfg?.baseDir
    if (!baseDir) return
    runtimePathValue.value = baseDir
    showRuntimePathHint.value = true
  } catch {
    // Ignore error
  }
}

async function createElseRuntimePath() {
  try {
    await SetEnvVar('ELSE_RUNTIME_PATH', runtimePathValue.value, false)
    ElMessage.success('ELSE_RUNTIME_PATH 已创建')
    showRuntimePathHint.value = false
    await loadEnvVars()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  }
}

// Key fix: Use item instead of idx, use envEditingName instead of envEditIdx
function envStartEdit(item: EnvVarItem) {
  envEditingName.value = item.name
  envEditName.value = item.name
  envEditValue.value = item.value
  envEditIsNew.value = false
}

function envCancelEdit() {
  envEditingName.value = ''
}

async function envConfirmEdit() {
  const name = envEditName.value.trim()
  const value = envEditValue.value
  if (!name) { ElMessage.warning('变量名不能为空'); return }
  const isSystem = envTab.value === 'system'
  try {
    await SetEnvVar(name, value, isSystem)
    ElMessage.success('已保存')
    envEditingName.value = ''
    await loadEnvVars()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function envDelete(item: EnvVarItem) {
  try {
    await ElMessageBox.confirm(`确定删除环境变量「${item.name}」？`, '删除确认', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    try {
      await DeleteEnvVar(item.name, envTab.value === 'system')
      ElMessage.success('已删除')
      await loadEnvVars()
    } catch (e: any) {
      ElMessage.error(e.message || '删除失败')
    }
  } catch {
    // User cancelled
  }
}

function envStartAdd() {
  envAddName.value = ''
  envAddValue.value = ''
  envAddVisible.value = true
}

async function envConfirmAdd() {
  const name = envAddName.value.trim()
  const value = envAddValue.value
  if (!name) { ElMessage.warning('请输入变量名'); return }
  const isSystem = envTab.value === 'system'
  try {
    await SetEnvVar(name, value, isSystem)
    envAddVisible.value = false
    ElMessage.success('已添加')
    await loadEnvVars()
  } catch (e: any) {
    ElMessage.error(e.message || '添加失败')
  }
}

function openPathDetail() {
  pathDetailView.value = true
  loadPathEntries()
}

// Helper functions
async function copyPath(p: string) {
  try {
    await navigator.clipboard.writeText(p)
    ElMessage.success('已复制')
  } catch {
    // Ignore error
  }
}

async function openPathDir(p: string) {
  try {
    await OpenInExplorer(p)
  } catch (e: any) {
    ElMessage.error(e.message || '打开失败')
  }
}

// ==================== Export ====================
export function useEnvVars() {
  return {
    // PATH state
    systemPathEntries,
    userPathEntries,
    userPathStrings,
    pathSearch,
    pathLoading,
    pathTab,
    pathDirty,
    pathEditIdx,
    pathEditVal,
    pathAddVisible,
    pathAddVal,
    filteredSystemPath,
    filteredUserPath,
    pathListStats,
    loadPathEntries,
    pathStartEdit,
    pathCancelEdit,
    pathConfirmEdit,
    pathDelete,
    pathMove,
    pathStartAdd,
    pathConfirmAdd,
    pathSave,
    invalidUserPathCount,
    handleOpenTerminal,
    handleCleanInvalidPaths,

    // PATH Profile state
    pathProfiles,
    selectedProfile,
    profileSaveDialogVisible,
    profileSaveName,
    profileManageDialogVisible,
    profileRenameTarget,
    profileRenameName,
    mergeDialogVisible,
    mergePreview,
    mergeLoading,
    mergedSet,
    loadPathProfiles,
    handleSaveAsProfile,
    handleApplyProfile,
    openMergePreview,
    handleDeleteProfile,
    openRenameProfile,
    handleRenameProfile,
    handleUpdateProfilePaths,

    // Environment Variables state
    envList,
    envLoading,
    envSearch,
    envTab,
    envEditingName,
    envEditName,
    envEditValue,
    envEditIsNew,
    envAddVisible,
    envAddName,
    envAddValue,
    pathDetailView,
    showRuntimePathHint,
    runtimePathValue,
    filteredEnvList,
    loadEnvVars,
    checkElseRuntimePath,
    createElseRuntimePath,
    envStartEdit,
    envCancelEdit,
    envConfirmEdit,
    envDelete,
    envStartAdd,
    envConfirmAdd,
    openPathDetail,
    copyPath,
    openPathDir,
  }
}
