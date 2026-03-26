<script setup lang="ts">
import { Calendar, User, UserFilled } from '@element-plus/icons-vue'
import { ElButton, ElCollapse, ElCollapseItem, ElDialog, ElEmpty, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface StaffMember {
  id: string
  name: string
  departmentId?: string
  position?: string
}

interface ShiftStaffInfo {
  shiftId: string
  shiftName: string
  startTime: string
  endTime: string
  staffCount: number
  staffList: StaffMember[]
}

interface StaffDetailsData {
  totalStaff: number
  totalTeams: number
  shifts: ShiftStaffInfo[]
}

const props = defineProps<{
  visible: boolean
  data: StaffDetailsData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const activeNames = ref<string[]>(['0'])

const stats = computed(() => {
  if (!props.data)
    return { totalStaff: 0, totalTeams: 0, totalShifts: 0 }
  return {
    totalStaff: props.data.totalStaff,
    totalTeams: props.data.totalTeams,
    totalShifts: props.data.shifts.length,
  }
})

function getShiftTypeInfo(startTime: string) {
  const hour = Number.parseInt(startTime.split(':')[0])
  if (hour >= 6 && hour < 12)
    return { type: '早班', color: '#67C23A', bgColor: '#f0f9ff' }
  else if (hour >= 12 && hour < 18)
    return { type: '中班', color: '#E6A23C', bgColor: '#fef0e6' }
  else
    return { type: '晚班', color: '#909399', bgColor: '#f4f4f5' }
}

watch(() => props.visible, (newVal) => {
  if (newVal)
    activeNames.value = ['0']
})

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :before-close="handleClose"
    :model-value="visible"
    title="人员详情"
    width="700px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="staff-details-content">
      <div class="stats-overview">
        <div class="stat-card">
          <div class="stat-icon" style="background: #ecf5ff; color: #409eff;">
            <el-icon :size="24">
              <UserFilled />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.totalStaff }}
            </div>
            <div class="stat-label">
              可用人员
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #f0f9ff; color: #67c23a;">
            <el-icon :size="24">
              <User />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.totalTeams }}
            </div>
            <div class="stat-label">
              分组数量
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #fef0e6; color: #e6a23c;">
            <el-icon :size="24">
              <Calendar />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.totalShifts }}
            </div>
            <div class="stat-label">
              班次数量
            </div>
          </div>
        </div>
      </div>

      <div class="shifts-section">
        <div class="section-title">
          各班次人员分配
        </div>
        <ElCollapse v-model="activeNames" class="shift-collapse">
          <ElCollapseItem
            v-for="(shift, index) in data.shifts"
            :key="shift.shiftId"
            :name="String(index)"
            class="shift-collapse-item"
          >
            <template #title>
              <div class="shift-header">
                <div class="shift-info">
                  <div
                    class="shift-type-badge"
                    :style="{
                      background: getShiftTypeInfo(shift.startTime).bgColor,
                      color: getShiftTypeInfo(shift.startTime).color,
                      borderColor: getShiftTypeInfo(shift.startTime).color,
                    }"
                  >
                    {{ getShiftTypeInfo(shift.startTime).type }}
                  </div>
                  <span class="shift-name">{{ shift.shiftName }}</span>
                  <span class="shift-time">{{ shift.startTime }} - {{ shift.endTime }}</span>
                </div>
                <ElTag :type="shift.staffCount > 0 ? 'success' : 'info'" effect="plain">
                  {{ shift.staffCount }} 人
                </ElTag>
              </div>
            </template>

            <div v-if="shift.staffList && shift.staffList.length > 0" class="staff-list-container">
              <div
                v-for="staff in shift.staffList"
                :key="staff.id"
                class="staff-item"
              >
                <div class="staff-avatar">
                  <el-icon>
                    <User />
                  </el-icon>
                </div>
                <div class="staff-info-detail">
                  <div class="staff-name">
                    {{ staff.name }}
                  </div>
                  <div v-if="staff.position" class="staff-position">
                    {{ staff.position }}
                  </div>
                </div>
              </div>
            </div>
            <ElEmpty v-else :image-size="80" description="该班次暂无可用人员" />
          </ElCollapseItem>
        </ElCollapse>
      </div>
    </div>

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.staff-details-content { padding: 8px 0; }

.stats-overview {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);
  }
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.shifts-section {
  margin-top: 24px;
  max-height: 500px;
  overflow-y: auto;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid var(--el-border-color-lighter);
}

.shift-collapse {
  border: none;

  :deep(.el-collapse-item) {
    margin-bottom: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    overflow: hidden;
  }

  :deep(.el-collapse-item__header) {
    height: auto;
    padding: 12px 16px;
    border: none;
  }

  :deep(.el-collapse-item__content) {
    padding: 16px;
  }
}

.shift-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 12px;
}

.shift-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.shift-type-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid;
}

.shift-name {
  font-size: 15px;
  font-weight: 600;
}

.shift-time {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.staff-list-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.staff-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  transition: all 0.2s;

  &:hover {
    border-color: var(--el-color-primary-light-5);
  }
}

.staff-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.staff-name {
  font-size: 14px;
  font-weight: 500;
}

.staff-position {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
