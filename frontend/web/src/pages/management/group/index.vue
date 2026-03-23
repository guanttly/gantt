<script setup lang="ts">
import { Delete, Edit, Plus, Refresh, Search, User, UserFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteGroup, getGroupList } from '@/api/group'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import GroupForm from './components/GroupForm.vue'
import MemberManage from './components/MemberManage.vue'
import { getTypeTagType, getTypeText, typeOptions } from './logic'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  type: undefined as Group.GroupType | undefined,
  parentId: '',
  keyword: '',
  page: 1,
  size: 10,
})

// 表格数据
const tableData = ref<Group.GroupInfo[]>([])
const total = ref(0)
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentGroup = ref<Group.GroupInfo | null>(null)

// 成员管理相关
const memberVisible = ref(false)
const currentGroupForMember = ref<Group.GroupInfo | null>(null)

// 获取分组列表
async function fetchGroupList() {
  loading.value = true
  try {
    const res = await getGroupList(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取分组列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchGroupList()
}

// 重置搜索
function handleReset() {
  queryParams.type = undefined
  queryParams.parentId = ''
  queryParams.keyword = ''
  queryParams.page = 1
  fetchGroupList()
}

// 新增分组
function handleAdd() {
  formMode.value = 'create'
  currentGroup.value = null
  formVisible.value = true
}

// 编辑分组
function handleEdit(row: Group.GroupInfo) {
  formMode.value = 'edit'
  currentGroup.value = row
  formVisible.value = true
}

// 删除分组
async function handleDelete(row: Group.GroupInfo) {
  try {
    await ElMessageBox.confirm('确认删除该分组吗？删除后该分组下的成员将不再归属此分组。', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteGroup(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchGroupList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 管理成员
function handleManageMembers(row: Group.GroupInfo) {
  currentGroupForMember.value = row
  memberVisible.value = true
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchGroupList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchGroupList()
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchGroupList()
}

// 成员管理成功
function handleMemberSuccess() {
  memberVisible.value = false
  fetchGroupList()
}

onMounted(() => {
  fetchGroupList()
})
</script>

<template>
  <PageContainer title="分组管理">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="分组名称"
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
            placeholder="类型"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in typeOptions"
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
        <div class="action-buttons">
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增分组
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
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="分组名称" width="200" />
          <el-table-column prop="type" label="类型" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)" effect="light">
                {{ getTypeText(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="parentId" label="父分组ID" width="180" show-overflow-tooltip />
          <el-table-column prop="memberCount" label="成员数" width="100" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain" class="member-badge">
                <el-icon class="member-icon">
                  <UserFilled />
                </el-icon>
                {{ row.memberCount || 0 }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
          <el-table-column label="操作" width="250" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" size="small" @click="handleEdit(row)">
                编辑
              </el-button>
              <el-button type="success" link :icon="User" size="small" @click="handleManageMembers(row)">
                成员
              </el-button>
              <el-button type="danger" link :icon="Delete" size="small" @click="handleDelete(row)">
                删除
              </el-button>
            </template>
          </el-table-column>

          <!-- 空状态 -->
          <template #empty>
            <EmptyState
              :icon="UserFilled"
              title="暂无分组数据"
              description="点击下方按钮创建第一个分组"
              button-text="新增分组"
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
    <GroupForm
      v-model:visible="formVisible"
      :mode="formMode"
      :group="currentGroup"
      :org-id="orgId"
      @success="handleFormSuccess"
    />

    <!-- 成员管理对话框 -->
    <MemberManage
      v-model:visible="memberVisible"
      :group="currentGroupForMember"
      :org-id="orgId"
      @success="handleMemberSuccess"
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
  width: 180px;
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

.member-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-weight: 600;

  .member-icon {
    font-size: 14px;
  }
}
</style>
