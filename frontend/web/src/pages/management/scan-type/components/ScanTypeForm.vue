<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { createScanType, updateScanType } from '@/api/scan-type'

interface ScanTypeFormData {
  orgId: string
  name: string
  code: string
  description: string
  sortOrder: number
}

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  scanType: ScanType.Info | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)

// 表单数据
const formData = reactive<ScanTypeFormData>({
  orgId: props.orgId,
  name: '',
  code: '',
  description: '',
  sortOrder: 0,
})

// 表单验证规则
const rules: FormRules = {
  name: [
    { required: true, message: '请输入检查类型名称', trigger: 'blur' },
  ],
  code: [
    { required: true, message: '请输入检查类型编码', trigger: 'blur' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '新增检查类型' : '编辑检查类型'
})

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    if (props.mode === 'edit' && props.scanType) {
      Object.assign(formData, {
        orgId: props.orgId,
        name: props.scanType.name,
        code: props.scanType.code,
        description: props.scanType.description || '',
        sortOrder: props.scanType.sortOrder || 0,
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
    description: '',
    sortOrder: 0,
  })
  formRef.value?.clearValidate()
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value)
    return

  try {
    await formRef.value.validate()
    loading.value = true

    if (props.mode === 'create') {
      await createScanType(formData)
      ElMessage.success('创建成功')
    }
    else {
      await updateScanType(props.scanType!.id, formData)
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
          placeholder="请输入检查类型编码，如 plain/enhanced"
        />
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入检查类型名称，如 平扫/增强" />
      </el-form-item>
      <el-form-item label="排序" prop="sortOrder">
        <el-input-number v-model="formData.sortOrder" :min="0" :max="999" />
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
