<script setup lang="ts">
import { ElDialog } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { getEmployeeList } from '@/api/employee'
import { getShiftList } from '@/api/shift'
import SchedulingGantt from '../SchedulingGantt.vue'

interface Props {
  visible: boolean
  draftSchedule: any // ScheduleDraft 数据（JSON 字符串或对象）
  startDate: string
  endDate: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: val => emit('update:visible', val),
})

// 甘特图日期范围
const dateRange = computed<[Date, Date]>(() => {
  const start = new Date(props.startDate)
  start.setHours(0, 0, 0, 0)
  const end = new Date(props.endDate)
  end.setHours(23, 59, 59, 999)
  return [start, end]
})

// 转换后的排班数据
const assignments = ref<Scheduling.Assignment[]>([])
const loading = ref(false)

// 解析 DraftSchedule 并转换为 assignments
async function convertDraftToAssignments() {
  if (!props.draftSchedule) {
    assignments.value = []
    return
  }

  loading.value = true
  try {
    // 解析 DraftSchedule（可能是 JSON 字符串或对象）
    let draft: any
    if (typeof props.draftSchedule === 'string') {
      try {
        draft = JSON.parse(props.draftSchedule)
      }
      catch {
        assignments.value = []
        return
      }
    }
    else {
      draft = props.draftSchedule
    }

    if (!draft || !draft.shifts) {
      assignments.value = []
      return
    }

    // 加载员工和班次信息（用于名称映射）
    // 从 localStorage 或环境变量获取 orgId
    const orgId = localStorage.getItem('orgId') || import.meta.env.VITE_DEFAULT_ORG_ID || 'default-org'
    const [empRes, shiftRes] = await Promise.all([
      getEmployeeList({ orgId, page: 1, size: 1000 }),
      getShiftList({ orgId, isActive: true, page: 1, size: 100 }),
    ])

    const employeeMap = new Map(empRes.items?.map(emp => [emp.id, emp]) || [])
    const shiftMap = new Map(shiftRes.items?.map(s => [s.id, s]) || [])

    // 转换 DraftSchedule 为 assignments
    const result: Scheduling.Assignment[] = []
    let assignmentIdCounter = 1

    // 遍历所有班次
    for (const [shiftId, shiftDraft] of Object.entries(draft.shifts || {})) {
      const shift = shiftMap.get(shiftId)
      const shiftName = shift?.name || `班次-${shiftId}`

      // 遍历该班次的所有日期
      const days = (shiftDraft as any).days || {}

      for (const [date, dayShift] of Object.entries(days)) {
        const dayData = dayShift as any

        // 遍历该日期的人员
        const staffIds = dayData.staffIds || dayData.staff || []
        if (Array.isArray(staffIds) && staffIds.length > 0) {
          for (const staffId of staffIds) {
            const employee = employeeMap.get(staffId)
            result.push({
              id: `preview-${assignmentIdCounter++}`,
              orgId,
              employeeId: staffId,
              employeeName: employee?.name || staffId,
              shiftId,
              shiftName,
              date,
              notes: '',
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            })
          }
        }
      }
    }

    assignments.value = result
  }
  catch {
    assignments.value = []
  }
  finally {
    loading.value = false
  }
}

// 监听对话框显示和 draftSchedule 变化
watch([() => props.visible, () => props.draftSchedule], ([visible, draft]) => {
  if (visible && draft) {
    convertDraftToAssignments()
  }
}, { immediate: true })
</script>

<template>
  <ElDialog
    v-model="dialogVisible"
    title="完整排班预览"
    width="90%"
    :close-on-click-modal="false"
    :close-on-press-escape="true"
    class="schedule-preview-dialog"
  >
    <div class="preview-container">
      <div class="preview-info">
        <p><strong>排班周期：</strong>{{ startDate }} 至 {{ endDate }}</p>
      </div>
      <div v-loading="loading" class="gantt-container">
        <SchedulingGantt
          v-if="!loading && assignments.length > 0"
          :date-range="dateRange"
          :assignments="assignments"
          :readonly="true"
        />
        <div v-else-if="loading" class="loading-placeholder">
          正在加载排班数据...
        </div>
        <div v-else class="empty-placeholder">
          暂无排班数据
        </div>
      </div>
    </div>
  </ElDialog>
</template>

<style lang="scss" scoped>
.schedule-preview-dialog {
  :deep(.el-dialog__body) {
    padding: 20px;
  }
}

.preview-container {
  display: flex;
  flex-direction: column;
  height: 70vh;
  min-height: 500px;
}

.preview-info {
  margin-bottom: 16px;
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
  flex-shrink: 0;

  p {
    margin: 0;
    font-size: 14px;
    color: #606266;
  }
}

.gantt-container {
  flex: 1;
  min-height: 400px;
  height: 100%;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
  position: relative;

  // 确保甘特图容器有明确的高度
  :deep(.timeline-container) {
    width: 100%;
    height: 100%;
    min-height: 400px;
  }
}

.loading-placeholder,
.empty-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 400px;
  color: #909399;
  font-size: 14px;
}
</style>
