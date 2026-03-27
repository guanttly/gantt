<script setup lang="ts">
import type { AppRoleGrant, AppRoleName } from '@/types/auth'
import type { Employee } from '@/types/employee'
import { Plus, RefreshRight, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { assignEmployeeAppRole, listEmployeeAppRoles, removeEmployeeAppRole } from '@/api/appRoles'
import { listEmployees } from '@/api/employees'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const roleOptions: Array<{ label: string, value: AppRoleName }> = [
  { label: '科室管理员', value: 'app:schedule_admin' },
  { label: '排班员', value: 'app:scheduler' },
  { label: '请假审批人', value: 'app:leave_approver' },
]

const roleLabelMap: Record<AppRoleName, string> = {
  'app:schedule_admin': '科室管理员',
  'app:scheduler': '排班员',
  'app:leave_approver': '请假审批人',
}

const roleTagTypeMap: Record<AppRoleName, '' | 'success' | 'warning'> = {
  'app:schedule_admin': 'success',
  'app:scheduler': '',
  'app:leave_approver': 'warning',
}

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Employee>({
  fetchFn: listEmployees,
})

const roleMap = ref<Record<string, AppRoleGrant[]>>({})
const roleLoadingMap = ref<Record<string, boolean>>({})
const dialogVisible = ref(false)
const submitLoading = ref(false)
const selectedEmployee = ref<Employee | null>(null)

const form = reactive({
  app_role: 'app:scheduler' as AppRoleName,
})

const canManageAppRoles = computed(() => auth.hasPermission('app-role:manage'))

async function loadEmployeeRoles(employeeId: string) {
  roleLoadingMap.value = { ...roleLoadingMap.value, [employeeId]: true }
  try {
    roleMap.value = {
      ...roleMap.value,
      [employeeId]: await listEmployeeAppRoles(employeeId),
    }
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '加载员工应用角色失败')
  }
  finally {
    roleLoadingMap.value = { ...roleLoadingMap.value, [employeeId]: false }
  }
}

async function loadVisibleRoles(rows: Employee[]) {
  await Promise.all(rows.map(row => loadEmployeeRoles(row.id)))
}

function handleGrant(row: Employee) {
  selectedEmployee.value = row
  form.app_role = 'app:scheduler'
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!selectedEmployee.value || !auth.currentNodeId)
    return

  submitLoading.value = true
  try {
    await assignEmployeeAppRole(selectedEmployee.value.id, {
      app_role: form.app_role,
      org_node_id: auth.currentNodeId,
    })
    ElMessage.success('授权成功')
    dialogVisible.value = false
    await loadEmployeeRoles(selectedEmployee.value.id)
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '授权失败')
  }
  finally {
    submitLoading.value = false
  }
}

async function handleRemoveRole(row: Employee, role: AppRoleGrant) {
  if (role.source !== 'manual')
    return

  await ElMessageBox.confirm(`确定移除 ${row.name} 的${getRoleLabel(role.app_role)}吗？`, '确认移除', { type: 'warning' })
  try {
    await removeEmployeeAppRole(row.id, role.id)
    ElMessage.success('移除成功')
    await loadEmployeeRoles(row.id)
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '移除失败')
  }
}

function getRoleLabel(roleName: string) {
  return roleLabelMap[roleName as AppRoleName] || roleName
}

function getRoleTagType(roleName: string) {
  return roleTagTypeMap[roleName as AppRoleName] || 'info'
}

watch(items, rows => loadVisibleRoles(rows), { immediate: true })
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索员工"
        clearable
        style="width: 240px"
        :prefix-icon="Search"
      />
      <el-button :icon="RefreshRight" @click="refresh">
        刷新
      </el-button>
    </div>

    <el-alert
      title="仅管理当前节点内员工的应用角色。继承自分组的角色不可在此直接移除。"
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
    />

    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="employee_no" label="工号" width="120" />
      <el-table-column prop="name" label="姓名" width="120" />
      <el-table-column prop="position" label="职位" width="160" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '在职' : row.status === 'on_leave' ? '请假中' : '离岗' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="当前应用角色" min-width="320">
        <template #default="{ row }">
          <div v-loading="roleLoadingMap[row.id]" class="role-list">
            <template v-if="(roleMap[row.id] || []).length">
              <el-tag
                v-for="role in roleMap[row.id] || []"
                :key="role.id"
                :type="getRoleTagType(role.app_role)"
                :closable="role.source === 'manual'"
                size="small"
                class="role-tag"
                @close="handleRemoveRole(row, role)"
              >
                {{ getRoleLabel(role.app_role) }}
                <span class="role-source">{{ role.source === 'manual' ? '手动' : role.source === 'group' ? '分组继承' : '系统' }}</span>
              </el-tag>
            </template>
            <span v-else class="empty-text">未授予</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column v-if="canManageAppRoles" label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link :icon="Plus" @click="handleGrant(row)">
            授权
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

    <el-dialog v-model="dialogVisible" title="授予应用角色" width="420px">
      <el-form label-width="90px">
        <el-form-item label="员工">
          <span>{{ selectedEmployee?.name || '-' }}</span>
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.app_role" style="width: 100%">
            <el-option v-for="item in roleOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">
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

.role-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 32px;
  align-items: center;
}

.role-tag {
  margin-right: 0;
}

.role-source {
  margin-left: 6px;
  opacity: 0.75;
}

.empty-text {
  color: #909399;
}
</style>