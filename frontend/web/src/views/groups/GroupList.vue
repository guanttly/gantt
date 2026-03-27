<script setup lang="ts">
import type { Group } from '@/api/groups'
import { Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref } from 'vue'
import { createGroup, deleteGroup, listGroups, updateGroup } from '@/api/groups'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canManageGroups = computed(() => auth.hasPermission('group:manage'))

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Group>({
  fetchFn: listGroups,
})

const dialogVisible = ref(false)
const dialogTitle = ref('新增分组')
const formLoading = ref(false)
const editingId = ref<string | null>(null)

const form = reactive({
  name: '',
  description: '',
})

const rules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }],
}

const formRef = ref()

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增分组'
  Object.assign(form, { name: '', description: '' })
  dialogVisible.value = true
}

function handleEdit(row: Group) {
  editingId.value = row.id
  dialogTitle.value = '编辑分组'
  Object.assign(form, { name: row.name, description: row.description || '' })
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
      await updateGroup(editingId.value, form)
      ElMessage.success('更新成功')
    }
    else {
      await createGroup(form)
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

async function handleDelete(row: Group) {
  await ElMessageBox.confirm(`确定删除分组「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteGroup(row.id)
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
      <el-input v-model="keyword" placeholder="搜索分组" clearable style="width: 240px" :prefix-icon="Search" />
      <el-button v-if="canManageGroups" type="primary" :icon="Plus" @click="handleAdd">
        新增分组
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="name" label="名称" width="200" />
      <el-table-column prop="member_count" label="成员数" width="100" />
      <el-table-column prop="description" label="描述" />
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column v-if="canManageGroups" label="操作" width="160" fixed="right">
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="可选" />
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
