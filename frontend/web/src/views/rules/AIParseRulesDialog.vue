<script setup lang="ts">
import type { ParsedRuleConfig, ParsedRuleConflictInfo, ParsedRuleDependencyInfo, ShiftCatalogItem } from '@/api/ai'
import { MagicStick } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, nextTick, onBeforeUnmount, reactive, ref } from 'vue'
import { parseRulesBatchStream } from '@/api/ai'
import { listEmployees } from '@/api/employees'
import { listGroups } from '@/api/groups'
import { batchSaveRules } from '@/api/rules'
import { listShifts } from '@/api/shifts'
import type { Employee } from '@/types/employee'
import type { Shift } from '@/types/shift'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
})

// ======== 解析状态 ========
const description = ref('')
const parsing = ref(false)
const reasoning = ref('')
type DialogRule = ParsedRuleConfig & { _selected: boolean, _editing: boolean, _saving: boolean, _validationErrors: string[] }

const parsedRules = ref<DialogRule[]>([])
const parsedDependencies = ref<ParsedRuleDependencyInfo[]>([])
const parsedConflicts = ref<ParsedRuleConflictInfo[]>([])
const step = ref<'input' | 'streaming' | 'preview'>('input')
const shiftCatalog = ref<ShiftCatalogItem[]>([])
const employeeCatalog = ref<Employee[]>([])
const groupCatalog = ref<Array<{ id: string, name: string, code?: string }>>([])

// ======== 流式状态 ========
const streamingReasoning = ref('')
const streamingContent = ref('')
const streamingPhase = ref<'thinking' | 'generating'>('thinking')
let abortController: AbortController | null = null
const streamingBoxRef = ref<HTMLElement>()

const categoryMap: Record<string, { label: string, type: string }> = {
  constraint: { label: '约束', type: 'danger' },
  preference: { label: '偏好', type: 'info' },
  dependency: { label: '依赖', type: 'warning' },
}

const ruleTypeMap: Record<string, string> = {
  exclusive: '排他规则',
  combinable: '可组合规则',
  required_together: '必须同时规则',
  periodic: '周期规则',
  maxCount: '次数限制规则',
  forbidden_day: '禁排日期规则',
  preferred: '偏好规则',
  source: '人员需求规则',
  order: '顺序约束规则',
  min_rest: '最小休息规则',
}

const subTypeMap: Record<string, string> = {
  forbid: '排他/禁止',
  limit: '数量限制',
  min_rest: '最小休息',
  must: '固定排班',
  prefer: '偏好权重',
  combinable: '可组合',
  source: '人员来源',
  order: '执行顺序',
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

const scopeTypeMap: Record<string, string> = {
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
}

const configValueMap: Record<string, string> = {
  forward: '向后',
  backward: '向前',
  true: '是',
  false: '否',
}

const shiftConfigKeys = new Set(['shift_id', 'shift_ids', 'target_shift_id', 'source_shift_id', 'before_shift_id', 'after_shift_id'])

type RuleLike = Partial<Pick<ParsedRuleConfig, 'config' | 'rule_type' | 'type' | 'subject_shifts' | 'object_shifts' | 'target_shifts' | 'scope_employees'>>

type ConfigField = {
  key: string
  label: string
  value: string
}

const shiftLabelMap = computed<Record<string, string>>(() => {
  const result: Record<string, string> = {}
  for (const item of shiftCatalog.value) {
    const code = item.code?.trim()
    const name = item.name?.trim()
    if (!code || !name)
      continue
    result[code] = `${name}（${code}）`
  }
  return result
})

const shiftOptions = computed(() => shiftCatalog.value.map(item => ({
  label: `${item.name}（${item.code}）`,
  value: item.code,
})))

const employeeOptions = computed(() => employeeCatalog.value
  .filter(employee => employee.status !== 'inactive')
  .map(employee => ({
    label: employee.employee_no ? `${employee.name}（${employee.employee_no}）` : employee.name,
    value: employee.id,
  })))

const groupOptions = computed(() => groupCatalog.value.map(group => ({
  label: group.code ? `${group.name}（${group.code}）` : group.name,
  value: group.id,
})))

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

const saveErrorMessage = ref('')

function buildDialogRule(rule: ParsedRuleConfig): DialogRule {
  const dialogRule: DialogRule = {
    ...rule,
    _selected: true,
    _editing: false,
    _saving: false,
    _validationErrors: [],
  }
  dialogRule._validationErrors = validateRule(dialogRule)
  return dialogRule
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function getTextValue(value: unknown): string {
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

function getFirstNonEmpty(...values: Array<string | undefined>) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim())
      return value.trim()
  }
  return ''
}

function inferConfigType(ruleType?: string) {
  switch (ruleType) {
    case 'exclusive':
      return 'exclusive_shifts'
    case 'maxCount':
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

function getConfigType(rule: RuleLike) {
  const directType = isRecord(rule.config) ? getTextValue(rule.config.type) : ''
  return directType || inferConfigType(getFirstNonEmpty(rule.rule_type, rule.type))
}

function normalizeStructuredConfig(rule: RuleLike) {
  const config = isRecord(rule.config) ? { ...rule.config } : {}
  const type = getConfigType(rule)
  const subjectShift = rule.subject_shifts?.[0]
  const objectShift = rule.object_shifts?.[0]
  const targetShift = rule.target_shifts?.[0]
  const scopeEmployee = rule.scope_employees?.[0]

  if (type && !getTextValue(config.type))
    config.type = type

  switch (type) {
    case 'exclusive_shifts': {
      const mergedShiftIDs = Array.from(new Set([
        ...getStringArrayValue(config.shift_ids),
        ...(rule.subject_shifts || []),
        ...(rule.object_shifts || []),
        ...(rule.target_shifts || []),
      ]))
      if (mergedShiftIDs.length)
        config.shift_ids = mergedShiftIDs
      if (!getTextValue(config.scope))
        config.scope = 'same_day'
      break
    }
    case 'max_count':
      if (!getTextValue(config.shift_id))
        config.shift_id = getFirstNonEmpty(getTextValue(config.shift_id), targetShift, subjectShift, objectShift)
      if (!getTextValue(config.period))
        config.period = 'week'
      break
    case 'required_together':
      if (!getTextValue(config.shift_id))
        config.shift_id = getFirstNonEmpty(getTextValue(config.shift_id), targetShift, subjectShift, objectShift)
      if (getStringArrayValue(config.employee_ids).length === 0 && rule.scope_employees?.length)
        config.employee_ids = [...rule.scope_employees]
      break
    case 'prefer_employee':
      if (!getTextValue(config.employee_id))
        config.employee_id = getFirstNonEmpty(getTextValue(config.employee_id), scopeEmployee)
      if (!getTextValue(config.shift_id))
        config.shift_id = getFirstNonEmpty(getTextValue(config.shift_id), targetShift, subjectShift, objectShift)
      break
    case 'staff_source':
      if (!getTextValue(config.target_shift_id))
        config.target_shift_id = getFirstNonEmpty(getTextValue(config.target_shift_id), targetShift, subjectShift)
      if (!getTextValue(config.source_shift_id))
        config.source_shift_id = getFirstNonEmpty(getTextValue(config.source_shift_id), objectShift, subjectShift)
      break
    case 'execution_order':
      if (!getTextValue(config.before_shift_id))
        config.before_shift_id = getFirstNonEmpty(getTextValue(config.before_shift_id), subjectShift, objectShift)
      if (!getTextValue(config.after_shift_id))
        config.after_shift_id = getFirstNonEmpty(getTextValue(config.after_shift_id), targetShift, objectShift)
      break
  }

  return config
}

function getConfigTypeText(rule: RuleLike) {
  const type = getConfigType(rule)
  if (!type)
    return '-'
  return configTypeMap[type] || type
}

function getConfigText(rule: RuleLike | undefined, key: string) {
  if (!rule)
    return ''
  return getTextValue(normalizeStructuredConfig(rule)[key])
}

function getConfigStringArray(rule: RuleLike | undefined, key: string) {
  if (!rule)
    return []
  return getStringArrayValue(normalizeStructuredConfig(rule)[key])
}

function getConfigNumber(rule: RuleLike | undefined, key: string) {
  if (!rule)
    return undefined
  return getNumberValue(normalizeStructuredConfig(rule)[key])
}

function getConfigBoolean(rule: RuleLike | undefined, key: string) {
  if (!rule)
    return false
  return getBooleanValue(normalizeStructuredConfig(rule)[key]) ?? false
}

function formatPlainList(values: string[]) {
  return values.length ? values.join('、') : '-'
}

function formatListInput(values: string[]) {
  return values.join('、')
}

function uniqueTextList(values: string[]) {
  return Array.from(new Set(values.map(value => value.trim()).filter(Boolean)))
}

function parseListInput(value: string) {
  return uniqueTextList(value
    .split(/[、,，\n]/)
    .map(item => item.trim())
    .filter(Boolean))
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

function hasKnownEmployeeReference(value?: string) {
  if (!value)
    return false
  return !!employeeLabelMap.value[value]
}

function hasKnownGroupReference(value?: string) {
  if (!value)
    return false
  return !!groupLabelMap.value[value]
}

function buildGenericConfigFields(rule: RuleLike): ConfigField[] {
  const config = normalizeStructuredConfig(rule)
  return Object.entries(config)
    .filter(([key]) => key !== 'type')
    .map(([key, value]) => ({
      key,
      label: configKeyMap[key] || key,
      value: formatConfigValue(value, key),
    }))
}

function getConfigFields(rule: RuleLike): ConfigField[] {
  const config = normalizeStructuredConfig(rule)
  const type = getConfigType(rule)

  switch (type) {
    case 'exclusive_shifts':
      return [
        { key: 'shift_ids', label: '互斥班次', value: formatShiftReferences(getStringArrayValue(config.shift_ids)) || '-' },
        { key: 'scope', label: '互斥范围', value: exclusiveScopeMap[getTextValue(config.scope)] || getTextValue(config.scope) || '-' },
      ]
    case 'max_count':
      return [
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(config.shift_id)) },
        { key: 'max', label: '最大次数', value: getNumberValue(config.max)?.toString() || '-' },
        { key: 'period', label: '统计周期', value: periodMap[getTextValue(config.period)] || getTextValue(config.period) || '-' },
      ]
    case 'min_rest':
      return [
        { key: 'days', label: '最少休息天数', value: getNumberValue(config.days)?.toString() || getNumberValue(config.min_rest_days)?.toString() || '-' },
        { key: 'min_rest_hours', label: '最少休息小时', value: getNumberValue(config.min_rest_hours)?.toString() || '-' },
        { key: 'must_consecutive', label: '必须连续休息', value: getBooleanValue(config.must_consecutive) === undefined ? '-' : formatConfigValue(getBooleanValue(config.must_consecutive)) },
      ]
    case 'required_together':
      return [
        { key: 'employee_ids', label: '同时排班人员', value: formatEmployeeReferences(getStringArrayValue(config.employee_ids)) || '-' },
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(config.shift_id)) },
      ]
    case 'prefer_employee':
      return [
        { key: 'employee_id', label: '偏好员工', value: formatEmployeeReference(getTextValue(config.employee_id)) },
        { key: 'shift_id', label: '目标班次', value: formatShiftReference(getTextValue(config.shift_id)) },
        { key: 'weight', label: '偏好权重', value: getNumberValue(config.weight)?.toString() || '-' },
      ]
    case 'staff_source':
      return [
        { key: 'target_shift_id', label: '目标班次', value: formatShiftReference(getTextValue(config.target_shift_id)) },
        { key: 'source_shift_id', label: '来源班次', value: formatShiftReference(getTextValue(config.source_shift_id)) },
      ]
    case 'execution_order':
      return [
        { key: 'before_shift_id', label: '前置班次', value: formatShiftReference(getTextValue(config.before_shift_id)) },
        { key: 'after_shift_id', label: '后置班次', value: formatShiftReference(getTextValue(config.after_shift_id)) },
      ]
    default:
      return buildGenericConfigFields(rule)
  }
}

function setEditConfigValue(idx: number, key: string, value: unknown) {
  const current = editForm[idx]
  if (!current)
    return
  current.config = {
    ...normalizeStructuredConfig(current),
    [key]: value,
  }
}

function setEditShiftList(idx: number, key: 'subject_shifts' | 'object_shifts' | 'target_shifts', value: string[]) {
  const current = editForm[idx]
  if (!current)
    return
  current[key] = uniqueTextList(value)
}

function setEditScopeList(idx: number, key: 'scope_employees' | 'scope_groups', value: string[]) {
  const current = editForm[idx]
  if (!current)
    return
  current[key] = uniqueTextList(value)
}

function setEditScopeType(idx: number, value: string) {
  const current = editForm[idx]
  if (!current)
    return
  current.scope_type = value
  if (value === 'all') {
    current.scope_employees = []
    current.scope_groups = []
  }
}

function needsEmployeeScopeInput(scopeType?: string) {
  return scopeType === 'employee' || scopeType === 'exclude_employee'
}

function needsGroupScopeInput(scopeType?: string) {
  return scopeType === 'group' || scopeType === 'exclude_group'
}

function validateRule(rule: RuleLike & Partial<Pick<ParsedRuleConfig, 'name' | 'category' | 'sub_type' | 'scope_type' | 'scope_groups' | 'scope_employees'>>) {
  const errors: string[] = []
  const config = normalizeStructuredConfig(rule)
  const configType = getConfigType(rule)
  const name = getTextValue(rule.name)
  const category = getTextValue(rule.category)
  const subType = getTextValue(rule.sub_type)

  if (!name)
    errors.push('规则名称不能为空')
  if (!category)
    errors.push('规则类别不能为空')
  if (!subType)
    errors.push('规则子类型不能为空')

  switch (configType) {
    case 'exclusive_shifts':
      if (getStringArrayValue(config.shift_ids).length < 2)
        errors.push('互斥班次至少需要 2 个班次')
      if (uniqueTextList(getStringArrayValue(config.shift_ids)).length !== getStringArrayValue(config.shift_ids).length)
        errors.push('互斥班次不能重复')
      break
    case 'max_count':
      if (!getTextValue(config.shift_id))
        errors.push('最大次数规则缺少目标班次')
      if (getNumberValue(config.max) === undefined)
        errors.push('最大次数规则缺少最大次数')
      if ((getNumberValue(config.max) ?? 0) < 0)
        errors.push('最大次数不能小于 0')
      break
    case 'min_rest': {
      const hasDays = getNumberValue(config.days) !== undefined || getNumberValue(config.min_rest_days) !== undefined
      const hasHours = getNumberValue(config.min_rest_hours) !== undefined
      if (!hasDays && !hasHours)
        errors.push('最小休息规则至少需要休息天数或休息小时')
      if ((getNumberValue(config.days) ?? getNumberValue(config.min_rest_days) ?? 0) < 0)
        errors.push('休息天数不能小于 0')
      if ((getNumberValue(config.min_rest_hours) ?? 0) < 0)
        errors.push('休息小时不能小于 0')
      break
    }
    case 'required_together':
      if (getStringArrayValue(config.employee_ids).length === 0)
        errors.push('必须同时规则缺少排班人员')
      if (!getTextValue(config.shift_id))
        errors.push('必须同时规则缺少目标班次')
      if (uniqueTextList(getStringArrayValue(config.employee_ids)).length !== getStringArrayValue(config.employee_ids).length)
        errors.push('必须同时规则中的人员不能重复')
      if (employeeCatalog.value.length > 0 && getStringArrayValue(config.employee_ids).some(value => !hasKnownEmployeeReference(value)))
        errors.push('必须同时规则中存在未匹配员工目录的人员')
      break
    case 'prefer_employee':
      if (!getTextValue(config.employee_id))
        errors.push('偏好规则缺少偏好员工')
      if (!getTextValue(config.shift_id))
        errors.push('偏好规则缺少目标班次')
      if ((getNumberValue(config.weight) ?? -1) < 0 || (getNumberValue(config.weight) ?? 101) > 100)
        errors.push('偏好权重必须在 0 到 100 之间')
      if (employeeCatalog.value.length > 0 && getTextValue(config.employee_id) && !hasKnownEmployeeReference(getTextValue(config.employee_id)))
        errors.push('偏好规则中的员工未匹配员工目录')
      break
    case 'staff_source':
      if (!getTextValue(config.target_shift_id))
        errors.push('人员来源规则缺少目标班次')
      if (!getTextValue(config.source_shift_id))
        errors.push('人员来源规则缺少来源班次')
      if (getTextValue(config.target_shift_id) && getTextValue(config.target_shift_id) === getTextValue(config.source_shift_id))
        errors.push('目标班次和来源班次不能相同')
      break
    case 'execution_order':
      if (!getTextValue(config.before_shift_id))
        errors.push('执行顺序规则缺少前置班次')
      if (!getTextValue(config.after_shift_id))
        errors.push('执行顺序规则缺少后置班次')
      if (getTextValue(config.before_shift_id) && getTextValue(config.before_shift_id) === getTextValue(config.after_shift_id))
        errors.push('前置班次和后置班次不能相同')
      break
  }

  if (needsEmployeeScopeInput(rule.scope_type) && !(rule.scope_employees?.length))
    errors.push('当前作用范围需要指定员工')
  if (needsGroupScopeInput(rule.scope_type) && !(rule.scope_groups?.length))
    errors.push('当前作用范围需要指定分组')
  if (uniqueTextList(rule.scope_employees || []).length !== (rule.scope_employees || []).length)
    errors.push('作用范围中的员工不能重复')
  if (uniqueTextList(rule.scope_groups || []).length !== (rule.scope_groups || []).length)
    errors.push('作用范围中的分组不能重复')
  if (employeeCatalog.value.length > 0 && (rule.scope_employees || []).some(value => !hasKnownEmployeeReference(value)))
    errors.push('作用范围中存在未匹配员工目录的员工')
  if (groupCatalog.value.length > 0 && (rule.scope_groups || []).some(value => !hasKnownGroupReference(value)))
    errors.push('作用范围中存在未匹配分组目录的分组')

  return Array.from(new Set(errors))
}

function syncRuleValidation(rule: DialogRule) {
  rule._validationErrors = validateRule(rule)
}

function formatValidationErrors(errors?: string[]) {
  return errors?.length ? errors.join('；') : ''
}

const selectedValidationErrors = computed(() => parsedRules.value
  .filter(rule => rule._selected && rule._validationErrors.length > 0)
  .flatMap(rule => rule._validationErrors.map(error => `${rule.name || '未命名规则'}：${error}`)))

function getTableRowClassName({ row }: { row: DialogRule }) {
  return row._validationErrors.length > 0 ? 'invalid-rule-row' : ''
}

function extractErrorMessage(error: unknown) {
  const parts: string[] = []

  if (isRecord(error) && isRecord(error.response) && isRecord(error.response.data)) {
    const responseData = error.response.data
    const message = getTextValue(responseData.message)
    if (message)
      parts.push(message)
    if (isRecord(responseData.details)) {
      for (const [field, value] of Object.entries(responseData.details)) {
        const detailValues = Array.isArray(value)
          ? value.map(item => typeof item === 'string' ? item.trim() : '').filter(Boolean)
          : []
        if (detailValues.length)
          parts.push(`${field}：${detailValues.join('、')}`)
      }
    }
  }

  if (parts.length === 0 && isRecord(error)) {
    const message = getTextValue(error.message)
    if (message)
      parts.push(message)
  }

  return parts.join('\n') || '保存失败，请检查规则配置后重试'
}

function getRuleTypeText(value?: string) {
  if (!value)
    return '-'
  return ruleTypeMap[value] || value
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

function getScopeTypeText(value?: string) {
  if (!value)
    return '-'
  return scopeTypeMap[value] || value
}

function getSourceTypeText(value?: string) {
  if (!value)
    return '-'
  return sourceTypeMap[value] || value
}

function formatShiftReference(value?: string) {
  if (!value)
    return '-'
  return shiftLabelMap.value[value] || value
}

function formatEmployeeReference(value?: string) {
  if (!value)
    return '-'
  return employeeLabelMap.value[value] || value
}

function formatEmployeeReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatEmployeeReference(value)).join('、')
}

function formatGroupReference(value?: string) {
  if (!value)
    return '-'
  return groupLabelMap.value[value] || value
}

function formatGroupReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatGroupReference(value)).join('、')
}

function formatShiftReferences(values?: string[]) {
  if (!values?.length)
    return ''
  return values.map(value => formatShiftReference(value)).join('、')
}

function formatSubjectLabel(row: ParsedRuleConfig) {
  if (row.subject_shifts?.length)
    return formatShiftReferences(row.subject_shifts)
  if (row.scope_employees?.length)
    return formatEmployeeReferences(row.scope_employees)
  if (row.scope_groups?.length)
    return formatGroupReferences(row.scope_groups)
  if (row.scope_type === 'all')
    return '所有员工'
  return '-'
}

function formatSubjectType(row: ParsedRuleConfig) {
  if (row.subject_shifts?.length)
    return '主体班次'
  if (row.scope_employees?.length)
    return '主体人'
  if (row.scope_groups?.length)
    return '主体分组'
  if (row.scope_type === 'all')
    return '适用对象'
  return '-'
}

function formatShiftRelation(row: ParsedRuleConfig) {
  const parts = [
    row.subject_shifts?.length ? `主体班次=${formatShiftReferences(row.subject_shifts)}` : '',
    row.object_shifts?.length ? `客体班次=${formatShiftReferences(row.object_shifts)}` : '',
    row.target_shifts?.length ? `目标班次=${formatShiftReferences(row.target_shifts)}` : '',
  ].filter(Boolean)
  return parts.join(' | ') || '-'
}

function formatScope(row: ParsedRuleConfig) {
  const parts = [getScopeTypeText(row.scope_type)]
  if (row.scope_employees?.length)
    parts.push(`员工=${formatEmployeeReferences(row.scope_employees)}`)
  if (row.scope_groups?.length)
    parts.push(`分组=${formatGroupReferences(row.scope_groups)}`)
  return parts.filter(Boolean).join(' | ') || '-'
}

function formatConfigValue(value: unknown, key?: string): string {
  if (typeof value === 'boolean')
    return value ? '是' : '否'
  if (typeof value === 'string') {
    if (key && shiftConfigKeys.has(key))
      return formatShiftReference(value)
    return configValueMap[value] || value
  }
  if (Array.isArray(value))
    return value.map(item => key && shiftConfigKeys.has(key) && typeof item === 'string' ? formatShiftReference(item) : formatConfigValue(item)).join('、')
  if (value && typeof value === 'object')
    return JSON.stringify(value)
  return String(value)
}

function formatConfig(config?: Record<string, unknown>) {
  if (!config || Object.keys(config).length === 0)
    return '-'
  return Object.entries(config)
    .map(([key, value]) => `${configKeyMap[key] || key}：${formatConfigValue(value, key)}`)
    .join('；')
}

function normalizeShiftCatalog(shifts: Shift[]): ShiftCatalogItem[] {
  return shifts
    .filter(shift => Boolean(shift.code?.trim()) && Boolean(shift.name?.trim()) && (shift.is_active ?? shift.status === 'active' ?? true))
    .map((shift) => {
      const aliases = Array.from(new Set([
        shift.name.trim(),
        shift.name.replace(/\s+/g, ''),
      ].filter(Boolean)))
      return {
        code: shift.code!.trim(),
        name: shift.name.trim(),
        aliases,
      }
    })
    .sort((left, right) => left.code.localeCompare(right.code, 'zh-CN'))
}

async function ensureShiftCatalog() {
  if (shiftCatalog.value.length > 0)
    return shiftCatalog.value

  const response = await listShifts({ page: 1, page_size: 500 })
  const items = Array.isArray(response) ? response : response.items
  shiftCatalog.value = normalizeShiftCatalog(items)
  return shiftCatalog.value
}

async function ensureReferenceCatalogs() {
  const tasks: Promise<unknown>[] = []

  if (employeeCatalog.value.length === 0) {
    tasks.push(
      listEmployees({ page: 1, page_size: 500 }).then((response) => {
        employeeCatalog.value = response.items || []
      }),
    )
  }

  if (groupCatalog.value.length === 0) {
    tasks.push(
      Promise.resolve(listGroups({ page: 1, page_size: 500 })).then((response) => {
        groupCatalog.value = Array.isArray(response) ? response : response.items || []
      }),
    )
  }

  await Promise.all(tasks)
}

async function handleParse() {
  if (!description.value.trim()) {
    ElMessage.warning('请输入规则描述')
    return
  }

  saveErrorMessage.value = ''

  try {
    const [catalog] = await Promise.all([ensureShiftCatalog(), ensureReferenceCatalogs()])
    if (catalog.length === 0) {
      ElMessage.warning('当前没有已配置短代号的班次，涉及班次的规则将无法严格映射')
    }
  }
  catch {
    ElMessage.error('加载班次、员工或分组目录失败，无法进行严格映射')
    return
  }

  parsing.value = true
  streamingReasoning.value = ''
  streamingContent.value = ''
  streamingPhase.value = 'thinking'
  step.value = 'streaming'

  abortController = parseRulesBatchStream(
    { description: description.value, shift_catalog: shiftCatalog.value },
    {
      onReasoning(text) {
        streamingReasoning.value += text
        nextTick(() => {
          if (streamingBoxRef.value)
            streamingBoxRef.value.scrollTop = streamingBoxRef.value.scrollHeight
        })
      },
      onChunk(text) {
        if (streamingPhase.value === 'thinking')
          streamingPhase.value = 'generating'
        streamingContent.value += text
        nextTick(() => {
          if (streamingBoxRef.value)
            streamingBoxRef.value.scrollTop = streamingBoxRef.value.scrollHeight
        })
      },
      onDone(result) {
        parsing.value = false
        const rules = result.rules?.length ? result.rules : (result.parsed_rules || [])
        parsedRules.value = rules.map(buildDialogRule)
        parsedDependencies.value = result.dependencies || []
        parsedConflicts.value = result.conflicts || []
        reasoning.value = result.reasoning || ''
        step.value = 'preview'
        abortController = null
      },
      onError(message) {
        parsing.value = false
        ElMessage.error(message)
        step.value = 'input'
        abortController = null
      },
    },
  )
}

function handleCancelStream() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
  parsing.value = false
  step.value = 'input'
}

function handleBack() {
  saveErrorMessage.value = ''
  step.value = 'input'
}

onBeforeUnmount(() => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
})

// ======== 编辑 ========
const editForm = reactive<Record<string, ParsedRuleConfig>>({})

function startEdit(idx: number) {
  const r = parsedRules.value[idx]
  r._editing = true
  saveErrorMessage.value = ''
  void ensureReferenceCatalogs()
  editForm[idx] = {
    name: r.name,
    type: r.type,
    rule_type: r.rule_type,
    category: r.category,
    sub_type: r.sub_type,
    apply_scope: r.apply_scope,
    time_scope: r.time_scope,
    time_offset_days: r.time_offset_days,
    rule_data: r.rule_data,
    priority: r.priority,
    source_type: r.source_type,
    version: r.version,
    config: normalizeStructuredConfig(r),
    description: r.description,
    subject_shifts: r.subject_shifts ? [...r.subject_shifts] : [],
    object_shifts: r.object_shifts ? [...r.object_shifts] : [],
    target_shifts: r.target_shifts ? [...r.target_shifts] : [],
    scope_type: r.scope_type,
    scope_employees: r.scope_employees ? [...r.scope_employees] : [],
    scope_groups: r.scope_groups ? [...r.scope_groups] : [],
  }
}

function confirmEdit(idx: number) {
  const r = parsedRules.value[idx]
  const f = editForm[idx]
  r.name = f.name
  r.type = f.type
  r.rule_type = f.rule_type
  r.category = f.category
  r.sub_type = f.sub_type
  r.apply_scope = f.apply_scope
  r.time_scope = f.time_scope
  r.time_offset_days = f.time_offset_days
  r.rule_data = f.rule_data
  r.priority = f.priority
  r.source_type = f.source_type
  r.version = f.version
  r.config = { ...normalizeStructuredConfig(f) }
  r.description = f.description
  r.subject_shifts = f.subject_shifts
  r.object_shifts = f.object_shifts
  r.target_shifts = f.target_shifts
  r.scope_type = f.scope_type
  r.scope_employees = f.scope_employees
  r.scope_groups = f.scope_groups
  syncRuleValidation(r)
  r._editing = false
  delete editForm[idx]
}

function cancelEdit(idx: number) {
  parsedRules.value[idx]._editing = false
  delete editForm[idx]
}

// ======== 保存 ========
const selectedCount = computed(() => parsedRules.value.filter(r => r._selected).length)
const saving = ref(false)

async function handleSave() {
  const selected = parsedRules.value.filter(r => r._selected)
  if (selected.length === 0) {
    ElMessage.warning('请至少选择一条规则')
    return
  }

  selected.forEach(syncRuleValidation)
  if (selected.some(rule => rule._validationErrors.length > 0)) {
    saveErrorMessage.value = `以下规则仍有必填缺失：\n${selectedValidationErrors.value.join('\n')}`
    ElMessage.error('存在未完善的规则，请先修正红色项')
    return
  }

  saving.value = true
  saveErrorMessage.value = ''
  selected.forEach(rule => rule._saving = true)

  try {
    await batchSaveRules({
      parsed_rules: selected.map(rule => ({
        name: rule.name,
        type: rule.type,
        rule_type: rule.rule_type,
        category: rule.category,
        sub_type: rule.sub_type,
        apply_scope: rule.apply_scope,
        time_scope: rule.time_scope,
        time_offset_days: rule.time_offset_days,
        rule_data: rule.rule_data,
        priority: rule.priority ?? 100,
        source_type: rule.source_type,
        version: rule.version,
        config: rule.config,
        description: rule.description,
        subject_shifts: rule.subject_shifts,
        object_shifts: rule.object_shifts,
        target_shifts: rule.target_shifts,
        scope_type: rule.scope_type,
        scope_employees: rule.scope_employees,
        scope_groups: rule.scope_groups,
      })),
      dependencies: parsedDependencies.value,
      conflicts: parsedConflicts.value,
    })
    selected.forEach(rule => {
      rule._selected = false
      rule._saving = false
    })
    saving.value = false
    ElMessage.success(`成功保存 ${selected.length} 条规则`)
    emit('saved')
    visible.value = false
    resetState()
  }
  catch (error) {
    selected.forEach(rule => rule._saving = false)
    saving.value = false
    saveErrorMessage.value = extractErrorMessage(error)
  }
}

function resetState() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
  description.value = ''
  parsedRules.value = []
  parsedDependencies.value = []
  parsedConflicts.value = []
  reasoning.value = ''
  saveErrorMessage.value = ''
  streamingReasoning.value = ''
  streamingContent.value = ''
  streamingPhase.value = 'thinking'
  parsing.value = false
  shiftCatalog.value = []
  employeeCatalog.value = []
  groupCatalog.value = []
  step.value = 'input'
}

function handleClose() {
  resetState()
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="AI 智能解析规则"
    width="800px"
    :close-on-click-modal="false"
    @closed="handleClose"
  >
    <!-- Step 1: 输入 -->
    <div v-if="step === 'input'" class="parse-input-step">
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 16px">
        <template #title>
          输入自然语言描述，AI 将自动解析为结构化排班规则。支持同时描述多条规则。
        </template>
      </el-alert>
      <el-input
        v-model="description"
        type="textarea"
        :rows="6"
        placeholder="示例：&#10;1. 夜班后不能上早班&#10;2. 每人每周最多 5 个夜班&#10;3. 护士长周一必须白班&#10;4. 连续工作不超过 5 天后必须休息 2 天"
        maxlength="2000"
        show-word-limit
      />
    </div>

    <!-- Step 2: 流式输出 -->
    <div v-if="step === 'streaming'" class="parse-streaming-step">
      <div class="streaming-header">
        <el-icon class="streaming-icon" :size="16">
          <MagicStick />
        </el-icon>
        <span>{{ streamingPhase === 'thinking' ? 'AI 正在思考分析…' : 'AI 正在生成规则…' }}</span>
      </div>
      <div ref="streamingBoxRef" class="streaming-box">
        <!-- 思考过程 -->
        <div v-if="streamingReasoning" class="streaming-reasoning">
          <div class="streaming-section-label">💭 思考过程</div>
          <pre class="streaming-text reasoning-text">{{ streamingReasoning }}<span v-if="streamingPhase === 'thinking'" class="cursor-blink">▊</span></pre>
        </div>
        <!-- 生成内容 -->
        <div v-if="streamingContent" class="streaming-output">
          <div class="streaming-section-label">📝 生成结果</div>
          <pre class="streaming-text output-text">{{ streamingContent }}<span v-if="streamingPhase === 'generating'" class="cursor-blink">▊</span></pre>
        </div>
        <!-- 等待状态 -->
        <div v-if="!streamingReasoning && !streamingContent" class="streaming-waiting">
          <span class="cursor-blink">▊</span> 等待 AI 响应…
        </div>
      </div>
    </div>

    <!-- Step 3: 预览结果 -->
    <div v-if="step === 'preview'" class="parse-preview-step">
      <div v-if="reasoning" class="reasoning-box">
        <el-icon style="margin-right: 4px; vertical-align: middle">
          <MagicStick />
        </el-icon>
        <span>{{ reasoning }}</span>
      </div>

      <el-alert
        v-if="saveErrorMessage"
        type="error"
        show-icon
        :closable="false"
        style="margin-top: 12px"
      >
        <template #title>
          保存失败
        </template>
        <div class="save-error-text">{{ saveErrorMessage }}</div>
      </el-alert>

      <el-table :data="parsedRules" border style="width: 100%; margin-top: 12px" :row-class-name="getTableRowClassName">
        <el-table-column type="expand">
          <template #default="{ row, $index }">
            <div v-if="row._editing" class="edit-panel">
              <el-form :model="editForm[$index]" label-width="80px" size="small">
                <el-alert
                  v-if="validateRule(editForm[$index]).length"
                  type="error"
                  :closable="false"
                  show-icon
                  style="margin-bottom: 12px"
                >
                  <template #title>
                    当前规则仍有未完成项
                  </template>
                  <div class="save-error-text">{{ formatValidationErrors(validateRule(editForm[$index])) }}</div>
                </el-alert>
                <el-form-item label="名称">
                  <el-input v-model="editForm[$index].name" />
                </el-form-item>
                <el-form-item label="类别">
                  <el-select v-model="editForm[$index].category" style="width: 100%">
                    <el-option label="约束" value="constraint" />
                    <el-option label="偏好" value="preference" />
                    <el-option label="依赖" value="dependency" />
                  </el-select>
                </el-form-item>
                <el-form-item label="规则类型">
                  <el-input v-model="editForm[$index].rule_type" />
                </el-form-item>
                <el-form-item label="子类型">
                  <el-select v-model="editForm[$index].sub_type" style="width: 100%">
                    <el-option v-for="(label, val) in subTypeMap" :key="val" :label="label" :value="val" />
                  </el-select>
                </el-form-item>
                <el-form-item label="作用范围">
                  <el-select v-model="editForm[$index].apply_scope" style="width: 100%">
                    <el-option v-for="(label, val) in applyScopeMap" :key="val" :label="label" :value="val" />
                  </el-select>
                </el-form-item>
                <el-form-item label="时间范围">
                  <el-select v-model="editForm[$index].time_scope" style="width: 100%">
                    <el-option v-for="(label, val) in timeScopeMap" :key="val" :label="label" :value="val" />
                  </el-select>
                </el-form-item>
                <el-form-item label="时间偏移">
                  <el-input-number v-model="editForm[$index].time_offset_days" :step="1" style="width: 100%" />
                </el-form-item>
                <el-form-item label="主体班次">
                  <el-select
                    :model-value="editForm[$index].subject_shifts || []"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    style="width: 100%"
                    @update:model-value="value => setEditShiftList($index, 'subject_shifts', value)"
                  >
                    <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="客体班次">
                  <el-select
                    :model-value="editForm[$index].object_shifts || []"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    style="width: 100%"
                    @update:model-value="value => setEditShiftList($index, 'object_shifts', value)"
                  >
                    <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="目标班次">
                  <el-select
                    :model-value="editForm[$index].target_shifts || []"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    style="width: 100%"
                    @update:model-value="value => setEditShiftList($index, 'target_shifts', value)"
                  >
                    <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                  </el-select>
                </el-form-item>
                <el-form-item label="适用对象">
                  <el-select
                    :model-value="editForm[$index].scope_type"
                    style="width: 100%"
                    @update:model-value="value => setEditScopeType($index, value)"
                  >
                    <el-option v-for="(label, val) in scopeTypeMap" :key="val" :label="label" :value="val" />
                  </el-select>
                </el-form-item>
                <el-form-item v-if="needsEmployeeScopeInput(editForm[$index].scope_type)" label="指定员工">
                  <el-select
                    :model-value="editForm[$index].scope_employees || []"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="value => setEditScopeList($index, 'scope_employees', value)"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(employeeOptions, editForm[$index].scope_employees || [])"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item v-if="needsGroupScopeInput(editForm[$index].scope_type)" label="指定分组">
                  <el-select
                    :model-value="editForm[$index].scope_groups || []"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    @update:model-value="value => setEditScopeList($index, 'scope_groups', value)"
                  >
                    <el-option
                      v-for="option in mergeSelectOptions(groupOptions, editForm[$index].scope_groups || [])"
                      :key="option.value"
                      :label="option.label"
                      :value="option.value"
                    />
                  </el-select>
                </el-form-item>
                <el-form-item label="配置">
                  <div class="config-editor">
                    <div class="config-type-label">{{ getConfigTypeText(editForm[$index]) }}</div>

                    <template v-if="getConfigType(editForm[$index]) === 'exclusive_shifts'">
                      <el-form-item label="互斥班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigStringArray(editForm[$index], 'shift_ids')"
                          multiple
                          filterable
                          allow-create
                          default-first-option
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'shift_ids', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="互斥范围" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'scope')"
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'scope', value)"
                        >
                          <el-option label="同日互斥" value="same_day" />
                          <el-option label="连续互斥" value="consecutive" />
                        </el-select>
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'max_count'">
                      <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="最大次数" label-width="92px" class="nested-form-item">
                        <el-input-number
                          :model-value="getConfigNumber(editForm[$index], 'max')"
                          :min="0"
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'max', value)"
                        />
                      </el-form-item>
                      <el-form-item label="统计周期" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'period')"
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'period', value)"
                        >
                          <el-option label="按天" value="day" />
                          <el-option label="按周" value="week" />
                          <el-option label="按月" value="month" />
                        </el-select>
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'min_rest'">
                      <el-form-item label="休息天数" label-width="92px" class="nested-form-item">
                        <el-input-number
                          :model-value="getConfigNumber(editForm[$index], 'days') ?? getConfigNumber(editForm[$index], 'min_rest_days')"
                          :min="0"
                          style="width: 100%"
                          @update:model-value="value => { setEditConfigValue($index, 'days', value); setEditConfigValue($index, 'min_rest_days', value) }"
                        />
                      </el-form-item>
                      <el-form-item label="休息小时" label-width="92px" class="nested-form-item">
                        <el-input-number
                          :model-value="getConfigNumber(editForm[$index], 'min_rest_hours')"
                          :min="0"
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'min_rest_hours', value)"
                        />
                      </el-form-item>
                      <el-form-item label="连续休息" label-width="92px" class="nested-form-item">
                        <el-switch
                          :model-value="getConfigBoolean(editForm[$index], 'must_consecutive')"
                          @update:model-value="value => setEditConfigValue($index, 'must_consecutive', value)"
                        />
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'required_together'">
                      <el-form-item label="排班人员" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigStringArray(editForm[$index], 'employee_ids')"
                          multiple
                          filterable
                          collapse-tags
                          collapse-tags-tooltip
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'employee_ids', uniqueTextList(value))"
                        >
                          <el-option
                            v-for="option in mergeSelectOptions(employeeOptions, getConfigStringArray(editForm[$index], 'employee_ids'))"
                            :key="option.value"
                            :label="option.label"
                            :value="option.value"
                          />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'prefer_employee'">
                      <el-form-item label="偏好员工" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'employee_id')"
                          filterable
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'employee_id', typeof value === 'string' ? value.trim() : '')"
                        >
                          <el-option
                            v-for="option in mergeSelectOptions(employeeOptions, [getConfigText(editForm[$index], 'employee_id')])"
                            :key="option.value"
                            :label="option.label"
                            :value="option.value"
                          />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="偏好权重" label-width="92px" class="nested-form-item">
                        <el-input-number
                          :model-value="getConfigNumber(editForm[$index], 'weight')"
                          :min="0"
                          :max="100"
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'weight', value)"
                        />
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'staff_source'">
                      <el-form-item label="目标班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'target_shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'target_shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="来源班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'source_shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'source_shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                    </template>

                    <template v-else-if="getConfigType(editForm[$index]) === 'execution_order'">
                      <el-form-item label="前置班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'before_shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'before_shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="后置班次" label-width="92px" class="nested-form-item">
                        <el-select
                          :model-value="getConfigText(editForm[$index], 'after_shift_id')"
                          filterable
                          allow-create
                          default-first-option
                          clearable
                          style="width: 100%"
                          @update:model-value="value => setEditConfigValue($index, 'after_shift_id', value)"
                        >
                          <el-option v-for="option in shiftOptions" :key="option.value" :label="option.label" :value="option.value" />
                        </el-select>
                      </el-form-item>
                    </template>

                    <template v-else>
                      <div class="config-field-list">
                        <div v-for="field in getConfigFields(editForm[$index])" :key="field.key" class="config-field-row">
                          <span class="config-field-name">{{ field.label }}</span>
                          <span class="config-field-value">{{ field.value }}</span>
                        </div>
                      </div>
                    </template>
                  </div>
                </el-form-item>
                <el-form-item label="描述">
                  <el-input v-model="editForm[$index].description" />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" size="small" @click="confirmEdit($index)">
                    确认修改
                  </el-button>
                  <el-button size="small" @click="cancelEdit($index)">
                    取消
                  </el-button>
                </el-form-item>
              </el-form>
            </div>
            <div v-else class="detail-panel">
              <p v-if="row._validationErrors.length" class="detail-error-line">
                <strong>待修正：</strong>{{ formatValidationErrors(row._validationErrors) }}
              </p>
              <p><strong>规则类型：</strong>{{ getRuleTypeText(row.rule_type || row.type) }}</p>
              <p><strong>作用范围：</strong>{{ getApplyScopeText(row.apply_scope) }}</p>
              <p><strong>时间范围：</strong>{{ getTimeScopeText(row.time_scope) }}<span v-if="row.time_offset_days !== undefined"> / 偏移 {{ row.time_offset_days }} 天</span></p>
              <p><strong>规则语义：</strong>{{ row.rule_data || '-' }}</p>
              <p><strong>{{ formatSubjectType(row) }}：</strong>{{ formatSubjectLabel(row) }}</p>
              <p><strong>班次关系：</strong>{{ formatShiftRelation(row) }}</p>
              <p><strong>适用对象：</strong>{{ formatScope(row) }}</p>
              <p><strong>来源：</strong>{{ getSourceTypeText(row.source_type) }}</p>
              <p><strong>配置类型：</strong>{{ getConfigTypeText(row) }}</p>
              <p v-for="field in getConfigFields(row)" :key="field.key">
                <strong>{{ field.label }}：</strong>{{ field.value }}
              </p>
              <p><strong>描述：</strong>{{ row.description }}</p>
            </div>
          </template>
        </el-table-column>
        <el-table-column width="45">
          <template #default="{ row }">
            <el-checkbox v-model="row._selected" />
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="category" label="类别" width="80">
          <template #default="{ row }">
            <el-tag :type="(categoryMap[row.category]?.type as any)" size="small">
              {{ categoryMap[row.category]?.label || row.category }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sub_type" label="子类型" width="100">
          <template #default="{ row }">
            {{ subTypeMap[row.sub_type] || row.sub_type }}
          </template>
        </el-table-column>
        <el-table-column label="主体班次/主体人" min-width="160">
          <template #default="{ row }">
            <span>{{ formatSubjectLabel(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="规则类型" min-width="120">
          <template #default="{ row }">
            {{ getRuleTypeText(row.rule_type || row.type) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="96">
          <template #default="{ row }">
            <el-tag v-if="row._validationErrors.length" type="danger" size="small">
              待修正
            </el-tag>
            <el-tag v-else type="success" size="small">
              可保存
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="作用范围" width="100">
          <template #default="{ row }">
            {{ getApplyScopeText(row.apply_scope) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="70" fixed="right">
          <template #default="{ $index, row }">
            <el-button v-if="!row._editing" link type="primary" size="small" @click="startEdit($index)">
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <template #footer>
      <template v-if="step === 'input'">
        <el-button @click="visible = false">
          取消
        </el-button>
        <el-button type="primary" :icon="MagicStick" @click="handleParse">
          开始解析
        </el-button>
      </template>
      <template v-else-if="step === 'streaming'">
        <el-button @click="handleCancelStream">
          取消解析
        </el-button>
      </template>
      <template v-else>
        <el-button @click="handleBack">
          返回修改
        </el-button>
        <el-button type="primary" :loading="saving" :disabled="selectedCount === 0" @click="handleSave">
          保存选中 ({{ selectedCount }})
        </el-button>
      </template>
    </template>
  </el-dialog>
</template>

<style scoped>
.parse-input-step {
  min-height: 200px;
}

.parse-streaming-step {
  min-height: 200px;
}

.streaming-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 14px;
  color: var(--el-color-primary);
  font-weight: 500;
}

.streaming-icon {
  animation: pulse 1.5s ease-in-out infinite;
}

.streaming-box {
  max-height: 360px;
  overflow-y: auto;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 14px 16px;
}

.streaming-text {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: "SF Mono", "Monaco", "Menlo", "Consolas", monospace;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-regular);
}

.streaming-section-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.streaming-reasoning {
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}

.reasoning-text {
  color: var(--el-text-color-secondary);
  font-style: italic;
}

.streaming-waiting {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.cursor-blink {
  animation: blink 0.8s step-end infinite;
  color: var(--el-color-primary);
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.edit-panel {
  padding: 12px 20px;
}

.config-editor {
  width: 100%;
}

.config-type-label {
  margin-bottom: 8px;
  color: var(--el-color-primary);
  font-weight: 600;
}

.nested-form-item {
  margin-bottom: 12px;
}

.detail-panel {
  padding: 8px 20px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.detail-panel p {
  margin: 4px 0;
}

.detail-error-line {
  color: var(--el-color-danger);
}

.config-field-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.config-field-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
}

.config-field-name {
  color: var(--el-text-color-secondary);
}

.config-field-value {
  text-align: right;
  word-break: break-word;
}

.save-error-text {
  white-space: pre-wrap;
  line-height: 1.6;
}

:deep(.invalid-rule-row) {
  --el-table-tr-bg-color: var(--el-color-danger-light-9);
}
</style>
