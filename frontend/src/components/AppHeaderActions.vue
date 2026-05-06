<script lang="ts" setup>
import { Check, RefreshRight } from '@element-plus/icons-vue'
import {
  ocActiveTab,
  ocOpenCodeSubTab,
  slimSubTab,
  ocDirty,
  ocModelsLoading,
  handleRefreshModels,
  saveOpenCodeConfig,
} from '../composables/useOpenCode'
</script>

<template>
  <el-button
    v-if="ocOpenCodeSubTab === 'model' || (ocActiveTab === 'slim' && slimSubTab === 'agent')"
    size="small"
    :loading="ocModelsLoading"
    @click="handleRefreshModels"
  >
    <el-icon v-if="!ocModelsLoading"><RefreshRight /></el-icon><span>刷新模型</span>
  </el-button>
  <el-button
    v-if="ocActiveTab === 'slim' && slimSubTab === 'agent'"
    type="primary"
    size="small"
    @click="saveOpenCodeConfig"
    :disabled="!ocDirty"
  >
    <el-icon><Check /></el-icon><span>保存配置</span>
  </el-button>
</template>
