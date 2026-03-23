<script setup lang="ts">
import { Document, InfoFilled, Tickets, UserFilled } from '@element-plus/icons-vue'
import { ElButton, ElCollapse, ElCollapseItem, ElDialog, ElEmpty, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface Rule {
  id: string
  name: string
  type: string
  description: string
  priority: number
}

interface ShiftRules {
  shiftId: string
  shiftName: string
  startTime: string
  endTime: string
  rulesCount: number
  rules: Rule[]
}

interface GroupRules {
  groupId: string
  groupName: string
  rulesCount: number
  rules: Rule[]
}

interface RulesDetailsData {
  totalRules: number
  globalRulesCount: number
  shiftRulesCount: number
  groupRulesCount: number
  globalRules: Rule[]
  shiftRules: ShiftRules[]
  groupRules: GroupRules[]
}

interface Props {
  visible: boolean
  data: RulesDetailsData | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 默认展开第一个分类
const activeNames = ref<string[]>(['global'])

// 虚拟滚动相关
const ruleScrollTops = ref<Map<string, number>>(new Map())
const ruleItemHeight = 88 // 每个规则卡片的高度
const visibleRuleCount = 5 // 可见规则数量
const bufferRuleCount = 2 // 缓冲区数量

// 计算统计信息
const stats = computed(() => {
  if (!props.data) {
    return {
      totalRules: 0,
      globalRulesCount: 0,
      shiftRulesCount: 0,
      groupRulesCount: 0,
    }
  }

  return {
    totalRules: props.data.totalRules,
    globalRulesCount: props.data.globalRulesCount,
    shiftRulesCount: props.data.shiftRulesCount,
    groupRulesCount: props.data.groupRulesCount,
  }
})

// 为规则列表计算虚拟滚动数据
function getVisibleRules(rules: Rule[], scrollKey: string) {
  const scrollTop = ruleScrollTops.value.get(scrollKey) || 0

  if (rules.length <= visibleRuleCount) {
    // 规则少，不需要虚拟滚动
    return {
      visible: rules.map((rule, index) => ({ ...rule, offset: index * ruleItemHeight })),
      totalHeight: rules.length * ruleItemHeight,
      needScroll: false,
    }
  }

  const startIndex = Math.max(0, Math.floor(scrollTop / ruleItemHeight) - bufferRuleCount)
  const endIndex = Math.min(
    rules.length,
    Math.ceil((scrollTop + visibleRuleCount * ruleItemHeight) / ruleItemHeight) + bufferRuleCount,
  )

  return {
    visible: rules.slice(startIndex, endIndex).map((rule, index) => ({
      ...rule,
      offset: (startIndex + index) * ruleItemHeight,
    })),
    totalHeight: rules.length * ruleItemHeight,
    needScroll: true,
  }
}

// 处理规则列表滚动
function handleRuleScroll(scrollKey: string, event: Event) {
  const target = event.target as HTMLElement
  ruleScrollTops.value.set(scrollKey, target.scrollTop)
}

// 获取优先级标签
function getPriorityTag(priority: number) {
  if (priority >= 8)
    return { text: `P${priority}`, type: 'danger' }
  else if (priority >= 4)
    return { text: `P${priority}`, type: 'warning' }
  else
    return { text: `P${priority}`, type: 'info' }
}

// 监听对话框打开，重置滚动状态
watch(() => props.visible, (newVal) => {
  if (newVal) {
    ruleScrollTops.value.clear()
    activeNames.value = ['global']
  }
})

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :model-value="visible"
    title="规则详情"
    width="750px"
    top="5vh"
    :before-close="handleClose"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="rules-details-content">
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
              {{ stats.totalRules }}
            </div>
            <div class="stat-label">
              规则总数
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
              {{ stats.globalRulesCount }}
            </div>
            <div class="stat-label">
              全局规则
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
              {{ stats.shiftRulesCount }}
            </div>
            <div class="stat-label">
              班次规则
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
              {{ stats.groupRulesCount }}
            </div>
            <div class="stat-label">
              分组规则
            </div>
          </div>
        </div>
      </div>

      <!-- 规则列表 -->
      <div class="rules-section">
        <div class="section-title">
          规则详细信息
        </div>

        <ElCollapse v-model="activeNames" class="rules-collapse">
          <!-- 全局规则 -->
          <ElCollapseItem v-if="data.globalRules && data.globalRules.length > 0" name="global">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#409eff">
                    <InfoFilled />
                  </el-icon>
                  <span class="category-name">全局规则</span>
                </div>
                <ElTag type="primary" effect="plain">
                  {{ data.globalRules.length }} 条
                </ElTag>
              </div>
            </template>

            <div class="rules-list-container">
              <div
                class="rules-scroll-wrapper"
                :style="{
                  maxHeight: getVisibleRules(data.globalRules, 'global').needScroll
                    ? `${visibleRuleCount * ruleItemHeight}px`
                    : 'none',
                }"
                @scroll="(e) => handleRuleScroll('global', e)"
              >
                <div
                  :style="{
                    height: `${getVisibleRules(data.globalRules, 'global').totalHeight}px`,
                    position: 'relative',
                  }"
                >
                  <div
                    v-for="rule in getVisibleRules(data.globalRules, 'global').visible"
                    :key="rule.id"
                    class="rule-item"
                    :style="{
                      position: 'absolute',
                      top: `${rule.offset}px`,
                      left: 0,
                      right: 0,
                    }"
                  >
                    <div class="rule-header">
                      <div class="rule-name">
                        {{ rule.name }}
                      </div>
                      <div class="rule-tags">
                        <ElTag :type="getPriorityTag(rule.priority).type as any" size="small" effect="plain">
                          优先级: {{ getPriorityTag(rule.priority).text }}
                        </ElTag>
                      </div>
                    </div>
                    <div class="rule-description">
                      {{ rule.description }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <!-- 班次规则 -->
          <ElCollapseItem v-if="data.shiftRules && data.shiftRules.length > 0" name="shift">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#67c23a">
                    <Tickets />
                  </el-icon>
                  <span class="category-name">班次规则</span>
                </div>
                <ElTag type="success" effect="plain">
                  {{ data.shiftRules.length }} 个班次
                </ElTag>
              </div>
            </template>

            <div class="shift-rules-container">
              <div
                v-for="shiftRule in data.shiftRules"
                :key="shiftRule.shiftId"
                class="shift-rule-section"
              >
                <div class="shift-rule-header">
                  <span class="shift-name">{{ shiftRule.shiftName }}</span>
                  <span class="shift-time">{{ shiftRule.startTime }} - {{ shiftRule.endTime }}</span>
                  <ElTag size="small" type="success">
                    {{ shiftRule.rulesCount }} 条规则
                  </ElTag>
                </div>

                <div v-if="shiftRule.rules && shiftRule.rules.length > 0" class="rules-list-container">
                  <div
                    class="rules-scroll-wrapper"
                    :style="{
                      maxHeight: getVisibleRules(shiftRule.rules, `shift_${shiftRule.shiftId}`).needScroll
                        ? `${visibleRuleCount * ruleItemHeight}px`
                        : 'none',
                    }"
                    @scroll="(e) => handleRuleScroll(`shift_${shiftRule.shiftId}`, e)"
                  >
                    <div
                      :style="{
                        height: `${getVisibleRules(shiftRule.rules, `shift_${shiftRule.shiftId}`).totalHeight}px`,
                        position: 'relative',
                      }"
                    >
                      <div
                        v-for="rule in getVisibleRules(shiftRule.rules, `shift_${shiftRule.shiftId}`).visible"
                        :key="rule.id"
                        class="rule-item"
                        :style="{
                          position: 'absolute',
                          top: `${rule.offset}px`,
                          left: 0,
                          right: 0,
                        }"
                      >
                        <div class="rule-header">
                          <div class="rule-name">
                            {{ rule.name }}
                          </div>
                          <div class="rule-tags">
                            <ElTag :type="getPriorityTag(rule.priority).type as any" size="small" effect="plain">
                              优先级: {{ getPriorityTag(rule.priority).text }}
                            </ElTag>
                          </div>
                        </div>
                        <div class="rule-description">
                          {{ rule.description }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <ElEmpty v-else description="该班次暂无规则" :image-size="60" />
              </div>
            </div>
          </ElCollapseItem>

          <!-- 分组规则 -->
          <ElCollapseItem v-if="data.groupRules && data.groupRules.length > 0" name="group">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#e6a23c">
                    <UserFilled />
                  </el-icon>
                  <span class="category-name">分组规则</span>
                </div>
                <ElTag type="warning" effect="plain">
                  {{ data.groupRules.length }} 个分组
                </ElTag>
              </div>
            </template>

            <div class="group-rules-container">
              <div
                v-for="groupRule in data.groupRules"
                :key="groupRule.groupId"
                class="group-rule-section"
              >
                <div class="group-rule-header">
                  <span class="group-name">{{ groupRule.groupName }}</span>
                  <ElTag size="small" type="warning">
                    {{ groupRule.rulesCount }} 条规则
                  </ElTag>
                </div>

                <div v-if="groupRule.rules && groupRule.rules.length > 0" class="rules-list-container">
                  <div
                    class="rules-scroll-wrapper"
                    :style="{
                      maxHeight: getVisibleRules(groupRule.rules, `group_${groupRule.groupId}`).needScroll
                        ? `${visibleRuleCount * ruleItemHeight}px`
                        : 'none',
                    }"
                    @scroll="(e) => handleRuleScroll(`group_${groupRule.groupId}`, e)"
                  >
                    <div
                      :style="{
                        height: `${getVisibleRules(groupRule.rules, `group_${groupRule.groupId}`).totalHeight}px`,
                        position: 'relative',
                      }"
                    >
                      <div
                        v-for="rule in getVisibleRules(groupRule.rules, `group_${groupRule.groupId}`).visible"
                        :key="rule.id"
                        class="rule-item"
                        :style="{
                          position: 'absolute',
                          top: `${rule.offset}px`,
                          left: 0,
                          right: 0,
                        }"
                      >
                        <div class="rule-header">
                          <div class="rule-name">
                            {{ rule.name }}
                          </div>
                          <div class="rule-tags">
                            <ElTag :type="getPriorityTag(rule.priority).type as any" size="small" effect="plain">
                              优先级: {{ getPriorityTag(rule.priority).text }}
                            </ElTag>
                          </div>
                        </div>
                        <div class="rule-description">
                          {{ rule.description }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <ElEmpty v-else description="该分组暂无规则" :image-size="60" />
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
.rules-details-content {
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

.rules-section {
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

.rules-collapse {
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

.rules-list-container {
  margin-top: 8px;
}

.rules-scroll-wrapper {
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

.rule-item {
  padding: 14px;
  margin-bottom: 10px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  }
}

.rule-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.rule-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  flex: 1;
}

.rule-tags {
  display: flex;
  gap: 6px;
}

.rule-description {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.shift-rules-container,
.group-rules-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.shift-rule-section,
.group-rule-section {
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.shift-rule-header,
.group-rule-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--el-border-color);
}

.shift-name,
.group-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  flex: 1;
}

.shift-time {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

// 响应式设计
@media (max-width: 768px) {
  .stats-overview {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
