<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import type { FixedAssignmentForm } from '../type'
import { Delete, Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { getEmployeeList } from '@/api/employee'
import { batchCreateFixedAssignments, deleteFixedAssignment, getFixedAssignments } from '@/api/shift'
import {
  fixedAssignmentWeekdayOptions,
  formatFixedAssignmentPattern,
  formatPatternTypeText,
  getMonthdayOptions,
  weekPatternOptions,
} from '../logic'

interface Props {
  visible: boolean
  shift: Shift.ShiftInfo | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

// 组织ID
const orgId = ref('default-org')

// 对话框可见性
const dialogVisible = computed({
  get: () => props.visible,
  set: value => emit('update:visible', value),
})

// 加载状态
const loading = ref(false)
const saving = ref(false)

// 已配置的固定人员列表
const assignments = ref<Shift.FixedAssignment[]>([])

// 员工列表
const staffList = ref<Employee.EmployeeInfo[]>([])

// 表单引用
const formRef = ref<FormInstance>()

// 添加表单数据
const form = reactive<FixedAssignmentForm>({
  staffId: '',
  patternType: 'weekly',
  weekdays: [],
  weekPattern: 'every',
  monthdays: [],
  specificDates: [],
  dateRange: null,
})

// 表单验证规则
const rules: FormRules = {
  staffId: [
    { required: true, message: '请选择人员', trigger: 'change' },
  ],
  patternType: [
    { required: true, message: '请选择配置模式', trigger: 'change' },
  ],
}

// 选项（从 logic.ts 导入）
const weekdayOptions = fixedAssignmentWeekdayOptions
const monthdayOptions = computed(() => getMonthdayOptions())

// 监听对话框打开
watch(() => props.visible, (visible) => {
  if (visible && props.shift) {
    loadData()
  }
})

// 加载数据
async function loadData() {
  if (!props.shift)
    return

  loading.value = true
  try {
    // 先加载员工列表，确保 staffList 可用
    await loadStaffList()
    // 然后加载配置，这样可以填充人员名称
    await loadAssignments()
  }
  catch (error) {
    console.error('Failed to load data:', error)
  }
  finally {
    loading.value = false
  }
}

// 加载固定人员配置
async function loadAssignments() {
  if (!props.shift)
    return

  try {
    const res = await getFixedAssignments(props.shift.id)
    const loadedAssignments = res || []
    // 为每个配置填充人员名称
    assignments.value = loadedAssignments.map(assignment => ({
      ...assignment,
      staffName: assignment.staffName || getStaffName(assignment.staffId),
    }))
  }
  catch (error) {
    console.error('Failed to load fixed assignments:', error)
    ElMessage.error('加载固定人员配置失败')
    throw error
  }
}

// 加载员工列表
async function loadStaffList() {
  try {
    const res = await getEmployeeList({
      orgId: orgId.value,
      page: 1,
      size: 1000, // 获取所有员工
    })
    staffList.value = res.items || []
  }
  catch (error) {
    console.error('Failed to load staff list:', error)
    ElMessage.error('加载员工列表失败')
    throw error
  }
}

// 格式化函数（从 logic.ts 导入）
const formatPatternText = formatPatternTypeText
const formatPattern = formatFixedAssignmentPattern

// 获取员工名称
function getStaffName(staffId: string): string {
  const staff = staffList.value.find(s => s.id === staffId)
  return staff ? staff.name : staffId
}

// 处理模式切换
function handlePatternTypeChange() {
  // 只清除其他模式的字段，保留当前选择的模式
  if (form.patternType === 'weekly') {
    form.monthdays = []
    form.specificDates = []
  }
  else if (form.patternType === 'monthly') {
    form.weekdays = []
    form.weekPattern = 'every'
    form.specificDates = []
  }
  else if (form.patternType === 'specific') {
    form.weekdays = []
    form.weekPattern = 'every'
    form.monthdays = []
  }
  formRef.value?.clearValidate()
}

// 重置表单
function resetForm() {
  form.staffId = ' '
  form.patternType = 'weekly'
  form.weekdays = []
  form.weekPattern = 'every'
  form.monthdays = []
  form.specificDates = []
  form.dateRange = null
}

// 添加配置
async function handleAdd() {
  if (!formRef.value)
    return

  try {
    await formRef.value.validate()
  }
  catch {
    return
  }

  // 验证规则
  if (form.patternType === 'weekly' && form.weekdays.length === 0) {
    ElMessage.warning('请至少选择一个周几')
    return
  }
  if (form.patternType === 'monthly' && form.monthdays.length === 0) {
    ElMessage.warning('请至少选择一个日期')
    return
  }
  if (form.patternType === 'specific' && form.specificDates.length === 0) {
    ElMessage.warning('请至少选择一个日期')
    return
  }

  // 检查是否已存在该人员
  const existingAssignment = assignments.value.find(a => a.staffId === form.staffId)
  if (existingAssignment) {
    ElMessage.warning('该人员已配置，请先删除旧配置')
    return
  }

  // 构建配置对象
  const newAssignment: Shift.FixedAssignment = {
    staffId: form.staffId,
    staffName: getStaffName(form.staffId),
    patternType: form.patternType,
    isActive: true,
  }

  if (form.patternType === 'weekly') {
    newAssignment.weekdays = [...form.weekdays].sort((a, b) => a - b)
    newAssignment.weekPattern = form.weekPattern || 'every'
  }
  else if (form.patternType === 'monthly') {
    newAssignment.monthdays = [...form.monthdays].sort((a, b) => a - b)
  }
  else if (form.patternType === 'specific') {
    newAssignment.specificDates = [...form.specificDates]
  }

  if (form.dateRange) {
    newAssignment.startDate = form.dateRange[0]
    newAssignment.endDate = form.dateRange[1]
  }

  // 添加到列表
  assignments.value.push(newAssignment)

  ElMessage.success('已添加配置')
  resetForm()
}

// 删除配置
async function handleDelete(assignment: Shift.FixedAssignment) {
  try {
    await ElMessageBox.confirm('确认删除该固定人员配置吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    // 如果有ID，调用API删除
    if (assignment.id && props.shift) {
      try {
        await deleteFixedAssignment(props.shift.id, assignment.id)
        ElMessage.success('删除成功')
      }
      catch (error) {
        console.error('Failed to delete assignment:', error)
        ElMessage.error('删除失败')
        return
      }
    }

    // 从列表中移除
    const index = assignments.value.findIndex(a => a.staffId === assignment.staffId)
    if (index > -1) {
      assignments.value.splice(index, 1)
    }
  }
  catch (error) {
    if (error !== 'cancel') {
      console.error('Error in handleDelete:', error)
    }
  }
}

// 保存配置
async function handleSave() {
  if (!props.shift)
    return

  // 如果列表为空但表单有数据，自动添加到列表
  if (assignments.value.length === 0) {
    if (form.staffId) {
      // 验证表单
      if (!formRef.value)
        return

      try {
        await formRef.value.validate()
      }
      catch {
        ElMessage.warning('请完善表单信息')
        return
      }

      // 验证规则
      if (form.patternType === 'weekly' && form.weekdays.length === 0) {
        ElMessage.warning('请至少选择一个周几')
        return
      }
      if (form.patternType === 'monthly' && form.monthdays.length === 0) {
        ElMessage.warning('请至少选择一个日期')
        return
      }
      if (form.patternType === 'specific' && form.specificDates.length === 0) {
        ElMessage.warning('请至少选择一个日期')
        return
      }

      // 检查是否已存在该人员
      const existingAssignment = assignments.value.find(a => a.staffId === form.staffId)
      if (existingAssignment) {
        ElMessage.warning('该人员已配置，请先删除旧配置')
        return
      }

      // 构建配置对象并添加到列表
      const newAssignment: Shift.FixedAssignment = {
        staffId: form.staffId,
        staffName: getStaffName(form.staffId),
        patternType: form.patternType,
        isActive: true,
      }

      if (form.patternType === 'weekly') {
        newAssignment.weekdays = [...form.weekdays].sort((a, b) => a - b)
        newAssignment.weekPattern = form.weekPattern || 'every'
      }
      else if (form.patternType === 'monthly') {
        newAssignment.monthdays = [...form.monthdays].sort((a, b) => a - b)
      }
      else if (form.patternType === 'specific') {
        newAssignment.specificDates = [...form.specificDates]
      }

      if (form.dateRange) {
        newAssignment.startDate = form.dateRange[0]
        newAssignment.endDate = form.dateRange[1]
      }

      assignments.value.push(newAssignment)
    }
    else {
      ElMessage.warning('请至少添加一个固定人员配置')
      return
    }
  }

  saving.value = true
  try {
    await batchCreateFixedAssignments({
      shiftId: props.shift.id,
      assignments: assignments.value,
    })

    ElMessage.success('保存成功')
    emit('success')
    handleClose()
  }
  catch (error) {
    console.error('Failed to save assignments:', error)
    ElMessage.error('保存失败')
  }
  finally {
    saving.value = false
  }
}

// 关闭对话框
function handleClose() {
  resetForm()
  dialogVisible.value = false
}

// 刷新列表
function handleRefresh() {
  loadData()
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :title="`配置固定人员 - ${shift?.name || ''}`"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-loading="loading" class="fixed-assignment-manager">
      <!-- 已配置人员列表 -->
      <div class="section">
        <div class="section-header">
          <h4>已配置人员 ({{ assignments.length }})</h4>
          <el-button :icon="Refresh" size="small" @click="handleRefresh">
            刷新
          </el-button>
        </div>

        <el-table
          :data="assignments"
          stripe
          border
          style="width: 100%"
          max-height="300"
        >
          <el-table-column prop="staffName" label="人员" width="120" />
          <el-table-column prop="patternType" label="模式" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.patternType === 'weekly' ? 'primary' : row.patternType === 'monthly' ? 'success' : 'info'" size="small">
                {{ formatPatternText(row.patternType) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="rule" label="规则" min-width="200">
            <template #default="{ row }">
              <span>{{ formatPattern(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="dateRange" label="生效时间" width="200">
            <template #default="{ row }">
              <span v-if="row.startDate || row.endDate" style="font-size: 12px">
                {{ row.startDate || '不限' }} ~ {{ row.endDate || '不限' }}
              </span>
              <span v-else style="color: #909399">永久生效</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                :icon="Delete"
                size="small"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>

          <template #empty>
            <el-empty description="暂无固定人员配置" :image-size="80" />
          </template>
        </el-table>
      </div>

      <el-divider />

      <!-- 添加新配置 -->
      <div class="section">
        <div class="section-header">
          <h4>添加固定人员</h4>
        </div>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-width="100px"
        >
          <el-form-item label="选择人员" prop="staffId">
            <el-select
              v-model="form.staffId"
              filterable
              placeholder="请选择人员"
              style="width: 100%"
            >
              <el-option
                v-for="staff in staffList"
                :key="staff.id"
                :label="staff.name"
                :value="staff.id"
                :disabled="assignments.some(a => a.staffId === staff.id)"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="配置模式" prop="patternType">
            <el-radio-group v-model="form.patternType" @change="handlePatternTypeChange">
              <el-radio value="weekly">
                按周重复
              </el-radio>
              <el-radio value="monthly">
                按月重复
              </el-radio>
              <el-radio value="specific">
                指定日期
              </el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- 按周重复 -->
          <template v-if="form.patternType === 'weekly'">
            <el-form-item label="周期">
              <el-radio-group v-model="form.weekPattern">
                <el-radio
                  v-for="option in weekPatternOptions"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                  <span style="color: #909399; font-size: 12px; margin-left: 4px">
                    {{ option.desc }}
                  </span>
                </el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="选择周几">
              <el-checkbox-group v-model="form.weekdays">
                <el-checkbox
                  v-for="option in weekdayOptions"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                </el-checkbox>
              </el-checkbox-group>
              <div class="form-tip">
                选择每周（或奇数周/偶数周）的哪几天上这个班次
              </div>
            </el-form-item>
          </template>

          <!-- 按月重复 -->
          <el-form-item v-if="form.patternType === 'monthly'" label="选择日期">
            <el-select
              v-model="form.monthdays"
              multiple
              collapse-tags
              collapse-tags-tooltip
              placeholder="请选择每月日期"
              style="width: 100%"
            >
              <el-option
                v-for="option in monthdayOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <div class="form-tip">
              选择每月的哪几天上这个班次（如：1号、15号、30号）
            </div>
          </el-form-item>

          <!-- 指定日期 -->
          <el-form-item v-if="form.patternType === 'specific'" label="选择日期">
            <el-date-picker
              v-model="form.specificDates"
              type="dates"
              placeholder="选择多个具体日期"
              value-format="YYYY-MM-DD"
              style="width: 100%"
            />
            <div class="form-tip">
              选择具体的某几天上这个班次
            </div>
          </el-form-item>

          <!-- 生效时间（可选） -->
          <el-form-item label="生效时间">
            <el-date-picker
              v-model="form.dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              value-format="YYYY-MM-DD"
              style="width: 100%"
            />
            <div class="form-tip">
              可选。不设置则永久生效
            </div>
          </el-form-item>

          <el-form-item>
            <el-button type="primary" :icon="Plus" @click="handleAdd">
              添加到列表
            </el-button>
            <el-button @click="resetForm">
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleSave"
        >
          保存配置
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.fixed-assignment-manager {
  .section {
    margin-bottom: 20px;

    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      h4 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }
    }
  }

  .form-tip {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
    line-height: 1.5;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
