<script setup lang="ts">
import type { Group } from '@/api/groups'
import type { Employee } from '@/types/employee'
import type { Rule, RuleApplyScope, RuleApplyScopeInfo, RuleApplyScopeInput, RuleAssociation, RuleAssociationInput, RuleCategory, RuleTimeScope, RuleType } from '@/types/rule'
import type { Shift } from '@/types/shift'
import { Delete, Edit, MagicStick, Plus, Search, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { listEmployees } from '@/api/employees'
import { listGroups } from '@/api/groups'
import { createRule, deleteRule, getRule, listRules, updateRule } from '@/api/rules'
import { listShifts } from '@/api/shifts'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'
import { RULE_CATEGORY_OPTIONS, RULE_TYPE_OPTIONS } from '@/types/rule'
import AIParseRulesDialog from './AIParseRulesDialog.vue'

type ExtendedRuleType = RuleType | 'source' | 'order' | 'min_rest'
type PendingIssueCategory = 'config' | 'scope' | 'conflict'
type PendingFilterValue = 'all' | PendingIssueCategory
type DetailSectionKey = 'interpretation' | 'summary' | 'config' | 'associations'

const REFERENCE_CATALOG_PAGE_SIZE = 5000

interface ConfigField {
  key: string
  label: string
  value: string
}
interface RuleConfigContext {
  subjectShiftIDs: string[]
  objectShiftIDs: string[]
  targetShiftIDs: string[]
}
type ScopeOptionValue = 'all' | 'employee' | 'group' | 'exclude_employee' | 'exclude_group'

const auth = useAuthStore()
const canManageRules = computed(() => auth.hasPermission('rule:manage'))
const aiDialogVisible = ref(false)
const onlyShowPendingRules = ref(false)
const prioritizePendingIssues = ref(false)
const selectedPendingCategory = ref<PendingFilterValue>('all')

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Rule>({
  fetchFn: listRules,
})

const currentPageRuleCount = computed(() => items.value.length)
const currentPagePendingCount = computed(() => items.value.filter(rule => getRuleWarningItems(rule).length > 0).length)
const currentPagePendingBreakdown = computed(() => {
  const result: Record<PendingIssueCategory, number> = {
    config: 0,
    scope: 0,
    conflict: 0,
  }

  for (const rule of items.value) {
    const categories = new Set(getRuleWarningItems(rule).map(item => getPendingIssueCategory(item)))
    for (const category of categories)
      result[category] += 1
  }

  return result
})
const displayedRules = computed(() => {
  const filtered = items.value.filter((rule) => {
    const warningItems = getRuleWarningItems(rule)
    if (onlyShowPendingRules.value && warningItems.length === 0)
      return false
    if (selectedPendingCategory.value !== 'all' && !warningItems.some(item => getPendingIssueCategory(item) === selectedPendingCategory.value))
      return false
    return true
  })

  if (!prioritizePendingIssues.value)
    return filtered

  return filtered
    .map((rule, index) => ({ rule, index }))
    .sort((left, right) => {
      const severityDiff = getRuleMaxPendingSeverity(right.rule) - getRuleMaxPendingSeverity(left.rule)
      if (severityDiff !== 0)
        return severityDiff

      const countDiff = getRuleWarningItems(right.rule).length - getRuleWarningItems(left.rule).length
      if (countDiff !== 0)
        return countDiff

      return left.index - right.index
    })
    .map(({ rule }) => rule)
})
const displayedRuleCount = computed(() => displayedRules.value.length)
const displayedPendingCount = computed(() => displayedRules.value.filter(rule => getRuleWarningItems(rule).length > 0).length)

const categoryMap: Record<string, { label: string, type: string }> = {
  constraint: { label: '约束', type: 'danger' },
  preference: { label: '偏好', type: 'info' },
  dependency: { label: '依赖', type: 'warning' },
  hard: { label: '硬约束', type: 'danger' },
  soft: { label: '软约束', type: 'warning' },
}

const ruleTypeMap: Record<string, string> = {
  exclusive: '排他规则',
  combinable: '可组合规则',
  required_together: '必须同时规则',
  periodic: '周期规则',
  maxCount: '次数限制规则',
  forbidden_day: '禁排日期规则',
  preferred: '偏好规则',
  source: '人员来源规则',
  order: '执行顺序规则',
  min_rest: '最小休息规则',
}

const subTypeMap: Record<string, string> = {
  forbid: '排他/禁止',
  must: '必须/固定',
  limit: '数量限制',
  prefer: '偏好',
  combinable: '可组合',
  source: '人员来源',
  order: '执行顺序',
  min_rest: '最小休息',
}

const applyScopeMap: Record<string, string> = {
  global: '全局',
  specific: '特定对象',
}

const timeScopeMap: Record<string, string> = {
  same_day: '同一天',
  same_week: '同一周',
  same_month: '同一月',
  custom: '自定义',
}

const scopeTypeMap: Record<ScopeOptionValue, string> = {
  all: '所有员工',
  employee: '指定员工',
  group: '指定分组',
  exclude_employee: '排除员工',
  exclude_group: '排除分组',
}

const sourceTypeMap: Record<string, string> = {
  manual: '手动创建',
  llm_parsed: 'AI 解析',
  migrated: '迁移导入',
}

const configTypeMap: Record<string, string> = {
  exclusive_shifts: '互斥班次配置',
  max_count: '最大次数配置',
  min_rest: '最小休息配置',
  required_together: '必须同时配置',
  prefer_employee: '人员偏好配置',
  staff_source: '人员来源配置',
  execution_order: '执行顺序配置',
}

const exclusiveScopeMap: Record<string, string> = {
  same_day: '同日互斥',
  consecutive: '连续互斥',
}

const periodMap: Record<string, string> = {
  day: '按天',
  week: '按周',
  month: '按月',
}

const configKeyMap: Record<string, string> = {
  forbidden_sequence: '禁止连续排班',
  direction: '方向',
  offset: '偏移天数',
  max_count: '最大次数',
  min_count: '最小次数',
  min_rest_hours: '最少休息小时',
  min_rest_days: '最少休息天数',
  must_consecutive: '必须连续',
  priority: '优先级',
  enabled: '启用',
  dates: '禁排日期',
  weekdays: '禁排星期',
  period: '统计周期',
  max: '最大次数',
  shift_ids: '班次列表',
  shift_id: '目标班次',
  employee_ids: '员工列表',
  employee_id: '员工',
  target_shift_id: '目标班次',
  source_shift_id: '来源班次',
  before_shift_id: '前置班次',
  after_shift_id: '后置班次',
  weight: '偏好权重',
}

const configValueMap: Record<string, string> = {
  forward: '向后',
  backward: '向前',
  true: '是',
  false: '否',
}

const shiftConfigKeys = new Set(['shift_id', 'shift_ids', 'target_shift_id', 'source_shift_id', 'before_shift_id', 'after_shift_id'])
const employeeConfigKeys = new Set(['employee_id', 'employee_ids'])

const extraRuleTypeOptions = [
  { label: '人员来源规则', value: 'source' },
  { label: '执行顺序规则', value: 'order' },
  { label: '最小休息规则', value: 'min_rest' },
] as const

const ruleTypes = [...RULE_TYPE_OPTIONS, ...extraRuleTypeOptions]
const categoryOptions = RULE_CATEGORY_OPTIONS

const dialogVisible = ref(false)
const dialogTitle = ref('新增规则')
const dialogLoading = ref(false)
const submitLoading = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref()
const formConfigErrors = ref<string[]>([])

const form = reactive({
  name: '',
  type: ruleTypes[0].value as ExtendedRuleType,
  category: categoryOptions[0].value as RuleCategory,
  apply_scope: 'global' as RuleApplyScope,
  time_scope: 'same_day' as RuleTimeScope,
  time_offset_days: undefined as number | undefined,
  priority: 100,
  enabled: true,
  description: '',
})

const formConfig = ref<Record<string, unknown>>({})
const formConfigContext = ref<RuleConfigContext | null>(null)
const formApplyScopeErrors = ref<string[]>([])
const timeScopeSectionRef = ref<HTMLElement | null>(null)
const applyScopeSectionRef = ref<HTMLElement | null>(null)
const configSectionRef = ref<HTMLElement | null>(null)
const shiftFieldRef = ref<HTMLElement | null>(null)
const employeeFieldRef = ref<HTMLElement | null>(null)
const restFieldRef = ref<HTMLElement | null>(null)
const weightFieldRef = ref<HTMLElement | null>(null)
const countFieldRef = ref<HTMLElement | null>(null)
const activePendingTargetKey = ref('')

let pendingHighlightTimer: ReturnType<typeof setTimeout> | null = null
const formApplyScopeState = reactive({
  all: false,
  employeeIDs: [] as string[],
  groupIDs: [] as string[],
  excludeEmployeeIDs: [] as string[],
  excludeGroupIDs: [] as string[],
})

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailRule = ref<Rule | null>(null)
const activeDetailSection = ref<DetailSectionKey>('summary')
const activeDetailPendingItem = ref('')

let detailPendingHighlightTimer: ReturnType<typeof setTimeout> | null = null
const detailPendingItemRefs = new Map<string, HTMLElement>()

const shiftCatalog = ref<Shift[]>([])
const employeeCatalog = ref<Employee[]>([])
const groupCatalog = ref<Group[]>([])

const rules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择规则类型', trigger: 'change' }],
}

const employeeOptions = computed(() => employeeCatalog.value
  .filter(employee => employee.status !== 'inactive')
  .map(employee => ({
    label: employee.employee_no ? `${employee.name}（${employee.employee_no}）` : employee.name,
    value: employee.id,
  })))

const groupOptions = computed(() => groupCatalog.value
  .map(group => ({
    label: group.code ? `${group.name}（${group.code}）` : group.name,
    value: group.id,
  })))

const shiftSelectOptions = computed(() => shiftCatalog.value
  .filter(shift => isShiftEnabled(shift) && Boolean(shift.id))
  .map(shift => ({
    label: formatShiftOptionLabel(shift),
    value: shift.id,
  })))

const shiftLabelMap = computed<Record<string, string>>(() => {
  const result: Record<string, string> = {}
  for (const shift of shiftCatalog.value) {
    const label = formatShiftOptionLabel(shift)
    if (shift.id)
      result[shift.id] = label
    if (shift.code)
      result[shift.code] = label
    if (shift.name)
      result[shift.name] = label
  }
  return result
})

const employeeLabelMap = computed<Record<string, string>>(() => {
  const result: Record<string, string> = {}
  for (const employee of employeeCatalog.value) {
    const label = employee.employee_no ? `${employee.name}（${employee.employee_no}）` : employee.name
    if (employee.id)
      result[employee.id] = label
    if (employee.name)
      result[employee.name] = label
  }
  return result
})

const groupLabelMap = computed<Record<string, string>>(() => {
  const result: Record<string, string> = {}
  for (const group of groupCatalog.value) {
    const label = group.code ? `${group.name}（${group.code}）` : group.name
    if (group.id)
      result[group.id] = label
    if (group.name)
      result[group.name] = label
  }
  return result
})

const operationWidth = computed(() => canManageRules.value ? 220 : 90)

function getShiftDisplayValue(shift: Shift) {
  return shift.code?.trim() || shift.id
}

function isShiftEnabled(shift: Shift) {
  if (typeof shift.is_active === 'boolean')
    return shift.is_active
  if (shift.status)
    return shift.status === 'active'
  return true
}

function formatShiftOptionLabel(shift: Shift) {
  const value = getShiftDisplayValue(shift)
  return value ? `${shift.name}（${value}）` : shift.name
}

function mergeShiftOptions(extras?: string[]) {
  const merged = new Map(shiftSelectOptions.value.map(option => [option.value, option]))
  for (const value of extras || []) {
    const trimmed = value.trim()
    if (!trimmed || merged.has(trimmed))
      continue
    merged.set(trimmed, { label: formatShiftReference(trimmed), value: trimmed })
  }
  return Array.from(merged.values())
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function getTextValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function getStringArrayValue(value: unknown): string[] {
  if (!Array.isArray(value))
    return []
  return value
    .map(item => typeof item === 'string' ? item.trim() : '')
    .filter(Boolean)
}

function getNumberValue(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value))
    return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed))
      return parsed
  }
  return undefined
}

function getBooleanValue(value: unknown): boolean | undefined {
  if (typeof value === 'boolean')
    return value
  if (value === 'true')
    return true
  if (value === 'false')
    return false
  return undefined
}

function uniqueTextList(values: string[]) {
  return Array.from(new Set(values.map(value => value.trim()).filter(Boolean)))
}

function mergeSelectOptions(baseOptions: Array<{ label: string, value: string }>, extras?: string[]) {
  const merged = new Map(baseOptions.map(option => [option.value, option]))
  for (const value of extras || []) {
    const trimmed = value.trim()
    if (!trimmed || merged.has(trimmed))
      continue
    merged.set(trimmed, { label: `${trimmed}（未匹配目录）`, value: trimmed })
  }
  return Array.from(merged.values())
}

function getRuleTypeText(value?: string) {
  if (!value)
    return '-'
  return ruleTypeMap[value] || value
}

function getSubTypeText(value?: string) {
  if (!value)
    return '-'
  return subTypeMap[value] || value
}

function getApplyScopeText(value?: string) {
  if (!value)
    return '-'
  return applyScopeMap[value] || value
}

function getTimeScopeText(value?: string) {
  if (!value)
    return '-'
  return timeScopeMap[value] || value
}

function getSourceTypeText(value?: string) {
  if (!value)
    return '-'
  return sourceTypeMap[value] || value
}

function padDateTimeUnit(value: number) {
  return String(value).padStart(2, '0')
}

function formatFriendlyDateTime(value?: string) {
  if (!value)
    return '-'

  const date = new Date(value)
  if (Number.isNaN(date.getTime()))
    return value

  const year = date.getFullYear()
  const month = padDateTimeUnit(date.getMonth() + 1)
  const day = padDateTimeUnit(date.getDate())
  const hours = padDateTimeUnit(date.getHours())
  const minutes = padDateTimeUnit(date.getMinutes())
  const seconds = padDateTimeUnit(date.getSeconds())

  return `${year}年${month}月${day}日 ${hours}:${minutes}:${seconds}`
}

function isLikelyUUID(value: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value)
}

function formatReferenceLabel(value: string | undefined, labelMap: Record<string, string>, fallbackLabel: string) {
  if (!value)
    return '-'

  const trimmed = value.trim()
  if (!trimmed)
    return '-'

  return labelMap[trimmed] || (isLikelyUUID(trimmed) ? `${fallbackLabel}（未匹配目录）` : trimmed)
}

function formatShiftReference(value?: string) {
  return formatReferenceLabel(value, shiftLabelMap.value, '班次')
}

function formatShiftReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatShiftReference(value)).join('、')
}

function mergeShiftRelationIDs(...groups: Array<string[] | undefined>) {
  return uniqueTextList(groups.flatMap(group => group || []))
}

function emptyRuleConfigContext(): RuleConfigContext {
  return {
    subjectShiftIDs: [],
    objectShiftIDs: [],
    targetShiftIDs: [],
  }
}

function formatEmployeeReference(value?: string) {
  return formatReferenceLabel(value, employeeLabelMap.value, '员工')
}

function formatEmployeeReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatEmployeeReference(value)).join('、')
}

function formatGroupReference(value?: string) {
  return formatReferenceLabel(value, groupLabelMap.value, '分组')
}

function formatGroupReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatGroupReference(value)).join('、')
}

function inferConfigType(ruleType?: string) {
  switch (ruleType) {
    case 'exclusive':
    case 'combinable':
    case 'forbidden_day':
      return 'exclusive_shifts'
    case 'maxCount':
    case 'periodic':
      return 'max_count'
    case 'min_rest':
      return 'min_rest'
    case 'required_together':
      return 'required_together'
    case 'preferred':
      return 'prefer_employee'
    case 'source':
      return 'staff_source'
    case 'order':
      return 'execution_order'
    default:
      return ''
  }
}

function getConfigType(ruleType?: string, config?: Record<string, unknown>) {
  const directType = isRecord(config) ? getTextValue(config.type) : ''
  return directType || inferConfigType(ruleType)
}

function normalizeStructuredConfig(ruleType?: string, sourceConfig?: Record<string, unknown>, context?: RuleConfigContext) {
  const config = isRecord(sourceConfig) ? { ...sourceConfig } : {}
  const type = getConfigType(ruleType, config)
  const primaryShiftIDs = mergeShiftRelationIDs(context?.subjectShiftIDs, context?.targetShiftIDs)
  const objectShiftIDs = mergeShiftRelationIDs(context?.objectShiftIDs)

  if (type && !getTextValue(config.type))
    config.type = type

  switch (type) {
    case 'exclusive_shifts': {
      const shiftIDs = uniqueTextList(getStringArrayValue(config.shift_ids))
      if (shiftIDs.length) {
        config.shift_ids = shiftIDs
      }
      else if (primaryShiftIDs.length || objectShiftIDs.length) {
        config.shift_ids = mergeShiftRelationIDs(primaryShiftIDs, objectShiftIDs)
      }
      if (!getTextValue(config.scope))
        config.scope = 'same_day'
      break
    }
    case 'max_count':
      if (!getTextValue(config.shift_id) && primaryShiftIDs[0])
        config.shift_id = primaryShiftIDs[0]
      if (!getTextValue(config.period))
        config.period = 'week'
      break
    case 'required_together':
      if (!getTextValue(config.shift_id) && primaryShiftIDs[0])
        config.shift_id = primaryShiftIDs[0]
      break
    case 'prefer_employee':
      if (!getTextValue(config.shift_id) && primaryShiftIDs[0])
        config.shift_id = primaryShiftIDs[0]
      if (getNumberValue(config.weight) === undefined)
        config.weight = 100
      break
    case 'staff_source':
      if (!getTextValue(config.target_shift_id) && primaryShiftIDs[0])
        config.target_shift_id = primaryShiftIDs[0]
      if (!getTextValue(config.source_shift_id) && objectShiftIDs[0])
        config.source_shift_id = objectShiftIDs[0]
      break
    case 'execution_order':
      if (!getTextValue(config.before_shift_id) && objectShiftIDs[0])
        config.before_shift_id = objectShiftIDs[0]
      if (!getTextValue(config.after_shift_id) && primaryShiftIDs[0])
        config.after_shift_id = primaryShiftIDs[0]
      break
  }

  return config
}

function getConfigText(ruleType: string, key: string) {
  return getTextValue(normalizeStructuredConfig(ruleType, formConfig.value, formConfigContext.value || undefined)[key])
}

function getConfigStringArray(ruleType: string, key: string) {
  return getStringArrayValue(normalizeStructuredConfig(ruleType, formConfig.value, formConfigContext.value || undefined)[key])
}

function getConfigNumber(ruleType: string, key: string) {
  return getNumberValue(normalizeStructuredConfig(ruleType, formConfig.value, formConfigContext.value || undefined)[key])
}

function getConfigBoolean(ruleType: string, key: string) {
  return getBooleanValue(normalizeStructuredConfig(ruleType, formConfig.value, formConfigContext.value || undefined)[key]) ?? false
}

function getConfigTypeText(ruleType?: string, config?: Record<string, unknown>) {
  const type = getConfigType(ruleType, config)
  if (!type)
    return '-'
  return configTypeMap[type] || type
}

function formatConfigValue(value: unknown, key?: string): string {
  if (typeof value === 'boolean')
    return value ? '是' : '否'
  if (typeof value === 'string') {
    if (key && shiftConfigKeys.has(key))
      return formatShiftReference(value)
    if (key && employeeConfigKeys.has(key))
      return key === 'employee_id' ? formatEmployeeReference(value) : formatEmployeeReferences([value])
    return configValueMap[value] || value
  }
  if (Array.isArray(value)) {
    if (key && shiftConfigKeys.has(key))
      return formatShiftReferences(value.filter(item => typeof item === 'string') as string[])
    if (key && employeeConfigKeys.has(key))
      return formatEmployeeReferences(value.filter(item => typeof item === 'string') as string[])
    return value.map(item => formatConfigValue(item)).join('、')
  }
  if (value && typeof value === 'object')
    return JSON.stringify(value)
  return value === undefined || value === null ? '-' : String(value)
}

function buildGenericConfigFields(ruleType?: string, config?: Record<string, unknown>, context?: RuleConfigContext): ConfigField[] {
  const normalized = normalizeStructuredConfig(ruleType, config, context)
  return Object.entries(normalized)
    .filter(([key]) => key !== 'type')
    .map(([key, value]) => ({
      key,
      label: configKeyMap[key] || key,
      value: formatConfigValue(value, key),
    }))
}

function getConfigFields(ruleType?: string, config?: Record<string, unknown>, context?: RuleConfigContext): ConfigField[] {
  const normalized = normalizeStructuredConfig(ruleType, config, context)
  const type = getConfigType(ruleType, normalized)

  switch (type) {
    case 'exclusive_shifts': {
      const directShiftIDs = getStringArrayValue(config?.shift_ids)
      const primaryShiftIDs = mergeShiftRelationIDs(context?.subjectShiftIDs, context?.targetShiftIDs)
      const objectShiftIDs = mergeShiftRelationIDs(context?.objectShiftIDs)
      if (directShiftIDs.length > 0) {
        return [
          { key: 'shift_ids', label: '互斥班次', value: formatShiftReferences(directShiftIDs) || '-' },
          { key: 'scope', label: '互斥范围', value: exclusiveScopeMap[getTextValue(normalized.scope)] || getTextValue(normalized.scope) || '-' },
        ]
      }
      if (primaryShiftIDs.length > 0 && objectShiftIDs.length > 0) {
        return [
          { key: 'exclusive_relation', label: '互斥关系', value: `${formatShiftReferences(primaryShiftIDs)} 排斥 ${formatShiftReferences(objectShiftIDs)}` },
          { key: 'scope', label: '互斥范围', value: exclusiveScopeMap[getTextValue(normalized.scope)] || getTextValue(normalized.scope) || '-' },
        ]
      }
      return [
        { key: 'shift_ids', label: '互斥班次', value: formatShiftReferences(getStringArrayValue(normalized.shift_ids)) || '-' },
        { key: 'scope', label: '互斥范围', value: exclusiveScopeMap[getTextValue(normalized.scope)] || getTextValue(normalized.scope) || '-' },
      ]
    }
    case 'max_count':
      return [
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(normalized.shift_id)) },
        { key: 'max', label: '最大次数', value: getNumberValue(normalized.max)?.toString() || '-' },
        { key: 'period', label: '统计周期', value: periodMap[getTextValue(normalized.period)] || getTextValue(normalized.period) || '-' },
      ]
    case 'min_rest':
      return [
        { key: 'days', label: '最少休息天数', value: getNumberValue(normalized.days)?.toString() || getNumberValue(normalized.min_rest_days)?.toString() || '-' },
        { key: 'min_rest_hours', label: '最少休息小时', value: getNumberValue(normalized.min_rest_hours)?.toString() || '-' },
        { key: 'must_consecutive', label: '必须连续休息', value: getBooleanValue(normalized.must_consecutive) === undefined ? '-' : formatConfigValue(getBooleanValue(normalized.must_consecutive)) },
      ]
    case 'required_together':
      return [
        { key: 'employee_ids', label: '同时排班人员', value: formatEmployeeReferences(getStringArrayValue(normalized.employee_ids)) || '-' },
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(normalized.shift_id)) },
      ]
    case 'prefer_employee':
      return [
        { key: 'employee_id', label: '偏好员工', value: formatEmployeeReference(getTextValue(normalized.employee_id)) },
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(normalized.shift_id)) },
        { key: 'weight', label: '偏好权重', value: getNumberValue(normalized.weight)?.toString() || '-' },
      ]
    case 'staff_source':
      return [
        { key: 'target_shift_id', label: '目标班次', value: formatShiftReference(getTextValue(normalized.target_shift_id)) },
        { key: 'source_shift_id', label: '来源班次', value: formatShiftReference(getTextValue(normalized.source_shift_id)) },
      ]
    case 'execution_order':
      return [
        { key: 'before_shift_id', label: '前置班次', value: formatShiftReference(getTextValue(normalized.before_shift_id)) },
        { key: 'after_shift_id', label: '后置班次', value: formatShiftReference(getTextValue(normalized.after_shift_id)) },
      ]
    default:
      return buildGenericConfigFields(ruleType, config, context)
  }
}

function validateConfig(ruleType?: string, config?: Record<string, unknown>, context?: RuleConfigContext) {
  const normalized = normalizeStructuredConfig(ruleType, config, context)
  const type = getConfigType(ruleType, normalized)
  const errors: string[] = []

  switch (type) {
    case 'exclusive_shifts': {
      const shiftIDs = getStringArrayValue(normalized.shift_ids)
      if (shiftIDs.length < 2)
        errors.push('互斥班次至少需要 2 个班次')
      if (uniqueTextList(shiftIDs).length !== shiftIDs.length)
        errors.push('互斥班次不能重复')
      break
    }
    case 'max_count': {
      if (!getTextValue(normalized.shift_id))
        errors.push('请设置目标班次')
      if (getNumberValue(normalized.max) === undefined)
        errors.push('请设置最大次数')
      if ((getNumberValue(normalized.max) ?? 0) < 0)
        errors.push('最大次数不能小于 0')
      break
    }
    case 'min_rest': {
      const hasDays = getNumberValue(normalized.days) !== undefined || getNumberValue(normalized.min_rest_days) !== undefined
      const hasHours = getNumberValue(normalized.min_rest_hours) !== undefined
      if (!hasDays && !hasHours)
        errors.push('最小休息规则至少需要休息天数或休息小时')
      if ((getNumberValue(normalized.days) ?? getNumberValue(normalized.min_rest_days) ?? 0) < 0)
        errors.push('休息天数不能小于 0')
      if ((getNumberValue(normalized.min_rest_hours) ?? 0) < 0)
        errors.push('休息小时不能小于 0')
      break
    }
    case 'required_together': {
      const employeeIDs = getStringArrayValue(normalized.employee_ids)
      if (employeeIDs.length === 0)
        errors.push('请至少选择 1 名员工')
      if (uniqueTextList(employeeIDs).length !== employeeIDs.length)
        errors.push('同时排班人员不能重复')
      if (!getTextValue(normalized.shift_id))
        errors.push('请设置目标班次')
      break
    }
    case 'prefer_employee': {
      if (!getTextValue(normalized.employee_id))
        errors.push('请设置偏好员工')
      if (!getTextValue(normalized.shift_id))
        errors.push('请设置目标班次')
      const weight = getNumberValue(normalized.weight)
      if (weight === undefined || weight < 0 || weight > 100)
        errors.push('偏好权重必须在 0 到 100 之间')
      break
    }
    case 'staff_source':
      if (!getTextValue(normalized.target_shift_id))
        errors.push('请设置目标班次')
      if (!getTextValue(normalized.source_shift_id))
        errors.push('请设置来源班次')
      if (getTextValue(normalized.target_shift_id) && getTextValue(normalized.target_shift_id) === getTextValue(normalized.source_shift_id))
        errors.push('目标班次和来源班次不能相同')
      break
    case 'execution_order':
      if (!getTextValue(normalized.before_shift_id))
        errors.push('请设置前置班次')
      if (!getTextValue(normalized.after_shift_id))
        errors.push('请设置后置班次')
      if (getTextValue(normalized.before_shift_id) && getTextValue(normalized.before_shift_id) === getTextValue(normalized.after_shift_id))
        errors.push('前置班次和后置班次不能相同')
      break
  }

  return Array.from(new Set(errors))
}

function setFormConfigValue(key: string, value: unknown) {
  formConfig.value = {
    ...normalizeStructuredConfig(form.type, formConfig.value, formConfigContext.value || undefined),
    [key]: value,
  }
  formConfigErrors.value = validateConfig(form.type, formConfig.value, formConfigContext.value || undefined)
}

function ensureFormConfigContext() {
  if (!formConfigContext.value)
    formConfigContext.value = emptyRuleConfigContext()
  return formConfigContext.value
}

function getFormPrimaryShiftIDs() {
  return mergeShiftRelationIDs(formConfigContext.value?.subjectShiftIDs, formConfigContext.value?.targetShiftIDs)
}

function updatePrimaryShiftRelations(value: string[]) {
  const context = ensureFormConfigContext()
  context.subjectShiftIDs = uniqueTextList(value)
  context.targetShiftIDs = []
  formConfig.value = normalizeStructuredConfig(form.type, formConfig.value, context)
  formConfigErrors.value = validateConfig(form.type, formConfig.value, context)
}

function updateObjectShiftRelations(value: string[]) {
  const context = ensureFormConfigContext()
  context.objectShiftIDs = uniqueTextList(value)
  formConfig.value = normalizeStructuredConfig(form.type, formConfig.value, context)
  formConfigErrors.value = validateConfig(form.type, formConfig.value, context)
}

function updateScopeMode(value: string | number | boolean | undefined) {
  setFormConfigValue('scope', typeof value === 'string' ? value : '')
}

function updateShiftId(key: string, value: string | number | boolean | undefined) {
  setFormConfigValue(key, typeof value === 'string' ? value : '')
}

function updateNumberValue(key: string, value: number | undefined) {
  setFormConfigValue(key, value)
}

function updateRestDays(value: number | undefined) {
  setFormConfigValue('days', value)
  setFormConfigValue('min_rest_days', value)
}

function updateBooleanValue(key: string, value: boolean) {
  setFormConfigValue(key, value)
}

function updateEmployeeIds(value: string[]) {
  setFormConfigValue('employee_ids', uniqueTextList(value))
}

function updateEmployeeId(value: string | number | boolean | undefined) {
  setFormConfigValue('employee_id', typeof value === 'string' ? value.trim() : '')
}

function updateMaxCountShift(value: string | number | boolean | undefined) {
  updateShiftId('shift_id', value)
}

function updateMaxCountValue(value: number | undefined) {
  updateNumberValue('max', value)
}

function updatePeriodValue(value: string | number | boolean | undefined) {
  updateShiftId('period', value)
}

function updateRestHours(value: number | undefined) {
  updateNumberValue('min_rest_hours', value)
}

function updateMustConsecutive(value: boolean) {
  updateBooleanValue('must_consecutive', value)
}

function updateRequiredTogetherShift(value: string | number | boolean | undefined) {
  updateShiftId('shift_id', value)
}

function updatePreferredShift(value: string | number | boolean | undefined) {
  updateShiftId('shift_id', value)
}

function updatePreferredWeight(value: number | undefined) {
  updateNumberValue('weight', value)
}

function updateTargetShift(value: string | number | boolean | undefined) {
  updateShiftId('target_shift_id', value)
}

function updateSourceShift(value: string | number | boolean | undefined) {
  updateShiftId('source_shift_id', value)
}

function updateBeforeShift(value: string | number | boolean | undefined) {
  updateShiftId('before_shift_id', value)
}

function updateAfterShift(value: string | number | boolean | undefined) {
  updateShiftId('after_shift_id', value)
}

function buildShiftAssociations(associationIDs: string[], role: RuleAssociationInput['role']): RuleAssociationInput[] {
  return uniqueTextList(associationIDs).map(associationID => ({
    association_type: 'shift',
    association_id: associationID,
    role,
  }))
}

function buildAssociationInputs(ruleType: ExtendedRuleType, config: Record<string, unknown>, context?: RuleConfigContext | null) {
  const normalized = normalizeStructuredConfig(ruleType, config, context || undefined)
  const type = getConfigType(ruleType, normalized)
  const primaryShiftIDs = mergeShiftRelationIDs(context?.subjectShiftIDs, context?.targetShiftIDs)
  const objectShiftIDs = mergeShiftRelationIDs(context?.objectShiftIDs)

  switch (type) {
    case 'exclusive_shifts': {
      if (primaryShiftIDs.length > 0 || objectShiftIDs.length > 0)
        return [...buildShiftAssociations(primaryShiftIDs, 'subject'), ...buildShiftAssociations(objectShiftIDs, 'object')]
      const symmetricShiftIDs = getStringArrayValue(normalized.shift_ids)
      return buildShiftAssociations(symmetricShiftIDs, 'subject')
    }
    case 'max_count':
      return buildShiftAssociations(getTextValue(normalized.shift_id) ? [getTextValue(normalized.shift_id)] : [], 'target')
    case 'required_together':
      return buildShiftAssociations(getTextValue(normalized.shift_id) ? [getTextValue(normalized.shift_id)] : [], 'target')
    case 'prefer_employee':
      return buildShiftAssociations(getTextValue(normalized.shift_id) ? [getTextValue(normalized.shift_id)] : [], 'target')
    case 'staff_source':
      return [
        ...buildShiftAssociations(getTextValue(normalized.target_shift_id) ? [getTextValue(normalized.target_shift_id)] : [], 'target'),
        ...buildShiftAssociations(getTextValue(normalized.source_shift_id) ? [getTextValue(normalized.source_shift_id)] : [], 'object'),
      ]
    case 'execution_order':
      return [
        ...buildShiftAssociations(getTextValue(normalized.after_shift_id) ? [getTextValue(normalized.after_shift_id)] : [], 'target'),
        ...buildShiftAssociations(getTextValue(normalized.before_shift_id) ? [getTextValue(normalized.before_shift_id)] : [], 'object'),
      ]
    default:
      return undefined
  }
}

function handleRuleTypeChange(value: ExtendedRuleType) {
  form.type = value
  if (!formConfigContext.value)
    formConfigContext.value = emptyRuleConfigContext()
  formConfig.value = normalizeStructuredConfig(value, formConfig.value, formConfigContext.value || undefined)
  formConfigErrors.value = validateConfig(value, formConfig.value, formConfigContext.value || undefined)
}

async function ensureCatalogs() {
  const tasks: Promise<unknown>[] = []

  if (shiftCatalog.value.length === 0) {
    tasks.push(
      listShifts({ page: 1, page_size: REFERENCE_CATALOG_PAGE_SIZE }).then((response) => {
        const items = Array.isArray(response) ? response : response.items || []
        shiftCatalog.value = items
      }),
    )
  }

  if (employeeCatalog.value.length === 0) {
    tasks.push(
      listEmployees({ page: 1, page_size: REFERENCE_CATALOG_PAGE_SIZE }).then((response) => {
        employeeCatalog.value = response.items || []
      }),
    )
  }

  if (groupCatalog.value.length === 0) {
    tasks.push(
      Promise.resolve(listGroups({ page: 1, page_size: REFERENCE_CATALOG_PAGE_SIZE })).then((response) => {
        groupCatalog.value = Array.isArray(response) ? response : response.items || []
      }),
    )
  }

  await Promise.all(tasks)
}

onMounted(() => {
  void ensureCatalogs()
})

function resetForm() {
  editingId.value = null
  Object.assign(form, {
    name: '',
    type: ruleTypes[0].value,
    category: categoryOptions[0].value,
    apply_scope: 'global',
    time_scope: 'same_day',
    time_offset_days: undefined,
    priority: 100,
    enabled: true,
    description: '',
  })
  formConfigContext.value = emptyRuleConfigContext()
  resetFormApplyScopeState()
  formConfig.value = normalizeStructuredConfig(ruleTypes[0].value, {}, formConfigContext.value)
  formConfigErrors.value = []
  formApplyScopeErrors.value = []
}

async function handleAdd() {
  resetForm()
  dialogTitle.value = '新增规则'
  dialogLoading.value = true
  dialogVisible.value = true
  try {
    await ensureCatalogs()
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '加载规则配置目录失败')
  }
  finally {
    dialogLoading.value = false
  }
}

async function handleEdit(row: Rule, pendingItem?: string) {
  editingId.value = row.id
  dialogTitle.value = '编辑规则'
  dialogLoading.value = true
  dialogVisible.value = true

  try {
    const [detail] = await Promise.all([getRule(row.id), ensureCatalogs()])
    formConfigContext.value = buildRuleConfigContext(detail)
    Object.assign(form, {
      name: detail.name,
      type: (detail.type || row.type) as ExtendedRuleType,
      category: detail.category,
      apply_scope: detail.apply_scope,
      time_scope: detail.time_scope,
      time_offset_days: detail.time_offset_days,
      priority: detail.priority,
      enabled: detail.enabled,
      description: detail.description || '',
    })
    hydrateFormApplyScopes(detail.apply_scopes)
    formConfig.value = normalizeStructuredConfig(detail.type, detail.config, formConfigContext.value)
    formConfigErrors.value = validateConfig(detail.type, formConfig.value, formConfigContext.value)
    syncFormApplyScopeErrors()
    if (pendingItem)
      await scrollToPendingItem(pendingItem)
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '加载规则详情失败')
    dialogVisible.value = false
  }
  finally {
    dialogLoading.value = false
  }
}

function setDetailPendingItemRef(item: string, el: HTMLElement | null) {
  if (el)
    detailPendingItemRefs.set(item, el)
  else
    detailPendingItemRefs.delete(item)
}

function getDefaultDetailSection(rule?: Rule | null): DetailSectionKey {
  if (rule && getRuleWarningItems(rule).length > 0)
    return 'interpretation'
  return 'summary'
}

function highlightDetailPendingItem(item: string) {
  activeDetailPendingItem.value = item
  if (detailPendingHighlightTimer)
    clearTimeout(detailPendingHighlightTimer)
  detailPendingHighlightTimer = setTimeout(() => {
    activeDetailPendingItem.value = ''
    detailPendingHighlightTimer = null
  }, 1800)
}

async function focusDetailPendingItem(item: string) {
  if (!item)
    return

  activeDetailSection.value = 'interpretation'
  await nextTick()
  const target = detailPendingItemRefs.get(item)
  target?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  highlightDetailPendingItem(item)
}

async function handleView(row: Rule, pendingItem?: string) {
  detailVisible.value = true
  detailLoading.value = true
  detailRule.value = null
  detailPendingItemRefs.clear()
  activeDetailSection.value = 'summary'
  activeDetailPendingItem.value = ''

  try {
    const [detail] = await Promise.all([getRule(row.id), ensureCatalogs()])
    detailRule.value = detail
    activeDetailSection.value = getDefaultDetailSection(detail)
    if (pendingItem)
      await focusDetailPendingItem(pendingItem)
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '加载规则详情失败')
    detailVisible.value = false
  }
  finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  detailVisible.value = false
  detailRule.value = null
  activeDetailSection.value = 'summary'
  activeDetailPendingItem.value = ''
  detailPendingItemRefs.clear()
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }

  const normalizedConfig = normalizeStructuredConfig(form.type, formConfig.value, formConfigContext.value || undefined)
  const configErrors = validateConfig(form.type, normalizedConfig, formConfigContext.value || undefined)
  formConfigErrors.value = configErrors
  if (configErrors.length > 0) {
    ElMessage.error(configErrors[0])
    return
  }

  const scopeErrors = validateFormApplyScopes()
  formApplyScopeErrors.value = scopeErrors
  if (scopeErrors.length > 0) {
    ElMessage.error(scopeErrors[0])
    return
  }

  submitLoading.value = true
  try {
    const associations = buildAssociationInputs(form.type, normalizedConfig, formConfigContext.value)
    const applyScopes = buildFormApplyScopes()
    const payload = {
      ...form,
      config: normalizedConfig,
      ...(associations ? { associations } : {}),
      apply_scopes: applyScopes,
    } as any

    if (editingId.value) {
      await updateRule(editingId.value, payload)
      ElMessage.success('更新成功')
    }
    else {
      await createRule(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    refresh()
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '操作失败')
  }
  finally {
    submitLoading.value = false
  }
}

async function handleDelete(row: Rule) {
  await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteRule(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '删除失败')
  }
}

function getAssociationIDs(rule: Rule | null, associationType: RuleAssociation['association_type'], role?: RuleAssociation['role']) {
  return (rule?.associations || [])
    .filter((item) => {
      if (item.association_type !== associationType)
        return false
      if (!role)
        return true
      if (role === 'target')
        return item.role === 'target' || !item.role
      return item.role === role
    })
    .map(item => item.association_id)
    .filter(Boolean)
}

function buildRuleConfigContext(rule: Rule | null): RuleConfigContext {
  return {
    subjectShiftIDs: getAssociationIDs(rule, 'shift', 'subject'),
    objectShiftIDs: getAssociationIDs(rule, 'shift', 'object'),
    targetShiftIDs: getAssociationIDs(rule, 'shift', 'target'),
  }
}

function getAssociatedEmployees(rule: Rule | null) {
  return uniqueTextList((rule?.associations || [])
    .filter(item => item.association_type === 'employee')
    .map(item => item.association_id)
    .filter(Boolean))
}

function getAssociatedGroups(rule: Rule | null) {
  return uniqueTextList((rule?.associations || [])
    .filter(item => item.association_type === 'group')
    .map(item => item.association_id)
    .filter(Boolean))
}

function formatScopeDetail(scope: RuleApplyScopeInfo) {
  const label = scopeTypeMap[scope.scope_type as ScopeOptionValue] || scope.scope_type
  if (scope.scope_name)
    return `${label}：${scope.scope_name}`
  if (scope.scope_id) {
    if (scope.scope_type === 'employee' || scope.scope_type === 'exclude_employee')
      return `${label}：${formatEmployeeReference(scope.scope_id)}`
    if (scope.scope_type === 'group' || scope.scope_type === 'exclude_group')
      return `${label}：${formatGroupReference(scope.scope_id)}`
  }
  return label
}

function formatApplyScopes(rule: Rule | null) {
  if (!rule)
    return '-'
  if (rule.apply_scopes?.length)
    return rule.apply_scopes.map(scope => formatScopeDetail(scope)).join('；')
  return getApplyScopeText(rule.apply_scope)
}

function formatApplyScopeSummaryByValues(applyScope?: string, scopes: Array<Pick<RuleApplyScopeInfo, 'scope_type' | 'scope_id' | 'scope_name'>> = []) {
  if (scopes.length === 0) {
    if (applyScope === 'global')
      return '适用于所有员工的排班约束'
    if (applyScope === 'specific')
      return '仅适用于指定对象，但尚未补充明细范围'
    return getApplyScopeText(applyScope)
  }

  const allScopes = scopes.filter(scope => scope.scope_type === 'all')
  const includeEmployees = uniqueTextList(scopes.filter(scope => scope.scope_type === 'employee').map(scope => scope.scope_id || scope.scope_name || '').filter(Boolean))
  const includeGroups = uniqueTextList(scopes.filter(scope => scope.scope_type === 'group').map(scope => scope.scope_id || scope.scope_name || '').filter(Boolean))
  const excludeEmployees = uniqueTextList(scopes.filter(scope => scope.scope_type === 'exclude_employee').map(scope => scope.scope_id || scope.scope_name || '').filter(Boolean))
  const excludeGroups = uniqueTextList(scopes.filter(scope => scope.scope_type === 'exclude_group').map(scope => scope.scope_id || scope.scope_name || '').filter(Boolean))

  const parts: string[] = []

  if (allScopes.length > 0)
    parts.push('适用于所有员工')
  if (includeEmployees.length > 0)
    parts.push(`仅适用于员工 ${formatEmployeeReferences(includeEmployees)}`)
  if (includeGroups.length > 0)
    parts.push(`适用于分组 ${formatGroupReferences(includeGroups)}`)
  if (excludeEmployees.length > 0)
    parts.push(`排除员工 ${formatEmployeeReferences(excludeEmployees)}`)
  if (excludeGroups.length > 0)
    parts.push(`排除分组 ${formatGroupReferences(excludeGroups)}`)

  return joinReadableParts(parts, '适用范围尚未配置完整')
}

function joinReadableParts(parts: string[], fallback = '-') {
  return parts.filter(Boolean).join('；') || fallback
}

function formatApplyScopeSummary(rule: Rule | null) {
  if (!rule)
    return '-'
  return formatApplyScopeSummaryByValues(rule.apply_scope, rule.apply_scopes || [])
}

function formatAssociationSummary(rule: Rule | null) {
  if (!rule)
    return '-'

  const subjectShifts = getAssociationIDs(rule, 'shift', 'subject')
  const objectShifts = getAssociationIDs(rule, 'shift', 'object')
  const targetShifts = getAssociationIDs(rule, 'shift', 'target')
  const employees = getAssociatedEmployees(rule)
  const groups = getAssociatedGroups(rule)

  const parts: string[] = []

  if (subjectShifts.length > 0)
    parts.push(`主体班次为 ${formatShiftReferences(subjectShifts)}`)
  if (objectShifts.length > 0)
    parts.push(`客体班次为 ${formatShiftReferences(objectShifts)}`)
  if (targetShifts.length > 0)
    parts.push(`目标班次为 ${formatShiftReferences(targetShifts)}`)
  if (employees.length > 0)
    parts.push(`直接关联员工 ${formatEmployeeReferences(employees)}`)
  if (groups.length > 0)
    parts.push(`直接关联分组 ${formatGroupReferences(groups)}`)

  return joinReadableParts(parts, '当前没有额外关联对象')
}

function formatTimeScopeSummaryByValues(timeScope?: string, timeOffsetDays?: number) {
  if (timeScope === 'custom') {
    const offset = typeof timeOffsetDays === 'number' ? `，偏移 ${timeOffsetDays} 天` : ''
    return `规则按自定义时间窗口判断${offset}`
  }

  const timeScopeText = getTimeScopeText(timeScope)
  return timeScopeText === '-' ? '-' : `规则按${timeScopeText}判断`
}

function formatTimeScopeSummary(rule: Rule | null) {
  if (!rule)
    return '-'

  return formatTimeScopeSummaryByValues(rule.time_scope, rule.time_offset_days)
}

function formatRuleInterpretation(rule: Rule | null) {
  if (!rule)
    return []

  const context = buildRuleConfigContext(rule)
  const summary = formatRuleSemanticSummary(rule.type, rule.config, context)
  const timeScopeSummary = formatTimeScopeSummary(rule)
  const applyScopeSummary = formatApplyScopeSummary(rule)
  const associationSummary = formatAssociationSummary(rule)

  const parts = [
    summary,
    timeScopeSummary,
    applyScopeSummary,
    associationSummary === '当前没有额外关联对象' ? '' : associationSummary,
    rule.enabled ? '' : '当前规则处于停用状态，不会参与排班计算',
  ]

  return parts.filter(Boolean)
}

function formatRuleListSecondarySummary(rule: Rule) {
  const parts = [
    formatTimeScopeSummary(rule),
    formatApplyScopeSummary(rule),
    rule.enabled ? '' : '已停用',
  ]

  return joinReadableParts(parts, '-')
}

function validateApplyScopeValues(applyScope?: string, scopes: Array<Pick<RuleApplyScopeInfo, 'scope_type' | 'scope_id' | 'scope_name'>> = []) {
  const errors: string[] = []

  if (applyScope === 'specific') {
    const includeCount = scopes.filter(scope => scope.scope_type === 'all' || scope.scope_type === 'employee' || scope.scope_type === 'group').length
    const excludeCount = scopes.filter(scope => scope.scope_type === 'exclude_employee' || scope.scope_type === 'exclude_group').length

    if (scopes.length === 0)
      errors.push('适用范围缺少指定对象')
    if (includeCount === 0 && excludeCount > 0)
      errors.push('适用范围仅排除了对象，但没有设置命中范围')
  }

  return Array.from(new Set(errors))
}

function getRuleWarningItems(rule: Rule) {
  const configErrors = validateConfig(rule.type, rule.config, buildRuleConfigContext(rule))
  const scopeErrors = validateApplyScopeValues(rule.apply_scope, rule.apply_scopes || [])
  return Array.from(new Set([...configErrors, ...scopeErrors]))
}

function getPendingIssueCategory(item: string): PendingIssueCategory {
  if (item.includes('适用范围') || item.includes('适用对象') || item.includes('排除对象') || item.includes('命中范围'))
    return 'scope'
  if (item.includes('不能相同') || item.includes('不能重复') || item.includes('必须在'))
    return 'conflict'
  return 'config'
}

function getPendingIssueMeta(item: string) {
  const category = getPendingIssueCategory(item)
  if (category === 'scope') {
    return {
      category,
      label: '范围问题',
      type: 'warning',
    } as const
  }
  if (category === 'conflict') {
    return {
      category,
      label: '逻辑冲突',
      type: 'danger',
    } as const
  }
  return {
    category,
    label: '配置缺失',
    type: 'info',
  } as const
}

function getPendingIssueSeverity(item: string) {
  const category = getPendingIssueCategory(item)
  if (category === 'conflict')
    return 3
  if (category === 'scope')
    return 2
  return 1
}

function getRuleMaxPendingSeverity(rule: Rule) {
  return getRuleWarningItems(rule).reduce((max, item) => Math.max(max, getPendingIssueSeverity(item)), 0)
}

function getRulePrimaryPendingItem(rule: Rule) {
  return getRuleWarningItems(rule)
    .slice()
    .sort((left, right) => getPendingIssueSeverity(right) - getPendingIssueSeverity(left))[0] || ''
}

function getPendingIssueSuggestion(item: string) {
  if (item.includes('适用范围') || item.includes('适用对象') || item.includes('排除对象') || item.includes('命中范围'))
    return '建议先补齐适用对象，再确认是否需要排除范围。'
  if (item.includes('时间范围') || item.includes('偏移天数') || item.includes('自定义'))
    return '建议补齐时间范围或偏移天数，避免规则命中周期不明确。'
  if (item.includes('不能相同') || item.includes('不能重复') || item.includes('必须在'))
    return '建议先检查主体、客体和时间边界是否互相冲突。'
  if (item.includes('班次'))
    return '建议补充主体班次或客体班次，先把规则作用对象说清楚。'
  if (item.includes('员工') || item.includes('排班人员'))
    return '建议补充员工或人员来源，避免规则无法落到具体人群。'
  if (item.includes('休息'))
    return '建议补齐最少休息时长或间隔要求。'
  if (item.includes('权重'))
    return '建议补充偏好权重，避免偏好规则没有执行力度。'
  if (item.includes('最大次数'))
    return '建议补充最大次数限制，避免约束上限缺失。'

  const category = getPendingIssueCategory(item)
  if (category === 'conflict')
    return '建议先处理逻辑冲突，再继续补其它配置。'
  if (category === 'scope')
    return '建议先补齐适用范围，确认规则到底管哪些对象。'
  return '建议先补齐核心配置，保证规则可解释、可执行。'
}

function getPendingIssueActionText(item: string) {
  const targetKey = getPendingTargetKey(item)
  if (targetKey === 'time_scope')
    return '去补时间范围'
  if (targetKey === 'apply_scope')
    return '去补适用范围'
  if (targetKey === 'shift')
    return '去补班次配置'
  if (targetKey === 'employee')
    return '去补人员配置'
  if (targetKey === 'rest')
    return '去补休息设置'
  if (targetKey === 'weight')
    return '去补偏好权重'
  if (targetKey === 'count')
    return '去补次数限制'
  return '去补规则配置'
}

function setPendingCategoryFilter(category: PendingFilterValue) {
  selectedPendingCategory.value = category
}

function togglePendingCategoryFilter(category: PendingFilterValue) {
  selectedPendingCategory.value = selectedPendingCategory.value === category ? 'all' : category
}

function getRuleWarningPreviewItems(rule: Rule) {
  const items = getRuleWarningItems(rule)
  return items.slice(0, 2)
}

async function handleRuleWarningShortcut(row: Rule, item?: string) {
  if (canManageRules.value) {
    await handleEdit(row, item)
    return
  }
  await handleView(row, item)
}

function resetFormApplyScopeState() {
  formApplyScopeState.all = false
  formApplyScopeState.employeeIDs = []
  formApplyScopeState.groupIDs = []
  formApplyScopeState.excludeEmployeeIDs = []
  formApplyScopeState.excludeGroupIDs = []
}

function buildFormApplyScopes(): RuleApplyScopeInput[] {
  if (form.apply_scope !== 'specific')
    return []

  const scopes: RuleApplyScopeInput[] = []

  if (formApplyScopeState.all)
    scopes.push({ scope_type: 'all' })
  for (const scopeID of uniqueTextList(formApplyScopeState.employeeIDs))
    scopes.push({ scope_type: 'employee', scope_id: scopeID })
  for (const scopeID of uniqueTextList(formApplyScopeState.groupIDs))
    scopes.push({ scope_type: 'group', scope_id: scopeID })
  for (const scopeID of uniqueTextList(formApplyScopeState.excludeEmployeeIDs))
    scopes.push({ scope_type: 'exclude_employee', scope_id: scopeID })
  for (const scopeID of uniqueTextList(formApplyScopeState.excludeGroupIDs))
    scopes.push({ scope_type: 'exclude_group', scope_id: scopeID })

  return scopes
}

function validateFormApplyScopes() {
  return validateApplyScopeValues(form.apply_scope, buildFormApplyScopes()).map((message) => {
    if (message === '适用范围缺少指定对象')
      return '请至少设置一种适用对象'
    if (message === '适用范围仅排除了对象，但没有设置命中范围')
      return '仅排除对象时，请先勾选“先匹配所有员工”或补充指定对象'
    return message
  })
}

function syncFormApplyScopeErrors() {
  formApplyScopeErrors.value = validateFormApplyScopes()
}

function hydrateFormApplyScopes(scopes?: RuleApplyScopeInfo[]) {
  resetFormApplyScopeState()
  for (const scope of scopes || []) {
    switch (scope.scope_type) {
      case 'all':
        formApplyScopeState.all = true
        break
      case 'employee':
        formApplyScopeState.employeeIDs = uniqueTextList([...formApplyScopeState.employeeIDs, scope.scope_id || scope.scope_name || ''])
        break
      case 'group':
        formApplyScopeState.groupIDs = uniqueTextList([...formApplyScopeState.groupIDs, scope.scope_id || scope.scope_name || ''])
        break
      case 'exclude_employee':
        formApplyScopeState.excludeEmployeeIDs = uniqueTextList([...formApplyScopeState.excludeEmployeeIDs, scope.scope_id || scope.scope_name || ''])
        break
      case 'exclude_group':
        formApplyScopeState.excludeGroupIDs = uniqueTextList([...formApplyScopeState.excludeGroupIDs, scope.scope_id || scope.scope_name || ''])
        break
    }
  }
}

function formatFormApplyScopeSummary() {
  return formatApplyScopeSummaryByValues(form.apply_scope, buildFormApplyScopes())
}

function formatFormTimeScopeSummary() {
  return formatTimeScopeSummaryByValues(form.time_scope, form.time_offset_days)
}

function updateFormApplyScope(value: RuleApplyScope) {
  form.apply_scope = value
  syncFormApplyScopeErrors()
}

function updateFormTimeScope(value: RuleTimeScope) {
  form.time_scope = value
  if (value !== 'custom')
    form.time_offset_days = undefined
}

function updateFormTimeOffsetDays(value: number | undefined) {
  form.time_offset_days = value
}

function updateFormScopeAll(value: boolean) {
  formApplyScopeState.all = value
  syncFormApplyScopeErrors()
}

function updateFormScopeList(key: 'employeeIDs' | 'groupIDs' | 'excludeEmployeeIDs' | 'excludeGroupIDs', value: string[]) {
  formApplyScopeState[key] = uniqueTextList(value)
  syncFormApplyScopeErrors()
}

function updateFormEmployeeScopes(value: string[]) {
  updateFormScopeList('employeeIDs', value)
}

function updateFormGroupScopes(value: string[]) {
  updateFormScopeList('groupIDs', value)
}

function updateFormExcludeEmployeeScopes(value: string[]) {
  updateFormScopeList('excludeEmployeeIDs', value)
}

function updateFormExcludeGroupScopes(value: string[]) {
  updateFormScopeList('excludeGroupIDs', value)
}

function formatFormAssociationSummary() {
  const context = formConfigContext.value
  if (!context)
    return '当前没有额外关联对象'

  const parts: string[] = []
  const primaryShifts = mergeShiftRelationIDs(context.subjectShiftIDs, context.targetShiftIDs)

  if (primaryShifts.length > 0)
    parts.push(`主体/目标班次为 ${formatShiftReferences(primaryShifts)}`)
  if (context.objectShiftIDs.length > 0)
    parts.push(`客体班次为 ${formatShiftReferences(context.objectShiftIDs)}`)

  return joinReadableParts(parts, '当前没有额外关联对象')
}

function formatFormInterpretation() {
  const summary = formatRuleSemanticSummary(form.type, formConfig.value, formConfigContext.value || undefined)
  const timeScopeSummary = formatFormTimeScopeSummary()
  const applyScopeSummary = formatFormApplyScopeSummary()
  const associationSummary = formatFormAssociationSummary()

  return [
    summary,
    timeScopeSummary,
    applyScopeSummary,
    associationSummary === '当前没有额外关联对象' ? '' : associationSummary,
    form.enabled ? '保存后将作为启用规则参与排班计算' : '保存后将保持停用状态，不参与排班计算',
  ].filter(Boolean)
}

function getFormPendingItems() {
  return Array.from(new Set([...formConfigErrors.value, ...formApplyScopeErrors.value]))
}

function getPendingTargetKey(item: string) {
  if (item.includes('时间') || item.includes('偏移'))
    return 'time_scope'
  if (item.includes('适用范围') || item.includes('适用对象') || item.includes('排除对象') || item.includes('命中范围'))
    return 'apply_scope'
  if (item.includes('偏好权重'))
    return 'weight'
  if (item.includes('最大次数'))
    return 'count'
  if (item.includes('休息'))
    return 'rest'
  if (item.includes('员工') || item.includes('排班人员'))
    return 'employee'
  if (item.includes('班次'))
    return 'shift'
  return 'config'
}

function highlightPendingTarget(targetKey: string) {
  activePendingTargetKey.value = targetKey
  if (pendingHighlightTimer)
    clearTimeout(pendingHighlightTimer)
  pendingHighlightTimer = setTimeout(() => {
    activePendingTargetKey.value = ''
    pendingHighlightTimer = null
  }, 1800)
}

async function scrollToPendingItem(item: string) {
  const targetKey = getPendingTargetKey(item)
  await nextTick()
  const targetMap: Record<string, HTMLElement | null> = {
    time_scope: timeScopeSectionRef.value,
    apply_scope: applyScopeSectionRef.value,
    shift: shiftFieldRef.value,
    employee: employeeFieldRef.value,
    rest: restFieldRef.value,
    weight: weightFieldRef.value,
    count: countFieldRef.value,
    config: configSectionRef.value,
  }
  const target = targetMap[targetKey] || configSectionRef.value
  target?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  highlightPendingTarget(targetKey)
}

async function handleFixPendingFromDetail(item: string) {
  const currentRule = detailRule.value
  if (!currentRule)
    return

  detailVisible.value = false
  await handleEdit(currentRule, item)
}

function formatExclusiveRuleSummary(shiftIDs: string[], scope: string, primaryShiftIDs: string[], objectShiftIDs: string[]) {
  const isConsecutive = scope === 'consecutive'
  const primaryText = formatShiftReferences(primaryShiftIDs)
  const objectText = formatShiftReferences(objectShiftIDs)
  const shiftText = formatShiftReferences(shiftIDs)

  if (primaryShiftIDs.length > 0 && objectShiftIDs.length > 0) {
    return isConsecutive
      ? `${primaryText} 不能与 ${objectText} 连续相邻安排`
      : `${primaryText} 不能与 ${objectText} 在同一天同时安排`
  }

  if (shiftIDs.length > 0) {
    return isConsecutive
      ? `${shiftText} 之间不能连续相邻安排`
      : `${shiftText} 在同一天内互斥`
  }

  return '互斥关系尚未配置完整'
}

function formatMaxCountRuleSummary(normalized: Record<string, unknown>) {
  const shiftText = formatShiftReference(getTextValue(normalized.shift_id))
  const max = getNumberValue(normalized.max)
  const period = getTextValue(normalized.period)
  const periodText = period === 'day' ? '每天' : period === 'month' ? '每月' : '每周'
  if (shiftText === '-' || max === undefined)
    return '次数限制尚未配置完整'
  return `${shiftText} ${periodText}最多安排 ${max} 次`
}

function formatMinRestRuleSummary(normalized: Record<string, unknown>) {
  const days = getNumberValue(normalized.days) ?? getNumberValue(normalized.min_rest_days)
  const hours = getNumberValue(normalized.min_rest_hours)
  const mustConsecutive = getBooleanValue(normalized.must_consecutive)
  const parts = [
    days !== undefined ? `${days} 天` : '',
    hours !== undefined ? `${hours} 小时` : '',
  ].filter(Boolean)

  if (parts.length === 0)
    return '最小休息条件尚未配置完整'

  return mustConsecutive
    ? `连续工作后至少连续休息 ${parts.join(' / ')}`
    : `连续工作后至少休息 ${parts.join(' / ')}`
}

function formatRequiredTogetherRuleSummary(normalized: Record<string, unknown>) {
  const employees = getStringArrayValue(normalized.employee_ids)
  const employeeText = formatEmployeeReferences(employees)
  const shiftText = formatShiftReference(getTextValue(normalized.shift_id))
  if (!employees.length || shiftText === '-')
    return '同时排班条件尚未配置完整'
  return `${employeeText} 需要一起安排到 ${shiftText}`
}

function formatPreferEmployeeRuleSummary(normalized: Record<string, unknown>) {
  const employeeText = formatEmployeeReference(getTextValue(normalized.employee_id))
  const shiftText = formatShiftReference(getTextValue(normalized.shift_id))
  const weight = getNumberValue(normalized.weight)
  if (employeeText === '-' || shiftText === '-')
    return '人员偏好尚未配置完整'
  return weight === undefined
    ? `优先安排 ${employeeText} 承担 ${shiftText}`
    : `优先安排 ${employeeText} 承担 ${shiftText}，偏好权重 ${weight}`
}

function formatStaffSourceRuleSummary(normalized: Record<string, unknown>) {
  const targetShiftText = formatShiftReference(getTextValue(normalized.target_shift_id))
  const sourceShiftText = formatShiftReference(getTextValue(normalized.source_shift_id))
  if (targetShiftText === '-' || sourceShiftText === '-')
    return '人员来源规则尚未配置完整'
  return `${targetShiftText} 优先从 ${sourceShiftText} 调配人员`
}

function formatExecutionOrderRuleSummary(normalized: Record<string, unknown>) {
  const beforeShiftText = formatShiftReference(getTextValue(normalized.before_shift_id))
  const afterShiftText = formatShiftReference(getTextValue(normalized.after_shift_id))
  if (beforeShiftText === '-' || afterShiftText === '-')
    return '执行顺序尚未配置完整'
  return `${beforeShiftText} 需要先于 ${afterShiftText} 执行`
}

function formatRuleSemanticSummary(ruleType?: string, config?: Record<string, unknown>, context?: RuleConfigContext) {
  const normalized = normalizeStructuredConfig(ruleType, config, context)
  const type = getConfigType(ruleType, normalized)

  switch (type) {
    case 'exclusive_shifts': {
      const shiftIDs = getStringArrayValue(normalized.shift_ids)
      const primaryShiftIDs = mergeShiftRelationIDs(context?.subjectShiftIDs, context?.targetShiftIDs)
      const objectShiftIDs = mergeShiftRelationIDs(context?.objectShiftIDs)
      return formatExclusiveRuleSummary(shiftIDs, getTextValue(normalized.scope), primaryShiftIDs, objectShiftIDs)
    }
    case 'max_count':
      return formatMaxCountRuleSummary(normalized)
    case 'min_rest':
      return formatMinRestRuleSummary(normalized)
    case 'required_together':
      return formatRequiredTogetherRuleSummary(normalized)
    case 'prefer_employee':
      return formatPreferEmployeeRuleSummary(normalized)
    case 'staff_source':
      return formatStaffSourceRuleSummary(normalized)
    case 'execution_order':
      return formatExecutionOrderRuleSummary(normalized)
    default: {
      const fields = getConfigFields(ruleType, config, context)
      return fields.map(field => `${field.label}：${field.value}`).join('；') || '-'
    }
  }
}

function formatSummaryConfig(rule: Rule) {
  return formatRuleSemanticSummary(rule.type, rule.config, buildRuleConfigContext(rule))
}
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <div class="toolbar-filters">
        <el-input
          v-model="keyword"
          placeholder="搜索规则"
          clearable
          style="width: 240px"
          :prefix-icon="Search"
        />
        <el-checkbox v-model="onlyShowPendingRules">
          仅看待完善规则
        </el-checkbox>
        <el-checkbox v-model="prioritizePendingIssues">
          问题优先排序
        </el-checkbox>
        <el-tag size="small" type="warning" effect="plain">
          当前页待完善 {{ currentPagePendingCount }} / {{ currentPageRuleCount }}
        </el-tag>
        <el-tag size="small" type="info" effect="plain">
          当前展示 {{ displayedPendingCount }} / {{ displayedRuleCount }}
        </el-tag>
        <el-tag
          size="small"
          type="warning"
          :effect="selectedPendingCategory === 'all' ? 'dark' : 'plain'"
          class="toolbar-filter-tag"
          @click="setPendingCategoryFilter('all')"
        >
          全部问题 {{ currentPagePendingCount }}
        </el-tag>
        <el-tag
          size="small"
          type="info"
          :effect="selectedPendingCategory === 'config' ? 'dark' : 'plain'"
          class="toolbar-filter-tag"
          @click="togglePendingCategoryFilter('config')"
        >
          配置缺失 {{ currentPagePendingBreakdown.config }}
        </el-tag>
        <el-tag
          size="small"
          type="warning"
          :effect="selectedPendingCategory === 'scope' ? 'dark' : 'plain'"
          class="toolbar-filter-tag"
          @click="togglePendingCategoryFilter('scope')"
        >
          范围问题 {{ currentPagePendingBreakdown.scope }}
        </el-tag>
        <el-tag
          size="small"
          type="danger"
          :effect="selectedPendingCategory === 'conflict' ? 'dark' : 'plain'"
          class="toolbar-filter-tag"
          @click="togglePendingCategoryFilter('conflict')"
        >
          逻辑冲突 {{ currentPagePendingBreakdown.conflict }}
        </el-tag>
      </div>
      <div class="toolbar-actions">
        <el-button v-if="canManageRules" :icon="MagicStick" @click="aiDialogVisible = true">
          AI 解析
        </el-button>
        <el-button v-if="canManageRules" type="primary" :icon="Plus" @click="handleAdd">
          新增规则
        </el-button>
      </div>
    </div>

    <el-table v-loading="loading" :data="displayedRules" border stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="220">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleView(row)">
            {{ row.name }}
          </el-button>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="160">
        <template #default="{ row }">
          {{ getRuleTypeText(row.type) }}
        </template>
      </el-table-column>
      <el-table-column prop="category" label="约束等级" width="100">
        <template #default="{ row }">
          <el-tag :type="(categoryMap[row.category]?.type as any)" size="small">
            {{ categoryMap[row.category]?.label || row.category }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column prop="enabled" label="启用" width="80">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
            {{ row.enabled ? '是' : '否' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="规则说明" min-width="380">
        <template #default="{ row }">
          <div class="rule-summary-cell">
            <div class="rule-summary-main">
              {{ formatSummaryConfig(row) }}
            </div>
            <div class="rule-summary-sub">
              {{ formatRuleListSecondarySummary(row) }}
            </div>
            <div v-if="getRuleWarningItems(row).length" class="rule-summary-warning-list">
              <el-tag
                v-for="(item, index) in getRuleWarningPreviewItems(row)"
                :key="`${row.id}-warning-preview-${index}`"
                size="small"
                :type="getPendingIssueMeta(item).type"
                effect="plain"
                class="rule-summary-warning rule-summary-warning-action"
                :title="getPendingIssueSuggestion(item)"
                @click.stop="handleRuleWarningShortcut(row, item)"
              >
                {{ getPendingIssueMeta(item).label }} · {{ item }}
              </el-tag>
              <el-tag
                v-if="getRuleWarningItems(row).length > getRuleWarningPreviewItems(row).length"
                size="small"
                type="info"
                effect="plain"
                class="rule-summary-warning rule-summary-warning-action"
                @click.stop="handleRuleWarningShortcut(row)"
              >
                +{{ getRuleWarningItems(row).length - getRuleWarningPreviewItems(row).length }} 项
              </el-tag>
            </div>
            <div v-if="getRuleWarningItems(row).length" class="rule-summary-hint">
              建议优先：{{ getPendingIssueSuggestion(getRulePrimaryPendingItem(row)) }}
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
      <el-table-column label="操作" :width="operationWidth" fixed="right">
        <template #default="{ row }">
          <el-button :icon="View" link type="primary" @click="handleView(row)">
            详情
          </el-button>
          <el-button v-if="canManageRules" :icon="Edit" link type="primary" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button v-if="canManageRules" :icon="Delete" link type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="page-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="currentPageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="760px" destroy-on-close class="rule-edit-dialog">
      <div v-loading="dialogLoading" class="edit-dialog-body">
        <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="如：最大连续工作 5 天" />
          </el-form-item>
          <el-form-item label="类型" prop="type">
            <el-select v-model="form.type" placeholder="选择规则类型" style="width: 100%" :disabled="!!editingId" @change="handleRuleTypeChange">
              <el-option v-for="t in ruleTypes" :key="t.value" :label="t.label" :value="t.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="约束等级" prop="category">
            <el-radio-group v-model="form.category">
              <el-radio v-for="category in categoryOptions" :key="category.value" :value="category.value">
                {{ category.label }}
              </el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="form.priority" :min="1" :max="1000" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.enabled" />
          </el-form-item>
          <el-form-item label="时间范围">
            <div ref="timeScopeSectionRef" class="scope-editor" :class="{ 'pending-target-active': activePendingTargetKey === 'time_scope' }">
              <el-select v-model="form.time_scope" style="width: 100%" @change="updateFormTimeScope">
                <el-option label="同一天" value="same_day" />
                <el-option label="同一周" value="same_week" />
                <el-option label="同一月" value="same_month" />
                <el-option label="自定义" value="custom" />
              </el-select>
              <el-form-item v-if="form.time_scope === 'custom'" label="偏移天数" label-width="92px" class="nested-form-item scope-nested-item">
                <el-input-number
                  :model-value="form.time_offset_days"
                  style="width: 100%"
                  @update:model-value="updateFormTimeOffsetDays"
                />
              </el-form-item>
              <div class="scope-summary-text">
                {{ formatFormTimeScopeSummary() }}
              </div>
            </div>
          </el-form-item>
          <el-form-item label="适用范围">
            <div ref="applyScopeSectionRef" class="scope-editor" :class="{ 'pending-target-active': activePendingTargetKey === 'apply_scope' }">
              <el-radio-group v-model="form.apply_scope" @change="updateFormApplyScope">
                <el-radio value="global">
                  全部员工
                </el-radio>
                <el-radio value="specific">
                  指定对象
                </el-radio>
              </el-radio-group>

              <template v-if="form.apply_scope === 'specific'">
                <div class="scope-toggle-row">
                  <el-checkbox :model-value="formApplyScopeState.all" @update:model-value="updateFormScopeAll">
                    先匹配所有员工
                  </el-checkbox>
                </div>
                <el-form-item label="指定员工" label-width="92px" class="nested-form-item scope-nested-item">
                  <el-select
                    :model-value="formApplyScopeState.employeeIDs"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="updateFormEmployeeScopes"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(employeeOptions, formApplyScopeState.employeeIDs)"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item label="指定分组" label-width="92px" class="nested-form-item scope-nested-item">
                  <el-select
                    :model-value="formApplyScopeState.groupIDs"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="updateFormGroupScopes"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(groupOptions, formApplyScopeState.groupIDs)"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item label="排除员工" label-width="92px" class="nested-form-item scope-nested-item">
                  <el-select
                    :model-value="formApplyScopeState.excludeEmployeeIDs"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="updateFormExcludeEmployeeScopes"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(employeeOptions, formApplyScopeState.excludeEmployeeIDs)"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item label="排除分组" label-width="92px" class="nested-form-item scope-nested-item">
                  <el-select
                    :model-value="formApplyScopeState.excludeGroupIDs"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="updateFormExcludeGroupScopes"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(groupOptions, formApplyScopeState.excludeGroupIDs)"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
              </template>

              <el-alert v-if="formApplyScopeErrors.length" type="warning" :closable="false" show-icon class="scope-alert">
                <template #title>
                  适用范围仍需补充
                </template>
                <div class="config-error-text">
                  {{ formApplyScopeErrors.join('；') }}
                </div>
              </el-alert>

              <div class="scope-summary-text">
                {{ formatFormApplyScopeSummary() }}
              </div>
            </div>
          </el-form-item>
          <el-form-item label="配置">
            <div ref="configSectionRef" class="config-editor" :class="{ 'pending-target-active': activePendingTargetKey === 'config' }">
              <div class="config-type-label">
                {{ getConfigTypeText(form.type, formConfig) }}
              </div>

              <el-alert v-if="formConfigErrors.length" type="error" :closable="false" show-icon style="margin-bottom: 12px;">
                <template #title>
                  配置仍有待完善项
                </template>
                <div class="config-error-text">
                  {{ formConfigErrors.join('；') }}
                </div>
              </el-alert>

              <template v-if="getConfigType(form.type, formConfig) === 'exclusive_shifts'">
                <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px;">
                  <template #title>
                    只填写主体班次表示主体集合内互斥；同时填写客体班次表示主体班次排斥客体班次。
                  </template>
                </el-alert>
                <el-form-item label="主体班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getFormPrimaryShiftIDs()"
                      multiple
                      filterable
                      style="width: 100%"
                      @update:model-value="updatePrimaryShiftRelations"
                    >
                      <el-option v-for="option in mergeShiftOptions(getFormPrimaryShiftIDs())" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="客体班次" label-width="92px" class="nested-form-item">
                  <el-select
                    :model-value="formConfigContext?.objectShiftIDs || []"
                    multiple
                    filterable
                    style="width: 100%"
                    @update:model-value="updateObjectShiftRelations"
                  >
                    <el-option v-for="option in mergeShiftOptions(formConfigContext?.objectShiftIDs || [])" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="互斥范围" label-width="92px" class="nested-form-item">
                  <el-select
                    :model-value="getConfigText(form.type, 'scope')"
                    style="width: 100%"
                    @update:model-value="updateScopeMode"
                  >
                    <el-option label="同日互斥" value="same_day" />
                    <el-option label="连续互斥" value="consecutive" />
                  </el-select>
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'max_count'">
                <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'shift_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updateMaxCountShift"
                    >
                      <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="最大次数" label-width="92px" class="nested-form-item">
                  <div ref="countFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'count' }">
                    <el-input-number
                      :model-value="getConfigNumber(form.type, 'max')"
                      :min="0"
                      style="width: 100%"
                      @update:model-value="updateMaxCountValue"
                    />
                  </div>
                </el-form-item>
                <el-form-item label="统计周期" label-width="92px" class="nested-form-item">
                  <el-select
                    :model-value="getConfigText(form.type, 'period')"
                    style="width: 100%"
                    @update:model-value="updatePeriodValue"
                  >
                    <el-option label="按天" value="day" />
                    <el-option label="按周" value="week" />
                    <el-option label="按月" value="month" />
                  </el-select>
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'min_rest'">
                <el-form-item label="休息天数" label-width="92px" class="nested-form-item">
                  <div ref="restFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'rest' }">
                    <el-input-number
                      :model-value="getConfigNumber(form.type, 'days') ?? getConfigNumber(form.type, 'min_rest_days')"
                      :min="0"
                      style="width: 100%"
                      @update:model-value="updateRestDays"
                    />
                  </div>
                </el-form-item>
                <el-form-item label="休息小时" label-width="92px" class="nested-form-item">
                  <el-input-number
                    :model-value="getConfigNumber(form.type, 'min_rest_hours')"
                    :min="0"
                    style="width: 100%"
                    @update:model-value="updateRestHours"
                  />
                </el-form-item>
                <el-form-item label="连续休息" label-width="92px" class="nested-form-item">
                  <el-switch
                    :model-value="getConfigBoolean(form.type, 'must_consecutive')"
                    @update:model-value="updateMustConsecutive"
                  />
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'required_together'">
                <el-form-item label="排班人员" label-width="92px" class="nested-form-item">
                  <div ref="employeeFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'employee' }">
                    <el-select
                      :model-value="getConfigStringArray(form.type, 'employee_ids')"
                      multiple
                      filterable
                      collapse-tags
                      collapse-tags-tooltip
                      style="width: 100%"
                      @update:model-value="updateEmployeeIds"
                    >
                      <el-option
                        v-for="option in mergeSelectOptions(employeeOptions, getConfigStringArray(form.type, 'employee_ids'))"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'shift_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updateRequiredTogetherShift"
                    >
                      <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'prefer_employee'">
                <el-form-item label="偏好员工" label-width="92px" class="nested-form-item">
                  <div ref="employeeFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'employee' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'employee_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updateEmployeeId"
                    >
                      <el-option
                        v-for="option in mergeSelectOptions(employeeOptions, [getConfigText(form.type, 'employee_id')])"
                        :key="option.value"
                        :label="option.label"
                        :value="option.value"
                      />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'shift_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updatePreferredShift"
                    >
                      <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="偏好权重" label-width="92px" class="nested-form-item">
                  <div ref="weightFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'weight' }">
                    <el-input-number
                      :model-value="getConfigNumber(form.type, 'weight')"
                      :min="0"
                      :max="100"
                      style="width: 100%"
                      @update:model-value="updatePreferredWeight"
                    />
                  </div>
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'staff_source'">
                <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'target_shift_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updateTargetShift"
                    >
                      <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'target_shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="来源班次" label-width="92px" class="nested-form-item">
                  <el-select
                    :model-value="getConfigText(form.type, 'source_shift_id')"
                    filterable
                    clearable
                    style="width: 100%"
                    @update:model-value="updateSourceShift"
                  >
                    <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'source_shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
              </template>

              <template v-else-if="getConfigType(form.type, formConfig) === 'execution_order'">
                <el-form-item label="前置班次" label-width="92px" class="nested-form-item">
                  <div ref="shiftFieldRef" class="field-anchor" :class="{ 'pending-target-active': activePendingTargetKey === 'shift' }">
                    <el-select
                      :model-value="getConfigText(form.type, 'before_shift_id')"
                      filterable
                      clearable
                      style="width: 100%"
                      @update:model-value="updateBeforeShift"
                    >
                      <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'before_shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                    </el-select>
                  </div>
                </el-form-item>
                <el-form-item label="后置班次" label-width="92px" class="nested-form-item">
                  <el-select
                    :model-value="getConfigText(form.type, 'after_shift_id')"
                    filterable
                    clearable
                    style="width: 100%"
                    @update:model-value="updateAfterShift"
                  >
                    <el-option v-for="option in mergeShiftOptions([getConfigText(form.type, 'after_shift_id')])" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
              </template>

              <template v-else>
                <el-alert type="info" :closable="false" show-icon>
                  <template #title>
                    当前规则类型暂未提供专用表单，下面展示已有配置字段。
                  </template>
                </el-alert>
                <div class="config-field-list">
                  <div v-for="field in getConfigFields(form.type, formConfig, formConfigContext || undefined)" :key="field.key" class="config-field-row">
                    <span class="config-field-name">{{ field.label }}</span>
                    <span class="config-field-value">{{ field.value }}</span>
                  </div>
                  <div v-if="getConfigFields(form.type, formConfig, formConfigContext || undefined).length === 0" class="config-field-row empty-field-row">
                    <span class="config-field-value">当前没有可展示的配置项</span>
                  </div>
                </div>
              </template>

              <div class="editor-interpretation-block">
                <div class="editor-interpretation-title">
                  实时规则解读
                </div>
                <div class="editor-interpretation-card">
                  <div
                    v-for="(item, index) in formatFormInterpretation()"
                    :key="`form-interpretation-${index}`"
                    class="editor-interpretation-item"
                  >
                    {{ item }}
                  </div>
                </div>
                <div v-if="getFormPendingItems().length" class="editor-pending-block">
                  <div class="editor-interpretation-title">
                    当前待完善项
                  </div>
                  <div class="editor-pending-card">
                    <div
                      v-for="(item, index) in getFormPendingItems()"
                      :key="`form-pending-${index}`"
                      class="editor-pending-item editor-pending-item-action"
                      @click="scrollToPendingItem(item)"
                    >
                      <div class="pending-item-main">
                        <el-tag size="small" :type="getPendingIssueMeta(item).type" effect="plain">
                          {{ getPendingIssueMeta(item).label }}
                        </el-tag>
                        <span>{{ item }}</span>
                      </div>
                      <div class="pending-item-hint">
                        {{ getPendingIssueSuggestion(item) }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="form.description" type="textarea" :rows="2" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="detailVisible" title="规则详情" width="760px" destroy-on-close class="rule-detail-dialog" @closed="closeDetail">
      <div v-loading="detailLoading" class="detail-dialog-body">
        <template v-if="detailRule">
          <div class="detail-header">
            <div>
              <div class="detail-title">
                {{ detailRule.name }}
              </div>
              <div class="detail-subtitle">
                {{ detailRule.description || '暂无描述' }}
              </div>
            </div>
            <div class="detail-tags">
              <el-tag :type="(categoryMap[detailRule.category]?.type as any)" size="small">
                {{ categoryMap[detailRule.category]?.label || detailRule.category }}
              </el-tag>
              <el-tag size="small" :type="detailRule.enabled ? 'success' : 'info'">
                {{ detailRule.enabled ? '启用' : '停用' }}
              </el-tag>
              <el-tag size="small" type="warning">
                优先级 {{ detailRule.priority }}
              </el-tag>
            </div>
          </div>

          <el-descriptions :column="2" border class="detail-descriptions">
            <el-descriptions-item label="规则类型">
              {{ getRuleTypeText(detailRule.type) }}
            </el-descriptions-item>
            <el-descriptions-item label="子类型">
              {{ getSubTypeText(detailRule.sub_category) }}
            </el-descriptions-item>
            <el-descriptions-item label="作用范围">
              {{ getApplyScopeText(detailRule.apply_scope) }}
            </el-descriptions-item>
            <el-descriptions-item label="时间范围">
              {{ getTimeScopeText(detailRule.time_scope) }}
            </el-descriptions-item>
            <el-descriptions-item label="规则来源">
              {{ getSourceTypeText(detailRule.source_type) }}
            </el-descriptions-item>
            <el-descriptions-item label="更新时间">
              {{ formatFriendlyDateTime(detailRule.updated_at) }}
            </el-descriptions-item>
          </el-descriptions>

          <el-collapse v-model="activeDetailSection" accordion class="detail-accordion">
            <el-collapse-item name="interpretation" class="detail-collapse-item">
              <template #title>
                <div class="detail-collapse-title-wrap">
                  <span class="detail-section-title">规则解读</span>
                  <span class="detail-collapse-meta">
                    {{ formatRuleInterpretation(detailRule).length }} 条解读
                    <template v-if="getRuleWarningItems(detailRule).length">
                      · {{ getRuleWarningItems(detailRule).length }} 项待完善
                    </template>
                  </span>
                </div>
              </template>
              <div class="detail-section detail-section-accordion">
                <div class="detail-interpretation-card">
                  <div
                    v-for="(item, index) in formatRuleInterpretation(detailRule)"
                    :key="`${detailRule.id}-interpretation-${index}`"
                    class="detail-interpretation-item"
                  >
                    {{ item }}
                  </div>
                </div>
                <div v-if="getRuleWarningItems(detailRule).length" class="detail-pending-card">
                  <div class="detail-pending-title">
                    当前待完善项
                  </div>
                  <div
                    v-for="(item, index) in getRuleWarningItems(detailRule)"
                    :key="`${detailRule.id}-warning-${index}`"
                    :ref="(el) => setDetailPendingItemRef(item, el as HTMLElement | null)"
                    class="detail-pending-item"
                    :class="{ 'detail-pending-item-active': activeDetailPendingItem === item }"
                  >
                    <div class="pending-item-main">
                      <el-tag size="small" :type="getPendingIssueMeta(item).type" effect="plain">
                        {{ getPendingIssueMeta(item).label }}
                      </el-tag>
                      <span>{{ item }}</span>
                    </div>
                    <div class="pending-item-side">
                      <div class="pending-item-hint pending-item-hint-compact">
                        {{ getPendingIssueSuggestion(item) }}
                      </div>
                      <el-button link type="primary" @click="handleFixPendingFromDetail(item)">
                        {{ getPendingIssueActionText(item) }}
                      </el-button>
                    </div>
                  </div>
                </div>
              </div>
            </el-collapse-item>

            <el-collapse-item name="summary" class="detail-collapse-item">
              <template #title>
                <div class="detail-collapse-title-wrap">
                  <span class="detail-section-title">业务说明</span>
                  <span class="detail-collapse-meta">一句话概览</span>
                </div>
              </template>
              <div class="detail-section detail-section-accordion">
                <div class="detail-semantic-summary">
                  {{ formatRuleSemanticSummary(detailRule.type, detailRule.config, buildRuleConfigContext(detailRule)) }}
                </div>
              </div>
            </el-collapse-item>

            <el-collapse-item name="config" class="detail-collapse-item">
              <template #title>
                <div class="detail-collapse-title-wrap">
                  <span class="detail-section-title">配置明细</span>
                  <span class="detail-collapse-meta">{{ getConfigFields(detailRule.type, detailRule.config, buildRuleConfigContext(detailRule)).length + 1 }} 项配置</span>
                </div>
              </template>
              <div class="detail-section detail-section-accordion">
                <div class="detail-field-list">
                  <div class="detail-field-row">
                    <span class="detail-field-label">配置类型</span>
                    <span class="detail-field-value">{{ getConfigTypeText(detailRule.type, detailRule.config) }}</span>
                  </div>
                  <div v-for="field in getConfigFields(detailRule.type, detailRule.config, buildRuleConfigContext(detailRule))" :key="field.key" class="detail-field-row">
                    <span class="detail-field-label">{{ field.label }}</span>
                    <span class="detail-field-value">{{ field.value }}</span>
                  </div>
                </div>
              </div>
            </el-collapse-item>

            <el-collapse-item name="associations" class="detail-collapse-item">
              <template #title>
                <div class="detail-collapse-title-wrap">
                  <span class="detail-section-title">关联与适用对象</span>
                  <span class="detail-collapse-meta">对象与范围</span>
                </div>
              </template>
              <div class="detail-section detail-section-accordion">
                <div class="detail-field-list detail-summary-list">
                  <div class="detail-field-row detail-field-row-summary">
                    <span class="detail-field-label">适用范围说明</span>
                    <span class="detail-field-value detail-field-value-left">{{ formatApplyScopeSummary(detailRule) }}</span>
                  </div>
                  <div class="detail-field-row detail-field-row-summary">
                    <span class="detail-field-label">关联对象说明</span>
                    <span class="detail-field-value detail-field-value-left">{{ formatAssociationSummary(detailRule) }}</span>
                  </div>
                </div>
                <div class="detail-field-list">
                  <div class="detail-field-row">
                    <span class="detail-field-label">主体班次</span>
                    <span class="detail-field-value">{{ formatShiftReferences(getAssociationIDs(detailRule, 'shift', 'subject')) || '-' }}</span>
                  </div>
                  <div class="detail-field-row">
                    <span class="detail-field-label">客体班次</span>
                    <span class="detail-field-value">{{ formatShiftReferences(getAssociationIDs(detailRule, 'shift', 'object')) || '-' }}</span>
                  </div>
                  <div class="detail-field-row">
                    <span class="detail-field-label">目标班次</span>
                    <span class="detail-field-value">{{ formatShiftReferences(getAssociationIDs(detailRule, 'shift', 'target')) || '-' }}</span>
                  </div>
                  <div class="detail-field-row">
                    <span class="detail-field-label">关联员工</span>
                    <span class="detail-field-value">{{ formatEmployeeReferences(getAssociatedEmployees(detailRule)) || '-' }}</span>
                  </div>
                  <div class="detail-field-row">
                    <span class="detail-field-label">关联分组</span>
                    <span class="detail-field-value">{{ formatGroupReferences(getAssociatedGroups(detailRule)) || '-' }}</span>
                  </div>
                  <div class="detail-field-row">
                    <span class="detail-field-label">适用范围明细</span>
                    <span class="detail-field-value">{{ formatApplyScopes(detailRule) }}</span>
                  </div>
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>
        </template>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">
          关闭
        </el-button>
      </template>
    </el-dialog>

    <AIParseRulesDialog v-model="aiDialogVisible" @saved="refresh" />
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow: hidden;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.rule-summary-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 2px 0;
}

.rule-summary-main {
  line-height: 1.7;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.rule-summary-sub {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  word-break: break-word;
}

.rule-summary-warning {
  width: fit-content;
  max-width: 100%;
  white-space: normal;
  height: auto;
  line-height: 1.5;
}

.rule-summary-warning-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.rule-summary-hint {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-color-primary);
}

.rule-summary-warning-action {
  cursor: pointer;
}

.toolbar-filter-tag {
  cursor: pointer;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
}

.config-editor {
  width: 100%;
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.scope-editor {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.scope-summary-text {
  font-size: 12px;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
}

.scope-toggle-row {
  display: flex;
  align-items: center;
}

.scope-alert {
  margin: 0;
}

.scope-nested-item {
  margin-bottom: 0;
}

.field-anchor {
  width: 100%;
}

.pending-target-active {
  border-radius: 8px;
  box-shadow: 0 0 0 2px var(--el-color-warning-light-5);
  background: color-mix(in srgb, var(--el-color-warning-light-9) 70%, white 30%);
  transition: box-shadow 0.2s ease, background 0.2s ease;
}

.config-type-label {
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-primary);
}

.editor-interpretation-block {
  margin-top: 14px;
}

.editor-pending-block {
  margin-top: 12px;
}

.editor-interpretation-title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.editor-interpretation-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-radius: 8px;
  background: linear-gradient(180deg, var(--el-fill-color-light) 0%, var(--el-fill-color-blank) 100%);
  border: 1px dashed var(--el-border-color);
}

.editor-interpretation-item {
  position: relative;
  padding-left: 14px;
  line-height: 1.7;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.editor-interpretation-item::before {
  content: '';
  position: absolute;
  top: 10px;
  left: 0;
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-color-success);
}

.editor-pending-card,
.detail-pending-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-radius: 8px;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-5);
}

.detail-pending-card {
  margin-top: 12px;
}

.detail-pending-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-warning-dark-2);
}

.editor-pending-item,
.detail-pending-item {
  position: relative;
  padding-left: 14px;
  line-height: 1.7;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.detail-pending-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.detail-pending-item-active {
  border-radius: 8px;
  box-shadow: 0 0 0 2px var(--el-color-warning-light-5);
  background: color-mix(in srgb, var(--el-color-warning-light-9) 75%, white 25%);
  transition: box-shadow 0.2s ease, background 0.2s ease;
}

.pending-item-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  max-width: 280px;
}

.pending-item-main {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.pending-item-hint {
  margin-top: 4px;
  padding-left: 34px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.pending-item-hint-compact {
  margin-top: 0;
  padding-left: 0;
  text-align: right;
}

.editor-pending-item-action {
  cursor: pointer;
  transition: color 0.2s ease, transform 0.2s ease;
}

.editor-pending-item-action:hover {
  color: var(--el-color-primary);
  transform: translateX(2px);
}

.editor-pending-item::before,
.detail-pending-item::before {
  content: '';
  position: absolute;
  top: 10px;
  left: 0;
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-color-warning);
}

.nested-form-item {
  margin-bottom: 12px;
}

.nested-form-item:last-child {
  margin-bottom: 0;
}

.config-field-list,
.detail-field-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.config-field-row,
.detail-field-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
}

.config-field-name,
.detail-field-label {
  flex: 0 0 120px;
  color: var(--el-text-color-secondary);
}

.config-field-value,
.detail-field-value {
  flex: 1;
  text-align: right;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.detail-field-value-left {
  text-align: left;
}

.empty-field-row {
  justify-content: center;
}

.config-error-text {
  line-height: 1.6;
  white-space: pre-wrap;
}

.edit-dialog-body,
.detail-dialog-body {
  min-height: 240px;
  max-height: calc(100vh - 240px);
  overflow-y: auto;
  padding-right: 4px;
}

:deep(.rule-edit-dialog .el-dialog),
:deep(.rule-detail-dialog .el-dialog) {
  max-height: calc(100vh - 32px);
  display: flex;
  flex-direction: column;
}

:deep(.rule-edit-dialog .el-dialog__body),
:deep(.rule-detail-dialog .el-dialog__body) {
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.detail-title {
  font-size: 18px;
  font-weight: 600;
  line-height: 1.4;
}

.detail-subtitle {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.detail-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.detail-descriptions {
  margin-bottom: 18px;
}

.detail-accordion {
  border-top: none;
  border-bottom: none;
}

.detail-collapse-item {
  margin-top: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  overflow: hidden;
  background: var(--el-fill-color-blank);
}

.detail-collapse-item :deep(.el-collapse-item__header) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  min-height: 58px;
  padding: 0 16px;
  border-bottom: none;
  background: linear-gradient(180deg, var(--el-fill-color-light) 0%, var(--el-fill-color-blank) 100%);
  text-align: left;
}

.detail-collapse-item :deep(.el-collapse-item__arrow) {
  margin: 0;
}

.detail-collapse-item :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.detail-collapse-item :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}

.detail-collapse-title-wrap {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 2px;
  width: 100%;
  min-width: 0;
  margin-right: auto;
  padding-right: 8px;
  text-align: left;
}

.detail-collapse-meta {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
}

.detail-section {
  margin-top: 18px;
}

.detail-section-accordion {
  margin-top: 0;
  padding: 0 16px 16px;
}

.detail-section-title {
  margin: 0;
  line-height: 1.4;
  font-size: 14px;
  font-weight: 600;
}

.detail-summary-list {
  margin-bottom: 12px;
}

.detail-field-row-summary {
  align-items: flex-start;
  background: var(--el-color-primary-light-9);
}

.detail-semantic-summary {
  padding: 14px 16px;
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  color: var(--el-text-color-primary);
  line-height: 1.8;
  word-break: break-word;
}

.detail-interpretation-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border-radius: 8px;
  background: linear-gradient(180deg, var(--el-color-primary-light-9) 0%, var(--el-fill-color-blank) 100%);
  border: 1px solid var(--el-color-primary-light-7);
}

.detail-interpretation-item {
  position: relative;
  padding-left: 16px;
  line-height: 1.8;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.detail-interpretation-item::before {
  content: '';
  position: absolute;
  top: 12px;
  left: 0;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--el-color-primary);
}
</style>
