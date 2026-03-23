<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { getSimpleEmployeeList } from '@/api/employee'
import { getGroupList } from '@/api/group'
import { batchCreateRuleAssociation, deleteRuleAssociation, getRuleAssociations } from '@/api/scheduling-rule'
import { getShiftList } from '@/api/shift'

interface Props {
  visible: boolean
  ruleId?: string
  ruleName?: string
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

// 数据
const loading = ref(false)
const associations = ref<SchedulingRule.AssociationInfo[]>([])
const activeTab = ref<'employee' | 'shift' | 'group'>('employee')

// 选择器数据
const employeeList = ref<Employee.SimpleEmployeeInfo[]>([])
const shiftList = ref<Shift.ShiftInfo[]>([])
const groupList = ref<Group.GroupInfo[]>([])
const employeeLoading = ref(false)
const shiftLoading = ref(false)
const groupLoading = ref(false)

// 分页参数
const employeePage = ref(1)
const shiftPage = ref(1)
const groupPage = ref(1)
const employeeHasMore = ref(true)
const shiftHasMore = ref(true)
const groupHasMore = ref(true)

// 搜索关键词
const employeeSearchKeyword = ref('')
const shiftSearchKeyword = ref('')
const groupSearchKeyword = ref('')

// 新增关联
const selectedEmployees = ref<string[]>([])
const selectedShifts = ref<string[]>([])
const selectedGroups = ref<string[]>([])
const adding = ref(false)

// 监听对话框打开
watch(() => props.visible, (val) => {
  if (val && props.ruleId) {
    loadAssociations()
    // 不自动加载员工和班次，只在用户搜索时加载
  }
  else {
    resetData()
  }
})

// 监听切换标签（也不自动加载）
watch(activeTab, () => {
  // 切换标签时不自动加载数据，等待用户搜索
})

// 加载关联关系
async function loadAssociations() {
  if (!props.ruleId)
    return

  loading.value = true
  try {
    const res = await getRuleAssociations({
      orgId: props.orgId,
      ruleId: props.ruleId,
    })
    associations.value = res.associations || []
  }
  catch {
    ElMessage.error('加载关联关系失败')
  }
  finally {
    loading.value = false
  }
}

// 加载员工列表（支持分页和搜索）
async function loadEmployees(loadMore = false) {
  if (!loadMore) {
    employeePage.value = 1
    employeeList.value = []
  }

  if (!employeeHasMore.value && loadMore)
    return

  employeeLoading.value = true
  try {
    const res = await getSimpleEmployeeList({
      orgId: props.orgId,
      keyword: employeeSearchKeyword.value,
      page: employeePage.value,
      size: 50,
    })

    if (loadMore) {
      employeeList.value = [...employeeList.value, ...(res.items || [])]
    }
    else {
      employeeList.value = res.items || []
    }

    // 判断是否还有更多数据
    employeeHasMore.value = (res.items?.length || 0) >= 50
    if (loadMore && employeeHasMore.value) {
      employeePage.value++
    }
  }
  catch {
    ElMessage.error('加载员工列表失败')
  }
  finally {
    employeeLoading.value = false
  }
}

// 远程搜索员工
async function searchEmployees(query: string) {
  employeeSearchKeyword.value = query
  employeePage.value = 1
  employeeHasMore.value = true
  await loadEmployees()
}

// 员工选择框聚焦时加载初始数据
function handleEmployeeFocus() {
  if (employeeList.value.length === 0) {
    searchEmployees('')
  }
}

// 加载班次列表（支持分页和搜索）
async function loadShifts(loadMore = false) {
  if (!loadMore) {
    shiftPage.value = 1
    shiftList.value = []
  }

  if (!shiftHasMore.value && loadMore)
    return

  shiftLoading.value = true
  try {
    const res = await getShiftList({
      orgId: props.orgId,
      keyword: shiftSearchKeyword.value,
      page: shiftPage.value,
      size: 50,
    })

    if (loadMore) {
      shiftList.value = [...shiftList.value, ...(res.items || [])]
    }
    else {
      shiftList.value = res.items || []
    }

    // 判断是否还有更多数据
    shiftHasMore.value = (res.items?.length || 0) >= 50
    if (loadMore && shiftHasMore.value) {
      shiftPage.value++
    }
  }
  catch {
    ElMessage.error('加载班次列表失败')
  }
  finally {
    shiftLoading.value = false
  }
}

// 远程搜索班次
async function searchShifts(query: string) {
  shiftSearchKeyword.value = query
  shiftPage.value = 1
  shiftHasMore.value = true
  await loadShifts()
}

// 班次选择框聚焦时加载初始数据
function handleShiftFocus() {
  if (shiftList.value.length === 0) {
    searchShifts('')
  }
}

// 加载分组列表（支持分页和搜索）
async function loadGroups(loadMore = false) {
  if (!loadMore) {
    groupPage.value = 1
    groupList.value = []
  }

  if (!groupHasMore.value && loadMore)
    return

  groupLoading.value = true
  try {
    const res = await getGroupList({
      orgId: props.orgId,
      keyword: groupSearchKeyword.value,
      page: groupPage.value,
      size: 50,
    })

    if (loadMore) {
      groupList.value = [...groupList.value, ...(res.items || [])]
    }
    else {
      groupList.value = res.items || []
    }

    // 判断是否还有更多数据
    groupHasMore.value = (res.items?.length || 0) >= 50
    if (loadMore && groupHasMore.value) {
      groupPage.value++
    }
  }
  catch {
    ElMessage.error('加载分组列表失败')
  }
  finally {
    groupLoading.value = false
  }
}

// 远程搜索分组
async function searchGroups(query: string) {
  groupSearchKeyword.value = query
  groupPage.value = 1
  groupHasMore.value = true
  await loadGroups()
}

// 分组选择框聚焦时加载初始数据
function handleGroupFocus() {
  if (groupList.value.length === 0) {
    searchGroups('')
  }
}

// 计算当前标签页的关联
const currentAssociations = computed(() => {
  return associations.value.filter(a => a.targetType === activeTab.value)
})

// 计算已关联的ID列表
const associatedEmployeeIds = computed(() => {
  return associations.value
    .filter(a => a.targetType === 'employee')
    .map(a => a.targetId)
})

const associatedShiftIds = computed(() => {
  return associations.value
    .filter(a => a.targetType === 'shift')
    .map(a => a.targetId)
})

const associatedGroupIds = computed(() => {
  return associations.value
    .filter(a => a.targetType === 'group')
    .map(a => a.targetId)
})

// 获取关联目标的名称
function getTargetName(association: SchedulingRule.AssociationInfo): string {
  if (association.targetName) {
    return association.targetName
  }

  if (association.targetType === 'employee') {
    const employee = employeeList.value.find(e => e.id === association.targetId)
    return employee ? employee.name : association.targetId
  }
  else if (association.targetType === 'shift') {
    const shift = shiftList.value.find(s => s.id === association.targetId)
    return shift ? shift.name : association.targetId
  }
  else if (association.targetType === 'group') {
    const group = groupList.value.find(g => g.id === association.targetId)
    return group ? group.name : association.targetId
  }

  return association.targetId
}

// 添加关联
async function handleAdd() {
  let targetIds: string[] = []
  let typeName = ''

  if (activeTab.value === 'employee') {
    targetIds = selectedEmployees.value
    typeName = '员工'
  }
  else if (activeTab.value === 'shift') {
    targetIds = selectedShifts.value
    typeName = '班次'
  }
  else if (activeTab.value === 'group') {
    targetIds = selectedGroups.value
    typeName = '分组'
  }

  if (targetIds.length === 0) {
    ElMessage.warning(`请选择要关联的${typeName}`)
    return
  }

  adding.value = true
  try {
    await batchCreateRuleAssociation({
      orgId: props.orgId,
      ruleId: props.ruleId!,
      associations: targetIds.map(id => ({
        targetType: activeTab.value,
        targetId: id,
      })),
    })

    ElMessage.success('添加关联成功')
    selectedEmployees.value = []
    selectedShifts.value = []
    selectedGroups.value = []
    loadAssociations()
    emit('success')
  }
  catch {
    ElMessage.error('添加关联失败')
  }
  finally {
    adding.value = false
  }
}

// 删除关联
async function handleDelete(association: SchedulingRule.AssociationInfo) {
  try {
    await ElMessageBox.confirm('确认删除该关联关系吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await deleteRuleAssociation(
      association.ruleId,
      association.targetType,
      association.targetId,
      props.orgId,
    )

    ElMessage.success('删除成功')
    loadAssociations()
    emit('success')
  }
  catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 重置数据
function resetData() {
  associations.value = []
  employeeList.value = []
  shiftList.value = []
  groupList.value = []
  selectedEmployees.value = []
  selectedShifts.value = []
  selectedGroups.value = []
  activeTab.value = 'employee'
  employeePage.value = 1
  shiftPage.value = 1
  groupPage.value = 1
  employeeHasMore.value = true
  shiftHasMore.value = true
  groupHasMore.value = true
  employeeSearchKeyword.value = ''
  shiftSearchKeyword.value = ''
  groupSearchKeyword.value = ''
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="`管理关联关系 - ${ruleName}`"
    width="800px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-tabs v-model="activeTab">
      <!-- 员工关联 -->
      <el-tab-pane label="员工关联" name="employee">
        <div class="association-section">
          <!-- 添加关联 -->
          <div class="add-section">
            <el-select
              v-model="selectedEmployees"
              multiple
              filterable
              remote
              reserve-keyword
              placeholder="输入关键词搜索员工（点击展开加载）"
              :remote-method="searchEmployees"
              :loading="employeeLoading"
              style="width: 100%; max-width: 500px"
              @focus="handleEmployeeFocus"
            >
              <el-option
                v-for="employee in employeeList.filter(e => !associatedEmployeeIds.includes(e.id))"
                :key="employee.id"
                :label="`${employee.name} (${employee.employeeId})`"
                :value="employee.id"
              />
            </el-select>
            <el-button
              type="primary"
              :icon="Plus"
              :loading="adding"
              :disabled="selectedEmployees.length === 0"
              @click="handleAdd"
            >
              添加关联
            </el-button>
          </div>

          <!-- 关联列表 -->
          <div v-loading="loading" class="association-list">
            <div v-if="currentAssociations.length === 0" class="empty-hint">
              暂无员工关联
            </div>
            <div
              v-for="association in currentAssociations"
              :key="`${association.targetType}-${association.targetId}`"
              class="association-item"
            >
              <span class="association-name">{{ getTargetName(association) }}</span>
              <el-button
                type="danger"
                link
                :icon="Delete"
                size="small"
                @click="handleDelete(association)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 班次关联 -->
      <el-tab-pane label="班次关联" name="shift">
        <div class="association-section">
          <!-- 添加关联 -->
          <div class="add-section">
            <el-select
              v-model="selectedShifts"
              multiple
              filterable
              remote
              reserve-keyword
              placeholder="输入关键词搜索班次（点击展开加载）"
              :remote-method="searchShifts"
              :loading="shiftLoading"
              style="width: 100%; max-width: 500px"
              @focus="handleShiftFocus"
            >
              <el-option
                v-for="shift in shiftList.filter(s => !associatedShiftIds.includes(s.id))"
                :key="shift.id"
                :label="shift.name"
                :value="shift.id"
              />
            </el-select>
            <el-button
              type="primary"
              :icon="Plus"
              :loading="adding"
              :disabled="selectedShifts.length === 0"
              @click="handleAdd"
            >
              添加关联
            </el-button>
          </div>

          <!-- 关联列表 -->
          <div v-loading="loading" class="association-list">
            <div v-if="currentAssociations.length === 0" class="empty-hint">
              暂无班次关联
            </div>
            <div
              v-for="association in currentAssociations"
              :key="`${association.targetType}-${association.targetId}`"
              class="association-item"
            >
              <span class="association-name">{{ getTargetName(association) }}</span>
              <el-button
                type="danger"
                link
                :icon="Delete"
                size="small"
                @click="handleDelete(association)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 分组关联 -->
      <el-tab-pane label="分组关联" name="group">
        <div class="association-section">
          <!-- 添加关联 -->
          <div class="add-section">
            <el-select
              v-model="selectedGroups"
              multiple
              filterable
              remote
              reserve-keyword
              placeholder="输入关键词搜索分组（点击展开加载）"
              :remote-method="searchGroups"
              :loading="groupLoading"
              style="width: 100%; max-width: 500px"
              @focus="handleGroupFocus"
            >
              <el-option
                v-for="group in groupList.filter(g => !associatedGroupIds.includes(g.id))"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
            <el-button
              type="primary"
              :icon="Plus"
              :loading="adding"
              :disabled="selectedGroups.length === 0"
              @click="handleAdd"
            >
              添加关联
            </el-button>
          </div>

          <!-- 关联列表 -->
          <div v-loading="loading" class="association-list">
            <div v-if="currentAssociations.length === 0" class="empty-hint">
              暂无分组关联
            </div>
            <div
              v-for="association in currentAssociations"
              :key="`${association.targetType}-${association.targetId}`"
              class="association-item"
            >
              <span class="association-name">{{ getTargetName(association) }}</span>
              <el-button
                type="danger"
                link
                :icon="Delete"
                size="small"
                @click="handleDelete(association)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="handleClose">
        关闭
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.association-section {
  min-height: 300px;
}

.add-section {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.association-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 200px;
}

.association-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  transition: all 0.3s;

  &:hover {
    background-color: var(--el-fill-color);
  }
}

.association-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.empty-hint {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
