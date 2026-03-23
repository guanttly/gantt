<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { ShiftFormData } from '../type'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { createShift, updateShift } from '@/api/shift'
import { calculateDuration, colorPresets, typeOptions } from '../logic'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  shift: Shift.ShiftInfo | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success', result?: { shiftId: string, shiftName: string, mode: 'create' | 'edit' }): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)

// 快速时间模板
const quickTimeTemplates = [
  { label: '上午岗', startTime: '08:00', endTime: '12:00', color: '#67C23A' },
  { label: '下午岗', startTime: '13:30', endTime: '17:30', color: '#409EFF' },
  { label: '晚班岗', startTime: '20:00', endTime: '24:00', color: '#E6A23C' },
  { label: '夜班岗', startTime: '00:00', endTime: '08:00', color: '#909399' },
]

// 表单数据
const formData = reactive<ShiftFormData>({
  orgId: props.orgId,
  name: '',
  code: '',
  type: 'regular',
  startTime: '',
  endTime: '',
  duration: 0,
  schedulingPriority: 0, // 排班优先级默认为0
  color: colorPresets[0],
  description: '',
})

// 表单验证规则
const rules: FormRules = {
  name: [
    { required: true, message: '请输入班次名称', trigger: 'blur' },
  ],
  code: [
    { required: true, message: '请输入班次编码', trigger: 'blur' },
  ],
  type: [
    { required: true, message: '请选择班次类型', trigger: 'change' },
  ],
  startTime: [
    { required: true, message: '请选择开始时间', trigger: 'change' },
  ],
  endTime: [
    { required: true, message: '请选择结束时间', trigger: 'change' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '新增班次' : '编辑班次'
})

// 监听时间变化自动计算时长
watch([() => formData.startTime, () => formData.endTime], () => {
  if (formData.startTime && formData.endTime) {
    formData.duration = calculateDuration(formData.startTime, formData.endTime)
  }
})

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    if (props.mode === 'edit' && props.shift) {
      Object.assign(formData, {
        orgId: props.orgId, // 使用 props.orgId 而不是 props.shift.orgId
        name: props.shift.name,
        code: props.shift.code,
        type: props.shift.type,
        startTime: props.shift.startTime,
        endTime: props.shift.endTime,
        duration: props.shift.duration,
        schedulingPriority: props.shift.schedulingPriority || 0,
        color: props.shift.color || colorPresets[0],
        description: props.shift.description || '',
      })
    }
    else {
      resetForm()
    }
  }
})

// 重置表单
function resetForm() {
  Object.assign(formData, {
    orgId: props.orgId,
    name: '',
    code: '',
    type: 'regular',
    startTime: '',
    endTime: '',
    duration: 0,
    schedulingPriority: 0,
    color: colorPresets[0],
    description: '',
  })
  formRef.value?.clearValidate()
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 应用快速时间模板
function applyQuickTime(template: typeof quickTimeTemplates[0]) {
  formData.startTime = template.startTime
  formData.endTime = template.endTime
  formData.color = template.color
  // 如果班次名称为空，自动填充模板名称
  if (!formData.name) {
    formData.name = template.label
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value)
    return

  try {
    await formRef.value.validate()
    loading.value = true

    if (props.mode === 'create') {
      const result = await createShift(formData)
      ElMessage.success('创建成功')
      // 返回新创建的班次 ID 和名称，供父组件自动打开人数配置对话框
      emit('success', { shiftId: result.id, shiftName: formData.name, mode: 'create' })
    }
    else {
      await updateShift(props.shift!.id, formData)
      ElMessage.success('更新成功')
      emit('success', { shiftId: props.shift!.id, shiftName: formData.name, mode: 'edit' })
    }

    handleClose()
  }
  catch {
    // 表单验证失败(error === false)或请求错误(由拦截器处理)
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="600px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="班次编码" prop="code">
        <el-input
          v-model="formData.code"
          :disabled="mode === 'edit'"
          placeholder="请输入班次编码"
        />
      </el-form-item>
      <el-form-item label="班次名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入班次名称" />
      </el-form-item>
      <el-form-item label="班次类型" prop="type">
        <el-select v-model="formData.type" placeholder="请选择班次类型" style="width: 100%">
          <el-option
            v-for="item in typeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <!-- 快速时间选择 -->
      <el-form-item label="快速选择">
        <el-space wrap>
          <el-button
            v-for="template in quickTimeTemplates"
            :key="template.label"
            size="small"
            plain
            @click="applyQuickTime(template)"
          >
            <span :style="{ color: template.color, fontWeight: '500' }">●</span>
            {{ template.label }}
            <span style="color: #909399; font-size: 12px; margin-left: 4px">
              {{ template.startTime }}-{{ template.endTime }}
            </span>
          </el-button>
        </el-space>
      </el-form-item>

      <el-form-item label="开始时间" prop="startTime">
        <el-time-picker
          v-model="formData.startTime"
          format="HH:mm"
          value-format="HH:mm"
          placeholder="请选择开始时间"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="结束时间" prop="endTime">
        <el-time-picker
          v-model="formData.endTime"
          format="HH:mm"
          value-format="HH:mm"
          placeholder="请选择结束时间"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="时长">
        <el-input :model-value="`${formData.duration} 分钟`" disabled />
      </el-form-item>
      <el-form-item label="排班优先级" prop="schedulingPriority">
        <el-input-number
          v-model="formData.schedulingPriority"
          :min="0"
          :max="999"
          :step="1"
          style="width: 100%"
          placeholder="请输入排班优先级"
        />
        <span style="color: #909399; font-size: 12px; margin-left: 8px">
          数值越小优先级越高，用于排班时的排序
        </span>
      </el-form-item>
      <el-form-item label="颜色" prop="color">
        <el-color-picker v-model="formData.color" :predefine="colorPresets" />
      </el-form-item>
      <el-form-item label="描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入描述"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>
