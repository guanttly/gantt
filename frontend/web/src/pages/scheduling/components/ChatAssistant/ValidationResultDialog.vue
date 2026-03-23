<script setup lang="ts">
import { ElButton, ElDialog, ElEmpty, ElScrollbar, ElTag, ElTooltip } from 'element-plus'
import { computed, ref, watch } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'

// 违规项接口
interface Violation {
  ruleId: string
  ruleName: string
  date: string
  shiftId: string
  shiftName: string
  staffId?: string
  staffName?: string
  description: string
  severity: string // error/warning
}

// 警告项接口
interface Warning {
  type: string
  description: string
  suggestion?: string
  severity?: string // 'warning'=需关注的语义警告 / 'info'=一般提示
}

// 校验结果数据接口
interface ValidationResultData {
  isValid: boolean
  violations: Violation[]
  warnings: Warning[]
  summary: string
}

interface Props {
  visible: boolean
  data: ValidationResultData | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

// 搜索关键词
const searchKeyword = ref('')

// 当前选中的 Tab
const activeTab = ref<'all' | 'violations' | 'semantic' | 'info'>('all')

// 过滤后的违规列表
const filteredViolations = computed(() => {
  if (!props.data?.violations)
    return []

  const keyword = searchKeyword.value.toLowerCase().trim()
  if (!keyword)
    return props.data.violations

  return props.data.violations.filter(v =>
    v.ruleName.toLowerCase().includes(keyword)
    || v.description.toLowerCase().includes(keyword)
    || v.staffName?.toLowerCase().includes(keyword)
    || v.shiftName?.toLowerCase().includes(keyword)
    || v.date?.includes(keyword),
  )
})

// 过滤后的警告列表
const filteredWarnings = computed(() => {
  if (!props.data?.warnings)
    return []

  const keyword = searchKeyword.value.toLowerCase().trim()
  let list = props.data.warnings

  // 按 tab 过滤 severity
  if (activeTab.value === 'semantic')
    list = list.filter(w => w.severity === 'warning')
  else if (activeTab.value === 'info')
    list = list.filter(w => w.severity !== 'warning')

  // 语义警告（severity==='warning'）置顶，一般提示在后
  const sortList = (arr: Warning[]) =>
    [...arr].sort((a, b) => {
      const aW = a.severity === 'warning' ? 0 : 1
      const bW = b.severity === 'warning' ? 0 : 1
      return aW - bW
    })

  if (!keyword)
    return sortList(list)

  return sortList(list.filter(w =>
    w.description.toLowerCase().includes(keyword)
    || w.suggestion?.toLowerCase().includes(keyword)
    || w.type?.toLowerCase().includes(keyword),
  ))
})

// 统计信息
const stats = computed(() => {
  if (!props.data) {
    return {
      isValid: true,
      violationCount: 0,
      warningCount: 0,
      errorCount: 0,
    }
  }

  const errorCount = (props.data.violations || []).filter(v => v.severity === 'error').length
  const warnViolationCount = (props.data.violations || []).filter(v => v.severity !== 'error').length
  const semanticWarningCount = (props.data.warnings || []).filter(w => w.severity === 'warning').length
  const infoWarningCount = (props.data.warnings || []).filter(w => w.severity !== 'warning').length

  return {
    isValid: props.data.isValid,
    violationCount: (props.data.violations || []).length,
    warningCount: (props.data.warnings || []).length,
    semanticWarningCount,
    infoWarningCount,
    errorCount,
    warnViolationCount,
  }
})

// 获取严重程度标签
function getSeverityTag(severity: string) {
  switch (severity) {
    case 'error':
      return { text: '错误', type: 'danger' as const, icon: 'x-circle' }
    case 'warning':
      return { text: '警告', type: 'warning' as const, icon: 'warning' }
    default:
      return { text: '提示', type: 'info' as const, icon: 'info-circle' }
  }
}

// 获取警告类型标签
function getWarningTypeTag(warning: Warning) {
  // 语义规则校验不合规：显示为 warning 级
  if (warning.severity === 'warning' || warning.type === '语义规则校验') {
    return { text: '语义警告', type: 'warning' as const, icon: 'brain' }
  }
  switch (warning.type) {
    case 'shortage':
      return { text: '人员不足', type: 'warning' as const, icon: 'user' }
    case 'unchecked_rule':
      return { text: '未校验规则', type: 'info' as const, icon: 'clipboard' }
    case '工作负载均衡':
      return { text: '负载不均', type: 'info' as const, icon: 'bar-chart' }
    default:
      return { text: warning.type || '提示', type: 'info' as const, icon: 'info-circle' }
  }
}

// 监听对话框打开，重置状态
watch(() => props.visible, (newVal) => {
  if (newVal) {
    searchKeyword.value = ''
    activeTab.value = 'all'
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
    title="排班检查结果"
    width="750px"
    top="5vh"
    :before-close="handleClose"
    class="validation-result-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="validation-result-content">
      <!-- 校验状态概览 -->
      <div class="status-overview" :class="{ 'is-valid': stats.isValid, 'is-invalid': !stats.isValid }">
        <div class="status-icon">
          <svg v-if="stats.isValid" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="12" cy="12" r="10" fill="rgba(255,255,255,0.3)"/>
            <path d="M7 12.5l3.5 3.5 6.5-7" stroke="#fff" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="12" cy="12" r="10" fill="rgba(255,255,255,0.3)"/>
            <path d="M12 7v5.5" stroke="#fff" stroke-width="2.2" stroke-linecap="round"/>
            <circle cx="12" cy="16.5" r="1.2" fill="#fff"/>
          </svg>
        </div>
        <div class="status-info">
          <div class="status-title">
            {{ stats.isValid ? '校验通过' : '校验发现问题' }}
          </div>
          <div class="status-summary">
            {{ data.summary }}
          </div>
        </div>
      </div>

      <!-- 统计卡片 -->
      <div class="stats-cards">
        <div class="stat-card" :class="{ active: activeTab === 'all' }" @click="activeTab = 'all'">
          <div class="stat-value">
            {{ stats.violationCount + stats.warningCount }}
          </div>
          <div class="stat-label">
            全部项目
          </div>
        </div>
        <div class="stat-card violations" :class="{ active: activeTab === 'violations' }" @click="activeTab = 'violations'">
          <div class="stat-value">
            {{ stats.violationCount }}
          </div>
          <div class="stat-label">
            违规项
          </div>
        </div>
        <div class="stat-card semantic-warnings" :class="{ active: activeTab === 'semantic' }" @click="activeTab = 'semantic'">
          <div class="stat-value">
            {{ stats.semanticWarningCount }}
          </div>
          <div class="stat-label">
            语义警告
          </div>
        </div>
        <div class="stat-card info-warnings" :class="{ active: activeTab === 'info' }" @click="activeTab = 'info'">
          <div class="stat-value">
            {{ stats.infoWarningCount }}
          </div>
          <div class="stat-label">
            一般提示
          </div>
        </div>
      </div>

      <!-- 搜索框 -->
      <div class="search-bar">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索规则名称、描述、人员或日期..."
          clearable
          prefix-icon="Search"
        />
      </div>

      <!-- 内容列表 -->
      <ElScrollbar max-height="400px" class="result-scrollbar">
        <!-- 违规项列表 -->
        <template v-if="activeTab === 'all' || activeTab === 'violations'">
          <div v-if="filteredViolations.length > 0" class="section">
            <div class="section-title">
              <span class="section-icon"><SvgIcon name="ban" size="1em" /></span>
              <span>违规项 ({{ filteredViolations.length }})</span>
            </div>
            <div class="items-list">
              <div
                v-for="(violation, index) in filteredViolations"
                :key="`v-${index}`"
                class="result-card violation-card"
                :class="{ 'is-error': violation.severity === 'error' }"
              >
                <div class="card-header">
                  <span class="card-index">{{ index + 1 }}.</span>
                  <span class="card-icon"><SvgIcon :name="getSeverityTag(violation.severity).icon" size="1em" /></span>
                  <span class="card-title">{{ violation.ruleName }}</span>
                  <div class="card-tags">
                    <ElTag
                      :type="getSeverityTag(violation.severity).type"
                      size="small"
                      effect="light"
                    >
                      {{ getSeverityTag(violation.severity).text }}
                    </ElTag>
                  </div>
                </div>
                <div class="card-description">
                  {{ violation.description }}
                </div>
                <div class="card-meta">
                  <span v-if="violation.date" class="meta-item">
                    <SvgIcon name="calendar" size="1em" /> {{ violation.date }}
                  </span>
                  <span v-if="violation.shiftName" class="meta-item">
                    <SvgIcon name="refresh" size="1em" /> {{ violation.shiftName }}
                  </span>
                  <span v-if="violation.staffName" class="meta-item">
                    <SvgIcon name="user" size="1em" /> {{ violation.staffName }}
                  </span>
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="activeTab === 'violations'" class="empty-section">
            <ElEmpty description="没有违规项" />
          </div>
        </template>

        <!-- 警告项列表 -->
        <template v-if="activeTab === 'all' || activeTab === 'semantic' || activeTab === 'info'">
          <div v-if="filteredWarnings.length > 0" class="section">
            <div class="section-title">
              <span class="section-icon"><SvgIcon name="warning" size="1em" /></span>
              <span>警告项 ({{ filteredWarnings.length }})</span>
            </div>
            <div class="items-list">
              <div
                v-for="(warning, index) in filteredWarnings"
                :key="`w-${index}`"
                class="result-card warning-card"
                :class="{ 'is-semantic-warning': warning.severity === 'warning' }"
              >
                <div class="card-header">
                  <span class="card-index">{{ index + 1 }}.</span>
                  <span class="card-icon"><SvgIcon :name="getWarningTypeTag(warning).icon" size="1em" /></span>
                  <span class="card-title">{{ warning.description }}</span>
                  <div class="card-tags">
                    <ElTag
                      :type="getWarningTypeTag(warning).type"
                      size="small"
                      effect="light"
                    >
                      {{ getWarningTypeTag(warning).text }}
                    </ElTag>
                  </div>
                </div>
                <div v-if="warning.suggestion" class="card-suggestion">
                  <ElTooltip :content="warning.suggestion" placement="top" :show-after="300">
                    <span class="suggestion-text"><SvgIcon name="lightbulb" size="1em" /> {{ warning.suggestion }}</span>
                  </ElTooltip>
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="activeTab === 'semantic' || activeTab === 'info'" class="empty-section">
            <ElEmpty description="没有警告项" />
          </div>
        </template>

        <!-- 全部为空 -->
        <div v-if="activeTab === 'all' && filteredViolations.length === 0 && filteredWarnings.length === 0" class="empty-section">
          <ElEmpty description="没有发现任何问题" />
        </div>
      </ElScrollbar>
    </div>

    <!-- 空状态 -->
    <div v-else class="empty-state">
      <ElEmpty description="暂无检查结果数据" />
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
.validation-result-content {
  .status-overview {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 20px;
    border-radius: 12px;
    margin-bottom: 16px;

    &.is-valid {
      background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
    }

    &.is-invalid {
      background: linear-gradient(135deg, #fa8c16 0%, #d48806 100%);
    }

    .status-icon {
      width: 44px;
      height: 44px;
      flex-shrink: 0;

      svg {
        width: 100%;
        height: 100%;
      }
    }

    .status-info {
      flex: 1;

      .status-title {
        font-size: 18px;
        font-weight: 700;
        color: #fff;
        margin-bottom: 4px;
      }

      .status-summary {
        font-size: 13px;
        color: rgba(255, 255, 255, 0.9);
        line-height: 1.4;
      }
    }
  }

  .stats-cards {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;

    .stat-card {
      flex: 1;
      text-align: center;
      padding: 12px 8px;
      border-radius: 10px;
      background: #f5f5f5;
      border: 2px solid transparent;
      cursor: pointer;
      transition: all 0.2s ease;

      &:hover {
        background: #ebebeb;
      }

      &.active {
        border-color: #409eff;
        background: #ecf5ff;
      }

      &.violations {
        &.active {
          border-color: #f56c6c;
          background: #fef0f0;
        }
      }

      &.warnings {
        &.active {
          border-color: #e6a23c;
          background: #fdf6ec;
        }
      }

      &.semantic-warnings {
        &.active {
          border-color: #e6a23c;
          background: #fdf6ec;
        }
        .stat-value {
          color: #e6a23c;
        }
      }

      &.info-warnings {
        &.active {
          border-color: #909399;
          background: #f4f4f5;
        }
        .stat-value {
          color: #909399;
        }
      }

      .stat-value {
        font-size: 24px;
        font-weight: 700;
        line-height: 1.2;
        color: #333;
      }

      .stat-label {
        font-size: 12px;
        color: #999;
        margin-top: 4px;
      }
    }
  }

  .search-bar {
    margin-bottom: 16px;

    :deep(.el-input__wrapper) {
      border-radius: 8px;
    }
  }

  .result-scrollbar {
    .section {
      margin-bottom: 16px;

      .section-title {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        font-weight: 600;
        color: #333;
        margin-bottom: 10px;
        padding-bottom: 6px;
        border-bottom: 1px solid #eee;

        .section-icon {
          font-size: 16px;
        }
      }

      .items-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
      }
    }

    .result-card {
      padding: 12px 16px;
      border-radius: 10px;
      border: 1px solid #eee;
      transition: all 0.2s ease;

      &:hover {
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
      }

      &.violation-card {
        background: #fff7f7;
        border-color: #fde2e2;

        &.is-error {
          background: #fef0f0;
          border-color: #f56c6c;
          border-left: 3px solid #f56c6c;
        }

        &:hover {
          border-color: #f56c6c;
          box-shadow: 0 2px 8px rgba(245, 108, 108, 0.15);
        }
      }

      &.warning-card {
        background: #fffbe6;
        border-color: #fff1b8;

        &.is-semantic-warning {
          background: #fdf6ec;
          border-color: #e6a23c;
          border-left: 3px solid #e6a23c;
        }

        &:hover {
          border-color: #e6a23c;
          box-shadow: 0 2px 8px rgba(230, 162, 60, 0.15);
        }
      }

      .card-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 6px;

        .card-index {
          font-size: 13px;
          color: #999;
          min-width: 24px;
        }

        .card-icon {
          font-size: 14px;
        }

        .card-title {
          font-weight: 600;
          font-size: 13px;
          color: #333;
          flex: 1;
        }

        .card-tags {
          display: flex;
          gap: 6px;
        }
      }

      .card-description {
        font-size: 13px;
        color: #666;
        line-height: 1.5;
        padding-left: 32px;
        margin-bottom: 6px;
      }

      .card-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        padding-left: 32px;
        font-size: 12px;
        color: #888;

        .meta-item {
          display: inline-flex;
          align-items: center;
          gap: 2px;
        }
      }

      .card-suggestion {
        padding-left: 32px;
        margin-top: 4px;

        .suggestion-text {
          font-size: 12px;
          color: #409eff;
          cursor: help;
        }
      }
    }

    .empty-section {
      padding: 30px 0;
    }
  }
}

.empty-state {
  padding: 40px 0;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

// 深色模式支持
:global(.dark) {
  .validation-result-content {
    .status-overview {
      &.is-valid {
        background: linear-gradient(135deg, #3a8c14 0%, #2a6d0a 100%);
      }

      &.is-invalid {
        background: linear-gradient(135deg, #c77a12 0%, #a06508 100%);
      }
    }

    .stats-cards .stat-card {
      background: #2a2a2a;

      &.active {
        background: #1a2a3a;
      }

      .stat-value {
        color: #eee;
      }
    }

    .result-card {
      &.violation-card {
        background: #3a2020;
        border-color: #5a3030;

        &.is-error {
          background: #4a2020;
          border-color: #7a3030;
        }
      }

      &.warning-card {
        background: #3a3520;
        border-color: #5a5030;
      }

      .card-title {
        color: #eee !important;
      }

      .card-description {
        color: #aaa;
      }
    }

    .section-title {
      color: #eee !important;
      border-bottom-color: #3a3a3a !important;
    }
  }
}
</style>
