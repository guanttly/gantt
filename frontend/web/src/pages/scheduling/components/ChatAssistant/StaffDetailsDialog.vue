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

interface Props {
  visible: boolean
  data: StaffDetailsData | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 默认展开第一个班次
const activeNames = ref<string[]>(['0'])

// 虚拟滚动相关 - 人员列表
const staffScrollTops = ref<Map<string, number>>(new Map())
const staffItemHeight = 68 // 每个人员卡片的高度
const visibleStaffCount = 6 // 可见人员数量
const bufferStaffCount = 2 // 缓冲区数量

// 计算统计信息
const stats = computed(() => {
  if (!props.data)
    return { totalStaff: 0, totalTeams: 0, totalShifts: 0 }

  return {
    totalStaff: props.data.totalStaff,
    totalTeams: props.data.totalTeams,
    totalShifts: props.data.shifts.length,
  }
})

// 为每个班次计算虚拟滚动数据
function getVisibleStaff(shift: ShiftStaffInfo) {
  const scrollTop = staffScrollTops.value.get(shift.shiftId) || 0
  const staffList = shift.staffList || []

  if (staffList.length <= visibleStaffCount) {
    // 人员少，不需要虚拟滚动
    return {
      visible: staffList.map((staff, index) => ({ ...staff, offset: index * staffItemHeight })),
      totalHeight: staffList.length * staffItemHeight,
      needScroll: false,
    }
  }

  const startIndex = Math.max(0, Math.floor(scrollTop / staffItemHeight) - bufferStaffCount)
  const endIndex = Math.min(
    staffList.length,
    Math.ceil((scrollTop + visibleStaffCount * staffItemHeight) / staffItemHeight) + bufferStaffCount,
  )

  return {
    visible: staffList.slice(startIndex, endIndex).map((staff, index) => ({
      ...staff,
      offset: (startIndex + index) * staffItemHeight,
    })),
    totalHeight: staffList.length * staffItemHeight,
    needScroll: true,
  }
}

// 处理人员列表滚动
function handleStaffScroll(shiftId: string, event: Event) {
  const target = event.target as HTMLElement
  staffScrollTops.value.set(shiftId, target.scrollTop)
}

// 监听对话框打开，重置滚动状态
watch(() => props.visible, (newVal) => {
  if (newVal) {
    staffScrollTops.value.clear()
    activeNames.value = ['0']
  }
})

// 根据班次时间判断类型和颜色
function getShiftTypeInfo(startTime: string) {
  const hour = Number.parseInt(startTime.split(':')[0])

  if (hour >= 6 && hour < 12) {
    return {
      type: '早班',
      color: '#67C23A',
      bgColor: '#f0f9ff',
    }
  }
  else if (hour >= 12 && hour < 18) {
    return {
      type: '中班',
      color: '#E6A23C',
      bgColor: '#fef0e6',
    }
  }
  else {
    return {
      type: '晚班',
      color: '#909399',
      bgColor: '#f4f4f5',
    }
  }
}

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :model-value="visible"
    title="人员详情"
    width="700px"
    :before-close="handleClose"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="staff-details-content">
      <!-- 统计概览 -->
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

      <!-- 班次列表 -->
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
                class="staff-scroll-wrapper"
                :style="{
                  maxHeight: getVisibleStaff(shift).needScroll
                    ? `${visibleStaffCount * staffItemHeight}px`
                    : 'none',
                }"
                @scroll="(e) => handleStaffScroll(shift.shiftId, e)"
              >
                <div
                  :style="{
                    height: `${getVisibleStaff(shift).totalHeight}px`,
                    position: 'relative',
                  }"
                >
                  <div
                    v-for="staff in getVisibleStaff(shift).visible"
                    :key="staff.id"
                    class="staff-item"
                    :style="{
                      position: 'absolute',
                      top: `${staff.offset}px`,
                      left: 0,
                      right: 0,
                    }"
                  >
                    <div class="staff-avatar">
                      <el-icon>
                        <User />
                      </el-icon>
                    </div>
                    <div class="staff-info">
                      <div class="staff-name">
                        {{ staff.name }}
                      </div>
                      <div v-if="staff.position" class="staff-position">
                        {{ staff.position }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <ElEmpty v-else description="该班次暂无可用人员" :image-size="80" />
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
.staff-details-content {
  padding: 8px 0;
}

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
    transform: translateY(-2px);
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

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  line-height: 1.2;
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

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--el-fill-color-lighter);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--el-border-color);
    border-radius: 3px;
    transition: background 0.3s;

    &:hover {
      background: var(--el-border-color-dark);
    }
  }
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
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

    &:last-child {
      margin-bottom: 0;
    }
  }

  :deep(.el-collapse-item__header) {
    height: auto;
    padding: 12px 16px;
    background: var(--el-bg-color);
    border: none;
    transition: all 0.3s ease;

    &:hover {
      background: var(--el-fill-color-light);
    }

    &.is-active {
      border-bottom: 1px solid var(--el-border-color-lighter);
    }
  }

  :deep(.el-collapse-item__wrap) {
    border: none;
  }

  :deep(.el-collapse-item__content) {
    padding: 16px;
    background: var(--el-fill-color-blank);
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
  flex: 1;
}

.shift-type-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid;
  white-space: nowrap;
}

.shift-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.shift-time {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.staff-list-container {
  margin-top: 8px;
}

.staff-scroll-wrapper {
  overflow-y: auto;
  overflow-x: hidden;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--el-fill-color-lighter);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--el-border-color);
    border-radius: 3px;
    transition: background 0.3s;

    &:hover {
      background: var(--el-border-color-dark);
    }
  }
}

.staff-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  margin-bottom: 8px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }
}

.staff-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--el-color-primary-light-8) 0%, var(--el-color-primary-light-9) 100%);
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 18px;
}

.staff-info {
  flex: 1;
  min-width: 0;
}

.staff-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.staff-position {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

// 响应式设计
@media (max-width: 768px) {
  .stats-overview {
    grid-template-columns: 1fr;
  }

  .staff-list {
    grid-template-columns: 1fr;
  }
}
</style>
