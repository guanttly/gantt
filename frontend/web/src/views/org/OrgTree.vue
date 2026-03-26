<script setup lang="ts">
import type { OrgTreeNode } from '@/api/org'
import { Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, ref } from 'vue'
import { createOrgNode, deleteOrgNode, getOrgTree, updateOrgNode } from '@/api/org'

const loading = ref(true)
const treeData = ref<OrgTreeNode[]>([])
const filterText = ref('')
const treeRef = ref()

// 弹窗
const dialogVisible = ref(false)
const dialogTitle = ref('新建节点')
const form = ref({
  name: '',
  type: 'department' as 'institution' | 'department' | 'team',
  parent_id: undefined as string | undefined,
})
const editingId = ref<string | null>(null)

async function loadTree() {
  loading.value = true
  try {
    treeData.value = await getOrgTree()
  }
  catch {
    ElMessage.error('加载组织树失败')
  }
  finally {
    loading.value = false
  }
}

function filterNode(value: string, data: OrgTreeNode) {
  if (!value)
    return true
  return data.name.includes(value)
}

function handleAdd(parent?: OrgTreeNode) {
  editingId.value = null
  dialogTitle.value = parent ? `在「${parent.name}」下新建节点` : '新建根节点'
  form.value = {
    name: '',
    type: 'department',
    parent_id: parent?.id,
  }
  dialogVisible.value = true
}

function handleEdit(node: OrgTreeNode) {
  editingId.value = node.id
  dialogTitle.value = '编辑节点'
  form.value = {
    name: node.name,
    type: node.type,
    parent_id: undefined,
  }
  dialogVisible.value = true
}

async function handleDelete(node: OrgTreeNode) {
  await ElMessageBox.confirm(`确定删除节点「${node.name}」吗？子节点也将被删除。`, '确认删除', { type: 'warning' })
  try {
    await deleteOrgNode(node.id)
    ElMessage.success('删除成功')
    loadTree()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入节点名称')
    return
  }
  try {
    if (editingId.value) {
      await updateOrgNode(editingId.value, { name: form.value.name })
      ElMessage.success('更新成功')
    }
    else {
      await createOrgNode(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadTree()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  }
}

onMounted(loadTree)
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <el-input v-model="filterText" placeholder="筛选节点" clearable style="width: 240px" :prefix-icon="Search" />
      <el-button type="primary" :icon="Plus" @click="handleAdd()">
        新建根节点
      </el-button>
    </div>

    <div v-loading="loading" class="tree-wrapper">
      <el-tree
        ref="treeRef"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        default-expand-all
        :filter-node-method="(filterNode as any)"
        :expand-on-click-node="false"
      >
        <template #default="{ data }">
          <div class="tree-node">
            <span class="node-label">{{ data.name }}</span>
            <span v-if="data.code" class="node-code">{{ data.code }}</span>
            <span class="node-actions">
              <el-button :icon="Plus" link size="small" @click.stop="handleAdd(data)" />
              <el-button :icon="Edit" link size="small" @click.stop="handleEdit(data)" />
              <el-button :icon="Delete" link type="danger" size="small" @click.stop="handleDelete(data)" />
            </span>
          </div>
        </template>
      </el-tree>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="420px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width: 100%">
            <el-option label="机构" value="institution" />
            <el-option label="部门" value="department" />
            <el-option label="团队" value="team" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" @click="handleSubmit">
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

.tree-wrapper {
  flex: 1;
  overflow-y: auto;
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e5e7eb;
}

.tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.node-label {
  font-weight: 500;
}

.node-code {
  font-size: 12px;
  color: #9ca3af;
}

.node-actions {
  margin-left: auto;
}
</style>
