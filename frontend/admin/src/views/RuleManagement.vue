<script setup lang="ts">
import type { PlatformRule, PlatformRulePayload } from '@/api/platform'
import { onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NModal, NSelect, NSpin, NTag, NSwitch, useMessage } from 'naive-ui'
import { createPlatformRule, deletePlatformRule, disablePlatformRule, listPlatformRules, restorePlatformRule, updatePlatformRule } from '@/api/platform'

const loading = ref(false)
const saving = ref(false)
const rules = ref<PlatformRule[]>([])
const dialogVisible = ref(false)
const disableVisible = ref(false)
const editingRule = ref<PlatformRule | null>(null)
const disablingRule = ref<PlatformRule | null>(null)
const disableReason = ref('')
const configText = ref('{}')
const message = useMessage()

const form = ref<PlatformRulePayload>({
  name: '',
  category: 'constraint',
  sub_type: 'limit',
  config: {},
  priority: 0,
  is_enabled: true,
  description: '',
})

const categoryOptions = [
  { label: '约束', value: 'constraint' },
  { label: '偏好', value: 'preference' },
  { label: '依赖', value: 'dependency' },
]

const subTypeOptions = [
  { label: '限制', value: 'limit' },
  { label: '禁止', value: 'forbid' },
  { label: '必须', value: 'must' },
  { label: '偏好', value: 'prefer' },
  { label: '来源', value: 'source' },
  { label: '顺序', value: 'order' },
  { label: '最小休息', value: 'min_rest' },
]

async function loadRules() {
  loading.value = true
  try {
    rules.value = await listPlatformRules()
  }
  finally {
    loading.value = false
  }
}

function openCreate() {
  editingRule.value = null
  form.value = { name: '', category: 'constraint', sub_type: 'limit', config: {}, priority: 0, is_enabled: true, description: '' }
  configText.value = '{\n  "type": "max_count"\n}'
  dialogVisible.value = true
}

function openEdit(rule: PlatformRule) {
  if (rule.is_inherited) {
    message.warning('继承规则请通过禁用/恢复或在下级新建规则处理')
    return
  }
  editingRule.value = rule
  form.value = {
    name: rule.name,
    category: rule.category,
    sub_type: rule.sub_type,
    config: rule.config,
    priority: rule.priority,
    is_enabled: rule.is_enabled,
    description: rule.description,
  }
  configText.value = JSON.stringify(rule.config, null, 2)
  dialogVisible.value = true
}

async function submit() {
  try {
    form.value.config = JSON.parse(configText.value)
  }
  catch {
    message.error('规则配置必须是合法 JSON')
    return
  }

  saving.value = true
  try {
    if (editingRule.value) {
      await updatePlatformRule(editingRule.value.id, form.value)
      message.success('规则已更新')
    }
    else {
      await createPlatformRule(form.value)
      message.success('规则已创建')
    }
    dialogVisible.value = false
    await loadRules()
  }
  finally {
    saving.value = false
  }
}

async function removeRule(rule: PlatformRule) {
  await deletePlatformRule(rule.id)
  message.success('规则已删除')
  await loadRules()
}

function openDisable(rule: PlatformRule) {
  disablingRule.value = rule
  disableReason.value = ''
  disableVisible.value = true
}

async function confirmDisable() {
  if (!disablingRule.value || !disableReason.value.trim()) {
    message.warning('请填写禁用原因')
    return
  }
  await disablePlatformRule(disablingRule.value.id, disableReason.value)
  disableVisible.value = false
  message.success('继承规则已禁用')
  await loadRules()
}

async function restoreRule(rule: PlatformRule) {
  await restorePlatformRule(rule.id)
  message.success('继承规则已恢复')
  await loadRules()
}

onMounted(loadRules)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">规则管理</h2>
          <p class="page-subtitle">在平台侧统一维护规则，并支持查看继承状态、禁用与恢复。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left" />
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增规则</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>规则</th>
                  <th>分类</th>
                  <th>来源</th>
                  <th>优先级</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in rules" :key="item.id" :class="{ disabled: item.disabled }">
                  <td>
                    <div>{{ item.name }}</div>
                    <div class="table-muted">{{ item.description || item.disabled_reason || '-' }}</div>
                  </td>
                  <td>{{ item.category }} / {{ item.sub_type }}</td>
                  <td>
                    <n-tag v-if="item.is_inherited" size="small" type="warning">继承</n-tag>
                    <span>{{ item.source_node || '本级' }}</span>
                  </td>
                  <td>{{ item.priority }}</td>
                  <td>
                    <n-tag v-if="item.disabled" size="small" type="error">已禁用</n-tag>
                    <n-tag v-else-if="item.is_enabled" size="small" type="success">启用</n-tag>
                    <n-tag v-else size="small">停用</n-tag>
                  </td>
                  <td>
                    <div class="table-actions">
                      <n-button v-if="!item.is_inherited" text type="primary" @click="openEdit(item)">编辑</n-button>
                      <n-button v-if="item.is_inherited && !item.disabled" text type="warning" @click="openDisable(item)">禁用</n-button>
                      <n-button v-if="item.disabled" text type="success" @click="restoreRule(item)">恢复</n-button>
                      <n-button v-if="!item.is_inherited" text type="error" @click="removeRule(item)">删除</n-button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!rules.length">
                  <td colspan="6" class="table-empty">暂无规则</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="editingRule ? '编辑规则' : '新增规则'" style="width: min(760px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="96">
          <div class="form-grid two-column">
            <n-form-item label="名称">
              <n-input v-model:value="form.name" placeholder="输入规则名称" />
            </n-form-item>
            <n-form-item label="优先级">
              <n-input-number v-model:value="form.priority" :min="0" style="width: 100%" />
            </n-form-item>
            <n-form-item label="分类">
              <n-select v-model:value="form.category" :options="categoryOptions" />
            </n-form-item>
            <n-form-item label="子类型">
              <n-select v-model:value="form.sub_type" :options="subTypeOptions" />
            </n-form-item>
          </div>
          <n-form-item label="启用">
            <n-switch v-model:value="form.is_enabled" />
          </n-form-item>
          <n-form-item label="描述">
            <n-input v-model:value="form.description" type="textarea" placeholder="输入规则说明（可选）" />
          </n-form-item>
          <n-form-item label="配置 JSON">
            <n-input v-model:value="configText" type="textarea" :autosize="{ minRows: 8, maxRows: 14 }" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" :loading="saving" @click="submit">保存</n-button>
          </div>
        </template>
      </n-modal>

      <n-modal v-model:show="disableVisible" preset="card" title="禁用继承规则" style="width: min(520px, calc(100vw - 32px))">
        <n-form label-placement="left" label-width="88">
          <n-form-item label="规则">
            <n-input :value="disablingRule?.name || ''" disabled />
          </n-form-item>
          <n-form-item label="禁用原因">
            <n-input v-model:value="disableReason" type="textarea" placeholder="请填写禁用原因" />
          </n-form-item>
        </n-form>
        <template #footer>
          <div class="modal-actions">
            <n-button @click="disableVisible = false">取消</n-button>
            <n-button type="warning" @click="confirmDisable">确认禁用</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.admin-table {
  width: 100%;
  border-collapse: collapse;
}

.admin-table th,
.admin-table td {
  padding: 14px 12px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  text-align: left;
  vertical-align: top;
}

.admin-table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.admin-table tr.disabled {
  color: rgba(15, 23, 42, 0.56);
}

.table-muted {
  color: var(--admin-text-muted);
  font-size: 12px;
}

.table-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.table-empty {
  padding: 36px 0;
  text-align: center;
  color: var(--admin-text-muted);
}

.form-grid.two-column {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 760px) {
  .form-grid.two-column {
    grid-template-columns: 1fr;
  }
}
</style>