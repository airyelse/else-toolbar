<script lang="ts" setup>
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ElTree } from 'element-plus'
import {
  IsInitialized,
  SetupMasterKey,
  Unlock as UnlockVault,
  Lock,
  GetEntries,
  CreateEntry,
  UpdateEntry,
  DeleteEntry,
  GetPassword,
  GetCategoryTree,
  CreateCategory,
  UpdateCategory,
  DeleteCategory,
  GetTags,
  CreateTag,
  UpdateTag,
  DeleteTag,
  GetHelloAvailability,
  IsHelloEnabled,
  SetupHello as SetupHelloBackend,
  UnlockWithHello,
  DisableHello,
  OpenWindowsHelloSettings,
  GetPathEntries,
  GetPathResult,
  OpenInExplorer,
  SavePathEntries,
  ListPathProfiles,
  SavePathProfile,
  DeletePathProfile,
  RenamePathProfile,
  ApplyPathProfile,
  ListEnvVars,
  SetEnvVar,
  DeleteEnvVar,
  ExpandEnvValue,
  ListSDKs,
  InstallSDK,
  UninstallSDK,
  SwitchSDK,
  GetRuntimeConfig,
  SetRuntimeConfig,
  FetchAvailableVersions,
  SelectDirectory as SelectDirDialog,
} from '../wailsjs/go/main/App'
import type { models } from '../wailsjs/go/models'
import { EventsOn } from '../wailsjs/runtime/runtime'

// ==================== Types ====================
interface Entry {
  id: number
  title: string
  username: string
  password?: string
  url: string
  categoryId?: number
  categoryName: string
  tagIds: number[]
  tags: { id: number; name: string; color: string }[]
  notes: string
}

interface CategoryNode {
  id: number
  label: string
  children?: CategoryNode[]
}

interface TagItem {
  id: number
  name: string
  color: string
}

interface PathEntryItem {
  rawPath: string
  path: string
  exists: boolean
  isDir: boolean
}

// ==================== Navigation ====================
type Tool = 'vault' | 'env' | 'runtime'
const currentTool = ref<Tool>('vault')

// ==================== Vault State ====================
const initialized = ref(false)
const unlocked = ref(false)
const loading = ref(false)
const entries = ref<Entry[]>([])
const searchQuery = ref('')

// Sidebar
const categoryTree = ref<CategoryNode[]>([])
const tags = ref<TagItem[]>([])
const selectedCategoryId = ref<number | null>(null)
const selectedTagId = ref<number | null>(null)
const categoryTreeRef = ref<InstanceType<typeof ElTree>>()

// Windows Hello
const helloAvailable = ref(false)
const helloEnabled = ref(false)

// Master password
const masterPassword = ref('')
const confirmPassword = ref('')
const setupDialogVisible = ref(false)
const unlockDialogVisible = ref(false)

// Entry edit
const editDialogVisible = ref(false)
const isEdit = ref(false)
const editForm = ref<Entry>({
  id: 0,
  title: '',
  username: '',
  password: '',
  url: '',
  categoryName: '',
  tagIds: [],
  tags: [],
  notes: '',
})

// Category dialog
const categoryDialogVisible = ref(false)
const categoryForm = ref({ id: 0, name: '', parentId: null as number | null })

// Tag dialog
const tagDialogVisible = ref(false)
const tagForm = ref({ id: 0, name: '', color: '#6366f1' })

// Tag color presets
const tagColorPresets = [
  '#6366f1', '#ec4899', '#ef4444', '#f59e0b', '#10b981',
  '#06b6d4', '#8b5cf6', '#f97316', '#64748b', '#84cc16',
]

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

const filteredSystemPath = computed(() => filterPathList(systemPathEntries.value))
const filteredUserPath = computed(() => filterPathList(userPathEntries.value))

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
  userPathEntries.value[pathEditIdx.value].path = val  // 简化：不做 env 展开，保存后 reload 会更新
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

// ==================== PATH Profile State ====================
interface PathProfileItem {
  name: string
  paths: string[]
}

const pathProfiles = ref<PathProfileItem[]>([])
const profileSaveDialogVisible = ref(false)
const profileSaveName = ref('')
const profileManageDialogVisible = ref(false)
const profileRenameTarget = ref('')
const profileRenameName = ref('')

async function loadPathProfiles() {
  try {
    pathProfiles.value = await ListPathProfiles() || []
  } catch { /* ignore */ }
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
      `将 Profile「${profileName}」的路径合并到当前用户 PATH 前面（自动去重）？`,
      '应用 Profile',
      { confirmButtonText: '应用', cancelButtonText: '取消', type: 'info' },
    )
    await ApplyPathProfile(profileName)
    ElMessage.success(`已应用 Profile「${profileName}」`)
    await loadPathEntries()
    pathDirty.value = false
  } catch { /* cancel */ }
}

async function handleDeleteProfile(name: string) {
  try {
    await ElMessageBox.confirm(`确定删除 Profile「${name}」？`, '删除 Profile', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    await DeletePathProfile(name)
    ElMessage.success('已删除')
    await loadPathProfiles()
  } catch { /* cancel */ }
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
interface EnvVarItem {
  name: string
  value: string
  expandedValue: string
  isPath: boolean
}

const envList = ref<EnvVarItem[]>([])
const envLoading = ref(false)
const envSearch = ref('')
const envTab = ref<'user' | 'system'>('user')
const envEditIdx = ref(-1) // index of item being edited, -1 = none
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
  } catch { /* ignore */ }
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

function envStartEdit(idx: number) {
  envEditIdx.value = idx
  envEditName.value = envList.value[idx].name
  envEditValue.value = envList.value[idx].value
  envEditIsNew.value = false
}

function envCancelEdit() {
  envEditIdx.value = -1
}

async function envConfirmEdit() {
  const name = envEditName.value.trim()
  const value = envEditValue.value
  if (!name) { ElMessage.warning('变量名不能为空'); return }
  const isSystem = envTab.value === 'system'
  try {
    await SetEnvVar(name, value, isSystem)
    ElMessage.success('已保存')
    envEditIdx.value = -1
    await loadEnvVars()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function envDelete(idx: number) {
  const item = envList.value[idx]
  try {
    await ElMessageBox.confirm(`确定删除环境变量「${item.name}」？`, '删除确认', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    await DeleteEnvVar(item.name, envTab.value === 'system')
    ElMessage.success('已删除')
    await loadEnvVars()
  } catch { /* cancel */ }
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

// ==================== Runtime State ====================
interface SDKVersionItem {
  version: string
  path: string
  active: boolean
}

interface SDKInfoItem {
  type: string
  name: string
  icon: string
  installed: SDKVersionItem[]
  current: string
}

const sdkList = ref<SDKInfoItem[]>([])
const sdkLoading = ref(false)
const selectedSdk = ref<string | null>(null)
const installLoading = ref(false)

// Install progress
const installProgress = ref<{ visible: boolean; phase: string; message: string; percent: number }>({
  visible: false, phase: '', message: '', percent: 0,
})

// Available versions
const availableVersions = ref<string[]>([])
const availableLoading = ref(false)
const availableError = ref('')
const versionSearch = ref('')
const sdkRequestSeq = ref(0)

const filteredAvailableVersions = computed(() => {
  if (!versionSearch.value) return availableVersions.value
  const q = versionSearch.value.toLowerCase()
  return availableVersions.value.filter(v => v.toLowerCase().includes(q))
})

async function loadSDKs() {
  sdkLoading.value = true
  try {
    await loadRuntimeConfig()
    sdkList.value = await ListSDKs() || []
    if (sdkList.value.length > 0 && !selectedSdk.value) {
      selectedSdk.value = sdkList.value[0].type
    }
    await loadAvailableVersions()
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  }
  sdkLoading.value = false
}

async function loadAvailableVersions(force = false) {
  if (!selectedSdk.value) return
  availableLoading.value = force
  availableError.value = ''
  const seq = ++sdkRequestSeq.value
  try {
    const result = await FetchAvailableVersions(selectedSdk.value, force) || []
    if (seq !== sdkRequestSeq.value) return // stale response, discard
    availableVersions.value = result
  } catch (e: any) {
    if (seq !== sdkRequestSeq.value) return
    availableError.value = e.message || '获取版本列表失败'
    availableVersions.value = []
  }
  availableLoading.value = false
}

function currentSdkVersions(): SDKVersionItem[] {
  const sdk = sdkList.value.find(s => s.type === selectedSdk.value)
  return sdk?.installed || []
}

function installedVersionSet(): Set<string> {
  return new Set(currentSdkVersions().map(v => v.version))
}

function phaseLabel(phase: string): string {
  switch (phase) {
    case 'download': return '⬇ 下载中'
    case 'extract': return '📦 解压中'
    case 'done': return '✅ 完成'
    default: return phase
  }
}

async function handleInstall(version: string) {
  if (!version || !selectedSdk.value) return
  installLoading.value = true
  installProgress.value = { visible: true, phase: 'download', message: '正在准备...', percent: 0 }
  try {
    await InstallSDK(selectedSdk.value, version)
    ElMessage.success(`${version} 安装成功`)
    await loadSDKs()
  } catch (e: any) {
    console.error('InstallSDK failed:', e)
    ElMessage.error(e?.message || e?.toString() || '安装失败')
  } finally {
    installLoading.value = false
    setTimeout(() => { installProgress.value.visible = false }, 1500)
  }
}

// ==================== Event Listeners ====================

// EventsOn is registered in onMounted to avoid duplicate subscriptions on HMR

async function handleUninstall(version: string) {
  if (!selectedSdk.value) return
  try {
    await ElMessageBox.confirm(`确定卸载 ${version}？`, '卸载确认', {
      confirmButtonText: '卸载',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await UninstallSDK(selectedSdk.value, version)
    ElMessage.success(`${version} 已卸载`)
    await loadSDKs()
  } catch { /* cancel */ }
}

async function handleSwitch(version: string) {
  if (!selectedSdk.value) return
  try {
    await SwitchSDK(selectedSdk.value, version)
    ElMessage.success(`已切换到 ${version}`)
    await loadSDKs()
  } catch (e: any) {
    ElMessage.error(e.message || '切换失败')
  }
}

const runtimeBaseDir = ref('')
const configDialogVisible = ref(false)

async function loadRuntimeConfig() {
  try {
    const cfg = await GetRuntimeConfig()
    runtimeBaseDir.value = cfg.baseDir || ''
  } catch { /* ignore */ }
}

function openConfigDialog() {
  loadRuntimeConfig()
  configDialogVisible.value = true
}

async function handleSaveConfig() {
  if (!runtimeBaseDir.value.trim()) {
    ElMessage.warning('请输入目录路径')
    return
  }
  try {
    await SetRuntimeConfig(runtimeBaseDir.value.trim())
    configDialogVisible.value = false
    ElMessage.success('配置已保存，列表将刷新')
    await loadSDKs()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function handleBrowseRuntimeDir() {
  try {
    const dir = await SelectDirDialog()
    if (dir) runtimeBaseDir.value = dir
  } catch { /* ignore */ }
}

async function copyPath(p: string) {
  try {
    await navigator.clipboard.writeText(p)
    ElMessage.success('已复制')
  } catch { /* ignore */ }
}

async function openPathDir(p: string) {
  try {
    await OpenInExplorer(p)
  } catch (e: any) {
    ElMessage.error(e.message || '打开失败')
  }
}

// ==================== Vault Computed ====================
const filteredEntries = computed(() => {
  if (!searchQuery.value) return entries.value
  const q = searchQuery.value.toLowerCase()
  return entries.value.filter(e =>
    e.title.toLowerCase().includes(q) ||
    e.username.toLowerCase().includes(q) ||
    e.url.toLowerCase().includes(q) ||
    e.categoryName.toLowerCase().includes(q) ||
    e.tags?.some(t => t.name.toLowerCase().includes(q))
  )
})

const totalEntries = ref(0)

const currentFilterLabel = computed(() => {
  if (selectedCategoryId.value) {
    const node = findCategoryNode(categoryTree.value, selectedCategoryId.value)
    return node ? node.label : ''
  }
  if (selectedTagId.value) {
    const tag = tags.value.find(t => t.id === selectedTagId.value)
    return tag ? tag.name : ''
  }
  return '全部密码'
})

// ==================== Helpers ====================
function findCategoryNode(nodes: CategoryNode[], id: number): CategoryNode | null {
  for (const n of nodes) {
    if (n.id === id) return n
    if (n.children) {
      const found = findCategoryNode(n.children, id)
      if (found) return found
    }
  }
  return null
}

function flattenCategories(nodes: CategoryNode[]): CategoryNode[] {
  const result: CategoryNode[] = []
  for (const n of nodes) {
    result.push(n)
    if (n.children) result.push(...flattenCategories(n.children))
  }
  return result
}

function getInitial(title: string): string {
  return title.charAt(0).toUpperCase()
}

function getAvatarColor(entry: Entry): string {
  if (entry.tags?.length > 0) return entry.tags[0].color
  return '#6366f1'
}

// ==================== Init ====================
let unlistenProgress: (() => void) | null = null

onMounted(async () => {
  initialized.value = await IsInitialized()
  if (!initialized.value) {
    setupDialogVisible.value = true
  }
  const availability = await GetHelloAvailability()
  helloAvailable.value = availability !== 'DeviceNotPresent' && availability !== 'Unknown'
  if (initialized.value) {
    helloEnabled.value = await IsHelloEnabled()
  }

  // Register event listener in onMounted to avoid duplicate subscriptions on HMR
  unlistenProgress = EventsOn('sdk:progress', (event: any) => {
    installProgress.value = { visible: true, phase: event.phase, message: event.message, percent: event.percent }
  })
})

onUnmounted(() => {
  if (typeof unlistenProgress === 'function') unlistenProgress()
})

// ==================== Auth ====================
async function handleSetup() {
  if (masterPassword.value.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  if (masterPassword.value !== confirmPassword.value) {
    ElMessage.error('两次密码不一致')
    return
  }
  try {
    await SetupMasterKey(masterPassword.value)
    initialized.value = true
    unlocked.value = true
    setupDialogVisible.value = false
    masterPassword.value = ''
    confirmPassword.value = ''
    await loadAll()
    helloEnabled.value = await IsHelloEnabled()
    ElMessage.success('设置成功')
    if (helloAvailable.value) {
      await promptEnableHello()
    }
  } catch (e: any) {
    ElMessage.error(e.message || '设置失败')
  }
}

async function handleUnlock() {
  try {
    const ok = await UnlockVault(masterPassword.value)
    if (ok) {
      unlocked.value = true
      unlockDialogVisible.value = false
      masterPassword.value = ''
      await loadAll()
      helloEnabled.value = await IsHelloEnabled()
      ElMessage.success('解锁成功')
    } else {
      ElMessage.error('密码错误')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '解锁失败')
  }
}

async function handleLock() {
  await Lock()
  unlocked.value = false
  masterPassword.value = ''
}

// ==================== Data Loading ====================
async function loadAll() {
  await Promise.all([loadEntries(), loadCategories(), loadTags()])
}

async function loadEntries() {
  loading.value = true
  try {
    const hasFilter = selectedCategoryId.value != null || selectedTagId.value != null
    entries.value = await GetEntries(
      selectedCategoryId.value ?? null,
      selectedTagId.value ?? null,
    ) || []
    if (!hasFilter) {
      totalEntries.value = entries.value.length
    }
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  }
  loading.value = false
}

async function loadCategories() {
  try {
    const tree = await GetCategoryTree() || []
    categoryTree.value = tree.map(mapCategoryNode)
  } catch { /* ignore */ }
}

function mapCategoryNode(c: any): CategoryNode {
  return {
    id: c.id,
    label: c.name,
    children: c.children?.length ? c.children.map(mapCategoryNode) : undefined,
  }
}

async function loadTags() {
  try {
    tags.value = await GetTags() || []
  } catch { /* ignore */ }
}

// ==================== Sidebar Navigation ====================
function selectAll() {
  selectedCategoryId.value = null
  selectedTagId.value = null
  categoryTreeRef.value?.setCurrentKey(null)
  loadEntries()
}

function handleCategoryClick(data: CategoryNode) {
  selectedCategoryId.value = data.id
  selectedTagId.value = null
  loadEntries()
}

function handleTagClick(tag: TagItem) {
  if (selectedTagId.value === tag.id) {
    selectedTagId.value = null
  } else {
    selectedTagId.value = tag.id
    selectedCategoryId.value = null
    categoryTreeRef.value?.setCurrentKey(null)
  }
  loadEntries()
}

// ==================== Category CRUD ====================
function openAddCategory(parentId?: number) {
  categoryForm.value = { id: 0, name: '', parentId: parentId ?? null }
  categoryDialogVisible.value = true
}

function findParentId(nodes: CategoryNode[], targetId: number): number | null {
  for (const node of nodes) {
    if (node.children) {
      for (const child of node.children) {
        if (child.id === targetId) return node.id
      }
      const found = findParentId(node.children, targetId)
      if (found !== null) return found
    }
  }
  return null
}

function openEditCategory(node: CategoryNode) {
  categoryForm.value = { id: node.id, name: node.label, parentId: findParentId(categoryTree.value, node.id) }
  categoryDialogVisible.value = true
}

async function handleSaveCategory() {
  if (!categoryForm.value.name) {
    ElMessage.warning('请输入分类名称')
    return
  }
  try {
    if (categoryForm.value.id) {
      await UpdateCategory(categoryForm.value.id, categoryForm.value.name, categoryForm.value.parentId ?? null)
      ElMessage.success('更新成功')
    } else {
      await CreateCategory(categoryForm.value.name, categoryForm.value.parentId ?? null)
      ElMessage.success('创建成功')
    }
    categoryDialogVisible.value = false
    await loadCategories()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDeleteCategory(node: CategoryNode) {
  try {
    await ElMessageBox.confirm(`确定要删除分类「${node.label}」吗？子分类会移至根级，条目不会被删除。`, '删除分类', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await DeleteCategory(node.id)
    if (selectedCategoryId.value === node.id) selectedCategoryId.value = null
    ElMessage.success('删除成功')
    await Promise.all([loadCategories(), loadEntries()])
  } catch { /* cancel */ }
}

// ==================== Tag CRUD ====================
function openAddTag() {
  tagForm.value = { id: 0, name: '', color: '#6366f1' }
  tagDialogVisible.value = true
}

function openEditTag(tag: TagItem) {
  tagForm.value = { id: tag.id, name: tag.name, color: tag.color }
  tagDialogVisible.value = true
}

async function handleSaveTag() {
  if (!tagForm.value.name) {
    ElMessage.warning('请输入标签名称')
    return
  }
  try {
    if (tagForm.value.id) {
      await UpdateTag(tagForm.value.id, tagForm.value.name, tagForm.value.color)
      ElMessage.success('更新成功')
    } else {
      await CreateTag(tagForm.value.name, tagForm.value.color)
      ElMessage.success('创建成功')
    }
    tagDialogVisible.value = false
    await loadTags()
    await loadEntries()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDeleteTag(tag: TagItem) {
  try {
    await ElMessageBox.confirm(`确定要删除标签「${tag.name}」吗？`, '删除标签', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await DeleteTag(tag.id)
    if (selectedTagId.value === tag.id) selectedTagId.value = null
    ElMessage.success('删除成功')
    await Promise.all([loadTags(), loadEntries()])
  } catch { /* cancel */ }
}

// ==================== Entry CRUD ====================
function openAddDialog() {
  isEdit.value = false
  editForm.value = {
    id: 0, title: '', username: '', password: '', url: '',
    categoryName: '', tagIds: [], tags: [], notes: '',
  }
  editDialogVisible.value = true
}

function openEditDialog(entry: Entry) {
  isEdit.value = true
  editForm.value = {
    ...entry,
    password: '',
    tagIds: entry.tags.map(t => t.id),
  }
  editDialogVisible.value = true
}

async function handleSave() {
  if (!editForm.value.title) {
    ElMessage.warning('请输入标题')
    return
  }
  try {
    const dto: any = {
      title: editForm.value.title,
      username: editForm.value.username,
      password: editForm.value.password,
      url: editForm.value.url,
      notes: editForm.value.notes,
      tagIds: editForm.value.tagIds,
    }
    if (editForm.value.categoryId) {
      dto.categoryId = editForm.value.categoryId
    }

    if (isEdit.value) {
      dto.id = editForm.value.id
      await UpdateEntry(dto)
      ElMessage.success('更新成功')
    } else {
      if (!editForm.value.password) {
        ElMessage.warning('请输入密码')
        return
      }
      await CreateEntry(dto)
      ElMessage.success('创建成功')
    }
    editDialogVisible.value = false
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function handleDelete(entry: Entry) {
  try {
    await ElMessageBox.confirm(`确定要删除「${entry.title}」吗？`, '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await DeleteEntry(entry.id)
    ElMessage.success('删除成功')
    await loadAll()
  } catch { /* cancel */ }
}

// ==================== Windows Hello ====================
async function promptEnableHello() {
  try {
    await ElMessageBox.confirm(
      '是否启用 Windows Hello 快速解锁？启用后可使用指纹或 PIN 快速解锁密码库。',
      '启用 Windows Hello',
      {
        confirmButtonText: '启用',
        cancelButtonText: '以后再说',
        type: 'info',
      },
    )
    const ok = await registerHello()
    if (ok) {
      ElMessage.success('Windows Hello 已启用')
    }
  } catch { /* user cancelled */ }
}

async function openHelloSettings() {
  try {
    await OpenWindowsHelloSettings()
  } catch (e: any) {
    ElMessage.error(e.message || '无法打开 Windows 登录选项')
  }
}

async function offerOpenHelloSettings(message: string) {
  try {
    await ElMessageBox.confirm(message, 'Windows Hello', {
      confirmButtonText: '打开设置',
      cancelButtonText: '取消',
      type: 'info',
    })
    await openHelloSettings()
  } catch { /* user cancelled */ }
}

async function registerHello() {
  try {
    const availability = await GetHelloAvailability()
    helloAvailable.value = availability !== 'DeviceNotPresent' && availability !== 'Unknown'
    if (!helloAvailable.value) {
      ElMessage.warning('此设备不支持 Windows Hello')
      await offerOpenHelloSettings('当前设备未启用或暂时无法使用 Windows Hello，是否打开系统登录选项进行检查？')
      return false
    }
    await SetupHelloBackend()
    helloEnabled.value = true
    return true
  } catch (e: any) {
    ElMessage.error(e.message || '注册失败')
    await offerOpenHelloSettings('当前窗口未能直接拉起 Windows Hello，是否打开系统登录选项？')
    return false
  }
}

async function handleHelloUnlock() {
  try {
    const ok = await UnlockWithHello()
    if (ok) {
      unlocked.value = true
      unlockDialogVisible.value = false
      await loadAll()
      helloEnabled.value = true
      ElMessage.success('解锁成功')
    } else {
      ElMessage.error('Windows Hello 解锁失败，请使用密码')
      await offerOpenHelloSettings('当前窗口未能完成 Windows Hello 验证，是否打开系统登录选项？')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '验证失败')
    await offerOpenHelloSettings('当前窗口未能直接拉起 Windows Hello，是否打开系统登录选项？')
  }
}

async function toggleHello() {
  if (helloEnabled.value) {
    try {
      await ElMessageBox.confirm('确定要关闭 Windows Hello 解锁吗？', '关闭 Windows Hello', {
        confirmButtonText: '关闭',
        cancelButtonText: '取消',
        type: 'warning',
      })
      await DisableHello()
      helloEnabled.value = false
      ElMessage.success('Windows Hello 已关闭')
    } catch { /* cancelled */ }
  } else {
    const ok = await registerHello()
    if (ok) {
      ElMessage.success('Windows Hello 已启用')
    }
  }
}

// ==================== Clipboard ====================
async function copyPassword(entry: Entry) {
  try {
    const pwd = await GetPassword(entry.id)
    await navigator.clipboard.writeText(pwd)
    ElMessage.success('密码已复制到剪贴板（30秒后自动清除）')
    setTimeout(async () => {
      try {
        const current = await navigator.clipboard.readText()
        if (current === pwd) {
          await navigator.clipboard.writeText('')
        }
      } catch { /* ignore */ }
    }, 30000)
  } catch (e: any) {
    ElMessage.error(e.message || '复制失败')
  }
}

async function copyUsername(username: string) {
  await navigator.clipboard.writeText(username)
  ElMessage.success('用户名已复制到剪贴板')
}

function openUrl(url: string) {
  if (!url) return
  try {
    const parsed = new URL(url)
    if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  } catch { /* ignore invalid URL */ }
}
</script>

<template>
  <div class="app-container">
    <!-- Left Nav Rail -->
    <nav class="nav-rail">
      <div class="nav-rail-logo">
        <div class="logo-icon"><el-icon size="18"><Box /></el-icon></div>
      </div>
      <div class="nav-rail-items">
        <el-tooltip content="密码保险箱" placement="right">
          <div
            class="nav-rail-item"
            :class="{ active: currentTool === 'vault' }"
            @click="currentTool = 'vault'"
          >
            <el-icon size="20"><Lock /></el-icon>
          </div>
        </el-tooltip>
        <el-tooltip content="环境变量" placement="right">
          <div
            class="nav-rail-item"
            :class="{ active: currentTool === 'env' }"
            @click="currentTool = 'env'; loadEnvVars()"
          >
            <el-icon size="20"><Guide /></el-icon>
          </div>
        </el-tooltip>
        <el-tooltip content="环境管理" placement="right">
          <div
            class="nav-rail-item"
            :class="{ active: currentTool === 'runtime' }"
            @click="currentTool = 'runtime'; loadSDKs()"
          >
            <el-icon size="20"><Monitor /></el-icon>
          </div>
        </el-tooltip>
      </div>
    </nav>

    <!-- Right area: Header + Body -->
    <div class="app-main">
      <!-- Header -->
      <header class="header">
        <div class="header-left">
          <h1 class="header-title">{{ currentTool === 'vault' ? '密码保险箱' : currentTool === 'env' ? '环境变量' : '环境管理' }}</h1>
        </div>
        <div class="header-actions" v-if="currentTool === 'vault' && unlocked">
          <el-input v-model="searchQuery" placeholder="搜索..." clearable class="search-input">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" @click="openAddDialog" class="btn-add">
            <el-icon><Plus /></el-icon><span>新增</span>
          </el-button>
          <el-button @click="handleLock" class="btn-lock">
            <el-icon><SwitchButton /></el-icon><span>锁定</span>
          </el-button>
          <el-tooltip :content="helloEnabled ? '关闭 Windows Hello' : '启用 Windows Hello'" v-if="helloAvailable">
            <el-button
              text
              class="btn-hello"
              :type="helloEnabled ? 'primary' : ''"
              @click="toggleHello"
            >
              <el-icon><Key /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
        <div class="header-actions" v-else-if="currentTool === 'vault' && initialized">
          <el-button type="primary" @click="unlockDialogVisible = true">
            <el-icon><Unlock /></el-icon><span>解锁密码库</span>
          </el-button>
        </div>
        <div class="header-actions" v-if="currentTool === 'env'">
          <el-input v-model="envSearch" placeholder="搜索变量..." clearable class="search-input">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" @click="loadEnvVars" class="btn-add">
            <el-icon><Refresh /></el-icon><span>刷新</span>
          </el-button>
        </div>
      </header>

      <!-- ==================== VAULT TOOL ==================== -->
      <template v-if="currentTool === 'vault'">
        <!-- Body: Sidebar + Main -->
        <div class="body" v-if="unlocked">
          <!-- Sidebar -->
          <aside class="sidebar">
            <div class="sidebar-section">
              <div
                class="sidebar-item"
                :class="{ active: !selectedCategoryId && !selectedTagId }"
                @click="selectAll"
              >
                <el-icon><Grid /></el-icon>
                <span class="sidebar-label">全部密码</span>
                <span class="sidebar-count">{{ totalEntries }}</span>
              </div>
            </div>

            <!-- Categories -->
            <div class="sidebar-section">
              <div class="sidebar-section-header">
                <span class="sidebar-section-title">分类</span>
                <el-button text size="small" @click="openAddCategory()" class="sidebar-add-btn">
                  <el-icon><Plus /></el-icon>
                </el-button>
              </div>
              <el-tree
                ref="categoryTreeRef"
                :data="categoryTree"
                :props="{ label: 'label', children: 'children' }"
                highlight-current
                :expand-on-click-node="false"
                node-key="id"
                @node-click="handleCategoryClick"
                class="category-tree"
                :indent="8"
              >
                <template #default="{ node, data }">
                  <div class="tree-node">
                    <el-icon size="14" class="tree-node-icon"><Folder /></el-icon>
                    <span class="tree-node-label">{{ data.label }}</span>
                    <span class="tree-node-actions" v-if="!node.expanded || data.children?.length === 0">
                      <el-icon size="14" @click.stop="openEditCategory(data)"><Edit /></el-icon>
                      <el-icon size="14" @click.stop="handleDeleteCategory(data)"><Delete /></el-icon>
                    </span>
                  </div>
                </template>
              </el-tree>
              <div class="sidebar-empty" v-if="categoryTree.length === 0">
                暂无分类
              </div>
            </div>

            <!-- Tags -->
            <div class="sidebar-section">
              <div class="sidebar-section-header">
                <span class="sidebar-section-title">标签</span>
                <el-button text size="small" @click="openAddTag()" class="sidebar-add-btn">
                  <el-icon><Plus /></el-icon>
                </el-button>
              </div>
              <div class="tag-list">
                <div
                  v-for="tag in tags"
                  :key="tag.id"
                  class="tag-item"
                  :class="{ active: selectedTagId === tag.id }"
                  @click="handleTagClick(tag)"
                >
                  <span class="tag-dot" :style="{ background: tag.color }"></span>
                  <span class="tag-name">{{ tag.name }}</span>
                  <el-icon size="14" class="tag-edit" @click.stop="openEditTag(tag)"><Edit /></el-icon>
                  <el-icon size="14" class="tag-delete" @click.stop="handleDeleteTag(tag)"><Delete /></el-icon>
                </div>
              </div>
              <div class="sidebar-empty" v-if="tags.length === 0">
                暂无标签
              </div>
            </div>
          </aside>

          <!-- Main Content -->
          <main class="main-content">
            <div class="content-header">
              <h2 class="content-title">{{ currentFilterLabel }}</h2>
              <span class="content-count">{{ filteredEntries.length }} 条</span>
            </div>

          <!-- Entries Grid -->
          <transition-group name="card" tag="div" class="entries-grid" v-loading="loading">
              <div v-for="entry in filteredEntries" :key="entry.id" class="entry-card">
                <div class="card-header">
                  <div class="entry-avatar" :style="{ background: getAvatarColor(entry) }">
                    {{ getInitial(entry.title) }}
                  </div>
                  <div class="entry-info">
                    <h3 class="entry-title">{{ entry.title }}</h3>
                    <span class="entry-username">{{ entry.username }}</span>
                  </div>
                </div>
                <div class="card-meta" v-if="entry.categoryName || entry.tags?.length">
                  <el-tag v-if="entry.categoryName" size="small" effect="plain" class="meta-tag">
                    <el-icon size="12"><Folder /></el-icon>
                    {{ entry.categoryName }}
                  </el-tag>
                  <el-tag
                    v-for="tag in entry.tags"
                    :key="tag.id"
                    size="small"
                    :color="tag.color"
                    effect="dark"
                    class="meta-tag"
                  >
                    {{ tag.name }}
                  </el-tag>
                </div>
                <div class="card-body" v-if="entry.url || entry.notes">
                  <div class="entry-url" v-if="entry.url" @click="openUrl(entry.url)">
                    <el-icon size="13"><Link /></el-icon>
                    <span>{{ entry.url }}</span>
                  </div>
                  <div class="entry-notes" v-if="entry.notes">{{ entry.notes }}</div>
                </div>
                <div class="card-actions">
                  <el-button text size="small" @click="copyUsername(entry.username)" class="action-btn">
                    <el-icon><User /></el-icon><span>用户名</span>
                  </el-button>
                  <el-button text size="small" @click="copyPassword(entry)" class="action-btn action-primary">
                    <el-icon><Key /></el-icon><span>密码</span>
                  </el-button>
                  <div class="action-spacer"></div>
                  <el-button text size="small" @click="openEditDialog(entry)" class="action-btn">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="handleDelete(entry)" class="action-btn action-danger">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </transition-group>

          <!-- Empty -->
          <div class="empty-state" v-if="!loading && entries.length === 0">
            <div class="empty-icon"><el-icon size="48"><Box /></el-icon></div>
            <h3 class="empty-title">密码库为空</h3>
            <p class="empty-desc">点击「新增」按钮添加你的第一个密码</p>
            <el-button type="primary" size="large" @click="openAddDialog" round>
              <el-icon><Plus /></el-icon> 添加第一个密码
            </el-button>
          </div>

          <!-- No search results -->
          <div class="empty-state" v-if="!loading && entries.length > 0 && filteredEntries.length === 0">
            <div class="empty-icon"><el-icon size="48"><Search /></el-icon></div>
            <h3 class="empty-title">未找到匹配结果</h3>
            <p class="empty-desc">试试其他关键词</p>
          </div>
        </main>
      </div>

      <!-- Locked State -->
      <div class="locked-state" v-else-if="initialized">
        <div class="locked-card">
          <div class="locked-icon"><el-icon size="56"><Lock /></el-icon></div>
          <h2 class="locked-title">密码库已锁定</h2>
          <p class="locked-desc">点击下方按钮输入主密码以解锁</p>
          <el-button type="primary" size="large" @click="unlockDialogVisible = true" round>
            <el-icon><Unlock /></el-icon> 解锁密码库
          </el-button>
        </div>
      </div>
    </template>

    <!-- ==================== ENV TOOL ==================== -->
    <template v-if="currentTool === 'env'">
      <div class="body" v-loading="envLoading">
        <main class="main-content" style="flex: 1">
          <!-- PATH detail view (existing PATH management) -->
          <template v-if="pathDetailView">
            <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 12px">
              <el-button size="small" @click="loadEnvVars" class="path-action-btn">
                <el-icon><ArrowLeft /></el-icon><span>返回环境变量</span>
              </el-button>
              <h2 style="font-size: 16px; font-weight: 600; color: var(--text); margin: 0">PATH 管理</h2>
            </div>

            <div class="path-tabs">
              <div
                class="path-tab"
                :class="{ active: pathTab === 'user' }"
                @click="pathTab = 'user'"
              >
                用户变量
                <span class="path-tab-count">{{ userPathEntries.length }}</span>
              </div>
              <div
                class="path-tab"
                :class="{ active: pathTab === 'system' }"
                @click="pathTab = 'system'"
              >
                系统变量
                <span class="path-tab-count">{{ systemPathEntries.length }}</span>
              </div>
              <div style="flex:1"></div>
              <template v-if="pathTab === 'user'">
                <el-button size="small" @click="pathStartAdd" class="path-action-btn">
                  <el-icon><Plus /></el-icon><span>添加</span>
                </el-button>
                <el-button size="small" type="primary" @click="pathSave" :disabled="!pathDirty" class="path-action-btn">
                  <el-icon><Check /></el-icon><span>保存</span>
                </el-button>
              </template>
            </div>

            <!-- User PATH (editable) -->
            <div class="path-list" v-if="pathTab === 'user'">
              <!-- Profile Bar -->
              <div class="profile-bar" v-if="pathProfiles.length > 0 || true">
                <div class="profile-bar-label">Profile</div>
                <el-select
                  placeholder="选择 Profile"
                  size="small"
                  clearable
                  style="width: 200px"
                  @change="(val: string) => { if (val) handleApplyProfile(val) }"
                >
                  <el-option
                    v-for="p in pathProfiles"
                    :key="p.name"
                    :label="p.name"
                    :value="p.name"
                  >
                    <div style="display: flex; justify-content: space-between; align-items: center">
                      <span>{{ p.name }}</span>
                      <span style="color: var(--text-muted); font-size: 11px">{{ p.paths.length }} 条</span>
                    </div>
                  </el-option>
                </el-select>
                <el-button size="small" @click="profileSaveDialogVisible = true; profileSaveName = ''" class="path-action-btn">
                  <el-icon><FolderAdd /></el-icon><span>保存为 Profile</span>
                </el-button>
                <el-button size="small" @click="profileManageDialogVisible = true" class="path-action-btn" :disabled="pathProfiles.length === 0">
                  <el-icon><Setting /></el-icon><span>管理</span>
                </el-button>
              </div>
              <div class="path-list-header">
                <div class="path-stats">
                  <el-tag type="success" effect="plain" size="small">{{ pathListStats(userPathEntries).valid }} 有效</el-tag>
                  <el-tag type="danger" effect="plain" size="small" v-if="pathListStats(userPathEntries).invalid > 0">{{ pathListStats(userPathEntries).invalid }} 无效</el-tag>
                </div>
              </div>
              <div
                v-for="(entry, idx) in filteredUserPath"
                :key="'u-' + idx"
                class="path-entry"
                :class="{ 'path-invalid': !entry.exists }"
              >
                <div class="path-entry-index">{{ idx + 1 }}</div>
                <div class="path-entry-body" v-if="pathEditIdx !== userPathEntries.indexOf(entry)">
                  <div class="path-entry-value" :title="entry.path">{{ entry.path }}</div>
                  <div class="path-entry-meta">
                    <el-tag v-if="entry.exists" type="success" size="small" effect="light">存在</el-tag>
                    <el-tag v-else type="danger" size="small" effect="light">不存在</el-tag>
                    <el-tag v-if="entry.isDir" size="small" effect="light">目录</el-tag>
                    <el-tag v-else-if="entry.exists" type="warning" size="small" effect="light">非目录</el-tag>
                  </div>
                </div>
                <div class="path-entry-body" v-else>
                  <el-input v-model="pathEditVal" size="small" class="path-edit-input" @keyup.enter="pathConfirmEdit" @keyup.escape="pathCancelEdit">
                    <template #append>
                      <el-button @click="pathConfirmEdit" size="small"><el-icon><Check /></el-icon></el-button>
                      <el-button @click="pathCancelEdit" size="small"><el-icon><Close /></el-icon></el-button>
                    </template>
                  </el-input>
                </div>
                <div class="path-entry-actions path-entry-actions-always">
                  <el-button text size="small" @click="pathMove(userPathEntries.indexOf(entry), -1)" class="action-btn" :disabled="idx === 0">
                    <el-icon><Top /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="pathMove(userPathEntries.indexOf(entry), 1)" class="action-btn" :disabled="idx >= filteredUserPath.length - 1">
                    <el-icon><Bottom /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="pathStartEdit(userPathEntries.indexOf(entry))" class="action-btn">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="copyPath(entry.path)" class="action-btn">
                    <el-icon><CopyDocument /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="openPathDir(entry.path)" class="action-btn" v-if="entry.exists && entry.isDir">
                    <el-icon><FolderOpened /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="pathDelete(userPathEntries.indexOf(entry))" class="action-btn action-danger">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
              <div class="sidebar-empty" v-if="userPathEntries.length === 0">未检测到用户 PATH</div>
            </div>

            <!-- System PATH (read-only) -->
            <div class="path-list" v-if="pathTab === 'system'">
              <div class="path-list-header">
                <div class="path-stats">
                  <el-tag type="success" effect="plain" size="small">{{ pathListStats(systemPathEntries).valid }} 有效</el-tag>
                  <el-tag type="danger" effect="plain" size="small" v-if="pathListStats(systemPathEntries).invalid > 0">{{ pathListStats(systemPathEntries).invalid }} 无效</el-tag>
                  <el-tag effect="plain" size="small" type="info">只读</el-tag>
                </div>
              </div>
              <div
                v-for="(entry, idx) in filteredSystemPath"
                :key="'s-' + idx"
                class="path-entry"
                :class="{ 'path-invalid': !entry.exists }"
              >
                <div class="path-entry-index">{{ idx + 1 }}</div>
                <div class="path-entry-body">
                  <div class="path-entry-value" :title="entry.path">{{ entry.path }}</div>
                  <div class="path-entry-meta">
                    <el-tag v-if="entry.exists" type="success" size="small" effect="light">存在</el-tag>
                    <el-tag v-else type="danger" size="small" effect="light">不存在</el-tag>
                    <el-tag v-if="entry.isDir" size="small" effect="light">目录</el-tag>
                    <el-tag v-else-if="entry.exists" type="warning" size="small" effect="light">非目录</el-tag>
                  </div>
                </div>
                <div class="path-entry-actions">
                  <el-button text size="small" @click="copyPath(entry.path)" class="action-btn">
                    <el-icon><CopyDocument /></el-icon>
                  </el-button>
                  <el-button text size="small" @click="openPathDir(entry.path)" class="action-btn" v-if="entry.exists && entry.isDir">
                    <el-icon><FolderOpened /></el-icon>
                  </el-button>
                </div>
              </div>
              <div class="sidebar-empty" v-if="systemPathEntries.length === 0">未检测到系统 PATH</div>
            </div>
          </template>

          <!-- General env var list view -->
          <template v-else>
            <!-- ELSE_RUNTIME_PATH hint banner -->
            <div class="env-hint-banner" v-if="showRuntimePathHint && envTab === 'user'">
              <div class="env-hint-text">
                <el-icon size="16" style="color: var(--primary); flex-shrink: 0"><InfoFilled /></el-icon>
                <span>建议创建 <strong>ELSE_RUNTIME_PATH</strong> 用户变量，指向运行时基础目录，方便终端引用 SDK 路径。</span>
                <code class="env-hint-code">{{ runtimePathValue }}</code>
              </div>
              <div class="env-hint-actions">
                <el-button size="small" type="primary" @click="createElseRuntimePath">创建</el-button>
                <el-button size="small" text @click="showRuntimePathHint = false">忽略</el-button>
              </div>
            </div>

            <div class="env-tabs">
              <div class="path-tab" :class="{ active: envTab === 'user' }" @click="envTab = 'user'; loadEnvVars()">
                用户变量
                <span class="path-tab-count">{{ envList.length }}</span>
              </div>
              <div class="path-tab" :class="{ active: envTab === 'system' }" @click="envTab = 'system'; loadEnvVars()">
                系统变量
                <span class="path-tab-count">{{ envList.length }}</span>
              </div>
              <div style="flex:1"></div>
              <el-button size="small" @click="envStartAdd" class="path-action-btn">
                <el-icon><Plus /></el-icon><span>添加</span>
              </el-button>
            </div>

            <div class="env-list">
              <div v-for="(item, idx) in filteredEnvList" :key="item.name" class="env-entry" :class="{ 'env-entry-path': item.isPath }">
                <div class="env-entry-name">{{ item.name }}</div>
                <div class="env-entry-body" v-if="envEditIdx !== idx">
                  <div class="env-entry-value" :title="item.expandedValue">{{ item.expandedValue || item.value }}</div>
                </div>
                <div class="env-entry-body" v-else>
                  <div style="display: flex; flex-direction: column; gap: 4px; flex: 1">
                    <el-input v-model="envEditName" size="small" placeholder="变量名" :disabled="!envEditIsNew" />
                    <el-input v-model="envEditValue" size="small" placeholder="变量值" />
                  </div>
                </div>
                <div class="env-entry-actions">
                  <template v-if="item.isPath">
                    <el-button text size="small" @click="openPathDetail" class="action-btn action-primary">
                      <el-icon><Guide /></el-icon><span>管理</span>
                    </el-button>
                  </template>
                  <template v-if="envEditIdx === idx">
                    <el-button text size="small" @click="envConfirmEdit" class="action-btn" type="primary">
                      <el-icon><Check /></el-icon>
                    </el-button>
                    <el-button text size="small" @click="envCancelEdit" class="action-btn">
                      <el-icon><Close /></el-icon>
                    </el-button>
                  </template>
                  <template v-else>
                    <el-button text size="small" @click="envStartEdit(idx)" class="action-btn">
                      <el-icon><Edit /></el-icon>
                    </el-button>
                    <el-button text size="small" @click="envDelete(idx)" class="action-btn action-danger">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </template>
                </div>
              </div>
              <div class="sidebar-empty" v-if="!envLoading && filteredEnvList.length === 0">
                {{ envSearch ? '未找到匹配变量' : '暂无环境变量' }}
              </div>
            </div>
          </template>
        </main>
      </div>
    </template>

    <!-- Add PATH dialog -->
    <el-dialog v-model="pathAddVisible" title="添加路径" width="480px" align-center>
      <el-form label-position="top">
        <el-form-item label="路径">
          <el-input v-model="pathAddVal" placeholder="输入或粘贴路径" size="large" @keyup.enter="pathConfirmAdd" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="pathAddVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="pathConfirmAdd" style="flex: 1">添加</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Add Env Var dialog -->
    <el-dialog v-model="envAddVisible" title="添加环境变量" width="480px" align-center>
      <el-form label-position="top">
        <el-form-item label="变量名" required>
          <el-input v-model="envAddName" placeholder="如：JAVA_HOME" size="large" @keyup.enter="envConfirmAdd" />
        </el-form-item>
        <el-form-item label="变量值">
          <el-input v-model="envAddValue" placeholder="变量值" size="large" @keyup.enter="envConfirmAdd" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="envAddVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="envConfirmAdd" style="flex: 1">添加</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Save as Profile dialog -->
    <el-dialog v-model="profileSaveDialogVisible" title="保存为 Profile" width="420px" align-center>
      <div class="dialog-desc">将当前用户 PATH 的所有条目保存为一个 Profile，之后可快速应用。</div>
      <el-form label-position="top">
        <el-form-item label="Profile 名称" required>
          <el-input v-model="profileSaveName" placeholder="如：开发环境、最小化" size="large" @keyup.enter="handleSaveAsProfile" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="profileSaveDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleSaveAsProfile" style="flex: 1">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Manage Profiles dialog -->
    <el-dialog v-model="profileManageDialogVisible" title="管理 Profile" width="520px" align-center>
      <div v-if="pathProfiles.length === 0" class="sidebar-empty" style="padding: 24px">暂无 Profile</div>
      <div v-else class="profile-manage-list">
        <div v-for="p in pathProfiles" :key="p.name" class="profile-manage-item">
          <div class="profile-manage-info">
            <div class="profile-manage-name">{{ p.name }}</div>
            <div class="profile-manage-count">{{ p.paths.length }} 条路径</div>
          </div>
          <div class="profile-manage-actions">
            <el-button text size="small" @click="handleApplyProfile(p.name)" class="action-btn action-primary">
              <el-icon><Check /></el-icon><span>应用</span>
            </el-button>
            <el-button text size="small" @click="handleUpdateProfilePaths(p.name)" class="action-btn">
              <el-icon><Refresh /></el-icon><span>更新</span>
            </el-button>
            <el-button text size="small" @click="openRenameProfile(p.name)" class="action-btn">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button text size="small" @click="handleDeleteProfile(p.name)" class="action-btn action-danger">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
        <!-- Inline rename -->
        <div v-if="profileRenameTarget" class="profile-rename-row">
          <el-input v-model="profileRenameName" size="small" placeholder="新名称" @keyup.enter="handleRenameProfile" />
          <el-button size="small" @click="handleRenameProfile" type="primary">确认</el-button>
          <el-button size="small" @click="profileRenameTarget = ''">取消</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- ==================== RUNTIME TOOL ==================== -->
    <template v-if="currentTool === 'runtime'">
      <div class="body" v-loading="sdkLoading">
        <main class="main-content" style="flex: 1">
          <!-- SDK Selector -->
          <div class="sdk-selector">
            <div
              v-for="sdk in sdkList"
              :key="sdk.type"
              class="sdk-card"
              :class="{ active: selectedSdk === sdk.type }"
              @click="selectedSdk = sdk.type; loadAvailableVersions()"
            >
              <div class="sdk-icon" :style="{ background: sdk.icon }">{{ sdk.name.charAt(0) }}</div>
              <div class="sdk-info">
                <div class="sdk-name">{{ sdk.name }}</div>
                <div class="sdk-current" v-if="sdk.current">当前: {{ sdk.current }}</div>
                <div class="sdk-count" v-else>未安装</div>
              </div>
            </div>
          </div>

          <!-- Version List -->
          <template v-if="selectedSdk">
            <div class="content-header">
              <h2 class="content-title">已安装版本</h2>
              <div style="display: flex; align-items: center; gap: 8px">
                <span class="sdk-basedir">{{ runtimeBaseDir }}</span>
                <el-button size="small" @click="openConfigDialog" class="path-action-btn">
                  <el-icon><Setting /></el-icon>
                </el-button>
                <el-button size="small" @click="loadAvailableVersions(true)" :loading="availableLoading">
                  <el-icon><Refresh /></el-icon><span>刷新版本</span>
                </el-button>
              </div>
            </div>

            <div class="sdk-version-list" v-if="currentSdkVersions().length > 0">
              <div
                v-for="ver in currentSdkVersions()"
                :key="ver.version"
                class="sdk-version-item"
                :class="{ active: ver.active }"
              >
                <div class="sdk-version-badge" :class="{ active: ver.active }">
                  <el-icon v-if="ver.active" size="14"><Check /></el-icon>
                  <span v-else>{{ ver.version.charAt(0) }}</span>
                </div>
                <div class="sdk-version-info">
                  <div class="sdk-version-name">{{ ver.version }}</div>
                  <el-tag v-if="ver.active" type="success" size="small" effect="plain">当前</el-tag>
                </div>
                <div class="sdk-version-actions">
                  <el-button
                    text size="small"
                    @click="handleSwitch(ver.version)"
                    :disabled="ver.active"
                    class="action-btn"
                  >
                    <el-icon><SwitchButton /></el-icon><span>使用</span>
                  </el-button>
                  <el-button
                    text size="small"
                    @click="handleUninstall(ver.version)"
                    :disabled="ver.active"
                    class="action-btn action-danger"
                  >
                    <el-icon><Delete /></el-icon><span>卸载</span>
                  </el-button>
                </div>
              </div>
            </div>

            <div class="empty-state" v-else-if="!sdkLoading">
              <div class="empty-icon"><el-icon size="48"><Box /></el-icon></div>
              <h3 class="empty-title">暂无已安装版本</h3>
              <p class="empty-desc">从下方可下载版本中选择安装</p>
            </div>

            <!-- Available Versions -->
            <div class="content-header" style="margin-top: 24px">
              <h2 class="content-title">可下载版本</h2>
              <div style="display: flex; align-items: center; gap: 8px">
                <el-input
                  v-model="versionSearch"
                  placeholder="搜索版本..."
                  clearable
                  size="small"
                  style="width: 180px"
                  :prefix-icon="''"
                />
                <span class="content-count">{{ filteredAvailableVersions.length }} 个</span>
              </div>
            </div>

            <div v-if="availableLoading" style="text-align: center; padding: 32px">
              <el-icon size="24" class="is-loading"><Loading /></el-icon>
              <div style="margin-top: 8px; color: var(--text-muted); font-size: 13px">正在获取版本列表...</div>
            </div>

            <div v-else-if="availableError" style="text-align: center; padding: 24px">
              <el-icon size="32" color="var(--danger)"><WarningFilled /></el-icon>
              <div style="margin-top: 8px; color: var(--text-muted); font-size: 13px">{{ availableError }}</div>
              <el-button size="small" @click="loadAvailableVersions(true)" style="margin-top: 8px">重试</el-button>
            </div>

            <div class="sdk-version-list" v-else-if="filteredAvailableVersions.length > 0">
              <div
                v-for="ver in filteredAvailableVersions"
                :key="'av-' + ver"
                class="sdk-version-item"
                :class="{ 'sdk-version-installed': installedVersionSet().has(ver) }"
              >
                <div class="sdk-version-badge" :class="{ active: installedVersionSet().has(ver) }">
                  <el-icon v-if="installedVersionSet().has(ver)" size="14"><Check /></el-icon>
                  <span v-else>{{ ver.charAt(0) === 'v' ? ver.charAt(1) : ver.charAt(0) }}</span>
                </div>
                <div class="sdk-version-info">
                  <div class="sdk-version-name">{{ ver }}</div>
                  <el-tag v-if="installedVersionSet().has(ver)" type="success" size="small" effect="plain">已安装</el-tag>
                </div>
                <div class="sdk-version-actions">
                  <el-button
                    v-if="!installedVersionSet().has(ver)"
                    type="primary"
                    text
                    size="small"
                    @click="handleInstall(ver)"
                    :loading="installLoading"
                    class="action-btn action-primary"
                  >
                    <el-icon><Download /></el-icon><span>安装</span>
                  </el-button>
                  <el-tag v-else type="info" size="small" effect="plain" style="font-size: 12px">—</el-tag>
                </div>
              </div>
            </div>

            <div v-else style="text-align: center; padding: 24px; color: var(--text-muted); font-size: 13px">
              未找到可下载版本
            </div>
          </template>
        </main>
      </div>
    </template>

    <!-- Runtime Config Dialog -->
    <el-dialog v-model="configDialogVisible" title="环境管理配置" width="420px" align-center>
      <el-form label-position="top">
        <el-form-item label="安装目录">
          <div class="config-dir-row">
            <el-input v-model="runtimeBaseDir" placeholder="SDK 存储目录" size="large" />
            <el-button size="large" @click="handleBrowseRuntimeDir">
              <el-icon><FolderOpened /></el-icon>
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      <div class="dialog-desc" style="margin-top: -8px">
        更改目录后，需要手动将已有版本迁移到新目录。当前目录下的已安装版本不受影响。
      </div>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="configDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleSaveConfig" style="flex: 1">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- ==================== Dialogs ==================== -->

    <!-- Install Progress -->
    <el-dialog
      v-model="installProgress.visible"
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      width="400px"
      align-center
      class="install-progress-dialog"
    >
      <div class="install-progress-content">
        <div class="install-progress-phase">{{ phaseLabel(installProgress.phase) }}</div>
        <el-progress
          :percentage="installProgress.percent"
          :stroke-width="8"
          :color="installProgress.phase === 'download' ? '#409EFF' : installProgress.phase === 'extract' ? '#67C23A' : '#6366f1'"
          style="margin: 16px 0"
        />
        <div class="install-progress-msg">{{ installProgress.message }}</div>
      </div>
    </el-dialog>

    <!-- Setup Master Password -->
    <el-dialog v-model="setupDialogVisible" title="设置主密码" width="420px" :close-on-click-modal="false" align-center>
      <div class="dialog-desc">请设置一个主密码来保护你的密码库。请妥善保管此密码，丢失后将无法恢复数据。</div>
      <el-form label-position="top">
        <el-form-item label="主密码">
          <el-input v-model="masterPassword" type="password" show-password placeholder="至少6位" size="large" />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input v-model="confirmPassword" type="password" show-password placeholder="再次输入主密码" size="large" @keyup.enter="handleSetup" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" size="large" @click="handleSetup" style="width: 100%">确认设置</el-button>
      </template>
    </el-dialog>

    <!-- Unlock -->
    <el-dialog v-model="unlockDialogVisible" title="解锁密码库" width="420px" align-center>
      <el-form label-position="top" @submit.prevent="handleUnlock">
        <el-form-item label="主密码">
          <el-input v-model="masterPassword" type="password" show-password placeholder="输入主密码" size="large" @keyup.enter="handleUnlock" autofocus />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="unlockDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleUnlock" style="flex: 1">解锁</el-button>
        </div>
      </template>
      <div v-if="helloAvailable && helloEnabled" class="hello-divider">
        <el-divider>或</el-divider>
        <el-button size="large" style="width: 100%" @click="handleHelloUnlock">
          <el-icon><Key /></el-icon><span>使用 Windows Hello 解锁</span>
        </el-button>
        <el-button text size="large" style="width: 100%; margin-top: 8px" @click="openHelloSettings">
          打开 Windows Hello 设置
        </el-button>
      </div>
    </el-dialog>

    <!-- Category Dialog -->
    <el-dialog v-model="categoryDialogVisible" :title="categoryForm.id ? '编辑分类' : '新增分类'" width="400px" align-center>
      <el-form label-position="top">
        <el-form-item label="分类名称" required>
          <el-input v-model="categoryForm.name" placeholder="输入分类名称" size="large" />
        </el-form-item>
        <el-form-item label="父级分类">
          <el-tree-select
            v-model="categoryForm.parentId"
            :data="categoryTree"
            :props="{ label: 'label', children: 'children', value: 'id' }"
            placeholder="无（顶级分类）"
            clearable
            check-strictly
            :render-after-expand="false"
            size="large"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="categoryDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleSaveCategory" style="flex: 1">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Tag Dialog -->
    <el-dialog v-model="tagDialogVisible" :title="tagForm.id ? '编辑标签' : '新增标签'" width="400px" align-center>
      <el-form label-position="top">
        <el-form-item label="标签名称" required>
          <el-input v-model="tagForm.name" placeholder="输入标签名称" size="large" />
        </el-form-item>
        <el-form-item label="颜色">
          <div class="color-picker">
            <div
              v-for="color in tagColorPresets"
              :key="color"
              class="color-dot"
              :class="{ active: tagForm.color === color }"
              :style="{ background: color }"
              @click="tagForm.color = color"
            />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="tagDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleSaveTag" style="flex: 1">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Entry Edit Dialog -->
    <el-dialog v-model="editDialogVisible" :title="isEdit ? '编辑密码' : '新增密码'" width="500px" align-center>
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="editForm.title" placeholder="如: GitHub" size="large" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="editForm.username" placeholder="用户名或邮箱" size="large" />
        </el-form-item>
        <el-form-item label="密码" :required="!isEdit">
          <el-input v-model="editForm.password" type="password" show-password :placeholder="isEdit ? '留空则不修改' : '密码'" size="large" />
        </el-form-item>
        <el-form-item label="网址">
          <el-input v-model="editForm.url" placeholder="https://..." size="large" />
        </el-form-item>
        <el-form-item label="分类">
          <el-tree-select
            v-model="editForm.categoryId"
            :data="categoryTree"
            :props="{ label: 'label', children: 'children', value: 'id' }"
            placeholder="选择分类"
            clearable
            check-strictly
            :render-after-expand="false"
            size="large"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-select
            v-model="editForm.tagIds"
            multiple
            placeholder="选择标签"
            size="large"
            style="width: 100%"
          >
            <el-option
              v-for="tag in tags"
              :key="tag.id"
              :label="tag.name"
              :value="tag.id"
            >
              <div style="display: flex; align-items: center; gap: 8px">
                <span class="tag-dot" :style="{ background: tag.color, width: '10px', height: '10px', borderRadius: '50%', display: 'inline-block' }"></span>
                {{ tag.name }}
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editForm.notes" type="textarea" rows="2" size="large" placeholder="可选备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; gap: 12px; width: 100%">
          <el-button size="large" @click="editDialogVisible = false" style="flex: 1">取消</el-button>
          <el-button type="primary" size="large" @click="handleSave" style="flex: 1">保存</el-button>
        </div>
      </template>
    </el-dialog>
    </div><!-- close app-main -->
  </div>
</template>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  background: var(--bg);
}

/* ===== Nav Rail ===== */
.nav-rail {
  width: 56px;
  flex-shrink: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 0;
  z-index: 10;
}

.nav-rail-logo {
  margin-bottom: 16px;
}

.nav-rail-logo .logo-icon {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), var(--primary-light));
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.nav-rail-items {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-rail-item {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
  position: relative;
}

.nav-rail-item:hover {
  background: var(--bg);
  color: var(--text);
}

.nav-rail-item.active {
  background: var(--primary-bg);
  color: var(--primary);
}

.nav-rail-item.active::before {
  content: '';
  position: absolute;
  left: -8px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  border-radius: 0 3px 3px 0;
  background: var(--primary);
}

/* ===== App Main ===== */
.app-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* ===== Header ===== */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  height: 56px;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-input {
  width: 220px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 8px !important;
  background: var(--bg) !important;
  box-shadow: none !important;
  border: 1px solid var(--border) !important;
}

.search-input :deep(.el-input__wrapper:hover),
.search-input :deep(.el-input__wrapper.is-focus) {
  border-color: var(--primary) !important;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1) !important;
}

.btn-add { border-radius: 8px !important; font-weight: 500; }
.btn-lock { border-radius: 8px !important; font-weight: 500; color: var(--text-secondary); }
.btn-hello { font-size: 18px !important; }
.btn-hello.is-active { color: var(--primary) !important; }
.hello-divider { margin-top: -8px; }
.hello-divider .el-divider__text { color: var(--text-muted); font-size: 12px; }

/* ===== Body Layout ===== */
.body {
  flex: 1;
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
  border-radius: 0;
  margin: 0;
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

/* Category Tree */
.category-tree {
  --el-tree-node-content-height: 32px;
  background: transparent !important;
  font-size: 13px;
}

.category-tree :deep(.el-tree-node__content) {
  padding-right: 8px !important;
}

.category-tree :deep(.el-tree-node__content:hover) {
  background: var(--bg);
}

.category-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: var(--primary-bg);
}

.tree-node {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
  gap: 6px;
  padding-right: 4px;
}

.tree-node-icon {
  color: var(--text-muted);
  flex-shrink: 0;
}

.tree-node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-node-actions {
  display: none;
  gap: 2px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.tree-node:hover .tree-node-actions {
  display: flex;
}

.tree-node-actions .el-icon:hover {
  color: var(--primary);
}

/* Tag List */
.tag-list {
  padding: 0 8px;
}

.tag-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  transition: all 0.15s;
}

.tag-item:hover {
  background: var(--bg);
  color: var(--text);
}

.tag-item.active {
  background: var(--primary-bg);
  color: var(--primary);
}

.tag-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.tag-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-edit,
.tag-delete {
  display: none;
  color: var(--text-muted);
}

.tag-item:hover .tag-edit,
.tag-item:hover .tag-delete {
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
  gap: 10px;
  margin-bottom: 16px;
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

/* ===== Entries Grid ===== */
.entries-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}

/* ===== Entry Card ===== */
.entry-card {
  background: var(--bg-card);
  border-radius: var(--radius);
  border: 1px solid var(--border);
  padding: 16px;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.entry-card:hover {
  box-shadow: var(--shadow-md);
  border-color: rgba(99, 102, 241, 0.2);
  transform: translateY(-1px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.entry-avatar {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 15px;
  flex-shrink: 0;
}

.entry-info {
  flex: 1;
  min-width: 0;
}

.entry-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.entry-username {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.card-meta {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-left: 50px;
}

.meta-tag {
  border: none !important;
  font-size: 11px !important;
  display: flex;
  align-items: center;
  gap: 3px;
}

.card-body {
  padding-left: 50px;
}

.entry-url {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--primary);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.entry-url:hover { text-decoration: underline; }

.entry-notes {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
  margin-top: auto;
}

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

/* ===== PATH Viewer ===== */
.path-tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 12px;
}

.path-tab {
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.path-tab:hover {
  color: var(--text);
}

.path-tab.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
}

.path-tab-count {
  font-size: 11px;
  background: var(--bg);
  color: var(--text-muted);
  padding: 1px 6px;
  border-radius: 10px;
}

.path-tab.active .path-tab-count {
  background: var(--primary-bg);
  color: var(--primary);
}

.path-list-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 6px;
}

.path-stats {
  display: flex;
  gap: 6px;
  align-items: center;
}

.path-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.path-entry {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  transition: all 0.15s;
}

.path-entry:hover {
  border-color: rgba(99, 102, 241, 0.2);
  box-shadow: var(--shadow-sm);
}

.path-entry.path-invalid {
  background: #fef2f2;
  border-color: #fecaca;
}

.path-entry-index {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  flex-shrink: 0;
}

.path-entry-body {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.path-entry-value {
  font-size: 13px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.path-entry-meta {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.path-entry-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.path-entry-actions .el-button {
  width: 28px !important;
  height: 28px !important;
  padding: 0 !important;
  margin: 0 !important;
  border-radius: 6px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

.path-entry:hover .path-entry-actions {
  opacity: 1;
}

.path-entry-actions-always {
  opacity: 1 !important;
}

.path-edit-input {
  flex: 1;
}

.path-edit-input :deep(.el-input-group__append) {
  display: flex;
  gap: 0;
  padding: 0;
}

.path-action-btn {
  border-radius: 6px !important;
}

.path-tab .path-action-btn {
  margin-left: auto;
}

/* ===== PATH Profile ===== */
.profile-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 8px;
}

.profile-bar-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.profile-manage-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.profile-manage-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.profile-manage-info {
  flex: 1;
  min-width: 0;
}

.profile-manage-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}

.profile-manage-count {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
}

.profile-manage-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.profile-rename-row {
  display: flex;
  gap: 8px;
  padding: 8px 0;
  align-items: center;
}

.profile-rename-row .el-input {
  flex: 1;
}

/* ===== Environment Variables ===== */
.env-hint-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  background: var(--primary-bg);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: var(--radius-sm);
  margin-bottom: 12px;
}

.env-hint-text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  flex: 1;
  min-width: 0;
}

.env-hint-code {
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 12px;
  background: var(--bg);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}

.env-hint-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.env-tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 12px;
}

.env-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.env-entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  transition: all 0.15s;
}

.env-entry:hover {
  border-color: rgba(99, 102, 241, 0.2);
  box-shadow: var(--shadow-sm);
}

.env-entry.env-entry-path {
  border-left: 3px solid var(--primary);
  background: var(--primary-bg);
}

.env-entry-name {
  width: 180px;
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.env-entry-path .env-entry-name {
  color: var(--primary);
}

.env-entry-body {
  flex: 1;
  min-width: 0;
}

.env-entry-value {
  font-size: 13px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.env-entry-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.env-entry:hover .env-entry-actions {
  opacity: 1;
}

.env-entry-path .env-entry-actions {
  opacity: 1;
}

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

/* ===== Locked State ===== */
.locked-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, var(--bg) 0%, var(--primary-bg) 100%);
}

.locked-card { text-align: center; padding: 48px; }

.locked-icon {
  width: 100px;
  height: 100px;
  border-radius: 28px;
  background: linear-gradient(135deg, var(--primary), var(--primary-light));
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  margin: 0 auto 24px;
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.3);
}

.locked-title { font-size: 22px; font-weight: 700; color: var(--text); margin-bottom: 8px; }
.locked-desc { font-size: 14px; color: var(--text-muted); margin-bottom: 28px; }

/* ===== Dialog Styles ===== */
.dialog-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 20px;
  padding: 12px;
  background: var(--primary-bg);
  border-radius: 8px;
  border-left: 3px solid var(--primary);
}

.color-picker {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.color-dot {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.15s;
  border: 2px solid transparent;
}

.color-dot:hover {
  transform: scale(1.15);
}

.color-dot.active {
  border-color: var(--text);
  transform: scale(1.15);
}

/* ===== Runtime Manager ===== */
.sdk-selector {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.sdk-basedir {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg);
  padding: 3px 8px;
  border-radius: 4px;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.config-dir-row {
  display: flex;
  gap: 8px;
}

.config-dir-row .el-input {
  flex: 1;
}

.sdk-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--bg-card);
  border: 2px solid var(--border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
}

.sdk-card:hover {
  border-color: rgba(99, 102, 241, 0.3);
  box-shadow: var(--shadow-sm);
}

.sdk-card.active {
  border-color: var(--primary);
  background: var(--primary-bg);
}

.sdk-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  font-size: 16px;
  flex-shrink: 0;
}

.sdk-info {
  flex: 1;
  min-width: 0;
}

.sdk-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.sdk-current {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.sdk-count {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
}

.sdk-version-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sdk-version-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  transition: all 0.15s;
}

.sdk-version-item:hover {
  border-color: rgba(99, 102, 241, 0.2);
  box-shadow: var(--shadow-sm);
}

.sdk-version-item.active {
  border-color: rgba(16, 185, 129, 0.4);
  background: #f0fdf4;
}

.sdk-version-installed {
  opacity: 0.6;
}

.install-progress-content {
  text-align: center;
  padding: 12px 0;
}
.install-progress-phase {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.install-progress-msg {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.sdk-version-badge {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  flex-shrink: 0;
}

.sdk-version-badge.active {
  background: var(--success);
  color: #fff;
}

.sdk-version-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sdk-version-name {
  font-size: 13px;
  font-weight: 500;
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  color: var(--text);
}

.sdk-version-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.sdk-version-actions .el-button {
  width: auto !important;
  height: 28px !important;
  padding: 0 8px !important;
}

/* ===== Transitions ===== */
.card-enter-active { transition: all 0.3s ease; }
.card-leave-active { transition: all 0.2s ease; }
.card-enter-from { opacity: 0; transform: translateY(12px); }
.card-leave-to { opacity: 0; transform: scale(0.95); }
</style>
