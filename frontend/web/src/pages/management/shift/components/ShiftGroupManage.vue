<script setup lang="ts">
import { Delete, Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ref, watch } from 'vue'
import { getGroupList } from '@/api/group'
import { addGroupToShift, getShiftGroups, removeGroupFromShift } from '@/api/shift'

interface Props {
  visible: boolean
  shift: Shift.ShiftInfo | null
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

// 当前班次的关联分组
const shiftGroups = ref<Shift.ShiftGroupInfo[]>([])
const loading = ref(false)

// 所有可用分组
const allGroups = ref<Group.GroupInfo[]>([])
const groupsLoading = ref(false)

// 添加分组相关
const addDialogVisible = ref(false)
const selectedGroupId = ref('')
const selectedPriority = ref(0)
const addLoading = ref(false)

// 获取班次的关联分组
async function fetchShiftGroups() {
  if (!props.shift)
    return

  loading.value = true
  try {
    const groups = await getShiftGroups(props.shift.id)
    shiftGroups.value = groups || []
  }
  catch (error: any) {
    console.error('获取班次关联分组失败:', error)
    ElMessage.error('获取关联分组失败')
  }
  finally {
    loading.value = false
  }
}

// 获取所有可用分组
async function fetchAllGroups() {
  groupsLoading.value = true
  try {
    const res = await getGroupList({
      orgId: props.orgId,
      page: 1,
      size: 100,
    })
    allGroups.value = res.items || []
  }
  catch (error: any) {
    console.error('获取分组列表失败:', error)
    ElMessage.error('获取分组列表失败')
  }
  finally {
    groupsLoading.value = false
  }
}

// 获取未关联的分组
const availableGroups = ref<Group.GroupInfo[]>([])
function updateAvailableGroups() {
  const linkedGroupIds = shiftGroups.value.map(sg => sg.groupId)
  availableGroups.value = allGroups.value.filter(g => !linkedGroupIds.includes(g.id))
}

// 打开添加分组对话框
function handleAdd() {
  updateAvailableGroups()
  if (availableGroups.value.length === 0) {
    ElMessage.warning('没有可添加的分组')
    return
  }
  selectedGroupId.value = ''
  selectedPriority.value = 0
  addDialogVisible.value = true
}

// 确认添加分组
async function confirmAdd() {
  if (!selectedGroupId.value) {
    ElMessage.warning('请选择分组')
    return
  }

  if (!props.shift)
    return

  addLoading.value = true
  try {
    await addGroupToShift(props.shift.id, selectedGroupId.value, selectedPriority.value)
    ElMessage.success('添加成功')
    addDialogVisible.value = false
    await fetchShiftGroups()
    emit('success')
  }
  catch (error) {
    console.error('添加分组失败:', error)
    // 错误已由 request 拦截器统一处理
  }
  finally {
    addLoading.value = false
  }
}

// 删除关联分组
async function handleDelete(groupId: string, groupName: string) {
  if (!props.shift)
    return

  try {
    await ElMessageBox.confirm(`确认从班次中移除分组"${groupName}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await removeGroupFromShift(props.shift.id, groupId)
    ElMessage.success('移除成功')
    await fetchShiftGroups()
    emit('success')
  }
  catch (error) {
    if (error !== 'cancel') {
      console.error('移除分组失败:', error)
      ElMessage.error('移除失败')
    }
  }
}

// 刷新数据
function handleRefresh() {
  fetchShiftGroups()
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 监听对话框打开
watch(() => props.visible, (val) => {
  if (val && props.shift) {
    fetchShiftGroups()
    fetchAllGroups()
  }
})
</script>

<template>
  <el-dialog
    :model-value="visible"
    width="700px"
    @close="handleClose"
  >
    <template #header>
      <div class="dialog-header">
        <span style="color: #606266">班次关联分组 - </span>
        <span style="color: #409EFF; font-weight: bold; font-size: 16px">{{ shift?.name || '' }}</span>
      </div>
    </template>

    <div class="toolbar">
      <el-button type="primary" :icon="Plus" @click="handleAdd">
        添加分组
      </el-button>
      <el-button :icon="Refresh" @click="handleRefresh">
        刷新
      </el-button>
    </div>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-top: 16px"
    >
      <template #title>
        <span style="font-size: 13px">优先级说明：数值越小优先级越高，0表示默认优先级</span>
      </template>
    </el-alert>

    <el-table
      v-loading="loading"
      :data="shiftGroups"
      border
      style="margin-top: 12px"
    >
      <el-table-column type="index" label="序号" width="60" align="center" />
      <el-table-column prop="groupName" label="分组名称" min-width="120">
        <template #default="{ row }">
          {{ row.groupName || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="groupCode" label="分组编码" width="120">
        <template #default="{ row }">
          {{ row.groupCode || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.priority === 0 ? 'primary' : ''">
            {{ row.priority }}
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
      <el-table-column prop="notes" label="备注" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.notes || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right" align="center">
        <template #default="{ row }">
          <el-button
            type="danger"
            link
            :icon="Delete"
            size="small"
            @click="handleDelete(row.groupId, row.groupName)"
          >
            移除
          </el-button>
        </template>
      </el-table-column>

      <!-- 空状态 -->
      <template #empty>
        <el-empty description="暂无关联分组" :image-size="80">
          <el-button type="primary" :icon="Plus" @click="handleAdd">
            添加分组
          </el-button>
        </el-empty>
      </template>
    </el-table>

    <template #footer>
      <el-button @click="handleClose">
        关闭
      </el-button>
    </template>
  </el-dialog>

  <!-- 添加分组对话框 -->
  <el-dialog
    v-model="addDialogVisible"
    title="添加关联分组"
    width="450px"
    :close-on-click-modal="false"
  >
    <el-form label-width="80px">
      <el-form-item label="选择分组" required>
        <el-select
          v-model="selectedGroupId"
          placeholder="请选择要添加的分组"
          filterable
          style="width: 100%"
          :loading="groupsLoading"
        >
          <el-option
            v-for="group in availableGroups"
            :key="group.id"
            :label="`${group.name} (${group.code})`"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="优先级">
        <el-input-number
          v-model="selectedPriority"
          :min="0"
          :max="999"
          :step="1"
          style="width: 100%"
          placeholder="请输入优先级"
        />
        <div style="color: #909399; font-size: 12px; margin-top: 4px">
          数值越小优先级越高，0表示默认优先级
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="addDialogVisible = false">
        取消
      </el-button>
      <el-button type="primary" :loading="addLoading" @click="confirmAdd">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.dialog-header {
  font-size: 16px;
  line-height: 24px;
}

.toolbar {
  display: flex;
  gap: 12px;
}
</style>
