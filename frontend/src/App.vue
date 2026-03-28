<script lang="ts" setup>
import { ref, onMounted, computed, nextTick } from 'vue'
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
} from '../wailsjs/go/main/App'
import type { models } from '../wailsjs/go/models'

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

// ==================== State ====================
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
    await loadAll()
    helloEnabled.value = await IsHelloEnabled()
    ElMessage.success('设置成功')
    // Prompt to enable Windows Hello
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
    categoryTree.value = tree.map((c: any) => ({
      id: c.id,
      label: c.name,
      children: c.children?.length ? c.children.map((ch: any) => ({ id: ch.id, label: ch.name })) : undefined,
    }))
  } catch { /* ignore */ }
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

function openEditCategory(node: CategoryNode) {
  // Find parent id
  let parentId: number | null = null
  for (const root of categoryTree.value) {
    if (root.children?.some(c => c.id === node.id)) {
      parentId = root.id
      break
    }
  }
  categoryForm.value = { id: node.id, name: node.label, parentId }
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
    ElMessage.success('密码已复制到剪贴板')
  } catch (e: any) {
    ElMessage.error(e.message || '复制失败')
  }
}

async function copyUsername(username: string) {
  await navigator.clipboard.writeText(username)
  ElMessage.success('用户名已复制到剪贴板')
}

function openUrl(url: string) {
  if (url) window.open(url, '_blank')
}
</script>

<template>
  <div class="app-container">
    <!-- Header -->
    <header class="header">
      <div class="header-left">
        <div class="logo">
          <div class="logo-icon"><el-icon size="20"><Lock /></el-icon></div>
          <span class="logo-text">VaultBox</span>
        </div>
        <el-divider direction="vertical" />
        <span class="header-subtitle">密码管理器</span>
      </div>
      <div class="header-actions" v-if="unlocked">
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
      <div class="header-actions" v-else-if="initialized">
        <el-button type="primary" @click="unlockDialogVisible = true">
          <el-icon><Unlock /></el-icon><span>解锁密码库</span>
        </el-button>
      </div>
    </header>

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

    <!-- ==================== Dialogs ==================== -->

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
  </div>
</template>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg);
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

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  background: linear-gradient(135deg, var(--primary), var(--primary-light));
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.logo-text {
  font-size: 17px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--primary), var(--primary-light));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.header-subtitle {
  font-size: 13px;
  color: var(--text-muted);
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
  align-items: baseline;
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

/* ===== Transitions ===== */
.card-enter-active { transition: all 0.3s ease; }
.card-leave-active { transition: all 0.2s ease; }
.card-enter-from { opacity: 0; transform: translateY(12px); }
.card-leave-to { opacity: 0; transform: scale(0.95); }
</style>
