<script setup lang="ts">
import type { Shift } from '@/types/shift'
import { Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { reactive, ref } from 'vue'
import { createShift, deleteShift, listShifts, updateShift } from '@/api/shifts'
import { usePagination } from '@/composables/usePagination'

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Shift>({
  fetchFn: listShifts,
})

// ======== 表单弹窗 ========

const dialogVisible = ref(false)
const dialogTitle = ref('新增班次')
const formLoading = ref(false)
const editingId = ref<string | null>(null)

const shiftTypes = [
  { value: 'regular', label: '常规' },
  { value: 'overtime', label: '加班' },
  { value: 'oncall', label: '值班' },
]

const form = reactive({
  name: '',
  code: '',
  color: '#409EFF',
  start_time: '',
  end_time: '',
  type: 'regular' as const,
  description: '',
})

const rules = {
  name: [{ required: true, message: '请输入班次名称', trigger: 'blur' }],
  start_time: [{ required: true, message: '请选择开始时间', trigger: 'change' }],
  end_time: [{ required: true, message: '请选择结束时间', trigger: 'change' }],
}

const formRef = ref()

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增班次'
  Object.assign(form, { name: '', code: '', color: '#409EFF', start_time: '', end_time: '', type: 'regular', description: '' })
  dialogVisible.value = true
}

function handleEdit(row: Shift) {
  editingId.value = row.id
  dialogTitle.value = '编辑班次'
  Object.assign(form, {
    name: row.name,
    code: row.code || '',
    color: row.color,
    start_time: row.start_time,
    end_time: row.end_time,
    type: row.type,
    description: row.description || '',
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
      await updateShift(editingId.value, form)
      ElMessage.success('更新成功')
    }
    else {
      await createShift(form)
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

async function handleDelete(row: Shift) {
  await ElMessageBox.confirm(`确定删除班次「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteShift(row.id)
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
    <div class="page-toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索班次"
        clearable
        style="width: 240px"
        :prefix-icon="Search"
      />
      <el-button type="primary" :icon="Plus" @click="handleAdd">
        新增班次
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="name" label="名称" width="140">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 8px">
            <span class="color-dot" :style="{ background: row.color }" />
            {{ row.name }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="code" label="编码" width="100" />
      <el-table-column prop="start_time" label="开始时间" width="100" />
      <el-table-column prop="end_time" label="结束时间" width="100" />
      <el-table-column prop="duration" label="时长(分)" width="100" />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'regular' ? '' : row.type === 'overtime' ? 'warning' : 'danger'" size="small">
            {{ shiftTypes.find(t => t.value === row.type)?.label || row.type }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" />
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
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="如：早班" />
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" placeholder="如：A" />
        </el-form-item>
        <el-form-item label="颜色" prop="color">
          <el-color-picker v-model="form.color" />
        </el-form-item>
        <el-form-item label="开始时间" prop="start_time">
          <el-time-select v-model="form.start_time" start="00:00" step="00:30" end="23:30" placeholder="选择时间" />
        </el-form-item>
        <el-form-item label="结束时间" prop="end_time">
          <el-time-select v-model="form.end_time" start="00:00" step="00:30" end="23:30" placeholder="选择时间" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="选择类型">
            <el-option v-for="t in shiftTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选" />
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

.color-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>
