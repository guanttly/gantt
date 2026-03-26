<script setup lang="ts">
import type { Employee } from '@/types/employee'
import { Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { reactive, ref } from 'vue'
import { createEmployee, deleteEmployee, listEmployees, updateEmployee } from '@/api/employees'
import { usePagination } from '@/composables/usePagination'

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Employee>({
  fetchFn: listEmployees,
})

// ======== 表单弹窗 ========

const dialogVisible = ref(false)
const dialogTitle = ref('新增员工')
const formLoading = ref(false)
const editingId = ref<string | null>(null)

const form = reactive({
  name: '',
  employee_no: '',
  phone: '',
  email: '',
  position: '',
})

const rules = {
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
}

const formRef = ref()

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增员工'
  Object.assign(form, { name: '', employee_no: '', phone: '', email: '', position: '' })
  dialogVisible.value = true
}

function handleEdit(row: Employee) {
  editingId.value = row.id
  dialogTitle.value = '编辑员工'
  Object.assign(form, {
    name: row.name,
    employee_no: row.employee_no || '',
    phone: row.phone || '',
    email: row.email || '',
    position: row.position || '',
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
    if (editingId.value) {
      await updateEmployee(editingId.value, form)
      ElMessage.success('更新成功')
    }
    else {
      await createEmployee(form)
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

async function handleDelete(row: Employee) {
  await ElMessageBox.confirm(`确定删除员工「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteEmployee(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}
</script>

<template>
  <div class="page-container">
    <!-- 工具栏 -->
    <div class="page-toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索员工"
        clearable
        style="width: 240px"
        :prefix-icon="Search"
      />
      <el-button type="primary" :icon="Plus" @click="handleAdd">
        新增员工
      </el-button>
    </div>

    <!-- 表格 -->
    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="employee_no" label="工号" width="120" />
      <el-table-column prop="name" label="姓名" width="120" />
      <el-table-column prop="position" label="职位" width="140" />
      <el-table-column prop="phone" label="电话" width="140" />
      <el-table-column prop="email" label="邮箱" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '在职' : '离职' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button :icon="Edit" link type="primary" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button :icon="Delete" link type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
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

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入姓名" />
        </el-form-item>
        <el-form-item label="工号" prop="employee_no">
          <el-input v-model="form.employee_no" placeholder="请输入工号" />
        </el-form-item>
        <el-form-item label="职位" prop="position">
          <el-input v-model="form.position" placeholder="请输入职位" />
        </el-form-item>
        <el-form-item label="电话" prop="phone">
          <el-input v-model="form.phone" placeholder="请输入电话" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
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
