<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NButton, NDivider, NForm, NFormItem, NInput, NSelect, NSpin, NSwitch, useDialog, useMessage } from 'naive-ui'
import { getSystemConfig, updateSystemConfig } from '@/api/admin'

const loading = ref(true)
const saving = ref(false)
const message = useMessage()
const dialog = useDialog()

const config = ref<Record<string, string>>({
  ai_enabled: 'false',
  ai_provider: 'openai',
  ai_model: '',
  ai_api_key: '',
  ai_base_url: '',
  system_name: '',
  system_logo: '',
})

function toBool(value: string) {
  return value === 'true'
}

function setBool(key: string, value: boolean) {
  config.value[key] = String(value)
}

async function loadConfig() {
  loading.value = true
  try {
    const res = await getSystemConfig()
    config.value = { ...config.value, ...res }
  }
  catch {
    message.error('加载系统配置失败')
  }
  finally {
    loading.value = false
  }
}

async function handleSave() {
  const confirmed = await new Promise<boolean>((resolve) => {
    dialog.warning({
      title: '确认保存',
      content: '确认保存系统配置？',
      positiveText: '保存',
      negativeText: '取消',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
    })
  })

  if (!confirmed)
    return

  saving.value = true
  try {
    await updateSystemConfig(config.value)
    message.success('保存成功')
  }
  catch {
    message.error('保存失败')
  }
  finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">系统配置</h2>
          <p class="page-subtitle">这里维护平台级运行参数，排班设置与工作流已迁移到应用配置。</p>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <template #description>
              正在加载系统配置
            </template>

            <n-form label-placement="left" label-width="140" class="config-form">
              <h3 class="section-title">平台 AI 默认配置</h3>
              <n-form-item label="启用 AI">
                <n-switch :value="toBool(config.ai_enabled)" @update:value="setBool('ai_enabled', $event)" />
              </n-form-item>
              <n-form-item label="AI 提供商">
                <n-select
                  v-model:value="config.ai_provider"
                  style="width: 240px"
                  :options="[
                    { label: 'OpenAI', value: 'openai' },
                    { label: 'Ollama', value: 'ollama' },
                    { label: '百炼', value: 'bailian' },
                  ]"
                />
              </n-form-item>
              <n-form-item label="模型">
                <n-input v-model:value="config.ai_model" style="width: 320px" placeholder="例：gpt-4o-mini" />
              </n-form-item>
              <n-form-item label="API Key">
                <n-input v-model:value="config.ai_api_key" type="password" show-password-on="click" style="width: 320px" />
              </n-form-item>
              <n-form-item label="Base URL">
                <n-input v-model:value="config.ai_base_url" style="width: 320px" placeholder="可选" />
              </n-form-item>

              <n-divider />

              <h3 class="section-title">系统配置</h3>
              <n-form-item label="系统名称">
                <n-input v-model:value="config.system_name" style="width: 320px" />
              </n-form-item>
              <n-form-item label="系统 Logo URL">
                <n-input v-model:value="config.system_logo" style="width: 320px" />
              </n-form-item>

              <n-form-item>
                <n-button type="primary" :loading="saving" @click="handleSave">保存配置</n-button>
              </n-form-item>
            </n-form>
          </n-spin>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.section-title { font-size: 16px; font-weight: 600; margin: 0 0 16px; color: #0f172a; }
.config-form { max-width: 600px; }
</style>
