<script setup lang="ts">
import { Clock, Document } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElScrollbar, ElTag, ElTooltip } from 'element-plus'
import { computed, ref, watch } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'

// 临时规则接口
interface TemporaryRule {
  id: string
  name: string
  category: string // constraint/preference
  subCategory: string // forbid/must/prefer/avoid
  ruleType: string
  description: string
  ruleData?: string
  priority: number
  associations?: Array<{
    associationType: string
    associationId: string
  }>
  // 关联名称（前端展示用）
  staffName?: string
  shiftName?: string
  targetDates?: string[]
}

interface TemporaryRulesData {
  totalRules: number
  constraintCount: number
  preferenceCount: number
  rules: TemporaryRule[]
}

interface Props {
  visible: boolean
  data: TemporaryRulesData | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 搜索关键词
const searchKeyword = ref('')

// 过滤后的规则列表
const filteredRules = computed(() => {
  if (!props.data?.rules)
    return []

  const keyword = searchKeyword.value.toLowerCase().trim()
  if (!keyword)
    return props.data.rules

  return props.data.rules.filter(rule =>
    rule.name.toLowerCase().includes(keyword)
    || rule.description.toLowerCase().includes(keyword)
    || rule.staffName?.toLowerCase().includes(keyword)
    || rule.shiftName?.toLowerCase().includes(keyword),
  )
})

// 统计信息
const stats = computed(() => {
  if (!props.data) {
    return {
      totalRules: 0,
      constraintCount: 0,
      preferenceCount: 0,
    }
  }

  return {
    totalRules: props.data.totalRules,
    constraintCount: props.data.constraintCount,
    preferenceCount: props.data.preferenceCount,
  }
})

// 获取分类图标
function getCategoryIcon(category: string, subCategory: string) {
  if (category === 'constraint') {
    if (subCategory === 'forbid')
      return 'ban'
    if (subCategory === 'must')
      return 'check-circle'
    return 'warning'
  }
  return 'lightbulb'
}

// 获取分类标签
function getCategoryTag(category: string, subCategory: string) {
  if (category === 'constraint') {
    if (subCategory === 'forbid')
      return { text: '禁止', type: 'danger' as const }
    if (subCategory === 'must')
      return { text: '必须', type: 'success' as const }
    if (subCategory === 'avoid')
      return { text: '回避', type: 'warning' as const }
    return { text: '约束', type: 'warning' as const }
  }
  return { text: '偏好', type: 'info' as const }
}

// 获取优先级标签
function getPriorityTag(priority: number) {
  if (priority <= 2)
    return { text: `P${priority}`, type: 'danger' as const }
  if (priority <= 4)
    return { text: `P${priority}`, type: 'warning' as const }
  return { text: `P${priority}`, type: 'info' as const }
}

// 格式化日期列表
function formatDates(dates?: string[]) {
  if (!dates || dates.length === 0)
    return ''
  if (dates.length <= 3)
    return dates.join(', ')
  return `${dates.slice(0, 3).join(', ')} 等${dates.length}天`
}

// 监听对话框打开，重置状态
watch(() => props.visible, (newVal) => {
  if (newVal) {
    searchKeyword.value = ''
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
    title="临时规则详情"
    width="700px"
    top="5vh"
    :before-close="handleClose"
    class="temporary-rules-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="temporary-rules-content">
      <!-- 统计概览 -->
      <div class="stats-overview">
        <div class="stat-item">
          <div class="stat-value">
            {{ stats.totalRules }}
          </div>
          <div class="stat-label">
            总规则数
          </div>
        </div>
        <div class="stat-item constraint">
          <div class="stat-value">
            {{ stats.constraintCount }}
          </div>
          <div class="stat-label">
            约束规则
          </div>
        </div>
        <div class="stat-item preference">
          <div class="stat-value">
            {{ stats.preferenceCount }}
          </div>
          <div class="stat-label">
            偏好规则
          </div>
        </div>
      </div>

      <!-- 搜索框 -->
      <div class="search-bar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索规则名称、描述或人员..."
          clearable
          prefix-icon="Search"
        />
      </div>

      <!-- 规则列表 -->
      <ElScrollbar max-height="450px" class="rules-scrollbar">
        <div v-if="filteredRules.length === 0" class="empty-state">
          <ElEmpty description="暂无匹配的规则" />
        </div>

        <div v-else class="rules-list">
          <div
            v-for="(rule, index) in filteredRules"
            :key="rule.id"
            class="rule-card"
          >
            <!-- 序号和图标 -->
            <div class="rule-header">
              <span class="rule-index">{{ index + 1 }}.</span>
              <span class="rule-icon"><SvgIcon :name="getCategoryIcon(rule.category, rule.subCategory)" size="1em" /></span>
              <span class="rule-name">{{ rule.name }}</span>
              <div class="rule-tags">
                <ElTag
                  :type="getCategoryTag(rule.category, rule.subCategory).type"
                  size="small"
                  effect="light"
                >
                  {{ getCategoryTag(rule.category, rule.subCategory).text }}
                </ElTag>
                <ElTag
                  :type="getPriorityTag(rule.priority).type"
                  size="small"
                  effect="plain"
                >
                  {{ getPriorityTag(rule.priority).text }}
                </ElTag>
              </div>
            </div>

            <!-- 规则描述 -->
            <div class="rule-description">
              {{ rule.description }}
            </div>

            <!-- 关联信息 -->
            <div v-if="rule.staffName || rule.shiftName || rule.targetDates?.length" class="rule-associations">
              <span v-if="rule.staffName" class="association-item">
                <el-icon><UserFilled /></el-icon>
                {{ rule.staffName }}
              </span>
              <span v-if="rule.shiftName" class="association-item">
                <el-icon><Clock /></el-icon>
                {{ rule.shiftName }}
              </span>
              <span v-if="rule.targetDates?.length" class="association-item">
                <el-icon><Document /></el-icon>
                <ElTooltip
                  v-if="rule.targetDates.length > 3"
                  :content="rule.targetDates.join(', ')"
                  placement="top"
                >
                  <span>{{ formatDates(rule.targetDates) }}</span>
                </ElTooltip>
                <span v-else>{{ formatDates(rule.targetDates) }}</span>
              </span>
            </div>
          </div>
        </div>
      </ElScrollbar>
    </div>

    <!-- 空状态 -->
    <div v-else class="empty-state">
      <ElEmpty description="暂无临时规则数据" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleClose">
          关闭
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.temporary-rules-content {
  .stats-overview {
    display: flex;
    justify-content: space-around;
    padding: 16px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-radius: 12px;
    margin-bottom: 16px;

    .stat-item {
      text-align: center;
      color: white;

      .stat-value {
        font-size: 28px;
        font-weight: 700;
        line-height: 1.2;
      }

      .stat-label {
        font-size: 12px;
        opacity: 0.9;
        margin-top: 4px;
      }

      &.constraint .stat-value {
        color: #ffd666;
      }

      &.preference .stat-value {
        color: #95de64;
      }
    }
  }

  .search-bar {
    margin-bottom: 16px;

    :deep(.el-input__wrapper) {
      border-radius: 8px;
    }
  }

  .rules-scrollbar {
    .rules-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .rule-card {
      padding: 14px 16px;
      background: #fafafa;
      border-radius: 10px;
      border: 1px solid #eee;
      transition: all 0.2s ease;

      &:hover {
        background: #fff;
        border-color: #667eea;
        box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
      }

      .rule-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;

        .rule-index {
          font-size: 14px;
          color: #999;
          min-width: 24px;
        }

        .rule-icon {
          font-size: 16px;
        }

        .rule-name {
          font-weight: 600;
          font-size: 14px;
          color: #333;
          flex: 1;
        }

        .rule-tags {
          display: flex;
          gap: 6px;
        }
      }

      .rule-description {
        font-size: 13px;
        color: #666;
        line-height: 1.5;
        padding-left: 32px;
        margin-bottom: 8px;
      }

      .rule-associations {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        padding-left: 32px;
        font-size: 12px;
        color: #888;

        .association-item {
          display: inline-flex;
          align-items: center;
          gap: 4px;

          .el-icon {
            font-size: 14px;
            color: #667eea;
          }
        }
      }
    }
  }

  .empty-state {
    padding: 40px 0;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

// 深色模式支持
:global(.dark) {
  .temporary-rules-content {
    .stats-overview {
      background: linear-gradient(135deg, #4c5ea0 0%, #5a3d7a 100%);
    }

    .rule-card {
      background: #2a2a2a;
      border-color: #3a3a3a;

      &:hover {
        background: #333;
        border-color: #667eea;
      }

      .rule-name {
        color: #eee;
      }

      .rule-description {
        color: #aaa;
      }

      .rule-associations {
        color: #888;
      }
    }
  }
}
</style>
