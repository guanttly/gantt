<script setup lang="ts">
import type { AIModelConfigView, AppConfigView, WorkflowConfigView, WorkflowNodeView } from '@/api/admin'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NSwitch, NTag, useMessage } from 'naive-ui'
import { listAppConfigs, updateAppSettings, updateAppWorkflow, updateWorkflowNodeModel } from '@/api/admin'

const message = useMessage()
const loading = ref(true)
const savingModel = ref(false)
const savingSettings = ref(false)
const savingWorkflow = ref(false)

const apps = ref<AppConfigView[]>([])
const selectedAppCode = ref('')
const selectedWorkflowKey = ref('')
const selectedNodeKey = ref('')

const modelForm = reactive<AIModelConfigView>({
  provider: '',
  model: '',
  timeout_seconds: 60,
  temperature: null,
  max_tokens: 0,
  enabled: true,
})

const providerOptions = [
  { label: '继承默认', value: '' },
  { label: '百炼', value: 'bailian' },
  { label: 'OpenAI', value: 'openai' },
  { label: 'Ollama', value: 'ollama' },
]

const selectedApp = computed(() => apps.value.find(app => app.code === selectedAppCode.value) || null)
const selectedWorkflow = computed(() => selectedApp.value?.workflows.find(workflow => workflow.key === selectedWorkflowKey.value) || null)
const sortedNodes = computed(() => [...(selectedWorkflow.value?.nodes || [])].sort((a, b) => a.position.x - b.position.x || a.position.y - b.position.y))
const selectedNode = computed(() => sortedNodes.value.find(node => node.key === selectedNodeKey.value) || sortedNodes.value.find(node => node.configurable) || sortedNodes.value[0] || null)
const llmNodeCount = computed(() => sortedNodes.value.filter(node => node.kind === 'llm').length)
const systemNodeCount = computed(() => sortedNodes.value.filter(node => node.kind !== 'llm').length)
const selectedNodeGuidance = computed(() => {
  if (!selectedNode.value)
    return ''
  if (selectedNode.value.key === 'ai_select')
    return '该节点只为剩余空缺生成候选建议，最终是否采纳仍受候选约束、规则校验和兜底逻辑控制。'
  if (selectedNode.value.kind === 'llm')
    return '该节点只输出解析结果或建议，不会绕过业务规则直接决定最终排班。'
  return '该节点由确定性业务逻辑执行，不依赖大模型。'
})

function copyModelToForm(config: AIModelConfigView) {
  modelForm.provider = config.provider || ''
  modelForm.model = config.model || ''
  modelForm.timeout_seconds = config.timeout_seconds || 60
  modelForm.temperature = config.temperature ?? null
  modelForm.max_tokens = config.max_tokens || 0
  modelForm.enabled = config.enabled !== false
}

function ensureSelection() {
  if (!apps.value.length)
    return
  if (!selectedApp.value)
    selectedAppCode.value = apps.value[0].code
  if (!selectedWorkflow.value)
    selectedWorkflowKey.value = selectedApp.value?.workflows[0]?.key || ''
  if (!selectedNode.value)
    selectedNodeKey.value = sortedNodes.value.find(node => node.configurable)?.key || sortedNodes.value[0]?.key || ''
  if (selectedNode.value)
    copyModelToForm(selectedNode.value.model_config)
}

async function loadApps() {
  loading.value = true
  try {
    apps.value = await listAppConfigs()
    ensureSelection()
  }
  catch {
    message.error('加载应用配置失败')
  }
  finally {
    loading.value = false
  }
}

function selectWorkflow(workflow: WorkflowConfigView) {
  selectedWorkflowKey.value = workflow.key
  selectedNodeKey.value = workflow.nodes.find(node => node.configurable)?.key || workflow.nodes[0]?.key || ''
}

function selectNode(node: WorkflowNodeView) {
  selectedNodeKey.value = node.key
}

async function saveSettings() {
  if (!selectedApp.value)
    return
  savingSettings.value = true
  try {
    const saved = await updateAppSettings(selectedApp.value.code, selectedApp.value.settings)
    selectedApp.value.settings = { ...selectedApp.value.settings, ...saved }
    message.success('应用设置已保存')
  }
  catch {
    message.error('保存应用设置失败')
  }
  finally {
    savingSettings.value = false
  }
}

async function saveWorkflow() {
  if (!selectedApp.value || !selectedWorkflow.value)
    return
  savingWorkflow.value = true
  try {
    const saved = await updateAppWorkflow(selectedApp.value.code, selectedWorkflow.value.key, {
      name: selectedWorkflow.value.name,
      version: selectedWorkflow.value.version,
      description: selectedWorkflow.value.description,
      enabled: selectedWorkflow.value.enabled,
    })
    selectedWorkflow.value.name = saved.name
    selectedWorkflow.value.version = saved.version
    selectedWorkflow.value.description = saved.description
    selectedWorkflow.value.enabled = saved.enabled
    message.success('工作流配置已保存')
  }
  catch {
    message.error('保存工作流配置失败')
  }
  finally {
    savingWorkflow.value = false
  }
}

async function saveNodeModel() {
  if (!selectedApp.value || !selectedWorkflow.value || !selectedNode.value || !selectedNode.value.configurable)
    return
  savingModel.value = true
  try {
    const saved = await updateWorkflowNodeModel(selectedApp.value.code, selectedWorkflow.value.key, selectedNode.value.key, { ...modelForm })
    selectedNode.value.model_config = saved
    copyModelToForm(saved)
    message.success('节点模型已保存')
  }
  catch {
    message.error('保存节点模型失败')
  }
  finally {
    savingModel.value = false
  }
}

function setBoolSetting(key: string, value: boolean) {
  if (selectedApp.value)
    selectedApp.value.settings[key] = String(value)
}

function setNumberSetting(key: string, value: number | null) {
  if (selectedApp.value)
    selectedApp.value.settings[key] = String(value ?? 0)
}

function isTrue(value?: string) {
  return value === 'true'
}

function nodeSubtitle(node: WorkflowNodeView) {
  if (!node.configurable)
    return '系统节点'
  const model = node.model_config.model || '继承默认模型'
  const provider = node.model_config.provider || '继承默认 Provider'
  return `${provider} / ${model}`
}

function selectedNodeKindLabel(node: WorkflowNodeView) {
  return node.kind === 'llm' ? '模型节点' : '系统节点'
}

watch(selectedWorkflow, (workflow) => {
  if (!workflow)
    return
  if (!workflow.nodes.some(node => node.key === selectedNodeKey.value))
    selectedNodeKey.value = workflow.nodes.find(node => node.configurable)?.key || workflow.nodes[0]?.key || ''
})

watch(selectedNode, (node) => {
  if (node)
    copyModelToForm(node.model_config)
})

onMounted(loadApps)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">应用配置</h2>
          <p class="page-subtitle">
            管理应用级排班设置、工作流启停与节点模型。
            <template v-if="selectedApp">
              当前应用：{{ selectedApp.name }}，共 {{ selectedApp.workflows.length }} 套工作流。
            </template>
          </p>
        </div>
      </section>

      <section class="app-workspace">
        <aside class="app-rail">
          <section class="app-sidebar page-card rail-card">
            <div class="rail-card-inner">
              <div class="rail-card-header">
                <h3 class="rail-title">应用</h3>
                <span class="rail-meta">{{ apps.length }}</span>
              </div>
              <div class="app-sidebar-inner">
                <button
                  v-for="app in apps"
                  :key="app.code"
                  class="app-item"
                  :class="{ active: app.code === selectedAppCode }"
                  type="button"
                  @click="selectedAppCode = app.code; ensureSelection()"
                >
                  <span class="app-item-title">{{ app.name }}</span>
                  <span class="app-item-subtitle">{{ app.workflows.length }} 套工作流</span>
                </button>
              </div>
            </div>
          </section>

          <section v-if="selectedApp" class="workflow-list page-card rail-card">
            <div class="rail-card-inner">
              <div class="rail-card-header">
                <h3 class="rail-title">工作流</h3>
                <span class="rail-meta">{{ selectedApp.workflows.length }}</span>
              </div>
              <div class="workflow-list-inner">
                <button
                  v-for="workflow in selectedApp.workflows"
                  :key="workflow.key"
                  class="workflow-item"
                  :class="{ active: workflow.key === selectedWorkflowKey }"
                  type="button"
                  @click="selectWorkflow(workflow)"
                >
                  <span class="workflow-name">{{ workflow.name }}</span>
                  <span class="workflow-meta">{{ workflow.version }} · {{ workflow.enabled ? '启用' : '停用' }}</span>
                </button>
              </div>
            </div>
          </section>
        </aside>

        <main v-if="selectedApp" class="app-main">
          <section class="page-card action-dock">
            <div class="action-dock-main">
              <div class="action-dock-title-group">
                <div class="action-dock-title-row">
                  <span class="action-dock-label">当前应用</span>
                  <strong class="action-dock-current">{{ selectedApp.name }}</strong>
                  <span class="action-dock-summary">{{ selectedWorkflow?.name || '未选择工作流' }}<template v-if="selectedNode"> / {{ selectedNode.name }}</template></span>
                </div>
              </div>
              <div class="action-dock-actions">
                <n-button size="small" :loading="loading" @click="loadApps">刷新</n-button>
                <n-button size="small" type="primary" :loading="savingSettings" @click="saveSettings">保存应用设置</n-button>
                <n-button v-if="selectedWorkflow" size="small" secondary type="primary" :loading="savingWorkflow" @click="saveWorkflow">保存工作流</n-button>
                <n-button v-if="selectedNode?.configurable" size="small" secondary type="primary" :loading="savingModel" @click="saveNodeModel">保存节点模型</n-button>
              </div>
            </div>
          </section>

          <section class="page-card app-settings-panel compact-panel">
            <div class="overview-head">
              <div>
                <p class="panel-subtitle">{{ selectedApp.description }}</p>
              </div>
              <div class="inline-meta-chip">{{ selectedApp.workflows.length }} 套工作流</div>
            </div>
            <div class="settings-grid">
              <label class="setting-row">
                <span>自动发布排班</span>
                <n-switch :value="isTrue(selectedApp.settings.schedule_auto_publish)" @update:value="setBoolSetting('schedule_auto_publish', $event)" />
              </label>
              <label class="setting-row">
                <span>锁定天数</span>
                <n-input-number :value="Number(selectedApp.settings.schedule_lock_days || 0)" :min="0" :max="90" @update:value="setNumberSetting('schedule_lock_days', $event)" />
              </label>
            </div>
          </section>

          <section class="workflow-area">
            <div v-if="selectedWorkflow" class="workflow-main page-card">
              <div class="workflow-main-inner">
                <div class="panel-header workflow-header">
                  <div>
                    <h3 class="panel-title">{{ selectedWorkflow.name }}</h3>
                    <p class="panel-subtitle">{{ selectedWorkflow.description }}</p>
                  </div>
                  <div class="workflow-actions">
                    <div class="workflow-toggle-state">
                      <span class="workflow-toggle-label">{{ selectedWorkflow.enabled ? '已启用' : '已停用' }}</span>
                      <n-switch v-model:value="selectedWorkflow.enabled" />
                    </div>
                  </div>
                </div>

                <div class="workflow-metrics">
                  <div class="metric-card">
                    <span class="metric-label">节点总数</span>
                    <strong class="metric-value">{{ sortedNodes.length }}</strong>
                  </div>
                  <div class="metric-card">
                    <span class="metric-label">模型节点</span>
                    <strong class="metric-value">{{ llmNodeCount }}</strong>
                  </div>
                  <div class="metric-card">
                    <span class="metric-label">系统节点</span>
                    <strong class="metric-value">{{ systemNodeCount }}</strong>
                  </div>
                  <div class="metric-card active-node">
                    <span class="metric-label">当前节点</span>
                    <strong class="metric-value">{{ selectedNode?.name || '未选择' }}</strong>
                  </div>
                </div>

                <div class="workflow-canvas-shell">
                  <div class="workflow-canvas-bar">
                    <span class="canvas-pill">{{ selectedNode?.configurable ? '当前节点可配置' : '当前节点为系统逻辑' }}</span>
                  </div>

                  <div class="workflow-canvas-scroll">
                    <div class="workflow-canvas">
                      <template v-for="(node, index) in sortedNodes" :key="node.key">
                        <button
                          class="flow-node"
                          :class="{ selected: selectedNode?.key === node.key, configurable: node.configurable }"
                          type="button"
                          @click="selectNode(node)"
                        >
                          <span class="flow-node-topline">
                            <span class="flow-node-name">{{ node.name }}</span>
                            <n-tag size="small" :type="node.kind === 'llm' ? 'success' : 'default'">{{ node.kind === 'llm' ? '模型' : '系统' }}</n-tag>
                          </span>
                          <span class="flow-node-subtitle">{{ nodeSubtitle(node) }}</span>
                        </button>
                        <div v-if="index < sortedNodes.length - 1" class="flow-edge" aria-hidden="true" />
                      </template>
                    </div>
                  </div>
                </div>

                <section v-if="selectedNode" class="node-detail-panel">
                  <div class="panel-header compact node-detail-header">
                    <div>
                      <div class="node-detail-eyebrow">节点详情</div>
                      <h3 class="panel-title">{{ selectedNode.name }}</h3>
                      <p class="panel-subtitle">{{ selectedNode.description }}</p>
                    </div>
                    <div class="node-tags">
                      <n-tag :type="selectedNode.kind === 'llm' ? 'success' : 'default'">{{ selectedNodeKindLabel(selectedNode) }}</n-tag>
                      <n-tag v-if="selectedNode.configurable" type="info">可配置</n-tag>
                    </div>
                  </div>

                  <div class="node-guidance-card">
                    <p>{{ selectedNodeGuidance }}</p>
                  </div>

                  <div v-if="selectedNode.configurable" class="node-config-card">
                    <n-form label-placement="top" class="node-form node-form-grid">
                      <n-form-item label="启用节点配置">
                        <n-switch v-model:value="modelForm.enabled" />
                      </n-form-item>
                      <n-form-item label="AI 提供商">
                        <n-select v-model:value="modelForm.provider" :options="providerOptions" />
                      </n-form-item>
                      <n-form-item label="模型">
                        <n-input v-model:value="modelForm.model" placeholder="留空则继承默认模型" />
                      </n-form-item>
                      <n-form-item label="超时时间（秒）">
                        <n-input-number v-model:value="modelForm.timeout_seconds" :min="5" :max="600" />
                      </n-form-item>
                      <n-form-item label="Temperature">
                        <n-input-number v-model:value="modelForm.temperature" :min="0" :max="2" :step="0.1" clearable />
                      </n-form-item>
                      <n-form-item label="Max Tokens">
                        <n-input-number v-model:value="modelForm.max_tokens" :min="0" :max="32000" />
                      </n-form-item>
                    </n-form>

                    <div class="node-config-footer">
                      <p class="node-config-note">留空表示继承应用默认 Provider 或模型。这里只配置该节点的调用策略，不改变确定性业务规则。</p>
                    </div>
                  </div>

                  <div v-else class="system-node-note">
                    <n-tag>系统节点</n-tag>
                    <p>该节点由排班引擎执行，不调用大模型。</p>
                  </div>
                </section>
              </div>
            </div>
          </section>
        </main>
      </section>
    </div>
  </div>
</template>

<style scoped>

.page-shell {
  --app-config-header-stack-offset: 68px;
}

.app-workspace {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}

.app-rail {
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-self: start;
}

.rail-card-inner,
.workflow-main-inner {
  padding: 14px;
}

.rail-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}

.rail-title {
  margin: 0;
  color: #0f172a;
  font-size: 13px;
  font-weight: 700;
}

.rail-meta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.1);
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
}

.app-sidebar-inner,
.workflow-list-inner {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: calc(50vh - 40px);
  overflow: auto;
}

.app-sidebar-inner,
.workflow-list-inner,
.workflow-canvas-scroll {
  scrollbar-width: thin;
  scrollbar-color: rgba(15, 118, 110, 0.38) rgba(148, 163, 184, 0.14);
}

.app-sidebar-inner::-webkit-scrollbar,
.workflow-list-inner::-webkit-scrollbar,
.workflow-canvas-scroll::-webkit-scrollbar {
  width: 12px;
  height: 12px;
}

.app-sidebar-inner::-webkit-scrollbar-track,
.workflow-list-inner::-webkit-scrollbar-track,
.workflow-canvas-scroll::-webkit-scrollbar-track {
  background: rgba(148, 163, 184, 0.12);
  border-radius: 999px;
}

.app-sidebar-inner::-webkit-scrollbar-thumb,
.workflow-list-inner::-webkit-scrollbar-thumb,
.workflow-canvas-scroll::-webkit-scrollbar-thumb {
  border: 3px solid transparent;
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(15, 118, 110, 0.5), rgba(59, 130, 246, 0.24));
  background-clip: content-box;
}

.app-sidebar-inner::-webkit-scrollbar-thumb:hover,
.workflow-list-inner::-webkit-scrollbar-thumb:hover,
.workflow-canvas-scroll::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(13, 148, 136, 0.64), rgba(59, 130, 246, 0.36));
  background-clip: content-box;
}

.app-item,
.workflow-item {
  width: 100%;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--admin-text);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 11px 12px;
  text-align: left;
  transition: all 0.18s;
}

.app-item:hover,
.workflow-item:hover {
  background: var(--admin-surface-soft);
  border-color: var(--admin-border);
}

.app-item.active,
.workflow-item.active {
  background: var(--admin-primary-soft);
  border-color: rgba(15, 118, 110, 0.28);
}

.app-item-title,
.workflow-name {
  font-size: 14px;
  font-weight: 700;
}

.app-item-subtitle,
.workflow-meta,
.panel-subtitle,
.flow-node-subtitle {
  color: var(--admin-text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.app-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 12px;
}

.action-dock {
  position: sticky;
  z-index: 11;
  border-color: rgba(15, 23, 42, 0.05);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(15, 118, 110, 0.12), rgba(255, 255, 255, 0.92)),
    var(--admin-surface);
  backdrop-filter: blur(12px);
  box-shadow: var(--admin-shadow);
}

.action-dock-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px;
}

.action-dock-title-group {
  min-width: 0;
}

.action-dock-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.action-dock-label {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
  color: #0f766e;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  white-space: nowrap;
}

.action-dock-current {
  color: #0f172a;
  font-size: 15px;
  line-height: 1.2;
  white-space: nowrap;
}

.action-dock-summary {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.action-dock-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.app-settings-panel,
.workflow-main,
.node-panel,
.workflow-list,
.app-sidebar {
  border-radius: 12px;
}

.app-settings-panel {
  padding: 12px 14px;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.panel-header.compact {
  margin-bottom: 10px;
}

.panel-title {
  margin: 0;
  color: #0f172a;
  font-size: 16px;
  line-height: 1.3;
}

.panel-subtitle {
  margin: 4px 0 0;
}

.overview-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.overview-eyebrow {
  color: #0f766e;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.overview-title {
  margin: 4px 0 0;
  color: #0f172a;
  font-size: 20px;
  line-height: 1.15;
}

.inline-meta-chip {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
  color: #0f766e;
  font-size: 11px;
  font-weight: 700;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(220px, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.setting-row {
  min-height: 40px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 10px;
  background: #fff;
  font-size: 12px;
}

.workflow-area {
  min-width: 0;
}

.workflow-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workflow-toggle-state {
  display: flex;
  align-items: center;
  gap: 10px;
}

.workflow-toggle-label {
  color: var(--admin-text-muted);
  font-size: 13px;
  white-space: nowrap;
}

.workflow-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.metric-card {
  border: 1px solid var(--admin-border);
  border-radius: 10px;
  background: linear-gradient(180deg, #ffffff 0%, #f7fbff 100%);
  padding: 8px 10px;
}

.metric-card.active-node {
  border-color: rgba(15, 118, 110, 0.26);
  background: linear-gradient(180deg, rgba(236, 253, 245, 0.96) 0%, rgba(240, 249, 255, 0.96) 100%);
}

.metric-label {
  display: block;
  color: var(--admin-text-muted);
  font-size: 11px;
}

.metric-value {
  display: block;
  margin-top: 2px;
  color: #0f172a;
  font-size: 16px;
  line-height: 1.2;
}

.workflow-canvas-shell {
  margin-top: 8px;
  border: 1px solid rgba(15, 118, 110, 0.14);
  border-radius: 14px;
  background: linear-gradient(180deg, #fcfffe 0%, #f8fbff 100%);
  padding: 8px;
}

.workflow-canvas-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-bottom: 4px;
}

.canvas-pill {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 8px;
  border: 1px solid rgba(15, 118, 110, 0.2);
  border-radius: 999px;
  background: rgba(236, 253, 245, 0.9);
  color: #0f766e;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.workflow-canvas-scroll {
  overflow: auto;
  padding: 2px 0 4px;
}

.workflow-canvas {
  display: inline-flex;
  min-width: max-content;
  min-height: 102px;
  align-items: center;
  gap: 8px;
}

.flow-node {
  width: 170px;
  min-width: 170px;
  min-height: 82px;
  border: 1px solid var(--admin-border);
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #fbfdff 100%);
  color: var(--admin-text);
  cursor: pointer;
  padding: 10px;
  text-align: left;
  transition: all 0.18s;
}

.flow-node.configurable {
  border-color: rgba(15, 118, 110, 0.28);
}

.flow-node.selected {
  border-color: #0f766e;
  box-shadow: 0 0 0 4px rgba(15, 118, 110, 0.12);
}

.flow-node-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.flow-node-name {
  font-size: 13px;
  font-weight: 700;
  line-height: 1.35;
}

.flow-node-subtitle {
  display: -webkit-box;
  margin-top: 6px;
  min-height: 26px;
  overflow: hidden;
  word-break: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.flow-edge {
  width: 38px;
  min-width: 38px;
  height: 2px;
  background: var(--admin-border-strong);
  position: relative;
}

.flow-edge::after {
  content: '';
  position: absolute;
  right: -1px;
  top: -4px;
  width: 0;
  height: 0;
  border-top: 5px solid transparent;
  border-bottom: 5px solid transparent;
  border-left: 8px solid var(--admin-border-strong);
}

.node-detail-panel {
  margin-top: 10px;
  border-top: 1px solid rgba(148, 163, 184, 0.18);
  padding-top: 10px;
}

.node-detail-header {
  align-items: flex-start;
}

.node-detail-eyebrow {
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.node-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.node-guidance-card,
.node-config-card,
.system-node-note {
  border: 1px solid var(--admin-border);
  border-radius: 14px;
  background: #fff;
}

.node-guidance-card {
  margin-top: 10px;
  padding: 10px 12px;
  background: linear-gradient(180deg, rgba(240, 249, 255, 0.96) 0%, rgba(248, 250, 252, 1) 100%);
}

.node-guidance-card p {
  margin: 0;
  color: #1e293b;
  font-size: 12px;
  line-height: 1.7;
}

.node-config-card {
  margin-top: 10px;
  padding: 12px;
}

.node-form {
  display: flex;
  flex-direction: column;
}

.node-form-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 12px;
}

.node-config-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 0;
}

.node-config-note {
  margin: 0;
  color: var(--admin-text-muted);
  font-size: 11px;
  line-height: 1.6;
}

.system-node-note {
  margin-top: 12px;
  padding: 12px 14px;
  background: var(--admin-surface-soft);
}

.system-node-note p {
  margin: 8px 0 0;
  color: var(--admin-text-muted);
  font-size: 12px;
  line-height: 1.65;
}

@media (max-width: 1240px) {
  .page-shell {
    --app-config-header-stack-offset: 68px;
  }

  .app-workspace {
    grid-template-columns: 1fr;
  }

  .app-rail {
    position: static;
  }

  .workflow-metrics,
  .node-form-grid {
    grid-template-columns: 1fr;
  }

  .workflow-canvas-bar,
  .node-config-footer,
  .action-dock-main,
  .action-dock-title-row,
  .overview-head,
  .panel-header,
  .node-detail-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .action-dock-actions {
    flex-wrap: wrap;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .page-shell {
    --app-config-header-stack-offset: 106px;
  }
}
</style>
