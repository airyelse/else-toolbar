<script lang="ts" setup>
import { ref } from 'vue'
import VaultPage from './pages/VaultPage.vue'
import EnvPage from './pages/EnvPage.vue'
import RuntimePage from './pages/RuntimePage.vue'
import OpenCodePage from './pages/OpenCodePage.vue'
import ConsolePage from './pages/ConsolePage.vue'
import {
  ocActiveTab,
  ocOpenCodeSubTab,
  slimSubTab,
  ocDirty,
  ocModelsLoading,
  handleRefreshModels,
  saveOpenCodeConfig,
} from './composables/useOpenCode'
import {
  Box,
  Check,
  Lock,
  Monitor,
  RefreshRight,
  Setting,
} from '@element-plus/icons-vue'

// ==================== Navigation ====================
type Tool = 'vault' | 'env' | 'runtime' | 'opencode' | 'console'
const currentTool = ref<Tool>('vault')

const toolComponents: Record<Tool, any> = {
  vault: VaultPage,
  env: EnvPage,
  runtime: RuntimePage,
  opencode: OpenCodePage,
  console: ConsolePage,
}

const toolMeta: Record<Tool, { label: string; icon: any }> = {
  vault: { label: '密码保险箱', icon: Lock },
  env: { label: '环境变量', icon: Monitor },
  runtime: { label: '环境管理', icon: Monitor },
  opencode: { label: 'OpenCode 配置', icon: Setting },
  console: { label: '脚本控制台', icon: Monitor },
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
        <el-tooltip v-for="(meta, key) in toolMeta" :key="key" :content="meta.label" placement="right">
          <div
            class="nav-rail-item"
            :class="{ active: currentTool === key }"
            @click="currentTool = key"
          >
            <el-icon size="20"><component :is="meta.icon" /></el-icon>
          </div>
        </el-tooltip>
      </div>
    </nav>

    <!-- Right area: Header + Body -->
    <div class="app-main">
      <!-- Header -->
      <header class="header">
        <div class="header-left">
          <h1 class="header-title">{{ toolMeta[currentTool]?.label }}</h1>
        </div>
        <div class="header-actions">
          <el-button
            v-if="currentTool === 'opencode' && (ocOpenCodeSubTab === 'model' || (ocActiveTab === 'slim' && slimSubTab === 'agent'))"
            size="small"
            :loading="ocModelsLoading"
            @click="handleRefreshModels"
          >
            <el-icon v-if="!ocModelsLoading"><RefreshRight /></el-icon><span>刷新模型</span>
          </el-button>
          <el-button
            v-if="currentTool === 'opencode' && ocActiveTab === 'slim' && slimSubTab === 'agent'"
            type="primary"
            size="small"
            @click="saveOpenCodeConfig"
            :disabled="!ocDirty"
          >
            <el-icon><Check /></el-icon><span>保存配置</span>
          </el-button>
        </div>
      </header>

      <!-- Page Content -->
      <div class="body">
        <KeepAlive class="keep-alive-page">
          <component :is="toolComponents[currentTool]" />
        </KeepAlive>
      </div>
    </div>
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
  gap: 8px;
}

/* ===== Body Layout ===== */
.body {
  flex: 1;
  overflow: hidden;
}

.keep-alive-page {
  height: 100%;
}
</style>
