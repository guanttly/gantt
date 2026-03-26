<script setup lang="ts">
import { Search, UserFilled } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElInput, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface TempRule {
  id: string
  name: string
  category: string
  description: string
  priority: number
  associatedStaffIds?: string[]
  associatedStaffNames?: string[]
  effectiveDates?: string[]
}

interface TemporaryRulesData {
  totalRules: number
  rules: TempRule[]
}

const props = defineProps<{
  visible: boolean
  data: TemporaryRulesData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const searchQuery = ref('')

const filteredRules = computed(() => {
  if (!props.data?.rules)
    return []
  if (!searchQuery.value)
    return props.data.rules
  const q = searchQuery.value.toLowerCase()
  return props.data.rules.filter(
    r => r.name.toLowerCase().includes(q) || r.description.toLowerCase().includes(q) || r.category.toLowerCase().includes(q),
  )
})

function getCategoryTagType(category: string) {
  const map: Record<string, string> = {
    排班约束: 'primary',
    工时限制: 'success',
    人员偏好: 'warning',
    公平性: 'info',
    覆盖规则: 'danger',
  }
  return (map[category] || 'info') as 'primary' | 'success' | 'warning' | 'info' | 'danger'
}

function getPriorityColor(priority: number): string {
  if (priority >= 8)
    return '#f56c6c'
  if (priority >= 4)
    return '#e6a23c'
  return '#909399'
}

watch(() => props.visible, (newVal) => {
  if (newVal)
    searchQuery.value = ''
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
    title="临时规则"
    top="5vh"
    width="700px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="temp-rules-content">
      <div class="search-bar">
        <ElInput
          v-model="searchQuery"
          :prefix-icon="Search"
          clearable
          placeholder="搜索规则名称、描述或类别..."
        />
        <ElTag class="total-tag" effect="plain" type="info">
          共 {{ data.totalRules }} 条规则
        </ElTag>
      </div>

      <div v-if="filteredRules.length > 0" class="rules-list">
        <div
          v-for="rule in filteredRules"
          :key="rule.id"
          class="rule-card"
        >
          <div class="rule-header">
            <div class="rule-title">
              <span class="rule-name">{{ rule.name }}</span>
              <ElTag :type="getCategoryTagType(rule.category)" effect="plain" size="small">
                {{ rule.category }}
              </ElTag>
            </div>
            <div class="rule-priority" :style="{ color: getPriorityColor(rule.priority) }">
              P{{ rule.priority }}
            </div>
          </div>
          <div class="rule-description">
            {{ rule.description }}
          </div>
          <div v-if="rule.associatedStaffNames?.length" class="rule-associations">
            <div class="assoc-label">
              <el-icon :size="14">
                <UserFilled />
              </el-icon>
              关联人员：
            </div>
            <div class="assoc-tags">
              <ElTag
                v-for="name in rule.associatedStaffNames"
                :key="name"
                effect="plain"
                size="small"
              >
                {{ name }}
              </ElTag>
            </div>
          </div>
          <div v-if="rule.effectiveDates?.length" class="rule-dates">
            <span class="dates-label">生效日期：</span>
            <span class="dates-value">{{ rule.effectiveDates.join(', ') }}</span>
          </div>
        </div>
      </div>
      <ElEmpty v-else :image-size="80" description="未找到匹配的规则" />
    </div>
    <ElEmpty v-else description="暂无临时规则数据" />

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.temp-rules-content { padding: 8px 0; }

.search-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;

  .el-input { flex: 1; }
  .total-tag { flex-shrink: 0; }
}

.rules-list {
  max-height: 550px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.rule-card {
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  transition: all 0.2s;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  }
}

.rule-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.rule-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.rule-name { font-size: 14px; font-weight: 600; }
.rule-priority { font-size: 14px; font-weight: 600; }
.rule-description { font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; margin-bottom: 10px; }

.rule-associations {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 8px;

  .assoc-label {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    margin-top: 3px;
  }

  .assoc-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
}

.rule-dates {
  font-size: 12px;
  color: var(--el-text-color-secondary);

  .dates-label { font-weight: 500; }
}
</style>
