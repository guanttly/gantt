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

interface Props {
  visible: boolean
  data: ShiftScheduleData | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 星期名称映射
const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

// 解析日期字符串
function parseDate(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number)
  return new Date(year, month - 1, day)
}

// 获取星期几（0-6）
function getWeekday(dateStr: string): number {
  return parseDate(dateStr).getDay()
}

// 格式化日期显示 (MM-DD 周X)
function formatDateWithWeekday(dateStr: string): string {
  const weekday = getWeekday(dateStr)
  const parts = dateStr.split('-')
  return `${parts[1]}-${parts[2]} ${WEEKDAY_NAMES[weekday]}`
}

// 判断是否是周六
function isSaturday(dateStr: string): boolean {
  return getWeekday(dateStr) === 6
}

// 判断是否是周日
function isSunday(dateStr: string): boolean {
  return getWeekday(dateStr) === 0
}

// 获取日期的行样式类
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
  return Object.keys(days).sort().map(date => ({
    date,
    ...days[date],
  }))
})

// 计算总天数
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
    width="700px"
    append-to-body
    destroy-on-close
    @close="handleClose"
  >
    <div v-if="data && tableData.length > 0" class="schedule-content">
      <div class="header-info mb-4">
        <ElTag effect="plain" class="mr-2">
          <el-icon class="mr-1">
            <Calendar />
          </el-icon>
          {{ data.startDate }} ~ {{ data.endDate }}
        </ElTag>
        <ElTag type="info" effect="plain">
          共 {{ totalDays }} 天
        </ElTag>
      </div>

      <ElTable :data="tableData" border stripe style="width: 100%" max-height="500" :row-class-name="getRowClassName">
        <ElTableColumn label="日期" width="130" sortable :sort-method="(a: any, b: any) => a.date.localeCompare(b.date)" fixed>
          <template #default="{ row }">
            <span :class="{ 'date-saturday': isSaturday(row.date), 'date-sunday': isSunday(row.date) }">
              {{ formatDateWithWeekday(row.date) }}
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="排班人员" min-width="200">
          <template #default="{ row }">
            <div class="staff-list">
              <!-- 当前排班人员 -->
              <ElTag
                v-for="(staff, index) in row.staff"
                :key="staff"
                size="small"
                :type="row.staffFlags?.[index]?.isAdded ? 'success' : undefined"
                :class="{
                  'mr-1 mb-1': true,
                  'staff-added': row.staffFlags?.[index]?.isAdded,
                  'staff-removed': row.staffFlags?.[index]?.isRemoved,
                }"
              >
                {{ staff }}
                <span v-if="row.staffFlags?.[index]?.isAdded" class="ml-1 text-xs">(新增)</span>
              </ElTag>
              <span v-if="!row.staff || row.staff.length === 0" class="text-gray-400 text-xs">
                未安排
              </span>
              <!-- 移除的人员 -->
              <div v-if="row.removedStaff && row.removedStaff.length > 0" class="removed-staff mt-2">
                <div class="text-xs text-gray-500 mb-1">移除的人员：</div>
                <ElTag
                  v-for="staff in row.removedStaff"
                  :key="staff"
                  size="small"
                  type="danger"
                  effect="plain"
                  class="mr-1 mb-1 staff-removed-tag"
                >
                  {{ staff }}
                </ElTag>
              </div>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn label="满足率" width="100" align="center">
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
      <div class="dialog-footer">
        <ElButton @click="handleClose">
          关闭
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<style scoped>
.staff-list {
  display: flex;
  flex-wrap: wrap;
}

/* 周六日期文字样式 - 橙色 */
.date-saturday {
  color: #e6a23c;
  font-weight: 500;
}

/* 周日日期文字样式 - 红色 */
.date-sunday {
  color: #f56c6c;
  font-weight: 500;
}

/* 周六行背景色 */
:deep(.row-saturday) {
  background-color: rgba(230, 162, 60, 0.08) !important;
}

/* 周日行背景色 */
:deep(.row-sunday) {
  background-color: rgba(245, 108, 108, 0.08) !important;
}

/* 有变化的行背景色 */
:deep(.row-changed) {
  background-color: rgba(64, 158, 255, 0.1) !important;
  border-left: 3px solid #409eff;
}

/* 新增人员标签样式 */
.staff-added {
  border: 1px solid #67c23a;
  background-color: rgba(103, 194, 58, 0.1);
}

/* 移除人员标签样式（如果显示） */
.staff-removed {
  text-decoration: line-through;
  opacity: 0.6;
}

/* 移除的人员区域 */
.removed-staff {
  border-top: 1px dashed #dcdfe6;
  padding-top: 8px;
  margin-top: 8px;
}

.staff-removed-tag {
  text-decoration: line-through;
  opacity: 0.8;
}
</style>
