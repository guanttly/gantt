<script setup lang="ts">
import type { DataViewConfig } from './type'
import { ElMessage } from 'element-plus'
import { computed, ref } from 'vue'

const visible = ref(false)
const config = ref<DataViewConfig>({
  title: '',
  data: null,
  mode: 'auto',
  maxHeight: '500px',
  showCopy: true,
  showExport: false,
})

const displayMode = computed(() => {
  if (config.value.mode !== 'auto')
    return config.value.mode
  const data = config.value.data
  if (Array.isArray(data))
    return 'table'
  if (typeof data === 'object' && data !== null)
    return 'json'
  return 'text'
})

const formattedData = computed(() => {
  if (displayMode.value === 'json')
    return JSON.stringify(config.value.data, null, 2)
  if (displayMode.value === 'text')
    return String(config.value.data)
  return ''
})

const tableColumns = computed(() => {
  if (displayMode.value !== 'table' || !Array.isArray(config.value.data) || config.value.data.length === 0)
    return []
  const firstRow = config.value.data[0]
  return Object.keys(firstRow).map(key => ({
    prop: key,
    label: key,
  }))
})

function open(cfg: DataViewConfig) {
  config.value = { ...config.value, ...cfg }
  visible.value = true
}

function close() {
  visible.value = false
}

async function handleCopy() {
  try {
    const text = displayMode.value === 'json'
      ? JSON.stringify(config.value.data, null, 2)
      : String(config.value.data)
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  }
  catch {
    ElMessage.error('复制失败')
  }
}

defineExpose({ open, close })
</script>

<template>
  <el-dialog v-model="visible" :title="config.title" width="70%" destroy-on-close>
    <div :style="{ maxHeight: config.maxHeight, overflow: 'auto' }">
      <!-- JSON 模式 -->
      <pre v-if="displayMode === 'json'" class="data-view-json">{{ formattedData }}</pre>

      <!-- 表格模式 -->
      <el-table v-else-if="displayMode === 'table'" :data="config.data" border stripe size="small">
        <el-table-column
          v-for="col in tableColumns"
          :key="col.prop"
          :prop="col.prop"
          :label="col.label"
          min-width="120"
        />
      </el-table>

      <!-- 文本模式 -->
      <pre v-else class="data-view-text">{{ formattedData }}</pre>
    </div>

    <template #footer>
      <el-button v-if="config.showCopy" @click="handleCopy">
        复制
      </el-button>
      <el-button @click="close">
        关闭
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.data-view-json,
.data-view-text {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
