import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  GetOpenCodeConfig,
  SaveOpenCodeConfig as SaveOpenCodeConfigBackend,
  GetOpenCodeConfigPath,
  FetchAvailableModels,
  ForceRefreshModels,
  DiffPresets,
  SyncPresetsToConfig,
  ImportPresetsFromConfig,
  ReadAppendPrompt,
  WriteAppendPrompt as WriteAppendPromptBackend,
  GetAppendPromptPath,
  ReadAllAppendPrompts,
  RestoreAppendPrompts,
  ImportAppendPromptsFromFiles,
  DiffAppendPrompts,
  GetOpenCodeMCPs,
  GetOpenCodeSkills,
  FetchMCPSkills,
  ReadMainConfig,
  SaveMainConfig as SaveMainConfigBackend,
  GetMainConfigPath,
  OpenOpenCodeConfigDir,
  OpenInExplorer,
} from '../../wailsjs/go/main/App'

// ==================== Types ====================
type AgentConfigItem = {
  model: string
  variant: string
  skills: string[]
  mcps: string[]
}

type PresetItem = {
  [agentName: string]: AgentConfigItem | undefined
  orchestrator?: AgentConfigItem
  oracle?: AgentConfigItem
  librarian?: AgentConfigItem
  explorer?: AgentConfigItem
  designer?: AgentConfigItem
  fixer?: AgentConfigItem
}

type OpenCodeConfig = {
  preset: string
  presets: Record<string, PresetItem>
}

type MainProviderInfo = {
  name: string
}

type MainAgentDisable = {
  name: string
  disable: boolean
}

// ==================== OpenCode Config State ====================
const ocConfig = ref<OpenCodeConfig | null>(null)
const ocConfigPath = ref('')
const ocLoading = ref(false)
const ocDirty = ref(false)
const ocAvailableModels = ref<string[]>([])
const ocModelsLoading = ref(false)

// Agent edit dialog
const ocAgentDialogVisible = ref(false)
const ocEditAgentName = ref('')
const ocEditPresetName = ref('')
const ocEditForm = ref<AgentConfigItem>({ model: '', variant: '', skills: [], mcps: [] })

// New preset dialog
const ocNewPresetDialogVisible = ref(false)
const ocNewPresetName = ref('')

// Rename preset dialog
const ocRenamePresetDialogVisible = ref(false)
const ocRenameOldName = ref('')
const ocRenameNewName = ref('')

// Preset diff
const ocPresetDiff = ref<{ store_active: string; file_active: string; differences: string[] } | null>(null)
const ocPresetDiffVisible = ref(false)

// Append prompt state
const ocAppendPrompts = ref<Record<string, string>>({})
const ocAppendPromptsLoading = ref(false)
const ocPromptDialogVisible = ref(false)
const ocPromptAgentName = ref('')
const ocPromptContent = ref('')
const ocPromptFilePath = ref('')
const ocPromptDirty = ref(false)
const ocPromptDiffs = ref<Array<{ agent: string; store: string; file: string }>>([])
const ocPromptDiffVisible = ref(false)

// Active tab in OpenCode page
const ocActiveTab = ref('opencode')
const ocOpenCodeSubTab = ref('overview')
const slimSubTab = ref('overview')

// Main OpenCode config state
const mainConfig = ref<any>(null)
const mainConfigPath = ref('')
const mainConfigLoading = ref(false)
const mainConfigDirty = ref(false)
const mainProviders = ref<MainProviderInfo[]>([])
const mainAgentDisables = ref<MainAgentDisable[]>([])

// MCP & Skill discovery
const ocMCPs = ref<Array<{ name: string; type: string; command: string; url: string; source: string }>>([])
const ocSkills = ref<Array<{ name: string; description: string; source: string }>>([])
const ocMCPSkillsRefreshing = ref(false)

// Agent metadata (static)
const ocAgentNames = ['orchestrator', 'oracle', 'librarian', 'explorer', 'designer', 'fixer']
const ocAgentLabels: Record<string, string> = {
  orchestrator: 'Orchestrator (主编排)',
  oracle: 'Oracle (架构师)',
  librarian: 'Librarian (文档研究员)',
  explorer: 'Explorer (代码搜索)',
  designer: 'Designer (UI 设计)',
  fixer: 'Fixer (快速实现)',
}
const ocAgentColors: Record<string, string> = {
  orchestrator: '#6366f1',
  oracle: '#f59e0b',
  librarian: '#10b981',
  explorer: '#06b6d4',
  designer: '#ec4899',
  fixer: '#64748b',
}

// ==================== Helper Functions ====================
function ocModelToString(model: any): string {
  if (!model) return ''
  if (typeof model === 'string') return model
  if (Array.isArray(model)) return model.join(', ')
  return String(model)
}

function ocStringToModel(s: string): any {
  if (!s) return ''
  const trimmed = s.trim()
  if (trimmed.startsWith('[')) {
    try { return JSON.parse(trimmed) } catch { return trimmed }
  }
  return trimmed
}

// ==================== Config Loading Functions ====================
async function loadOpenCodeConfig() {
  ocLoading.value = true
  try {
    const store = await GetOpenCodeConfig()
    if (!store) {
      ocConfig.value = null
    } else {
      const path = await GetOpenCodeConfigPath()
      ocConfigPath.value = path || ''

      // Normalize presets: convert AgentConfig objects to plain AgentConfigItem
      const presets: Record<string, PresetItem> = {}
      for (const [name, preset] of Object.entries(store.presets || {})) {
        const p: PresetItem = {}
        if (preset) {
          for (const agentName of ocAgentNames) {
            const agent = (preset as any)[agentName]
            if (agent) {
              p[agentName] = {
                model: ocModelToString(agent.model),
                variant: agent.variant || '',
                skills: agent.skills || [],
                mcps: agent.mcps || [],
              }
            }
          }
        }
        presets[name] = p
      }

      ocConfig.value = { preset: store.active_preset, presets }
      ocDirty.value = false
    }
  } catch (e: any) {
    ElMessage.error(e.message || '加载 OpenCode 配置失败')
  }
  ocLoading.value = false

  // 加载可用模型列表、附加提示词和预设差异
  loadAvailableModels()
  loadAppendPrompts()
  checkPresetDiff()
  loadMCPSkills()
  loadMainConfig()
}

async function loadMCPSkills() {
  try {
    const result = await FetchMCPSkills()
    ocMCPs.value = result?.mcps || []
    ocSkills.value = result?.skills || []
  } catch { /* ignore */ }
}

async function loadAvailableModels() {
  ocModelsLoading.value = true
  try {
    ocAvailableModels.value = await FetchAvailableModels() || []
  } catch { /* ignore - models are optional */ }
  ocModelsLoading.value = false
}

async function handleRefreshModels() {
  ocModelsLoading.value = true
  try {
    ocAvailableModels.value = await ForceRefreshModels() || []
    ElMessage.success('模型列表已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '更新模型列表失败')
  }
  ocModelsLoading.value = false
}

async function handleRefreshMCPSkills() {
  ocMCPSkillsRefreshing.value = true
  try {
    const [mcps, skills] = await Promise.all([GetOpenCodeMCPs(), GetOpenCodeSkills()])
    ocMCPs.value = mcps || []
    ocSkills.value = skills || []
    ElMessage.success('MCP & Skills 已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
  }
  ocMCPSkillsRefreshing.value = false
}

async function saveOpenCodeConfig() {
  if (!ocConfig.value) return
  try {
    // Convert back to Go-compatible structure
    const presets: Record<string, any> = {}
    for (const [name, preset] of Object.entries(ocConfig.value.presets)) {
      const p: any = {}
      for (const agentName of ocAgentNames) {
        const agent = preset[agentName]
        if (agent) {
          p[agentName] = {
            model: ocStringToModel(agent.model),
            variant: agent.variant || undefined,
            skills: agent.skills.length > 0 ? agent.skills : undefined,
            mcps: agent.mcps.length > 0 ? agent.mcps : undefined,
          }
        }
      }
      presets[name] = p
    }

    await SaveOpenCodeConfigBackend({
      active_preset: ocConfig.value.preset,
      presets,
    } as any)
    ocDirty.value = false
    ElMessage.success('配置已保存')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

// ==================== Preset Functions ====================
function ocSwitchPreset(name: string) {
  if (!ocConfig.value) return
  ocConfig.value.preset = name
  ocDirty.value = true
}

function ocOpenAgentEdit(agentName: string) {
  if (!ocConfig.value) return
  const preset = ocConfig.value.presets[ocConfig.value.preset]
  ocEditAgentName.value = agentName
  ocEditPresetName.value = ocConfig.value.preset
  const existing = preset?.[agentName]
  if (existing) {
    ocEditForm.value = { ...existing }
  } else {
    ocEditForm.value = { model: '', variant: '', skills: [], mcps: [] }
  }
  ocAgentDialogVisible.value = true
}

function ocSaveAgentEdit() {
  if (!ocConfig.value) return
  const preset = ocConfig.value.presets[ocConfig.value.preset]
  if (!preset) return
  preset[ocEditAgentName.value] = {
    model: ocEditForm.value.model,
    variant: ocEditForm.value.variant,
    skills: ocEditForm.value.skills || [],
    mcps: ocEditForm.value.mcps || [],
  }
  ocDirty.value = true
  ocAgentDialogVisible.value = false
}

function ocOpenNewPreset() {
  ocNewPresetName.value = ''
  ocNewPresetDialogVisible.value = true
}

function ocCreatePreset() {
  if (!ocConfig.value || !ocNewPresetName.value.trim()) {
    ElMessage.warning('请输入预设名称')
    return
  }
  const name = ocNewPresetName.value.trim()
  if (ocConfig.value.presets[name]) {
    ElMessage.warning('预设已存在')
    return
  }
  ocConfig.value.presets[name] = {}
  ocNewPresetDialogVisible.value = false
  ocDirty.value = true
  ElMessage.success('预设已创建')
}

async function ocDeletePreset(name: string) {
  if (!ocConfig.value) return
  if (name === ocConfig.value.preset) {
    ElMessage.warning('不能删除当前活跃预设')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除预设「${name}」？`, '删除预设', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
    delete ocConfig.value.presets[name]
    ocDirty.value = true
    ElMessage.success('已删除')
  } catch { /* cancel */ }
}

function ocOpenRenamePreset(name: string) {
  ocRenameOldName.value = name
  ocRenameNewName.value = name
  ocRenamePresetDialogVisible.value = true
}

async function ocRenamePreset() {
  if (!ocConfig.value) return
  const newName = ocRenameNewName.value.trim()
  if (!newName) {
    ElMessage.warning('请输入预设名称')
    return
  }
  if (newName === ocRenameOldName.value) {
    ocRenamePresetDialogVisible.value = false
    return
  }
  if (ocConfig.value.presets[newName]) {
    ElMessage.warning('预设名称已存在')
    return
  }
  // In-memory rename
  const presetData = ocConfig.value.presets[ocRenameOldName.value]
  delete ocConfig.value.presets[ocRenameOldName.value]
  if (presetData) ocConfig.value.presets[newName] = presetData
  if (ocConfig.value.preset === ocRenameOldName.value) {
    ocConfig.value.preset = newName
  }
  ocDirty.value = true
  ocRenamePresetDialogVisible.value = false
  ElMessage.success('预设已重命名（保存后生效）')
}

function ocGetActivePreset(): PresetItem | null {
  if (!ocConfig.value) return null
  return ocConfig.value.presets[ocConfig.value.preset] || null
}

// ==================== Preset Diff Functions ====================
async function checkPresetDiff() {
  try {
    const diff = await DiffPresets()
    if (diff) {
      ocPresetDiff.value = diff
      ocPresetDiffVisible.value = true
    }
  } catch { /* ignore */ }
}

async function ocSyncPresetsToConfig() {
  try {
    await SyncPresetsToConfig()
    ElMessage.success('预设已同步到配置文件')
    ocPresetDiffVisible.value = false
    ocPresetDiff.value = null
  } catch (e: any) {
    ElMessage.error(e.message || '同步失败')
  }
}

async function ocImportPresetsFromConfig() {
  try {
    await ElMessageBox.confirm(
      '将用配置文件的预设覆盖持久存储，当前存储内容将被替换。是否继续？',
      '从文件导入',
      { confirmButtonText: '导入', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await ImportPresetsFromConfig()
    ElMessage.success('预设已从配置文件导入')
    ocPresetDiffVisible.value = false
    ocPresetDiff.value = null
    await loadOpenCodeConfig()
  } catch (e) {
    ElMessage.error(String(e))
  }
}

function ocDismissPresetDiff() {
  ocPresetDiffVisible.value = false
  ocPresetDiff.value = null
}

// ==================== Append Prompt Functions ====================
async function loadAppendPrompts() {
  ocAppendPromptsLoading.value = true
  try {
    ocAppendPrompts.value = await ReadAllAppendPrompts() || {}
    // 检测 .md 文件与持久存储的差异
    const diffs = await DiffAppendPrompts() || []
    ocPromptDiffs.value = diffs
    if (diffs.length > 0) {
      ocPromptDiffVisible.value = true
    }
  } catch {
    ocAppendPrompts.value = {}
  }
  ocAppendPromptsLoading.value = false
}

async function ocOpenPromptEdit(agentName: string) {
  ocPromptAgentName.value = agentName
  ocPromptDirty.value = false

  try {
    ocPromptFilePath.value = await GetAppendPromptPath(agentName) || ''
  } catch {
    ocPromptFilePath.value = ''
  }

  try {
    ocPromptContent.value = await ReadAppendPrompt(agentName) || ''
  } catch {
    ocPromptContent.value = ''
  }

  ocPromptDialogVisible.value = true
}

async function ocSavePrompt() {
  try {
    await WriteAppendPromptBackend(ocPromptAgentName.value, ocPromptContent.value)
    ocPromptDirty.value = false
    ocPromptDialogVisible.value = false
    ElMessage.success('附加提示词已保存')
    await loadAppendPrompts()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

// 同步：以持久存储覆盖 .md 文件
async function ocSyncPromptsToFiles() {
  try {
    const count = await RestoreAppendPrompts()
    ElMessage.success(`已同步 ${count} 个文件`)
    ocPromptDiffVisible.value = false
    ocPromptDiffs.value = []
    await loadAppendPrompts()
  } catch (e: any) {
    ElMessage.error(e.message || '同步失败')
  }
}

// 导入：以 .md 文件覆盖持久存储
async function ocImportPromptsFromFiles() {
  try {
    await ElMessageBox.confirm(
      '将用 .md 文件的内容覆盖持久存储，当前存储内容将被替换。是否继续？',
      '从文件导入',
      { confirmButtonText: '导入', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    const count = await ImportAppendPromptsFromFiles()
    ElMessage.success(`已导入 ${count} 条提示词`)
    ocPromptDiffVisible.value = false
    ocPromptDiffs.value = []
    await loadAppendPrompts()
  } catch (e) {
    ElMessage.error(String(e))
  }
}

// 忽略差异
function ocDismissDiff() {
  ocPromptDiffVisible.value = false
  ocPromptDiffs.value = []
}

// 手动检查差异
async function ocCheckDiff() {
  const diffs = await DiffAppendPrompts() || []
  if (diffs.length === 0) {
    ElMessage.success('存储与文件一致，无差异')
    return
  }
  ocPromptDiffs.value = diffs
  ocPromptDiffVisible.value = true
}

// ==================== Main OpenCode Config ====================
async function loadMainConfig() {
  mainConfigLoading.value = true
  try {
    mainConfig.value = await ReadMainConfig() || null
    mainConfigPath.value = await GetMainConfigPath() || ''
    // Normalize providers
    const provs: MainProviderInfo[] = []
    if (mainConfig.value?.provider) {
      for (const [name] of Object.entries(mainConfig.value.provider)) {
        provs.push({ name })
      }
    }
    mainProviders.value = provs
    // Normalize agent disables
    const agents: MainAgentDisable[] = []
    if (mainConfig.value?.agent) {
      for (const [name, cfg] of Object.entries(mainConfig.value.agent)) {
        agents.push({ name, disable: !!(cfg as any)?.disable })
      }
    }
    mainAgentDisables.value = agents
    mainConfigDirty.value = false
  } catch {
    mainConfig.value = null
  }
  mainConfigLoading.value = false
}

async function saveMainConfig() {
  if (!mainConfig.value) return
  // Don't rebuild provider — preserve original config fields (OAuth, etc.)
  // Rebuild agent disables
  const agentObj: any = {}
  for (const a of mainAgentDisables.value) {
    if (a.disable) agentObj[a.name] = { disable: true }
  }
  mainConfig.value.agent = Object.keys(agentObj).length > 0 ? agentObj : undefined

  try {
    await SaveMainConfigBackend(mainConfig.value)
    mainConfigDirty.value = false
    ElMessage.success('配置已保存')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

function onMainModelChange() { mainConfigDirty.value = true }
function onMainSmallModelChange() { mainConfigDirty.value = true }

// ==================== Exports ====================
export {
  // State
  ocConfig,
  ocConfigPath,
  ocLoading,
  ocDirty,
  ocAvailableModels,
  ocModelsLoading,
  ocAgentDialogVisible,
  ocEditAgentName,
  ocEditPresetName,
  ocEditForm,
  ocNewPresetDialogVisible,
  ocNewPresetName,
  ocRenamePresetDialogVisible,
  ocRenameOldName,
  ocRenameNewName,
  ocPresetDiff,
  ocPresetDiffVisible,
  ocAppendPrompts,
  ocAppendPromptsLoading,
  ocPromptDialogVisible,
  ocPromptAgentName,
  ocPromptContent,
  ocPromptFilePath,
  ocPromptDirty,
  ocPromptDiffs,
  ocPromptDiffVisible,
  ocActiveTab,
  ocOpenCodeSubTab,
  slimSubTab,
  mainConfig,
  mainConfigPath,
  mainConfigLoading,
  mainConfigDirty,
  mainProviders,
  mainAgentDisables,
  ocMCPs,
  ocSkills,
  ocMCPSkillsRefreshing,
  ocAgentNames,
  ocAgentLabels,
  ocAgentColors,

  // Functions
  loadOpenCodeConfig,
  loadMCPSkills,
  loadAvailableModels,
  handleRefreshModels,
  handleRefreshMCPSkills,
  saveOpenCodeConfig,
  ocSwitchPreset,
  ocOpenAgentEdit,
  ocSaveAgentEdit,
  ocOpenNewPreset,
  ocCreatePreset,
  ocDeletePreset,
  ocOpenRenamePreset,
  ocRenamePreset,
  ocGetActivePreset,
  checkPresetDiff,
  ocSyncPresetsToConfig,
  ocImportPresetsFromConfig,
  ocDismissPresetDiff,
  loadAppendPrompts,
  ocOpenPromptEdit,
  ocSavePrompt,
  ocSyncPromptsToFiles,
  ocImportPromptsFromFiles,
  ocDismissDiff,
  ocCheckDiff,
  loadMainConfig,
  saveMainConfig,
  onMainModelChange,
  onMainSmallModelChange,
  OpenOpenCodeConfigDir,
  OpenInExplorer,
}

// Types
export type { AgentConfigItem, PresetItem, OpenCodeConfig, MainProviderInfo, MainAgentDisable }
