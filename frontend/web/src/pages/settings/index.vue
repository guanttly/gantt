<script setup lang="ts">
import { ElMessage, ElSelect, ElOption } from 'element-plus'
import { onMounted, ref } from 'vue'
import { getSetting, setSetting } from '@/api/systemSetting'
import { userPreferenceApi } from '@/services/api'

/**
 * 获取当前组织 ID
 * 从 localStorage 或环境变量获取
 */
function getCurrentOrgId(): string {
  return localStorage.getItem('current_org_id') || import.meta.env.VITE_DEFAULT_ORG_ID || 'default-org'
}

const orgId = getCurrentOrgId()
const continuousScheduling = ref(true)
const workflowVersion = ref<'v2' | 'v3' | 'v4'>('v2')

// 获取当前用户ID
function getCurrentUserId(): string {
  return localStorage.getItem('userId') || 'default-user'
}

async function loadSettings() {
  try {
    // 加载连续排班设置
    const response = await getSetting(orgId, 'continuous_scheduling')
    continuousScheduling.value = response.value === 'true'
  }
  catch (error: any) {
    console.error('Failed to load settings:', error)
    // 如果设置不存在，使用默认值 true
    continuousScheduling.value = true
  }

  try {
    // 加载工作流版本偏好
    const userId = getCurrentUserId()
    const version = await userPreferenceApi.getUserWorkflowVersion(userId, orgId)
    workflowVersion.value = version
  }
  catch (error: any) {
    console.error('Failed to load workflow version preference:', error)
    // 如果加载失败，使用默认值 v2
    workflowVersion.value = 'v2'
  }
}

async function handleContinuousSchedulingChange(value: boolean) {
  try {
    await setSetting(orgId, 'continuous_scheduling', value ? 'true' : 'false', '连续排班配置')
    ElMessage.success('设置已保存')
  }
  catch (error: any) {
    console.error('Failed to save setting:', error)
    ElMessage.error('保存设置失败')
    // 恢复原值
    continuousScheduling.value = !value
  }
}

async function handleWorkflowVersionChange(value: 'v2' | 'v3' | 'v4') {
  const oldValue = workflowVersion.value
  // 先更新本地状态，以便UI立即响应
  workflowVersion.value = value

  try {
    const userId = getCurrentUserId()
    await userPreferenceApi.setUserWorkflowVersion(userId, orgId, value)
    ElMessage.success(`已切换到 ${value.toUpperCase()} 版本`)
  }
  catch (error: any) {
    console.error('Failed to save workflow version preference:', error)
    ElMessage.error('保存设置失败')
    // 恢复原值
    workflowVersion.value = oldValue
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<template>
  <div class="settings-page">
    <el-card shadow="never" class="settings-card">
      <template #header>
        <div class="card-header">
          <h2 class="page-title">
            系统设置
          </h2>
        </div>
      </template>

      <div class="settings-content">
        <div class="setting-item">
          <div class="setting-header">
            <div class="setting-info">
              <h3 class="setting-title">
                连续排班
              </h3>
              <p class="setting-desc">
                关闭后，每个班次排班完成后需要用户审核和调整，才能继续下一个班次
              </p>
            </div>
            <div class="setting-action">
              <el-switch
                v-model="continuousScheduling"
                :active-value="true"
                :inactive-value="false"
                size="large"
                @change="handleContinuousSchedulingChange"
              />
              <span class="setting-status">
                {{ continuousScheduling ? '已开启' : '已关闭' }}
              </span>
            </div>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-header">
            <div class="setting-info">
              <h3 class="setting-title">
                排班工作流版本
              </h3>
              <p class="setting-desc">
                选择排班工作流版本。V2为传统版本，V3为渐进式排班版本，V4为AI增强版本
              </p>
            </div>
            <div class="setting-action">
              <el-select
                v-model="workflowVersion"
                size="large"
                style="width: 120px"
                @change="handleWorkflowVersionChange"
              >
                <el-option label="V2" value="v2" />
                <el-option label="V3" value="v3" />
                <el-option label="V4" value="v4" />
              </el-select>
              <span class="setting-status">
                当前版本：{{ workflowVersion.toUpperCase() }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style lang="scss" scoped>
.settings-page {
  padding: 24px;
  background: var(--el-bg-color-page, #f5f7fa);
  min-height: calc(100vh - 64px);
}

.settings-card {
  max-width: 1200px;
  margin: 0 auto;
  border-radius: 8px;

  :deep(.el-card__header) {
    padding: 20px 24px;
    border-bottom: 1px solid var(--el-border-color-lighter, #ebeef5);
    background: var(--el-bg-color, #ffffff);
  }

  :deep(.el-card__body) {
    padding: 32px;
  }
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary, #303133);
  line-height: 1.5;
}

.settings-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.setting-item {
  padding: 24px;
  background: var(--el-bg-color, #ffffff);
  border: 1px solid var(--el-border-color-light, #e4e7ed);
  border-radius: 8px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5, #a0cfff);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  }
}

.setting-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.setting-info {
  flex: 1;
  min-width: 0;
}

.setting-title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary, #303133);
  line-height: 1.5;
}

.setting-desc {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-regular, #606266);
  line-height: 1.6;
}

.setting-action {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

.setting-status {
  font-size: 14px;
  color: var(--el-text-color-regular, #606266);
  white-space: nowrap;
  font-weight: 500;
}

// 响应式设计
@media (max-width: 768px) {
  .settings-page {
    padding: 16px;
  }

  .settings-card {
    :deep(.el-card__body) {
      padding: 20px;
    }
  }

  .setting-header {
    flex-direction: column;
    gap: 16px;
  }

  .setting-action {
    flex-direction: row;
    align-items: center;
    width: 100%;
    justify-content: space-between;
  }
}
</style>
