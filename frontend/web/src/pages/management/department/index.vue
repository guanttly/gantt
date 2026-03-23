<script setup lang="ts">
import type { DepartmentTree } from '@/api/department/model'
import { Delete, Edit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, ref } from 'vue'
import { deleteDepartment, getDepartmentTree } from '@/api/department'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import DepartmentForm from './components/DepartmentForm.vue'

// 组织ID - 实际应用中应从全局状态获取
const orgId = ref('default-org')

// 搜索关键字
const searchKeyword = ref('')

// 表格数据
const tableData = ref<DepartmentTree[]>([])
const filteredData = ref<DepartmentTree[]>([])
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentDepartment = ref<DepartmentTree | null>(null)
const parentDepartment = ref<DepartmentTree | null>(null)

// 获取部门树
async function fetchDepartmentTree() {
  loading.value = true
  try {
    const res = await getDepartmentTree(orgId.value)
    tableData.value = res || []
    filteredData.value = tableData.value
  }
  catch (error: any) {
    ElMessage.error(`获取部门列表失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
}

// 搜索过滤
function handleSearch() {
  if (!searchKeyword.value.trim()) {
    filteredData.value = tableData.value
    return
  }

  const keyword = searchKeyword.value.toLowerCase()
  filteredData.value = filterTree(tableData.value, keyword)
}

// 递归过滤树形数据
function filterTree(tree: DepartmentTree[], keyword: string): DepartmentTree[] {
  const result: DepartmentTree[] = []

  for (const node of tree) {
    const nameMatch = node.name.toLowerCase().includes(keyword)
    const codeMatch = node.code.toLowerCase().includes(keyword)

    if (nameMatch || codeMatch) {
      // 当前节点匹配，保留整个子树
      result.push({ ...node })
    }
    else if (node.children && node.children.length > 0) {
      // 当前节点不匹配，但检查子节点
      const filteredChildren = filterTree(node.children, keyword)
      if (filteredChildren.length > 0) {
        result.push({ ...node, children: filteredChildren })
      }
    }
  }

  return result
}

// 重置搜索
function handleReset() {
  searchKeyword.value = ''
  filteredData.value = tableData.value
}

// 新增部门
function handleAdd(parent?: DepartmentTree) {
  formMode.value = 'create'
  currentDepartment.value = null
  parentDepartment.value = parent || null
  formVisible.value = true
}

// 编辑部门
function handleEdit(row: DepartmentTree) {
  formMode.value = 'edit'
  currentDepartment.value = row
  parentDepartment.value = null
  formVisible.value = true
}

// 删除部门
async function handleDelete(row: DepartmentTree) {
  try {
    const hasChildren = row.children && row.children.length > 0
    const hasEmployees = row.employeeCount && row.employeeCount > 0

    let warningMsg = '确认删除该部门吗？'
    if (hasChildren) {
      warningMsg = '该部门下有子部门，无法删除！'
      ElMessage.warning(warningMsg)
      return
    }
    if (hasEmployees) {
      warningMsg = `该部门下有 ${row.employeeCount} 名员工，无法删除！`
      ElMessage.warning(warningMsg)
      return
    }

    await ElMessageBox.confirm(warningMsg, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await deleteDepartment(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchDepartmentTree()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchDepartmentTree()
}

// 刷新数据
function handleRefresh() {
  searchKeyword.value = ''
  fetchDepartmentTree()
}

onMounted(() => {
  fetchDepartmentTree()
})
</script>

<template>
  <PageContainer>
    <template #header>
      <div class="flex items-center justify-between">
        <h2 class="text-xl font-semibold">
          部门管理
        </h2>
        <div class="flex gap-2">
          <el-button :icon="Plus" type="primary" @click="handleAdd()">
            新增顶级部门
          </el-button>
          <el-button :icon="Refresh" @click="handleRefresh">
            刷新
          </el-button>
        </div>
      </div>
    </template>

    <!-- 搜索栏 -->
    <el-card class="mb-4">
      <el-form inline>
        <el-form-item label="关键字">
          <el-input
            v-model="searchKeyword"
            placeholder="部门名称或编码"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button :icon="Search" type="primary" @click="handleSearch">
            搜索
          </el-button>
          <el-button @click="handleReset">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 部门树形表格 -->
    <el-card>
      <TableSkeleton v-if="loading" />
      <EmptyState v-else-if="filteredData.length === 0" description="暂无部门数据" />
      <el-table
        v-else
        :data="filteredData"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        default-expand-all
        border
      >
        <el-table-column prop="name" label="部门名称" min-width="200" />
        <el-table-column prop="code" label="部门编码" width="150" />
        <el-table-column prop="level" label="层级" width="80" align="center" />
        <el-table-column prop="managerName" label="部门经理" width="120">
          <template #default="{ row }">
            {{ row.managerName || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="employeeCount" label="员工数" width="100" align="center">
          <template #default="{ row }">
            {{ row.employeeCount || 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="sortOrder" label="排序" width="80" align="center" />
        <el-table-column prop="isActive" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.isActive ? 'success' : 'info'">
              {{ row.isActive ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleAdd(row)">
              添加子部门
            </el-button>
            <el-button link type="primary" size="small" :icon="Edit" @click="handleEdit(row)">
              编辑
            </el-button>
            <el-button link type="danger" size="small" :icon="Delete" @click="handleDelete(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 表单对话框 -->
    <DepartmentForm
      v-model:visible="formVisible"
      :mode="formMode"
      :department="currentDepartment"
      :parent="parentDepartment"
      :org-id="orgId"
      @success="handleFormSuccess"
    />
  </PageContainer>
</template>

<style scoped>
</style>
