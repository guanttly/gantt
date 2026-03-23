<script setup lang="ts">
import { Calendar, Delete, Edit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteLeave, getLeaveList } from '@/api/leave'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import LeaveFormDialog from './components/LeaveFormDialog.vue'
import { getLeaveTypeTagType, getLeaveTypeText, leaveTypeOptions } from './logic'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  keyword: '', // 员工姓名或工号搜索
  type: undefined as Leave.LeaveType | undefined,
  startDate: '',
  endDate: '',
  page: 1,
  size: 10,
})

// 表格数据
const tableData = ref<Leave.LeaveInfo[]>([])
const total = ref(0)
const loading = ref(false)

// 对话框相关
const dialogVisible = ref(false)
const editingLeaveId = ref<string>()

// 获取假期列表
async function fetchLeaveList() {
  loading.value = true
  try {
    const res = await getLeaveList(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取假期列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchLeaveList()
}

// 重置搜索
function handleReset() {
  queryParams.keyword = ''
  queryParams.type = undefined
  queryParams.startDate = ''
  queryParams.endDate = ''
  queryParams.page = 1
  fetchLeaveList()
}

// 新增假期
function handleAdd() {
  editingLeaveId.value = undefined
  dialogVisible.value = true
}

// 编辑假期
function handleEdit(row: Leave.LeaveInfo) {
  editingLeaveId.value = row.id
  dialogVisible.value = true
}

// 对话框成功回调
function handleDialogSuccess() {
  fetchLeaveList()
}

// 删除假期
async function handleDelete(row: Leave.LeaveInfo) {
  try {
    await ElMessageBox.confirm('确认删除该假期记录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteLeave(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchLeaveList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchLeaveList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchLeaveList()
}

// 格式化日期时间显示
function formatDateTime(date: string, time?: string) {
  if (!date)
    return '-'

  // 处理 ISO 8601 格式 (2025-11-03T00:00:00+08:00)
  let dateStr = date
  if (date.includes('T')) {
    dateStr = date.substring(0, 10)
  }

  if (time) {
    return `${dateStr} ${time}`
  }
  return dateStr
}

// 格式化创建时间（显示相对时间）
function formatCreatedTime(dateTimeStr: string) {
  if (!dateTimeStr)
    return '-'

  const date = new Date(dateTimeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1)
    return '刚刚'
  if (minutes < 60)
    return `${minutes}分钟前`
  if (hours < 24)
    return `${hours}小时前`
  if (days < 7)
    return `${days}天前`

  // 超过7天显示完整日期
  return dateTimeStr.replace('T', ' ').substring(0, 16)
}

onMounted(() => {
  fetchLeaveList()
})
</script>

<template>
  <PageContainer title="请假管理">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="员工姓名或工号"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <el-select
            v-model="queryParams.type"
            placeholder="假期类型"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in leaveTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

          <el-date-picker
            v-model="queryParams.startDate"
            type="date"
            placeholder="开始日期"
            value-format="YYYY-MM-DD"
            class="search-date"
          />

          <el-date-picker
            v-model="queryParams.endDate"
            type="date"
            placeholder="结束日期"
            value-format="YYYY-MM-DD"
            class="search-date"
          />

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
            新增假期记录
          </el-button>
        </div>
      </div>
    </template>

    <!-- 表格内容 -->
    <el-card shadow="never">
      <!-- 加载骨架屏 -->
      <TableSkeleton v-if="loading && !tableData.length" :rows="10" :columns="7" />

      <!-- 数据表格 -->
      <template v-else>
        <el-table
          v-loading="loading"
          :data="tableData"
          stripe
          style="width: 100%"
        >
          <el-table-column prop="employeeName" label="员工姓名" width="120">
            <template #default="{ row }">
              {{ row.employeeName || row.employeeId }}
            </template>
          </el-table-column>
          <el-table-column prop="type" label="假期类型" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="getLeaveTypeTagType(row.type)" effect="plain">
                {{ getLeaveTypeText(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="开始时间" width="140" align="center">
            <template #default="{ row }">
              <div style="display: flex; flex-direction: column; gap: 2px;">
                <div style="font-size: 13px; color: var(--el-text-color-primary);">
                  {{ formatDateTime(row.startDate, row.startTime) }}
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="结束时间" width="140" align="center">
            <template #default="{ row }">
              <div style="display: flex; flex-direction: column; gap: 2px;">
                <div style="font-size: 13px; color: var(--el-text-color-primary);">
                  {{ formatDateTime(row.endDate, row.endTime) }}
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="days" label="天数" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain" class="days-badge">
                {{ row.days }} 天
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="请假事由" min-width="200" show-overflow-tooltip />
          <el-table-column label="创建时间" width="140" align="center">
            <template #default="{ row }">
              <div style="display: flex; flex-direction: column; gap: 2px;">
                <div style="font-size: 13px; color: var(--el-text-color-primary);">
                  {{ formatCreatedTime(row.createdAt) }}
                </div>
                <div style="font-size: 11px; color: var(--el-text-color-placeholder);">
                  {{ row.createdAt?.substring(0, 10) }}
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right" align="center">
            <template #default="{ row }">
              <el-button
                type="primary"
                link
                :icon="Edit"
                size="small"
                @click="handleEdit(row)"
              >
                编辑
              </el-button>
              <el-button type="danger" link :icon="Delete" size="small" @click="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>

          <!-- 空状态 -->
          <template #empty>
            <EmptyState
              :icon="Calendar"
              title="暂无假期记录"
              description="点击下方按钮创建第一条假期记录"
              button-text="新增假期记录"
              :show-button="true"
              @action="handleAdd"
            />
          </template>
        </el-table>

        <!-- 分页 -->
        <div v-if="tableData.length" class="pagination-container">
          <el-pagination
            v-model:current-page="queryParams.page"
            v-model:page-size="queryParams.size"
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
    <LeaveFormDialog
      v-model:visible="dialogVisible"
      :org-id="orgId"
      :leave-id="editingLeaveId"
      @success="handleDialogSuccess"
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
  flex-wrap: wrap;
}

.search-input {
  width: 200px;
}

.search-select {
  width: 160px;
}

.search-date {
  width: 180px;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.days-badge {
  font-weight: 500;
}

// 时间列样式优化
:deep(.el-table) {
  .el-table__cell {
    padding: 12px 0;
  }
}
</style>
