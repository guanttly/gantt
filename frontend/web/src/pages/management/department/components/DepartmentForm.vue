<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { CreateDepartmentRequest, DepartmentInfo, DepartmentTree, UpdateDepartmentRequest } from '@/api/department/model'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { createDepartment, getActiveDepartments, updateDepartment } from '@/api/department'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  department: DepartmentTree | null
  parent: DepartmentTree | null // 父部门（新增子部门时使用）
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)

// 部门列表（用于选择父部门）
const departmentList = ref<DepartmentInfo[]>([])

// 表单数据
const formData = reactive<CreateDepartmentRequest & UpdateDepartmentRequest>({
  orgId: props.orgId,
  code: '',
  name: '',
  parentId: undefined,
  description: '',
  managerId: undefined,
  sortOrder: 0,
  isActive: true,
})

// 表单验证规则
const rules: FormRules = {
  code: [
    { required: true, message: '请输入部门编码', trigger: 'blur' },
    { pattern: /^[A-Z0-9_-]+$/, message: '编码只能包含大写字母、数字、下划线和连字符', trigger: 'blur' },
  ],
  name: [
    { required: true, message: '请输入部门名称', trigger: 'blur' },
  ],
  sortOrder: [
    { type: 'number', message: '排序必须是数字', trigger: 'blur' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  if (props.mode === 'create') {
    return props.parent ? `添加子部门 (${props.parent.name})` : '新增顶级部门'
  }
  return '编辑部门'
})

// 是否可以编辑编码（创建时可编辑，编辑时不可编辑）
const isCodeEditable = computed(() => props.mode === 'create')

// 获取启用的部门列表
async function fetchDepartmentList() {
  try {
    const res = await getActiveDepartments(props.orgId)
    departmentList.value = res || []
  }
  catch (error: any) {
    ElMessage.error(`获取部门列表失败: ${error.message}`)
  }
}

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    fetchDepartmentList()

    if (props.mode === 'edit' && props.department) {
      // 编辑模式
      Object.assign(formData, {
        orgId: props.department.orgId,
        code: props.department.code,
        name: props.department.name,
        parentId: props.department.parentId || undefined,
        description: props.department.description || '',
        managerId: props.department.managerId || undefined,
        sortOrder: props.department.sortOrder || 0,
        isActive: props.department.isActive,
      })
    }
    else {
      // 创建模式
      resetForm()
      if (props.parent) {
        formData.parentId = props.parent.id
      }
    }
  }
})

// 重置表单
function resetForm() {
  Object.assign(formData, {
    orgId: props.orgId,
    code: '',
    name: '',
    parentId: undefined,
    description: '',
    managerId: undefined,
    sortOrder: 0,
    isActive: true,
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
      const createData: CreateDepartmentRequest = {
        orgId: formData.orgId,
        code: formData.code,
        name: formData.name,
        parentId: formData.parentId,
        description: formData.description,
        managerId: formData.managerId,
        sortOrder: formData.sortOrder,
      }
      await createDepartment(createData)
      ElMessage.success('创建成功')
    }
    else {
      const updateData: UpdateDepartmentRequest = {
        orgId: formData.orgId,
        name: formData.name,
        description: formData.description,
        managerId: formData.managerId,
        sortOrder: formData.sortOrder,
        isActive: formData.isActive,
      }
      await updateDepartment(props.department!.id, updateData)
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
      <el-form-item label="部门编码" prop="code">
        <el-input
          v-model="formData.code"
          :disabled="!isCodeEditable"
          placeholder="如: TECH, HR, ADMIN"
        />
        <div v-if="!isCodeEditable" class="text-xs text-gray-500 mt-1">
          部门编码创建后不可修改
        </div>
      </el-form-item>

      <el-form-item label="部门名称" prop="name">
        <el-input
          v-model="formData.name"
          placeholder="请输入部门名称"
        />
      </el-form-item>

      <el-form-item label="父部门" prop="parentId">
        <el-select
          v-model="formData.parentId"
          :disabled="mode === 'create' && !!parent"
          placeholder="选择父部门（不选则为顶级部门）"
          clearable
          filterable
          style="width: 100%"
        >
          <el-option
            v-for="dept in departmentList"
            :key="dept.id"
            :label="`${dept.name} (${dept.code})`"
            :value="dept.id"
            :disabled="mode === 'edit' && dept.id === department?.id"
          />
        </el-select>
        <div v-if="mode === 'create' && parent" class="text-xs text-gray-500 mt-1">
          当前将在 "{{ parent.name }}" 下创建子部门
        </div>
      </el-form-item>

      <el-form-item label="部门描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入部门描述"
        />
      </el-form-item>

      <el-form-item label="排序" prop="sortOrder">
        <el-input-number
          v-model="formData.sortOrder"
          :min="0"
          :step="1"
        />
        <div class="text-xs text-gray-500 mt-1">
          数字越小越靠前
        </div>
      </el-form-item>

      <el-form-item v-if="mode === 'edit'" label="状态" prop="isActive">
        <el-switch
          v-model="formData.isActive"
          active-text="启用"
          inactive-text="禁用"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">
        {{ mode === 'create' ? '创建' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
</style>
