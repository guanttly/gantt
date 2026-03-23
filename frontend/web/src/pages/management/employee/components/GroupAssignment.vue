<script setup lang="ts">
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { addGroupMember, getGroupList, removeGroupMember } from '@/api/group'

interface Props {
  employee: Employee.EmployeeInfo
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  success: []
}>()

// 所有可用分组
const allGroups = ref<Group.GroupInfo[]>([])
const loading = ref(false)

// 添加分组弹窗
const addDialogVisible = ref(false)
const selectedGroupId = ref('')
const addLoading = ref(false)

// 获取所有分组列表
async function fetchAllGroups() {
  loading.value = true
  try {
    const res = await getGroupList({
      orgId: props.orgId,
      page: 1,
      size: 1000, // 获取所有分组
    })
    allGroups.value = res.items || []
  }
  catch {
    ElMessage.error('获取分组列表失败')
  }
  finally {
    loading.value = false
  }
}

// 获取可添加的分组（排除已有的）
const availableGroups = computed(() => {
  const currentGroupIds = props.employee.groups?.map(g => g.id) || []
  return allGroups.value.filter(g => !currentGroupIds.includes(g.id))
})

// 打开添加分组对话框
function handleAddGroup() {
  if (availableGroups.value.length === 0) {
    ElMessage.warning('该员工已加入所有分组')
    return
  }
  selectedGroupId.value = ''
  addDialogVisible.value = true
}

// 确认添加分组
async function confirmAddGroup() {
  if (!selectedGroupId.value) {
    ElMessage.warning('请选择要添加的分组')
    return
  }

  addLoading.value = true
  try {
    await addGroupMember({
      groupId: selectedGroupId.value,
      orgId: props.orgId,
      employeeId: props.employee.id,
    })
    ElMessage.success('添加成功')
    addDialogVisible.value = false
    emit('success')
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
  finally {
    addLoading.value = false
  }
}

// 移除分组
async function handleRemoveGroup(group: { id: string, code: string, name: string }) {
  try {
    await removeGroupMember({
      groupId: group.id,
      employeeId: props.employee.id,
      orgId: props.orgId,
    })
    ElMessage.success('移除成功')
    emit('success')
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
}

onMounted(() => {
  fetchAllGroups()
})
</script>

<template>
  <div class="group-assignment">
    <!-- 添加分组按钮放在前面 -->
    <el-button
      :icon="Plus"
      size="small"
      class="add-btn"
      @click="handleAddGroup"
    >
      添加分组
    </el-button>

    <div v-if="employee.groups && employee.groups.length > 0" class="groups-list">
      <el-tag
        v-for="group in employee.groups"
        :key="group.id"
        closable
        type="primary"
        effect="light"
        size="small"
        class="group-tag"
        @close="handleRemoveGroup(group)"
      >
        <!-- <span class="group-code">{{ group.code }}</span> -->
        <span class="group-name">{{ group.name }}</span>
      </el-tag>
    </div>
    <div v-else class="no-groups">
      <span class="no-groups-text">暂无分组</span>
    </div>

    <!-- 添加分组对话框 -->
    <el-dialog
      v-model="addDialogVisible"
      title="添加分组"
      width="500px"
      :close-on-click-modal="false"
      :append-to-body="true"
    >
      <el-form label-width="80px">
        <el-form-item label="员工">
          <!-- <span class="employee-id">{{ employee.employeeId }}</span> -->
          <div class="employee-name">
            {{ employee.name }}
          </div>
        </el-form-item>
        <el-form-item label="选择分组" required>
          <el-select
            v-model="selectedGroupId"
            placeholder="请选择分组"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="group in availableGroups"
              :key="group.id"
              :label="`${group.code} - ${group.name}`"
              :value="group.id"
            >
              <div class="group-option">
                <span class="option-code">{{ group.code }}</span>
                <span class="option-name">{{ group.name }}</span>
                <el-tag
                  v-if="group.type"
                  type="info"
                  size="small"
                  effect="plain"
                  class="option-type"
                >
                  {{ group.type }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="addDialogVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="addLoading"
          @click="confirmAddGroup"
        >
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.group-assignment {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.add-btn {
  flex-shrink: 0;
  order: -1; // 确保按钮在最前面
}

.groups-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1;
}

.group-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.3);
  }

  .group-code {
    font-weight: 600;
    color: #409eff;
  }

  .group-name {
    color: #606266;
  }
}

.no-groups {
  flex: 1;

  .no-groups-text {
    color: #909399;
    font-size: 13px;
  }
}

// 对话框内样式
.employee-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;

  .employee-id {
    font-weight: 600;
    color: #409eff;
  }

  .employee-name {
    color: #303133;
    font-weight: bold;
  }
}

.group-option {
  display: flex;
  align-items: center;
  gap: 8px;

  .option-code {
    font-weight: 600;
    color: #409eff;
    min-width: 60px;
  }

  .option-name {
    flex: 1;
    color: #606266;
  }

  .option-type {
    font-size: 12px;
  }
}
</style>
