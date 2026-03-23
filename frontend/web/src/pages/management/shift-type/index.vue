<script setup lang="ts">
import { Delete, Edit, Plus, Refresh, Search, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  workflowPhase: undefined as string | undefined,
  isActive: true as boolean | undefined,
  page: 1,
  size: 20,
})

// 表格数据
const tableData = ref<any[]>([])
const total = ref(0)
const loading = ref(false)

// 工作流阶段选项
const phaseOptions = [
  { label: '固定班次', value: 'fixed' },
  { label: '特殊班次', value: 'special' },
  { label: '普通班次', value: 'normal' },
  { label: '科研班次', value: 'research' },
  { label: '填充班次', value: 'fill' },
]

// 活跃状态选项
const activeOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
]

// 获取班次类型列表
async function fetchShiftTypeList() {
  loading.value = true
  try {
    // TODO: 调用API获取班次类型列表
    // const res = await getShiftTypes(queryParams)
    // tableData.value = res.items || []
    // total.value = res.total || 0
    
    // 模拟数据
    tableData.value = [
      {
        id: '1',
        code: 'regular',
        name: '常规班次',
        schedulingPriority: 50,
        workflowPhase: 'normal',
        color: '#409EFF',
        isAiScheduling: true,
        isFixedSchedule: false,
        isOvertime: false,
        requiresSpecialSkill: false,
        isActive: true,
        isSystem: true,
      },
      {
        id: '2',
        code: 'overtime',
        name: '加班班次',
        schedulingPriority: 31,
        workflowPhase: 'special',
        color: '#F56C6C',
        isAiScheduling: true,
        isFixedSchedule: false,
        isOvertime: true,
        requiresSpecialSkill: false,
        isActive: true,
        isSystem: true,
      },
      {
        id: '3',
        code: 'standby',
        name: '备班班次',
        schedulingPriority: 32,
        workflowPhase: 'special',
        color: '#C71585',
        isAiScheduling: true,
        isFixedSchedule: false,
        isOvertime: false,
        requiresSpecialSkill: true,
        isActive: true,
        isSystem: true,
      },
    ]
    total.value = 3
  }
  catch (error) {
    console.error('获取班次类型列表失败', error)
    ElMessage.error('获取班次类型列表失败')
  }
  finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchShiftTypeList()
}

// 重置搜索
function handleReset() {
  queryParams.workflowPhase = undefined
  queryParams.isActive = undefined
  queryParams.page = 1
  fetchShiftTypeList()
}

// 新增班次类型
function handleAdd() {
  ElMessage.info('班次类型新增功能开发中...')
}

// 编辑班次类型
function handleEdit(row: any) {
  if (row.isSystem) {
    ElMessage.warning('系统内置类型不可编辑')
    return
  }
  ElMessage.info('班次类型编辑功能开发中...')
}

// 删除班次类型
async function handleDelete(row: any) {
  if (row.isSystem) {
    ElMessage.warning('系统内置类型不可删除')
    return
  }
  
  try {
    await ElMessageBox.confirm('确认删除该班次类型吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    // TODO: 调用删除API
    ElMessage.success('删除成功')
    fetchShiftTypeList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 切换启用状态
async function handleToggleStatus(row: any) {
  if (row.isSystem) {
    ElMessage.warning('系统内置类型不可禁用')
    return
  }
  
  try {
    // TODO: 调用API切换状态
    ElMessage.success(`已${row.isActive ? '禁用' : '启用'}`)
    fetchShiftTypeList()
  }
  catch (error) {
    ElMessage.error('操作失败')
  }
}

// 获取工作流阶段显示文本
function getPhaseText(phase: string): string {
  const map: Record<string, string> = {
    'fixed': '固定班次',
    'special': '特殊班次',
    'normal': '普通班次',
    'research': '科研班次',
    'fill': '填充班次',
  }
  return map[phase] || phase
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchShiftTypeList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchShiftTypeList()
}

onMounted(() => {
  fetchShiftTypeList()
})
</script>

<template>
  <PageContainer title="班次类型管理">
    <!-- 工具栏 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-select
            v-model="queryParams.workflowPhase"
            placeholder="工作流阶段"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in phaseOptions"
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
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            新增类型
          </el-button>
        </div>
      </div>
    </template>

    <!-- 表格内容 -->
    <el-card shadow="never">
      <!-- 加载骨架屏 -->
      <TableSkeleton v-if="loading && !tableData.length" :rows="10" :columns="8" />

      <!-- 数据表格 -->
      <template v-else>
        <el-table
          v-loading="loading"
          :data="tableData"
          stripe
        >
          <el-table-column prop="code" label="编码" min-width="120" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="schedulingPriority" label="优先级" width="90" align="center">
            <template #default="{ row }">
              <el-tag type="warning" effect="plain" size="small">
                {{ row.schedulingPriority }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="workflowPhase" label="工作流阶段" width="120" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="light" size="small">
                {{ getPhaseText(row.workflowPhase) }}
              </el-tag>
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
          <el-table-column prop="isAiScheduling" label="AI排班" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isAiScheduling ? 'success' : 'info'" effect="light" size="small">
                {{ row.isAiScheduling ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="isOvertime" label="算加班" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isOvertime ? 'warning' : 'info'" effect="light" size="small">
                {{ row.isOvertime ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="requiresSpecialSkill" label="需技能" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.requiresSpecialSkill ? 'warning' : 'info'" effect="light" size="small">
                {{ row.requiresSpecialSkill ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="isSystem" label="类型" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isSystem ? '' : 'success'" effect="light" size="small">
                {{ row.isSystem ? '系统' : '自定义' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="isActive" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" effect="light" size="small">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right" align="center">
            <template #default="{ row }">
              <el-button 
                type="primary" 
                link 
                :icon="Edit" 
                size="small" 
                :disabled="row.isSystem"
                @click="handleEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                :type="row.isActive ? 'warning' : 'success'"
                link
                size="small"
                :disabled="row.isSystem"
                @click="handleToggleStatus(row)"
              >
                {{ row.isActive ? '禁用' : '启用' }}
              </el-button>
              <el-button 
                type="danger" 
                link 
                :icon="Delete" 
                size="small" 
                :disabled="row.isSystem"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>

          <!-- 空状态 -->
          <template #empty>
            <EmptyState
              :icon="Setting"
              title="暂无班次类型数据"
              description="点击下方按钮添加第一个班次类型"
              button-text="新增类型"
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
  width: 40px;
  height: 24px;
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
</style>

