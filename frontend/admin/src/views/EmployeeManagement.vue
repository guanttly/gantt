<script setup lang="ts">
import type { PlatformEmployee, PlatformEmployeePayload } from '@/api/platform'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { getOrgTree } from '@/api/org'
import { createPlatformEmployee, deletePlatformEmployee, listPlatformEmployees, updatePlatformEmployee } from '@/api/platform'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const employees = ref<PlatformEmployee[]>([])
const orgTree = ref<any[]>([])
const keyword = ref('')
const dialogVisible = ref(false)
const editingEmployee = ref<PlatformEmployee | null>(null)
const message = useMessage()
const dialog = useDialog()

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

const filteredEmployees = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) {
    return employees.value
  }
  return employees.value.filter(item =>
    item.name.toLowerCase().includes(text)
    || item.employee_no?.toLowerCase().includes(text)
    || item.phone?.toLowerCase().includes(text)
    || item.email?.toLowerCase().includes(text)
    || item.position?.toLowerCase().includes(text),
  )
})

const statusOptions = [
  { label: '在职', value: 'active' },
  { label: '停用', value: 'inactive' },
]

const orgOptions = computed(() => flattenOrgTree(orgTree.value))

function normalizePayload(payload: PlatformEmployeePayload): PlatformEmployeePayload {
  return Object.fromEntries(Object.entries(payload).filter(([, value]) => value !== '' && value !== undefined)) as PlatformEmployeePayload
}

function flattenOrgTree(nodes: any[], level = 0): Array<{ label: string, value: string }> {
  return nodes.flatMap(node => [
    { label: `${'　'.repeat(level)}${node.name}`, value: node.id },
    ...flattenOrgTree(node.children || [], level + 1),
  ])
}

function orgNodeName(orgNodeId: string) {
  return orgOptions.value.find(item => item.value === orgNodeId)?.label.trim() || orgNodeId
}

function resetForm() {
  form.value = {
    org_node_id: auth.currentNode?.node_id || orgOptions.value[0]?.value || '',
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
    const [result, tree] = await Promise.all([
      listPlatformEmployees({ page: 1, size: 200 }),
      getOrgTree(),
    ])
    employees.value = result.data
    orgTree.value = tree
  }
  finally {
    loading.value = false
  }
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
    message.warning('请选择所属组织节点')
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

onMounted(loadEmployees)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">员工管理</h2>
          <p class="page-subtitle">维护机构内员工档案，并同步生成排班应用初始账号。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-input v-model:value="keyword" clearable placeholder="搜索姓名、工号、手机号、邮箱" style="width: 320px" />
        </div>
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增员工</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>所属节点</th>
                  <th>姓名</th>
                  <th>工号</th>
                  <th>职位</th>
                  <th>联系方式</th>
                  <th>状态</th>
                  <th>需改密</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in filteredEmployees" :key="item.id">
                  <td>{{ orgNodeName(item.org_node_id) }}</td>
                  <td>{{ item.name }}</td>
                  <td>{{ item.employee_no || '-' }}</td>
                  <td>{{ item.position || '-' }}</td>
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
                      <n-button text type="error" @click="removeEmployee(item)">删除</n-button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!filteredEmployees.length">
                  <td colspan="8" class="table-empty">暂无员工数据</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="editingEmployee ? '编辑员工' : '新增员工'" style="width: min(640px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="88">
          <div class="form-grid two-column">
            <n-form-item label="所属节点">
              <n-select v-model:value="form.org_node_id" filterable :options="orgOptions" placeholder="选择员工所属组织节点" />
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
  gap: 8px;
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

@media (max-width: 760px) {
  .form-grid.two-column {
    grid-template-columns: 1fr;
  }
}
</style>