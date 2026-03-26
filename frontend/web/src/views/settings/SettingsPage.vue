<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, ref } from 'vue'
import { getSetting, setSetting } from '@/api/settings'

const continuousScheduling = ref(true)

async function loadSettings() {
  try {
    const response = await getSetting('continuous_scheduling')
    continuousScheduling.value = response.value === 'true'
  }
  catch {
    continuousScheduling.value = true
  }
}

async function handleContinuousSchedulingChange(value: boolean | string | number) {
  try {
    await setSetting('continuous_scheduling', value ? 'true' : 'false', '连续排班配置')
    ElMessage.success('设置已保存')
  }
  catch {
    ElMessage.error('保存设置失败')
    continuousScheduling.value = !value
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
        <h2 class="page-title">
          系统设置
        </h2>
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
              <span class="setting-status">{{ continuousScheduling ? '已开启' : '已关闭' }}</span>
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
                版本选择已废弃，当前系统统一使用 V4 工作流。
              </p>
            </div>
            <div class="setting-action">
              <el-tag type="success" size="large">
                V4
              </el-tag>
              <span class="setting-status">当前版本：V4</span>
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
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.settings-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.setting-item {
  padding: 24px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  }
}

.setting-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
}

.setting-info {
  flex: 1;
}

.setting-title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
}

.setting-desc {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.setting-action {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}

.setting-status {
  font-size: 14px;
  color: var(--el-text-color-regular);
}
</style>
