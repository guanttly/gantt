<script setup lang="ts">
import type { WorkflowActionField } from '@/types/chat'
import { Delete, Edit, Plus } from '@element-plus/icons-vue'
import { ElButton, ElCol, ElDatePicker, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage, ElOption, ElRow, ElSelect, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { computed, h, reactive, ref, watch } from 'vue'
import MultiSelectField from './MultiSelectField.vue'

interface Props {
  modelValue: any[]
  field: WorkflowActionField
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
}>()

// 从 extra 中获取配置
const extra = computed(() => (props.field.extra || {}) as Record<string, any>)
const tableColumns = computed(() => extra.value.tableColumns || [])
const formFields = computed(() => extra.value.formFields || [] as any[])
const initialItems = computed(() => extra.value.initialItems || [] as any[])

// 表格数据
const tableData = reactive<any[]>([])

// 表单数据（用于新增/编辑）
const formData = reactive<Record<string, any>>({})
const editingIndex = ref<number | null>(null) // 正在编辑的行索引，null 表示新增

// 初始化表格数据
function initializeTableData(items: any[]) {
  items.forEach((item) => {
    // 如果还没有格式化显示，则格式化
    if (!item.targetDatesDisplay) {
      // 处理 targetDates：可能是数组，也可能是字符串（逗号分隔）
      let datesArray: string[] = []
      if (item.targetDates) {
        if (Array.isArray(item.targetDates)) {
          datesArray = item.targetDates
        }
        else if (typeof item.targetDates === 'string') {
          // 如果是字符串，尝试按逗号分割
          datesArray = item.targetDates.split(',').map((d: string) => d.trim()).filter((d: string) => d)
        }
      }

      if (datesArray.length > 0) {
        const formattedDates = datesArray.map((date: string) => {
          if (date && date.includes('-')) {
            const parts = date.split('-')
            if (parts.length >= 3) {
              const month = Number.parseInt(parts[1], 10)
              const day = Number.parseInt(parts[2], 10)
              return `${month}月${day}日`
            }
          }
          return date
        })
        item.targetDatesDisplay = formattedDates.join('、')
        // 确保 targetDates 保存为数组格式（用于后续编辑）
        item.targetDates = datesArray
      }
      else {
        item.targetDatesDisplay = '整个周期'
        // 如果没有日期，确保 targetDates 是空数组
        if (!item.targetDates || !Array.isArray(item.targetDates)) {
          item.targetDates = []
        }
      }
    }
  })
  return items
}

if (props.modelValue && Array.isArray(props.modelValue) && props.modelValue.length > 0) {
  tableData.splice(0, 0, ...initializeTableData([...props.modelValue]))
}
else if (initialItems.value.length > 0) {
  tableData.splice(0, 0, ...initializeTableData([...initialItems.value]))
}

// 重置表单
function resetForm() {
  Object.keys(formData).forEach(key => delete formData[key])
  formFields.value.forEach((field: any) => {
    formData[field.name] = field.defaultValue ?? null
  })
  editingIndex.value = null
}

// 初始化表单
resetForm()

// 添加/更新需求
function saveItem() {
  // 验证必填字段
  const requiredFields = formFields.value.filter((f: any) => f.required)
  const missingFields = requiredFields.filter((f: any) => {
    const value = formData[f.name]
    return !value || (Array.isArray(value) && value.length === 0)
  })

  if (missingFields.length > 0) {
    const fieldNames = missingFields.map((f: any) => f.label).join('、')
    ElMessage.warning(`请填写必填字段：${fieldNames}`)
    return
  }

  // 构建需求项（需要包含表格显示所需的字段）
  const item: any = { ...formData }

  // 查找人员名称
  const staffField = formFields.value.find((f: any) => f.name === 'staffId')
  if (staffField && item.staffId) {
    const staffOption = staffField.options?.find((opt: any) => opt.value === item.staffId)
    item.staffName = staffOption?.label || item.staffId
  }

  // 查找班次名称
  const shiftField = formFields.value.find((f: any) => f.name === 'targetShiftId')
  if (shiftField && item.targetShiftId) {
    const shiftOption = shiftField.options?.find((opt: any) => opt.value === item.targetShiftId)
    item.targetShiftName = shiftOption?.label || item.targetShiftId || '任意班次'
  }
  else {
    item.targetShiftName = '任意班次'
  }

  // 格式化日期显示（只显示月日，不显示年份）
  // 处理 targetDates：可能是数组，也可能是字符串（逗号分隔）
  let datesArray: string[] = []
  if (item.targetDates) {
    if (Array.isArray(item.targetDates)) {
      datesArray = item.targetDates
    }
    else if (typeof item.targetDates === 'string') {
      // 如果是字符串，尝试按逗号分割
      datesArray = item.targetDates.split(',').map((d: string) => d.trim()).filter((d: string) => d)
    }
  }

  if (datesArray.length > 0) {
    const formattedDates = datesArray.map((date: string) => {
      // 将 "2025-12-22" 格式转换为 "12月22日"
      if (date && date.includes('-')) {
        const parts = date.split('-')
        if (parts.length >= 3) {
          const month = Number.parseInt(parts[1], 10)
          const day = Number.parseInt(parts[2], 10)
          return `${month}月${day}日`
        }
      }
      return date
    })
    item.targetDatesDisplay = formattedDates.join('、')
    // 确保 targetDates 保存为数组格式（用于后续编辑）
    item.targetDates = datesArray
  }
  else {
    item.targetDatesDisplay = '整个周期'
    // 如果没有日期，确保 targetDates 是空数组
    if (!item.targetDates || !Array.isArray(item.targetDates)) {
      item.targetDates = []
    }
  }

  // 格式化请求类型显示
  const requestTypeMap: Record<string, string> = {
    prefer: '偏好',
    must: '必须',
    avoid: '回避',
  }
  item.requestTypeDisplay = requestTypeMap[item.requestType] || item.requestType

  if (editingIndex.value !== null) {
    // 更新现有项
    tableData[editingIndex.value] = item
  }
  else {
    // 添加新项
    tableData.push(item)
  }

  resetForm()
  emitChange()
}

// 编辑需求
function editItem(index: number) {
  const item = tableData[index]
  // 复制数据到表单
  Object.keys(formData).forEach(key => delete formData[key])
  formFields.value.forEach((field: any) => {
    let value = item[field.name] ?? field.defaultValue ?? null
    // 对于多选日期字段，确保是数组格式
    if (field.type === 'date' && field.extra?.multiple === true) {
      // 如果 value 是字符串，尝试解析（可能是逗号分隔的日期字符串）
      if (typeof value === 'string' && value.includes(',')) {
        value = value.split(',').map((d: string) => d.trim()).filter((d: string) => d)
      }
      // 确保是数组格式
      formData[field.name] = Array.isArray(value) ? value : (value ? [value] : [])
    }
    else {
      formData[field.name] = value
    }
  })
  editingIndex.value = index
}

// 删除需求
function deleteItem(index: number) {
  tableData.splice(index, 1)
  if (editingIndex.value === index) {
    resetForm()
  }
  else if (editingIndex.value !== null && editingIndex.value > index) {
    editingIndex.value--
  }
  emitChange()
}

// 取消编辑
function cancelEdit() {
  resetForm()
}

// 发送更新事件
function emitChange() {
  emit('update:modelValue', [...tableData])
}

// 创建表单输入控件
function createFormInput(field: any) {
  const fieldName = field.name
  const fieldValue = formData[fieldName]

  switch (field.type) {
    case 'select':
      return h(
        ElSelect,
        {
          'modelValue': fieldValue,
          'onUpdate:modelValue': (value: string) => {
            formData[fieldName] = value
          },
          'placeholder': field.placeholder || '请选择',
          'filterable': true,
          'style': { width: '100%' },
        },
        () => (field.options || []).map((opt: any) =>
          h(ElOption, { value: opt.value, label: opt.label }),
        ),
      )

    case 'date': {
      const isMultiple = field.extra?.multiple === true
      const startDate = field.extra?.startDate as string | undefined
      const endDate = field.extra?.endDate as string | undefined
      return h(ElDatePicker, {
        'modelValue': fieldValue,
        'onUpdate:modelValue': (value: string | string[]) => {
          // 确保多选时始终是数组格式
          if (isMultiple) {
            formData[fieldName] = Array.isArray(value) ? value : (value ? [value] : [])
          }
          else {
            formData[fieldName] = value
          }
        },
        'type': isMultiple ? 'dates' : 'date',
        'placeholder': field.placeholder || '请选择日期',
        'format': 'YYYY-MM-DD',
        'valueFormat': 'YYYY-MM-DD',
        'startPlaceholder': isMultiple ? '开始日期' : undefined,
        'endPlaceholder': isMultiple ? '结束日期' : undefined,
        'disabledDate': (date: Date) => {
          if (!startDate || !endDate) {
            return false
          }
          const dateStr = date.toISOString().split('T')[0]
          return dateStr < startDate || dateStr > endDate
        },
        'style': { width: '100%' },
      })
    }

    case 'number':
      return h(ElInputNumber, {
        'modelValue': fieldValue,
        'onUpdate:modelValue': (value: number | undefined) => {
          formData[fieldName] = value
        },
        'placeholder': field.placeholder,
        'min': field.validation?.min,
        'max': field.validation?.max,
        'style': { width: '100%' },
      })

    case 'textarea':
      return h(ElInput, {
        'modelValue': fieldValue,
        'onUpdate:modelValue': (value: string) => {
          formData[fieldName] = value
        },
        'type': 'textarea',
        'placeholder': field.placeholder,
        'rows': 3,
        'style': { width: '100%' },
      })

    case 'multi-select':
      return h(MultiSelectField, {
        'modelValue': fieldValue || [],
        'onUpdate:modelValue': (value: any[]) => {
          formData[fieldName] = value
        },
        'options': field.options || [],
        'label': field.label,
        'placeholder': field.placeholder || '搜索...',
      })

    case 'text':
    default:
      return h(ElInput, {
        'modelValue': fieldValue,
        'onUpdate:modelValue': (value: string) => {
          formData[fieldName] = value
        },
        'placeholder': field.placeholder,
        'style': { width: '100%' },
      })
  }
}

// 格式化目标日期显示（只显示月日，不显示年份）
function formatTargetDates(dates: string | string[] | undefined): string {
  if (!dates) {
    return '整个周期'
  }
  if (typeof dates === 'string') {
    // 如果已经是格式化后的字符串（包含"月"和"日"），直接返回
    if (dates.includes('月') && dates.includes('日')) {
      return dates
    }
    // 如果是单个日期字符串，格式化它
    if (dates.includes('-')) {
      const parts = dates.split('-')
      if (parts.length >= 3) {
        const month = Number.parseInt(parts[1], 10)
        const day = Number.parseInt(parts[2], 10)
        return `${month}月${day}日`
      }
    }
    return dates
  }
  if (Array.isArray(dates)) {
    if (dates.length === 0) {
      return '整个周期'
    }
    const formattedDates = dates.map((date: string) => {
      // 将 "2025-12-22" 格式转换为 "12月22日"
      if (date && date.includes('-')) {
        const parts = date.split('-')
        if (parts.length >= 3) {
          const month = Number.parseInt(parts[1], 10)
          const day = Number.parseInt(parts[2], 10)
          return `${month}月${day}日`
        }
      }
      return date
    })
    return formattedDates.join('、')
  }
  return '整个周期'
}

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (newValue && Array.isArray(newValue) && newValue.length > 0) {
    tableData.splice(0, tableData.length, ...newValue)
  }
}, { deep: true })
</script>

<template>
  <div class="table-form-field">
    <!-- 上半部分：表格 -->
    <div class="table-section">
      <div class="table-header">
        <div class="table-header-left">
          <label class="form-label">
            {{ field.label }}
            <span v-if="field.required" class="required">*</span>
          </label>
          <span class="table-count">【共计：<span class="count-number">{{ tableData.length }}</span>条】</span>
        </div>
      </div>
      <ElTable
        :data="tableData"
        border
        stripe
        style="width: 100%"
        max-height="180"
      >
        <ElTableColumn
          v-for="col in tableColumns"
          :key="col.prop"
          :prop="col.prop"
          :label="col.label"
          :width="col.width"
          :min-width="col.minWidth"
        >
          <template #default="{ row }">
            <span v-if="col.prop === 'requestType'">{{ row.requestTypeDisplay || row.requestType }}</span>
            <span v-else-if="col.prop === 'targetDates'">
              {{ row.targetDatesDisplay || formatTargetDates(row.targetDates) }}
            </span>
            <span v-else>{{ row[col.prop] }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="165" fixed="right" align="center">
          <template #default="{ $index }">
            <ElButton
              type="primary"
              :icon="Edit"
              size="small"
              text
              @click="editItem($index)"
            >
              编辑
            </ElButton>
            <ElButton
              type="danger"
              :icon="Delete"
              size="small"
              text
              @click="deleteItem($index)"
            >
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <!-- 下半部分：表单 -->
    <div class="form-section">
      <div class="form-header">
        <span class="form-title">{{ editingIndex !== null ? '编辑需求' : '新增需求' }}</span>
        <ElButton
          v-if="editingIndex !== null"
          size="small"
          @click="cancelEdit"
        >
          取消
        </ElButton>
      </div>
      <ElForm :model="formData" label-width="120px" label-position="left">
        <ElRow :gutter="16">
          <ElCol
            v-for="formField in formFields"
            :key="formField.name"
            :span="formField.span || 24"
          >
            <ElFormItem
              :label="formField.label"
              :required="formField.required"
            >
              <component
                :is="createFormInput(formField)"
                :key="`${formField.name}-${editingIndex}`"
              />
              <div
                v-if="formField.placeholder && formField.type === 'number'"
                style="font-size: 12px; color: var(--el-text-color-placeholder); margin-top: 4px;"
              >
                {{ formField.placeholder }}
              </div>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem>
          <div style="display: flex; justify-content: flex-end; gap: 8px; width: 100%;">
            <ElButton
              v-if="editingIndex !== null"
              @click="cancelEdit"
            >
              取消
            </ElButton>
            <ElButton
              type="primary"
              @click="saveItem"
            >
              {{ editingIndex !== null ? '更新' : '添加' }}
            </ElButton>
          </div>
        </ElFormItem>
      </ElForm>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.table-form-field {
  display: flex;
  flex-direction: column;
  gap: 20px;

  .table-section {
    .table-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      .table-header-left {
        display: flex;
        align-items: center;
        gap: 12px;

        .form-label {
          font-weight: 500;
          color: var(--el-text-color-primary);

          .required {
            color: var(--el-color-danger);
            margin-left: 4px;
          }
        }

        .table-count {
          font-size: 14px;
          color: var(--el-text-color-regular);

          .count-number {
            color: var(--el-color-primary);
            font-weight: 600;
            margin: 0 2px;
          }
        }
      }
    }
  }

  .form-section {
    border-top: 1px solid var(--el-border-color);
    padding-top: 20px;

    .form-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      .form-title {
        font-weight: 500;
        font-size: 16px;
        color: var(--el-text-color-primary);
      }
    }
  }
}
</style>
