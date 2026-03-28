<script setup lang="ts">
import type { Group, GroupMember } from '@/api/groups'
import type { Employee } from '@/types/employee'
import { Delete, Edit, Plus, Search, User } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref } from 'vue'
import { listEmployees } from '@/api/employees'
import { addGroupMember, createGroup, deleteGroup, getGroupMembers, listGroups, removeGroupMember, updateGroup } from '@/api/groups'
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
const memberDialogVisible = ref(false)
const memberDialogLoading = ref(false)
const memberSaving = ref(false)
const employeeOptionsLoading = ref(false)
const currentGroup = ref<Group | null>(null)
const members = ref<GroupMember[]>([])
const memberCurrentPage = ref(1)
const memberPageSize = ref(10)
const employeeOptions = ref<Employee[]>([])
const employeeKeyword = ref('')
const selectedEmployeeId = ref('')

const form = reactive({
  name: '',
  description: '',
})

const rules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }],
}

const formRef = ref()

const availableEmployees = computed(() => {
  const memberIds = new Set(members.value.map(member => member.employee_id))
  return employeeOptions.value.map(employee => ({
    ...employee,
    isAdded: memberIds.has(employee.id),
  }))
})

const pagedMembers = computed(() => {
  const start = (memberCurrentPage.value - 1) * memberPageSize.value
  return members.value.slice(start, start + memberPageSize.value)
})

const memberTotal = computed(() => members.value.length)

function syncMemberPagination() {
  const maxPage = Math.max(1, Math.ceil(memberTotal.value / memberPageSize.value))
  if (memberCurrentPage.value > maxPage) {
    memberCurrentPage.value = maxPage
  }
}

function formatExactDateTime(value?: string) {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

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

async function loadEmployeeOptions(keyword = '') {
  employeeOptionsLoading.value = true
  employeeOptions.value = []
  try {
    const res = await listEmployees({ page: 1, page_size: 100, keyword })
    employeeOptions.value = Array.isArray(res) ? res : res.items
  }
  finally {
    employeeOptionsLoading.value = false
  }
}

async function loadMembers(groupId: string) {
  members.value = await getGroupMembers(groupId)
  syncMemberPagination()
}

async function handleManageMembers(row: Group) {
  currentGroup.value = row
  memberDialogVisible.value = true
  memberCurrentPage.value = 1
  memberPageSize.value = 10
  selectedEmployeeId.value = ''
  employeeKeyword.value = ''
  memberDialogLoading.value = true
  try {
    await Promise.all([
      loadMembers(row.id),
      loadEmployeeOptions(),
    ])
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加载分组成员失败')
    memberDialogVisible.value = false
  }
  finally {
    memberDialogLoading.value = false
  }
}

async function handleAddMember() {
  if (!currentGroup.value || !selectedEmployeeId.value) {
    ElMessage.warning('请选择要加入分组的员工')
    return
  }

  memberSaving.value = true
  try {
    await addGroupMember(currentGroup.value.id, { employee_id: selectedEmployeeId.value })
    ElMessage.success('已加入分组')
    selectedEmployeeId.value = ''
    await Promise.all([
      loadMembers(currentGroup.value.id),
      loadEmployeeOptions(employeeKeyword.value),
    ])
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加入分组失败')
  }
  finally {
    memberSaving.value = false
  }
}

async function handleRemoveMember(member: GroupMember) {
  if (!currentGroup.value) {
    return
  }

  await ElMessageBox.confirm(`确定将「${member.employee_name || member.employee_no || member.employee_id}」移出该分组吗？`, '移除成员', { type: 'warning' })
  memberSaving.value = true
  try {
    await removeGroupMember(currentGroup.value.id, member.employee_id)
    ElMessage.success('已移出分组')
    await Promise.all([
      loadMembers(currentGroup.value.id),
      loadEmployeeOptions(employeeKeyword.value),
    ])
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '移除成员失败')
  }
  finally {
    memberSaving.value = false
  }
}

async function handleEmployeeSearch(query: string) {
  employeeKeyword.value = query
  try {
    await loadEmployeeOptions(query)
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '查询员工失败')
  }
}

function handleEmployeeSelectVisible(visible: boolean) {
  if (visible && !employeeOptions.value.length) {
    void handleEmployeeSearch('')
  }
}

function handleMemberPageChange(page: number) {
  memberCurrentPage.value = page
}

function handleMemberSizeChange(size: number) {
  memberPageSize.value = size
  memberCurrentPage.value = 1
  syncMemberPagination()
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
      <el-table-column prop="member_count" label="成员数" width="100">
        <template #default="{ row }">
          {{ row.member_count ?? 0 }}
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.description" class="description-text">{{ row.description }}</span>
          <span v-else class="cell-placeholder">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">
          <span class="time-text">{{ formatExactDateTime(row.created_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column v-if="canManageGroups" label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button :icon="User" link type="primary" @click="handleManageMembers(row)">
            人员分配
          </el-button>
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

    <el-dialog v-model="memberDialogVisible" class="member-dialog-panel" :title="currentGroup ? `${currentGroup.name} · 人员分配` : '人员分配'" width="760px">
      <div v-loading="memberDialogLoading" class="member-dialog">
        <div class="member-toolbar">
          <el-select
            v-model="selectedEmployeeId"
            filterable
            remote
            clearable
            reserve-keyword
            placeholder="搜索员工姓名 / 工号后加入分组"
            style="flex: 1"
            :loading="employeeOptionsLoading"
            loading-text="搜索中..."
            no-data-text="暂无匹配员工"
            remote-show-suffix
            @visible-change="handleEmployeeSelectVisible"
            :remote-method="handleEmployeeSearch"
          >
            <el-option
              v-for="employee in availableEmployees"
              :key="employee.id"
              :label="`${employee.name}${employee.employee_no ? `（${employee.employee_no}）` : ''}`"
              :value="employee.id"
              :disabled="employee.isAdded"
            >
              <div class="employee-option" :class="{ 'is-added': employee.isAdded }">
                <div class="employee-option-main">
                  <span>{{ employee.name }}</span>
                  <span class="employee-option-meta">{{ employee.employee_no || employee.position || '未设置工号' }}</span>
                </div>
                <span v-if="employee.isAdded" class="employee-option-state">已添加</span>
              </div>
            </el-option>
          </el-select>
          <el-button type="primary" :loading="memberSaving" @click="handleAddMember">
            加入分组
          </el-button>
        </div>

        <el-empty v-if="!members.length && !memberDialogLoading" description="当前分组还没有成员" />

        <div v-else class="member-table-section">
          <el-table :data="pagedMembers" border stripe class="member-table" :max-height="420">
            <el-table-column label="姓名" min-width="140">
              <template #default="{ row }">
                {{ row.employee_name || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="工号" width="140">
              <template #default="{ row }">
                {{ row.employee_no || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="岗位" width="140">
              <template #default="{ row }">
                {{ row.position || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="加入时间" width="180">
              <template #default="{ row }">
                {{ formatExactDateTime(row.joined_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" :loading="memberSaving" @click="handleRemoveMember(row)">
                  移除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="member-pagination">
            <el-pagination
              v-model:current-page="memberCurrentPage"
              v-model:page-size="memberPageSize"
              :total="memberTotal"
              :page-sizes="[10, 20, 50]"
              layout="total, sizes, prev, pager, next"
              @current-change="handleMemberPageChange"
              @size-change="handleMemberSizeChange"
            />
          </div>
        </div>
      </div>
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

.member-dialog {
  min-height: 240px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.member-toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.member-table-section {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.member-table {
  width: 100%;
}

.member-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  flex-shrink: 0;
}

.employee-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.employee-option-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.employee-option-meta {
  color: var(--el-text-color-secondary);
}

.employee-option-state {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  flex-shrink: 0;
}

.employee-option.is-added {
  color: var(--el-text-color-secondary);
}

.employee-option.is-added .employee-option-meta {
  color: var(--el-text-color-placeholder);
}

.description-text {
  color: var(--el-text-color-primary);
}

.cell-placeholder {
  color: var(--el-text-color-placeholder);
}

.time-text {
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

:deep(.member-dialog-panel .el-dialog__body) {
  max-height: calc(100vh - 220px);
  overflow: hidden;
}
</style>
