<script setup lang="ts">
import { ElDialog } from 'element-plus'
import { ref } from 'vue'

interface Props {
  visible: boolean
  employeeName: string
  date: string
  shifts: Shift.ShiftInfo[]
}

defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [shiftId: string]
  'cancel': []
}>()

const selectedShiftId = ref<string>('')

function handleShiftClick(shiftId: string) {
  selectedShiftId.value = shiftId
}

function handleConfirm() {
  if (!selectedShiftId.value) {
    return
  }
  emit('confirm', selectedShiftId.value)
  emit('update:visible', false)
  selectedShiftId.value = ''
}

function handleCancel() {
  emit('cancel')
  emit('update:visible', false)
  selectedShiftId.value = ''
}

function handleClose() {
  handleCancel()
}
</script>

<template>
  <ElDialog
    :model-value="visible"
    :title="`为 ${employeeName} 添加排班`"
    width="500px"
    @close="handleClose"
  >
    <div class="shift-select-dialog">
      <div class="date-info">
        日期: {{ date }}
      </div>
      <div class="shift-list">
        <div
          v-for="shift in shifts"
          :key="shift.id"
          class="shift-item"
          :class="{ selected: selectedShiftId === shift.id }"
          @click="handleShiftClick(shift.id)"
        >
          <div class="shift-name" :style="{ color: shift.color || '#409eff' }">
            {{ shift.name }}
          </div>
          <div class="shift-info">
            {{ shift.startTime }} - {{ shift.endTime }}
            <span v-if="shift.weeklyStaffSummary"> | {{ shift.weeklyStaffSummary }}</span>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="handleCancel">
        取消
      </el-button>
      <el-button type="primary" :disabled="!selectedShiftId" @click="handleConfirm">
        确定
      </el-button>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.shift-select-dialog {
  .date-info {
    margin-bottom: 16px;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }

  .shift-list {
    max-height: 400px;
    overflow-y: auto;
  }

  .shift-item {
    padding: 12px;
    margin-bottom: 8px;
    border: 2px solid transparent;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
    background-color: #f5f7fa;

    &:hover {
      background-color: #e6f7ff;
    }

    &.selected {
      background-color: #e6f7ff;
      border-color: #409eff;
    }

    .shift-name {
      font-weight: 500;
      font-size: 15px;
      margin-bottom: 6px;
    }

    .shift-info {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }
}
</style>
