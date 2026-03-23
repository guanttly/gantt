<script setup lang="ts">
import { ArrowDown, ArrowUp, Check, Close, Loading, Operation, Warning } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { useSchedulingSessionStore } from '@/store/schedulingSession'

const store = useSchedulingSessionStore()

// 计算班次进度百分比
const shiftProgressPercent = computed(() => {
  if (!store.shiftProgress)
    return 0
  return Math.round((store.shiftProgress.current / store.shiftProgress.total) * 100)
})

// 计算天进度百分比
const dayProgressPercent = computed(() => {
  if (!store.shiftProgress?.totalDays)
    return 0
  return Math.round(((store.shiftProgress.currentDay || 0) / store.shiftProgress.totalDays) * 100)
})

// 获取进度状态类型
const progressStatus = computed(() => {
  const status = store.shiftProgress?.status
  if (status === 'shift_success' || status === 'day_completed')
    return 'success'
  if (status === 'shift_failed')
    return 'exception'
  if (status === 'shift_retrying')
    return 'warning'
  return undefined
})

// 格式化日期显示
function formatDate(date: string | undefined): string {
  if (!date)
    return ''
  // 只显示月-日
  const parts = date.split('-')
  if (parts.length === 3) {
    return `${parts[1]}-${parts[2]}`
  }
  return date
}

// 获取日期状态
function getDateStatus(date: string): 'completed' | 'current' | 'pending' {
  if (!store.shiftProgress)
    return 'pending'
  if (store.shiftProgress.completedDates?.includes(date))
    return 'completed'
  if (store.shiftProgress.currentDate === date)
    return 'current'
  return 'pending'
}

// 解析草案获取日期列表
const draftDates = computed(() => {
  if (!store.realtimeDraft?.schedule)
    return []
  return Object.keys(store.realtimeDraft.schedule).sort()
})

// 获取日期排班人数
function getDateStaffCount(date: string): number {
  if (!store.realtimeDraft?.schedule?.[date])
    return 0
  return store.realtimeDraft.schedule[date].length
}

// 折叠状态
const collapsed = computed(() => !store.showProgressBar)

// 手动折叠
function toggleCollapse() {
  store.showProgressBar = !store.showProgressBar
}
</script>

<template>
  <Transition name="slide-down">
    <div v-if="store.shiftProgress" class="scheduling-progress-bar" :class="{ collapsed }">
      <!-- 折叠状态下的简洁视图 -->
      <div v-if="collapsed" class="progress-collapsed" @click="toggleCollapse">
        <el-icon class="expand-icon">
          <ArrowDown />
        </el-icon>
        <span class="collapsed-text">
          {{ store.shiftProgress.shiftName }}
          ({{ store.shiftProgress.current }}/{{ store.shiftProgress.total }})
        </span>
      </div>

      <!-- 展开状态下的完整视图 -->
      <div v-else class="progress-expanded">
        <!-- 顶部标题栏 -->
        <div class="progress-header">
          <div class="header-left">
            <el-icon class="status-icon" :class="store.shiftProgress.status">
              <Loading v-if="store.shiftProgress.status === 'day_generating'" />
              <Check v-else-if="store.shiftProgress.status === 'day_completed' || store.shiftProgress.status === 'shift_success'" />
              <Warning v-else-if="store.shiftProgress.status === 'shift_retrying'" />
              <Close v-else-if="store.shiftProgress.status === 'shift_failed'" />
              <Operation v-else />
            </el-icon>
            <span class="shift-name">{{ store.shiftProgress.shiftName }}</span>
            <el-tag size="small" type="info">
              班次 {{ store.shiftProgress.current }}/{{ store.shiftProgress.total }}
            </el-tag>
          </div>
          <div class="header-right">
            <el-button text size="small" @click="toggleCollapse">
              <el-icon><ArrowUp /></el-icon>
            </el-button>
          </div>
        </div>

        <!-- 进度条区域 -->
        <div class="progress-bars">
          <!-- 班次进度 -->
          <div class="progress-item">
            <span class="progress-label">班次进度</span>
            <el-progress
              :percentage="shiftProgressPercent"
              :status="progressStatus"
              :stroke-width="8"
              style="flex: 1"
            />
          </div>

          <!-- 天进度（如果有） -->
          <div v-if="store.shiftProgress.totalDays" class="progress-item">
            <span class="progress-label">
              {{ formatDate(store.shiftProgress.currentDate) }}
              ({{ store.shiftProgress.currentDay }}/{{ store.shiftProgress.totalDays }}天)
            </span>
            <el-progress
              :percentage="dayProgressPercent"
              :status="progressStatus"
              :stroke-width="8"
              style="flex: 1"
            />
          </div>
        </div>

        <!-- 状态消息 -->
        <div class="progress-message">
          {{ store.shiftProgress.message }}
        </div>

        <!-- 实时草案预览（紧凑表格） -->
        <div v-if="draftDates.length > 0" class="draft-preview">
          <div class="preview-title">
            当前草案预览
          </div>
          <div class="preview-dates">
            <div
              v-for="date in draftDates"
              :key="date"
              class="date-item"
              :class="getDateStatus(date)"
            >
              <span class="date-label">{{ formatDate(date) }}</span>
              <span class="date-count">{{ getDateStaffCount(date) }}人</span>
              <el-icon v-if="getDateStatus(date) === 'completed'" class="status-check">
                <Check />
              </el-icon>
              <el-icon v-else-if="getDateStatus(date) === 'current'" class="status-loading">
                <Loading />
              </el-icon>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style lang="scss" scoped>
.scheduling-progress-bar {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 12px 16px;
  border-radius: 0 0 8px 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  margin-bottom: 8px;

  &.collapsed {
    padding: 8px 16px;
    cursor: pointer;

    &:hover {
      background: linear-gradient(135deg, #5a6fd6 0%, #6a4190 100%);
    }
  }
}

.progress-collapsed {
  display: flex;
  align-items: center;
  gap: 8px;

  .expand-icon {
    font-size: 14px;
  }

  .collapsed-text {
    font-size: 13px;
  }
}

.progress-expanded {
  .progress-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    .header-left {
      display: flex;
      align-items: center;
      gap: 8px;

      .status-icon {
        font-size: 18px;

        &.day_generating {
          animation: spin 1s linear infinite;
        }
        &.day_completed, &.shift_success {
          color: #67c23a;
        }
        &.shift_retrying {
          color: #e6a23c;
        }
        &.shift_failed {
          color: #f56c6c;
        }
      }

      .shift-name {
        font-weight: 600;
        font-size: 14px;
      }
    }

    .header-right {
      .el-button {
        color: white;
      }
    }
  }

  .progress-bars {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 8px;

    .progress-item {
      display: flex;
      align-items: center;
      gap: 12px;

      .progress-label {
        font-size: 12px;
        min-width: 100px;
        opacity: 0.9;
      }
    }

    :deep(.el-progress-bar__outer) {
      background-color: rgba(255, 255, 255, 0.3);
    }
    :deep(.el-progress-bar__inner) {
      background: white;
    }
    :deep(.el-progress__text) {
      color: white;
    }
  }

  .progress-message {
    font-size: 12px;
    opacity: 0.9;
    margin-bottom: 12px;
    padding: 6px 10px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 4px;
  }

  .draft-preview {
    .preview-title {
      font-size: 12px;
      opacity: 0.8;
      margin-bottom: 8px;
    }

    .preview-dates {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;

      .date-item {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 4px 8px;
        background: rgba(255, 255, 255, 0.15);
        border-radius: 4px;
        font-size: 12px;

        &.completed {
          background: rgba(103, 194, 58, 0.3);
        }
        &.current {
          background: rgba(64, 158, 255, 0.4);
          animation: pulse 1.5s ease-in-out infinite;
        }
        &.pending {
          opacity: 0.6;
        }

        .date-label {
          font-weight: 500;
        }
        .date-count {
          opacity: 0.9;
        }
        .status-check {
          color: #67c23a;
        }
        .status-loading {
          animation: spin 1s linear infinite;
        }
      }
    }
  }
}

// 动画
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

// 滑入动画
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}
.slide-down-enter-from,
.slide-down-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}
</style>
