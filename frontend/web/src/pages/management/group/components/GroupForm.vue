<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { GroupFormData } from '../type'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { createGroup, updateGroup } from '@/api/group'
import { typeOptions } from '../logic'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  group: Group.GroupInfo | null
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
const formData = reactive<GroupFormData>({
  orgId: props.orgId,
  code: '',
  name: '',
  type: 'team',
  parentId: '',
  description: '',
})

// 表单验证规则
const rules: FormRules = {
  code: [
    { required: true, message: '请输入分组编码', trigger: 'blur' },
  ],
  name: [
    { required: true, message: '请输入分组名称', trigger: 'blur' },
  ],
  type: [
    { required: true, message: '请选择分组类型', trigger: 'change' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '新增分组' : '编辑分组'
})

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    if (props.mode === 'edit' && props.group) {
      Object.assign(formData, {
        orgId: props.group.orgId,
        code: props.group.code,
        name: props.group.name,
        type: props.group.type,
        parentId: props.group.parentId || '',
        description: props.group.description || '',
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
    code: '',
    name: '',
    type: 'team',
    parentId: '',
    description: '',
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
      await createGroup(formData)
      ElMessage.success('创建成功')
    }
    else {
      await updateGroup(props.group!.id, formData)
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
    width="600px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="分组编码" prop="code">
        <el-input
          v-model="formData.code"
          :disabled="mode === 'edit'"
          placeholder="请输入分组编码"
        />
        <div v-if="mode === 'edit'" class="text-xs text-gray-500 mt-1">
          分组编码创建后不可修改
        </div>
      </el-form-item>
      <el-form-item label="分组名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入分组名称" />
      </el-form-item>
      <el-form-item label="分组类型" prop="type">
        <el-select v-model="formData.type" placeholder="请选择分组类型" style="width: 100%">
          <el-option
            v-for="item in typeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="父分组ID" prop="parentId">
        <el-input v-model="formData.parentId" placeholder="留空表示顶级分组" />
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
