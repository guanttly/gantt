# 03. 前端规则配置 UI 设计

> **改造范围**: `frontend/web/src/pages/management/scheduling-rule/`  
> **框架**: Vue 3 + Element Plus + TypeScript  
> **原则**: 增量改造，V3 兼容代码隔离到独立文件 `v3-compat.ts`，全量迁移后一键删除

## 1. 页面结构改造总览

### 1.1 现有 vs 改造后

```
现有结构:
├── index.vue                    # 列表页
├── logic.ts                     # 业务常量
├── type.d.ts                    # 页面类型
└── components/
    ├── RuleFormDialog.vue        # 创建/编辑对话框
    └── RuleAssociationDialog.vue # 关联管理对话框

改造后结构:
├── index.vue                    # 列表页（改造: 新增分类Tab + 统计卡片）
├── logic.ts                     # 业务常量（改造: 统一枚举值）
├── type.d.ts                    # 页面类型（改造: 新增V4类型）
└── components/
    ├── RuleFormDialog.vue        # 创建/编辑对话框（改造: 新增V4字段区域）
    ├── RuleAssociationDialog.vue # 关联管理对话框（改造: 新增Role选择）
    ├── RuleParseDialog.vue       # 【新增】LLM规则解析对话框
    ├── ParseResultReview.vue     # 【新增】解析结果审核组件
    ├── RuleStatisticsCard.vue    # 【新增】规则统计概览卡片
    ├── RuleDependencyPanel.vue   # 【新增】规则依赖关系面板
    ├── RuleMigrationDialog.vue   # 【新增】V3→V4迁移对话框 (@deprecated V3: 迁移完成后删除)
    └── v3-compat.ts              # 【新增】V3 兼容映射层 (@deprecated V3: 迁移完成后删除)
```

### 1.2 页面功能布局

```
┌──────────────────────────────────────────────────────────────┐
│  排班规则管理                                    [+ LLM解析] [+ 手动新增] │
├──────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────┐    │
│  │ 统计卡片: 总计42 | 约束25 | 偏好12 | 依赖5 | 待迁移0 │    │
│  │ ("待迁移" 为过渡指标，V3 全部迁移后此指标及相关 UI 可移除)│    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  [全部] [约束型] [偏好型] [依赖型] [未分类/V3]    <- 分类Tab  │
│  ("未分类/V3" Tab 为过渡态，V3 全部迁移后移除此 Tab)          │
│                                                              │
│  ┌─ 搜索栏 ─────────────────────────────────────────────┐   │
│  │ 关键词 | 规则类型 | 应用范围 | 时间范围 | 来源 | 状态 │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─ 规则列表 ───────────────────────────────────────────┐   │
│  │ 名称 | 类型 | 分类 | 子分类 | 范围 | 优先级 | 来源     │  │
│  │       | 关联 | 状态 | 版本 | 操作                      │  │
│  └───────────────────────────────────────────────────────┘   │
│                                                              │
│  分页                                                        │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. 类型定义改造

### 2.1 全局 API 类型

**文件**: `frontend/web/src/api/scheduling-rule/model.d.ts`

```typescript
declare namespace SchedulingRule {
  // ============================================================
  // 枚举类型（统一为后端值）
  // ============================================================

  /** 规则类型 - 统一为后端枚举值 */
  type RuleType =
    | 'exclusive'         // 互斥（原 forbidden_pattern 的一部分）
    | 'combinable'        // 可组合
    | 'required_together' // 必须同排
    | 'periodic'          // 周期性
    | 'maxCount'          // 数量限制（原 max_shifts/consecutive_shifts/rest_days）
    | 'forbidden_day'     // 禁止日期
    | 'preferred'         // 偏好（原 preferred_pattern）

  /** 应用范围 - 统一为后端枚举值 */
  type ApplyScope = 'global' | 'specific'

  /** 时间范围 - 统一为后端枚举值 */
  type TimeScope = 'same_day' | 'same_week' | 'same_month' | 'custom'

  // ============================================================
  // V4 新增枚举
  // ============================================================

  /** 规则分类 */
  type Category = 'constraint' | 'preference' | 'dependency'

  /** 规则子分类 */
  type SubCategory =
    | 'forbid' | 'limit' | 'must'           // constraint 下
    | 'prefer' | 'suggest' | 'combinable'   // preference 下
    | 'source' | 'resource' | 'order'       // dependency 下

  /** 关联角色 */
  type AssociationRole = 'target' | 'source' | 'reference'

  /** 规则来源 */
  type SourceType = 'manual' | 'llm_parsed' | 'migrated'

  /** 规则版本 */
  type RuleVersion = '' | 'v3' | 'v4'

  // ============================================================
  // 核心数据结构
  // ============================================================

  /** 规则信息（V4 扩展） */
  interface RuleInfo {
    id: string
    orgId: string
    name: string
    ruleType: RuleType
    applyScope: ApplyScope
    timeScope: TimeScope
    priority: number
    isActive: boolean
    ruleData: string
    description?: string
    
    // 量化参数
    maxCount?: number
    consecutiveMax?: number
    intervalDays?: number
    minRestDays?: number
    
    // V4 新增字段
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string
    sourceType?: SourceType
    parseConfidence?: number
    version?: RuleVersion
    
    // 关联统计
    associationCount?: number
    employeeCount?: number
    shiftCount?: number
    groupCount?: number
    
    createdAt: string
    updatedAt: string
  }

  /** 查询参数（V4 扩展） */
  interface ListParams {
    orgId: string
    ruleType?: RuleType
    applyScope?: ApplyScope
    timeScope?: TimeScope
    isActive?: boolean
    keyword?: string
    category?: Category      // V4 新增
    subCategory?: SubCategory // V4 新增
    sourceType?: SourceType   // V4 新增
    version?: RuleVersion     // V4 新增
    page?: number
    size?: number
  }

  /** 列表响应 */
  interface ListData {
    items: RuleInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建规则请求（V4 扩展） */
  interface CreateRequest {
    orgId: string
    name: string
    ruleType: RuleType
    applyScope: ApplyScope
    timeScope: TimeScope
    priority: number
    ruleData: string
    description?: string
    maxCount?: number
    consecutiveMax?: number
    intervalDays?: number
    minRestDays?: number
    // V4 新增
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string
    sourceType?: SourceType
    parseConfidence?: number
    version?: RuleVersion
  }

  /** 更新规则请求（V4 扩展） */
  interface UpdateRequest {
    name?: string
    priority?: number
    ruleData?: string
    description?: string
    maxCount?: number
    consecutiveMax?: number
    intervalDays?: number
    minRestDays?: number
    // V4 新增
    category?: Category
    subCategory?: SubCategory
    version?: RuleVersion
  }

  // ============================================================
  // 关联（V4 扩展）
  // ============================================================

  /** 关联信息 */
  interface AssociationInfo {
    ruleId: string
    targetType: 'employee' | 'group' | 'shift'
    targetId: string
    targetName?: string
    role?: AssociationRole  // V4 新增
    createdAt: string
  }

  /** 创建关联请求 */
  interface CreateAssociationRequest {
    orgId: string
    ruleId: string
    targetType: 'employee' | 'group' | 'shift'
    targetId: string
    role?: AssociationRole  // V4 新增，默认 target
  }

  // ============================================================
  // LLM 规则解析（V4 新增）
  // ============================================================

  /** 解析请求 */
  interface ParseRequest {
    orgId: string
    ruleText: string
    shiftNames?: string[]
    groupNames?: string[]
  }

  /** 批量解析请求 */
  interface BatchParseRequest {
    orgId: string
    ruleTexts: string[]
    shiftNames?: string[]
    groupNames?: string[]
  }

  /** 解析响应 */
  interface ParseResponse {
    success: boolean
    results: ParseResult[]
    error?: string
  }

  /** 单条解析结果 */
  interface ParseResult {
    confidence: number
    name: string
    description: string
    ruleType: RuleType
    category: Category
    subCategory: SubCategory
    applyScope: ApplyScope
    timeScope: TimeScope
    priority: number
    maxCount?: number
    consecutiveMax?: number
    intervalDays?: number
    minRestDays?: number
    ruleData: string
    associations: ParsedAssociation[]
    backTranslation: string
    validation: ValidationResult
    originalText: string
  }

  /** 解析出的关联 */
  interface ParsedAssociation {
    type: 'shift' | 'group' | 'employee'
    name: string
    id: string
    role: AssociationRole
    matched: boolean
  }

  /** 验证结果 */
  interface ValidationResult {
    isValid: boolean
    errors?: ValidationItem[]
    warnings?: ValidationItem[]
    suggestions?: ValidationItem[]
  }

  /** 验证项 */
  interface ValidationItem {
    field: string
    code: string
    message: string
  }

  // ============================================================
  // 规则统计（V4 新增）
  // ============================================================

  /** 规则统计 */
  interface RuleStatistics {
    total: number
    activeCount: number
    inactiveCount: number
    byCategory: Record<string, number>
    byVersion: Record<string, number>
    bySourceType: Record<string, number>
    v3Count: number
    dependencyCount: number
    conflictCount: number
  }

  // ============================================================
  // 规则依赖/冲突（V4 新增）
  // ============================================================

  /** 规则依赖 */
  interface RuleDependency {
    id: string
    orgId: string
    dependentRuleId: string
    dependsOnRuleId: string
    dependencyType: 'time' | 'source' | 'resource' | 'order'
    description: string
    createdAt: string
  }

  /** 规则冲突 */
  interface RuleConflict {
    id: string
    orgId: string
    ruleId1: string
    ruleId2: string
    conflictType: 'exclusive' | 'resource' | 'time' | 'frequency'
    description: string
    resolutionPriority: number
    createdAt: string
  }

  // ============================================================
  // 迁移（V4 新增）
  // ============================================================

  /** 迁移预览 */
  interface MigrationPreview {
    totalV3Rules: number
    autoMigratable: MigrationItem[]
    needsReview: MigrationItem[]
  }

  /** 迁移项 */
  interface MigrationItem {
    ruleId: string
    ruleName: string
    currentRuleType: string
    suggestedCategory: Category
    suggestedSubCategory: SubCategory
    confidence: number
    reason: string
  }
}
```

### 2.2 页面类型

**文件**: `frontend/web/src/pages/management/scheduling-rule/type.d.ts`

```typescript
/** 分类Tab */
export type CategoryTab = 'all' | 'constraint' | 'preference' | 'dependency' | 'uncategorized'

/** 规则表单数据（V4） */
export interface RuleFormData {
  name: string
  ruleType: SchedulingRule.RuleType | ''
  applyScope: SchedulingRule.ApplyScope | ''
  timeScope: SchedulingRule.TimeScope | ''
  priority: number
  description: string
  ruleData: string
  // 量化参数
  maxCount?: number
  consecutiveMax?: number
  intervalDays?: number
  minRestDays?: number
  // V4 字段
  category: SchedulingRule.Category | ''
  subCategory: SchedulingRule.SubCategory | ''
}

/** LLM解析状态 */
export interface ParseState {
  loading: boolean
  inputText: string
  results: SchedulingRule.ParseResult[]
  selectedIndices: number[]   // 用户选中要保存的解析结果
  editingIndex: number | null // 当前正在编辑的结果索引
}
```

---

## 3. 业务常量改造

### 3.1 枚举统一

**文件**: `frontend/web/src/pages/management/scheduling-rule/logic.ts`

```typescript
// ============================================================
// V4 规则类型选项（统一为后端枚举值）
// ============================================================

/** 规则类型选项 */
export const ruleTypeOptions = [
  { label: '互斥', value: 'exclusive', description: '两个对象不能同时排班' },
  { label: '可组合', value: 'combinable', description: '可以同时排班' },
  { label: '必须同排', value: 'required_together', description: '必须一起排班' },
  { label: '周期性', value: 'periodic', description: '按周期轮换' },
  { label: '数量限制', value: 'maxCount', description: '限制排班次数/天数' },
  { label: '禁止日期', value: 'forbidden_day', description: '禁止特定日期排班' },
  { label: '偏好', value: 'preferred', description: '优先考虑但非强制' },
]

/** 应用范围选项 */
export const applyScopeOptions = [
  { label: '全局', value: 'global', description: '对所有人员/班次生效' },
  { label: '特定对象', value: 'specific', description: '需关联具体班次/分组/员工' },
]

/** 时间范围选项 */
export const timeScopeOptions = [
  { label: '同一天', value: 'same_day' },
  { label: '同一周', value: 'same_week' },
  { label: '同一月', value: 'same_month' },
  { label: '自定义', value: 'custom' },
]

// ============================================================
// V4 新增枚举选项
// ============================================================

/** 规则分类选项 */
export const categoryOptions = [
  {
    label: '约束型',
    value: 'constraint',
    description: '必须遵守的硬规则',
    color: '#F56C6C',
    icon: 'Lock',
    subCategories: [
      { label: '禁止', value: 'forbid', description: '绝对不允许' },
      { label: '限制', value: 'limit', description: '有数量/频率上限' },
      { label: '必须', value: 'must', description: '必须满足的条件' },
    ],
  },
  {
    label: '偏好型',
    value: 'preference',
    description: '尽量满足的软规则',
    color: '#E6A23C',
    icon: 'Star',
    subCategories: [
      { label: '优先', value: 'prefer', description: '优先考虑' },
      { label: '建议', value: 'suggest', description: '可以不满足' },
      { label: '可合并', value: 'combinable', description: '允许同时排' },
    ],
  },
  {
    label: '依赖型',
    value: 'dependency',
    description: '定义执行顺序或来源关系',
    color: '#409EFF',
    icon: 'Connection',
    subCategories: [
      { label: '来源依赖', value: 'source', description: '人员必须来自特定班次' },
      { label: '资源预留', value: 'resource', description: '需预留人员' },
      { label: '顺序依赖', value: 'order', description: '必须先排某班次' },
    ],
  },
]

/** 关联角色选项 */
export const associationRoleOptions = [
  { label: '约束目标', value: 'target', description: '规则作用的对象' },
  { label: '数据来源', value: 'source', description: '依赖的数据来源' },
  { label: '引用对象', value: 'reference', description: '被引用的对象（如互斥的另一方）' },
]

/** 规则来源选项 */
export const sourceTypeOptions = [
  { label: '手动创建', value: 'manual', icon: 'Edit' },
  { label: 'LLM 解析', value: 'llm_parsed', icon: 'MagicStick' },
  { label: 'V3 迁移', value: 'migrated', icon: 'Upload' },
]

// ============================================================
// 分类Tab相关
// ============================================================

/** 分类Tab选项 */
export const categoryTabs = [
  { label: '全部', value: 'all' },
  { label: '约束型', value: 'constraint', color: '#F56C6C' },
  { label: '偏好型', value: 'preference', color: '#E6A23C' },
  { label: '依赖型', value: 'dependency', color: '#409EFF' },
  { label: '未分类', value: 'uncategorized', color: '#909399' },
]

// ============================================================
// 工具函数
// ============================================================

/** 获取规则类型文本 */
export function getRuleTypeText(type: SchedulingRule.RuleType): string {
  return ruleTypeOptions.find(o => o.value === type)?.label ?? type
}

/** 获取应用范围文本 */
export function getApplyScopeText(scope: SchedulingRule.ApplyScope): string {
  return applyScopeOptions.find(o => o.value === scope)?.label ?? scope
}

/** 获取时间范围文本 */
export function getTimeScopeText(scope: SchedulingRule.TimeScope): string {
  return timeScopeOptions.find(o => o.value === scope)?.label ?? scope
}

/** 获取分类文本 */
export function getCategoryText(category: string): string {
  return categoryOptions.find(o => o.value === category)?.label ?? '未分类'
}

/** 获取分类颜色 */
export function getCategoryColor(category: string): string {
  return categoryOptions.find(o => o.value === category)?.color ?? '#909399'
}

/** 获取子分类文本 */
export function getSubCategoryText(category: string, subCategory: string): string {
  const cat = categoryOptions.find(o => o.value === category)
  if (!cat) return subCategory
  return cat.subCategories.find(s => s.value === subCategory)?.label ?? subCategory
}

/** 获取来源文本 */
export function getSourceTypeText(sourceType: string): string {
  return sourceTypeOptions.find(o => o.value === sourceType)?.label ?? '手动创建'
}

/** 获取关联角色文本 */
export function getAssociationRoleText(role: string): string {
  return associationRoleOptions.find(o => o.value === role)?.label ?? '约束目标'
}

/** 获取子分类选项（根据分类过滤） */
export function getSubCategoryOptions(category: string) {
  return categoryOptions.find(o => o.value === category)?.subCategories ?? []
}
```

---

## 4. API 层扩展

### 4.1 新增 API 方法

**文件**: `frontend/web/src/api/scheduling-rule/index.ts` (新增)

```typescript
// ============================================================
// V4 新增 API
// ============================================================

/** LLM 规则解析 */
export function parseRule(data: SchedulingRule.ParseRequest) {
  return request<SchedulingRule.ParseResponse>({
    url: `${prefix}/rules/parse`,
    method: 'post',
    data,
  })
}

/** 批量 LLM 规则解析 */
export function batchParseRules(data: SchedulingRule.BatchParseRequest) {
  return request<SchedulingRule.ParseResponse>({
    url: `${prefix}/rules/batch-parse`,
    method: 'post',
    data,
  })
}

/** 获取规则统计 */
export function getRuleStatistics(orgId: string) {
  return request<SchedulingRule.RuleStatistics>({
    url: `${prefix}/rules/statistics`,
    method: 'get',
    params: { orgId },
  })
}

/** 获取规则依赖 */
export function getRuleDependencies(orgId: string) {
  return request<{ dependencies: SchedulingRule.RuleDependency[] }>({
    url: `${prefix}/rules/dependencies`,
    method: 'get',
    params: { orgId },
  })
}

/** 创建规则依赖 */
export function createRuleDependency(data: Omit<SchedulingRule.RuleDependency, 'id' | 'createdAt'>) {
  return request({
    url: `${prefix}/rules/dependencies`,
    method: 'post',
    data,
  })
}

/** 删除规则依赖 */
export function deleteRuleDependency(id: string) {
  return request({
    url: `${prefix}/rules/dependencies/${id}`,
    method: 'delete',
  })
}

/** 获取规则冲突 */
export function getRuleConflicts(orgId: string) {
  return request<{ conflicts: SchedulingRule.RuleConflict[] }>({
    url: `${prefix}/rules/conflicts`,
    method: 'get',
    params: { orgId },
  })
}

/** 迁移预览 */
export function previewMigration(orgId: string) {
  return request<SchedulingRule.MigrationPreview>({
    url: `${prefix}/rules/migration/preview`,
    method: 'get',
    params: { orgId },
  })
}

/** 执行迁移 */
export function executeMigration(data: { orgId: string; ruleIds: string[]; autoApply: boolean }) {
  return request({
    url: `${prefix}/rules/migration/execute`,
    method: 'post',
    data,
  })
}
```

---

## 5. 核心组件设计

### 5.1 RuleParseDialog.vue - LLM 规则解析对话框

> 用户的主要入口：输入自然语言 → LLM 解析 → 审核 → 保存

```vue
<script setup lang="ts">
/**
 * RuleParseDialog - LLM 辅助规则解析对话框
 * 
 * 交互流程：
 * 1. 用户在文本框输入自然语言规则描述
 * 2. 点击"智能解析"按钮，调用 POST /v1/rules/parse
 * 3. 展示解析结果列表（每条包含结构化字段 + 回译 + 验证状态）
 * 4. 用户可以：
 *    a. 勾选要保存的规则
 *    b. 点击"编辑"修改任意字段
 *    c. 点击"重新解析"
 * 5. 确认后，逐条调用 POST /v1/scheduling-rules 创建
 */

import type { ParseResult } from './type'
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { createSchedulingRule, parseRule } from '@/api/scheduling-rule'

interface Props {
  visible: boolean
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

// 解析状态
const state = reactive({
  inputText: '',
  loading: false,
  saving: false,
  results: [] as SchedulingRule.ParseResult[],
  selectedIndices: [] as number[],
  editingIndex: null as number | null,
})

// 步骤
const currentStep = ref<'input' | 'review'>('input')

// 执行解析
async function handleParse() {
  if (!state.inputText.trim()) {
    ElMessage.warning('请输入规则描述')
    return
  }

  state.loading = true
  try {
    const resp = await parseRule({
      orgId: props.orgId,
      ruleText: state.inputText,
    })

    if (resp.success && resp.results.length > 0) {
      state.results = resp.results
      state.selectedIndices = resp.results.map((_, i) => i) // 默认全选
      currentStep.value = 'review'
    } else {
      ElMessage.error(resp.error || '解析失败，请检查规则描述')
    }
  } catch {
    ElMessage.error('解析服务异常')
  } finally {
    state.loading = false
  }
}

// 保存选中的规则
async function handleSaveSelected() {
  const selected = state.selectedIndices.map(i => state.results[i])
  if (selected.length === 0) {
    ElMessage.warning('请至少选择一条规则')
    return
  }

  state.saving = true
  let successCount = 0

  for (const result of selected) {
    try {
      await createSchedulingRule({
        orgId: props.orgId,
        name: result.name,
        ruleType: result.ruleType,
        applyScope: result.applyScope,
        timeScope: result.timeScope,
        priority: result.priority,
        ruleData: result.ruleData,
        description: result.description,
        maxCount: result.maxCount,
        consecutiveMax: result.consecutiveMax,
        intervalDays: result.intervalDays,
        minRestDays: result.minRestDays,
        // V4 字段
        category: result.category,
        subCategory: result.subCategory,
        sourceType: 'llm_parsed',
        parseConfidence: result.confidence,
        version: 'v4',
      })
      successCount++
    } catch {
      ElMessage.error(`规则"${result.name}"保存失败`)
    }
  }

  if (successCount > 0) {
    ElMessage.success(`成功保存 ${successCount} 条规则`)
    emit('success')
    handleClose()
  }

  state.saving = false
}

// 编辑某条结果
function handleEditResult(index: number) {
  state.editingIndex = index
  // 打开内嵌编辑表单
}

// 返回输入步骤
function handleBack() {
  currentStep.value = 'input'
}

// 关闭
function handleClose() {
  state.inputText = ''
  state.results = []
  state.selectedIndices = []
  state.editingIndex = null
  currentStep.value = 'input'
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="智能规则解析"
    width="900px"
    top="3vh"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 步骤一: 输入 -->
    <template v-if="currentStep === 'input'">
      <div class="parse-input-section">
        <el-alert
          title="使用自然语言描述排班规则，AI 将自动解析为结构化格式"
          description="支持一次输入多条规则，用换行或句号分隔。例如：早班每人每周最多3次，夜班和早班不能连续排"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 16px;"
        />

        <el-input
          v-model="state.inputText"
          type="textarea"
          :rows="8"
          placeholder="请输入排班规则描述...

例如：
• 早班每人每周最多排3次
• 夜班和早班不能连续排
• 尽量安排高年资护士上夜班
• 每月每人至少休息4天
• 下夜班人员必须来自上半夜班"
          maxlength="2000"
          show-word-limit
        />
      </div>
    </template>

    <!-- 步骤二: 审核 -->
    <template v-if="currentStep === 'review'">
      <div class="parse-review-section">
        <el-alert
          title="请审核以下解析结果，确认无误后保存"
          type="success"
          :closable="false"
          show-icon
          style="margin-bottom: 16px;"
        />

        <!-- 原始输入 -->
        <div class="original-text">
          <strong>原始输入：</strong>{{ state.inputText }}
        </div>

        <!-- 解析结果列表 -->
        <div
          v-for="(result, index) in state.results"
          :key="index"
          class="parse-result-card"
        >
          <div class="result-header">
            <el-checkbox
              :model-value="state.selectedIndices.includes(index)"
              @change="(val: boolean) => {
                if (val) state.selectedIndices.push(index)
                else state.selectedIndices = state.selectedIndices.filter(i => i !== index)
              }"
            />
            <span class="result-name">{{ result.name }}</span>
            <el-tag :type="result.validation?.isValid ? 'success' : 'warning'" size="small">
              {{ result.validation?.isValid ? '验证通过' : '需要确认' }}
            </el-tag>
            <el-tag type="info" size="small">
              置信度: {{ (result.confidence * 100).toFixed(0) }}%
            </el-tag>
            <el-button type="primary" link size="small" @click="handleEditResult(index)">
              编辑
            </el-button>
          </div>

          <!-- 回译对比 -->
          <div class="back-translation">
            <el-icon><InfoFilled /></el-icon>
            {{ result.backTranslation }}
          </div>

          <!-- 结构化字段展示 -->
          <div class="result-fields">
            <el-tag type="danger" effect="plain">{{ getCategoryText(result.category) }}</el-tag>
            <el-tag type="warning" effect="plain">{{ getSubCategoryText(result.category, result.subCategory) }}</el-tag>
            <el-tag effect="plain">{{ getRuleTypeText(result.ruleType) }}</el-tag>
            <el-tag type="info" effect="plain">{{ getTimeScopeText(result.timeScope) }}</el-tag>
            <el-tag type="success" effect="plain">P{{ result.priority }}</el-tag>
          </div>

          <!-- 量化参数 -->
          <div v-if="result.maxCount || result.consecutiveMax || result.intervalDays || result.minRestDays" class="result-params">
            <span v-if="result.maxCount">最大次数: {{ result.maxCount }}</span>
            <span v-if="result.consecutiveMax">连续上限: {{ result.consecutiveMax }}</span>
            <span v-if="result.intervalDays">间隔天数: {{ result.intervalDays }}</span>
            <span v-if="result.minRestDays">最少休息: {{ result.minRestDays }}天</span>
          </div>

          <!-- 关联对象 -->
          <div v-if="result.associations?.length" class="result-associations">
            <span>关联对象：</span>
            <el-tag
              v-for="assoc in result.associations"
              :key="assoc.name"
              :type="assoc.matched ? 'success' : 'danger'"
              size="small"
              effect="light"
            >
              {{ assoc.name }} ({{ getAssociationRoleText(assoc.role) }})
              <el-icon v-if="!assoc.matched" style="margin-left: 4px;"><WarningFilled /></el-icon>
            </el-tag>
          </div>

          <!-- 验证警告 -->
          <div v-if="result.validation?.warnings?.length" class="result-warnings">
            <el-alert
              v-for="warn in result.validation.warnings"
              :key="warn.code"
              :title="warn.message"
              type="warning"
              :closable="false"
              show-icon
              style="margin-top: 4px;"
            />
          </div>
        </div>
      </div>
    </template>

    <template #footer>
      <template v-if="currentStep === 'input'">
        <el-button @click="handleClose">取消</el-button>
        <el-button
          type="primary"
          :loading="state.loading"
          :disabled="!state.inputText.trim()"
          @click="handleParse"
        >
          🤖 智能解析
        </el-button>
      </template>
      <template v-if="currentStep === 'review'">
        <el-button @click="handleBack">返回修改</el-button>
        <el-button @click="handleClose">取消</el-button>
        <el-button
          type="primary"
          :loading="state.saving"
          :disabled="state.selectedIndices.length === 0"
          @click="handleSaveSelected"
        >
          保存选中 ({{ state.selectedIndices.length }})
        </el-button>
      </template>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.parse-input-section {
  min-height: 300px;
}

.parse-review-section {
  max-height: 65vh;
  overflow-y: auto;
}

.original-text {
  background: var(--el-fill-color-light);
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.parse-result-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;

  &:hover {
    border-color: var(--el-color-primary-light-5);
  }
}

.result-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.result-name {
  font-weight: 600;
  font-size: 15px;
  flex: 1;
}

.back-translation {
  background: var(--el-color-info-light-9);
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 8px;
  display: flex;
  align-items: flex-start;
  gap: 6px;
}

.result-fields {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.result-params {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  display: flex;
  gap: 16px;
  margin-bottom: 8px;
}

.result-associations {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 13px;
}
</style>
```

### 5.2 RuleFormDialog.vue 改造 - 新增 V4 字段区域

在现有表单中新增一个 **"V4 分类配置"** 区块：

```vue
<!-- 在现有 "规则配置" 区块之后新增 -->

<!-- V4 规则分类（可选） -->
<el-divider content-position="left">
  规则分类
  <el-tag type="info" size="small" style="margin-left: 8px;">V4</el-tag>
</el-divider>

<el-form-item label="规则分类" prop="category">
  <el-radio-group v-model="formData.category" @change="handleCategoryChange">
    <el-radio-button
      v-for="cat in categoryOptions"
      :key="cat.value"
      :value="cat.value"
    >
      <span :style="{ color: cat.color }">{{ cat.label }}</span>
    </el-radio-button>
  </el-radio-group>
  <div class="form-tip" v-if="formData.category">
    {{ categoryOptions.find(c => c.value === formData.category)?.description }}
  </div>
</el-form-item>

<el-form-item v-if="formData.category" label="子分类" prop="subCategory">
  <el-select v-model="formData.subCategory" placeholder="请选择子分类" style="width: 100%;">
    <el-option
      v-for="sub in getSubCategoryOptions(formData.category)"
      :key="sub.value"
      :label="sub.label"
      :value="sub.value"
    >
      <span>{{ sub.label }}</span>
      <span style="color: var(--el-text-color-secondary); font-size: 12px; margin-left: 8px;">
        {{ sub.description }}
      </span>
    </el-option>
  </el-select>
</el-form-item>

<!-- 量化参数（根据规则类型动态显示） -->
<el-divider v-if="showQuantParams" content-position="left">
  量化参数
</el-divider>

<el-form-item v-if="formData.ruleType === 'maxCount'" label="最大次数">
  <el-input-number v-model="formData.maxCount" :min="1" :max="100" style="width: 100%;" />
</el-form-item>

<el-form-item v-if="['maxCount', 'exclusive'].includes(formData.ruleType)" label="连续上限">
  <el-input-number v-model="formData.consecutiveMax" :min="1" :max="30" style="width: 100%;" />
  <div class="form-tip">连续工作最大天数</div>
</el-form-item>

<el-form-item v-if="formData.ruleType === 'periodic'" label="间隔天数">
  <el-input-number v-model="formData.intervalDays" :min="1" :max="90" style="width: 100%;" />
</el-form-item>

<el-form-item v-if="['maxCount', 'exclusive'].includes(formData.ruleType)" label="最少休息天数">
  <el-input-number v-model="formData.minRestDays" :min="0" :max="30" style="width: 100%;" />
</el-form-item>
```

### 5.3 RuleAssociationDialog.vue 改造 - 新增 Role 选择

在添加关联时新增角色选择：

```vue
<!-- 在关联目标选择之后新增 -->

<el-form-item label="关联角色">
  <el-select v-model="newAssociation.role" placeholder="选择角色" style="width: 200px;">
    <el-option
      v-for="role in associationRoleOptions"
      :key="role.value"
      :label="role.label"
      :value="role.value"
    >
      <span>{{ role.label }}</span>
      <span style="color: var(--el-text-color-secondary); font-size: 12px; margin-left: 8px;">
        {{ role.description }}
      </span>
    </el-option>
  </el-select>
  <div class="form-tip">
    target=被约束对象(默认), source=数据来源, reference=引用对象
  </div>
</el-form-item>
```

### 5.4 RuleStatisticsCard.vue - 统计概览

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getRuleStatistics } from '@/api/scheduling-rule'

interface Props {
  orgId: string
}

const props = defineProps<Props>()
const stats = ref<SchedulingRule.RuleStatistics | null>(null)
const loading = ref(false)

async function fetchStats() {
  loading.value = true
  try {
    stats.value = await getRuleStatistics(props.orgId)
  } finally {
    loading.value = false
  }
}

onMounted(fetchStats)
defineExpose({ refresh: fetchStats })
</script>

<template>
  <div v-loading="loading" class="stats-container">
    <template v-if="stats">
      <div class="stat-item">
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">总计</div>
      </div>
      <div class="stat-item" style="--stat-color: #F56C6C;">
        <div class="stat-value">{{ stats.byCategory?.constraint || 0 }}</div>
        <div class="stat-label">约束型</div>
      </div>
      <div class="stat-item" style="--stat-color: #E6A23C;">
        <div class="stat-value">{{ stats.byCategory?.preference || 0 }}</div>
        <div class="stat-label">偏好型</div>
      </div>
      <div class="stat-item" style="--stat-color: #409EFF;">
        <div class="stat-value">{{ stats.byCategory?.dependency || 0 }}</div>
        <div class="stat-label">依赖型</div>
      </div>
      <div v-if="stats.v3Count > 0" class="stat-item" style="--stat-color: #909399;">
        <div class="stat-value">{{ stats.v3Count }}</div>
        <div class="stat-label">待迁移</div>
      </div>
    </template>
  </div>
</template>

<style lang="scss" scoped>
.stats-container {
  display: flex;
  gap: 24px;
  padding: 16px 24px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  margin-bottom: 16px;
}

.stat-item {
  text-align: center;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--stat-color, var(--el-text-color-primary));
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
</style>
```

### 5.5 RuleMigrationDialog.vue - V3→V4 迁移对话框

```vue
<script setup lang="ts">
/**
 * RuleMigrationDialog - V3规则迁移到V4
 * 
 * 流程：
 * 1. 点击"迁移预览"，显示所有V3规则的推荐分类结果
 * 2. 分为"自动迁移"和"需审核"两组
 * 3. 用户可修改推荐的分类
 * 4. 确认后执行迁移
 */
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { executeMigration, previewMigration } from '@/api/scheduling-rule'

interface Props {
  visible: boolean
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const loading = ref(false)
const executing = ref(false)
const preview = ref<SchedulingRule.MigrationPreview | null>(null)

async function handlePreview() {
  loading.value = true
  try {
    preview.value = await previewMigration(props.orgId)
  } catch {
    ElMessage.error('获取迁移预览失败')
  } finally {
    loading.value = false
  }
}

async function handleExecute(autoApply: boolean) {
  if (!preview.value) return

  const ruleIds = autoApply
    ? preview.value.autoMigratable.map(r => r.ruleId)
    : [...preview.value.autoMigratable, ...preview.value.needsReview].map(r => r.ruleId)

  executing.value = true
  try {
    await executeMigration({
      orgId: props.orgId,
      ruleIds,
      autoApply,
    })
    ElMessage.success(`成功迁移 ${ruleIds.length} 条规则`)
    emit('success')
    emit('update:visible', false)
  } catch {
    ElMessage.error('迁移执行失败')
  } finally {
    executing.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="V3 规则迁移到 V4"
    width="800px"
    :close-on-click-modal="false"
    @close="$emit('update:visible', false)"
    @open="handlePreview"
  >
    <div v-loading="loading">
      <template v-if="preview">
        <el-alert
          :title="`共 ${preview.totalV3Rules} 条V3规则待迁移`"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 16px;"
        />

        <!-- 自动迁移组 -->
        <h4>✅ 可自动迁移 ({{ preview.autoMigratable.length }})</h4>
        <el-table :data="preview.autoMigratable" stripe size="small" style="margin-bottom: 16px;">
          <el-table-column prop="ruleName" label="规则名称" width="200" />
          <el-table-column prop="currentRuleType" label="当前类型" width="120" />
          <el-table-column prop="suggestedCategory" label="建议分类" width="100">
            <template #default="{ row }">
              <el-tag :color="getCategoryColor(row.suggestedCategory)" effect="dark" size="small">
                {{ getCategoryText(row.suggestedCategory) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="suggestedSubCategory" label="建议子分类" width="100" />
          <el-table-column prop="reason" label="推理依据" />
        </el-table>

        <!-- 需审核组 -->
        <h4 v-if="preview.needsReview.length">⚠️ 需人工审核 ({{ preview.needsReview.length }})</h4>
        <el-table v-if="preview.needsReview.length" :data="preview.needsReview" stripe size="small">
          <el-table-column prop="ruleName" label="规则名称" width="200" />
          <el-table-column prop="currentRuleType" label="当前类型" width="120" />
          <el-table-column label="分类" width="150">
            <template #default="{ row }">
              <el-select v-model="row.suggestedCategory" size="small">
                <el-option label="约束型" value="constraint" />
                <el-option label="偏好型" value="preference" />
                <el-option label="依赖型" value="dependency" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="需审核原因" />
        </el-table>
      </template>
    </div>

    <template #footer>
      <el-button @click="$emit('update:visible', false)">取消</el-button>
      <el-button
        v-if="preview?.autoMigratable.length"
        type="success"
        :loading="executing"
        @click="handleExecute(true)"
      >
        仅迁移可自动处理的
      </el-button>
      <el-button
        v-if="preview"
        type="primary"
        :loading="executing"
        @click="handleExecute(false)"
      >
        全部迁移
      </el-button>
    </template>
  </el-dialog>
</template>
```

---

## 6. index.vue 列表页改造

### 6.1 改造点摘要

| 位置 | 改造 | 说明 |
|------|------|------|
| 工具栏 | 新增 "LLM解析" 按钮 | 打开 RuleParseDialog |
| 工具栏 | 新增 "迁移" 按钮（条件显示） | 有 V3 规则时显示 |
| 列表上方 | 新增 RuleStatisticsCard | 统计概览 |
| 列表上方 | 新增分类 Tab | all/constraint/preference/dependency/uncategorized |
| 搜索栏 | 新增 "来源" 筛选 | sourceType 下拉 |
| 表格列 | 新增 "分类" 列 | category + subCategory Tag |
| 表格列 | 新增 "来源" 列 | sourceType Icon + 文字 |
| 表格列 | 新增 "版本" 列 | v3/v4 Badge |

### 6.2 关键代码改动

```vue
<!-- 工具栏改造 -->
<div class="action-buttons">
  <el-button type="success" @click="parseDialogVisible = true">
    🤖 LLM 解析
  </el-button>
  <el-button type="primary" :icon="Plus" @click="handleAdd">
    手动新增
  </el-button>
  <el-button
    v-if="statistics?.v3Count > 0"
    type="warning"
    @click="migrationDialogVisible = true"
  >
    迁移 V3 规则 ({{ statistics.v3Count }})
  </el-button>
</div>

<!-- 统计卡片 -->
<RuleStatisticsCard ref="statsRef" :org-id="orgId" />

<!-- 分类 Tab -->
<el-tabs v-model="activeCategory" @tab-change="handleCategoryChange">
  <el-tab-pane
    v-for="tab in categoryTabs"
    :key="tab.value"
    :label="tab.label"
    :name="tab.value"
  />
</el-tabs>

<!-- 表格新增列 -->
<el-table-column prop="category" label="分类" width="120" align="center">
  <template #default="{ row }">
    <el-tag
      v-if="row.category"
      :color="getCategoryColor(row.category)"
      effect="dark"
      size="small"
    >
      {{ getCategoryText(row.category) }}
    </el-tag>
    <el-tag v-else type="info" effect="light" size="small">
      未分类
    </el-tag>
  </template>
</el-table-column>

<el-table-column prop="sourceType" label="来源" width="100" align="center">
  <template #default="{ row }">
    <el-tag effect="plain" size="small">
      {{ getSourceTypeText(row.sourceType) }}
    </el-tag>
  </template>
</el-table-column>
```

---

## 7. i18n 国际化

### 7.1 新增翻译 key

**文件**: `frontend/web/locales/zh-CN/scheduling-rule.json` (新增)

```json
{
  "scheduling-rule": {
    "title": "排班规则管理",
    "llm-parse": "智能解析",
    "manual-add": "手动新增",
    "migrate": "迁移 V3 规则",
    "category": {
      "all": "全部",
      "constraint": "约束型",
      "preference": "偏好型",
      "dependency": "依赖型",
      "uncategorized": "未分类"
    },
    "sub-category": {
      "forbid": "禁止",
      "limit": "限制",
      "must": "必须",
      "prefer": "优先",
      "suggest": "建议",
      "combinable": "可合并",
      "source": "来源依赖",
      "resource": "资源预留",
      "order": "顺序依赖"
    },
    "source-type": {
      "manual": "手动创建",
      "llm_parsed": "AI 解析",
      "migrated": "V3 迁移"
    },
    "parse-dialog": {
      "title": "智能规则解析",
      "input-placeholder": "请输入排班规则描述...",
      "parse-btn": "智能解析",
      "save-selected": "保存选中",
      "back": "返回修改",
      "confidence": "置信度",
      "back-translation": "解析结果回译"
    }
  }
}
```

---

## 8. 改造量评估

| 文件 | 改造类型 | 预估工时 |
|------|---------|---------|
| `model.d.ts` | 重写 | 1天 |
| `logic.ts` | 重写 | 0.5天 |
| `type.d.ts` | 扩展 | 0.5天 |
| `index.ts` (API) | 扩展 | 0.5天 |
| `index.vue` | 改造 | 1天 |
| `RuleFormDialog.vue` | 改造 | 1天 |
| `RuleAssociationDialog.vue` | 改造 | 0.5天 |
| `RuleParseDialog.vue` | 新建 | 2天 |
| `ParseResultReview.vue` | 新建 | 1天 |
| `RuleStatisticsCard.vue` | 新建 | 0.5天 |
| `RuleMigrationDialog.vue` | 新建 | 1天 |
| `RuleDependencyPanel.vue` | 新建 | 1天 |
| **合计** | | **~10人天** |

---

## 9. V3 兼容隔离与清理设计

> ⚠️ **V3 兼容是过渡态，不是终态**。所有 V3 兼容代码隔离到 `v3-compat.ts`，迁移完成后删除该文件即可。

### 9.1 隔离文件设计

**文件**: `frontend/web/src/pages/management/scheduling-rule/v3-compat.ts`

```typescript
/**
 * @deprecated V3 兼容层
 * 全量迁移完成后删除此文件，并移除所有引用处的 v3Compat.xxx 调用
 * 清理条件：后端 GET /v1/rules/migration/status 返回 v3Count === 0
 */

import type { SchedulingRule } from './type'

// ---- V3 → V4 枚举映射 ----

/** @deprecated V3 */
export const V3_RULE_TYPE_MAP: Record<string, string> = {
  'max_shifts': 'maxCount',
  'consecutive_shifts': 'maxCount',
  'rest_days': 'maxCount',
  'forbidden_pattern': 'exclusive',
  'preferred_pattern': 'preferred',
}

/** @deprecated V3 */
export const V3_APPLY_SCOPE_MAP: Record<string, string> = {
  'shift': 'specific',
  'group': 'specific',
  'employee': 'specific',
}

/** @deprecated V3 */
export const V3_TIME_SCOPE_MAP: Record<string, string> = {
  'daily': 'same_day',
  'weekly': 'same_week',
  'monthly': 'same_month',
}

// ---- V3 显示文本兼容 ----

/** @deprecated V3: 合并到主映射表后移除 */
export const V3_RULE_TYPE_LABELS: Record<string, string> = {
  'max_shifts': '数量限制',
  'consecutive_shifts': '连续班次限制',
  'rest_days': '休息日要求',
  'forbidden_pattern': '互斥',
  'preferred_pattern': '偏好',
}

/** @deprecated V3 */
export const V3_APPLY_SCOPE_LABELS: Record<string, string> = {
  'shift': '特定班次',
  'group': '特定分组',
  'employee': '特定员工',
}

/** @deprecated V3 */
export const V3_TIME_SCOPE_LABELS: Record<string, string> = {
  'daily': '同一天',
  'weekly': '同一周',
  'monthly': '同一月',
}

// ---- V3 工具函数 ----

/** @deprecated V3: 判断是否为 V3 规则 */
export function isV3Rule(rule: SchedulingRule.Detail): boolean {
  return !rule.version || rule.version === 'v3'
}

/** @deprecated V3: 获取显示文本（兼容 V3 枚举值） */
export function getDisplayLabel(
  value: string,
  v4Labels: Record<string, string>,
  v3Labels: Record<string, string>,
): string {
  return v4Labels[value] ?? v3Labels[value] ?? value
}
```

### 9.2 主文件引用方式

```typescript
// logic.ts 中引用 V3 兼容（清理时删除此 import 和所有 v3Compat 引用）
import * as v3Compat from './v3-compat' // @deprecated V3

export const RULE_TYPE_LABELS: Record<string, string> = {
  'exclusive': '互斥',
  'combinable': '可组合',
  'required_together': '必须同排',
  'periodic': '周期性',
  'maxCount': '数量限制',
  'forbidden_day': '禁止日期',
  'preferred': '偏好',
  ...v3Compat.V3_RULE_TYPE_LABELS, // @deprecated V3
}
```

### 9.3 前端清理步骤（全量迁移完成后）

```
Step 1: 确认前提条件
  调用 GET /v1/rules/migration/status → v3Count === 0

Step 2: 删除兼容文件
  $ rm frontend/web/src/pages/management/scheduling-rule/v3-compat.ts

Step 3: 删除迁移组件
  $ rm frontend/web/src/pages/management/scheduling-rule/components/RuleMigrationDialog.vue

Step 4: 搜索并移除所有 @deprecated V3 引用
  $ grep -rn '@deprecated V3\|v3Compat\|v3-compat' \
      frontend/web/src/pages/management/scheduling-rule/
  → 逐个移除标记行

Step 5: 移除过渡 UI 元素
  - 移除 index.vue 中 "未分类/V3" Tab
  - 移除统计卡片中 "待迁移" 指标
  - 移除版本列（version column）

Step 6: 编译验证
  $ pnpm build
```
