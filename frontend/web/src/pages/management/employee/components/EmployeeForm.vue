<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { EmployeeFormData } from '../type'
import type { DepartmentInfo } from '@/api/department/model'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { getActiveDepartments } from '@/api/department'
import { createEmployee, updateEmployee } from '@/api/employee'
import { statusOptions } from '../logic'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  employee: Employee.EmployeeInfo | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)
let serNum = 18

// 部门列表
const departmentList = ref<DepartmentInfo[]>([])

// 表单数据
const formData = reactive<EmployeeFormData>({
  orgId: props.orgId,
  employeeId: '',
  name: '',
  email: '',
  phone: '',
  department: '',
  position: '',
  hireDate: '',
  status: 'active',
})

// 表单验证规则
const rules: FormRules = {
  employeeId: [
    { required: true, message: '请输入工号', trigger: 'blur' },
  ],
  name: [
    { required: true, message: '请输入姓名', trigger: 'blur' },
  ],
  email: [
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' },
  ],
  phone: [
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号码', trigger: 'blur' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '新增员工' : '编辑员工'
})

// 获取启用的部门列表
async function fetchDepartmentList() {
  try {
    const res = await getActiveDepartments(props.orgId)
    departmentList.value = res || []
  }
  catch (error: any) {
    console.error('获取部门列表失败:', error)
  }
}

// 监听 visible 变化，初始化表单
watch(() => props.visible, (val) => {
  if (val) {
    fetchDepartmentList() // 加载部门列表

    if (props.mode === 'edit' && props.employee) {
      Object.assign(formData, {
        orgId: props.employee.orgId,
        employeeId: props.employee.employeeId,
        name: props.employee.name,
        email: props.employee.email || '',
        phone: props.employee.phone || '',
        department: props.employee.departmentId || '',
        position: props.employee.position || '',
        hireDate: props.employee.hireDate || '',
        status: props.employee.status,
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
    employeeId: (serNum++).toString(),
    name: '',
    email: '',
    phone: '',
    department: '',
    position: '',
    hireDate: '',
    status: 'active',
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
      await createEmployee(formData)
      ElMessage.success('创建成功')
    }
    else {
      await updateEmployee(props.employee!.id, formData)
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
      <el-form-item label="工号" prop="employeeId">
        <el-input
          v-model="formData.employeeId"
          :disabled="mode === 'edit'"
          placeholder="请输入工号"
        />
      </el-form-item>
      <el-form-item label="姓名" prop="name">
        <el-input v-model="formData.name" placeholder="请输入姓名" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="formData.email" placeholder="请输入邮箱" />
      </el-form-item>
      <el-form-item label="电话" prop="phone">
        <el-input v-model="formData.phone" placeholder="请输入电话" />
      </el-form-item>
      <el-form-item label="部门" prop="department">
        <el-select
          v-model="formData.department"
          placeholder="请选择部门"
          clearable
          filterable
          style="width: 100%"
        >
          <el-option
            v-for="dept in departmentList"
            :key="dept.id"
            :label="`${dept.name} (${dept.code})`"
            :value="dept.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="职位" prop="position">
        <el-input v-model="formData.position" placeholder="请输入职位" />
      </el-form-item>
      <el-form-item label="入职日期" prop="hireDate">
        <el-date-picker
          v-model="formData.hireDate"
          type="date"
          placeholder="请选择入职日期"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item v-if="mode === 'edit'" label="状态" prop="status">
        <el-select v-model="formData.status" placeholder="请选择状态" style="width: 100%">
          <el-option
            v-for="item in statusOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
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
