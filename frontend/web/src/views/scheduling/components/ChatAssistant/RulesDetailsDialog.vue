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

const props = defineProps<{
  visible: boolean
  data: RulesDetailsData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const activeNames = ref<string[]>(['global'])

const stats = computed(() => {
  if (!props.data)
    return { totalRules: 0, globalRulesCount: 0, shiftRulesCount: 0, groupRulesCount: 0 }
  return {
    totalRules: props.data.totalRules,
    globalRulesCount: props.data.globalRulesCount,
    shiftRulesCount: props.data.shiftRulesCount,
    groupRulesCount: props.data.groupRulesCount,
  }
})

function getPriorityTag(priority: number) {
  if (priority >= 8)
    return { text: `P${priority}`, type: 'danger' as const }
  else if (priority >= 4)
    return { text: `P${priority}`, type: 'warning' as const }
  else
    return { text: `P${priority}`, type: 'info' as const }
}

watch(() => props.visible, (newVal) => {
  if (newVal)
    activeNames.value = ['global']
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
    title="规则详情"
    top="5vh"
    width="750px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="rules-details-content">
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

      <div class="rules-section">
        <div class="section-title">
          规则详细信息
        </div>
        <ElCollapse v-model="activeNames" class="rules-collapse">
          <ElCollapseItem v-if="data.globalRules && data.globalRules.length > 0" name="global">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#409eff">
                    <InfoFilled />
                  </el-icon>
                  <span class="category-name">全局规则</span>
                </div>
                <ElTag effect="plain" type="primary">
                  {{ data.globalRules.length }} 条
                </ElTag>
              </div>
            </template>
            <div class="rules-list">
              <div v-for="rule in data.globalRules" :key="rule.id" class="rule-item">
                <div class="rule-header">
                  <div class="rule-name">
                    {{ rule.name }}
                  </div>
                  <ElTag :type="getPriorityTag(rule.priority).type" effect="plain" size="small">
                    优先级: {{ getPriorityTag(rule.priority).text }}
                  </ElTag>
                </div>
                <div class="rule-description">
                  {{ rule.description }}
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <ElCollapseItem v-if="data.shiftRules && data.shiftRules.length > 0" name="shift">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#67c23a">
                    <Tickets />
                  </el-icon>
                  <span class="category-name">班次规则</span>
                </div>
                <ElTag effect="plain" type="success">
                  {{ data.shiftRules.length }} 个班次
                </ElTag>
              </div>
            </template>
            <div v-for="shiftRule in data.shiftRules" :key="shiftRule.shiftId" class="sub-section">
              <div class="sub-header">
                <span class="sub-name">{{ shiftRule.shiftName }}</span>
                <span class="sub-time">{{ shiftRule.startTime }} - {{ shiftRule.endTime }}</span>
                <ElTag size="small" type="success">
                  {{ shiftRule.rulesCount }} 条
                </ElTag>
              </div>
              <div v-if="shiftRule.rules.length > 0" class="rules-list">
                <div v-for="rule in shiftRule.rules" :key="rule.id" class="rule-item">
                  <div class="rule-header">
                    <div class="rule-name">
                      {{ rule.name }}
                    </div>
                    <ElTag :type="getPriorityTag(rule.priority).type" effect="plain" size="small">
                      优先级: {{ getPriorityTag(rule.priority).text }}
                    </ElTag>
                  </div>
                  <div class="rule-description">
                    {{ rule.description }}
                  </div>
                </div>
              </div>
              <ElEmpty v-else :image-size="60" description="该班次暂无规则" />
            </div>
          </ElCollapseItem>

          <ElCollapseItem v-if="data.groupRules && data.groupRules.length > 0" name="group">
            <template #title>
              <div class="category-header">
                <div class="category-info">
                  <el-icon :size="18" color="#e6a23c">
                    <UserFilled />
                  </el-icon>
                  <span class="category-name">分组规则</span>
                </div>
                <ElTag effect="plain" type="warning">
                  {{ data.groupRules.length }} 个分组
                </ElTag>
              </div>
            </template>
            <div v-for="groupRule in data.groupRules" :key="groupRule.groupId" class="sub-section">
              <div class="sub-header">
                <span class="sub-name">{{ groupRule.groupName }}</span>
                <ElTag size="small" type="warning">
                  {{ groupRule.rulesCount }} 条
                </ElTag>
              </div>
              <div v-if="groupRule.rules.length > 0" class="rules-list">
                <div v-for="rule in groupRule.rules" :key="rule.id" class="rule-item">
                  <div class="rule-header">
                    <div class="rule-name">
                      {{ rule.name }}
                    </div>
                    <ElTag :type="getPriorityTag(rule.priority).type" effect="plain" size="small">
                      优先级: {{ getPriorityTag(rule.priority).text }}
                    </ElTag>
                  </div>
                  <div class="rule-description">
                    {{ rule.description }}
                  </div>
                </div>
              </div>
              <ElEmpty v-else :image-size="60" description="该分组暂无规则" />
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
.rules-details-content { padding: 8px 0; }

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
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-value { font-size: 22px; font-weight: 600; }
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }

.rules-section { margin-top: 24px; max-height: 550px; overflow-y: auto; }
.section-title { font-size: 15px; font-weight: 600; margin-bottom: 16px; padding-bottom: 8px; border-bottom: 2px solid var(--el-border-color-lighter); }

.rules-collapse {
  border: none;
  :deep(.el-collapse-item) { margin-bottom: 12px; border: 1px solid var(--el-border-color-light); border-radius: 8px; overflow: hidden; }
  :deep(.el-collapse-item__header) { height: auto; padding: 14px 16px; border: none; }
  :deep(.el-collapse-item__content) { padding: 16px; }
}

.category-header { display: flex; align-items: center; justify-content: space-between; width: 100%; padding-right: 12px; }
.category-info { display: flex; align-items: center; gap: 10px; }
.category-name { font-size: 15px; font-weight: 600; }

.rules-list { display: flex; flex-direction: column; gap: 10px; }
.rule-item { padding: 14px; background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 6px; }
.rule-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.rule-name { font-size: 14px; font-weight: 600; }
.rule-description { font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; }

.sub-section { padding: 12px; background: var(--el-fill-color-lighter); border-radius: 6px; margin-bottom: 12px; }
.sub-header { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; padding-bottom: 10px; border-bottom: 1px dashed var(--el-border-color); }
.sub-name { font-size: 14px; font-weight: 600; flex: 1; }
.sub-time { font-size: 13px; color: var(--el-text-color-secondary); }
</style>
