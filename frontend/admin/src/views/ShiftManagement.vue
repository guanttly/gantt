<script setup lang="ts">
import type { PlatformShift, PlatformShiftPayload } from '@/api/platform'
import { onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NModal, NSpin, NTag, NSwitch, useDialog, useMessage } from 'naive-ui'
import { createPlatformShift, deletePlatformShift, listPlatformShifts, togglePlatformShift, updatePlatformShift } from '@/api/platform'

const loading = ref(false)
const saving = ref(false)
const shifts = ref<PlatformShift[]>([])
const dialogVisible = ref(false)
const editingShift = ref<PlatformShift | null>(null)
const message = useMessage()
const dialog = useDialog()

const form = ref<PlatformShiftPayload>({
  name: '',
  code: '',
  start_time: '08:00',
  end_time: '17:00',
  duration: 540,
  is_cross_day: false,
  color: '#0f766e',
  priority: 0,
})

async function loadShifts() {
  loading.value = true
  try {
    shifts.value = await listPlatformShifts()
  }
  finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    name: '',
    code: '',
    start_time: '08:00',
    end_time: '17:00',
    duration: 540,
    is_cross_day: false,
    color: '#0f766e',
    priority: 0,
  }
}

function openCreate() {
  editingShift.value = null
  resetForm()
  dialogVisible.value = true
}

function openEdit(shift: PlatformShift) {
  editingShift.value = shift
  form.value = {
    name: shift.name,
    code: shift.code,
    start_time: shift.start_time,
    end_time: shift.end_time,
    duration: shift.duration,
    is_cross_day: shift.is_cross_day,
    color: shift.color,
    priority: shift.priority,
  }
  dialogVisible.value = true
}

async function submit() {
  if (!form.value.name.trim() || !form.value.code.trim()) {
    message.warning('请填写班次名称和编码')
    return
  }

  saving.value = true
  try {
    if (editingShift.value) {
      await updatePlatformShift(editingShift.value.id, form.value)
      message.success('班次已更新')
    }
    else {
      await createPlatformShift(form.value)
      message.success('班次已创建')
    }
    dialogVisible.value = false
    await loadShifts()
  }
  finally {
    saving.value = false
  }
}

function removeShift(shift: PlatformShift) {
  dialog.warning({
    title: '确认删除班次',
    content: `确定删除「${shift.name}」？`,
    positiveText: '删除',
    negativeText: '取消',
    async onPositiveClick() {
      await deletePlatformShift(shift.id)
      message.success('班次已删除')
      await loadShifts()
    },
  })
}

async function toggleStatus(shift: PlatformShift) {
  await togglePlatformShift(shift.id)
  message.success(`班次已${shift.status === 'active' ? '停用' : '启用'}`)
  await loadShifts()
}

onMounted(loadShifts)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">班次管理</h2>
          <p class="page-subtitle">统一维护平台侧班次模板，排班应用仅读取启用中的班次。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left" />
        <div class="toolbar-right">
          <n-button type="primary" @click="openCreate">新增班次</n-button>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <table class="admin-table">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>编码</th>
                  <th>时段</th>
                  <th>时长</th>
                  <th>优先级</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in shifts" :key="item.id">
                  <td>
                    <div>{{ item.name }}</div>
                    <div class="table-muted">{{ item.color }}</div>
                  </td>
                  <td>{{ item.code }}</td>
                  <td>{{ item.start_time }} - {{ item.end_time }}</td>
                  <td>{{ item.duration }} 分钟</td>
                  <td>{{ item.priority }}</td>
                  <td>
                    <n-tag :type="item.status === 'active' ? 'success' : 'default'" size="small">
                      {{ item.status === 'active' ? '启用' : '停用' }}
                    </n-tag>
                  </td>
                  <td>
                    <div class="table-actions">
                      <n-button text type="primary" @click="openEdit(item)">编辑</n-button>
                      <n-button text @click="toggleStatus(item)">{{ item.status === 'active' ? '停用' : '启用' }}</n-button>
                      <n-button text type="error" @click="removeShift(item)">删除</n-button>
                    </div>
                  </td>
                </tr>
                <tr v-if="!shifts.length">
                  <td colspan="7" class="table-empty">暂无班次数据</td>
                </tr>
              </tbody>
            </table>
          </n-spin>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" :title="editingShift ? '编辑班次' : '新增班次'" style="width: min(640px, calc(100vw - 32px))">
        <n-form :model="form" label-placement="left" label-width="96">
          <div class="form-grid two-column">
            <n-form-item label="名称">
              <n-input v-model:value="form.name" placeholder="输入班次名称" />
            </n-form-item>
            <n-form-item label="编码">
              <n-input v-model:value="form.code" :disabled="!!editingShift" placeholder="例如 day / night" />
            </n-form-item>
            <n-form-item label="开始时间">
              <n-input v-model:value="form.start_time" placeholder="08:00" />
            </n-form-item>
            <n-form-item label="结束时间">
              <n-input v-model:value="form.end_time" placeholder="17:00" />
            </n-form-item>
            <n-form-item label="时长(分钟)">
              <n-input-number v-model:value="form.duration" :min="1" style="width: 100%" />
            </n-form-item>
            <n-form-item label="优先级">
              <n-input-number v-model:value="form.priority" :min="0" style="width: 100%" />
            </n-form-item>
            <n-form-item label="颜色">
              <n-input v-model:value="form.color" placeholder="#0f766e" />
            </n-form-item>
            <n-form-item label="跨天班次">
              <n-switch v-model:value="form.is_cross_day" />
            </n-form-item>
          </div>
        </n-form>

        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" :loading="saving" @click="submit">保存</n-button>
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
}

.admin-table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.table-muted {
  color: var(--admin-text-muted);
  font-size: 12px;
}

.table-actions {
  display: flex;
  gap: 8px;
}

.table-empty {
  padding: 48px 0;
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