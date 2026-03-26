<script setup lang="ts">
import { Calendar } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { computed } from 'vue'

interface StaffFlag {
  isAdded?: boolean
  isRemoved?: boolean
}

interface DayShift {
  staff: string[]
  staffIds: string[]
  staffFlags?: StaffFlag[]
  removedStaff?: string[]
  removedStaffIds?: string[]
  requiredCount: number
  actualCount: number
  isChanged?: boolean
}

interface ShiftDraft {
  shiftId: string
  priority: number
  days: Record<string, DayShift>
}

interface ShiftScheduleData {
  shiftId: string
  shiftName: string
  startDate: string
  endDate: string
  schedule: ShiftDraft
}

const props = defineProps<{
  visible: boolean
  data: ShiftScheduleData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

function parseDate(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function getWeekday(dateStr: string): number {
  return parseDate(dateStr).getDay()
}

function formatDateWithWeekday(dateStr: string): string {
  const weekday = getWeekday(dateStr)
  const parts = dateStr.split('-')
  return `${parts[1]}-${parts[2]} ${WEEKDAY_NAMES[weekday]}`
}

function isSaturday(dateStr: string): boolean {
  return getWeekday(dateStr) === 6
}

function isSunday(dateStr: string): boolean {
  return getWeekday(dateStr) === 0
}

function getRowClassName({ row }: { row: DayShift & { date: string } }): string {
  const classes: string[] = []
  if (isSunday(row.date))
    classes.push('row-sunday')
  if (isSaturday(row.date))
    classes.push('row-saturday')
  if (row.isChanged)
    classes.push('row-changed')
  return classes.join(' ')
}

const tableData = computed(() => {
  if (!props.data?.schedule?.days)
    return []
  const days = props.data.schedule.days
  return Object.keys(days).sort().map(date => ({ date, ...days[date] }))
})

const totalDays = computed(() => tableData.value.length)

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :model-value="visible"
    :title="data?.shiftName ? `${data.shiftName} - 排班详情` : '排班详情'"
    append-to-body
    destroy-on-close
    width="700px"
    @close="handleClose"
  >
    <div v-if="data && tableData.length > 0" class="schedule-content">
      <div class="header-info">
        <ElTag class="mr-2" effect="plain">
          <el-icon class="mr-1">
            <Calendar />
          </el-icon>
          {{ data.startDate }} ~ {{ data.endDate }}
        </ElTag>
        <ElTag effect="plain" type="info">
          共 {{ totalDays }} 天
        </ElTag>
      </div>

      <ElTable :data="tableData" :row-class-name="getRowClassName" border max-height="500" stripe style="width: 100%">
        <ElTableColumn fixed label="日期" sortable width="130" :sort-method="(a: any, b: any) => a.date.localeCompare(b.date)">
          <template #default="{ row }">
            <span :class="{ 'date-saturday': isSaturday(row.date), 'date-sunday': isSunday(row.date) }">
              {{ formatDateWithWeekday(row.date) }}
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="排班人员" min-width="200">
          <template #default="{ row }">
            <div class="staff-list">
              <ElTag
                v-for="(staff, index) in row.staff"
                :key="staff"
                :class="{ 'staff-added': row.staffFlags?.[index]?.isAdded }"
                :type="row.staffFlags?.[index]?.isAdded ? 'success' : undefined"
                class="mr-1 mb-1"
                size="small"
              >
                {{ staff }}
                <span v-if="row.staffFlags?.[index]?.isAdded" class="ml-1 text-xs">(新增)</span>
              </ElTag>
              <span v-if="!row.staff || row.staff.length === 0" class="text-gray-400 text-xs">未安排</span>
              <div v-if="row.removedStaff && row.removedStaff.length > 0" class="removed-staff">
                <div class="text-xs text-gray-500 mb-1">
                  移除的人员：
                </div>
                <ElTag v-for="s in row.removedStaff" :key="s" class="mr-1 mb-1 staff-removed-tag" effect="plain" size="small" type="danger">
                  {{ s }}
                </ElTag>
              </div>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn align="center" label="满足率" width="100">
          <template #default="{ row }">
            <span :class="{ 'text-red-500': row.actualCount < row.requiredCount, 'text-green-500': row.actualCount >= row.requiredCount }">
              {{ row.actualCount }} / {{ row.requiredCount }}
            </span>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>
    <ElEmpty v-else description="暂无排班数据" />

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style scoped>
.header-info {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.staff-list { display: flex; flex-wrap: wrap; }
.date-saturday { color: #e6a23c; font-weight: 500; }
.date-sunday { color: #f56c6c; font-weight: 500; }

:deep(.row-saturday) { background-color: rgba(230, 162, 60, 0.08) !important; }
:deep(.row-sunday) { background-color: rgba(245, 108, 108, 0.08) !important; }
:deep(.row-changed) { background-color: rgba(64, 158, 255, 0.1) !important; border-left: 3px solid #409eff; }

.staff-added { border: 1px solid #67c23a; background-color: rgba(103, 194, 58, 0.1); }
.removed-staff { border-top: 1px dashed #dcdfe6; padding-top: 8px; margin-top: 8px; }
.staff-removed-tag { text-decoration: line-through; opacity: 0.8; }

.mr-1 { margin-right: 4px; }
.mr-2 { margin-right: 8px; }
.mb-1 { margin-bottom: 4px; }
.ml-1 { margin-left: 4px; }
.text-xs { font-size: 12px; }
.text-gray-400 { color: #9ca3af; }
.text-gray-500 { color: #6b7280; }
.text-red-500 { color: #ef4444; }
.text-green-500 { color: #10b981; }
</style>
