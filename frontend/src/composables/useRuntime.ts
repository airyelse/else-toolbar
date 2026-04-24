import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ListSDKs,
  InstallSDK,
  UninstallSDK,
  SwitchSDK,
  GetRuntimeConfig,
  SetRuntimeConfig,
  FetchAvailableVersions,
  SelectDirectory as SelectDirDialog,
  OpenInExplorer,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

// ==================== Types ====================
export interface SDKVersionItem {
  version: string
  path: string
  active: boolean
}

export interface SDKInfoItem {
  type: string
  name: string
  icon: string
  installed: SDKVersionItem[]
  current: string
}

// ==================== State ====================
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

const runtimeBaseDir = ref('')
const configDialogVisible = ref(false)

// ==================== Computed ====================
const filteredAvailableVersions = computed(() => {
  if (!versionSearch.value) return availableVersions.value
  const q = versionSearch.value.toLowerCase()
  return availableVersions.value.filter(v => v.toLowerCase().includes(q))
})

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

// ==================== Functions ====================
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
  } catch (e: any) {
    // User cancelled the confirm dialog
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e?.message || '卸载失败')
    }
  }
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

// ==================== Event Listener ====================
// Register event listener for install progress
let unlistenProgress: (() => void) | null = null

function setupProgressListener() {
  unlistenProgress = EventsOn('sdk:progress', (event: any) => {
    installProgress.value = { visible: true, phase: event.phase, message: event.message, percent: event.percent }
  })
}

function cleanupProgressListener() {
  if (typeof unlistenProgress === 'function') unlistenProgress()
}

// ==================== Exports ====================
export {
  // State
  sdkList,
  sdkLoading,
  selectedSdk,
  installLoading,
  installProgress,
  availableVersions,
  availableLoading,
  availableError,
  versionSearch,
  sdkRequestSeq,
  runtimeBaseDir,
  configDialogVisible,
  // Computed
  filteredAvailableVersions,
  // Functions
  currentSdkVersions,
  installedVersionSet,
  phaseLabel,
  loadSDKs,
  loadAvailableVersions,
  handleInstall,
  handleUninstall,
  handleSwitch,
  loadRuntimeConfig,
  openConfigDialog,
  handleSaveConfig,
  handleBrowseRuntimeDir,
  copyPath,
  openPathDir,
  setupProgressListener,
  cleanupProgressListener,
}
