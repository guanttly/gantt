<script setup lang="ts">
import { Connection, Delete, Edit, Plus, Refresh, Search, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { deleteSchedulingRule, getSchedulingRuleList, organizeRules, toggleSchedulingRuleStatus } from '@/api/scheduling-rule'
import EmptyState from '@/components/EmptyState.vue'
import PageContainer from '@/components/PageContainer.vue'
import TableSkeleton from '@/components/TableSkeleton.vue'
import RuleDependencyGraph from './components/RuleDependencyGraph.vue'
import RuleFormDialog from './components/RuleFormDialogV4.vue'
import RuleMigrationDialog from './components/RuleMigrationDialog.vue'
import RuleParseDialog from './components/RuleParseDialog.vue'
import RuleStatisticsCard from './components/RuleStatisticsCard.vue'
import {
  applyScopeOptions,
  categoryOptions,
  getApplyScopeText,
  getCategoryTagType,
  getCategoryText,
  getRuleTypeText,
  getSourceTypeTagType,
  getSourceTypeText,
  getSubCategoryText,
  getTimeScopeText,
  ruleTypeOptions,
  sourceTypeOptions,
  timeScopeOptions,
} from './logic'

// 组织ID
const orgId = ref('default-org')

// 查询参数
const queryParams = reactive({
  orgId: orgId.value,
  ruleType: undefined as SchedulingRule.RuleType | undefined,
  applyScope: undefined as SchedulingRule.ApplyScope | undefined,
  timeScope: undefined as SchedulingRule.TimeScope | undefined,
  isActive: undefined as boolean | undefined,
  keyword: '',
  // V4新增筛选字段
  category: undefined as SchedulingRule.Category | undefined,
  sourceType: undefined as 'manual' | 'llm_parsed' | 'migrated' | undefined,
  version: undefined as 'v3' | 'v4' | undefined,
  page: 1,
  size: 10,
})

// 表格数据
const tableData = ref<SchedulingRule.RuleInfo[]>([])
const total = ref(0)
const loading = ref(false)

// 对话框相关
const dialogVisible = ref(false)
const editingRuleId = ref<string>()

// V4 语义化规则解析对话框
const parseDialogVisible = ref(false)

// 规则组织结果对话框
const organizationDialogVisible = ref(false)
const organizationResult = ref<SchedulingRule.RuleOrganizationResult | null>(null)
const organizing = ref(false)

// V4 分类 Tab
const activeCategory = ref<string>('')
const categoryTabs = [
  { label: '全部', value: '' },
  { label: '约束规则', value: 'constraint' },
  { label: '偏好规则', value: 'preference' },
  { label: '依赖规则', value: 'dependency' },
  { label: '未分类V3', value: 'v3_unclassified' },
]

// 统计信息
const statistics = ref({
  total: 0,
  constraint: 0,
  preference: 0,
  dependency: 0,
  v3: 0,
  v4: 0,
  active: 0,
  inactive: 0,
})

// 迁移对话框
const migrationDialogVisible = ref(false)

// 活跃状态选项
const activeOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
]

// 获取规则列表
async function fetchRuleList() {
  loading.value = true
  try {
    const res = await getSchedulingRuleList(queryParams)
    tableData.value = res.items || []
    total.value = res.total || 0
    // 更新统计信息
    updateStatistics(res.items || [])
  }
  catch {
    ElMessage.error('获取规则列表失败')
  }
  finally {
    loading.value = false
  }
}

// 更新统计信息
function updateStatistics(items: SchedulingRule.RuleInfo[]) {
  statistics.value = {
    total: items.length,
    constraint: items.filter(r => (r as any).category === 'constraint').length,
    preference: items.filter(r => (r as any).category === 'preference').length,
    dependency: items.filter(r => (r as any).category === 'dependency').length,
    v3: items.filter(r => (r as any).version === 'v3' || !(r as any).version).length,
    v4: items.filter(r => (r as any).version === 'v4').length,
    active: items.filter(r => r.isActive).length,
    inactive: items.filter(r => !r.isActive).length,
  }
}

// 搜索
function handleSearch() {
  queryParams.page = 1
  fetchRuleList()
}

// 重置搜索
function handleReset() {
  queryParams.ruleType = undefined
  queryParams.applyScope = undefined
  queryParams.timeScope = undefined
  queryParams.isActive = undefined
  queryParams.keyword = ''
  queryParams.category = undefined
  queryParams.sourceType = undefined
  queryParams.version = undefined
  activeCategory.value = ''
  queryParams.page = 1
  fetchRuleList()
}

// 切换分类 Tab
function handleCategoryChange(category: string) {
  activeCategory.value = category
  if (category === 'v3_unclassified') {
    queryParams.category = undefined
    queryParams.version = 'v3'
  }
  else if (category === '') {
    // 全部：清空 category 和 version 筛选
    queryParams.category = undefined
    queryParams.version = undefined
  }
  else {
    queryParams.category = category as SchedulingRule.Category | undefined
    queryParams.version = undefined
  }
  queryParams.page = 1
  fetchRuleList()
}

// 打开迁移对话框
function handleOpenMigrationDialog() {
  migrationDialogVisible.value = true
}

// 新增规则
function handleAdd() {
  editingRuleId.value = undefined
  dialogVisible.value = true
}

// 编辑规则
function handleEdit(row: SchedulingRule.RuleInfo) {
  editingRuleId.value = row.id
  dialogVisible.value = true
}

// 对话框成功回调
function handleDialogSuccess() {
  fetchRuleList()
}

// 删除规则
async function handleDelete(row: SchedulingRule.RuleInfo) {
  try {
    await ElMessageBox.confirm('确认删除该规则吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteSchedulingRule(row.id, orgId.value)
    ElMessage.success('删除成功')
    fetchRuleList()
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 切换启用状态
async function handleToggleStatus(row: SchedulingRule.RuleInfo) {
  try {
    await toggleSchedulingRuleStatus(row.id, orgId.value, !row.isActive)
    ElMessage.success(`已${row.isActive ? '禁用' : '启用'}`)
    fetchRuleList()
  }
  catch {
    ElMessage.error('操作失败')
  }
}

// 打开语义化规则解析对话框
function handleOpenParseDialog() {
  parseDialogVisible.value = true
}

// 组织规则（V4）
async function handleOrganizeRules() {
  organizing.value = true
  try {
    const result = await organizeRules(orgId.value)
    organizationResult.value = result
    organizationDialogVisible.value = true
    ElMessage.success('规则组织成功')
  }
  catch {
    ElMessage.error('规则组织失败')
  }
  finally {
    organizing.value = false
  }
}

// 分页改变
function handlePageChange(page: number) {
  queryParams.page = page
  fetchRuleList()
}

function handleSizeChange(size: number) {
  queryParams.size = size
  queryParams.page = 1
  fetchRuleList()
}

onMounted(() => {
  fetchRuleList()
})
</script>

<template>
  <PageContainer title="排班规则">
    <!-- 工具栏:搜索和操作 -->
    <template #toolbar>
      <div class="toolbar-container">
        <!-- 搜索表单 -->
        <div class="search-form">
          <el-input
            v-model="queryParams.keyword"
            placeholder="规则名称"
            clearable
            class="search-input"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <el-select
            v-model="queryParams.ruleType"
            placeholder="规则类型"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in ruleTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

          <el-select
            v-model="queryParams.applyScope"
            placeholder="应用范围"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in applyScopeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

          <el-select
            v-model="queryParams.timeScope"
            placeholder="时间范围"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in timeScopeOptions"
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

          <el-select
            v-model="queryParams.category"
            placeholder="分类"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in categoryOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>

          <el-select
            v-model="queryParams.sourceType"
            placeholder="来源"
            clearable
            class="search-select"
          >
            <el-option
              v-for="item in sourceTypeOptions"
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
            新增规则
          </el-button>
          <el-button type="success" @click="handleOpenParseDialog">
            语义化规则（V4）
          </el-button>
          <el-button type="info" :icon="Connection" :loading="organizing" @click="handleOrganizeRules">
            组织规则（V4）
          </el-button>
          <el-button type="warning" @click="handleOpenMigrationDialog">
            规则迁移（V3→V4）
          </el-button>
        </div>
      </div>
    </template>

    <!-- 统计卡片 -->
    <RuleStatisticsCard
      :statistics="statistics"
      @refresh="fetchRuleList"
    />

    <!-- 分类 Tab -->
    <el-tabs
      v-model="activeCategory"
      style="margin-bottom: 20px"
      @tab-change="handleCategoryChange"
    >
      <el-tab-pane
        v-for="tab in categoryTabs"
        :key="tab.value || 'all'"
        :label="tab.label"
        :name="tab.value"
      />
    </el-tabs>

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
          <el-table-column prop="name" label="规则名称" width="200" />
          <el-table-column prop="category" label="分类" width="100" align="center">
            <template #default="{ row }">
              <el-tag
                v-if="(row as SchedulingRule.RuleInfoV4).category"
                :type="getCategoryTagType((row as SchedulingRule.RuleInfoV4).category)"
                effect="plain"
                size="small"
              >
                {{ getCategoryText((row as SchedulingRule.RuleInfoV4).category) }}
              </el-tag>
              <span v-else style="color: var(--el-text-color-placeholder);">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="subCategory" label="子分类" width="100" align="center">
            <template #default="{ row }">
              <el-tag
                v-if="(row as SchedulingRule.RuleInfo).subCategory"
                type="info"
                effect="plain"
                size="small"
              >
                {{ getSubCategoryText((row as SchedulingRule.RuleInfo).subCategory) }}
              </el-tag>
              <span v-else style="color: var(--el-text-color-placeholder);">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="sourceType" label="来源" width="100" align="center">
            <template #default="{ row }">
              <el-tag
                v-if="(row as SchedulingRule.RuleInfo).sourceType"
                :type="getSourceTypeTagType((row as SchedulingRule.RuleInfo).sourceType)"
                effect="plain"
                size="small"
              >
                {{ getSourceTypeText((row as SchedulingRule.RuleInfo).sourceType) }}
              </el-tag>
              <span v-else style="color: var(--el-text-color-placeholder);">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="version" label="版本" width="80" align="center">
            <template #default="{ row }">
              <el-tag
                v-if="(row as SchedulingRule.RuleInfo).version"
                :type="(row as SchedulingRule.RuleInfo).version === 'v4' ? 'success' : 'warning'"
                effect="plain"
                size="small"
              >
                {{ (row as SchedulingRule.RuleInfo).version?.toUpperCase() || 'V3' }}
              </el-tag>
              <span v-else style="color: var(--el-text-color-placeholder);">V3</span>
            </template>
          </el-table-column>
          <el-table-column prop="ruleType" label="规则类型" width="140" align="center">
            <template #default="{ row }">
              <el-tag type="primary" effect="plain">
                {{ getRuleTypeText(row.ruleType) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="applyScope" label="应用范围" width="110" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain">
                {{ getApplyScopeText(row.applyScope) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="timeScope" label="时间范围" width="110" align="center">
            <template #default="{ row }">
              <el-tag type="warning" effect="plain">
                {{ getTimeScopeText(row.timeScope) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="priority" label="优先级" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="row.priority > 5 ? 'danger' : 'success'" effect="light" class="priority-badge">
                P{{ row.priority }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="associationCount" label="关联数量" width="120" align="center">
            <template #default="{ row }">
              <div v-if="row.applyScope === 'global'" style="color: var(--el-text-color-secondary);">
                全局规则
              </div>
              <div v-else-if="row.associationCount > 0" style="display: flex; flex-direction: column; gap: 4px;">
                <div v-if="row.employeeCount > 0" style="font-size: 12px;">
                  <el-tag type="success" size="small" effect="plain">
                    员工 {{ row.employeeCount }}
                  </el-tag>
                </div>
                <div v-if="row.shiftCount > 0" style="font-size: 12px;">
                  <el-tag type="warning" size="small" effect="plain">
                    班次 {{ row.shiftCount }}
                  </el-tag>
                </div>
                <div v-if="row.groupCount > 0" style="font-size: 12px;">
                  <el-tag type="info" size="small" effect="plain">
                    分组 {{ row.groupCount }}
                  </el-tag>
                </div>
              </div>
              <div v-else style="color: var(--el-text-color-placeholder);">
                未关联
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="isActive" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" effect="light">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="ruleData" label="规则内容" min-width="200" show-overflow-tooltip />
          <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip />
          <el-table-column label="操作" width="200" fixed="right" align="center">
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
              :icon="Setting"
              title="暂无排班规则"
              description="点击下方按钮创建第一条规则"
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

    <!-- 规则表单对话框 -->
    <RuleFormDialog
      v-model:visible="dialogVisible"
      :rule-id="editingRuleId"
      :org-id="orgId"
      @success="handleDialogSuccess"
    />

    <!-- V4 语义化规则解析对话框 -->
    <RuleParseDialog
      v-model:visible="parseDialogVisible"
      :org-id="orgId"
      @success="handleDialogSuccess"
    />

    <!-- 规则组织结果对话框 -->
    <el-dialog
      v-model="organizationDialogVisible"
      title="规则组织结果（V4）"
      width="1000px"
      top="5vh"
    >
      <div v-if="organizationResult" class="organization-result">
        <el-tabs>
          <el-tab-pane label="约束规则" name="constraint">
            <el-table :data="organizationResult.constraintRules" stripe>
              <el-table-column prop="ruleName" label="规则名称" width="200" />
              <el-table-column prop="category" label="分类" width="100">
                <template #default="{ row }">
                  <el-tag :type="getCategoryTagType(row.category)" effect="plain">
                    {{ getCategoryText(row.category) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="subCategory" label="子分类" width="100" />
              <el-table-column prop="ruleType" label="类型" width="120" />
              <el-table-column prop="priority" label="优先级" width="80" align="center" />
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="偏好规则" name="preference">
            <el-table :data="organizationResult.preferenceRules" stripe>
              <el-table-column prop="ruleName" label="规则名称" width="200" />
              <el-table-column prop="category" label="分类" width="100">
                <template #default="{ row }">
                  <el-tag :type="getCategoryTagType(row.category)" effect="plain">
                    {{ getCategoryText(row.category) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="subCategory" label="子分类" width="100" />
              <el-table-column prop="ruleType" label="类型" width="120" />
              <el-table-column prop="priority" label="优先级" width="80" align="center" />
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="依赖规则" name="dependency">
            <el-table :data="organizationResult.dependencyRules" stripe>
              <el-table-column prop="ruleName" label="规则名称" width="200" />
              <el-table-column prop="category" label="分类" width="100">
                <template #default="{ row }">
                  <el-tag :type="getCategoryTagType(row.category)" effect="plain">
                    {{ getCategoryText(row.category) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="subCategory" label="子分类" width="100" />
              <el-table-column prop="ruleType" label="类型" width="120" />
              <el-table-column prop="priority" label="优先级" width="80" align="center" />
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="依赖关系" name="dependencies">
            <el-table :data="organizationResult.ruleDependencies" stripe>
              <el-table-column prop="dependentRuleID" label="被依赖规则" width="200" />
              <el-table-column prop="dependentOnRuleID" label="依赖规则" width="200" />
              <el-table-column prop="dependencyType" label="依赖类型" width="120" />
              <el-table-column prop="description" label="描述" min-width="300" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="冲突关系" name="conflicts">
            <el-table :data="organizationResult.ruleConflicts" stripe>
              <el-table-column prop="ruleID1" label="规则1" width="200" />
              <el-table-column prop="ruleID2" label="规则2" width="200" />
              <el-table-column prop="conflictType" label="冲突类型" width="120" />
              <el-table-column prop="description" label="描述" min-width="300" show-overflow-tooltip />
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="执行顺序" name="order">
            <div style="margin-bottom: 16px;">
              <h4>规则执行顺序：</h4>
              <el-tag
                v-for="(ruleId, index) in organizationResult.ruleExecutionOrder"
                :key="ruleId"
                style="margin: 4px;"
              >
                {{ index + 1 }}. {{ ruleId }}
              </el-tag>
            </div>
            <div>
              <h4>班次执行顺序：</h4>
              <el-tag
                v-for="(shiftId, index) in organizationResult.shiftExecutionOrder"
                :key="shiftId"
                style="margin: 4px;"
              >
                {{ index + 1 }}. {{ shiftId }}
              </el-tag>
            </div>
          </el-tab-pane>
          <el-tab-pane label="依赖关系图" name="graph">
            <RuleDependencyGraph
              :rule-dependencies="organizationResult.ruleDependencies"
              :rule-conflicts="organizationResult.ruleConflicts"
              :rules="[
                ...organizationResult.constraintRules,
                ...organizationResult.preferenceRules,
                ...organizationResult.dependencyRules,
              ]"
              :rule-execution-order="organizationResult.ruleExecutionOrder"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <!-- 规则迁移对话框 -->
    <RuleMigrationDialog
      v-model:visible="migrationDialogVisible"
      :org-id="orgId"
      @success="fetchRuleList"
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

.priority-badge {
  font-weight: 700;
  font-size: 12px;
  padding: 4px 10px;
}
</style>
