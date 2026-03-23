<script setup lang="ts">
import { ArrowDown, ArrowLeft, ArrowRight, Calendar } from '@element-plus/icons-vue'
import { computed, ref, watch } from 'vue'

interface Props {
  /** 班次名称 */
  shiftName: string
  /** 班次时间 (如 "08:00-16:00") */
  shiftTime?: string
  /** 班次颜色 */
  shiftColor?: string
  /** 排班开始日期 (YYYY-MM-DD) */
  startDate: string
  /** 排班结束日期 (YYYY-MM-DD) */
  endDate: string
  /** 每天人数配置 { "2024-01-01": 2, ... } */
  modelValue: Record<string, number>
  /** 是否折叠 */
  collapsed?: boolean
  /** 是否为首个（默认展开） */
  isFirst?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  shiftColor: '#409eff',
  collapsed: false,
  isFirst: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, number>]
  'update:collapsed': [value: boolean]
}>()

// 星期简写
const WEEKDAY_SHORT = ['一', '二', '三', '四', '五', '六', '日']

// 内部状态
const isCollapsed = ref(props.collapsed)
const currentWeekIndex = ref(0)
const uniformValue = ref(1)
const applyScope = ref<'week' | 'all'>('week')

// 监听外部 collapsed 变化
watch(() => props.collapsed, (val) => {
  isCollapsed.value = val
})

// 初始化：首个默认展开
watch(() => props.isFirst, (val) => {
  if (val) {
    isCollapsed.value = false
  }
}, { immediate: true })

// 解析日期
function parseDate(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number)
  return new Date(year, month - 1, day)
}

// 格式化日期
function formatDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 格式化显示日期 (MM.DD)
function formatDisplayDate(dateStr: string): string {
  const parts = dateStr.split('-')
  return `${parts[1]}.${parts[2]}`
}

// 获取日期范围内的所有日期
const allDates = computed(() => {
  const dates: string[] = []
  const start = parseDate(props.startDate)
  const end = parseDate(props.endDate)
  const endTime = end.getTime()

  let currentTime = start.getTime()
  while (currentTime <= endTime) {
    dates.push(formatDate(new Date(currentTime)))
    currentTime += 24 * 60 * 60 * 1000 // 加一天
  }
  return dates
})

// 获取包含排班日期的所有自然周
const weeks = computed(() => {
  if (allDates.value.length === 0)
    return []

  const result: { weekStart: string, weekEnd: string, dates: string[] }[] = []
  const start = parseDate(props.startDate)
  const end = parseDate(props.endDate)
  const endTime = end.getTime()

  // 找到第一个周一
  const firstMonday = new Date(start)
  const dayOfWeek = firstMonday.getDay()
  const daysToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
  firstMonday.setDate(firstMonday.getDate() + daysToMonday)

  // 生成每一周
  let mondayTime = firstMonday.getTime()
  while (mondayTime <= endTime) {
    const weekDates: string[] = []
    for (let i = 0; i < 7; i++) {
      const date = new Date(mondayTime + i * 24 * 60 * 60 * 1000)
      weekDates.push(formatDate(date))
    }

    const weekEnd = new Date(mondayTime + 6 * 24 * 60 * 60 * 1000)

    result.push({
      weekStart: formatDate(new Date(mondayTime)),
      weekEnd: formatDate(weekEnd),
      dates: weekDates,
    })

    mondayTime += 7 * 24 * 60 * 60 * 1000 // 加一周
  }

  return result
})

// 当前周数据
const currentWeek = computed(() => {
  return weeks.value[currentWeekIndex.value] || null
})

// 当前周显示信息
const currentWeekLabel = computed(() => {
  if (!currentWeek.value)
    return ''
  return `第 ${currentWeekIndex.value + 1} 周 (${formatDisplayDate(currentWeek.value.weekStart)} - ${formatDisplayDate(currentWeek.value.weekEnd)})`
})

// 是否可以上一周/下一周
const canPrevWeek = computed(() => currentWeekIndex.value > 0)
const canNextWeek = computed(() => currentWeekIndex.value < weeks.value.length - 1)

// 判断日期是否在排班范围内
function isInRange(dateStr: string): boolean {
  return allDates.value.includes(dateStr)
}

// 判断是否是周末
function isWeekend(dateStr: string): boolean {
  const date = parseDate(dateStr)
  const day = date.getDay()
  return day === 0 || day === 6
}

// 获取某天的人数
function getStaffCount(dateStr: string): number {
  return props.modelValue[dateStr] ?? 0
}

// 设置某天的人数
function setStaffCount(dateStr: string, count: number) {
  const newValue = { ...props.modelValue }
  newValue[dateStr] = Math.max(0, Math.min(99, count))
  emit('update:modelValue', newValue)
}

// 切换折叠状态
function toggleCollapse() {
  isCollapsed.value = !isCollapsed.value
  emit('update:collapsed', isCollapsed.value)
}

// 上一周
function prevWeek() {
  if (canPrevWeek.value) {
    currentWeekIndex.value--
  }
}

// 下一周
function nextWeek() {
  if (canNextWeek.value) {
    currentWeekIndex.value++
  }
}

// 应用统一值
function applyUniform() {
  const newValue = { ...props.modelValue }
  const targetDates = applyScope.value === 'week'
    ? (currentWeek.value?.dates.filter(d => isInRange(d)) || [])
    : allDates.value

  targetDates.forEach((dateStr) => {
    newValue[dateStr] = Math.max(1, Math.min(99, uniformValue.value))
  })
  emit('update:modelValue', newValue)
}

// 摘要信息
const summary = computed(() => {
  const configuredDays = allDates.value.filter(d => props.modelValue[d] > 0)
  const totalCount = configuredDays.reduce((sum, d) => sum + (props.modelValue[d] || 0), 0)
  const avgCount = configuredDays.length > 0 ? (totalCount / configuredDays.length).toFixed(1) : '0'

  return {
    configuredDays: configuredDays.length,
    totalDays: allDates.value.length,
    avgCount,
  }
})

// 图标背景色
const iconBgColor = computed(() => `${props.shiftColor}20`)
</script>

<template>
  <div class="daily-grid-field">
    <!-- 头部区域 -->
    <div class="field-header" @click="toggleCollapse">
      <!-- 左侧：折叠按钮 + 班次信息 -->
      <div class="header-left">
        <el-icon class="collapse-icon" :class="{ collapsed: isCollapsed }">
          <ArrowDown v-if="!isCollapsed" />
          <ArrowRight v-else />
        </el-icon>
        <div class="shift-icon" :style="{ backgroundColor: iconBgColor, borderColor: shiftColor }">
          <el-icon :color="shiftColor" :size="18">
            <Calendar />
          </el-icon>
        </div>
        <div class="shift-info">
          <div class="shift-name" :style="{ color: shiftColor }">
            {{ shiftName }}
          </div>
          <div v-if="shiftTime" class="shift-time">
            {{ shiftTime }}
          </div>
        </div>
      </div>

      <!-- 右侧：摘要信息（折叠时显示） -->
      <div v-if="isCollapsed" class="header-summary">
        <el-tag size="small" type="info">
          已配置 {{ summary.configuredDays }}/{{ summary.totalDays }} 天
        </el-tag>
        <el-tag size="small">
          平均 {{ summary.avgCount }} 人/天
        </el-tag>
      </div>
    </div>

    <!-- 内容区域（折叠时隐藏） -->
    <div v-show="!isCollapsed" class="field-content">
      <!-- 快捷操作区 -->
      <div class="quick-actions" @click.stop>
        <el-input-number
          v-model="uniformValue"
          :min="0"
          :max="99"
          size="small"
          controls-position="right"
          style="width: 100px"
        />
        <el-select v-model="applyScope" size="small" style="width: 100px">
          <el-option label="本周" value="week" />
          <el-option label="全部" value="all" />
        </el-select>
        <el-button size="small" @click="applyUniform">
          统一设置
        </el-button>
      </div>

      <!-- 周分页器 -->
      <div v-if="weeks.length > 1" class="week-pagination">
        <el-button
          :icon="ArrowLeft"
          size="small"
          :disabled="!canPrevWeek"
          @click="prevWeek"
        />
        <span class="week-label">{{ currentWeekLabel }}</span>
        <el-button
          :icon="ArrowRight"
          size="small"
          :disabled="!canNextWeek"
          @click="nextWeek"
        />
      </div>

      <!-- 7天卡片网格 -->
      <div v-if="currentWeek" class="weekday-cards">
        <div
          v-for="(dateStr, index) in currentWeek.dates"
          :key="dateStr"
          class="weekday-card"
          :class="{
            weekend: isWeekend(dateStr),
            disabled: !isInRange(dateStr),
          }"
        >
          <div class="weekday-name">
            {{ WEEKDAY_SHORT[index === 0 ? 0 : index] }}
          </div>
          <div class="date-display">
            {{ formatDisplayDate(dateStr) }}
          </div>
          <el-input-number
            v-if="isInRange(dateStr)"
            :model-value="getStaffCount(dateStr)"
            :min="0"
            :max="99"
            size="default"
            controls-position="right"
            class="staff-input"
            @update:model-value="(val: number | undefined) => setStaffCount(dateStr, val ?? 0)"
            @click.stop
          />
          <div v-else class="disabled-text">
            不在范围
          </div>
          <div v-if="isInRange(dateStr)" class="unit">
            人
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.daily-grid-field {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  overflow: hidden;
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  cursor: pointer;
  background: var(--el-fill-color-lighter);
  transition: background 0.2s;

  &:hover {
    background: var(--el-fill-color-light);
  }
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.collapse-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  transition: transform 0.2s;

  &.collapsed {
    transform: rotate(0deg);
  }
}

.shift-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid;
  flex-shrink: 0;
}

.shift-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.shift-name {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
}

.shift-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.header-summary {
  display: flex;
  gap: 8px;
}

.field-content {
  padding: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.week-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 12px;

  .week-label {
    font-size: 13px;
    color: var(--el-text-color-regular);
    min-width: 180px;
    text-align: center;
  }
}

.weekday-cards {
  display: flex;
  gap: 8px;
  justify-content: space-between;

  .weekday-card {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 12px 4px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    background-color: var(--el-bg-color);
    transition: all 0.2s;
    min-width: 0;

    &:hover:not(.disabled) {
      border-color: var(--el-color-primary);
      box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
    }

    &.weekend:not(.disabled) {
      background-color: #fef0f0;
      border-color: #fde2e2;

      &:hover {
        border-color: #f56c6c;
        box-shadow: 0 2px 8px rgba(245, 108, 108, 0.1);
      }

      .weekday-name {
        color: #f56c6c;
      }
    }

    &.disabled {
      background-color: var(--el-fill-color-light);
      opacity: 0.6;
      cursor: not-allowed;

      .weekday-name,
      .date-display {
        color: var(--el-text-color-disabled);
      }
    }

    .weekday-name {
      font-size: 13px;
      font-weight: 500;
      color: var(--el-text-color-regular);
      margin-bottom: 4px;
    }

    .date-display {
      font-size: 12px;
      color: var(--el-text-color-secondary);
      margin-bottom: 8px;
    }

    .staff-input {
      width: 70px;

      :deep(.el-input__wrapper) {
        padding: 0 1.5rem 0 0;
      }

      :deep(.el-input__inner) {
        text-align: center;
        font-size: 16px;
        font-weight: 600;
      }

      :deep(.el-input-number__decrease),
      :deep(.el-input-number__increase) {
        width: 20px;
      }
    }

    .disabled-text {
      font-size: 11px;
      color: var(--el-text-color-disabled);
      text-align: center;
      padding: 8px 0;
    }

    .unit {
      margin-top: 4px;
      font-size: 11px;
      color: var(--el-text-color-secondary);
    }
  }
}

// 响应式
@media (max-width: 768px) {
  .weekday-cards {
    flex-wrap: wrap;

    .weekday-card {
      flex: 0 0 calc(25% - 6px);
      margin-bottom: 8px;
    }
  }
}
</style>
