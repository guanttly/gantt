<script setup lang="ts">
import { computed } from 'vue'
import { useSchedulingSessionStore } from '@/stores/schedulingSession'

const sessionStore = useSchedulingSessionStore()

const progress = computed(() => sessionStore.shiftProgress)
const show = computed(() => sessionStore.showProgressBar)

const progressPercent = computed(() => {
  if (!progress.value)
    return 0
  return progress.value.progress || 0
})

const statusText = computed(() => {
  if (!progress.value)
    return ''
  const p = progress.value
  return `${p.shift_name || ''} - ${p.message || p.status}`
})
</script>

<template>
  <div v-if="show && progress" class="scheduling-progress-bar">
    <div class="progress-info">
      <span class="progress-label">{{ statusText }}</span>
      <span class="progress-value">{{ progressPercent }}%</span>
    </div>
    <el-progress
      :percentage="progressPercent"
      :stroke-width="6"
      :show-text="false"
      :status="progress.status === 'failed' ? 'exception' : progress.status === 'success' ? 'success' : undefined"
    />
  </div>
</template>

<style scoped>
.scheduling-progress-bar {
  padding: 8px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fafafa;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 12px;
  color: #606266;
}
</style>
