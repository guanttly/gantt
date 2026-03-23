# 08. 前端设计

> **开发负责人**: Agent-6  
> **依赖**: Agent-5 (API 接口)  
> **技术栈**: Vue 3 + TypeScript + Vite  
> **包路径**: `frontend/web/src/`

## 1. 设计目标

在现有前端基础上新增/改造以下页面：
1. **规则录入对话框**：支持自然语言输入 + 解析预览 + 确认保存
2. **规则组织视图**：按分类展示规则树 + 依赖/冲突可视化
3. **规则依赖管理**：规则间依赖/冲突关系的 CRUD
4. **班次依赖管理**：班次间执行顺序的配置

## 2. 新增/改造页面列表

| 页面 | 路由 | 类型 | 说明 |
|------|------|------|------|
| RuleParseDialog | 组件(弹窗) | 新增 | 规则自然语言解析对话框 |
| RuleOrganizationView | `/rules/organization` | 新增 | 规则分类组织视图 |
| RuleDependencyPanel | 组件(面板) | 新增 | 规则依赖/冲突管理 |
| ShiftDependencyConfig | `/shifts/dependencies` | 新增 | 班次依赖配置 |
| RuleListPage | `/rules` | 改造 | 增加 V4 字段展示 |

## 3. 组件设计

### 3.1 RuleParseDialog 规则解析对话框

**交互流程**:

```
步骤1: 输入                    步骤2: 解析预览                步骤3: 确认保存
┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐
│ 输入规则描述:     │  解析→  │ 解析结果预览:     │  确认→  │ 保存成功!        │
│                  │         │                  │         │                  │
│ ┌──────────────┐ │         │ 名称: 早班周频次  │         │ ✅ 规则已保存     │
│ │早班每人每周  │ │         │ 类型: maxCount    │         │ ID: rule_001     │
│ │最多排3次     │ │         │ 分类: 约束-频次   │         │                  │
│ └──────────────┘ │         │ 量化: 最多3次/周  │         │ [继续添加]       │
│                  │         │ 关联: 早班(target)│         │ [关闭]           │
│ [解析] [批量]    │         │                  │         │                  │
│                  │         │ 回译: "早班每位   │         │                  │
│                  │         │ 员工每周最多排班  │         │                  │
│                  │         │ 3次"             │         │                  │
│                  │         │                  │         │                  │
│                  │         │ 验证: ✅✅✅      │         │                  │
│                  │         │                  │         │                  │
│                  │         │ [修改] [确认保存] │         │                  │
└──────────────────┘         └──────────────────┘         └──────────────────┘
```

**文件**: `src/views/rules/components/RuleParseDialog.vue`

```typescript
// Props
interface RuleParseDialogProps {
  visible: boolean
  orgId: string
  shiftOptions: ShiftOption[]   // 班次列表（供名称匹配）
  groupOptions: GroupOption[]   // 分组列表
}

// Emits
interface RuleParseDialogEmits {
  (e: 'close'): void
  (e: 'saved', rule: Rule): void
}

// 内部状态
interface ParseDialogState {
  step: 'input' | 'preview' | 'saved'
  ruleText: string
  batchMode: boolean
  batchTexts: string         // 多行文本
  loading: boolean
  parseResult: ParseResult | null
  batchResults: BatchParseResult | null
  editMode: boolean          // 用户是否在手动修改解析结果
}
```

### 3.2 RuleOrganizationView 规则组织视图

**布局**:

```
┌────────────────────────────────────────────────────────┐
│ 规则组织管理                            [添加规则] [批量] │
├───────────────────────┬────────────────────────────────┤
│                       │                                │
│  约束型 (8)           │  规则详情                       │
│  ├── 频次限制 (3)     │  ┌───────────────────────────┐ │
│  │   ├── 早班周频次   │  │ 早班周频次限制            │ │
│  │   ├── 夜班周频次   │  │                           │ │
│  │   └── 全局月频次   │  │ 类型: maxCount            │ │
│  ├── 连续性 (2)       │  │ 限制: 每周最多3次          │ │
│  │   ├── 连续排班5天  │  │ 关联: 早班 [target]       │ │
│  │   └── 连续夜班3天  │  │ 优先级: 8                 │ │
│  ├── 休息 (1)         │  │                           │ │
│  │   └── 夜班后休1天  │  │ 依赖关系:                 │ │
│  ├── 排他 (1)         │  │  → 依赖: 夜班后休息规则   │ │
│  │   └── 早夜互斥     │  │                           │ │
│  └── 禁止 (1)         │  │ 冲突声明:                 │ │
│      └── 周日禁排     │  │  ⚠ 与"排班均衡"可能冲突  │ │
│                       │  │                           │ │
│  偏好型 (3)           │  │ [编辑] [删除] [测试]      │ │
│  ├── 均衡 (1)         │  └───────────────────────────┘ │
│  └── 人员偏好 (2)     │                                │
│                       │                                │
│  依赖型 (0)           │                                │
│                       │                                │
├───────────────────────┴────────────────────────────────┤
│ 依赖关系图                                              │
│                                                        │
│   [频次限制] ──depends──> [休息规则]                     │
│       │                      │                          │
│       └──conflicts──> [均衡偏好]                        │
│                                                        │
└────────────────────────────────────────────────────────┘
```

**文件**: `src/views/rules/RuleOrganizationView.vue`

```typescript
interface RuleOrganizationState {
  organizationData: RuleOrganizationData | null
  selectedRuleId: string | null
  loading: boolean
  treeExpandedKeys: string[]
}

interface RuleOrganizationData {
  categories: Record<string, {
    label: string
    count: number
    subCategories: Record<string, {
      label: string
      rules: RuleSummary[]
    }>
  }>
  dependencies: DependencyEdge[]
  conflicts: ConflictEdge[]
  shiftDependencies: ShiftDependencyEdge[]
  statistics: Statistics
}
```

### 3.3 ShiftDependencyConfig 班次依赖配置

**文件**: `src/views/shifts/ShiftDependencyConfig.vue`

```
┌────────────────────────────────────────────────────────┐
│ 班次执行顺序配置                         [添加依赖关系] │
├────────────────────────────────────────────────────────┤
│                                                        │
│  排班执行顺序预览:                                      │
│  ┌─────┐    ┌─────┐    ┌─────┐                         │
│  │ 夜班 │ ──→│ 中班 │ ──→│ 早班 │                      │
│  │ P:1  │    │ P:2  │    │ P:3  │                      │
│  └─────┘    └─────┘    └─────┘                         │
│                                                        │
│  依赖关系列表:                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │ 夜班 → 早班  |  schedule_after  | 夜班排完再排早班 │  │
│  │ 中班 → 早班  |  schedule_after  | 中班排完再排早班 │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  ⚠ 循环依赖检测: 无循环                                │
└────────────────────────────────────────────────────────┘
```

## 4. TypeScript 类型定义

**文件**: `src/types/rule.ts`

```typescript
// V4 新增类型

// 规则分类
export type RuleCategory = 'constraint' | 'preference' | 'dependency'
export type RuleSubCategory = 
  | 'frequency' | 'continuity' | 'rest' | 'exclusive' | 'forbidden' | 'headcount'
  | 'balance' | 'personnel' | 'time'
  | 'prerequisite' | 'sequence'

// 关联角色
export type AssociationRole = 'target' | 'source' | 'reference'

// 扩展已有 Rule 类型
export interface RuleV4 extends Rule {
  category?: RuleCategory
  subCategory?: RuleSubCategory
  originalRuleId?: string
}

// 解析结果
export interface ParseResult {
  success: boolean
  ruleName: string
  ruleType: string
  category: RuleCategory
  subCategory: RuleSubCategory
  description: string
  maxCount?: number
  consecutiveMax?: number
  intervalDays?: number
  minRestDays?: number
  timeScope: string
  applyScope: string
  associationTargets: ParsedAssociation[]
  suggestedPriority: number
  backTranslation: string
  validation: ValidationStatus
}

export interface ParsedAssociation {
  type: 'shift' | 'group' | 'employee'
  name: string
  id: string
  role: AssociationRole
}

export interface ValidationStatus {
  structuralValid: boolean
  backTransValid: boolean
  simulationValid: boolean
  structuralErrors?: string[]
  simulationWarnings?: string[]
}

// 规则依赖
export interface RuleDependency {
  id: string
  ruleId: string
  dependsOnRuleId: string
  dependencyType: string
  description: string
}

// 规则冲突
export interface RuleConflict {
  id: string
  ruleAId: string
  ruleBId: string
  conflictType: string
  resolution: string
  description: string
}

// 班次依赖
export interface ShiftDependency {
  id: string
  dependentShiftId: string
  dependsOnShiftId: string
  dependencyType: string
  description: string
}
```

## 5. API 调用层

**文件**: `src/api/ruleV4.ts`

```typescript
import request from '@/utils/request'

// 解析规则
export function parseRule(data: {
  orgId: string
  ruleText: string
  shiftNames: string[]
  groupNames: string[]
}) {
  return request.post('/api/v1/rules/parse', data)
}

// 批量解析
export function parseRuleBatch(data: {
  orgId: string
  ruleTexts: string[]
  shiftNames: string[]
  groupNames: string[]
}) {
  return request.post('/api/v1/rules/parse/batch', data)
}

// 从解析结果保存规则
export function saveFromParseResult(data: {
  orgId: string
  parseResult: ParseResult
}) {
  return request.post('/api/v1/rules/from-parse', data)
}

// 规则组织视图
export function getRuleOrganization(orgId: string) {
  return request.get('/api/v1/rules/organization', { params: { orgId } })
}

// 规则依赖 CRUD
export function getRuleDependencies(ruleId: string) {
  return request.get(`/api/v1/rules/${ruleId}/dependencies`)
}

export function createRuleDependency(data: Omit<RuleDependency, 'id'>) {
  return request.post('/api/v1/rules/dependencies', data)
}

export function deleteRuleDependency(id: string) {
  return request.delete(`/api/v1/rules/dependencies/${id}`)
}

// 规则冲突 CRUD
export function getRuleConflicts(ruleId: string) {
  return request.get(`/api/v1/rules/${ruleId}/conflicts`)
}

export function createRuleConflict(data: Omit<RuleConflict, 'id'>) {
  return request.post('/api/v1/rules/conflicts', data)
}

export function deleteRuleConflict(id: string) {
  return request.delete(`/api/v1/rules/conflicts/${id}`)
}

// 班次依赖 CRUD
export function getShiftDependencies(orgId: string) {
  return request.get('/api/v1/shifts/dependencies', { params: { orgId } })
}

export function createShiftDependency(data: Omit<ShiftDependency, 'id'>) {
  return request.post('/api/v1/shifts/dependencies', data)
}

export function deleteShiftDependency(id: string) {
  return request.delete(`/api/v1/shifts/dependencies/${id}`)
}
```

## 6. 路由配置

```typescript
// 新增路由
{
  path: '/rules/organization',
  name: 'RuleOrganization',
  component: () => import('@/views/rules/RuleOrganizationView.vue'),
  meta: { title: '规则组织管理' }
},
{
  path: '/shifts/dependencies',
  name: 'ShiftDependencies',
  component: () => import('@/views/shifts/ShiftDependencyConfig.vue'),
  meta: { title: '班次依赖配置' }
}
```

## 7. 国际化

**文件**: `locales/zh-CN/ruleV4.json`

```json
{
  "rule.parse.title": "规则智能解析",
  "rule.parse.input.placeholder": "请输入规则描述，例如：早班每人每周最多排3次",
  "rule.parse.batch.placeholder": "每行输入一条规则",
  "rule.parse.btn.parse": "解析",
  "rule.parse.btn.batch": "批量解析",
  "rule.parse.preview.title": "解析结果预览",
  "rule.parse.preview.backTranslation": "系统理解",
  "rule.parse.preview.validation": "验证状态",
  "rule.parse.preview.confirm": "确认保存",
  "rule.parse.preview.edit": "手动修改",
  
  "rule.organization.title": "规则组织管理",
  "rule.organization.category.constraint": "约束型",
  "rule.organization.category.preference": "偏好型",
  "rule.organization.category.dependency": "依赖型",
  "rule.organization.dependency.graph": "依赖关系图",
  
  "rule.dependency.title": "规则依赖关系",
  "rule.conflict.title": "规则冲突声明",
  
  "shift.dependency.title": "班次执行顺序",
  "shift.dependency.preview": "执行顺序预览",
  "shift.dependency.cycle.warning": "检测到循环依赖"
}
```
