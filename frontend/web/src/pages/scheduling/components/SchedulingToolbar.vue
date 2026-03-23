<script setup lang="ts">
import { ChatDotRound, MagicStick, Refresh } from '@element-plus/icons-vue'
import { computed } from 'vue'

const props = defineProps<{
  dateRange: [Date | string, Date | string]
}>()

const emit = defineEmits<{
  'refresh': []
  'toggleChat': []
  'dateRangeChange': [range: [Date | string, Date | string]]
  'update:dateRange': [range: [Date | string, Date | string]]
  'quickStart': [] // 一键启动排班
}>()

// 日期范围选择
const localDateRange = computed({
  get: () => props.dateRange,
  set: (value) => {
    emit('update:dateRange', value)
    emit('dateRangeChange', value)
  },
})

// 快捷选项
const shortcuts = [
  {
    text: '本周',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setDate(start.getDate() - start.getDay())
      return [start, end]
    },
  },
  {
    text: '本月',
    value: () => {
      const start = new Date()
      start.setDate(1)
      start.setHours(0, 0, 0, 0)
      const end = new Date(start.getFullYear(), start.getMonth() + 1, 0)
      end.setHours(23, 59, 59, 999)
      return [start, end]
    },
  },
  {
    text: '未来7天',
    value: () => {
      const start = new Date()
      const end = new Date()
      end.setDate(end.getDate() + 6)
      return [start, end]
    },
  },
  {
    text: '未来30天',
    value: () => {
      const start = new Date()
      const end = new Date()
      end.setDate(end.getDate() + 29)
      return [start, end]
    },
  },
]

// 禁用超过30天的日期范围
function disabledDate(time: Date) {
  if (!localDateRange.value || !localDateRange.value[0])
    return false

  // 确保 localDateRange.value[0] 是 Date 对象
  const startDate = localDateRange.value[0] instanceof Date
    ? localDateRange.value[0]
    : new Date(localDateRange.value[0])

  const start = startDate.getTime()
  const maxDays = 30 * 24 * 60 * 60 * 1000
  return Math.abs(time.getTime() - start) > maxDays
}// 刷新
function handleRefresh() {
  emit('refresh')
}

// 切换聊天助手
function handleToggleChat() {
  emit('toggleChat')
}

// 一键启动排班
function handleQuickStart() {
  emit('quickStart')
}
</script>

<template>
  <div class="scheduling-toolbar">
    <div class="toolbar-left">
      <el-date-picker
        v-model="localDateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        :shortcuts="shortcuts"
        :disabled-date="disabledDate"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        :clearable="false"
        style="width: 320px"
      />

      <el-button :icon="Refresh" @click="handleRefresh">
        刷新
      </el-button>
    </div>

    <div class="toolbar-right">
      <el-button :icon="MagicStick" type="success" @click="handleQuickStart">
        一键启动排班
      </el-button>

      <el-button :icon="ChatDotRound" type="primary" @click="handleToggleChat">
        智能排班助手
      </el-button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.scheduling-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}
</style>
