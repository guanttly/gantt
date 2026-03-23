<script setup lang="ts">
import type { ChangeDetailPreview, ScheduleChangeType } from '@/types/schedule'
import { computed } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'

interface Props {
  /** 变更预览数据 */
  changePreview: ChangeDetailPreview
}

const props = defineProps<Props>()

// 计算统计信息
const stats = computed(() => {
  let addCount = 0
  let modifyCount = 0
  let removeCount = 0
  let totalStaffSlots = 0
  const affectedShifts = new Set<string>()
  const affectedDates = new Set<string>()

  // 【防御性检查】确保 shifts 存在且为数组
  if (!props.changePreview?.shifts || !Array.isArray(props.changePreview.shifts)) {
    return {
      addCount: 0,
      modifyCount: 0,
      removeCount: 0,
      affectedShiftsCount: 0,
      affectedDatesCount: 0,
      totalStaffSlots: 0,
    }
  }

  // 遍历班次数组
  props.changePreview.shifts.forEach((shift) => {
    if (!shift) return
    affectedShifts.add(shift.shiftId)
    // 遍历该班次下的所有日期变更
    if (shift.changes && Array.isArray(shift.changes)) {
      shift.changes.forEach((change) => {
        if (!change) return
        affectedDates.add(change.date)
        switch (change.changeType) {
          case 'add':
            addCount++
            break
          case 'modify':
            modifyCount++
            break
          case 'remove':
            removeCount++
            break
        }
        totalStaffSlots += change.afterIds?.length || 0
      })
    }
  })

  return {
    addCount,
    modifyCount,
    removeCount,
    affectedShiftsCount: affectedShifts.size,
    affectedDatesCount: affectedDates.size,
    totalStaffSlots,
  }
})

// 安全的 shifts 数组（防止 null/undefined）
const safeShifts = computed(() => {
  return props.changePreview?.shifts || []
})

// 变更类型样式映射
function changeTypeClass(type: ScheduleChangeType): string {
  switch (type) {
    case 'add':
      return 'change-type-add'
    case 'modify':
      return 'change-type-modify'
    case 'remove':
      return 'change-type-remove'
    default:
      return ''
  }
}

// 变更类型图标
function changeTypeIcon(type: ScheduleChangeType): string {
  switch (type) {
    case 'add':
      return 'add-new'
    case 'modify':
      return 'pencil'
    case 'remove':
      return 'trash'
    default:
      return ''
  }
}

// 变更类型标签
function changeTypeLabel(type: ScheduleChangeType): string {
  switch (type) {
    case 'add':
      return '新增'
    case 'modify':
      return '修改'
    case 'remove':
      return '删除'
    default:
      return ''
  }
}
</script>

<template>
  <div class="change-detail-view">
    <!-- 头部：任务信息和统计 -->
    <div class="header">
      <div class="task-info">
        <h3>{{ changePreview.taskTitle }} - 变更详情</h3>
        <div class="task-meta">
          <span class="task-index">任务 #{{ changePreview.taskIndex }}</span>
          <span class="timestamp">{{ changePreview.timestamp }}</span>
        </div>
      </div>
      <div class="stats">
        <div v-if="stats.addCount > 0" class="stat-item add">
          <span class="icon"><SvgIcon name="add-new" size="1em" /></span>
          <span class="label">新增</span>
          <span class="value">{{ stats.addCount }}</span>
        </div>
        <div v-if="stats.modifyCount > 0" class="stat-item modify">
          <span class="icon"><SvgIcon name="pencil" size="1em" /></span>
          <span class="label">修改</span>
          <span class="value">{{ stats.modifyCount }}</span>
        </div>
        <div v-if="stats.removeCount > 0" class="stat-item remove">
          <span class="icon"><SvgIcon name="trash" size="1em" /></span>
          <span class="label">删除</span>
          <span class="value">{{ stats.removeCount }}</span>
        </div>
        <div class="stat-item info">
          <span class="icon"><SvgIcon name="bar-chart" size="1em" /></span>
          <span class="label">涉及</span>
          <span class="value">{{ stats.affectedShiftsCount }} 班次 / {{ stats.affectedDatesCount }} 天</span>
        </div>
        <div class="stat-item info">
          <span class="icon"><SvgIcon name="users" size="1em" /></span>
          <span class="label">总人次</span>
          <span class="value">{{ stats.totalStaffSlots }}</span>
        </div>
      </div>
    </div>

    <!-- 变更列表：按班次分组 -->
    <div class="changes-container">
      <div
        v-for="shift in safeShifts"
        :key="shift.shiftId"
        class="shift-group"
      >
        <!-- 班次标题 -->
        <div class="shift-header">
          <span class="shift-name">{{ shift.shiftName || shift.shiftId }}</span>
          <span class="change-count">{{ shift.changes.length }} 条变更</span>
        </div>

        <!-- 该班次的所有变更（按日期排序） -->
        <div class="change-list">
          <div
            v-for="change in shift.changes"
            :key="change.date"
            class="change-item"
            :class="changeTypeClass(change.changeType)"
          >
            <!-- 变更标题行 -->
            <div class="change-header">
              <span class="change-icon"><SvgIcon v-if="changeTypeIcon(change.changeType)" :name="changeTypeIcon(change.changeType)" size="1em" /></span>
              <span class="change-date">{{ change.date }}</span>
              <span class="change-type-label">{{ changeTypeLabel(change.changeType) }}</span>
            </div>

            <!-- 变更内容 -->
            <div class="change-content">
              <!-- 变更前 -->
              <div v-if="change.before && change.before.length > 0" class="before-section">
                <span class="section-label">变更前：</span>
                <span class="staff-names">{{ change.before.join(', ') }}</span>
                <span class="staff-count">({{ change.before.length }} 人)</span>
              </div>

              <!-- 箭头指示器（仅修改类型） -->
              <div v-if="change.changeType === 'modify'" class="arrow-indicator">
                <el-icon><ArrowRight /></el-icon>
              </div>

              <!-- 变更后 -->
              <div class="after-section">
                <span class="section-label">
                  {{
                    change.changeType === 'add'
                      ? '新增：'
                      : change.changeType === 'remove'
                        ? '删除：'
                        : '变更后：'
                  }}
                </span>
                <span class="staff-names">{{
                  change.after && change.after.length > 0 ? change.after.join(', ') : '（无）'
                }}</span>
                <span v-if="change.after && change.after.length > 0" class="staff-count">({{ change.after.length }} 人)</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="safeShifts.length === 0" class="empty-state">
      <p>暂无变更数据</p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.change-detail-view {
  padding: 16px;
  max-height: 600px;
  overflow-y: auto;
}

.header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-light);

  .task-info {
    margin-bottom: 12px;

    h3 {
      margin: 0 0 8px;
      font-size: 18px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .task-meta {
      display: flex;
      gap: 16px;
      font-size: 13px;
      color: var(--el-text-color-secondary);

      .task-index {
        font-weight: 500;
      }
    }
  }

  .stats {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;

    .stat-item {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 6px 12px;
      border-radius: 4px;
      font-size: 13px;

      .icon {
        font-size: 16px;
      }

      .label {
        color: var(--el-text-color-secondary);
      }

      .value {
        font-weight: 600;
      }

      &.add {
        background-color: #f0f9ff;
        color: #0284c7;
      }

      &.modify {
        background-color: #fff7ed;
        color: #ea580c;
      }

      &.remove {
        background-color: #fef2f2;
        color: #dc2626;
      }

      &.info {
        background-color: #f9fafb;
        color: var(--el-text-color-regular);
      }
    }
  }
}

.changes-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.shift-group {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;

  .shift-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    background-color: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);

    .shift-name {
      font-weight: 600;
      font-size: 15px;
      color: var(--el-text-color-primary);
    }

    .change-count {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .change-list {
    display: flex;
    flex-direction: column;
  }
}

.change-item {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }

  .change-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .change-icon {
      font-size: 16px;
    }

    .change-date {
      font-weight: 500;
      color: var(--el-text-color-primary);
    }

    .change-type-label {
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 12px;
      font-weight: 500;
    }
  }

  .change-content {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding-left: 24px;

    .before-section,
    .after-section {
      display: flex;
      align-items: baseline;
      gap: 6px;
      font-size: 14px;

      .section-label {
        font-weight: 500;
        color: var(--el-text-color-secondary);
        min-width: 60px;
      }

      .staff-names {
        color: var(--el-text-color-primary);
        flex: 1;
      }

      .staff-count {
        color: var(--el-text-color-placeholder);
        font-size: 13px;
      }
    }

    .arrow-indicator {
      display: flex;
      align-items: center;
      margin-left: 60px;
      color: var(--el-text-color-placeholder);
    }
  }

  // 变更类型样式
  &.change-type-add {
    background-color: #f0f9ff;
    border-left: 3px solid #0284c7;

    .change-type-label {
      background-color: #0284c7;
      color: white;
    }
  }

  &.change-type-modify {
    background-color: #fff7ed;
    border-left: 3px solid #ea580c;

    .change-type-label {
      background-color: #ea580c;
      color: white;
    }
  }

  &.change-type-remove {
    background-color: #fef2f2;
    border-left: 3px solid #dc2626;

    .change-type-label {
      background-color: #dc2626;
      color: white;
    }
  }
}

.empty-state {
  text-align: center;
  padding: 48px 0;
  color: var(--el-text-color-placeholder);
}
</style>
