<script setup lang="ts">
import type { OrgNodeType, OrgTreeNode } from '@/api/org'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin, useDialog, useMessage } from 'naive-ui'
import OrgTreeBranch from '@/components/OrgTreeBranch.vue'
import { createOrgNode, deleteOrgNode, getOrgTree, isProtectedOrgNode, updateOrgNode } from '@/api/org'

const loading = ref(true)
const treeData = ref<OrgTreeNode[]>([])
const filterText = ref('')
const message = useMessage()
const dialog = useDialog()

const dialogVisible = ref(false)
const dialogTitle = ref('')
const editingId = ref<string | null>(null)
const form = ref({
  name: '',
  code: '',
  node_type: 'organization' as OrgNodeType,
  parent_id: undefined as string | undefined,
})

const nodeTypeOptions = [
  { label: '机构', value: 'organization' },
  { label: '院区', value: 'campus' },
  { label: '部门', value: 'department' },
  { label: '自定义', value: 'custom' },
]

const rootNodeTypeOptions = nodeTypeOptions.filter(option => option.value === 'organization')
const childNodeTypeOptions = nodeTypeOptions.filter(option => option.value !== 'organization')

const availableNodeTypeOptions = computed(() => {
  return form.value.parent_id ? childNodeTypeOptions : rootNodeTypeOptions
})

async function loadTree() {
  loading.value = true
  try {
    treeData.value = await getOrgTree()
  }
  catch {
    message.error('加载组织树失败')
  }
  finally {
    loading.value = false
  }
}

function handleAdd(parent?: OrgTreeNode) {
  editingId.value = null
  dialogTitle.value = parent ? `新建子组织 - ${parent.name}` : '新建根组织'
  form.value = { name: '', code: '', node_type: parent ? 'department' : 'organization', parent_id: parent?.id }
  dialogVisible.value = true
}

function handleEdit(node: OrgTreeNode) {
  editingId.value = node.id
  dialogTitle.value = '编辑组织'
  form.value = { name: node.name, code: node.code, node_type: node.node_type, parent_id: undefined }
  dialogVisible.value = true
}

async function handleDelete(node: OrgTreeNode) {
  if (isProtectedOrgNode(node)) {
    message.warning('平台管理根节点不允许删除')
    return
  }

  const confirmed = await new Promise<boolean>((resolve) => {
    dialog.warning({
      title: '确认删除',
      content: `确定删除「${node.name}」？仅允许删除无子节点的叶子节点。`,
      positiveText: '删除',
      negativeText: '取消',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
    })
  })

  if (!confirmed)
    return

  try {
    await deleteOrgNode(node.id)
    message.success('删除成功')
    loadTree()
  }
  catch (e: any) {
    message.error(e?.response?.data?.message || '删除失败')
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    message.warning('请输入名称')
    return
  }
  if (!editingId.value && !form.value.code.trim()) {
    message.warning('请输入组织编码')
    return
  }
  try {
    if (editingId.value) {
      await updateOrgNode(editingId.value, { name: form.value.name })
    }
    else {
      await createOrgNode(form.value)
    }
    message.success('操作成功')
    dialogVisible.value = false
    loadTree()
  }
  catch (e: any) {
    message.error(e?.response?.data?.message || '操作失败')
  }
}

onMounted(loadTree)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">机构管理</h2>
          <p class="page-subtitle">维护平台组织树结构，支持新增根节点、层级扩展和基础名称编辑。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-input v-model:value="filterText" clearable placeholder="搜索组织名称或编码" style="width: 260px" />
        </div>
        <div class="toolbar-right">
          <n-button type="primary" @click="handleAdd()">新建根组织</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <template #description>
              正在加载组织树
            </template>

            <div class="tree-wrapper">
              <OrgTreeBranch
                v-if="treeData.length"
                class="tree-root"
                :nodes="treeData"
                :filter-text="filterText"
                @add="handleAdd"
                @edit="handleEdit"
                @delete="handleDelete"
              />
              <div v-else class="tree-empty">暂无组织数据</div>
            </div>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="dialogTitle" style="width: min(420px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="80">
          <n-form-item label="名称">
            <n-input v-model:value="form.name" placeholder="输入组织名称" />
          </n-form-item>
          <n-form-item label="编码">
            <n-input v-model:value="form.code" :readonly="!!editingId" :disabled="!!editingId" placeholder="输入唯一编码，例如 dept-cardiology" />
          </n-form-item>
          <n-form-item label="类型">
            <n-select
              v-model:value="form.node_type"
              :options="availableNodeTypeOptions"
              :disabled="!!editingId"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" @click="handleSubmit">确定</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.tree-wrapper {
  min-height: 420px;
}

.tree-root {
  padding-left: 0;
}

.tree-empty {
  display: grid;
  min-height: 260px;
  place-items: center;
  color: var(--admin-text-muted);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
