<script setup lang="ts">
import { nextTick, onMounted, ref, watchEffect } from 'vue'
import ChatAssistant from './components/ChatAssistant/index.vue'
import SchedulingGantt from './components/SchedulingGantt.vue'
import SchedulingToolbar from './components/SchedulingToolbar.vue'
import SchedulingProgressBar from './components/SchedulingProgressBar.vue'

// 侧边栏状态
const showChatPanel = ref(false)

// ChatAssistant 组件引用
const chatAssistantRef = ref()

// 当前选择的日期范围 - 默认本周一到周日（用于查看当前排班）
const today = new Date()
const dayOfWeek = today.getDay() // 0(周日) 到 6(周六)

// 计算本周一（如果今天是周日，则周一是明天）
const daysFromMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
const startOfWeek = new Date(today)
startOfWeek.setDate(today.getDate() + daysFromMonday)
startOfWeek.setHours(0, 0, 0, 0)

// 本周日
const endOfWeek = new Date(startOfWeek)
endOfWeek.setDate(startOfWeek.getDate() + 6)
endOfWeek.setHours(23, 59, 59, 999)

// 使用字符串格式以匹配 el-date-picker 的 value-format="YYYY-MM-DD"
// 注意：toISOString() 返回 UTC 时间，可能与本地时间相差一天
// 使用本地时间格式化函数
function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const dateRange = ref<[string, string]>([
  formatDateLocal(startOfWeek),
  formatDateLocal(endOfWeek),
])

console.log('[DateInit] dateRange:', dateRange.value)

watchEffect(() => {
  console.log('[Index] watchEffect - dateRange.value:', dateRange.value)
})

// 刷新甘特图
const ganttRef = ref()
function handleRefresh() {
  ganttRef.value?.refresh()
}

// 打开/关闭聊天助手
function toggleChatPanel() {
  showChatPanel.value = !showChatPanel.value
}

// 一键启动排班
function handleQuickStart() {
  // 确保聊天助手面板打开
  if (!showChatPanel.value) {
    showChatPanel.value = true
  }

  // 通过 ChatAssistant 组件引用触发工作流
  // 使用 nextTick 确保组件已渲染
  nextTick(() => {
    if (chatAssistantRef.value && chatAssistantRef.value.startScheduleCreationWorkflow) {
      // 计算下周一到周日（用于排班）
      const today = new Date()
      const dayOfWeek = today.getDay() // 0(周日) 到 6(周六)

      // 下周一距离今天的天数
      const daysUntilNextMonday = dayOfWeek === 0 ? 1 : (8 - dayOfWeek)

      const nextMonday = new Date(today)
      nextMonday.setDate(today.getDate() + daysUntilNextMonday)
      nextMonday.setHours(0, 0, 0, 0)

      const nextSunday = new Date(nextMonday)
      nextSunday.setDate(nextMonday.getDate() + 6)
      nextSunday.setHours(23, 59, 59, 999)

      const startDate = formatDateLocal(nextMonday)
      const endDate = formatDateLocal(nextSunday)

      console.log('[QuickStart] 下周排班:', { startDate, endDate })

      chatAssistantRef.value.startScheduleCreationWorkflow({
        startDate,
        endDate,
      })
    }
  })
}

onMounted(() => {
  // 页面初始化
})
</script>

<template>
  <div class="scheduling-page">
    <div class="scheduling-container">
      <!-- 主工作区 -->
      <div class="scheduling-main">
        <!-- 排班进度条（新增） -->
        <SchedulingProgressBar />
        
        <!-- 工具栏 -->
        <SchedulingToolbar
          v-model:date-range="dateRange"
          @refresh="handleRefresh"
          @toggle-chat="toggleChatPanel"
          @quick-start="handleQuickStart"
        />

        <!-- 甘特图区域 -->
        <SchedulingGantt
          ref="ganttRef"
          :date-range="dateRange"
        />
      </div>

      <!-- 智能排班助手侧边栏 -->
      <transition name="slide">
        <div v-show="showChatPanel" class="scheduling-aside">
          <ChatAssistant ref="chatAssistantRef" @close="toggleChatPanel" />
        </div>
      </transition>
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

.scheduling-aside {
  width: 550px;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border-left: 1px solid #e4e7ed;
  overflow: hidden;
}

// 滑动动画
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

// 响应式设计
@media (max-width: 1200px) {
  .scheduling-aside {
    width: 360px;
  }
}

@media (max-width: 768px) {
  .scheduling-container {
    flex-direction: column;
  }

  .scheduling-aside {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 100%;
    max-width: 400px;
    z-index: 100;
    box-shadow: -2px 0 8px rgba(0, 0, 0, 0.15);
  }
}
</style>
