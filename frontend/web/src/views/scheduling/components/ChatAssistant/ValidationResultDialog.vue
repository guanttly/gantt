<script setup lang="ts">
import { CircleCheck, Search, Warning, WarningFilled } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElInput, ElTabPane, ElTabs, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface ValidationItem {
  id: string
  type: 'violation' | 'warning'
  severity: 'high' | 'medium' | 'low'
  ruleName: string
  description: string
  affectedStaff?: string[]
  affectedDates?: string[]
}

interface ValidationResultData {
  totalIssues: number
  violationCount: number
  warningCount: number
  items: ValidationItem[]
  isValid: boolean
}

const props = defineProps<{
  visible: boolean
  data: ValidationResultData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const activeTab = ref<string>('all')
const searchQuery = ref('')

const filteredItems = computed(() => {
  if (!props.data?.items)
    return []
  let items = props.data.items
  if (activeTab.value === 'violations')
    items = items.filter(i => i.type === 'violation')
  else if (activeTab.value === 'warnings')
    items = items.filter(i => i.type === 'warning')

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    items = items.filter(
      i => i.ruleName.toLowerCase().includes(q) || i.description.toLowerCase().includes(q),
    )
  }
  return items
})

function getSeverityTagType(severity: string) {
  const map: Record<string, 'danger' | 'warning' | 'info'> = {
    high: 'danger',
    medium: 'warning',
    low: 'info',
  }
  return map[severity] || 'info'
}

function getSeverityLabel(severity: string): string {
  const map: Record<string, string> = { high: '高', medium: '中', low: '低' }
  return map[severity] || severity
}

watch(() => props.visible, (newVal) => {
  if (newVal) {
    activeTab.value = 'all'
    searchQuery.value = ''
  }
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
    title="验证结果"
    top="5vh"
    width="750px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="validation-content">
      <div class="validation-summary" :class="{ 'is-valid': data.isValid }">
        <el-icon v-if="data.isValid" :size="32" color="#67c23a">
          <CircleCheck />
        </el-icon>
        <el-icon v-else :size="32" color="#f56c6c">
          <WarningFilled />
        </el-icon>
        <div class="summary-text">
          <div v-if="data.isValid" class="summary-title text-success">
            验证通过
          </div>
          <div v-else class="summary-title text-danger">
            发现 {{ data.totalIssues }} 个问题
          </div>
          <div class="summary-detail">
            {{ data.violationCount }} 个违规 · {{ data.warningCount }} 个警告
          </div>
        </div>
      </div>

      <div class="search-bar">
        <ElInput
          v-model="searchQuery"
          :prefix-icon="Search"
          clearable
          placeholder="搜索规则或描述..."
        />
      </div>

      <ElTabs v-model="activeTab">
        <ElTabPane label="全部" name="all">
          <template #label>
            <span>全部 ({{ data.items.length }})</span>
          </template>
        </ElTabPane>
        <ElTabPane label="违规" name="violations">
          <template #label>
            <span style="color: #f56c6c;">违规 ({{ data.violationCount }})</span>
          </template>
        </ElTabPane>
        <ElTabPane label="警告" name="warnings">
          <template #label>
            <span style="color: #e6a23c;">警告 ({{ data.warningCount }})</span>
          </template>
        </ElTabPane>
      </ElTabs>

      <div v-if="filteredItems.length > 0" class="issues-list">
        <div
          v-for="item in filteredItems"
          :key="item.id"
          class="issue-card"
          :class="{ 'issue-violation': item.type === 'violation', 'issue-warning': item.type === 'warning' }"
        >
          <div class="issue-header">
            <div class="issue-type">
              <el-icon v-if="item.type === 'violation'" color="#f56c6c">
                <WarningFilled />
              </el-icon>
              <el-icon v-else color="#e6a23c">
                <Warning />
              </el-icon>
              <span class="issue-rule">{{ item.ruleName }}</span>
            </div>
            <ElTag :type="getSeverityTagType(item.severity)" effect="plain" size="small">
              {{ getSeverityLabel(item.severity) }}
            </ElTag>
          </div>
          <div class="issue-description">
            {{ item.description }}
          </div>
          <div v-if="item.affectedStaff?.length" class="issue-meta">
            <span class="meta-label">影响人员：</span>
            <ElTag v-for="name in item.affectedStaff" :key="name" effect="plain" size="small">
              {{ name }}
            </ElTag>
          </div>
          <div v-if="item.affectedDates?.length" class="issue-meta">
            <span class="meta-label">影响日期：</span>
            <span class="meta-value">{{ item.affectedDates.join(', ') }}</span>
          </div>
        </div>
      </div>
      <ElEmpty v-else :image-size="80" description="没有找到匹配的问题" />
    </div>
    <ElEmpty v-else description="暂无验证结果" />

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.validation-content { padding: 8px 0; }

.validation-summary {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);

  &.is-valid { background: rgba(103, 194, 58, 0.06); border-color: rgba(103, 194, 58, 0.3); }
}

.summary-title { font-size: 18px; font-weight: 600; }
.summary-detail { font-size: 13px; color: var(--el-text-color-secondary); margin-top: 4px; }
.text-success { color: #67c23a; }
.text-danger { color: #f56c6c; }

.search-bar { margin-bottom: 12px; }

.issues-list {
  max-height: 450px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 12px;
}

.issue-card {
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}

.issue-violation { border-left: 3px solid #f56c6c; }
.issue-warning { border-left: 3px solid #e6a23c; }

.issue-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.issue-type { display: flex; align-items: center; gap: 8px; }
.issue-rule { font-size: 14px; font-weight: 600; }
.issue-description { font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; margin-bottom: 8px; }

.issue-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;

  .meta-label { font-size: 12px; color: var(--el-text-color-secondary); white-space: nowrap; }
  .meta-value { font-size: 12px; color: var(--el-text-color-regular); }
}
</style>
