<script setup lang="ts">
import { Refresh } from '@element-plus/icons-vue'

const emit = defineEmits<{
  refresh: []
  toggleChat: []
  quickStart: []
}>()

const dateRange = defineModel<[string, string]>('dateRange', { required: true })

function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function handleThisWeek() {
  const today = new Date()
  const dayOfWeek = today.getDay()
  const daysFromMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
  const startOfWeek = new Date(today)
  startOfWeek.setDate(today.getDate() + daysFromMonday)
  const endOfWeek = new Date(startOfWeek)
  endOfWeek.setDate(startOfWeek.getDate() + 6)
  dateRange.value = [formatDateLocal(startOfWeek), formatDateLocal(endOfWeek)]
}

function handleNextWeek() {
  const today = new Date()
  const dayOfWeek = today.getDay()
  const daysUntilNextMonday = dayOfWeek === 0 ? 1 : (8 - dayOfWeek)
  const nextMonday = new Date(today)
  nextMonday.setDate(today.getDate() + daysUntilNextMonday)
  const nextSunday = new Date(nextMonday)
  nextSunday.setDate(nextMonday.getDate() + 6)
  dateRange.value = [formatDateLocal(nextMonday), formatDateLocal(nextSunday)]
}
</script>

<template>
  <div class="scheduling-toolbar">
    <div class="toolbar-left">
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        value-format="YYYY-MM-DD"
        size="default"
        style="width: 280px"
      />
      <el-button-group>
        <el-button size="default" @click="handleThisWeek">
          本周
        </el-button>
        <el-button size="default" @click="handleNextWeek">
          下周
        </el-button>
      </el-button-group>
      <el-button :icon="Refresh" size="default" @click="emit('refresh')">
        刷新
      </el-button>
    </div>
    <div class="toolbar-right">
      <el-button type="primary" size="default" @click="emit('quickStart')">
        一键排班
      </el-button>
      <el-button size="default" @click="emit('toggleChat')">
        排班助手
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.scheduling-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
