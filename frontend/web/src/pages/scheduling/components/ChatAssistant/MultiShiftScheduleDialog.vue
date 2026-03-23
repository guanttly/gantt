<script setup lang="ts">
import { Calendar, Download } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElMessage, ElTable, ElTableColumn, ElTabPane, ElTabs, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { exportScheduleByPersonToExcel } from './exportScheduleByPerson'

interface DayShift {
  staff: string[]
  staffIds: string[]
  requiredCount: number
  actualCount: number
}

interface ShiftDraft {
  shiftId: string
  priority: number
  days: Record<string, DayShift>
}

interface ShiftInfo {
  id: string
  name: string
  startTime?: string   // "HH:MM" 开始时间，用于判断夜班
  endTime?: string     // "HH:MM" 结束时间
  isOvernight?: boolean
  type?: string
}

interface MultiShiftScheduleData {
  startDate: string
  endDate: string
  shifts: Record<string, ShiftDraft>
  shiftInfoList: ShiftInfo[]
}

interface Props {
  visible: boolean
  data: MultiShiftScheduleData | null
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '排班详情',
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 当前选中的标签页
const activeTab = ref<string>('')

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
function getRowClassName({ row }: { row: { date: string } }): string {
  if (isSunday(row.date))
    return 'row-sunday'
  if (isSaturday(row.date))
    return 'row-saturday'
  return ''
}

// 获取班次列表（按优先级排序）
const shiftList = computed(() => {
  if (!props.data?.shifts || !props.data?.shiftInfoList) {
    return []
  }

  const shiftMap = new Map<string, ShiftInfo>()
  props.data.shiftInfoList.forEach((info) => {
    shiftMap.set(info.id, info)
  })

  return Object.keys(props.data.shifts)
    .map((shiftId) => {
      const shiftInfo = shiftMap.get(shiftId)
      const shiftDraft = props.data!.shifts[shiftId]

      return {
        shiftId,
        shiftName: shiftInfo?.name || `班次-${shiftId}`,
        priority: shiftDraft.priority || 0,
        draft: shiftDraft,
      }
    })
    .sort((a, b) => a.priority - b.priority)
})

// 初始化时设置第一个标签页为激活状态
watch(() => props.visible, (newVal) => {
  if (newVal && shiftList.value.length > 0) {
    activeTab.value = shiftList.value[0].shiftId
  }
})

// 监听数据变化，确保标签页正确更新
watch(() => props.data, (newData) => {
  if (newData && shiftList.value.length > 0) {
    // 如果当前activeTab不在新的shiftList中，重置为第一个
    const isValidTab = shiftList.value.some(s => s.shiftId === activeTab.value)
    if (!isValidTab) {
      activeTab.value = shiftList.value[0].shiftId
    }
  }
}, { deep: true })

// 获取当前选中班次的表格数据
const currentTableData = computed(() => {
  if (!activeTab.value || !props.data?.shifts) {
    return []
  }

  const shiftDraft = props.data.shifts[activeTab.value]

  if (!shiftDraft?.days) {
    return []
  }

  const days = shiftDraft.days
  return Object.keys(days).sort().map(date => ({
    date,
    ...days[date],
  }))
})

// 计算总天数
const totalDays = computed(() => currentTableData.value.length)

function handleClose() {
  emit('update:visible', false)
  emit('close')
}

function handleExport() {
  if (!props.data) {
    ElMessage.warning('暂无排班数据可导出')
    return
  }
  try {
    exportScheduleByPersonToExcel(props.data)
    ElMessage.success('导出成功')
  }
  catch (err) {
    const msg = err instanceof Error ? err.message : '导出失败'
    ElMessage.error(msg)
  }
}
</script>

<template>
  <ElDialog
    :model-value="visible"
    :title="props.title"
    width="900px"
    append-to-body
    destroy-on-close
    @close="handleClose"
  >
    <div v-if="data && shiftList.length > 0" class="multi-shift-schedule-content">
      <!-- 头部信息 -->
      <div class="header-info mb-4">
        <ElTag effect="plain" class="mr-2">
          <el-icon class="mr-1">
            <Calendar />
          </el-icon>
          {{ data.startDate }} ~ {{ data.endDate }}
        </ElTag>
        <ElTag type="info" effect="plain">
          共 {{ shiftList.length }} 个班次
        </ElTag>
      </div>

      <!-- 标签页 -->
      <ElTabs v-model="activeTab" type="card" class="shift-tabs">
        <ElTabPane
          v-for="shift in shiftList"
          :key="shift.shiftId"
          :label="shift.shiftName"
          :name="shift.shiftId"
        >
          <div :key="activeTab" class="tab-content">
            <div class="shift-header mb-3">
              <ElTag type="info" effect="plain">
                共 {{ totalDays }} 天
              </ElTag>
            </div>

            <ElTable
              :key="`table-${activeTab}`"
              :data="currentTableData"
              border
              stripe
              style="width: 100%"
              max-height="500"
              :row-class-name="getRowClassName"
            >
              <ElTableColumn
                label="日期"
                width="130"
                sortable
                :sort-method="(a: any, b: any) => a.date.localeCompare(b.date)"
                fixed
              >
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
                      v-for="staff in row.staff"
                      :key="staff"
                      size="small"
                      class="mr-1 mb-1"
                    >
                      {{ staff }}
                    </ElTag>
                    <span v-if="!row.staff || row.staff.length === 0" class="text-gray-400 text-xs">
                      未安排
                    </span>
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
        </ElTabPane>
      </ElTabs>
    </div>
    <ElEmpty v-else description="暂无排班数据" />

    <template #footer>
      <div class="dialog-footer">
        <ElButton
          type="primary"
          :icon="Download"
          :disabled="!data || shiftList.length === 0"
          @click="handleExport"
        >
          导出
        </ElButton>
        <ElButton @click="handleClose">
          关闭
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.multi-shift-schedule-content {
  .header-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .shift-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 16px;
    }

    .tab-content {
      padding: 0;
    }

    .shift-header {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }
}

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

.text-gray-400 {
  color: #9ca3af;
}

.text-xs {
  font-size: 12px;
}

.text-red-500 {
  color: #ef4444;
}

.text-green-500 {
  color: #10b981;
}

.mr-1 {
  margin-right: 4px;
}

.mb-1 {
  margin-bottom: 4px;
}

.mr-2 {
  margin-right: 8px;
}

.mb-3 {
  margin-bottom: 12px;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
