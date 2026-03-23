<script setup lang="ts">
import { Delete, Edit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteEmployee, getEmployeeList } from '@/api/employee'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import EmployeeForm from './components/EmployeeForm.vue'
import GroupAssignment from './components/GroupAssignment.vue'
import { getStatusText, getStatusType, statusOptions } from './logic'

// 组织ID - 实际应用中应从全局状态获取
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  keyword: '',
  department: '',
  status: undefined as Employee.EmployeeStatus | undefined,
  page: 1,
  size: 10,
})

// 表格数据
const tableData = ref<Employee.EmployeeInfo[]>([])
const total = ref(0)
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentEmployee = ref<Employee.EmployeeInfo | null>(null)

// 获取员工列表
async function fetchEmployeeList() {
  loading.value = true
  try {
    // 员工管理页面需要显示分组信息，添加 includeGroups 参数
    const res = await getEmployeeList({
      ...queryParams,
      includeGroups: true,
    })
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch (error: any) {
    ElMessage.error('获取员工列表失败:', error)
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchEmployeeList()
}

// 重置搜索
function handleReset() {
  queryParams.keyword = ''
  queryParams.department = ''
  queryParams.status = undefined
  queryParams.page = 1
  fetchEmployeeList()
}

// 新增员工
function handleAdd() {
  formMode.value = 'create'
  currentEmployee.value = null
  formVisible.value = true
}

// 编辑员工
function handleEdit(row: Employee.EmployeeInfo) {
  formMode.value = 'edit'
  currentEmployee.value = row
  formVisible.value = true
}

// 删除员工
async function handleDelete(row: Employee.EmployeeInfo) {
  try {
    await ElMessageBox.confirm('确认删除该员工吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteEmployee(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchEmployeeList()
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
  fetchEmployeeList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchEmployeeList()
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchEmployeeList()
}

onMounted(() => {
  fetchEmployeeList()
})
</script>

<template>
  <PageContainer title="员工管理">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="搜索姓名、工号"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <el-input
            v-model="queryParams.department"
            placeholder="搜索科室"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          />

          <el-select
            v-model="queryParams.status"
            placeholder="状态"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
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
        <div class="toolbar-actions">
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增员工
          </el-button>
        </div>
      </div>
    </template>

    <!-- 内容区:数据表格 -->
    <div class="table-wrapper">
      <!-- 加载骨架屏 -->
      <TableSkeleton v-if="loading && !tableData.length" :rows="10" :columns="8" />

      <!-- 数据表格 -->
      <template v-else>
        <el-table
          v-loading="loading"
          :data="tableData"
          stripe
          class="modern-table"
        >
          <el-table-column prop="employeeId" label="工号" width="120" />
          <el-table-column prop="name" label="姓名" width="120" />
          <el-table-column prop="department" label="部门" min-width="80">
            <template #default="{ row }">
              {{ row.department?.name || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="groups" label="所属分组" min-width="280">
            <template #default="{ row }">
              <GroupAssignment
                :employee="row"
                :org-id="orgId"
                @success="fetchEmployeeList"
              />
            </template>
          </el-table-column>
          <el-table-column prop="position" label="职位" min-width="80" />
          <el-table-column prop="phone" label="电话" width="130" />
          <el-table-column prop="email" label="邮箱" width="200" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" effect="light" size="small">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="hireDate" label="入职日期" width="120" />
          <el-table-column label="操作" width="160" fixed="right" align="center">
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
              <el-button
                type="danger"
                link
                :icon="Delete"
                size="small"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>

          <!-- 空状态 -->
          <template #empty>
            <EmptyState
              title="暂无员工数据"
              description="点击右上角按钮添加第一个员工"
            />
          </template>
        </el-table>

        <!-- 分页 -->
        <div v-if="tableData.length" class="pagination-wrapper">
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
    </div>

    <!-- 表单对话框 -->
    <EmployeeForm
      v-model:visible="formVisible"
      :mode="formMode"
      :employee="currentEmployee"
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
  flex-wrap: wrap;
}

.search-form {
  display: flex;
  gap: 12px;
  flex: 1;
  flex-wrap: wrap;

  .search-input {
    width: 200px;
  }

  .search-select {
    width: 120px;
  }
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.modern-table {
  :deep(.el-table__header-wrapper) {
    .el-table__header thead th {
      background: #fafafa;
      color: #303133;
      font-weight: 600;
      font-size: 14px;
    }
  }

  :deep(.el-table__row) {
    transition: all 0.2s;

    &:hover {
      background: #f5f7fa;
    }
  }
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

// 响应式
@media (max-width: 1200px) {
  .toolbar-container {
    flex-direction: column;
    align-items: stretch;
  }

  .search-form {
    .search-input,
    .search-select {
      flex: 1;
      min-width: 150px;
    }
  }

  .toolbar-actions {
    justify-content: stretch;

    .el-button {
      flex: 1;
    }
  }
}
</style>
