<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { TimePeriodFormData } from '../type'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { createTimePeriod, updateTimePeriod } from '@/api/time-period'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  timePeriod: TimePeriod.Info | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)

// 快速时间模板
const quickTimeTemplates = [
  { label: '上午', startTime: '08:00', endTime: '12:00', isCrossDay: false },
  { label: '下午', startTime: '13:30', endTime: '17:30', isCrossDay: false },
  { label: '晚上', startTime: '18:00', endTime: '22:00', isCrossDay: false },
  { label: '凌晨', startTime: '00:00', endTime: '06:00', isCrossDay: false },
  { label: '夜班(跨天)', startTime: '20:00', endTime: '14:00', isCrossDay: true },
]

// 表单数据
const formData = reactive<TimePeriodFormData>({
  orgId: props.orgId,
  name: '',
  code: '',
  startTime: '',
  endTime: '',
  isCrossDay: false,
  description: '',
})

// 表单验证规则
const rules: FormRules = {
  name: [
    { required: true, message: '请输入时间段名称', trigger: 'blur' },
  ],
  code: [
    { required: true, message: '请输入时间段编码', trigger: 'blur' },
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
  return props.mode === 'create' ? '新增时间段' : '编辑时间段'
})

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    if (props.mode === 'edit' && props.timePeriod) {
      Object.assign(formData, {
        orgId: props.orgId,
        name: props.timePeriod.name,
        code: props.timePeriod.code,
        startTime: props.timePeriod.startTime,
        endTime: props.timePeriod.endTime,
        isCrossDay: props.timePeriod.isCrossDay || false,
        description: props.timePeriod.description || '',
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
    startTime: '',
    endTime: '',
    isCrossDay: false,
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
  formData.isCrossDay = template.isCrossDay
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
      await createTimePeriod(formData)
      ElMessage.success('创建成功')
    }
    else {
      await updateTimePeriod(props.timePeriod!.id, formData)
      ElMessage.success('更新成功')
    }

    emit('success')
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
    width="500px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="编码" prop="code">
        <el-input
          v-model="formData.code"
          :disabled="mode === 'edit'"
          placeholder="请输入时间段编码"
        />
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入时间段名称" />
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
      <el-form-item label="跨天">
        <div class="cross-day-control">
          <el-switch
            v-model="formData.isCrossDay"
            active-text="是"
            inactive-text="否"
          />
          <span class="cross-day-tip">
            开启表示开始时间为前一天（如：前日20:00 到 当日14:00）
          </span>
        </div>
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

<style lang="scss" scoped>
.cross-day-control {
  display: flex;
  align-items: flex-start;
  flex-direction: column;
  gap: 8px;
}

.cross-day-tip {
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
</style>
