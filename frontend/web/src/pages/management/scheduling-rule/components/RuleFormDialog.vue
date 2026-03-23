<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { createSchedulingRule, getSchedulingRuleDetail, updateSchedulingRule } from '@/api/scheduling-rule'
import { applyScopeOptions, ruleTypeOptions, timeScopeOptions } from '../logic'
import { normalizeRuleType, normalizeApplyScope, normalizeTimeScope, denormalizeRuleType, denormalizeApplyScope, denormalizeTimeScope } from '../v3-compat'

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

// 表单引用
const formRef = ref<FormInstance>()
const loading = ref(false)
const detailLoading = ref(false)
const isLoadingDetail = ref(false) // 标记是否正在加载详情

// 表单数据
interface FormData {
  name: string
  ruleType: SchedulingRule.RuleType | '' // V4 枚举值
  applyScope: SchedulingRule.ApplyScope | '' // V4 枚举值
  timeScope: SchedulingRule.TimeScope | '' // V4 枚举值
  priority: number
  description: string
  ruleData: string // 规则的语义化描述
  // V4新增字段
  category?: SchedulingRule.Category
  subCategory?: SchedulingRule.SubCategory
  sourceType?: 'manual' | 'llm_parsed' | 'migrated'
  version?: 'v3' | 'v4'
}

const formData = reactive<FormData>({
  name: '',
  ruleType: '',
  applyScope: '',
  timeScope: '',
  priority: 5,
  description: '',
  ruleData: '',
  version: 'v4', // 默认 v4
  sourceType: 'manual', // 默认手动创建
})

// 对话框标题
const dialogTitle = computed(() => {
  return props.ruleId ? '编辑排班规则' : '新增排班规则'
})

// 是否为编辑模式
const isEdit = computed(() => !!props.ruleId)

// 表单验证规则
const rules: FormRules<FormData> = {
  name: [
    { required: true, message: '请输入规则名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' },
  ],
  ruleType: [
    { required: true, message: '请选择规则类型', trigger: 'change' },
  ],
  applyScope: [
    { required: true, message: '请选择应用范围', trigger: 'change' },
  ],
  timeScope: [
    { required: true, message: '请选择时间范围', trigger: 'change' },
  ],
  priority: [
    { required: true, message: '请输入优先级', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '优先级范围 1-10', trigger: 'blur' },
  ],
  ruleData: [
    { required: true, message: '请输入规则配置', trigger: 'blur' },
  ],
}

// 根据规则类型获取规则配置的提示文本
const ruleDataPlaceholder = computed(() => {
  switch (formData.ruleType) {
    case 'max_shifts':
      return '例如：每周最多工作5天\n每月最多工作22天'
    case 'consecutive_shifts':
      return '例如：最多连续工作6天\n最少连续工作2天后休息'
    case 'rest_days':
      return '例如：每周至少休息1天\n每月至少休息4天'
    case 'forbidden_pattern':
      return '例如：禁止连续上夜班\n禁止早班后立即上夜班'
    case 'preferred_pattern':
      return '例如：优先安排周末休息\n优先安排固定班次'
    default:
      return '请输入规则的语义化描述'
  }
})

// 根据规则类型获取规则配置的示例
const ruleDataExamples = computed(() => {
  switch (formData.ruleType) {
    case 'max_shifts':
      return [
        '每天最多1个班次',
        '每周最多5个班次',
        '每月最多22个班次',
      ]
    case 'consecutive_shifts':
      return [
        '最多连续工作6天',
        '最少连续工作2天',
        '连续工作不超过48小时',
      ]
    case 'rest_days':
      return [
        '每周至少休息1天',
        '每月至少休息4天',
        '连续工作后至少休息1天',
      ]
    case 'forbidden_pattern':
      return [
        '禁止连续上夜班',
        '禁止早班后立即上夜班',
        '禁止周末加班',
      ]
    case 'preferred_pattern':
      return [
        '优先安排周末休息',
        '优先安排固定班次',
        '优先上午班',
      ]
    default:
      return []
  }
})

// 监听对话框打开
watch(() => props.visible, (val) => {
  if (val) {
    if (props.ruleId) {
      loadRuleDetail()
    }
    else {
      resetForm()
    }
  }
})

// 监听规则类型变化，重置ruleData（仅在非加载详情时）
watch(() => formData.ruleType, () => {
  // 如果是正在加载详情，不清空 ruleData
  if (!isLoadingDetail.value) {
    formData.ruleData = ''
  }
})

// 加载规则详情
async function loadRuleDetail() {
  if (!props.ruleId)
    return

  detailLoading.value = true
  isLoadingDetail.value = true // 标记开始加载详情
  try {
    const detail = await getSchedulingRuleDetail(props.ruleId, props.orgId)
    formData.name = detail.name
    // 后端返回的是 V4 枚举值，直接使用
    formData.ruleType = detail.ruleType as SchedulingRule.RuleType
    formData.applyScope = detail.applyScope as SchedulingRule.ApplyScope
    formData.timeScope = detail.timeScope as SchedulingRule.TimeScope
    formData.priority = detail.priority
    formData.description = detail.description || ''
    formData.ruleData = detail.ruleData || ''
    // V4新增字段
    if ('category' in detail) {
      formData.category = (detail as any).category
      formData.subCategory = (detail as any).subCategory
      formData.sourceType = (detail as any).sourceType
      formData.version = (detail as any).version || 'v4'
    }
  }
  catch {
    ElMessage.error('加载规则详情失败')
    handleClose()
  }
  finally {
    detailLoading.value = false
    // 使用 nextTick 确保所有响应式更新完成后再重置标记
    nextTick(() => {
      isLoadingDetail.value = false
    })
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value)
    return

  await formRef.value.validate(async (valid) => {
    if (!valid)
      return

    loading.value = true
    try {
      // @deprecated V3: 规范化 V3 枚举值
      const normalizedRuleType = normalizeRuleType(formData.ruleType) as SchedulingRule.RuleType
      const normalizedApplyScope = normalizeApplyScope(formData.applyScope) as SchedulingRule.ApplyScope
      const normalizedTimeScope = normalizeTimeScope(formData.timeScope) as SchedulingRule.TimeScope

      const requestData = {
        orgId: props.orgId,
        name: formData.name,
        ruleType: normalizedRuleType,
        applyScope: normalizedApplyScope,
        timeScope: normalizedTimeScope,
        priority: formData.priority,
        ruleData: formData.ruleData,
        description: formData.description,
        version: 'v4', // 新创建的规则默认为 v4
        sourceType: 'manual', // 手动创建的规则
      }

      if (isEdit.value) {
        await updateSchedulingRule(props.ruleId!, props.orgId, {
          name: requestData.name,
          priority: requestData.priority,
          ruleData: requestData.ruleData,
          description: requestData.description,
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

// 重置表单
function resetForm() {
  formData.name = ''
  formData.ruleType = ''
  formData.applyScope = ''
  formData.timeScope = ''
  formData.priority = 5
  formData.description = ''
  formData.ruleData = ''

  nextTick(() => {
    formRef.value?.clearValidate()
  })
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="700px"
    top="5vh"
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
          />
        </el-select>
      </el-form-item>

      <el-form-item label="应用范围" prop="applyScope">
        <el-select
          v-model="formData.applyScope"
          placeholder="请选择应用范围"
          :disabled="isEdit"
          style="width: 100%"
        >
          <el-option
            v-for="item in applyScopeOptions"
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
          style="width: 100%"
        />
        <div class="form-tip">
          优先级范围 1-10，数字越大优先级越高
        </div>
      </el-form-item>

      <el-form-item label="描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入规则描述"
          maxlength="200"
          show-word-limit
        />
      </el-form-item>

      <!-- 规则配置 -->
      <el-divider content-position="left">
        规则配置
      </el-divider>

      <el-form-item label="规则内容" prop="ruleData">
        <el-input
          v-model="formData.ruleData"
          type="textarea"
          :rows="6"
          :placeholder="ruleDataPlaceholder"
          maxlength="500"
          show-word-limit
        />
        <div v-if="ruleDataExamples.length > 0" class="form-tip">
          <div style="margin-bottom: 4px;">
            示例：
          </div>
          <div
            v-for="(example, index) in ruleDataExamples"
            :key="index"
            class="example-item"
          >
            • {{ example }}
          </div>
        </div>
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
  // max-height: calc(85vh - 120px);
  overflow-y: auto;
  padding-right: 8px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.5;
}

.example-item {
  color: var(--el-color-info);
  margin-left: 8px;
  line-height: 1.8;
}

:deep(.el-divider__text) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
