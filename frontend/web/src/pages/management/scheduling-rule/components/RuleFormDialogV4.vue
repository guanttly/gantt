<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { getSimpleEmployeeList } from '@/api/employee'
import { getGroupList } from '@/api/group'
import { createSchedulingRule, getSchedulingRuleDetail, updateSchedulingRule } from '@/api/scheduling-rule'
import { getShiftList } from '@/api/shift'
import { categoryOptions, subCategoryOptions } from '../logic'

interface Props {
  visible: boolean
  ruleId?: string
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

// ============================================================================
// 选项配置
// ============================================================================

// V4.1 规则类型选项（带说明）
const ruleTypeOptions = [
  { value: 'exclusive', label: '排他规则', desc: '排了A班次就不能排B班次', needsBinary: true },
  { value: 'combinable', label: '可组合规则', desc: 'A和B班次可以同时排给同一人', needsBinary: true },
  { value: 'required_together', label: '必须同时规则', desc: '排了A班次必须同时排B班次', needsBinary: true },
  { value: 'maxCount', label: '最大次数规则', desc: '某班次在时间范围内最多排N次', needsBinary: false },
  { value: 'periodic', label: '周期性规则', desc: '某班次隔N天/N周排一次', needsBinary: false },
  { value: 'forbidden_day', label: '禁止日期规则', desc: '某些日期不排指定班次', needsBinary: false },
  { value: 'preferred', label: '偏好规则', desc: '优先或避免某些班次/日期', needsBinary: false },
]

// 时间范围选项
const timeScopeOptions = [
  { value: 'same_day', label: '同一天' },
  { value: 'same_week', label: '同一周' },
  { value: 'same_month', label: '同一月' },
  { value: 'custom', label: '自定义' },
]

// 范围类型选项
const scopeTypeOptions = [
  { value: 'all', label: '全局（所有员工）' },
  { value: 'employee', label: '指定员工' },
  { value: 'group', label: '指定分组' },
  { value: 'exclude_employee', label: '排除员工' },
  { value: 'exclude_group', label: '排除分组' },
]

// ============================================================================
// 状态管理
// ============================================================================

const formRef = ref<FormInstance>()
const loading = ref(false)
const detailLoading = ref(false)

// 班次列表
const shiftList = ref<Shift.ShiftInfo[]>([])
const shiftLoading = ref(false)

// 员工列表
const employeeList = ref<Employee.SimpleEmployeeInfo[]>([])
const employeeLoading = ref(false)

// 分组列表
const groupList = ref<Group.GroupInfo[]>([])
const groupLoading = ref(false)

// 是否正在加载详情（用于防止 watcher 清空数据）
const isLoadingDetail = ref(false)

// 表单数据
interface FormData {
  name: string
  ruleType: SchedulingRule.RuleType | ''
  timeScope: SchedulingRule.TimeScope | ''
  priority: number
  description: string
  category: SchedulingRule.Category | ''
  subCategory: SchedulingRule.SubCategory | ''
  // 数值参数
  timeOffsetDays?: number
  maxCount?: number
  consecutiveMax?: number
  intervalDays?: number
  minRestDays?: number
  // V4.1 班次关系
  subjectShifts: string[] // 主体班次ID列表
  objectShifts: string[] // 客体班次ID列表
  targetShifts: string[] // 目标班次ID列表
  // V4.1 适用范围
  scopeType: SchedulingRule.ScopeType
  scopeEmployees: string[] // 指定/排除的员工ID列表
  scopeGroups: string[] // 指定/排除的分组ID列表
  // forbidden_day 专用
  forbiddenWeekdays: number[] // 禁止的星期几（Go约定：0=周日，1=周一...6=周六），timeScope=same_week 时使用
  targetDates: string[] // 禁止的具体日期（YYYY-MM-DD），timeScope=custom/same_day 时使用
  forbiddenMonthDays: number[] // 禁止的每月内日期（1～31），timeScope=same_month 时使用
}

const formData = reactive<FormData>({
  name: '',
  ruleType: '',
  category: '',
  subCategory: '',
  timeScope: 'same_day',
  priority: 5,
  description: '',
  timeOffsetDays: 0,
  subjectShifts: [],
  objectShifts: [],
  targetShifts: [],
  scopeType: 'all',
  scopeEmployees: [],
  scopeGroups: [],
  forbiddenWeekdays: [],
  targetDates: [],
  forbiddenMonthDays: [],
})

// ============================================================================
// 计算属性
// ============================================================================

const dialogTitle = computed(() => props.ruleId ? '编辑排班规则' : '新增排班规则')
const isEdit = computed(() => !!props.ruleId)

// 当前规则类型是否需要二元关系（主体+客体）
const needsBinaryRelation = computed(() => {
  const option = ruleTypeOptions.find(o => o.value === formData.ruleType)
  return option?.needsBinary ?? false
})

// 当前规则类型是否需要目标班次
const needsTargetShift = computed(() => {
  return ['maxCount', 'periodic', 'forbidden_day', 'preferred'].includes(formData.ruleType)
})

// 是否显示数值参数
const showMaxCount = computed(() => formData.ruleType === 'maxCount')
const showConsecutiveMax = computed(() => formData.ruleType === 'maxCount')
const showIntervalDays = computed(() => formData.ruleType === 'periodic')
const showMinRestDays = computed(() => ['maxCount', 'periodic'].includes(formData.ruleType))

// forbidden_day 专用：显示星期几选择 or 具体日期选择
const showForbiddenWeekdays = computed(() =>
  formData.ruleType === 'forbidden_day' && formData.timeScope === 'same_week',
)
const showTargetDates = computed(() =>
  formData.ruleType === 'forbidden_day' && (formData.timeScope === 'custom' || formData.timeScope === 'same_day'),
)
const showForbiddenMonthDays = computed(() =>
  formData.ruleType === 'forbidden_day' && formData.timeScope === 'same_month',
)

// 月内日期选项（1～31日）
const monthDayOptions = Array.from({ length: 31 }, (_, i) => ({ value: i + 1, label: `${i + 1}日` }))

// 星期几选项（顺序：周一~周日，value 使用 Go time.Weekday 约定：0=周日,1=周一...6=周六）
const weekdayOptions = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 0, label: '周日' },
]

// 范围是否需要选择对象
const needsScopeSelection = computed(() => {
  return ['employee', 'group', 'exclude_employee', 'exclude_group'].includes(formData.scopeType)
})

// 当前规则类型的描述
const currentRuleTypeDesc = computed(() => {
  const option = ruleTypeOptions.find(o => o.value === formData.ruleType)
  return option?.desc ?? ''
})

// ============================================================================
// 表单验证
// ============================================================================

function validateScopeSelection(_rule: any, _value: any, callback: any) {
  if (needsScopeSelection.value) {
    if (['employee', 'exclude_employee'].includes(formData.scopeType) && formData.scopeEmployees.length === 0) {
      callback(new Error('请选择至少一个员工'))
      return
    }
    if (['group', 'exclude_group'].includes(formData.scopeType) && formData.scopeGroups.length === 0) {
      callback(new Error('请选择至少一个分组'))
      return
    }
  }
  callback()
}

function validateSubjectShifts(_rule: any, value: any, callback: any) {
  if (needsBinaryRelation.value) {
    if (!value || value.length === 0) {
      callback(new Error('请选择至少一个主体班次'))
      return
    }
  }
  callback()
}

function validateObjectShifts(_rule: any, value: any, callback: any) {
  if (needsBinaryRelation.value) {
    if (!value || value.length === 0) {
      callback(new Error('请选择至少一个客体班次'))
      return
    }
  }
  callback()
}

function validateTargetShifts(_rule: any, value: any, callback: any) {
  if (needsTargetShift.value) {
    if (!value || value.length === 0) {
      callback(new Error('请选择至少一个目标班次'))
      return
    }
  }
  callback()
}

function validateForbiddenWeekdays(_rule: any, _value: any, callback: any) {
  if (showForbiddenWeekdays.value && formData.forbiddenWeekdays.length === 0) {
    callback(new Error('请至少选择一个禁止的星期几'))
    return
  }
  callback()
}

function validateTargetDates(_rule: any, _value: any, callback: any) {
  if (showTargetDates.value && formData.targetDates.length === 0) {
    callback(new Error('请至少选择一个禁止的日期'))
    return
  }
  callback()
}

function validateForbiddenMonthDays(_rule: any, _value: any, callback: any) {
  if (showForbiddenMonthDays.value && formData.forbiddenMonthDays.length === 0) {
    callback(new Error('请至少选择一个禁止的每月日期'))
    return
  }
  callback()
}

const rules: FormRules<FormData> = {
  name: [
    { required: true, message: '请输入规则名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' },
  ],
  ruleType: [
    { required: true, message: '请选择规则类型', trigger: 'change' },
  ],
  category: [
    { required: true, message: '请选择分类', trigger: 'change' },
  ],
  subCategory: [
    { required: true, message: '请选择子分类', trigger: 'change' },
  ],
  timeScope: [
    { required: true, message: '请选择时间范围', trigger: 'change' },
  ],
  priority: [
    { required: true, message: '请输入优先级', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '优先级范围 1-10', trigger: 'blur' },
  ],
  subjectShifts: [
    { validator: validateSubjectShifts, trigger: 'change' },
  ],
  objectShifts: [
    { validator: validateObjectShifts, trigger: 'change' },
  ],
  targetShifts: [
    { validator: validateTargetShifts, trigger: 'change' },
  ],
  scopeType: [
    { validator: validateScopeSelection, trigger: 'change' },
  ],
  forbiddenWeekdays: [
    { validator: validateForbiddenWeekdays, trigger: 'change' },
  ],
  targetDates: [
    { validator: validateTargetDates, trigger: 'change' },
  ],
  forbiddenMonthDays: [
    { validator: validateForbiddenMonthDays, trigger: 'change' },
  ],
}

// ============================================================================
// 数据加载
// ============================================================================

async function loadShifts() {
  if (shiftList.value.length > 0)
    return
  shiftLoading.value = true
  try {
    const res = await getShiftList({ orgId: props.orgId, page: 1, size: 200 })
    shiftList.value = res.items || []
  }
  catch {
    ElMessage.error('加载班次列表失败')
  }
  finally {
    shiftLoading.value = false
  }
}

async function loadEmployees() {
  if (employeeList.value.length > 0)
    return
  employeeLoading.value = true
  try {
    const res = await getSimpleEmployeeList({ orgId: props.orgId, page: 1, size: 500 })
    employeeList.value = res.items || []
  }
  catch {
    ElMessage.error('加载员工列表失败')
  }
  finally {
    employeeLoading.value = false
  }
}

async function loadGroups() {
  if (groupList.value.length > 0)
    return
  groupLoading.value = true
  try {
    const res = await getGroupList({ orgId: props.orgId, page: 1, size: 100 })
    groupList.value = res.items || []
  }
  catch {
    ElMessage.error('加载分组列表失败')
  }
  finally {
    groupLoading.value = false
  }
}

async function loadRuleDetail() {
  if (!props.ruleId)
    return
  detailLoading.value = true
  isLoadingDetail.value = true // 标记正在加载详情
  try {
    const detail = await getSchedulingRuleDetail(props.ruleId, props.orgId)
    formData.name = detail.name
    formData.ruleType = detail.ruleType
    formData.category = detail.category || ''
    formData.subCategory = detail.subCategory || ''
    formData.timeScope = detail.timeScope
    formData.timeOffsetDays = detail.timeOffsetDays || 0
    formData.priority = detail.priority
    formData.description = detail.description || ''

    // 使用 nextTick 确保 ruleType 的 watcher 先执行完毕
    await nextTick()

    // 从 associations 中提取班次关联（按 associationType='shift' 过滤，用 role 区分角色）
    if (detail.associations) {
      const shiftAssocs = detail.associations.filter(a => a.associationType === 'shift')
      formData.subjectShifts = shiftAssocs
        .filter(a => a.role === 'subject')
        .map(a => a.associationId)
      formData.objectShifts = shiftAssocs
        .filter(a => a.role === 'object')
        .map(a => a.associationId)
      formData.targetShifts = shiftAssocs
        .filter(a => a.role === 'target')
        .map(a => a.associationId)
    }

    // 解析 forbidden_day 规则的 ruleData
    if (detail.ruleType === 'forbidden_day' && detail.ruleData) {
      try {
        const rd = JSON.parse(detail.ruleData)
        if (Array.isArray(rd.forbiddenWeekdays))
          formData.forbiddenWeekdays = rd.forbiddenWeekdays
        if (Array.isArray(rd.targetDates))
          formData.targetDates = rd.targetDates
        if (Array.isArray(rd.forbiddenMonthDays))
          formData.forbiddenMonthDays = rd.forbiddenMonthDays
      }
      catch { /* ruleData 为旧版文本格式，忽略 */ }
    }

    if (detail.applyScopes && detail.applyScopes.length > 0) {
      const firstScope = detail.applyScopes[0]
      formData.scopeType = firstScope.scopeType
      if (['employee', 'exclude_employee'].includes(firstScope.scopeType)) {
        formData.scopeEmployees = detail.applyScopes.map(s => s.scopeId!).filter(Boolean)
      }
      if (['group', 'exclude_group'].includes(firstScope.scopeType)) {
        formData.scopeGroups = detail.applyScopes.map(s => s.scopeId!).filter(Boolean)
      }
    }
  }
  catch {
    ElMessage.error('加载规则详情失败')
    handleClose()
  }
  finally {
    detailLoading.value = false
    isLoadingDetail.value = false // 加载完成
  }
}

// ============================================================================
// 监听器
// ============================================================================

watch(() => props.visible, async (val) => {
  if (val) {
    await loadShifts()
    if (props.ruleId) {
      await loadRuleDetail()
    }
    else {
      resetForm()
    }
  }
})

watch(() => formData.ruleType, (newVal, oldVal) => {
  // 加载详情时不清空班次选择，只有用户手动切换规则类型时才清空
  // 条件：不在加载详情中，且确实发生了切换（旧值存在）
  if (isLoadingDetail.value || !oldVal) {
    return
  }
  // 切换规则类型时，清空班次选择
  formData.subjectShifts = []
  formData.objectShifts = []
  formData.targetShifts = []
})

watch(() => formData.scopeType, () => {
  // 切换范围类型时，加载对应数据
  if (['employee', 'exclude_employee'].includes(formData.scopeType)) {
    loadEmployees()
    formData.scopeGroups = []
  }
  if (['group', 'exclude_group'].includes(formData.scopeType)) {
    loadGroups()
    formData.scopeEmployees = []
  }
  if (formData.scopeType === 'all') {
    formData.scopeEmployees = []
    formData.scopeGroups = []
  }
})

// ============================================================================
// 表单操作
// ============================================================================

async function handleSubmit() {
  if (!formRef.value)
    return

  await formRef.value.validate(async (valid) => {
    if (!valid)
      return

    loading.value = true
    try {
      // 构建关联数据
      const associations: SchedulingRule.AssociationInput[] = []

      if (needsBinaryRelation.value) {
        formData.subjectShifts.forEach((shiftId) => {
          associations.push({ associationType: 'shift', associationId: shiftId, role: 'subject' })
        })
        formData.objectShifts.forEach((shiftId) => {
          associations.push({ associationType: 'shift', associationId: shiftId, role: 'object' })
        })
      }

      if (needsTargetShift.value) {
        formData.targetShifts.forEach((shiftId) => {
          associations.push({ associationType: 'shift', associationId: shiftId, role: 'target' })
        })
      }

      // 构建适用范围
      const applyScopes: SchedulingRule.ApplyScopeInput[] = []

      if (formData.scopeType === 'all') {
        applyScopes.push({ scopeType: 'all' })
      }
      else if (['employee', 'exclude_employee'].includes(formData.scopeType)) {
        formData.scopeEmployees.forEach((empId) => {
          const emp = employeeList.value.find(e => e.id === empId)
          applyScopes.push({
            scopeType: formData.scopeType,
            scopeId: empId,
            scopeName: emp?.name,
          })
        })
      }
      else if (['group', 'exclude_group'].includes(formData.scopeType)) {
        formData.scopeGroups.forEach((groupId) => {
          const group = groupList.value.find(g => g.id === groupId)
          applyScopes.push({
            scopeType: formData.scopeType,
            scopeId: groupId,
            scopeName: group?.name,
          })
        })
      }

      // 构建规则描述
      const ruleData = buildRuleDescription()

      const requestData: SchedulingRule.CreateRequest = {
        orgId: props.orgId,
        name: formData.name,
        ruleType: formData.ruleType as SchedulingRule.RuleType,
        category: formData.category as SchedulingRule.Category,
        subCategory: formData.subCategory as SchedulingRule.SubCategory,
        applyScope: formData.scopeType === 'all' ? 'global' : 'specific',
        timeScope: formData.timeScope as SchedulingRule.TimeScope,
        timeOffsetDays: formData.timeOffsetDays,
        priority: formData.priority,
        ruleData,
        description: formData.description,
        version: 'v4',
        sourceType: 'manual',
        associations,
        applyScopes,
      }

      if (isEdit.value) {
        await updateSchedulingRule(props.ruleId!, props.orgId, {
          name: requestData.name,
          priority: requestData.priority,
          ruleData: requestData.ruleData,
          description: requestData.description,
          associations,
          applyScopes,
        })
        ElMessage.success('更新成功')
      }
      else {
        await createSchedulingRule(requestData)
        ElMessage.success('创建成功')
      }

      emit('success')
      handleClose()
    }
    catch {
      ElMessage.error(isEdit.value ? '更新失败' : '创建失败')
    }
    finally {
      loading.value = false
    }
  })
}

function buildRuleDescription(): string {
  const parts: string[] = []
  const timeScopeLabel = timeScopeOptions.find(o => o.value === formData.timeScope)?.label || ''

  if (needsBinaryRelation.value) {
    const subjectNames = formData.subjectShifts.map(id =>
      shiftList.value.find(s => s.id === id)?.name || id,
    ).join('、')
    const objectNames = formData.objectShifts.map(id =>
      shiftList.value.find(s => s.id === id)?.name || id,
    ).join('、')

    if (formData.ruleType === 'exclusive') {
      let offsetStr = ''
      if (formData.timeOffsetDays)
        offsetStr = `的${formData.timeOffsetDays > 0 ? `${formData.timeOffsetDays}天后` : `${Math.abs(formData.timeOffsetDays)}天前`}`
      parts.push(`排了[${subjectNames}]${offsetStr}就不能排[${objectNames}]`)
    }
    else if (formData.ruleType === 'combinable') {
      let offsetStr = ''
      if (formData.timeOffsetDays)
        offsetStr = `的${formData.timeOffsetDays > 0 ? `${formData.timeOffsetDays}天后` : `${Math.abs(formData.timeOffsetDays)}天前`}`
      parts.push(`[${subjectNames}]和${offsetStr}[${objectNames}]可以同时排给同一人`)
    }
    else if (formData.ruleType === 'required_together') {
      let offsetStr = ''
      if (formData.timeOffsetDays)
        offsetStr = `的${formData.timeOffsetDays > 0 ? `${formData.timeOffsetDays}天后` : `${Math.abs(formData.timeOffsetDays)}天前`}`
      parts.push(`排了[${subjectNames}]${offsetStr}必须同时排[${objectNames}]`)
    }
  }

  if (needsTargetShift.value) {
    const targetNames = formData.targetShifts.map(id =>
      shiftList.value.find(s => s.id === id)?.name || id,
    ).join('、')

    if (formData.ruleType === 'maxCount' && formData.maxCount) {
      parts.push(`[${targetNames}]${timeScopeLabel}最多${formData.maxCount}次`)
    }
    if (formData.ruleType === 'periodic' && formData.intervalDays) {
      parts.push(`[${targetNames}]每隔${formData.intervalDays}天排一次`)
    }
    if (formData.ruleType === 'forbidden_day') {
      // forbidden_day 的 ruleData 直接返回 JSON，供后端解析
      if (showForbiddenWeekdays.value && formData.forbiddenWeekdays.length > 0) {
        const dayNames = formData.forbiddenWeekdays
          .map(v => weekdayOptions.find(o => o.value === v)?.label || v)
          .join('、')
        parts.push(`[${targetNames}]禁止在${dayNames}排班`)
        return JSON.stringify({ forbiddenWeekdays: formData.forbiddenWeekdays })
      }
      else if (showTargetDates.value && formData.targetDates.length > 0) {
        parts.push(`[${targetNames}]禁止在指定日期排班`)
        return JSON.stringify({ targetDates: formData.targetDates })
      }
    }
  }

  return parts.join('；')
}

function resetForm() {
  formData.name = ''
  formData.ruleType = ''
  formData.category = ''
  formData.subCategory = ''
  formData.timeScope = 'same_day'
  formData.priority = 5
  formData.description = ''
  formData.timeOffsetDays = 0
  formData.maxCount = undefined
  formData.consecutiveMax = undefined
  formData.intervalDays = undefined
  formData.minRestDays = undefined
  formData.subjectShifts = []
  formData.objectShifts = []
  formData.targetShifts = []
  formData.scopeType = 'all'
  formData.scopeEmployees = []
  formData.scopeGroups = []
  formData.forbiddenWeekdays = []
  formData.targetDates = []
  formData.forbiddenMonthDays = []

  nextTick(() => {
    formRef.value?.clearValidate()
  })
}

function handleClose() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="800px"
    top="3vh"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      v-loading="detailLoading"
      :model="formData"
      :rules="rules"
      label-width="100px"
      class="dialog-form"
    >
      <!-- 基本信息 -->
      <el-divider content-position="left">
        基本信息
      </el-divider>

      <el-form-item label="规则名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="请输入规则名称"
          maxlength="50"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="规则类型" prop="ruleType">
        <el-select
          v-model="formData.ruleType"
          placeholder="请选择规则类型"
          :disabled="isEdit"
          style="width: 100%"
        >
          <el-option
            v-for="item in ruleTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          >
            <div class="rule-type-option">
              <span>{{ item.label }}</span>
              <span class="rule-type-desc">{{ item.desc }}</span>
            </div>
          </el-option>
        </el-select>
        <div v-if="currentRuleTypeDesc" class="form-tip">
          {{ currentRuleTypeDesc }}
        </div>
      </el-form-item>

      <el-form-item label="规则分类" prop="category">
        <el-select
          v-model="formData.category"
          placeholder="请选择规则分类"
          style="width: 100%"
        >
          <el-option
            v-for="item in categoryOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="规则子分类" prop="subCategory">
        <el-select
          v-model="formData.subCategory"
          placeholder="请选择规则子分类"
          style="width: 100%"
        >
          <el-option
            v-for="item in subCategoryOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="时间范围" prop="timeScope">
        <el-select
          v-model="formData.timeScope"
          placeholder="请选择时间范围"
          style="width: 100%"
        >
          <el-option
            v-for="item in timeScopeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="优先级" prop="priority">
        <el-input-number
          v-model="formData.priority"
          :min="1"
          :max="10"
          :step="1"
          style="width: 200px"
        />
        <span class="form-tip-inline">数字越大优先级越高</span>
      </el-form-item>

      <!-- 班次关系配置 - 二元关系 -->
      <template v-if="needsBinaryRelation">
        <el-divider content-position="left">
          班次关系配置
        </el-divider>

        <el-form-item label="主体班次" prop="subjectShifts" required>
          <el-select
            v-model="formData.subjectShifts"
            multiple
            placeholder="选择主体班次（触发规则的班次）"
            style="width: 100%"
            :loading="shiftLoading"
            filterable
          >
            <el-option
              v-for="shift in shiftList"
              :key="shift.id"
              :label="shift.name"
              :value="shift.id"
              :disabled="formData.objectShifts.includes(shift.id)"
            />
          </el-select>
          <div class="form-tip">
            <template v-if="formData.ruleType === 'exclusive'">
              排了这些班次后，将不能再排客体班次
            </template>
            <template v-else-if="formData.ruleType === 'combinable'">
              这些班次可以和客体班次同时排给同一人
            </template>
            <template v-else-if="formData.ruleType === 'required_together'">
              排了这些班次后，必须同时排客体班次
            </template>
          </div>
        </el-form-item>

        <el-form-item label="客体班次" prop="objectShifts" required>
          <el-select
            v-model="formData.objectShifts"
            multiple
            placeholder="选择客体班次（被约束的班次）"
            style="width: 100%"
            :loading="shiftLoading"
            filterable
          >
            <el-option
              v-for="shift in shiftList"
              :key="shift.id"
              :label="shift.name"
              :value="shift.id"
              :disabled="formData.subjectShifts.includes(shift.id)"
            />
          </el-select>
          <div class="form-tip">
            <template v-if="formData.ruleType === 'exclusive'">
              当主体班次被排后，这些班次不能再排
            </template>
            <template v-else-if="formData.ruleType === 'combinable'">
              这些班次可以和主体班次同时排
            </template>
            <template v-else-if="formData.ruleType === 'required_together'">
              当主体班次被排后，这些班次必须同时被排
            </template>
          </div>
        </el-form-item>

        <el-form-item label="偏移天数" prop="timeOffsetDays">
          <el-input-number
            v-model="formData.timeOffsetDays"
            :step="1"
            placeholder="时间偏移天数"
            style="width: 200px"
          />
          <span class="form-tip-inline">客体班次发生在主体班次几天后，负数表示提前（如：排了夜班后，昨天是早班则为 -1）</span>
        </el-form-item>
      </template>

      <!-- 班次关系配置 - 单班次 -->
      <template v-if="needsTargetShift">
        <el-divider content-position="left">
          目标班次配置
        </el-divider>

        <el-form-item label="目标班次" prop="targetShifts" required>
          <el-select
            v-model="formData.targetShifts"
            multiple
            placeholder="选择规则作用的目标班次"
            style="width: 100%"
            :loading="shiftLoading"
            filterable
          >
            <el-option
              v-for="shift in shiftList"
              :key="shift.id"
              :label="shift.name"
              :value="shift.id"
            />
          </el-select>
        </el-form-item>

        <!-- 数值参数 -->
        <el-form-item v-if="showMaxCount" label="最大次数">
          <el-input-number
            v-model="formData.maxCount"
            :min="1"
            :max="100"
            placeholder="最大排班次数"
            style="width: 200px"
          />
          <span class="form-tip-inline">在时间范围内最多排班次数</span>
        </el-form-item>

        <el-form-item v-if="showConsecutiveMax" label="连续上限">
          <el-input-number
            v-model="formData.consecutiveMax"
            :min="1"
            :max="30"
            placeholder="连续天数上限"
            style="width: 200px"
          />
          <span class="form-tip-inline">最多连续排班天数</span>
        </el-form-item>

        <el-form-item v-if="showIntervalDays" label="间隔天数">
          <el-input-number
            v-model="formData.intervalDays"
            :min="1"
            :max="365"
            placeholder="间隔天数"
            style="width: 200px"
          />
          <span class="form-tip-inline">每隔多少天排一次（7=每周，14=隔周）</span>
        </el-form-item>

        <el-form-item v-if="showMinRestDays" label="最少休息">
          <el-input-number
            v-model="formData.minRestDays"
            :min="0"
            :max="30"
            placeholder="最少休息天数"
            style="width: 200px"
          />
          <span class="form-tip-inline">排班后最少休息天数</span>
        </el-form-item>

        <!-- forbidden_day：禁止星期几（same_week） -->
        <el-form-item v-if="showForbiddenWeekdays" label="禁止星期" prop="forbiddenWeekdays" required>
          <el-checkbox-group v-model="formData.forbiddenWeekdays">
            <el-checkbox
              v-for="wd in weekdayOptions"
              :key="wd.value"
              :value="wd.value"
              :label="wd.label"
            />
          </el-checkbox-group>
          <div class="form-tip">
            选中的星期几将禁止排该目标班次（可多选）
          </div>
        </el-form-item>

        <!-- forbidden_day：禁止每月内日期（same_month） -->
        <el-form-item v-if="showForbiddenMonthDays" label="禁止日" prop="forbiddenMonthDays" required>
          <el-select
            v-model="formData.forbiddenMonthDays"
            multiple
            placeholder="选择禁止排班的每月日期（可多选）"
            style="width: 100%"
          >
            <el-option
              v-for="d in monthDayOptions"
              :key="d.value"
              :label="d.label"
              :value="d.value"
            />
          </el-select>
          <div class="form-tip">
            选中的日期（如 1日、15日）每月均禁止排该目标班次
          </div>
        </el-form-item>

        <!-- forbidden_day：禁止具体日期（custom / same_day） -->
        <el-form-item v-if="showTargetDates" :label="formData.timeScope === 'same_day' ? '禁止当天' : '禁止日期'" prop="targetDates" required>
          <el-date-picker
            v-model="formData.targetDates"
            type="dates"
            placeholder="选择禁止排班的日期（可多选）"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
          <div class="form-tip">
            所选日期将禁止排该目标班次
          </div>
        </el-form-item>
      </template>

      <!-- 适用范围配置 -->
      <el-divider content-position="left">
        适用范围
      </el-divider>

      <el-form-item label="范围类型" prop="scopeType">
        <el-select
          v-model="formData.scopeType"
          placeholder="选择规则适用范围"
          style="width: 100%"
        >
          <el-option
            v-for="item in scopeTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item
        v-if="['employee', 'exclude_employee'].includes(formData.scopeType)"
        :label="formData.scopeType === 'employee' ? '指定员工' : '排除员工'"
        required
      >
        <el-select
          v-model="formData.scopeEmployees"
          multiple
          placeholder="选择员工"
          style="width: 100%"
          :loading="employeeLoading"
          filterable
        >
          <el-option
            v-for="emp in employeeList"
            :key="emp.id"
            :label="emp.name"
            :value="emp.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item
        v-if="['group', 'exclude_group'].includes(formData.scopeType)"
        :label="formData.scopeType === 'group' ? '指定分组' : '排除分组'"
        required
      >
        <el-select
          v-model="formData.scopeGroups"
          multiple
          placeholder="选择分组"
          style="width: 100%"
          :loading="groupLoading"
          filterable
        >
          <el-option
            v-for="group in groupList"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>

      <!-- 描述 -->
      <el-divider content-position="left">
        补充说明
      </el-divider>

      <el-form-item label="描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入规则的补充说明（可选）"
          maxlength="200"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">
        {{ isEdit ? '更新' : '创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.dialog-form {
  max-height: calc(85vh - 120px);
  overflow-y: auto;
  padding-right: 8px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.5;
}

.form-tip-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 12px;
}

.rule-type-option {
  display: flex;
  flex-direction: column;

  .rule-type-desc {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

:deep(.el-divider__text) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

:deep(.el-select-dropdown__item) {
  height: auto;
  padding: 8px 12px;
}
</style>
