<script setup lang="ts">
import type { ChangeDetailPreview, ScheduleChangeType } from '@/types/scheduling'
import { CirclePlus, Edit, Remove } from '@element-plus/icons-vue'
import { ElEmpty, ElTag } from 'element-plus'

import { computed } from 'vue'

interface FlatChange {
  shiftName: string
  date: string
  changeType: ScheduleChangeType
  before: string[]
  after: string[]
}

const props = defineProps<{
  changePreview: ChangeDetailPreview
}>()

const flatChanges = computed(() => {
  const result: FlatChange[] = []
  if (!props.changePreview?.shifts)
    return result
  for (const shift of props.changePreview.shifts) {
    for (const change of shift.changes) {
      result.push({
        shiftName: shift.shift_name,
        date: change.date,
        changeType: change.change_type,
        before: change.before,
        after: change.after,
      })
    }
  }
  return result
})

const groupedChanges = computed(() => {
  const groups: Record<string, FlatChange[]> = {
    add: [],
    modify: [],
    remove: [],
  }
  for (const change of flatChanges.value) {
    const type = change.changeType || 'modify'
    if (groups[type])
      groups[type].push(change)
    else
      groups.modify.push(change)
  }
  return groups
})

const addedCount = computed(() => groupedChanges.value.add.length)
const modifiedCount = computed(() => groupedChanges.value.modify.length)
const removedCount = computed(() => groupedChanges.value.remove.length)

function getChangeIcon(type: string) {
  switch (type) {
    case 'add':
      return CirclePlus
    case 'remove':
      return Remove
    default:
      return Edit
  }
}

function getChangeLabel(type: string): string {
  switch (type) {
    case 'add':
      return '新增'
    case 'remove':
      return '移除'
    default:
      return '修改'
  }
}

function getChangeTagType(type: string): 'success' | 'danger' | 'warning' {
  switch (type) {
    case 'add':
      return 'success'
    case 'remove':
      return 'danger'
    default:
      return 'warning'
  }
}
</script>

<template>
  <div class="change-detail-view">
    <div v-if="changePreview.task_title" class="change-title">
      <span class="title-text">{{ changePreview.task_title }}</span>
      <ElTag effect="plain" size="small" type="info">
        任务 #{{ changePreview.task_index }}
      </ElTag>
    </div>

    <div class="change-summary">
      <ElTag v-if="addedCount > 0" effect="plain" type="success">
        新增 {{ addedCount }}
      </ElTag>
      <ElTag v-if="modifiedCount > 0" effect="plain" type="warning">
        修改 {{ modifiedCount }}
      </ElTag>
      <ElTag v-if="removedCount > 0" effect="plain" type="danger">
        移除 {{ removedCount }}
      </ElTag>
    </div>

    <div v-if="flatChanges.length > 0" class="changes-list">
      <template v-for="(items, type) in groupedChanges" :key="type">
        <div v-if="items.length > 0" class="change-group">
          <div class="group-title">
            <el-icon :size="16">
              <component :is="getChangeIcon(type as string)" />
            </el-icon>
            <span>{{ getChangeLabel(type as string) }} ({{ items.length }})</span>
          </div>
          <div class="group-items">
            <div
              v-for="(change, idx) in items"
              :key="`${type}-${idx}`"
              class="change-item"
              :class="`change-${type}`"
            >
              <div class="item-header">
                <span class="item-shift">{{ change.shiftName }}</span>
                <ElTag :type="getChangeTagType(type as string)" effect="plain" size="small">
                  {{ getChangeLabel(type as string) }}
                </ElTag>
              </div>
              <div class="item-detail">
                <span class="detail-date">{{ change.date }}</span>
              </div>
              <div v-if="change.before.length > 0 || change.after.length > 0" class="item-diff">
                <span v-if="change.before.length > 0" class="diff-old">{{ change.before.join(', ') }}</span>
                <span v-if="change.before.length > 0 && change.after.length > 0" class="diff-arrow">→</span>
                <span v-if="change.after.length > 0" class="diff-new">{{ change.after.join(', ') }}</span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
    <ElEmpty v-else :image-size="60" description="暂无变更信息" />
  </div>
</template>

<style lang="scss" scoped>
.change-detail-view {
  padding: 8px 0;
}

.change-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.title-text {
  font-size: 15px;
  font-weight: 600;
}

.change-summary {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.changes-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.group-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.change-item {
  padding: 12px;
  border-radius: 6px;
  border: 1px solid var(--el-border-color-lighter);
}

.change-add {
  border-left: 3px solid #67c23a;
  background: rgba(103, 194, 58, 0.04);
}

.change-modify {
  border-left: 3px solid #e6a23c;
  background: rgba(230, 162, 60, 0.04);
}

.change-remove {
  border-left: 3px solid #f56c6c;
  background: rgba(245, 108, 108, 0.04);
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.item-shift {
  font-size: 14px;
  font-weight: 600;
}

.item-detail {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.item-diff {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  padding: 6px 10px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.diff-old {
  color: #f56c6c;
  text-decoration: line-through;
}

.diff-arrow {
  color: var(--el-text-color-placeholder);
}

.diff-new {
  color: #67c23a;
  font-weight: 500;
}
</style>
