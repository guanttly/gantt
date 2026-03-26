<script setup lang="ts">
import type { Subscription } from '@/api/admin'
import { computed, onMounted, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NModal, NPagination, NSelect, NSpin, NTag, useMessage } from 'naive-ui'
import { listSubscriptions, updateSubscription } from '@/api/admin'

const loading = ref(true)
const subscriptions = ref<Subscription[]>([])
const planFilter = ref('')
const statusFilter = ref('')
const keyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const message = useMessage()

const dialogVisible = ref(false)
const editForm = ref<Partial<Subscription>>({})

const filteredSubscriptions = computed(() => {
  const normalized = keyword.value.trim().toLowerCase()
  if (!normalized) return subscriptions.value
  return subscriptions.value.filter(item => {
    return [item.org_name, item.org_node_id, item.plan, item.status].some(field => field?.toLowerCase().includes(normalized))
  })
})

async function loadData() {
  loading.value = true
  try {
    const res = await listSubscriptions({
      page: currentPage.value,
      size: pageSize.value,
      plan: planFilter.value || undefined,
      status: statusFilter.value || undefined,
    })
    subscriptions.value = res.data
    total.value = res.total
    pageSize.value = res.size
  }
  catch {
    message.error('加载订阅数据失败')
  }
  finally {
    loading.value = false
  }
}

function handleEdit(row: Subscription) {
  editForm.value = { ...row }
  dialogVisible.value = true
}

async function handleSave() {
  if (!editForm.value.id) return
  try {
    await updateSubscription(editForm.value.id, {
      plan: editForm.value.plan,
      end_date: editForm.value.end_date,
      status: editForm.value.status,
    })
    message.success('更新成功')
    dialogVisible.value = false
    await loadData()
  }
  catch (e: any) {
    message.error(e?.response?.data?.message || '更新失败')
  }
}

function handleSearch() {
  currentPage.value = 1
  loadData()
}

const planMap: Record<string, { label: string, type: string }> = {
  free: { label: '免费版', type: 'info' },
  standard: { label: '标准版', type: 'default' },
  premium: { label: '高级版', type: 'success' },
}

const statusMap: Record<Subscription['status'], { label: string, type: string }> = {
  active: { label: '有效', type: 'success' },
  expired: { label: '过期', type: 'warning' },
  cancelled: { label: '已取消', type: 'info' },
}

onMounted(loadData)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">订阅管理</h2>
          <p class="page-subtitle">按套餐与状态筛选平台订阅，可在当前页做轻量本地搜索。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-select
            v-model:value="planFilter"
            clearable
            placeholder="筛选套餐"
            style="width: 160px"
            :options="[
              { label: '免费版', value: 'free' },
              { label: '标准版', value: 'standard' },
              { label: '高级版', value: 'premium' },
            ]"
            @update:value="handleSearch"
          />
          <n-select
            v-model:value="statusFilter"
            clearable
            placeholder="筛选状态"
            style="width: 160px"
            :options="[
              { label: '有效', value: 'active' },
              { label: '过期', value: 'expired' },
              { label: '已取消', value: 'cancelled' },
            ]"
            @update:value="handleSearch"
          />
        </div>
        <div class="toolbar-right">
          <n-input v-model:value="keyword" clearable placeholder="当前页搜索组织/套餐/状态" style="width: 260px" />
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <template #description>
              正在加载订阅数据
            </template>

            <div class="table-shell">
              <table class="data-table">
                <thead>
                  <tr>
                    <th>组织名称</th>
                    <th>订阅套餐</th>
                    <th>状态</th>
                    <th>开始日期</th>
                    <th>到期日期</th>
                    <th class="actions-column">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in filteredSubscriptions" :key="row.id">
                    <td>{{ row.org_name }}</td>
                    <td>
                      <n-tag size="small" :bordered="false" :type="planMap[row.plan]?.type as any">
                        {{ planMap[row.plan]?.label || row.plan }}
                      </n-tag>
                    </td>
                    <td>
                      <n-tag size="small" :bordered="false" :type="statusMap[row.status]?.type as any">
                        {{ statusMap[row.status]?.label || row.status }}
                      </n-tag>
                    </td>
                    <td>{{ row.start_date }}</td>
                    <td>{{ row.end_date }}</td>
                    <td class="actions-column">
                      <n-button text type="primary" @click="handleEdit(row)">编辑</n-button>
                    </td>
                  </tr>
                  <tr v-if="!filteredSubscriptions.length">
                    <td colspan="6" class="table-empty">暂无订阅数据</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </n-spin>

          <div class="page-pagination">
            <n-pagination
              :page="currentPage"
              :page-size="pageSize"
              :item-count="total"
              :page-sizes="[20, 50, 100]"
              show-size-picker
              @update:page="(page) => { currentPage = page; loadData() }"
              @update:page-size="(size) => { pageSize = size; handleSearch() }"
            />
          </div>
        </div>
      </section>

      <n-modal v-model:show="dialogVisible" preset="card" title="编辑订阅" style="width: min(420px, calc(100vw - 32px))">
        <n-form :model="editForm" label-placement="left" label-width="100">
          <n-form-item label="订阅套餐">
            <n-select
              v-model:value="editForm.plan"
              :options="[
                { label: '免费版', value: 'free' },
                { label: '标准版', value: 'standard' },
                { label: '高级版', value: 'premium' },
              ]"
            />
          </n-form-item>
          <n-form-item label="状态">
            <n-select
              v-model:value="editForm.status"
              :options="[
                { label: '有效', value: 'active' },
                { label: '过期', value: 'expired' },
                { label: '已取消', value: 'cancelled' },
              ]"
            />
          </n-form-item>
          <n-form-item label="到期日期">
            <input v-model="editForm.end_date" class="native-date-input" type="date">
          </n-form-item>
        </n-form>

        <template #footer>
          <div class="modal-actions">
            <n-button @click="dialogVisible = false">取消</n-button>
            <n-button type="primary" @click="handleSave">保存</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.table-shell {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th,
.data-table td {
  padding: 14px 16px;
  border-bottom: 1px solid var(--admin-border);
  text-align: left;
  white-space: nowrap;
}

.data-table thead th {
  background: #f8fafc;
  color: var(--admin-text-muted);
  font-size: 13px;
  font-weight: 600;
}

.data-table tbody tr:hover {
  background: rgba(15, 118, 110, 0.04);
}

.actions-column {
  width: 88px;
}

.table-empty {
  padding: 48px 16px;
  color: var(--admin-text-muted);
  text-align: center;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.native-date-input {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
  border: 1px solid var(--admin-border);
  border-radius: 12px;
  color: var(--admin-text);
  font: inherit;
}
</style>
