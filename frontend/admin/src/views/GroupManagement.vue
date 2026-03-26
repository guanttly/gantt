<script setup lang="ts">
import type { GroupDefaultAppRole, PlatformEmployee, PlatformGroup } from '@/api/platform'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { addPlatformGroupMember, assignGroupDefaultAppRole, createPlatformGroup, deletePlatformGroup, listGroupDefaultAppRoles, listPlatformEmployees, listPlatformGroupMembers, listPlatformGroups, removeGroupDefaultAppRole, removePlatformGroupMember, updatePlatformGroup } from '@/api/platform'

const loading = ref(false)
const saving = ref(false)
const groups = ref<PlatformGroup[]>([])
const employees = ref<PlatformEmployee[]>([])
const dialogVisible = ref(false)
const manageVisible = ref(false)
const editingGroup = ref<PlatformGroup | null>(null)
const currentGroup = ref<PlatformGroup | null>(null)
const members = ref<Array<{ employee_id: string, created_at: string }>>([])
const defaultRoles = ref<GroupDefaultAppRole[]>([])
const memberEmployeeId = ref<string | null>(null)
const defaultRole = ref<'app:schedule_admin' | 'app:scheduler' | 'app:leave_approver'>('app:scheduler')
const message = useMessage()
const dialog = useDialog()

const form = ref({ name: '', description: '' })
const employeeOptions = computed(() => employees.value.map(item => ({ label: `${item.name}${item.employee_no ? ` (${item.employee_no})` : ''}`, value: item.id })))
const appRoleOptions = [
  { label: '排班管理员', value: 'app:schedule_admin' },
  { label: '排班负责人', value: 'app:scheduler' },
  { label: '请假审批人', value: 'app:leave_approver' },
]

async function loadBaseData() {
  loading.value = true
  try {
    groups.value = await listPlatformGroups()
    employees.value = (await listPlatformEmployees({ page: 1, size: 200 })).data
  }
  finally {
    loading.value = false
  }
}

function employeeName(employeeId: string) {
  return employees.value.find(item => item.id === employeeId)?.name || employeeId
}

function openCreate() {
  editingGroup.value = null
  form.value = { name: '', description: '' }
  dialogVisible.value = true
}

function openEdit(group: PlatformGroup) {
  editingGroup.value = group
  form.value = { name: group.name, description: group.description || '' }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.name.trim()) {
    message.warning('请输入分组名称')
    return
  }

  saving.value = true
  try {
    if (editingGroup.value) {
      await updatePlatformGroup(editingGroup.value.id, { name: form.value.name, description: form.value.description || undefined })
      message.success('分组已更新')
    }
    else {
      await createPlatformGroup({ name: form.value.name, description: form.value.description || undefined })
      message.success('分组已创建')
    }
    dialogVisible.value = false
    await loadBaseData()
  }
  finally {
    saving.value = false
  }
}

function removeGroup(group: PlatformGroup) {
  dialog.warning({
    title: '确认删除分组',
    content: `确定删除「${group.name}」？这会同时移除成员关系和分组默认角色。`,
    positiveText: '删除',
    negativeText: '取消',
    async onPositiveClick() {
      await deletePlatformGroup(group.id)
      message.success('分组已删除')
      await loadBaseData()
    },
  })
}

async function openManage(group: PlatformGroup) {
  currentGroup.value = group
  manageVisible.value = true
  const [groupMembers, roles] = await Promise.all([
    listPlatformGroupMembers(group.id),
    listGroupDefaultAppRoles(group.id),
  ])
  members.value = groupMembers.map(item => ({ employee_id: item.employee_id, created_at: item.created_at }))
  defaultRoles.value = roles
}

async function addMember() {
  if (!currentGroup.value || !memberEmployeeId.value) {
    return
  }
  await addPlatformGroupMember(currentGroup.value.id, memberEmployeeId.value)
  memberEmployeeId.value = null
  await openManage(currentGroup.value)
  message.success('成员已加入分组')
}

async function removeMember(employeeId: string) {
  if (!currentGroup.value) {
    return
  }
  await removePlatformGroupMember(currentGroup.value.id, employeeId)
  await openManage(currentGroup.value)
  message.success('成员已移出分组')
}

async function addDefaultRole() {
  if (!currentGroup.value) {
    return
  }
  await assignGroupDefaultAppRole(currentGroup.value.id, { app_role: defaultRole.value, org_node_id: currentGroup.value.org_node_id })
  await openManage(currentGroup.value)
  message.success('分组默认应用角色已添加')
}

async function removeDefaultRole(roleId: string) {
  if (!currentGroup.value) {
    return
  }
  await removeGroupDefaultAppRole(currentGroup.value.id, roleId)
  await openManage(currentGroup.value)
  message.success('分组默认应用角色已移除')
}

onMounted(loadBaseData)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">分组管理</h2>
          <p class="page-subtitle">维护员工分组、成员关系和分组默认应用角色。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left" />
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增分组</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>分组名称</th>
                  <th>描述</th>
                  <th>组织节点</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in groups" :key="item.id">
                  <td>{{ item.name }}</td>
                  <td>{{ item.description || '-' }}</td>
                  <td>{{ item.org_node_id }}</td>
                  <td>
                    <div class="table-actions">
                      <n-button text type="primary" @click="openManage(item)">成员与角色</n-button>
                      <n-button text @click="openEdit(item)">编辑</n-button>
                      <n-button text type="error" @click="removeGroup(item)">删除</n-button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!groups.length">
                  <td colspan="4" class="table-empty">暂无分组</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="editingGroup ? '编辑分组' : '新增分组'" style="width: min(520px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="88">
          <n-form-item label="名称">
            <n-input v-model:value="form.name" placeholder="输入分组名称" />
          </n-form-item>
          <n-form-item label="描述">
            <n-input v-model:value="form.description" type="textarea" placeholder="输入描述（可选）" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" :loading="saving" @click="submit">保存</n-button>
          </div>
        </template>
      </n-modal>

      <n-modal v-model:show="manageVisible" preset="card" :title="currentGroup ? `管理分组：${currentGroup.name}` : '管理分组'" style="width: min(860px, calc(100vw - 32px))">
        <div class="manage-grid">
          <section>
            <h3 class="section-title">分组成员</h3>
            <div class="inline-actions">
              <n-select v-model:value="memberEmployeeId" filterable :options="employeeOptions" placeholder="选择员工" />
              <n-button type="primary" @click="addMember">添加成员</n-button>
            </div>
            <table class="admin-table compact">
              <thead>
                <tr>
                  <th>员工</th>
                  <th>加入时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="member in members" :key="member.employee_id">
                  <td>{{ employeeName(member.employee_id) }}</td>
                  <td>{{ member.created_at }}</td>
                  <td><n-button text type="error" @click="removeMember(member.employee_id)">移除</n-button></td>
                </tr>
                <tr v-if="!members.length">
                  <td colspan="3" class="table-empty">暂无成员</td>
                </tr>
              </tbody>
            </table>
          </section>

          <section>
            <h3 class="section-title">默认应用角色</h3>
            <div class="inline-actions">
              <n-select v-model:value="defaultRole" :options="appRoleOptions" />
              <n-button type="primary" @click="addDefaultRole">添加默认角色</n-button>
            </div>
            <table class="admin-table compact">
              <thead>
                <tr>
                  <th>角色</th>
                  <th>创建时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="role in defaultRoles" :key="role.id">
                  <td><n-tag size="small">{{ role.app_role }}</n-tag></td>
                  <td>{{ role.created_at }}</td>
                  <td><n-button text type="error" @click="removeDefaultRole(role.id)">移除</n-button></td>
                </tr>
                <tr v-if="!defaultRoles.length">
                  <td colspan="3" class="table-empty">暂无默认角色</td>
                </tr>
              </tbody>
            </table>
          </section>
        </div>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.admin-table {
  width: 100%;
  border-collapse: collapse;
}

.admin-table th,
.admin-table td {
  padding: 14px 12px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  text-align: left;
}

.admin-table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.admin-table.compact th,
.admin-table.compact td {
  padding: 10px 8px;
}

.table-actions,
.inline-actions {
  display: flex;
  gap: 8px;
}

.table-empty {
  padding: 36px 0;
  text-align: center;
  color: var(--admin-text-muted);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.manage-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.section-title {
  margin: 0 0 12px;
}

@media (max-width: 900px) {
  .manage-grid {
    grid-template-columns: 1fr;
  }
}
</style>