<script setup lang="ts">
import { Download } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElTabPane, ElTabs, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

import { exportScheduleByPersonToExcel } from './exportScheduleByPerson'

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

interface ShiftScheduleInfo {
  shiftId: string
  shiftName: string
  startTime: string
  endTime: string
  startDate: string
  endDate: string
  schedule: ShiftDraft
}

interface MultiShiftData {
  title?: string
  shifts: ShiftScheduleInfo[]
}

const props = defineProps<{
  visible: boolean
  data: MultiShiftData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const activeTab = ref<string>('')

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

function isWeekend(dateStr: string): boolean {
  const day = getWeekday(dateStr)
  return day === 0 || day === 6
}

watch(() => props.visible, (newVal) => {
  if (newVal && props.data?.shifts?.length)
    activeTab.value = props.data.shifts[0].shiftId
})

const shiftTableData = computed(() => {
  if (!props.data?.shifts)
    return {}
  const result: Record<string, Array<{ date: string } & DayShift>> = {}
  for (const shift of props.data.shifts) {
    if (!shift.schedule?.days)
      continue
    result[shift.shiftId] = Object.keys(shift.schedule.days)
      .sort()
      .map(date => ({ date, ...shift.schedule.days[date] }))
  }
  return result
})

function handleExport() {
  if (!props.data)
    return
  const shiftsRecord: Record<string, { shiftId: string, priority: number, days: Record<string, DayShift> }> = {}
  const shiftInfoList: Array<{ id: string, name: string, startTime?: string, endTime?: string }> = []
  for (const s of props.data.shifts) {
    shiftsRecord[s.shiftId] = {
      shiftId: s.shiftId,
      priority: s.schedule?.priority ?? 0,
      days: s.schedule?.days ?? {},
    }
    shiftInfoList.push({
      id: s.shiftId,
      name: s.shiftName,
      startTime: s.startTime,
      endTime: s.endTime,
    })
  }
  exportScheduleByPersonToExcel({
    shifts: shiftsRecord,
    shiftInfoList,
    startDate: props.data.shifts[0]?.startDate ?? '',
    endDate: props.data.shifts[0]?.endDate ?? '',
  })
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
    :title="data?.title ?? '全部班次排班'"
    destroy-on-close
    top="5vh"
    width="850px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data && data.shifts && data.shifts.length > 0" class="multi-shift-content">
      <div class="toolbar">
        <ElButton :icon="Download" type="primary" @click="handleExport">
          导出 Excel
        </ElButton>
      </div>

      <ElTabs v-model="activeTab" type="border-card">
        <ElTabPane
          v-for="shift in data.shifts"
          :key="shift.shiftId"
          :label="shift.shiftName"
          :name="shift.shiftId"
        >
          <div class="shift-info-header">
            <ElTag effect="plain" size="small">
              {{ shift.startTime }} - {{ shift.endTime }}
            </ElTag>
            <ElTag effect="plain" size="small" type="info">
              {{ shift.startDate }} ~ {{ shift.endDate }}
            </ElTag>
          </div>

          <div v-if="shiftTableData[shift.shiftId]?.length" class="schedule-table">
            <table class="table-bordered">
              <thead>
                <tr>
                  <th class="col-date">
                    日期
                  </th>
                  <th class="col-staff">
                    排班人员
                  </th>
                  <th class="col-count">
                    满足率
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in shiftTableData[shift.shiftId]"
                  :key="row.date"
                  :class="{ 'row-weekend': isWeekend(row.date), 'row-changed': row.isChanged }"
                >
                  <td :class="{ 'text-weekend': isWeekend(row.date) }" class="col-date">
                    {{ formatDateWithWeekday(row.date) }}
                  </td>
                  <td class="col-staff">
                    <div class="staff-tags">
                      <ElTag
                        v-for="(staff, index) in row.staff"
                        :key="staff"
                        :type="row.staffFlags?.[index]?.isAdded ? 'success' : undefined"
                        size="small"
                      >
                        {{ staff }}
                      </ElTag>
                      <span v-if="!row.staff?.length" class="no-staff">未安排</span>
                    </div>
                  </td>
                  <td class="col-count" :class="{ deficit: row.actualCount < row.requiredCount }">
                    {{ row.actualCount }}/{{ row.requiredCount }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <ElEmpty v-else :image-size="80" description="该班次暂无排班数据" />
        </ElTabPane>
      </ElTabs>
    </div>
    <ElEmpty v-else description="暂无排班数据" />

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.multi-shift-content {
  .toolbar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  }
}

.shift-info-header {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.schedule-table {
  max-height: 500px;
  overflow-y: auto;
}

.table-bordered {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;

  th, td {
    border: 1px solid var(--el-border-color-light);
    padding: 8px 12px;
    text-align: left;
  }

  th {
    background: var(--el-fill-color-light);
    font-weight: 600;
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .col-date { width: 130px; }
  .col-count { width: 80px; text-align: center; }
}

.row-weekend { background: rgba(245, 108, 108, 0.06); }
.row-changed { background: rgba(64, 158, 255, 0.08); border-left: 3px solid #409eff; }
.text-weekend { color: #f56c6c; font-weight: 500; }
.deficit { color: #f56c6c; font-weight: 600; }
.no-staff { color: #c0c4cc; font-size: 12px; }

.staff-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
</style>
