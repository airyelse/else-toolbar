<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import { GetCloseBehavior, SetCloseBehavior, QuitApp, HideWindow } from '../../bindings/else-toolbox/app'

const closeDialogVisible = ref(false)
const closeRemember = ref(false)
let closeCleanup: (() => void) | null = null

onMounted(() => {
  closeCleanup = Events.On('window:close-requested', async () => {
    const behavior = await GetCloseBehavior()
    if (behavior === 'quit') {
      QuitApp()
      return
    }
    if (behavior === 'minimize') {
      HideWindow()
      return
    }
    closeRemember.value = false
    closeDialogVisible.value = true
  })
})

onUnmounted(() => {
  closeCleanup?.()
})

async function handleCloseAction(behavior: 'quit' | 'minimize') {
  closeDialogVisible.value = false

  if (closeRemember.value) {
    await SetCloseBehavior(behavior)
  }

  if (behavior === 'quit') {
    QuitApp()
    return
  }

  HideWindow()
}
</script>

<template>
  <el-dialog
    v-model="closeDialogVisible"
    title="关闭窗口"
    width="420px"
    align-center
    :close-on-click-modal="false"
  >
    <div class="close-dialog-content">
      <p>关闭程序将退出应用，所有功能停止运行。最小化到托盘可保持后台运行，通过系统托盘图标恢复窗口。</p>
      <el-checkbox v-model="closeRemember">记住选择，不再询问</el-checkbox>
    </div>
    <template #footer>
      <div style="display: flex; gap: 12px; width: 100%">
        <el-button size="large" @click="closeDialogVisible = false" style="flex: 1">暂不关闭</el-button>
        <el-button type="danger" size="large" @click="handleCloseAction('quit')" style="flex: 1">关闭程序</el-button>
        <el-button type="primary" size="large" @click="handleCloseAction('minimize')" style="flex: 1">最小化到托盘</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.close-dialog-content p {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 16px;
  line-height: 1.6;
}
</style>
