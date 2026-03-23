<script setup lang="ts">
import { DataAnalysis, Delete, Edit, Plus, Refresh, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { getShiftList } from '@/api/shift'
import { deleteStaffingRule, getStaffingRules } from '@/api/staffing'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import CalculateDialog from './components/CalculateDialog.vue'
import RuleForm from './components/RuleForm.vue'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  shiftId: undefined as string | undefined,
  page: 1,
  pageSize: 10,
})

// 表格数据
const tableData = ref<Staffing.Rule[]>([])
const total = ref(0)
const loading = ref(false)

// 班次列表
const shifts = ref<Shift.ShiftInfo[]>([])
const shiftsMap = ref<Map<string, Shift.ShiftInfo>>(new Map())

// 表单相关
const formVisible = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const currentRule = ref<Staffing.Rule | null>(null)

// 计算对话框
const calculateVisible = ref(false)

// 已有规则的班次ID列表
const existingShiftIds = computed(() => {
  return tableData.value.map(r => r.shiftId)
})

// 是否有规则
const hasRules = computed(() => tableData.value.length > 0)

// 加载班次列表
async function loadShifts() {
  try {
    const res = await getShiftList({
      orgId: orgId.value,
      isActive: true,
      page: 1,
      size: 100,
    })
    shifts.value = res.items || []
    shiftsMap.value = new Map(shifts.value.map(s => [s.id, s]))
  }
  catch {
    ElMessage.error('加载班次列表失败')
  }
}

// 获取班次名称
function getShiftName(shiftId: string): string {
  const shift = shiftsMap.value.get(shiftId)
  return shift ? `${shift.name} (${shift.startTime}-${shift.endTime})` : '-'
}

// 获取规则列表
async function fetchRuleList() {
  loading.value = true
  try {
    const res = await getStaffingRules(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
  }
  catch {
    ElMessage.error('获取规则列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchRuleList()
}

// 重置搜索
function handleReset() {
  queryParams.shiftId = undefined
  queryParams.page = 1
  fetchRuleList()
}

// 新增规则
function handleAdd() {
  formMode.value = 'create'
  currentRule.value = null
  formVisible.value = true
}

// 编辑规则
function handleEdit(row: Staffing.Rule) {
  formMode.value = 'edit'
  currentRule.value = row
  formVisible.value = true
}

// 删除规则
async function handleDelete(row: Staffing.Rule) {
  try {
    await ElMessageBox.confirm('确认删除该计算规则吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteStaffingRule(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchRuleList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 打开计算对话框
function handleCalculate() {
  calculateVisible.value = true
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchRuleList()
}

function handleSizeChange(size: number) {
  queryParams.pageSize = size
  queryParams.page = 1
  fetchRuleList()
}

// 表单提交成功
function handleFormSuccess() {
  formVisible.value = false
  fetchRuleList()
}

// 计算应用成功
function handleCalculateSuccess() {
  calculateVisible.value = false
  ElMessage.success('计算结果已应用到班次配置')
}

onMounted(async () => {
  await loadShifts()
  await fetchRuleList()
})
</script>

<template>
  <PageContainer title="排班人数计算">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-select
            v-model="queryParams.shiftId"
            placeholder="选择班次"
            clearable
            class="search-select"
          >
            <el-option
              v-for="shift in shifts"
              :key="shift.id"
              :label="shift.name"
              :value="shift.id"
            />
          </el-select>

          <el-button type="primary" :icon="Refresh" @click="handleSearch">
            刷新
          </el-button>
          <el-button :icon="Refresh" @click="handleReset">
            重置
          </el-button>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-button
            type="success"
            :icon="DataAnalysis"
            @click="handleCalculate"
          >
            人数计算
          </el-button>
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增规则
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
          <el-table-column prop="shiftId" label="班次" min-width="180">
            <template #default="{ row }">
              {{ row.shiftName || getShiftName(row.shiftId) }}
            </template>
          </el-table-column>
          <el-table-column prop="modalityRoomIds" label="关联机房" min-width="200">
            <template #default="{ row }">
              <template v-if="row.modalityRooms?.length">
                <el-tag
                  v-for="room in row.modalityRooms"
                  :key="room.id"
                  type="info"
                  effect="plain"
                  size="small"
                  class="room-tag"
                >
                  {{ room.name }}
                </el-tag>
              </template>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="timePeriodName" label="时间段" width="150">
            <template #default="{ row }">
              {{ row.timePeriodName || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="avgReportLimit" label="人均报告上限" width="130" align="center">
            <template #default="{ row }">
              <el-tag type="primary" effect="plain">
                {{ row.avgReportLimit }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="roundingMode" label="取整方式" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.roundingMode === 'ceil' ? 'success' : 'warning'" effect="plain">
                {{ row.roundingMode === 'ceil' ? '向上' : '向下' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="isActive" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" effect="light">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="Edit" size="small" @click="handleEdit(row)">
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
              :icon="Setting"
              title="暂无计算规则"
              description="点击下方按钮添加第一个计算规则"
              button-text="新增规则"
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

    <!-- 规则表单对话框 -->
    <RuleForm
      v-model:visible="formVisible"
      :mode="formMode"
      :rule="currentRule"
      :org-id="orgId"
      :existing-shift-ids="existingShiftIds"
      @success="handleFormSuccess"
    />

    <!-- 计算对话框 -->
    <CalculateDialog
      v-model:visible="calculateVisible"
      :org-id="orgId"
      :has-rules="hasRules"
      @success="handleCalculateSuccess"
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

.search-select {
  width: 200px;
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

.room-tag {
  margin-right: 4px;
  margin-bottom: 2px;
}

.text-muted {
  color: #909399;
}
</style>
