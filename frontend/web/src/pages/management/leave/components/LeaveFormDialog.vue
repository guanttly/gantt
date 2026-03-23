<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { getSimpleEmployeeList } from '@/api/employee'
import { createLeave, getLeaveDetail, updateLeave } from '@/api/leave'
import { leaveTypeOptions } from '../logic'

interface Props {
  visible: boolean
  orgId: string
  leaveId?: string
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 表单数据
const formData = reactive({
  orgId: props.orgId,
  employeeId: '',
  type: 'annual' as Leave.LeaveType,
  startDate: '',
  endDate: '',
  startTime: '',
  endTime: '',
  reason: '',
})

// 表单引用
const formRef = ref()

// 提交中状态
const submitting = ref(false)
const loading = ref(false)

// 员工列表
const employeeList = ref<Employee.SimpleEmployeeInfo[]>([])
const employeeLoading = ref(false)

// 是否编辑模式
const isEdit = computed(() => !!props.leaveId)

// 对话框标题
const dialogTitle = computed(() => isEdit.value ? '编辑请假记录' : '新增请假记录')

// 表单验证规则
const rules = {
  employeeId: [{ required: true, message: '请选择员工', trigger: 'change' }],
  type: [{ required: true, message: '请选择假期类型', trigger: 'change' }],
  startDate: [{ required: true, message: '请选择开始日期', trigger: 'change' }],
  endDate: [{ required: true, message: '请选择结束日期', trigger: 'change' }],
}

// 监听对话框显示
watch(() => props.visible, (visible) => {
  if (visible) {
    resetForm()
    if (props.leaveId) {
      loadLeaveDetail()
    }
    else {
      loadEmployees()
    }
  }
})

// 重置表单
function resetForm() {
  formData.orgId = props.orgId
  formData.employeeId = ''
  formData.type = 'annual'
  formData.startDate = ''
  formData.endDate = ''
  formData.startTime = ''
  formData.endTime = ''
  formData.reason = ''
  formRef.value?.clearValidate()
}

// 加载请假详情
async function loadLeaveDetail() {
  if (!props.leaveId)
    return

  loading.value = true
  try {
    const data = await getLeaveDetail(props.leaveId, props.orgId)
    formData.employeeId = data.employeeId
    formData.type = data.type
    formData.startDate = data.startDate
    formData.endDate = data.endDate
    formData.startTime = data.startTime || ''
    formData.endTime = data.endTime || ''
    formData.reason = data.reason || ''
  }
  catch {
    ElMessage.error('加载请假详情失败')
    handleClose()
  }
  finally {
    loading.value = false
  }
}

// 加载员工列表（远程搜索）
async function loadEmployees(keyword = '') {
  employeeLoading.value = true
  try {
    const res = await getSimpleEmployeeList({
      orgId: props.orgId,
      keyword,
      page: 1,
      size: 50,
    })
    employeeList.value = res.items || []
  }
  catch {
    // 静默失败
  }
  finally {
    employeeLoading.value = false
  }
}

// 远程搜索员工
function handleSearchEmployee(query: string) {
  if (query) {
    loadEmployees(query)
  }
  else {
    employeeList.value = []
  }
}

// 员工选择器获得焦点时加载数据
function handleEmployeeFocus() {
  if (employeeList.value.length === 0) {
    loadEmployees()
  }
}

// 提交表单
async function handleSubmit() {
  await formRef.value.validate()

  submitting.value = true
  try {
    const requestData: Leave.CreateRequest | Leave.UpdateRequest = {
      orgId: formData.orgId,
      ...(isEdit.value ? {} : { employeeId: formData.employeeId }),
      ...(isEdit.value ? {} : { type: formData.type }),
      startDate: formData.startDate,
      endDate: formData.endDate,
      startTime: formData.startTime || undefined,
      endTime: formData.endTime || undefined,
      reason: formData.reason || undefined,
    }

    if (isEdit.value) {
      await updateLeave(props.leaveId!, requestData as Leave.UpdateRequest)
      ElMessage.success('更新成功')
    }
    else {
      await createLeave(requestData as Leave.CreateRequest)
      ElMessage.success('创建成功')
    }

    emit('success')
    handleClose()
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
  finally {
    submitting.value = false
  }
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
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      v-loading="loading"
      :model="formData"
      :rules="rules"
      label-width="100px"
    >
      <el-form-item label="员工" prop="employeeId">
        <el-select
          v-model="formData.employeeId"
          placeholder="请选择员工"
          filterable
          remote
          :remote-method="handleSearchEmployee"
          :loading="employeeLoading"
          :disabled="isEdit"
          clearable
          style="width: 100%"
          @focus="handleEmployeeFocus"
        >
          <el-option
            v-for="emp in employeeList"
            :key="emp.id"
            :label="`${emp.name} (${emp.employeeId})`"
            :value="emp.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="假期类型" prop="type">
        <el-select
          v-model="formData.type"
          placeholder="请选择假期类型"
          :disabled="isEdit"
          style="width: 100%"
        >
          <el-option
            v-for="item in leaveTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="开始日期" prop="startDate">
        <el-date-picker
          v-model="formData.startDate"
          type="date"
          placeholder="选择日期"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="结束日期" prop="endDate">
        <el-date-picker
          v-model="formData.endDate"
          type="date"
          placeholder="选择日期"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="开始时间">
        <el-time-picker
          v-model="formData.startTime"
          placeholder="选择时间（可选）"
          format="HH:mm"
          value-format="HH:mm"
          style="width: 100%"
        />
        <div style="color: var(--el-text-color-secondary); font-size: 12px; margin-top: 4px;">
          小时级请假需要填写具体时间
        </div>
      </el-form-item>

      <el-form-item label="结束时间">
        <el-time-picker
          v-model="formData.endTime"
          placeholder="选择时间（可选）"
          format="HH:mm"
          value-format="HH:mm"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="请假事由">
        <el-input
          v-model="formData.reason"
          type="textarea"
          :rows="3"
          placeholder="请输入请假事由"
          maxlength="200"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
</style>
