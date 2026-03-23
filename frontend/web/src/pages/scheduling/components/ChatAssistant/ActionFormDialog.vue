<script setup lang="ts">
import type { WorkflowAction } from '@/types/chat'
import { Calendar } from '@element-plus/icons-vue'
import { ElButton, ElDatePicker, ElDialog, ElInput, ElInputNumber, ElMessage, ElOption, ElSelect } from 'element-plus'
import { computed, h, reactive, watch } from 'vue'
import DailyGridField from './DailyGridField.vue'
import MultiSelectField from './MultiSelectField.vue'
import TableFormField from './TableFormField.vue'

const props = defineProps<{
  visible: boolean
  action: WorkflowAction | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [formData: Record<string, any>]
  'cancel': []
}>()

// 表单数据
const formData = reactive<Record<string, any>>({})

// 折叠状态管理 (shiftId -> collapsed)
const collapseMap = reactive<Record<string, boolean>>({})

// 分离不同类型的字段
const topFields = computed(() => {
  if (!props.action?.fields)
    return []
  return props.action.fields.filter(f => f.name === 'period_info')
})

// daily-grid 类型字段（按天人数配置）
const dailyGridFields = computed(() => {
  if (!props.action?.fields)
    return []
  return props.action.fields.filter(f => f.type === 'daily-grid')
})

// table-form 类型字段（表格+表单组合）
const tableFormFields = computed(() => {
  if (!props.action?.fields)
    return []
  return props.action.fields.filter(f => f.type === 'table-form')
})

const regularFields = computed(() => {
  if (!props.action?.fields)
    return []
  // 排除特殊字段
  return props.action.fields.filter(f =>
    f.name !== 'period_info'
    && f.type !== 'daily-grid'
    && f.type !== 'table-form',
  )
})

// 初始化表单数据
function initFormData() {
  if (!props.action?.fields)
    return

  Object.keys(formData).forEach(key => delete formData[key])
  props.action.fields.forEach((field) => {
    if (field.type === 'table-form') {
      // table-form 类型：如果有 initialItems，使用它们；否则初始化为空数组
      const initialItems = (field.extra as any)?.initialItems
      if (initialItems && Array.isArray(initialItems) && initialItems.length > 0) {
        formData[field.name] = [...initialItems]
      }
      else {
        formData[field.name] = []
      }
    }
    else {
      formData[field.name] = field.defaultValue ?? null
    }
  })

  // 初始化折叠状态：第一个展开，其余折叠
  Object.keys(collapseMap).forEach(key => delete collapseMap[key])
  dailyGridFields.value.forEach((field, index) => {
    const extra = field.options?.[0]?.extra
    const shiftId = extra?.shiftId as string || field.name
    collapseMap[shiftId] = index !== 0 // 第一个展开
  })
}

// 处理折叠状态变化：展开某个班次时自动折叠其他班次
function handleCollapseChange(shiftId: string, collapsed: boolean) {
  if (!collapsed) {
    // 展开当前班次时，折叠其他所有班次
    Object.keys(collapseMap).forEach((key) => {
      if (key !== shiftId) {
        collapseMap[key] = true
      }
    })
  }
  collapseMap[shiftId] = collapsed
}

// 验证表单
function validateForm() {
  if (!props.action?.fields)
    return true

  const missingFields = props.action.fields
    .filter((f) => {
      if (!f.required)
        return false
      const value = formData[f.name]
      if (f.type === 'daily-grid') {
        // daily-grid 类型：检查是否有配置
        return !value || Object.keys(value).length === 0
      }
      if (f.type === 'table-form') {
        // table-form 类型：检查数组是否有有效项
        if (!Array.isArray(value) || value.length === 0) {
          // 如果字段是必填的，且列表为空，则验证失败
          if (f.required) {
            return true
          }
          return false // 非必填字段允许为空
        }
        // table-form 类型不需要检查每个项目的必填字段（因为表单已经验证过了）
        return false
      }
      return !value
    })
    .map(f => f.label)

  if (missingFields.length > 0) {
    ElMessage.warning(`请填写：${missingFields.join('、')}`)
    return false
  }

  return true
}

// 确认
function handleConfirm() {
  if (!validateForm())
    return

  emit('confirm', { ...formData })
  emit('update:visible', false)
}

// 取消
function handleCancel() {
  emit('cancel')
  emit('update:visible', false)
}

// 创建表单输入控件
function createFormInput(field: any) {
  switch (field.type) {
    case 'multi-select':
      return h(MultiSelectField, {
        'modelValue': formData[field.name] || [],
        'onUpdate:modelValue': (value: any[]) => {
          formData[field.name] = value
        },
        'options': field.options || [],
        'label': field.label,
        'placeholder': field.placeholder || '搜索...',
      })

    case 'date': {
      // 检查是否支持多选（通过 extra.multiple 标记）
      const isMultiple = (field as any).extra?.multiple === true
      return h(ElDatePicker, {
        'modelValue': formData[field.name],
        'onUpdate:modelValue': (value: string | string[]) => {
          formData[field.name] = value
        },
        'type': isMultiple ? 'dates' : 'date', // 多选使用 'dates'，单选使用 'date'
        'placeholder': field.placeholder || '请选择日期',
        'format': 'YYYY-MM-DD',
        'valueFormat': 'YYYY-MM-DD',
        'style': { width: '100%' },
      })
    }

    case 'datetime':
      return h(ElDatePicker, {
        'modelValue': formData[field.name],
        'onUpdate:modelValue': (value: string) => {
          formData[field.name] = value
        },
        'type': 'datetime',
        'placeholder': field.placeholder || '请选择日期时间',
        'format': 'YYYY-MM-DD HH:mm:ss',
        'valueFormat': 'YYYY-MM-DD HH:mm:ss',
        'style': { width: '100%' },
      })

    case 'number':
      return h(ElInputNumber, {
        'modelValue': formData[field.name],
        'onUpdate:modelValue': (value: number | undefined) => {
          formData[field.name] = value
        },
        'placeholder': field.placeholder,
        'min': field.validation?.min,
        'max': field.validation?.max,
        'style': { width: '100%' },
      })

    case 'textarea': {
      // 检查是否启用Markdown支持
      const enableMarkdown = (field as any).extra?.markdown === true
      const rows = (field as any).extra?.rows ?? 3
      
      if (enableMarkdown) {
        // Markdown模式的textarea，可以使用更大的行数
        return h(ElInput, {
          'modelValue': formData[field.name],
          'onUpdate:modelValue': (value: string) => {
            formData[field.name] = value
          },
          'type': 'textarea',
          'placeholder': field.placeholder || '支持 Markdown 格式，多行粘贴即可',
          'rows': rows > 0 ? rows : 8,
          'style': { width: '100%', fontFamily: 'monospace' },
          'autosize': { minRows: rows > 0 ? rows : 4, maxRows: 15 },
        })
      } else {
        // 普通textarea
        return h(ElInput, {
          'modelValue': formData[field.name],
          'onUpdate:modelValue': (value: string) => {
            formData[field.name] = value
          },
          'type': 'textarea',
          'placeholder': field.placeholder,
          'rows': rows > 0 ? rows : 3,
          'style': { width: '100%' },
        })
      }
    }

    case 'select':
      return h(
        ElSelect,
        {
          'modelValue': formData[field.name],
          'onUpdate:modelValue': (value: string) => {
            formData[field.name] = value
          },
          'placeholder': field.placeholder || '请选择',
          'style': { width: '100%' },
        },
        () => (field.options || []).map((opt: any) =>
          h(ElOption, { value: opt.value, label: opt.label }),
        ),
      )

    case 'text':
    default:
      return h(ElInput, {
        'modelValue': formData[field.name],
        'onUpdate:modelValue': (value: string) => {
          formData[field.name] = value
        },
        'placeholder': field.placeholder,
        'style': { width: '100%' },
      })
  }
}

// 获取 daily-grid 字段的额外信息
function getDailyGridExtra(field: any) {
  return field.options?.[0]?.extra || {}
}

// 对话框宽度（根据字段类型动态调整）
const dialogWidth = computed(() => {
  const hasTableForm = props.action?.fields?.some(f => f.type === 'table-form')
  const hasDailyGrid = dailyGridFields.value.length > 0
  const hasMultiSelect = props.action?.fields?.some(f => f.type === 'multi-select')
  if (hasTableForm)
    return '1200px' // table-form 需要更宽，避免内容挤在一起
  if (hasDailyGrid)
    return '800px' // daily-grid 需要更宽
  return hasMultiSelect ? '700px' : '500px'
})

// 监听 action 变化初始化表单
watch(
  () => props.action,
  (newAction) => {
    if (newAction) {
      initFormData()
    }
  },
  { immediate: true },
)
</script>

<template>
  <ElDialog
    :model-value="visible"
    :title="action?.label"
    :width="dialogWidth"
    :before-close="handleCancel"
    :close-on-click-modal="false"
    class="action-form-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="action?.fields" class="action-form">
      <!-- 信息字段放在顶部 -->
      <div
        v-for="field in topFields"
        :key="field.name"
        class="form-item info-field"
      >
        <div class="period-info-banner">
          <el-icon class="info-icon">
            <Calendar />
          </el-icon>
          <span>{{ field.label }}</span>
        </div>
      </div>

      <!-- daily-grid 类型字段（按天人数配置） -->
      <div v-if="dailyGridFields.length > 0" class="daily-grid-container">
        <DailyGridField
          v-for="(field, index) in dailyGridFields"
          :key="field.name"
          v-model="formData[field.name]"
          :collapsed="collapseMap[getDailyGridExtra(field).shiftId || field.name]"
          :shift-name="getDailyGridExtra(field).shiftName || field.label"
          :shift-time="getDailyGridExtra(field).startTime && getDailyGridExtra(field).endTime
            ? `${getDailyGridExtra(field).startTime}-${getDailyGridExtra(field).endTime}`
            : undefined"
          :shift-color="getDailyGridExtra(field).shiftColor"
          :start-date="getDailyGridExtra(field).startDate"
          :end-date="getDailyGridExtra(field).endDate"
          :is-first="index === 0"
          @update:collapsed="handleCollapseChange(getDailyGridExtra(field).shiftId || field.name, $event)"
        />
      </div>

      <!-- table-form 类型字段（表格+表单组合） -->
      <div
        v-for="field in tableFormFields"
        :key="field.name"
        class="form-item"
      >
        <TableFormField
          v-model="formData[field.name]"
          :field="field"
        />
      </div>

      <!-- 普通字段 -->
      <div
        v-for="field in regularFields"
        :key="field.name"
        class="form-item"
      >
        <label class="form-label">
          {{ field.label }}
          <span v-if="field.required" class="required">*</span>
        </label>
        <component :is="createFormInput(field)" />
      </div>
    </div>

    <template #footer>
      <ElButton @click="handleCancel">
        取消
      </ElButton>
      <ElButton type="primary" @click="handleConfirm">
        确定
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
:deep(.action-form-dialog) {
  .el-dialog__body {
    padding: 16px 20px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }
}

.action-form {
  padding: 8px 0;

  .form-item {
    margin-bottom: 24px;

    &:last-child {
      margin-bottom: 0;
    }

    &.info-field {
      margin-bottom: 20px;
      margin-top: -8px;
    }
  }

  .form-label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-primary);
    margin-bottom: 10px;
    line-height: 1.5;

    .required {
      color: var(--el-color-danger);
      margin-left: 4px;
    }
  }

  .period-info-banner {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    background: linear-gradient(135deg, var(--el-color-primary-light-9) 0%, var(--el-color-primary-light-8) 100%);
    border-left: 4px solid var(--el-color-primary);
    border-radius: 6px;
    font-size: 14px;
    color: var(--el-color-primary);
    font-weight: 500;

    .info-icon {
      font-size: 18px;
      flex-shrink: 0;
    }

    span {
      flex: 1;
    }
  }

  .daily-grid-container {
    margin-bottom: 16px;
    max-height: 500px;
    overflow-y: auto;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-track {
      background: var(--el-fill-color-lighter);
      border-radius: 3px;
    }

    &::-webkit-scrollbar-thumb {
      background: var(--el-border-color);
      border-radius: 3px;

      &:hover {
        background: var(--el-border-color-dark);
      }
    }
  }

  .shift-fields-container {
    margin-bottom: 16px;
  }
}
</style>
