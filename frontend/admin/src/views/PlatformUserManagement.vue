<script setup lang="ts">
import type { CreatePlatformUserPayload, PlatformUser } from '@/api/platform'
import type { OrgTreeNode } from '@/api/org'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin, NTag, useDialog, useMessage } from 'naive-ui'
import { getOrgTree } from '@/api/org'
import { createPlatformUser, disablePlatformUser, listPlatformUsers, resetPlatformUserPassword } from '@/api/platform'

const loading = ref(false)
const saving = ref(false)
const users = ref<PlatformUser[]>([])
const orgTree = ref<OrgTreeNode[]>([])
const selectedOrgNodeId = ref<string | null>(null)
const dialogVisible = ref(false)
const message = useMessage()
const dialog = useDialog()

const form = ref<CreatePlatformUserPayload>({
  username: '',
  email: '',
  phone: '',
  org_node_id: '',
  role_name: 'dept_admin',
})

const roleOptions = [
  { label: '机构管理员', value: 'org_admin' },
  { label: '科室管理员', value: 'dept_admin' },
]

const orgOptions = computed(() => flattenOrgTree(orgTree.value))

function flattenOrgTree(nodes: OrgTreeNode[], level = 0): Array<{ label: string, value: string }> {
  return nodes.flatMap(node => [
    { label: `${'　'.repeat(level)}${node.name}`, value: node.id },
    ...flattenOrgTree(node.children || [], level + 1),
  ])
}

async function loadUsers() {
  loading.value = true
  try {
    users.value = await listPlatformUsers(selectedOrgNodeId.value ? { org_node_id: selectedOrgNodeId.value } : undefined)
  }
  finally {
    loading.value = false
  }
}

async function loadOrgTree() {
  orgTree.value = await getOrgTree()
}

function openCreate() {
  form.value = {
    username: '',
    email: '',
    phone: '',
    org_node_id: selectedOrgNodeId.value || '',
    role_name: 'dept_admin',
  }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.username.trim() || !form.value.email.trim() || !form.value.org_node_id) {
    message.warning('请填写用户名、邮箱和所属节点')
    return
  }

  saving.value = true
  try {
    const created = await createPlatformUser({
      ...form.value,
      phone: form.value.phone || undefined,
    })
    dialogVisible.value = false
    await loadUsers()
    dialog.success({
      title: '平台账号已创建',
      content: `账号 ${created.user.username} 的默认密码为：${created.default_password}`,
      positiveText: '知道了',
    })
  }
  finally {
    saving.value = false
  }
}

function roleSummary(user: PlatformUser) {
  return user.roles.map(role => `${role.org_node_name} / ${role.role_name}`).join('、') || '-'
}

function resetPassword(user: PlatformUser) {
  dialog.warning({
    title: '重置平台账号密码',
    content: `确定重置 ${user.username} 的默认密码？`,
    positiveText: '重置',
    negativeText: '取消',
    async onPositiveClick() {
      const result = await resetPlatformUserPassword(user.id)
      dialog.success({
        title: '密码已重置',
        content: `新默认密码：${result.default_password}`,
        positiveText: '知道了',
      })
      await loadUsers()
    },
  })
}

function disableUser(user: PlatformUser) {
  dialog.warning({
    title: '禁用平台账号',
    content: `确定禁用 ${user.username}？`,
    positiveText: '禁用',
    negativeText: '取消',
    async onPositiveClick() {
      await disablePlatformUser(user.id)
      message.success('账号已禁用')
      await loadUsers()
    },
  })
}

onMounted(async () => {
  await loadOrgTree()
  await loadUsers()
})
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">平台账号</h2>
          <p class="page-subtitle">管理机构管理员与科室管理员账号，支持发放默认密码和强制改密。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-select v-model:value="selectedOrgNodeId" clearable filterable placeholder="按组织节点筛选" :options="orgOptions" style="width: 320px" @update:value="loadUsers" />
        </div>
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增平台账号</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>用户名</th>
                  <th>邮箱</th>
                  <th>绑定角色</th>
                  <th>状态</th>
                  <th>强制改密</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in users" :key="item.id">
                  <td>{{ item.username }}</td>
                  <td>{{ item.email }}</td>
                  <td>{{ roleSummary(item) }}</td>
                  <td>
                    <n-tag :type="item.status === 'active' ? 'success' : 'default'" size="small">
                      {{ item.status === 'active' ? '启用' : '禁用' }}
                    </n-tag>
                  </td>
                  <td>{{ item.must_reset_pwd ? '是' : '否' }}</td>
                  <td>
                    <div class="table-actions">
                      <n-button text type="primary" @click="resetPassword(item)">重置密码</n-button>
                      <n-button v-if="item.status === 'active'" text type="error" @click="disableUser(item)">禁用</n-button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!users.length">
                  <td colspan="6" class="table-empty">暂无平台账号</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" title="新增平台账号" style="width: min(520px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="96">
          <n-form-item label="用户名">
            <n-input v-model:value="form.username" placeholder="输入登录用户名" />
          </n-form-item>
          <n-form-item label="邮箱">
            <n-input v-model:value="form.email" placeholder="输入邮箱" />
          </n-form-item>
          <n-form-item label="手机号">
            <n-input v-model:value="form.phone" placeholder="输入手机号（可选）" />
          </n-form-item>
          <n-form-item label="所属节点">
            <n-select v-model:value="form.org_node_id" filterable :options="orgOptions" placeholder="选择组织节点" />
          </n-form-item>
          <n-form-item label="角色">
            <n-select v-model:value="form.role_name" :options="roleOptions" />
          </n-form-item>
        </n-form>

        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" :loading="saving" @click="submit">创建</n-button>
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
}

.admin-table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 700;
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>