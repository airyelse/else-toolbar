import { ref, computed, nextTick } from 'vue'
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
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

// ==================== Types ====================
export interface Entry {
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

export interface CategoryNode {
  id: number
  label: string
  children?: CategoryNode[]
}

export interface TagItem {
  id: number
  name: string
  color: string
}

// ==================== useVault Composable ====================
export function useVault() {
  // State
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

  const totalEntries = ref(0)

  // ==================== Computed ====================
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
    } catch {
      return
    }
    try {
      await DeleteCategory(node.id)
      if (selectedCategoryId.value === node.id) selectedCategoryId.value = null
      ElMessage.success('删除成功')
      await Promise.all([loadCategories(), loadEntries()])
    } catch (e) {
      ElMessage.error(String(e))
    }
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
    } catch {
      return
    }
    try {
      await DeleteTag(tag.id)
      if (selectedTagId.value === tag.id) selectedTagId.value = null
      ElMessage.success('删除成功')
      await Promise.all([loadTags(), loadEntries()])
    } catch (e) {
      ElMessage.error(String(e))
    }
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
    } catch {
      return
    }
    try {
      await DeleteEntry(entry.id)
      ElMessage.success('删除成功')
      await loadAll()
    } catch (e) {
      ElMessage.error(String(e))
    }
  }

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
    } catch {
      return
    }
    try {
      const ok = await registerHello()
      if (ok) {
        ElMessage.success('Windows Hello 已启用')
      }
    } catch (e) {
      ElMessage.error(String(e))
    }
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
    } catch {
      return
    }
    try {
      await openHelloSettings()
    } catch (e) {
      ElMessage.error(String(e))
    }
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
      } catch {
        return
      }
      try {
        await DisableHello()
        helloEnabled.value = false
        ElMessage.success('Windows Hello 已关闭')
      } catch (e) {
        ElMessage.error(String(e))
      }
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

  // ==================== Lifecycle ====================
  async function init() {
    initialized.value = await IsInitialized()
    if (!initialized.value) {
      setupDialogVisible.value = true
    }
    const availability = await GetHelloAvailability()
    helloAvailable.value = availability !== 'DeviceNotPresent' && availability !== 'Unknown'
    if (initialized.value) {
      helloEnabled.value = await IsHelloEnabled()
    }
  }

  function cleanup() {
    // Cleanup handled by KeepAlive component in App.vue
  }

  return {
    // State
    initialized,
    unlocked,
    loading,
    entries,
    searchQuery,
    categoryTree,
    tags,
    selectedCategoryId,
    selectedTagId,
    categoryTreeRef,
    helloAvailable,
    helloEnabled,
    masterPassword,
    confirmPassword,
    setupDialogVisible,
    unlockDialogVisible,
    editDialogVisible,
    isEdit,
    editForm,
    categoryDialogVisible,
    categoryForm,
    tagDialogVisible,
    tagForm,
    tagColorPresets,
    totalEntries,
    filteredEntries,
    currentFilterLabel,

    // Functions
    loadAll,
    loadEntries,
    loadCategories,
    loadTags,
    selectAll,
    handleCategoryClick,
    handleTagClick,
    openAddCategory,
    openEditCategory,
    handleSaveCategory,
    handleDeleteCategory,
    openAddTag,
    openEditTag,
    handleSaveTag,
    handleDeleteTag,
    openAddDialog,
    openEditDialog,
    handleSave,
    handleDelete,
    handleSetup,
    handleUnlock,
    handleLock,
    promptEnableHello,
    openHelloSettings,
    offerOpenHelloSettings,
    registerHello,
    handleHelloUnlock,
    toggleHello,
    copyPassword,
    copyUsername,
    openUrl,
    getInitial,
    getAvatarColor,
    findCategoryNode,
    flattenCategories,
    init,
    cleanup,
  }
}
