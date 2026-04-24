<script lang="ts" setup>
import { onMounted, onUnmounted } from 'vue'
import {
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
  openConfigDialog,
  handleSaveConfig,
  handleBrowseRuntimeDir,
  copyPath,
  openPathDir,
  setupProgressListener,
  cleanupProgressListener,
} from '../composables/useRuntime'

onMounted(() => {
  loadSDKs()
  setupProgressListener()
})

onUnmounted(() => {
  cleanupProgressListener()
})
</script>

<template>
  <div class="body" v-loading="sdkLoading">
    <main class="main-content">
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
  height: 28px !important;
  padding: 0 8px !important;
}

.path-action-btn {
  border-radius: 6px !important;
}

/* ===== Action Buttons ===== */
.action-btn {
  font-size: 12px !important;
  border-radius: 6px !important;
  color: var(--text-secondary) !important;
}

.action-btn:hover { background: var(--bg) !important; }
.action-primary { color: var(--primary) !important; }
.action-primary:hover { background: var(--primary-bg) !important; }
.action-danger:hover { color: var(--danger) !important; background: #fef2f2 !important; }
</style>
