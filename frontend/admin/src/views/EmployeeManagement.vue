<script setup lang="ts">
import type { PlatformEmployee, PlatformEmployeeAppRole, PlatformEmployeePayload } from '@/api/platform'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NPagination, NSelect, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { getOrgTree } from '@/api/org'
import { assignEmployeeAppRole, createPlatformEmployee, deletePlatformEmployee, listPlatformEmployees, removeEmployeeAppRole, resetPlatformEmployeePassword, transferEmployee, updatePlatformEmployee } from '@/api/platform'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const employees = ref<PlatformEmployee[]>([])
const employeeRoleMap = ref<Record<string, PlatformEmployeeAppRole[]>>({})
const orgTree = ref<any[]>([])
const keyword = ref('')
const currentPage = ref(1)
const pageSize = ref(5)
const total = ref(0)
const dialogVisible = ref(false)
const editingEmployee = ref<PlatformEmployee | null>(null)
const message = useMessage()
const dialog = useDialog()

const transferVisible = ref(false)
const transferring = ref(false)
const roleUpdatingEmployeeId = ref<string | null>(null)
const transferTarget = ref<PlatformEmployee | null>(null)
const transferForm = ref({ target_org_node_id: '', reason: '' })

const form = ref<PlatformEmployeePayload>({
  org_node_id: '',
  name: '',
  employee_no: '',
  phone: '',
  email: '',
  position: '',
  category: '',
  hire_date: '',
  status: 'active',
})

const statusOptions = [
  { label: '在职', value: 'active' },
  { label: '停用', value: 'inactive' },
]

const orgOptions = computed(() => flattenOrgTree(orgTree.value))
const departmentOptions = computed(() => flattenOrgTree(orgTree.value).filter(item => item.nodeType === 'department'))

type OrgOption = {
  label: string
  value: string
  nodeType: string
}

function getDefaultDepartmentId() {
  const currentNodeId = auth.currentNode?.node_id
  if (currentNodeId && departmentOptions.value.some(item => item.value === currentNodeId)) {
    return currentNodeId
  }
  return departmentOptions.value[0]?.value || ''
}

function getDefaultTransferDepartmentId(currentOrgNodeId: string) {
  return departmentOptions.value.find(item => item.value !== currentOrgNodeId)?.value || ''
}

function normalizePayload(payload: PlatformEmployeePayload): PlatformEmployeePayload {
  return Object.fromEntries(Object.entries(payload).filter(([, value]) => value !== '' && value !== undefined)) as PlatformEmployeePayload
}

function flattenOrgTree(nodes: any[], ancestors: string[] = []): OrgOption[] {
  return nodes.flatMap((node): OrgOption[] => {
    const nextAncestors = [...ancestors, node.name]
    return [
      { label: nextAncestors.join(' / '), value: node.id, nodeType: node.node_type },
      ...flattenOrgTree(node.children || [], nextAncestors),
    ]
  })
}

function orgNodeName(orgNodeId: string) {
  return orgOptions.value.find(item => item.value === orgNodeId)?.label.trim() || orgNodeId
}

function resetForm() {
  form.value = {
    org_node_id: getDefaultDepartmentId(),
    name: '',
    employee_no: '',
    phone: '',
    email: '',
    position: '',
    category: '',
    hire_date: '',
    status: 'active',
  }
}

async function loadEmployees() {
  loading.value = true
  try {
    const result = await listPlatformEmployees({
      page: currentPage.value,
      size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    employees.value = result.data
    total.value = result.total
    pageSize.value = result.size
    employeeRoleMap.value = Object.fromEntries(result.data.map(employee => [employee.id, employee.app_roles || []]))
  }
  finally {
    loading.value = false
  }
}

async function loadOrgTree() {
  if (orgTree.value.length > 0) {
    return
  }
  orgTree.value = await getOrgTree()
}
onMounted(async () => {
  await Promise.all([loadOrgTree(), loadEmployees()])
})

function handleSearch() {
  currentPage.value = 1
  loadEmployees()
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadEmployees()
}

function handlePageSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  loadEmployees()
}

function employeeRoles(employeeId: string) {
  return employeeRoleMap.value[employeeId] || []
}

function currentDepartmentAdminRole(employee: PlatformEmployee) {
  return employeeRoles(employee.id).find(role => role.app_role === 'app:schedule_admin' && role.org_node_id === employee.org_node_id)
}

function hasCurrentDepartmentAdminRole(employee: PlatformEmployee) {
  return !!currentDepartmentAdminRole(employee)
}

function appRoleLabel(role: PlatformEmployeeAppRole) {
  if (role.app_role === 'app:schedule_admin') {
    return '科室管理员'
  }
  if (role.app_role === 'app:scheduler') {
    return '排班员'
  }
  if (role.app_role === 'app:leave_approver') {
    return '请假审批人'
  }
  return role.app_role
}

function openCreate() {
  editingEmployee.value = null
  resetForm()
  dialogVisible.value = true
}

function openEdit(employee: PlatformEmployee) {
  editingEmployee.value = employee
  form.value = {
    org_node_id: employee.org_node_id,
    name: employee.name,
    employee_no: employee.employee_no,
    phone: employee.phone,
    email: employee.email,
    position: employee.position,
    category: employee.category,
    hire_date: employee.hire_date,
    status: employee.status,
  }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.name?.trim()) {
    message.warning('请输入员工姓名')
    return
  }
  if (!form.value.org_node_id) {
    message.warning('请选择所属科室')
    return
  }

  saving.value = true
  try {
    if (editingEmployee.value) {
      await updatePlatformEmployee(editingEmployee.value.id, normalizePayload(form.value))
      message.success('员工信息已更新')
    }
    else {
      const created = await createPlatformEmployee(normalizePayload(form.value))
      message.success('员工已创建')
      if (created.app_default_password) {
        dialog.success({
          title: '员工初始应用密码',
          content: `员工 ${created.name} 的初始排班应用密码为：${created.app_default_password}`,
          positiveText: '知道了',
        })
      }
    }
    dialogVisible.value = false
    await loadEmployees()
  }
  finally {
    saving.value = false
  }
}

async function removeEmployee(employee: PlatformEmployee) {
  dialog.warning({
    title: '确认删除员工',
    content: `确定删除「${employee.name}」？该操作不可撤销。`,
    positiveText: '删除',
    negativeText: '取消',
    async onPositiveClick() {
      await deletePlatformEmployee(employee.id)
      message.success('员工已删除')
      await loadEmployees()
    },
  })
}

function resetEmployeePassword(employee: PlatformEmployee) {
  dialog.warning({
    title: '重置员工应用密码',
    content: `确定重置 ${employee.name} 的默认密码？`,
    positiveText: '重置',
    negativeText: '取消',
    async onPositiveClick() {
      const result = await resetPlatformEmployeePassword(employee.id)
      dialog.success({
        title: '密码已重置',
        content: `员工 ${employee.name} 的新默认密码为：${result.default_password}`,
        positiveText: '知道了',
      })
      await loadEmployees()
    },
  })
}

function toggleDepartmentAdmin(employee: PlatformEmployee) {
  const existingRole = currentDepartmentAdminRole(employee)
  if (existingRole) {
    dialog.warning({
      title: '取消科室管理员',
      content: `确定取消「${employee.name}」在当前科室的管理员权限？`,
      positiveText: '取消权限',
      negativeText: '保留',
      async onPositiveClick() {
        roleUpdatingEmployeeId.value = employee.id
        try {
          await removeEmployeeAppRole(employee.id, existingRole.id)
          message.success('已取消科室管理员权限')
          await loadEmployees()
        }
        finally {
          roleUpdatingEmployeeId.value = null
        }
      },
    })
    return
  }

  dialog.info({
    title: '指定科室管理员',
    content: `确定将「${employee.name}」指定为当前科室管理员？`,
    positiveText: '指定',
    negativeText: '取消',
    async onPositiveClick() {
      roleUpdatingEmployeeId.value = employee.id
      try {
        await assignEmployeeAppRole(employee.id, {
          app_role: 'app:schedule_admin',
          org_node_id: employee.org_node_id,
        })
        message.success('已指定为科室管理员')
        await loadEmployees()
      }
      finally {
        roleUpdatingEmployeeId.value = null
      }
    },
  })
}

function openTransfer(employee: PlatformEmployee) {
  transferTarget.value = employee
  transferForm.value = {
    target_org_node_id: getDefaultTransferDepartmentId(employee.org_node_id),
    reason: '',
  }
  transferVisible.value = true
}

async function submitTransfer() {
  if (!transferTarget.value || !transferForm.value.target_org_node_id) {
    message.warning('请选择目标组织节点')
    return
  }
  transferring.value = true
  try {
    const result = await transferEmployee(transferTarget.value.id, transferForm.value)
    message.success(`已将「${transferTarget.value.name}」从「${result.from_org_node.name}」调动至「${result.to_org_node.name}」`)
    transferVisible.value = false
    await loadEmployees()
  }
  catch (e: any) {
    message.error(e?.response?.data?.message || '调动失败')
  }
  finally {
    transferring.value = false
  }
}

onMounted(loadEmployees)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">员工管理</h2>
          <p class="page-subtitle">维护机构内员工档案，并将员工绑定到具体科室，同时同步生成排班应用初始账号。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-input
            v-model:value="keyword"
            clearable
            placeholder="搜索姓名、工号、手机号"
            style="width: 320px"
            @keyup.enter="handleSearch"
            @clear="handleSearch"
          />
        </div>
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增员工</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <div class="table-shell">
              <table class="admin-table">
                <thead>
                  <tr>
                    <th>所属组织</th>
                    <th>姓名</th>
                    <th>工号</th>
                    <th>职位</th>
                    <th>应用角色</th>
                    <th>联系方式</th>
                    <th>状态</th>
                    <th>需改密</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in employees" :key="item.id">
                    <td>
                      <div>{{ item.org_node_path_display || orgNodeName(item.org_node_id) }}</div>
                      <div v-if="item.org_node_type" class="table-muted">{{ item.org_node_type }}</div>
                    </td>
                    <td>{{ item.name }}</td>
                    <td>{{ item.employee_no || '-' }}</td>
                    <td>{{ item.position || '-' }}</td>
                    <td>
                      <div class="table-role-list">
                        <n-tag v-for="role in employeeRoles(item.id)" :key="role.id" size="small" :type="role.app_role === 'app:schedule_admin' ? 'success' : 'default'">
                          {{ appRoleLabel(role) }}
                        </n-tag>
                        <span v-if="!employeeRoles(item.id).length" class="table-muted">-</span>
                      </div>
                    </td>
                    <td>
                      <div>{{ item.phone || '-' }}</div>
                      <div class="table-muted">{{ item.email || '-' }}</div>
                    </td>
                    <td>
                      <n-tag :type="item.status === 'active' ? 'success' : 'default'" size="small">
                        {{ item.status === 'active' ? '在职' : '停用' }}
                      </n-tag>
                    </td>
                    <td>{{ item.app_must_reset_pwd ? '是' : '否' }}</td>
                    <td>
                      <div class="table-actions">
                        <n-button text type="primary" @click="openEdit(item)">编辑</n-button>
                        <n-button text type="primary" @click="resetEmployeePassword(item)">重置密码</n-button>
                        <n-button
                          text
                          :type="hasCurrentDepartmentAdminRole(item) ? 'warning' : 'success'"
                          :loading="roleUpdatingEmployeeId === item.id"
                          @click="toggleDepartmentAdmin(item)"
                        >
                          {{ hasCurrentDepartmentAdminRole(item) ? '撤管理员' : '设管理员' }}
                        </n-button>
                        <n-button text type="warning" @click="openTransfer(item)">调动</n-button>
                        <n-button text type="error" @click="removeEmployee(item)">删除</n-button>
                      </div>
                    </td>
                  </tr>
                  <tr v-if="!employees.length">
                    <td colspan="9" class="table-empty">暂无员工数据</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </n-spin>

          <div class="page-pagination">
            <n-pagination
              :page="currentPage"
              :page-size="pageSize"
              :item-count="total"
              :page-sizes="[5, 10, 20, 50]"
              show-size-picker
              @update:page="handlePageChange"
              @update:page-size="handlePageSizeChange"
            />
          </div>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="editingEmployee ? '编辑员工' : '新增员工'" style="width: min(640px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="88">
          <div class="form-grid two-column">
            <n-form-item label="所属科室">
              <n-select v-model:value="form.org_node_id" filterable :options="departmentOptions" placeholder="选择员工所属科室" />
            </n-form-item>
            <n-form-item label="姓名">
              <n-input v-model:value="form.name" placeholder="输入员工姓名" />
            </n-form-item>
            <n-form-item label="工号">
              <n-input v-model:value="form.employee_no" placeholder="输入员工工号" />
            </n-form-item>
            <n-form-item label="手机号">
              <n-input v-model:value="form.phone" placeholder="输入手机号" />
            </n-form-item>
            <n-form-item label="邮箱">
              <n-input v-model:value="form.email" placeholder="输入邮箱" />
            </n-form-item>
            <n-form-item label="职位">
              <n-input v-model:value="form.position" placeholder="输入职位" />
            </n-form-item>
            <n-form-item label="分类">
              <n-input v-model:value="form.category" placeholder="输入分类，例如一线/二线" />
            </n-form-item>
            <n-form-item label="入职日期">
              <n-input v-model:value="form.hire_date" placeholder="YYYY-MM-DD" />
            </n-form-item>
            <n-form-item label="状态">
              <n-select v-model:value="form.status" :options="statusOptions" />
            </n-form-item>
          </div>
        </n-form>

        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" :loading="saving" @click="submit">保存</n-button>
          </div>
        </template>
      </n-modal>

      <n-modal v-model:show="transferVisible" preset="card" title="员工调动" style="width: min(480px, calc(100vw - 32px))">
        <p style="margin-bottom: 16px; color: var(--admin-text-muted); font-size: 13px;">
          将「{{ transferTarget?.name }}」从「{{ transferTarget?.org_node_path_display || orgNodeName(transferTarget?.org_node_id || '') }}」调动至新的组织节点。调动后将<strong>清除原有应用角色和分组关系</strong>。
        </p>
        <n-form :model="transferForm" label-placement="left" label-width="88">
          <n-form-item label="目标节点">
            <n-select v-model:value="transferForm.target_org_node_id" filterable :options="departmentOptions" placeholder="选择目标科室" />
          </n-form-item>
          <n-form-item label="调动原因">
            <n-input v-model:value="transferForm.reason" type="textarea" placeholder="可选，填写调动原因" :rows="2" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="transferVisible = false">取消</n-button>
            <n-button type="warning" :loading="transferring" @click="submitTransfer">确认调动</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.table-shell {
  overflow-x: auto;
}

.admin-table {
  width: 100%;
  border-collapse: collapse;
}

.admin-table th,
.admin-table td {
  padding: 14px 12px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  text-align: left;
  vertical-align: top;
}

.admin-table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.table-muted {
  color: var(--admin-text-muted);
  font-size: 12px;
}

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.table-role-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.table-empty {
  padding: 48px 0;
  text-align: center;
  color: var(--admin-text-muted);
}

.form-grid.two-column {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

@media (max-width: 760px) {
  .form-grid.two-column {
    grid-template-columns: 1fr;
  }
}
</style>