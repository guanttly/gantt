<script setup lang="ts">
import { ElSelect, ElOption, ElMessage } from 'element-plus'
import { ref, onMounted, watch } from 'vue'
import { userPreferenceApi } from '@/services/api'

// 获取当前组织ID和用户ID
function getCurrentOrgId(): string {
  return localStorage.getItem('orgId') || 'default-org'
}

function getCurrentUserId(): string {
  return localStorage.getItem('userId') || 'default-user'
}

const props = defineProps<{
  modelValue?: 'v2' | 'v3' | 'v4'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: 'v2' | 'v3' | 'v4']
  'change': [value: 'v2' | 'v3' | 'v4']
}>()

const currentVersion = ref<'v2' | 'v3' | 'v4'>('v2')
const loading = ref(false)

// 加载用户偏好
async function loadUserPreference() {
  loading.value = true
  try {
    const orgId = getCurrentOrgId()
    const userId = getCurrentUserId()
    const version = await userPreferenceApi.getUserWorkflowVersion(userId, orgId)
    currentVersion.value = version
    emit('update:modelValue', version)
  }
  catch (error) {
    console.error('加载用户工作流版本偏好失败:', error)
    // 使用默认值，不显示错误提示
  }
  finally {
    loading.value = false
  }
}

// 切换版本
async function handleVersionChange(value: 'v2' | 'v3' | 'v4') {
  if (value === currentVersion.value) {
    return
  }

  loading.value = true
  try {
    const orgId = getCurrentOrgId()
    const userId = getCurrentUserId()
    await userPreferenceApi.setUserWorkflowVersion(userId, orgId, value)
    currentVersion.value = value
    emit('update:modelValue', value)
    emit('change', value)
    ElMessage.success(`已切换到 ${value.toUpperCase()} 版本`)
  }
  catch (error) {
    console.error('设置用户工作流版本偏好失败:', error)
    ElMessage.error('切换版本失败，请重试')
    // 恢复原值
    await loadUserPreference()
  }
  finally {
    loading.value = false
  }
}

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal && newVal !== currentVersion.value) {
    currentVersion.value = newVal
  }
}, { immediate: true })

onMounted(() => {
  // 如果外部没有传入值，则加载用户偏好
  if (!props.modelValue) {
    loadUserPreference()
  }
  else {
    currentVersion.value = props.modelValue
  }
})
</script>

<template>
  <div class="workflow-version-selector">
    <el-select
      v-model="currentVersion"
      :loading="loading"
      size="small"
      style="width: 100px"
      @change="handleVersionChange"
    >
      <el-option label="V2" value="v2" />
      <el-option label="V3" value="v3" />
      <el-option label="V4" value="v4" />
    </el-select>
  </div>
</template>

<style lang="scss" scoped>
.workflow-version-selector {
  display: inline-flex;
  align-items: center;
}
</style>
