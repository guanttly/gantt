<script setup lang="ts">
import type { ElTable } from 'element-plus'
import { Clock, Delete, Edit, InfoFilled, Link, Lock, Plus, Refresh, Search, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { deleteShift, getFixedAssignments, getShiftGroups, getShiftList, toggleShiftStatus } from '@/api/shift'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import FixedAssignmentDialog from './components/FixedAssignmentDialog.vue'
import ShiftForm from './components/ShiftForm.vue'
import ShiftGroupManage from './components/ShiftGroupManage.vue'
import WeeklyStaffDialog from './components/WeeklyStaffDialog.vue'
import { allTypeOptions, formatDuration, getTypeTagType, getTypeText } from './logic'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  type: undefined as string | undefined,
  isActive: true as boolean | undefined,
  keyword: '',
  page: 1,
  size: 10,
})

// 表格数据
const tableData = ref<Shift.ShiftInfo[]>([])
const total = ref(0)
const loading = ref(false)

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentShift = ref<Shift.ShiftInfo | null>(null)

// 分组管理相关
const groupManageVisible = ref(false)
const groupManageShift = ref<Shift.ShiftInfo | null>(null)

// 周人数配置对话框
const weeklyStaffVisible = ref(false)
const weeklyStaffShiftIds = ref<string[]>([])
const weeklyStaffShiftNames = ref<string[]>([])

// 固定人员配置对话框
const fixedDialogVisible = ref(false)
const fixedDialogShift = ref<Shift.ShiftInfo | null>(null)

// 表格多选
const tableRef = ref<InstanceType<typeof ElTable> | null>(null)
const selectedShifts = ref<Shift.ShiftInfo[]>([])

// 是否有选中
const hasSelection = computed(() => selectedShifts.value.length > 0)

// 活跃状态选项
const activeOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
]

// 获取班次列表
async function fetchShiftList() {
  loading.value = true
  try {
    const res = await getShiftList(queryParams)
    // request 函数默认返回 res.data，所以这里直接使用 res
    tableData.value = res.items || []
    total.value = res.total || 0

    // 批量获取每个班次的扩展信息
    await Promise.all([
      fetchShiftGroupsInfo(),
      fetchFixedAssignmentStatus(),
    ])
  }
  catch {
    ElMessage.error('获取班次列表失败')
  }
  finally {
    loading.value = false
  }
}

// 批量获取班次的分组信息
async function fetchShiftGroupsInfo() {
  if (tableData.value.length === 0)
    return

  // 批量查询所有班次的分组信息
  const promises = tableData.value.map(async (shift) => {
    try {
      const groups = await getShiftGroups(shift.id)
      // 扩展 shift 对象，添加分组信息
      ;(shift as any).groupCount = groups.length
      ;(shift as any).groupNames = groups.map(g => g.groupName || g.groupCode || g.groupId).join('、')
    }
    catch (error) {
      console.error(`Failed to get groups for shift ${shift.id}:`, error)
      ;(shift as any).groupCount = 0
      ;(shift as any).groupNames = ''
    }
  })

  await Promise.all(promises)
}

// 批量获取班次的固定人员配置状态
async function fetchFixedAssignmentStatus() {
  if (tableData.value.length === 0)
    return

  // 批量查询所有班次的固定人员配置状态
  const promises = tableData.value.map(async (shift) => {
    try {
      const assignments = await getFixedAssignments(shift.id)
      // 扩展 shift 对象，添加固定人员状态
      ;(shift as any).hasFixedAssignments = assignments && assignments.length > 0
      ;(shift as any).fixedAssignmentCount = assignments ? assignments.length : 0
    }
    catch (error) {
      console.error(`Failed to get fixed assignments for shift ${shift.id}:`, error)
      ;(shift as any).hasFixedAssignments = false
      ;(shift as any).fixedAssignmentCount = 0
    }
  })

  await Promise.all(promises)
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchShiftList()
}

// 重置搜索
function handleReset() {
  queryParams.type = undefined
  queryParams.isActive = undefined
  queryParams.keyword = ''
  queryParams.page = 1
  fetchShiftList()
}

// 新增班次
function handleAdd() {
  formMode.value = 'create'
  currentShift.value = null
  formVisible.value = true
}

// 编辑班次
function handleEdit(row: Shift.ShiftInfo) {
  formMode.value = 'edit'
  currentShift.value = row
  formVisible.value = true
}

// 删除班次
async function handleDelete(row: Shift.ShiftInfo) {
  try {
    await ElMessageBox.confirm('确认删除该班次吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteShift(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchShiftList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 切换启用状态
async function handleToggleStatus(row: Shift.ShiftInfo) {
  try {
    await toggleShiftStatus(row.id, orgId.value, !row.isActive)
    ElMessage.success(`已${row.isActive ? '禁用' : '启用'}`)
    fetchShiftList()
  }
  catch {
    ElMessage.error('操作失败')
  }
}

// 配置固定人员
function handleConfigFixed(row: Shift.ShiftInfo) {
  fixedDialogShift.value = row
  fixedDialogVisible.value = true
}

// 固定人员配置成功
function handleFixedAssignmentSuccess() {
  fixedDialogVisible.value = false
  fetchShiftList() // 刷新列表以更新固定人员状态
}

// 管理分组
function handleManageGroups(row: Shift.ShiftInfo) {
  groupManageShift.value = row
  groupManageVisible.value = true
}

// 分组管理成功回调
function handleGroupManageSuccess() {
  // 可以在这里刷新列表，显示更新后的分组信息
  fetchShiftList()
}

// 打开周人数配置对话框（单个）
function handleConfigWeeklyStaff(row: Shift.ShiftInfo) {
  weeklyStaffShiftIds.value = [row.id]
  weeklyStaffShiftNames.value = [row.name]
  weeklyStaffVisible.value = true
}

// 批量配置周人数
function handleBatchConfigWeeklyStaff() {
  if (selectedShifts.value.length === 0) {
    ElMessage.warning('请先选择班次')
    return
  }
  weeklyStaffShiftIds.value = selectedShifts.value.map(s => s.id)
  weeklyStaffShiftNames.value = selectedShifts.value.map(s => s.name)
  weeklyStaffVisible.value = true
}

// 周人数配置成功回调
function handleWeeklyStaffSuccess() {
  fetchShiftList()
  // 清空选择
  tableRef.value?.clearSelection()
  selectedShifts.value = []
}

// 表格多选变化
function handleSelectionChange(selection: Shift.ShiftInfo[]) {
  selectedShifts.value = selection
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchShiftList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchShiftList()
}

// 表单提交成功
function handleFormSuccess(result?: { shiftId: string, shiftName: string, mode: 'create' | 'edit' }) {
  formVisible.value = false
  fetchShiftList()

  // 如果是新建班次，自动弹出人数配置对话框
  if (result?.mode === 'create' && result.shiftId) {
    // 延迟一下确保列表刷新完成
    setTimeout(() => {
      weeklyStaffShiftIds.value = [result.shiftId]
      weeklyStaffShiftNames.value = [result.shiftName]
      weeklyStaffVisible.value = true
    }, 300)
  }
}

onMounted(() => {
  fetchShiftList()
})
</script>

<template>
  <PageContainer title="班次管理">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="班次名称、编码"
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
              v-for="item in allTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

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
          <el-button
            v-if="hasSelection"
            type="warning"
            :icon="Setting"
            @click="handleBatchConfigWeeklyStaff"
          >
            批量配置人数 ({{ selectedShifts.length }})
          </el-button>
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增班次
          </el-button>
        </div>
      </div>
    </template>

    <!-- 表格内容 -->
    <el-card shadow="never">
      <!-- 加载骨架屏 -->
      <TableSkeleton v-if="loading && !tableData.length" :rows="10" :columns="9" />

      <!-- 数据表格 -->
      <template v-else>
        <el-table
          ref="tableRef"
          v-loading="loading"
          :data="tableData"
          stripe
          style="width: 100%"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="50" />
          <el-table-column prop="code" label="编码" width="80" />
          <el-table-column prop="name" label="名称" width="150" />
          <el-table-column prop="type" label="类型" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)" effect="light">
                {{ getTypeText(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="startTime" label="开始时间" width="100" align="center" />
          <el-table-column prop="endTime" label="结束时间" width="100" align="center" />
          <el-table-column prop="duration" label="时长" width="120" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain">
                {{ formatDuration(row.duration) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="schedulingPriority" label="排班优先级" width="110" align="center">
            <template #default="{ row }">
              <el-tag type="warning" effect="plain">
                {{ row.schedulingPriority ?? 0 }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="hasFixedAssignments" label="固定人员" width="100" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.hasFixedAssignments" type="success" size="small" effect="plain">
                <el-icon style="margin-right: 4px"><Lock /></el-icon>
                {{ row.fixedAssignmentCount }}人
              </el-tag>
              <span v-else style="color: #909399">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="weeklyStaffSummary" label="周人数配置" width="180" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="row.weeklyStaffSummary && row.weeklyStaffSummary !== '未配置'"
                effect="dark"
                content="点击配置按钮可修改"
                placement="top"
              >
                <el-tag type="success" effect="plain">
                  {{ row.weeklyStaffSummary }}
                </el-tag>
              </el-tooltip>
              <span v-else style="color: #909399">未配置</span>
            </template>
          </el-table-column>
          <el-table-column prop="color" label="颜色" width="80" align="center">
            <template #default="{ row }">
              <div
                v-if="row.color"
                class="color-preview"
                :style="{ backgroundColor: row.color }"
                :title="row.color"
              />
            </template>
          </el-table-column>
          <el-table-column prop="isActive" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" effect="light">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="groups" label="关联分组" width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <div v-if="row.groupCount > 0" class="group-tags">
                <el-tag type="info" size="small" effect="plain">
                  {{ row.groupCount }} 个分组
                </el-tag>
                <el-tooltip v-if="row.groupNames" effect="dark" :content="row.groupNames" placement="top">
                  <el-icon style="margin-left: 4px; cursor: help; color: #909399">
                    <InfoFilled />
                  </el-icon>
                </el-tooltip>
              </div>
              <span v-else style="color: #909399">未关联</span>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" show-overflow-tooltip />
          <el-table-column label="操作" width="400" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" size="small" @click="handleEdit(row)">
                编辑
              </el-button>
              <el-button type="success" link :icon="Lock" size="small" @click="handleConfigFixed(row)">
                固定人员
              </el-button>
              <el-button type="primary" link :icon="Setting" size="small" @click="handleConfigWeeklyStaff(row)">
                人数
              </el-button>
              <el-button type="primary" link :icon="Link" size="small" @click="handleManageGroups(row)">
                分组
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
              :icon="Clock"
              title="暂无班次数据"
              description="点击下方按钮添加第一个班次"
              button-text="新增班次"
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
    <ShiftForm
      v-model:visible="formVisible"
      :mode="formMode"
      :shift="currentShift"
      :org-id="orgId"
      @success="handleFormSuccess"
    />

    <!-- 分组管理对话框 -->
    <ShiftGroupManage
      v-model:visible="groupManageVisible"
      :shift="groupManageShift"
      :org-id="orgId"
      @success="handleGroupManageSuccess"
    />

    <!-- 周人数配置对话框 -->
    <WeeklyStaffDialog
      v-model:visible="weeklyStaffVisible"
      :org-id="orgId"
      :shift-ids="weeklyStaffShiftIds"
      :shift-names="weeklyStaffShiftNames"
      @success="handleWeeklyStaffSuccess"
    />

    <!-- 固定人员配置对话框 -->
    <FixedAssignmentDialog
      v-model:visible="fixedDialogVisible"
      :shift="fixedDialogShift"
      @success="handleFixedAssignmentSuccess"
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

.color-preview {
  width: 32px;
  height: 20px;
  border-radius: 4px;
  display: inline-block;
  border: 1px solid #dcdfe6;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    transform: scale(1.1);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
}

.group-tags {
  display: flex;
  align-items: center;
  justify-content: flex-start;
}
</style>
