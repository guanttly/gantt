<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { exportSchedule } from '@/api/schedules'
import { useAuthStore } from '@/stores/auth'
import SchedulingToolbar from './components/SchedulingToolbar.vue'

const SchedulingGantt = defineAsyncComponent(() => import('./components/SchedulingGantt.vue'))
const route = useRoute()
const auth = useAuthStore()
const ganttRef = ref()
const isReadonly = computed(() => !auth.hasPermission('schedule:adjust'))

// 日期范围 — 默认本周
function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const dayOfWeek = today.getDay()
const daysFromMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
const startOfWeek = new Date(today)
startOfWeek.setDate(today.getDate() + daysFromMonday)
const endOfWeek = new Date(startOfWeek)
endOfWeek.setDate(startOfWeek.getDate() + 6)

const dateRange = ref<[string, string]>([
  formatDateLocal(startOfWeek),
  formatDateLocal(endOfWeek),
])

function handleRefresh() {
  ganttRef.value?.refresh()
}

function readQueryDate(value: unknown) {
  if (typeof value !== 'string') {
    return null
  }
  const normalized = value.trim()
  return /^\d{4}-\d{2}-\d{2}$/.test(normalized) ? normalized : null
}

function syncRangeFromRoute() {
  const startDate = readQueryDate(route.query.start_date)
  const endDate = readQueryDate(route.query.end_date)
  if (!startDate || !endDate) {
    return
  }
  dateRange.value = [startDate, endDate]
}

function resolveExportFilename() {
  return `schedule-${dateRange.value[0]}-${dateRange.value[1]}.xlsx`
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

async function handleExport() {
  try {
    const response = await exportSchedule({
      start_date: dateRange.value[0],
      end_date: dateRange.value[1],
      format: 'xlsx',
    })
    downloadBlob(response.data as Blob, resolveExportFilename())
    ElMessage.success('排班导出成功')
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '排班导出失败')
  }
}

watch(() => [route.query.start_date, route.query.end_date], () => {
  syncRangeFromRoute()
}, { immediate: true })
</script>

<template>
  <div class="scheduling-page">
    <div class="scheduling-container">
      <div class="scheduling-main">
        <SchedulingToolbar
          v-model:date-range="dateRange"
          :readonly="isReadonly"
          @refresh="handleRefresh"
          @export="handleExport"
        />
        <SchedulingGantt
          ref="ganttRef"
          :date-range="dateRange"
          :readonly="isReadonly"
        />
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.scheduling-page {
  height: 100%;
  overflow: hidden;
  background: #f5f7fa;
}

.scheduling-container {
  display: flex;
  height: 100%;
  position: relative;
}

.scheduling-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  overflow: hidden;
}

@media (max-width: 768px) {
  .scheduling-page,
  .scheduling-container,
  .scheduling-main {
    min-height: 0;
  }
}
</style>
