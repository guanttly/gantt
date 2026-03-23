<script setup lang="ts">
import { Delete, Edit, List, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteScanType, getScanTypeList, toggleScanTypeStatus } from '@/api/scan-type'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import ScanTypeForm from './components/ScanTypeForm.vue'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  isActive: undefined as boolean | undefined,
  keyword: '',
  page: 1,
  pageSize: 10,
})

// 表格数据
const tableData = ref<ScanType.Info[]>([])
const total = ref(0)
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentScanType = ref<ScanType.Info | null>(null)

// 活跃状态选项
const activeOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
]

// 获取检查类型列表
async function fetchScanTypeList() {
  loading.value = true
  try {
    const res = await getScanTypeList(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取检查类型列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchScanTypeList()
}

// 重置搜索
function handleReset() {
  queryParams.isActive = undefined
  queryParams.keyword = ''
  queryParams.page = 1
  fetchScanTypeList()
}

// 新增检查类型
function handleAdd() {
  formMode.value = 'create'
  currentScanType.value = null
  formVisible.value = true
}

// 编辑检查类型
function handleEdit(row: ScanType.Info) {
  formMode.value = 'edit'
  currentScanType.value = row
  formVisible.value = true
}

// 删除检查类型
async function handleDelete(row: ScanType.Info) {
  try {
    await ElMessageBox.confirm('确认删除该检查类型吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteScanType(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchScanTypeList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 切换启用状态
async function handleToggleStatus(row: ScanType.Info) {
  try {
    await toggleScanTypeStatus(row.id, orgId.value, !row.isActive)
    ElMessage.success(`已${row.isActive ? '禁用' : '启用'}`)
    fetchScanTypeList()
  }
  catch {
    ElMessage.error('操作失败')
  }
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchScanTypeList()
}

function handleSizeChange(size: number) {
  queryParams.pageSize = size
  queryParams.page = 1
  fetchScanTypeList()
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchScanTypeList()
}

onMounted(() => {
  fetchScanTypeList()
})
</script>

<template>
  <PageContainer title="检查类型管理">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="名称、编码"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <el-select
            v-model="queryParams.isActive"
            placeholder="状态"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in activeOptions"
              :key="String(item.value)"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

          <el-button type="primary" :icon="Search" @click="handleSearch">
            搜索
          </el-button>
          <el-button :icon="Refresh" @click="handleReset">
            重置
          </el-button>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增检查类型
          </el-button>
        </div>
      </div>
    </template>

    <!-- 表格内容 -->
    <el-card shadow="never">
      <!-- 加载骨架屏 -->
      <TableSkeleton v-if="loading && !tableData.length" :rows="10" :columns="6" />

      <!-- 数据表格 -->
      <template v-else>
        <el-table
          v-loading="loading"
          :data="tableData"
          stripe
          style="width: 100%"
        >
          <el-table-column prop="code" label="编码" width="150" />
          <el-table-column prop="name" label="名称" width="200" />
          <el-table-column prop="isActive" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" effect="light">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sortOrder" label="排序" width="80" align="center" />
          <el-table-column prop="description" label="描述" show-overflow-tooltip />
          <el-table-column label="操作" width="220" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" size="small" @click="handleEdit(row)">
                编辑
              </el-button>
              <el-button
                :type="row.isActive ? 'warning' : 'success'"
                link
                size="small"
                @click="handleToggleStatus(row)"
              >
                {{ row.isActive ? '禁用' : '启用' }}
              </el-button>
              <el-button type="danger" link :icon="Delete" size="small" @click="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>

          <!-- 空状态 -->
          <template #empty>
            <EmptyState
              :icon="List"
              title="暂无检查类型数据"
              description="点击下方按钮添加第一个检查类型"
              button-text="新增检查类型"
              :show-button="true"
              @action="handleAdd"
            />
          </template>
        </el-table>

        <!-- 分页 -->
        <div v-if="tableData.length" class="pagination-container">
          <el-pagination
            v-model:current-page="queryParams.page"
            v-model:page-size="queryParams.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            :background="true"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </template>
    </el-card>

    <!-- 表单对话框 -->
    <ScanTypeForm
      v-model:visible="formVisible"
      :mode="formMode"
      :scan-type="currentScanType"
      :org-id="orgId"
      @success="handleFormSuccess"
    />
  </PageContainer>
</template>

<style lang="scss" scoped>
.toolbar-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-form {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 240px;
}

.search-select {
  width: 140px;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}
</style>
