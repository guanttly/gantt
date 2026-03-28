<script setup lang="ts">
import type { SchedulePlan } from '@/api/schedules'
import { Delete, Plus, Search, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { deleteSchedule, listSchedules } from '@/api/schedules'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const canCreateSchedule = computed(() => auth.hasPermission('schedule:create'))

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<SchedulePlan>({
  fetchFn: listSchedules,
})

const statusMap: Record<string, { label: string, type: string }> = {
  draft: { label: '草稿', type: 'info' },
  generating: { label: '生成中', type: 'warning' },
  generated: { label: '已生成', type: '' },
  published: { label: '已发布', type: 'success' },
}

async function handleDelete(row: SchedulePlan) {
  await ElMessageBox.confirm(`确定删除排班计划「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteSchedule(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <el-input v-model="keyword" placeholder="搜索排班计划" clearable style="width: 240px" :prefix-icon="Search" />
      <el-button v-if="canCreateSchedule" type="primary" :icon="Plus" @click="router.push('/scheduling/create')">
        创建排班
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe class="schedule-table">
      <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
      <el-table-column prop="start_date" label="开始日期" min-width="140" />
      <el-table-column prop="end_date" label="结束日期" min-width="140" />
      <el-table-column prop="status" label="状态" min-width="140">
        <template #default="{ row }">
          <el-tag :type="(statusMap[row.status]?.type as any)" size="small">
            {{ statusMap[row.status]?.label || row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" min-width="200" />
      <el-table-column label="操作" min-width="160" align="right">
        <template #default="{ row }">
          <el-button :icon="View" link type="primary" @click="router.push(`/scheduling/${row.id}`)">
            查看
          </el-button>
          <el-button :icon="Delete" link type="danger" :disabled="row.status === 'published'" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="page-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="currentPageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow: hidden;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.schedule-table {
  width: 100%;
}

.schedule-table :deep(.el-table__empty-block) {
  width: 100% !important;
}
</style>
