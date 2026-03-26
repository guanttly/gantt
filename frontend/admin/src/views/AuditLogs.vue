<script setup lang="ts">
import type { AuditLog } from '@/api/admin'
import { onMounted, ref, watch } from 'vue'
import { NInput, NPagination, NSpin, NTag, useMessage } from 'naive-ui'
import { listAuditLogs } from '@/api/admin'

const loading = ref(true)
const logs = ref<AuditLog[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const action = ref('')
const resourceType = ref('')
const userId = ref('')
const startDate = ref('')
const endDate = ref('')
const message = useMessage()

async function loadData() {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page: currentPage.value,
      size: pageSize.value,
    }
    if (action.value) params.action = action.value
    if (resourceType.value) params.resource_type = resourceType.value
    if (userId.value) params.user_id = userId.value
    if (startDate.value) params.start = startDate.value
    if (endDate.value) params.end = endDate.value

    const res = await listAuditLogs(params)
    logs.value = res.data
    total.value = res.total
    pageSize.value = res.size
  }
  catch {
    message.error('加载审计日志失败')
  }
  finally {
    loading.value = false
  }
}

function handleSearch() {
  currentPage.value = 1
  loadData()
}

watch(currentPage, loadData)
onMounted(loadData)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">审计日志</h2>
          <p class="page-subtitle">按用户、动作、资源类型和时间范围回溯平台级操作。</p>
        </div>
      </section>

      <section class="page-toolbar">
        <div class="toolbar-left">
          <n-input v-model:value="userId" placeholder="用户 ID" clearable style="width: 180px" @keyup.enter="handleSearch" @clear="handleSearch" />
          <n-input v-model:value="action" placeholder="操作类型" clearable style="width: 160px" @keyup.enter="handleSearch" @clear="handleSearch" />
          <n-input v-model:value="resourceType" placeholder="资源类型" clearable style="width: 160px" @keyup.enter="handleSearch" @clear="handleSearch" />
          <div class="date-range-fields">
            <input v-model="startDate" class="native-date-input" type="date" @change="handleSearch">
            <span class="date-range-separator">至</span>
            <input v-model="endDate" class="native-date-input" type="date" @change="handleSearch">
          </div>
        </div>
      </section>

      <section class="page-card">
        <div class="page-card-inner">
          <n-spin :show="loading">
            <template #description>
              正在加载审计日志
            </template>

            <div class="table-shell">
              <table class="data-table">
                <thead>
                  <tr>
                    <th>时间</th>
                    <th>用户</th>
                    <th>操作</th>
                    <th>资源类型</th>
                    <th>资源ID</th>
                    <th>状态码</th>
                    <th>IP 地址</th>
                    <th>详情</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in logs" :key="row.id">
                    <td>{{ row.created_at }}</td>
                    <td>{{ row.username }}</td>
                    <td>
                      <n-tag
                        size="small"
                        :bordered="false"
                        :type="row.action === 'delete' ? 'error' : row.action === 'create' ? 'success' : row.action === 'update' ? 'warning' : 'info'"
                      >
                        {{ row.action }}
                      </n-tag>
                    </td>
                    <td>{{ row.resource_type }}</td>
                    <td>{{ row.resource_id || '-' }}</td>
                    <td>{{ row.status_code }}</td>
                    <td>{{ row.ip }}</td>
                    <td class="detail-column">{{ row.detail ? JSON.stringify(row.detail) : '-' }}</td>
                  </tr>
                  <tr v-if="!logs.length">
                    <td colspan="8" class="table-empty">暂无审计日志</td>
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
    </div>
  </div>
</template>

<style scoped>
.toolbar-left { width: 100%; }

.date-range-fields {
  display: flex;
  align-items: center;
  gap: 10px;
}

.date-range-separator {
  color: var(--admin-text-muted);
  font-size: 13px;
}

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
  vertical-align: top;
}

.data-table thead th {
  background: #f8fafc;
  color: var(--admin-text-muted);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.data-table tbody tr:hover {
  background: rgba(15, 118, 110, 0.04);
}

.detail-column {
  min-width: 280px;
  max-width: 420px;
  white-space: normal;
  word-break: break-word;
}

.table-empty {
  padding: 48px 16px;
  color: var(--admin-text-muted);
  text-align: center;
}

.native-date-input {
  min-height: 40px;
  padding: 0 12px;
  border: 1px solid var(--admin-border);
  border-radius: 12px;
  color: var(--admin-text);
  background: #fff;
  font: inherit;
}
</style>
