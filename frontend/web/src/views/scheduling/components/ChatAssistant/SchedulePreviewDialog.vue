<script setup lang="ts">
import { Check, Close, View } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElMessage, ElTag } from 'element-plus'
import { computed, ref } from 'vue'

interface PreviewShiftDay {
  date: string
  staffNames: string[]
  staffCount: number
  requiredCount: number
}

interface PreviewShift {
  shiftId: string
  shiftName: string
  startTime: string
  endTime: string
  days: PreviewShiftDay[]
}

interface SchedulePreviewData {
  title?: string
  startDate: string
  endDate: string
  shifts: PreviewShift[]
  canPublish?: boolean
}

const props = defineProps<{
  visible: boolean
  data: SchedulePreviewData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
  'confirm': []
}>()

const isLoading = ref(false)

const totalDays = computed(() => {
  if (!props.data?.shifts?.[0]?.days)
    return 0
  return props.data.shifts[0].days.length
})

const totalStaff = computed(() => {
  if (!props.data?.shifts)
    return 0
  const staffSet = new Set<string>()
  for (const shift of props.data.shifts) {
    for (const day of shift.days) {
      for (const name of day.staffNames) {
        staffSet.add(name)
      }
    }
  }
  return staffSet.size
})

const fulfillmentRate = computed(() => {
  if (!props.data?.shifts)
    return '0%'
  let totalRequired = 0
  let totalActual = 0
  for (const shift of props.data.shifts) {
    for (const day of shift.days) {
      totalRequired += day.requiredCount
      totalActual += day.staffCount
    }
  }
  if (totalRequired === 0)
    return '100%'
  return `${Math.round((totalActual / totalRequired) * 100)}%`
})

async function handleConfirm() {
  isLoading.value = true
  try {
    emit('confirm')
    ElMessage.success('排班已确认')
  }
  finally {
    isLoading.value = false
  }
}

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :before-close="handleClose"
    :model-value="visible"
    :title="data?.title ?? '排班预览'"
    destroy-on-close
    top="5vh"
    width="800px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="preview-content">
      <div class="preview-summary">
        <div class="summary-item">
          <div class="summary-label">
            排班周期
          </div>
          <div class="summary-value">
            {{ data.startDate }} ~ {{ data.endDate }}
          </div>
        </div>
        <div class="summary-item">
          <div class="summary-label">
            班次数
          </div>
          <div class="summary-value">
            {{ data.shifts.length }}
          </div>
        </div>
        <div class="summary-item">
          <div class="summary-label">
            天数
          </div>
          <div class="summary-value">
            {{ totalDays }}
          </div>
        </div>
        <div class="summary-item">
          <div class="summary-label">
            涉及人员
          </div>
          <div class="summary-value">
            {{ totalStaff }}
          </div>
        </div>
        <div class="summary-item">
          <div class="summary-label">
            满足率
          </div>
          <div class="summary-value" :class="{ 'text-success': fulfillmentRate === '100%', 'text-warning': fulfillmentRate !== '100%' }">
            {{ fulfillmentRate }}
          </div>
        </div>
      </div>

      <div class="shifts-preview">
        <div
          v-for="shift in data.shifts"
          :key="shift.shiftId"
          class="shift-section"
        >
          <div class="shift-header">
            <el-icon><View /></el-icon>
            <span class="shift-name">{{ shift.shiftName }}</span>
            <ElTag effect="plain" size="small">
              {{ shift.startTime }} - {{ shift.endTime }}
            </ElTag>
          </div>
          <div class="days-grid">
            <div
              v-for="day in shift.days"
              :key="day.date"
              class="day-cell"
              :class="{ 'day-deficit': day.staffCount < day.requiredCount }"
            >
              <div class="day-date">
                {{ day.date.slice(5) }}
              </div>
              <div class="day-count" :class="{ deficit: day.staffCount < day.requiredCount }">
                {{ day.staffCount }}/{{ day.requiredCount }}
              </div>
              <div class="day-staff">
                <span v-for="name in day.staffNames" :key="name" class="staff-chip">{{ name }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <ElEmpty v-else description="暂无预览数据" />

    <template #footer>
      <ElButton :icon="Close" @click="handleClose">
        取消
      </ElButton>
      <ElButton v-if="data?.canPublish" :icon="Check" :loading="isLoading" type="primary" @click="handleConfirm">
        确认发布
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.preview-content { padding: 8px 0; }

.preview-summary {
  display: flex;
  gap: 20px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  margin-bottom: 20px;
}

.summary-item { text-align: center; flex: 1; }
.summary-label { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.summary-value { font-size: 18px; font-weight: 600; }
.text-success { color: #67c23a; }
.text-warning { color: #e6a23c; }

.shifts-preview {
  max-height: 500px;
  overflow-y: auto;
}

.shift-section {
  margin-bottom: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 16px;
}

.shift-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.shift-name { font-size: 15px; font-weight: 600; }

.days-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
}

.day-cell {
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  text-align: center;
  min-height: 60px;
}

.day-deficit { border-color: rgba(245, 108, 108, 0.5); background: rgba(245, 108, 108, 0.05); }
.day-date { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.day-count { font-size: 14px; font-weight: 600; margin-bottom: 4px; }
.deficit { color: #f56c6c; }

.day-staff {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 2px;
}

.staff-chip {
  font-size: 10px;
  padding: 1px 4px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 3px;
}
</style>
