<script setup lang="ts">
import { Delete, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref, watch } from 'vue'
import { getEmployeeList } from '@/api/employee'
import { batchAddGroupMembers, getGroupMembers, removeGroupMember } from '@/api/group'

interface Props {
  visible: boolean
  group: Group.GroupInfo | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const memberLoading = ref(false)

// 成员列表
const members = ref<Group.MemberInfo[]>([])
const total = ref(0)
const queryParams = reactive({
  page: 1,
  size: 20,
})

// 添加成员相关
const addMemberVisible = ref(false)
const employeeList = ref<Employee.EmployeeInfo[]>([])
const employeeTotal = ref(0)
const employeeLoading = ref(false)
const employeeQueryParams = reactive({
  page: 1,
  size: 20,
  keyword: '',
})
const selectedEmployeeIds = ref<string[]>([])

// 获取分组成员
async function fetchMembers() {
  if (!props.group)
    return

  memberLoading.value = true
  try {
    const res = await getGroupMembers({
      orgId: props.orgId,
      groupId: props.group.id,
      page: queryParams.page,
      size: queryParams.size,
    })
    members.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取成员列表失败')
  }
  finally {
    memberLoading.value = false
  }
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchMembers()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchMembers()
}

// 获取员工列表（用于添加成员）
async function fetchEmployeeList() {
  employeeLoading.value = true
  try {
    const res = await getEmployeeList({
      orgId: props.orgId,
      status: 'active',
      keyword: employeeQueryParams.keyword,
      page: employeeQueryParams.page,
      size: employeeQueryParams.size,
    })
    // 过滤掉已经是成员的员工
    const memberIds = new Set(members.value.map(m => m.id))
    employeeList.value = (res.items || []).filter(emp => !memberIds.has(emp.id))
    employeeTotal.value = res.total || 0
  }
  catch {
    ElMessage.error('获取员工列表失败')
  }
  finally {
    employeeLoading.value = false
  }
}

// 员工列表分页改变
function handleEmployeePageChange(page: number) {
  employeeQueryParams.page = page
  fetchEmployeeList()
}

function handleEmployeeSizeChange(size: number) {
  employeeQueryParams.size = size
  employeeQueryParams.page = 1
  fetchEmployeeList()
}

// 搜索员工
function handleEmployeeSearch() {
  employeeQueryParams.page = 1
  fetchEmployeeList()
}

// 移除成员
async function handleRemoveMember(member: Group.MemberInfo) {
  try {
    await ElMessageBox.confirm(`确认将 ${member.name} 从分组中移除吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await removeGroupMember({
      orgId: props.orgId,
      groupId: props.group!.id,
      employeeId: member.id,
    })
    ElMessage.success('移除成功')
    fetchMembers()
    emit('success')
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('移除失败')
    }
  }
}

// 显示添加成员对话框
function handleShowAddMember() {
  selectedEmployeeIds.value = []
  employeeQueryParams.keyword = ''
  employeeQueryParams.page = 1
  fetchEmployeeList()
  addMemberVisible.value = true
}

// 添加成员
async function handleAddMembers() {
  if (selectedEmployeeIds.value.length === 0) {
    ElMessage.warning('请选择要添加的员工')
    return
  }

  loading.value = true
  try {
    // 批量添加成员
    await batchAddGroupMembers({
      orgId: props.orgId,
      groupId: props.group!.id,
      employeeIds: selectedEmployeeIds.value,
    })
    ElMessage.success(`成功添加 ${selectedEmployeeIds.value.length} 名成员`)
    addMemberVisible.value = false
    // 重置分页到第一页
    queryParams.page = 1
    fetchMembers()
    emit('success')
  }
  catch {
    ElMessage.error('添加失败')
  }
  finally {
    loading.value = false
  }
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 监听对话框显示
watch(() => props.visible, (val) => {
  if (val && props.group) {
    queryParams.page = 1
    fetchMembers()
  }
})

onMounted(() => {
  if (props.visible && props.group) {
    fetchMembers()
  }
})
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="`管理成员 - ${group?.name}`"
    width="700px"
    @close="handleClose"
  >
    <div class="member-manage">
      <div class="toolbar">
        <div class="toolbar-info">
          <span>共 {{ total }} 名成员</span>
        </div>
        <el-button type="primary" :icon="Plus" @click="handleShowAddMember">
          添加成员
        </el-button>
      </div>

      <el-table
        v-loading="memberLoading"
        :data="members"
        stripe
        border
        max-height="450"
        style="width: 100%; margin-top: 16px"
      >
        <el-table-column prop="employeeId" label="工号" width="120" />
        <el-table-column prop="name" label="姓名" width="150" />
        <el-table-column prop="department" label="科室" width="150" />
        <el-table-column prop="position" label="职位" width="150" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="danger" link :icon="Delete" @click="handleRemoveMember(row)">
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div v-if="total > 0" class="pagination-container">
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
    </div>

    <!-- 添加成员对话框 -->
    <el-dialog
      v-model="addMemberVisible"
      title="选择要添加的员工"
      width="600px"
      append-to-body
    >
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-input
          v-model="employeeQueryParams.keyword"
          placeholder="搜索员工姓名或工号"
          clearable
          style="width: 300px"
          @clear="handleEmployeeSearch"
          @keyup.enter="handleEmployeeSearch"
        >
          <template #append>
            <el-button :icon="Search" @click="handleEmployeeSearch" />
          </template>
        </el-input>
      </div>

      <!-- 员工列表 -->
      <div v-loading="employeeLoading" class="employee-list">
        <el-checkbox-group v-model="selectedEmployeeIds">
          <div v-for="emp in employeeList" :key="emp.id" class="employee-item">
            <el-checkbox :label="emp.id">
              <span class="employee-name">{{ emp.name }}</span>
              <span class="employee-id">({{ emp.employeeId }})</span>
              <span v-if="emp.department" class="employee-dept">- {{ emp.department.name }}</span>
            </el-checkbox>
          </div>
        </el-checkbox-group>

        <!-- 空状态 -->
        <el-empty v-if="employeeList.length === 0 && !employeeLoading" description="暂无可添加的员工" :image-size="80" />
      </div>

      <!-- 分页 -->
      <div v-if="employeeTotal > 0" class="pagination-container">
        <el-pagination
          v-model:current-page="employeeQueryParams.page"
          v-model:page-size="employeeQueryParams.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="employeeTotal"
          :background="true"
          small
          layout="total, prev, pager, next"
          @size-change="handleEmployeeSizeChange"
          @current-change="handleEmployeePageChange"
        />
      </div>

      <template #footer>
        <div class="dialog-footer">
          <div class="selected-info">
            已选择 {{ selectedEmployeeIds.length }} 名员工
          </div>
          <div class="footer-buttons">
            <el-button @click="addMemberVisible = false">
              取消
            </el-button>
            <el-button type="primary" :loading="loading" @click="handleAddMembers">
              确定
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<style lang="scss" scoped>
.member-manage {
  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .toolbar-info {
      color: #606266;
      font-size: 14px;
    }
  }

  .pagination-container {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #f0f0f0;
  }
}

.search-bar {
  margin-bottom: 16px;
}

.employee-list {
  max-height: 400px;
  overflow-y: auto;
  min-height: 200px;
  padding: 8px 0;
}

.employee-item {
  padding: 10px 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;

  &:hover {
    background-color: #f5f7fa;
  }

  &:last-child {
    border-bottom: none;
  }

  .employee-name {
    font-weight: 500;
    color: #303133;
  }

  .employee-id {
    color: #909399;
    margin-left: 4px;
  }

  .employee-dept {
    color: #606266;
    margin-left: 8px;
  }
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .selected-info {
    color: #606266;
    font-size: 14px;
  }

  .footer-buttons {
    display: flex;
    gap: 8px;
  }
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}
</style>
