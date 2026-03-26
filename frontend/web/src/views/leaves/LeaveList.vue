<script setup lang="ts">
import type { Leave } from '@/api/leaves'
import { Check, Close, Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref } from 'vue'
import { approveLeave, createLeave, deleteLeave, listLeaves, updateLeave } from '@/api/leaves'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Leave>({
  fetchFn: listLeaves,
})
const auth = useAuthStore()

const canCreateOwnLeave = computed(() => auth.hasPermission('leave:create:self'))
const canApproveLeave = computed(() => auth.hasPermission('leave:approve'))
const currentEmployeeId = computed(() => auth.boundEmployeeId)

const statusMap: Record<string, { label: string, type: string }> = {
  pending: { label: '待审批', type: 'warning' },
  approved: { label: '已通过', type: 'success' },
  rejected: { label: '已驳回', type: 'danger' },
}

const dialogVisible = ref(false)
const dialogTitle = ref('新增请假')
const formLoading = ref(false)
const editingId = ref<string | null>(null)

const form = reactive({
  employee_id: '',
  type: '年假',
  start_date: '',
  end_date: '',
  reason: '',
})

const rules = {
  employee_id: [{ required: true, message: '请选择员工', trigger: 'change' }],
  start_date: [{ required: true, message: '请选择开始日期', trigger: 'change' }],
  end_date: [{ required: true, message: '请选择结束日期', trigger: 'change' }],
}

const formRef = ref()

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增请假'
  Object.assign(form, { employee_id: currentEmployeeId.value || '', type: '年假', start_date: '', end_date: '', reason: '' })
  dialogVisible.value = true
}

function handleEdit(row: Leave) {
  editingId.value = row.id
  dialogTitle.value = '编辑请假'
  Object.assign(form, {
    employee_id: row.employee_id,
    type: row.type,
    start_date: row.start_date,
    end_date: row.end_date,
    reason: row.reason || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }
  formLoading.value = true
  try {
    form.employee_id = currentEmployeeId.value || form.employee_id
    if (editingId.value) {
      await updateLeave(editingId.value, form)
      ElMessage.success('更新成功')
    }
    else {
      await createLeave(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  }
  finally {
    formLoading.value = false
  }
}

async function handleDelete(row: Leave) {
  await ElMessageBox.confirm(`确定删除此请假记录吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteLeave(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}

async function handleApprove(row: Leave, approved: boolean) {
  try {
    await approveLeave(row.id, { approved })
    ElMessage.success(approved ? '审批通过' : '已驳回')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '审批失败')
  }
}

function canEditOwnLeave(row: Leave) {
  return canCreateOwnLeave.value && row.status === 'pending' && row.employee_id === currentEmployeeId.value
}
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <el-input v-model="keyword" placeholder="搜索请假" clearable style="width: 240px" :prefix-icon="Search" />
      <el-button v-if="canCreateOwnLeave" type="primary" :icon="Plus" @click="handleAdd">
        新增请假
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="employee_name" label="员工" width="120" />
      <el-table-column prop="type" label="类型" width="100" />
      <el-table-column prop="start_date" label="开始日期" width="120" />
      <el-table-column prop="end_date" label="结束日期" width="120" />
      <el-table-column prop="reason" label="原因" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="(statusMap[row.status]?.type as any)" size="small">
            {{ statusMap[row.status]?.label || row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" :width="canApproveLeave ? 220 : 160" fixed="right">
        <template #default="{ row }">
          <el-button v-if="canEditOwnLeave(row)" :icon="Edit" link type="primary" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button v-if="canEditOwnLeave(row)" :icon="Delete" link type="danger" @click="handleDelete(row)">
            删除
          </el-button>
          <el-button v-if="canApproveLeave && row.status === 'pending'" :icon="Check" link type="success" @click="handleApprove(row, true)">
            通过
          </el-button>
          <el-button v-if="canApproveLeave && row.status === 'pending'" :icon="Close" link type="danger" @click="handleApprove(row, false)">
            驳回
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="page-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="currentPageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="员工ID" prop="employee_id">
          <el-input v-model="form.employee_id" disabled placeholder="当前登录员工" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" placeholder="选择类型">
            <el-option label="年假" value="年假" />
            <el-option label="病假" value="病假" />
            <el-option label="事假" value="事假" />
            <el-option label="婚假" value="婚假" />
            <el-option label="产假" value="产假" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始日期" prop="start_date">
          <el-date-picker v-model="form.start_date" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="结束日期" prop="end_date">
          <el-date-picker v-model="form.end_date" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="form.reason" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="formLoading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow: hidden;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
