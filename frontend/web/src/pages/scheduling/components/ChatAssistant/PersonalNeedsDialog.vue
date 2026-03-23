<script setup lang="ts">
import { Document, InfoFilled, Tickets, UserFilled } from '@element-plus/icons-vue'
import { ElButton, ElCollapse, ElCollapseItem, ElDialog, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface PersonalNeed {
  staffId: string
  staffName: string
  needType: 'permanent' | 'temporary'
  requestType: 'prefer' | 'avoid' | 'must'
  targetShiftId?: string
  targetShiftName?: string
  targetDates?: string[]
  description: string
  priority: number
  ruleId?: string
  source: 'rule' | 'user'
  confirmed: boolean
}

interface PersonalNeedsData {
  totalNeeds: number
  permanentNeedsCount: number
  temporaryNeedsCount: number
  needsByStaff: Record<string, PersonalNeed[]>
  allNeeds: PersonalNeed[]
}

interface Props {
  visible: boolean
  data: PersonalNeedsData | null
}

interface NeedType {
  text: string
  type: 'primary' | 'success' | 'warning' | 'info' | 'danger'
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 默认展开第一个分类
const activeNames = ref<string[]>(['permanent'])

// 计算统计信息
const stats = computed(() => {
  if (!props.data) {
    return {
      totalNeeds: 0,
      permanentNeedsCount: 0,
      temporaryNeedsCount: 0,
    }
  }

  return {
    totalNeeds: props.data.totalNeeds,
    permanentNeedsCount: props.data.permanentNeedsCount,
    temporaryNeedsCount: props.data.temporaryNeedsCount,
  }
})

// 获取需求类型标签
function getNeedTypeTag(type: string): NeedType {
  if (type === 'permanent')
    return { text: '常态化', type: 'primary' }
  else
    return { text: '临时', type: 'warning' }
}

// 获取请求类型标签
function getRequestTypeTag(type: string): NeedType {
  if (type === 'must')
    return { text: '必须', type: 'danger' }
  else if (type === 'prefer')
    return { text: '偏好', type: 'success' }
  else
    return { text: '回避', type: 'info' }
}

// 获取优先级标签
function getPriorityTag(priority: number): NeedType {
  if (priority >= 8)
    return { text: `P${priority}`, type: 'danger' }
  else if (priority >= 4)
    return { text: `P${priority}`, type: 'warning' }
  else
    return { text: `P${priority}`, type: 'info' }
}

// 监听对话框打开，重置状态
watch(() => props.visible, (newVal) => {
  if (newVal) {
    activeNames.value = ['permanent']
  }
})

function handleClose() {
  emit('update:visible', false)
  emit('close')
}

// 按人员分组的需求列表
const needsByStaff = computed(() => {
  if (!props.data?.needsByStaff)
    return []

  return Object.entries(props.data.needsByStaff)
    .map(([staffId, needs]) => {
      // 从需求列表中获取人员姓名（所有需求应该属于同一人员，取第一个即可）
      // 遍历所有需求，找到第一个有 staffName 的
      let staffName = staffId // 默认使用 ID
      if (needs && needs.length > 0) {
        for (const need of needs) {
          if (need?.staffName && need.staffName !== staffId) {
            staffName = need.staffName
            break
          }
        }
      }

      return {
        staffId,
        staffName,
        needs,
      }
    })
    .filter(staff => staff.needs && staff.needs.length > 0) // 过滤掉空的需求列表
})

// 常态化需求
const permanentNeeds = computed(() => {
  if (!props.data?.allNeeds)
    return []
  return props.data.allNeeds.filter(n => n.needType === 'permanent')
})

// 临时需求
const temporaryNeeds = computed(() => {
  if (!props.data?.allNeeds)
    return []
  return props.data.allNeeds.filter(n => n.needType === 'temporary')
})
</script>

<template>
  <ElDialog
    :model-value="visible"
    title="个人需求详情"
    width="800px"
    top="5vh"
    :before-close="handleClose"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="personal-needs-content">
      <!-- 统计概览 -->
      <div class="stats-overview">
        <div class="stat-card">
          <div class="stat-icon" style="background: #ecf5ff; color: #409eff;">
            <el-icon :size="24">
              <Document />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.totalNeeds }}
            </div>
            <div class="stat-label">
              需求总数
            </div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: #f0f9ff; color: #67c23a;">
            <el-icon :size="24">
              <InfoFilled />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.permanentNeedsCount }}
            </div>
            <div class="stat-label">
              常态化需求
            </div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: #fef0e6; color: #e6a23c;">
            <el-icon :size="24">
              <Tickets />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.temporaryNeedsCount }}
            </div>
            <div class="stat-label">
              临时需求
            </div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: #fef0f0; color: #f56c6c;">
            <el-icon :size="24">
              <UserFilled />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ needsByStaff.length }}
            </div>
            <div class="stat-label">
              涉及人员
            </div>
          </div>
        </div>
      </div>

      <!-- 需求列表 -->
      <div class="needs-section">
        <div class="section-title">
          需求详细信息
        </div>

        <ElCollapse v-model="activeNames" class="needs-collapse">
          <!-- 常态化需求 -->
          <ElCollapseItem v-if="permanentNeeds.length > 0" name="permanent">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#409eff">
                    <InfoFilled />
                  </el-icon>
                  <span class="category-name">常态化需求</span>
                </div>
                <ElTag type="primary" effect="plain">
                  {{ permanentNeeds.length }} 条
                </ElTag>
              </div>
            </template>

            <div class="needs-list-container">
              <div
                v-for="need in permanentNeeds"
                :key="`${need.staffId}-${need.ruleId || 'user'}-${need.description}`"
                class="need-item"
              >
                <div class="need-header">
                  <div class="need-staff">
                    <el-icon :size="16" color="#409eff">
                      <UserFilled />
                    </el-icon>
                    <span class="staff-name">{{ need.staffName }}</span>
                  </div>
                  <div class="need-tags">
                    <ElTag :type="getNeedTypeTag(need.needType).type" size="small" effect="plain">
                      {{ getNeedTypeTag(need.needType).text }}
                    </ElTag>
                    <ElTag :type="getRequestTypeTag(need.requestType).type" size="small" effect="plain">
                      {{ getRequestTypeTag(need.requestType).text }}
                    </ElTag>
                    <ElTag :type="getPriorityTag(need.priority).type" size="small" effect="plain">
                      优先级: {{ getPriorityTag(need.priority).text }}
                    </ElTag>
                  </div>
                </div>
                <div class="need-content">
                  <div class="need-description">
                    {{ need.description }}
                  </div>
                  <div v-if="need.targetShiftName" class="need-details">
                    <span class="detail-label">班次：</span>
                    <span class="detail-value">{{ need.targetShiftName }}</span>
                  </div>
                  <div v-if="need.targetDates && need.targetDates.length > 0" class="need-details">
                    <span class="detail-label">日期：</span>
                    <span class="detail-value">{{ need.targetDates.join('、') }}</span>
                  </div>
                  <div v-if="need.source === 'rule'" class="need-details">
                    <span class="detail-label">来源：</span>
                    <span class="detail-value">规则系统</span>
                  </div>
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <!-- 临时需求 -->
          <ElCollapseItem v-if="temporaryNeeds.length > 0" name="temporary">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#e6a23c">
                    <Tickets />
                  </el-icon>
                  <span class="category-name">临时需求</span>
                </div>
                <ElTag type="warning" effect="plain">
                  {{ temporaryNeeds.length }} 条
                </ElTag>
              </div>
            </template>

            <div class="needs-list-container">
              <div
                v-for="need in temporaryNeeds"
                :key="`${need.staffId}-${need.ruleId || 'user'}-${need.description}`"
                class="need-item"
              >
                <div class="need-header">
                  <div class="need-staff">
                    <el-icon :size="16" color="#e6a23c">
                      <UserFilled />
                    </el-icon>
                    <span class="staff-name">{{ need.staffName }}</span>
                  </div>
                  <div class="need-tags">
                    <ElTag :type="getNeedTypeTag(need.needType).type" size="small" effect="plain">
                      {{ getNeedTypeTag(need.needType).text }}
                    </ElTag>
                    <ElTag :type="getRequestTypeTag(need.requestType).type" size="small" effect="plain">
                      {{ getRequestTypeTag(need.requestType).text }}
                    </ElTag>
                    <ElTag :type="getPriorityTag(need.priority).type" size="small" effect="plain">
                      优先级: {{ getPriorityTag(need.priority).text }}
                    </ElTag>
                  </div>
                </div>
                <div class="need-content">
                  <div v-if="need.description" class="need-description">
                    {{ need.description }}
                  </div>
                  <div v-else class="need-description" style="color: var(--el-text-color-placeholder); font-style: italic;">
                    暂无描述
                  </div>
                  <div v-if="need.targetShiftName" class="need-details">
                    <span class="detail-label">班次：</span>
                    <span class="detail-value">{{ need.targetShiftName }}</span>
                  </div>
                  <div v-if="need.targetDates && need.targetDates.length > 0" class="need-details">
                    <span class="detail-label">日期：</span>
                    <span class="detail-value">{{ need.targetDates.join('、') }}</span>
                  </div>
                  <div v-if="need.source === 'user'" class="need-details">
                    <span class="detail-label">来源：</span>
                    <span class="detail-value">用户补充</span>
                  </div>
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <!-- 按人员分组 -->
          <ElCollapseItem v-if="needsByStaff.length > 0" name="by-staff">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#67c23a">
                    <UserFilled />
                  </el-icon>
                  <span class="category-name">按人员分组</span>
                </div>
                <ElTag type="success" effect="plain">
                  {{ needsByStaff.length }} 人
                </ElTag>
              </div>
            </template>

            <div class="staff-needs-container">
              <div
                v-for="staff in needsByStaff"
                :key="staff.staffId"
                class="staff-need-section"
              >
                <div class="staff-need-header">
                  <span class="staff-name">{{ staff.staffName || staff.staffId }}</span>
                  <ElTag size="small" type="success">
                    {{ staff.needs.length }} 条需求
                  </ElTag>
                </div>

                <div class="needs-list-container">
                  <div
                    v-for="need in staff.needs"
                    :key="`${need.staffId}-${need.ruleId || 'user'}-${need.description}`"
                    class="need-item"
                  >
                    <div class="need-header">
                      <div class="need-tags">
                        <ElTag :type="getNeedTypeTag(need.needType).type" size="small" effect="plain">
                          {{ getNeedTypeTag(need.needType).text }}
                        </ElTag>
                        <ElTag :type="getRequestTypeTag(need.requestType).type" size="small" effect="plain">
                          {{ getRequestTypeTag(need.requestType).text }}
                        </ElTag>
                        <ElTag :type="getPriorityTag(need.priority).type" size="small" effect="plain">
                          优先级: {{ getPriorityTag(need.priority).text }}
                        </ElTag>
                      </div>
                    </div>
                    <div class="need-content">
                      <div v-if="need.description" class="need-description">
                        {{ need.description }}
                      </div>
                      <div v-else class="need-description" style="color: var(--el-text-color-placeholder); font-style: italic;">
                        暂无描述
                      </div>
                      <div v-if="need.targetShiftName" class="need-details">
                        <span class="detail-label">班次：</span>
                        <span class="detail-value">{{ need.targetShiftName }}</span>
                      </div>
                      <div v-if="need.targetDates && need.targetDates.length > 0" class="need-details">
                        <span class="detail-label">日期：</span>
                        <span class="detail-value">{{ need.targetDates.join('、') }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
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
.personal-needs-content {
  padding: 8px 0;
}

.stats-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px;
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
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--el-text-color-primary);
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.needs-section {
  margin-top: 24px;
  max-height: 550px;
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

.needs-collapse {
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
    padding: 14px 16px;
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

.category-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 12px;
}

.category-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.category-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.needs-list-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.need-item {
  padding: 14px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }
}

.need-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.need-staff {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
}

.staff-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.need-tags {
  display: flex;
  gap: 6px;
}

.need-content {
  margin-top: 8px;
}

.need-description {
  font-size: 13px;
  color: var(--el-text-color-primary);
  line-height: 1.6;
  margin-bottom: 8px;
}

.need-details {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  display: flex;
  gap: 4px;
}

.detail-label {
  font-weight: 500;
}

.detail-value {
  color: var(--el-text-color-primary);
}

.staff-needs-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.staff-need-section {
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.staff-need-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--el-border-color);
}

// 响应式设计
@media (max-width: 768px) {
  .stats-overview {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
