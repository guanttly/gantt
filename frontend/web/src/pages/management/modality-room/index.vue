<script setup lang="ts">
import { DataAnalysis, Delete, Edit, OfficeBuilding, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteModalityRoom, getModalityRoomList, toggleModalityRoomStatus } from '@/api/modality-room'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import ModalityRoomForm from './components/ModalityRoomForm.vue'
import WeeklyVolumeDialog from './components/WeeklyVolumeDialog.vue'

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
const tableData = ref<ModalityRoom.Info[]>([])
const total = ref(0)
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentModalityRoom = ref<ModalityRoom.Info | null>(null)

// 周检查量配置弹框
const volumeDialogVisible = ref(false)
const volumeModalityRoom = ref<ModalityRoom.Info | null>(null)

// 活跃状态选项
const activeOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
]

// 获取机房列表
async function fetchModalityRoomList() {
  loading.value = true
  try {
    const res = await getModalityRoomList(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取机房列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchModalityRoomList()
}

// 重置搜索
function handleReset() {
  queryParams.isActive = undefined
  queryParams.keyword = ''
  queryParams.page = 1
  fetchModalityRoomList()
}

// 新增机房
function handleAdd() {
  formMode.value = 'create'
  currentModalityRoom.value = null
  formVisible.value = true
}

// 编辑机房
function handleEdit(row: ModalityRoom.Info) {
  formMode.value = 'edit'
  currentModalityRoom.value = row
  formVisible.value = true
}

// 删除机房
async function handleDelete(row: ModalityRoom.Info) {
  try {
    await ElMessageBox.confirm('确认删除该机房吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteModalityRoom(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchModalityRoomList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 切换启用状态
async function handleToggleStatus(row: ModalityRoom.Info) {
  try {
    await toggleModalityRoomStatus(row.id, orgId.value, !row.isActive)
    ElMessage.success(`已${row.isActive ? '禁用' : '启用'}`)
    fetchModalityRoomList()
  }
  catch {
    ElMessage.error('操作失败')
  }
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchModalityRoomList()
}

function handleSizeChange(size: number) {
  queryParams.pageSize = size
  queryParams.page = 1
  fetchModalityRoomList()
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchModalityRoomList()
}

// 打开检查量配置弹框
function handleVolumeConfig(row: ModalityRoom.Info) {
  volumeModalityRoom.value = row
  volumeDialogVisible.value = true
}

// 检查量配置保存成功
function handleVolumeSuccess() {
  volumeDialogVisible.value = false
}

onMounted(() => {
  fetchModalityRoomList()
})
</script>

<template>
  <PageContainer title="机房管理">
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
            新增机房
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
          <el-table-column prop="description" label="描述" show-overflow-tooltip />
          <el-table-column label="操作" width="300" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" size="small" @click="handleEdit(row)">
                编辑
              </el-button>
              <el-button type="primary" link :icon="DataAnalysis" size="small" @click="handleVolumeConfig(row)">
                检查量
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
              :icon="OfficeBuilding"
              title="暂无机房数据"
              description="点击下方按钮添加第一个机房"
              button-text="新增机房"
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
    <ModalityRoomForm
      v-model:visible="formVisible"
      :mode="formMode"
      :modality-room="currentModalityRoom"
      :org-id="orgId"
      @success="handleFormSuccess"
    />

    <!-- 周检查量配置弹框 -->
    <WeeklyVolumeDialog
      v-model:visible="volumeDialogVisible"
      :modality-room="volumeModalityRoom"
      :org-id="orgId"
      @success="handleVolumeSuccess"
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
