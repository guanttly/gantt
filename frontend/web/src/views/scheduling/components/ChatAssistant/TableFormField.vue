<script setup lang="ts">
import type { WorkflowActionField } from '@/types/chat'

import { Delete, Edit } from '@element-plus/icons-vue'
import { ElButton, ElCol, ElDatePicker, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage, ElOption, ElRow, ElSelect, ElTable, ElTableColumn } from 'element-plus'
import { computed, h, reactive, ref, watch } from 'vue'

import MultiSelectField from './MultiSelectField.vue'

interface Props {
  modelValue?: any[]
  field: WorkflowActionField
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
}>()

const extra = computed(() => (props.field.extra || {}) as Record<string, any>)
const tableColumns = computed(() => extra.value.tableColumns || [])
const formFields = computed(() => extra.value.formFields || [] as any[])
const initialItems = computed(() => extra.value.initialItems || [] as any[])

const tableData = reactive<any[]>([])
const formData = reactive<Record<string, any>>({})
const editingIndex = ref<number | null>(null)

function initializeTableData(items: any[]) {
  items.forEach((item) => {
    if (!item.targetDatesDisplay) {
      let datesArray: string[] = []
      if (item.targetDates) {
        if (Array.isArray(item.targetDates))
          datesArray = item.targetDates
        else if (typeof item.targetDates === 'string')
          datesArray = item.targetDates.split(',').map((d: string) => d.trim()).filter((d: string) => d)
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
        item.targetDates = datesArray
      }
      else {
        item.targetDatesDisplay = '整个周期'
        if (!item.targetDates || !Array.isArray(item.targetDates))
          item.targetDates = []
      }
    }
  })
  return items
}

if (props.modelValue && Array.isArray(props.modelValue) && props.modelValue.length > 0)
  tableData.splice(0, 0, ...initializeTableData([...props.modelValue]))
else if (initialItems.value.length > 0)
  tableData.splice(0, 0, ...initializeTableData([...initialItems.value]))

function resetForm() {
  Object.keys(formData).forEach(key => delete formData[key])
  formFields.value.forEach((field: any) => {
    formData[field.name] = field.defaultValue ?? null
  })
  editingIndex.value = null
}

resetForm()

function saveItem() {
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

  const item: any = { ...formData }

  const staffField = formFields.value.find((f: any) => f.name === 'staffId')
  if (staffField && item.staffId) {
    const staffOption = staffField.options?.find((opt: any) => opt.value === item.staffId)
    item.staffName = staffOption?.label || item.staffId
  }

  const shiftField = formFields.value.find((f: any) => f.name === 'targetShiftId')
  if (shiftField && item.targetShiftId) {
    const shiftOption = shiftField.options?.find((opt: any) => opt.value === item.targetShiftId)
    item.targetShiftName = shiftOption?.label || item.targetShiftId || '任意班次'
  }
  else {
    item.targetShiftName = '任意班次'
  }

  let datesArray: string[] = []
  if (item.targetDates) {
    if (Array.isArray(item.targetDates))
      datesArray = item.targetDates
    else if (typeof item.targetDates === 'string')
      datesArray = item.targetDates.split(',').map((d: string) => d.trim()).filter((d: string) => d)
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
    item.targetDates = datesArray
  }
  else {
    item.targetDatesDisplay = '整个周期'
    if (!item.targetDates || !Array.isArray(item.targetDates))
      item.targetDates = []
  }

  const requestTypeMap: Record<string, string> = {
    avoid: '回避',
    must: '必须',
    prefer: '偏好',
  }
  item.requestTypeDisplay = requestTypeMap[item.requestType] || item.requestType

  if (editingIndex.value !== null)
    tableData[editingIndex.value] = item
  else
    tableData.push(item)

  resetForm()
  emitChange()
}

function editItem(index: number) {
  const item = tableData[index]
  Object.keys(formData).forEach(key => delete formData[key])
  formFields.value.forEach((field: any) => {
    let value = item[field.name] ?? field.defaultValue ?? null
    if (field.type === 'date' && field.extra?.multiple === true) {
      if (typeof value === 'string' && value.includes(','))
        value = value.split(',').map((d: string) => d.trim()).filter((d: string) => d)
      formData[field.name] = Array.isArray(value) ? value : (value ? [value] : [])
    }
    else {
      formData[field.name] = value
    }
  })
  editingIndex.value = index
}

function deleteItem(index: number) {
  tableData.splice(index, 1)
  if (editingIndex.value === index)
    resetForm()
  else if (editingIndex.value !== null && editingIndex.value > index)
    editingIndex.value--
  emitChange()
}

function cancelEdit() {
  resetForm()
}

function emitChange() {
  emit('update:modelValue', [...tableData])
}

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
          if (isMultiple)
            formData[fieldName] = Array.isArray(value) ? value : (value ? [value] : [])
          else
            formData[fieldName] = value
        },
        'type': isMultiple ? 'dates' : 'date',
        'placeholder': field.placeholder || '请选择日期',
        'format': 'YYYY-MM-DD',
        'valueFormat': 'YYYY-MM-DD',
        'disabledDate': (date: Date) => {
          if (!startDate || !endDate)
            return false
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

function formatTargetDates(dates: string | string[] | undefined): string {
  if (!dates)
    return '整个周期'
  if (typeof dates === 'string') {
    if (dates.includes('月') && dates.includes('日'))
      return dates
    if (dates.includes('-')) {
      const parts = dates.split('-')
      if (parts.length >= 3)
        return `${Number.parseInt(parts[1], 10)}月${Number.parseInt(parts[2], 10)}日`
    }
    return dates
  }
  if (Array.isArray(dates)) {
    if (dates.length === 0)
      return '整个周期'
    return dates.map((date: string) => {
      if (date && date.includes('-')) {
        const parts = date.split('-')
        if (parts.length >= 3)
          return `${Number.parseInt(parts[1], 10)}月${Number.parseInt(parts[2], 10)}日`
      }
      return date
    }).join('、')
  }
  return '整个周期'
}

watch(() => props.modelValue, (newValue) => {
  if (newValue && Array.isArray(newValue) && newValue.length > 0)
    tableData.splice(0, tableData.length, ...newValue)
}, { deep: true })
</script>

<template>
  <div class="table-form-field">
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
      <ElTable :data="tableData" border max-height="180" stripe style="width: 100%">
        <ElTableColumn
          v-for="col in tableColumns"
          :key="col.prop"
          :label="col.label"
          :min-width="col.minWidth"
          :prop="col.prop"
          :width="col.width"
        >
          <template #default="{ row }">
            <span v-if="col.prop === 'requestType'">{{ row.requestTypeDisplay || row.requestType }}</span>
            <span v-else-if="col.prop === 'targetDates'">{{ row.targetDatesDisplay || formatTargetDates(row.targetDates) }}</span>
            <span v-else>{{ row[col.prop] }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn align="center" fixed="right" label="操作" width="165">
          <template #default="{ $index }">
            <ElButton :icon="Edit" size="small" text type="primary" @click="editItem($index)">
              编辑
            </ElButton>
            <ElButton :icon="Delete" size="small" text type="danger" @click="deleteItem($index)">
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <div class="form-section">
      <div class="form-header">
        <span class="form-title">{{ editingIndex !== null ? '编辑需求' : '新增需求' }}</span>
        <ElButton v-if="editingIndex !== null" size="small" @click="cancelEdit">
          取消
        </ElButton>
      </div>
      <ElForm :model="formData" label-position="left" label-width="120px">
        <ElRow :gutter="16">
          <ElCol v-for="formField in formFields" :key="formField.name" :span="formField.span || 24">
            <ElFormItem :label="formField.label" :required="formField.required">
              <component :is="createFormInput(formField)" :key="`${formField.name}-${editingIndex}`" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem>
          <div style="display: flex; justify-content: flex-end; gap: 8px; width: 100%;">
            <ElButton v-if="editingIndex !== null" @click="cancelEdit">
              取消
            </ElButton>
            <ElButton type="primary" @click="saveItem">
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
          .required { color: var(--el-color-danger); margin-left: 4px; }
        }

        .table-count {
          font-size: 14px;
          color: var(--el-text-color-regular);
          .count-number { color: var(--el-color-primary); font-weight: 600; margin: 0 2px; }
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
