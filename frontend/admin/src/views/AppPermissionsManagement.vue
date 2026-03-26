<script setup lang="ts">
import type { AssignEmployeeAppRolePayload, PlatformEmployee, PlatformEmployeeAppRole } from '@/api/platform'
import type { OrgTreeNode } from '@/api/org'
import { computed, onMounted, ref, watch } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin, NTag, useMessage } from 'naive-ui'
import { getOrgTree } from '@/api/org'
import { assignEmployeeAppRole, batchAssignEmployeeAppRoles, listAppRoleSummary, listEmployeeAppRoles, listExpiringAppRoles, listPlatformEmployees, removeEmployeeAppRole } from '@/api/platform'

const loading = ref(false)
const employees = ref<PlatformEmployee[]>([])
const orgTree = ref<OrgTreeNode[]>([])
const summary = ref<Array<{ org_node_id: string, org_node_name: string, app_role: string, count: number }>>([])
const expiring = ref<Array<PlatformEmployeeAppRole & { employee_name?: string }>>([])
const selectedEmployeeId = ref<string | null>(null)
const employeeRoles = ref<PlatformEmployeeAppRole[]>([])
const assignVisible = ref(false)
const batchVisible = ref(false)
const message = useMessage()

const employeeOptions = computed(() => employees.value.map(item => ({ label: `${item.name}${item.employee_no ? ` (${item.employee_no})` : ''}`, value: item.id })))
const orgOptions = computed(() => flattenOrgTree(orgTree.value))
const appRoleOptions = [
  { label: '排班管理员', value: 'app:schedule_admin' },
  { label: '排班负责人', value: 'app:scheduler' },
  { label: '请假审批人', value: 'app:leave_approver' },
]

const assignForm = ref<AssignEmployeeAppRolePayload>({
  app_role: 'app:scheduler',
  org_node_id: '',
  expires_at: null,
})

const batchForm = ref<{ employee_ids: string[], app_role: 'app:schedule_admin' | 'app:scheduler' | 'app:leave_approver', org_node_id: string, expires_at: string | null }>({
  employee_ids: [],
  app_role: 'app:scheduler',
  org_node_id: '',
  expires_at: null,
})

function flattenOrgTree(nodes: OrgTreeNode[], level = 0): Array<{ label: string, value: string }> {
  return nodes.flatMap(node => [
    { label: `${'　'.repeat(level)}${node.name}`, value: node.id },
    ...flattenOrgTree(node.children || [], level + 1),
  ])
}

async function loadBaseData() {
  loading.value = true
  try {
    const [employeeResult, tree, summaryResult, expiringResult] = await Promise.all([
      listPlatformEmployees({ page: 1, size: 200 }),
      getOrgTree(),
      listAppRoleSummary(),
      listExpiringAppRoles(7),
    ])
    employees.value = employeeResult.data
    orgTree.value = tree
    summary.value = summaryResult
    expiring.value = expiringResult
  }
  finally {
    loading.value = false
  }
}

async function loadEmployeeRoles() {
  if (!selectedEmployeeId.value) {
    employeeRoles.value = []
    return
  }
  employeeRoles.value = await listEmployeeAppRoles(selectedEmployeeId.value)
}

watch(selectedEmployeeId, loadEmployeeRoles)

async function assignRole() {
  if (!selectedEmployeeId.value || !assignForm.value.org_node_id) {
    message.warning('请选择员工和生效节点')
    return
  }
  await assignEmployeeAppRole(selectedEmployeeId.value, assignForm.value)
  assignVisible.value = false
  message.success('应用角色已分配')
  await Promise.all([loadEmployeeRoles(), loadBaseData()])
}

async function batchAssign() {
  if (!batchForm.value.employee_ids.length || !batchForm.value.org_node_id) {
    message.warning('请选择员工和生效节点')
    return
  }
  const result = await batchAssignEmployeeAppRoles(batchForm.value)
  batchVisible.value = false
  message.success(`批量授权完成，新增 ${result.created.length} 条，跳过 ${result.skipped_employee_ids.length} 条`)
  await Promise.all([loadEmployeeRoles(), loadBaseData()])
}

async function removeRole(role: PlatformEmployeeAppRole) {
  if (!selectedEmployeeId.value) {
    return
  }
  await removeEmployeeAppRole(selectedEmployeeId.value, role.id)
  message.success('应用角色已移除')
  await Promise.all([loadEmployeeRoles(), loadBaseData()])
}

function employeeName(employeeId: string) {
  return employees.value.find(item => item.id === employeeId)?.name || employeeId
}

onMounted(loadBaseData)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">应用权限</h2>
          <p class="page-subtitle">查看各节点应用角色分布，管理员工的排班应用权限和即将到期授权。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-select v-model:value="selectedEmployeeId" filterable clearable :options="employeeOptions" placeholder="选择员工查看角色" style="width: 320px" />
        </div>
        <div class="toolbar-right">
          <n-button @click="batchVisible = true">批量授权</n-button>
          <n-button type="primary" :disabled="!selectedEmployeeId" @click="assignVisible = true">为当前员工授权</n-button>
        </div>
      </section>

      <div class="permissions-grid">
        <section class="page-card">
          <div class="page-card-inner">
            <h3 class="section-title">角色汇总</h3>
            <n-spin :show="loading">
              <table class="admin-table compact">
                <thead>
                  <tr>
                    <th>节点</th>
                    <th>角色</th>
                    <th>人数</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in summary" :key="`${item.org_node_id}-${item.app_role}`">
                    <td>{{ item.org_node_name }}</td>
                    <td><n-tag size="small">{{ item.app_role }}</n-tag></td>
                    <td>{{ item.count }}</td>
                  </tr>
                  <tr v-if="!summary.length">
                    <td colspan="3" class="table-empty">暂无汇总数据</td>
                  </tr>
                </tbody>
              </table>
            </n-spin>
          </div>
        </section>

        <section class="page-card">
          <div class="page-card-inner">
            <h3 class="section-title">7 天内即将过期</h3>
            <n-spin :show="loading">
              <table class="admin-table compact">
                <thead>
                  <tr>
                    <th>员工</th>
                    <th>角色</th>
                    <th>节点</th>
                    <th>过期时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in expiring" :key="item.id">
                    <td>{{ item.employee_name || employeeName(item.employee_id) }}</td>
                    <td><n-tag size="small">{{ item.app_role }}</n-tag></td>
                    <td>{{ item.org_node_name }}</td>
                    <td>{{ item.expires_at || '-' }}</td>
                  </tr>
                  <tr v-if="!expiring.length">
                    <td colspan="4" class="table-empty">暂无即将过期授权</td>
                  </tr>
                </tbody>
              </table>
            </n-spin>
          </div>
        </section>
      </div>

      <section class="page-card">
        <div class="page-card-inner">
          <h3 class="section-title">员工角色详情</h3>
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>角色</th>
                  <th>来源</th>
                  <th>节点</th>
                  <th>过期时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in employeeRoles" :key="item.id">
                  <td><n-tag size="small">{{ item.app_role }}</n-tag></td>
                  <td>{{ item.source_group_name || item.source }}</td>
                  <td>{{ item.org_node_name }}</td>
                  <td>{{ item.expires_at || '-' }}</td>
                  <td>
                    <n-button v-if="item.source === 'manual'" text type="error" @click="removeRole(item)">移除</n-button>
                    <span v-else class="table-muted">由 {{ item.source_group_name || item.source }} 授予</span>
                  </td>
                </tr>
                <tr v-if="selectedEmployeeId && !employeeRoles.length">
                  <td colspan="5" class="table-empty">当前员工暂无额外应用角色</td>
                </tr>
                <tr v-if="!selectedEmployeeId">
                  <td colspan="5" class="table-empty">请选择员工查看角色详情</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="assignVisible" preset="card" title="为员工分配应用角色" style="width: min(520px, calc(100vw - 32px))">
        <n-form :model="assignForm" label-placement="left" label-width="88">
          <n-form-item label="角色">
            <n-select v-model:value="assignForm.app_role" :options="appRoleOptions" />
          </n-form-item>
          <n-form-item label="生效节点">
            <n-select v-model:value="assignForm.org_node_id" filterable :options="orgOptions" />
          </n-form-item>
          <n-form-item label="过期时间">
            <n-input v-model:value="assignForm.expires_at" placeholder="可选，ISO 时间串" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="assignVisible = false">取消</n-button>
            <n-button type="primary" @click="assignRole">确认授权</n-button>
          </div>
        </template>
      </n-modal>

      <n-modal v-model:show="batchVisible" preset="card" title="批量分配应用角色" style="width: min(640px, calc(100vw - 32px))">
        <n-form :model="batchForm" label-placement="left" label-width="88">
          <n-form-item label="员工">
            <n-select v-model:value="batchForm.employee_ids" multiple filterable :options="employeeOptions" placeholder="选择多个员工" />
          </n-form-item>
          <n-form-item label="角色">
            <n-select v-model:value="batchForm.app_role" :options="appRoleOptions" />
          </n-form-item>
          <n-form-item label="生效节点">
            <n-select v-model:value="batchForm.org_node_id" filterable :options="orgOptions" />
          </n-form-item>
          <n-form-item label="过期时间">
            <n-input v-model:value="batchForm.expires_at" placeholder="可选，ISO 时间串" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="batchVisible = false">取消</n-button>
            <n-button type="primary" @click="batchAssign">批量授权</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.permissions-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.section-title {
  margin: 0 0 12px;
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

.table-empty {
  padding: 36px 0;
  text-align: center;
  color: var(--admin-text-muted);
}

.table-muted {
  color: var(--admin-text-muted);
  font-size: 12px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 960px) {
  .permissions-grid {
    grid-template-columns: 1fr;
  }
}
</style>