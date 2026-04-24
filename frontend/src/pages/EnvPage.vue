<script lang="ts" setup>
import { onMounted } from 'vue'
import { useEnvVars } from '../composables/useEnvVars'
const {
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
} = useEnvVars()

onMounted(() => {
  loadEnvVars()
})
</script>

<template>
  <div class="body" v-loading="envLoading">
    <main class="main-content">
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
            <el-button size="small" @click="handleOpenTerminal" class="path-action-btn">
              <el-icon><Monitor /></el-icon><span>终端</span>
            </el-button>
            <el-button size="small" @click="handleCleanInvalidPaths" class="path-action-btn" :disabled="invalidUserPathCount === 0">
              <el-icon><Delete /></el-icon><span>清理无效{{ invalidUserPathCount > 0 ? ` (${invalidUserPathCount})` : '' }}</span>
            </el-button>
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
              v-model="selectedProfile"
              placeholder="选择 Profile"
              size="small"
              clearable
              style="width: 200px"
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
            <el-button size="small" type="primary" @click="openMergePreview" class="path-action-btn" :disabled="!selectedProfile">
              <el-icon><Connection /></el-icon><span>合并</span>
            </el-button>
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
            v-for="(item, idx) in filteredUserPath"
            :key="'u-' + idx"
            class="path-entry"
            :class="{ 'path-invalid': !item.entry.exists }"
          >
            <div class="path-entry-index">{{ idx + 1 }}</div>
            <div class="path-entry-body" v-if="pathEditIdx !== item.originalIndex">
              <div class="path-entry-value" :title="item.entry.path">{{ item.entry.path }}</div>
              <div class="path-entry-meta">
                <el-tag v-if="item.entry.exists" type="success" size="small" effect="light">存在</el-tag>
                <el-tag v-else type="danger" size="small" effect="light">不存在</el-tag>
                <el-tag v-if="item.entry.isDir" size="small" effect="light">目录</el-tag>
                <el-tag v-else-if="item.entry.exists" type="warning" size="small" effect="light">非目录</el-tag>
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
              <el-button text size="small" @click="pathMove(item.originalIndex, -1)" class="action-btn" :disabled="idx === 0">
                <el-icon><Top /></el-icon>
              </el-button>
              <el-button text size="small" @click="pathMove(item.originalIndex, 1)" class="action-btn" :disabled="idx >= filteredUserPath.length - 1">
                <el-icon><Bottom /></el-icon>
              </el-button>
              <el-button text size="small" @click="pathStartEdit(item.originalIndex)" class="action-btn">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button text size="small" @click="copyPath(item.entry.path)" class="action-btn">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button text size="small" @click="openPathDir(item.entry.path)" class="action-btn" v-if="item.entry.exists && item.entry.isDir">
                <el-icon><FolderOpened /></el-icon>
              </el-button>
              <el-button text size="small" @click="pathDelete(item.originalIndex)" class="action-btn action-danger">
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
            v-for="(item, idx) in filteredSystemPath"
            :key="'s-' + idx"
            class="path-entry"
            :class="{ 'path-invalid': !item.entry.exists }"
          >
            <div class="path-entry-index">{{ idx + 1 }}</div>
            <div class="path-entry-body">
              <div class="path-entry-value" :title="item.entry.path">{{ item.entry.path }}</div>
              <div class="path-entry-meta">
                <el-tag v-if="item.entry.exists" type="success" size="small" effect="light">存在</el-tag>
                <el-tag v-else type="danger" size="small" effect="light">不存在</el-tag>
                <el-tag v-if="item.entry.isDir" size="small" effect="light">目录</el-tag>
                <el-tag v-else-if="item.entry.exists" type="warning" size="small" effect="light">非目录</el-tag>
              </div>
            </div>
            <div class="path-entry-actions">
              <el-button text size="small" @click="copyPath(item.entry.path)" class="action-btn">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
              <el-button text size="small" @click="openPathDir(item.entry.path)" class="action-btn" v-if="item.entry.exists && item.entry.isDir">
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
          <div v-for="item in filteredEnvList" :key="item.name" class="env-entry" :class="{ 'env-entry-path': item.isPath }">
            <div class="env-entry-name">{{ item.name }}</div>
            <div class="env-entry-body" v-if="envEditingName !== item.name">
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
              <template v-if="envEditingName === item.name">
                <el-button text size="small" @click="envConfirmEdit" class="action-btn" type="primary">
                  <el-icon><Check /></el-icon>
                </el-button>
                <el-button text size="small" @click="envCancelEdit" class="action-btn">
                  <el-icon><Close /></el-icon>
                </el-button>
              </template>
              <template v-else>
                <el-button text size="small" @click="envStartEdit(item)" class="action-btn">
                  <el-icon><Edit /></el-icon>
                </el-button>
                <el-button text size="small" @click="envDelete(item)" class="action-btn action-danger">
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

  <!-- Merge Preview dialog -->
  <el-dialog v-model="mergeDialogVisible" :title="`合并预览 — ${selectedProfile}`" width="640px" align-center>
    <div class="dialog-desc" style="margin-bottom: 12px">
      <span style="color: var(--primary)">{{ selectedProfile }}</span> 的路径将置于最前，与当前用户 PATH 自动去重合并。
    </div>
    <div v-loading="mergeLoading">
      <div v-if="mergePreview.length === 0 && !mergeLoading" class="sidebar-empty" style="padding: 16px">预览为空</div>
      <div v-else class="merge-preview-list">
        <div
          v-for="(p, idx) in mergePreview"
          :key="idx"
          class="path-entry"
          style="padding: 4px 8px"
        >
          <div class="path-entry-index" style="min-width: 28px">{{ idx + 1 }}</div>
          <div class="path-entry-value" style="flex: 1">{{ p }}</div>
          <el-tag
            v-if="!userPathStrings.some((u: string) => u.toLowerCase() === p.toLowerCase())"
            type="primary" size="small" effect="light"
          >新增</el-tag>
        </div>
      </div>
      <div style="margin-top: 8px; color: var(--text-muted); font-size: 12px">
        共 {{ mergePreview.length }} 条路径，
        <span style="color: var(--primary)">{{ mergePreview.filter(p => !userPathStrings.some((u: string) => u.toLowerCase() === p.toLowerCase())).length }} 条新增</span>
      </div>
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="mergeDialogVisible = false" style="flex: 1">取消</el-button>
        <el-button type="primary" size="large" @click="handleApplyProfile(selectedProfile)" style="flex: 1" :disabled="mergeLoading">确认合并</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
/* ===== Body Layout ===== */
.body {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ===== Main Content ===== */
.main-content {
  flex: 1;
  padding: 20px 24px;
  overflow-y: auto;
}

/* ===== PATH Viewer ===== */
.path-tabs {
  display: flex;
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
  padding: 0;
}

.path-action-btn {
  border-radius: 6px !important;
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

.action-btn {
  font-size: 12px !important;
  border-radius: 6px !important;
  color: var(--text-secondary) !important;
}

.action-btn:hover { background: var(--bg) !important; }
.action-primary { color: var(--primary) !important; }
.action-primary:hover { background: var(--primary-bg) !important; }
.action-danger:hover { color: var(--danger) !important; background: #fef2f2 !important; }

.sidebar-empty {
  padding: 8px 16px;
  font-size: 13px;
  color: var(--text-muted);
}

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
</style>
