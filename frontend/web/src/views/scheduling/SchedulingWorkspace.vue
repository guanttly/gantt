<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import ChatAssistant from './components/ChatAssistant/index.vue'
import SchedulingGantt from './components/SchedulingGantt.vue'
import SchedulingProgressBar from './components/SchedulingProgressBar.vue'
import SchedulingToolbar from './components/SchedulingToolbar.vue'

// 侧边栏状态
const showChatPanel = ref(false)
const chatAssistantRef = ref()
const ganttRef = ref()

// 日期范围 — 默认本周
function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const dayOfWeek = today.getDay()
const daysFromMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
const startOfWeek = new Date(today)
startOfWeek.setDate(today.getDate() + daysFromMonday)
const endOfWeek = new Date(startOfWeek)
endOfWeek.setDate(startOfWeek.getDate() + 6)

const dateRange = ref<[string, string]>([
  formatDateLocal(startOfWeek),
  formatDateLocal(endOfWeek),
])

function handleRefresh() {
  ganttRef.value?.refresh()
}

function toggleChatPanel() {
  showChatPanel.value = !showChatPanel.value
}

function handleQuickStart() {
  if (!showChatPanel.value) {
    showChatPanel.value = true
  }

  nextTick(() => {
    if (chatAssistantRef.value?.startScheduleCreationWorkflow) {
      const daysUntilNextMonday = dayOfWeek === 0 ? 1 : (8 - dayOfWeek)
      const nextMonday = new Date(today)
      nextMonday.setDate(today.getDate() + daysUntilNextMonday)
      const nextSunday = new Date(nextMonday)
      nextSunday.setDate(nextMonday.getDate() + 6)

      chatAssistantRef.value.startScheduleCreationWorkflow({
        startDate: formatDateLocal(nextMonday),
        endDate: formatDateLocal(nextSunday),
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
        <SchedulingProgressBar />
        <SchedulingToolbar
          v-model:date-range="dateRange"
          @refresh="handleRefresh"
          @toggle-chat="toggleChatPanel"
          @quick-start="handleQuickStart"
        />
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

.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

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
