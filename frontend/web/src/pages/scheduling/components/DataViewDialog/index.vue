<script setup lang="ts">
import type { DataViewDialogExpose } from './type'
import { CopyDocument, Download } from '@element-plus/icons-vue'
import { useDataViewDialog } from './logic'

// 使用业务逻辑 Hook
const {
  visible,
  config,
  actualMode,
  formattedJson,
  tableData,
  textData,
  open,
  close,
  copyToClipboard,
  exportData,
} = useDataViewDialog()

// 对外暴露方法
defineExpose<DataViewDialogExpose>({
  open,
  close,
})
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="config.title"
    width="800px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <div class="data-view-content" :style="{ maxHeight: config.maxHeight }">
      <!-- JSON 模式 -->
      <pre v-if="actualMode === 'json'" class="json-view">{{ formattedJson }}</pre>

      <!-- 表格模式 -->
      <el-table
        v-else-if="actualMode === 'table'"
        :data="tableData.rows"
        border
        stripe
        style="width: 100%"
        max-height="400"
      >
        <el-table-column
          v-for="col in tableData.columns"
          :key="col.prop"
          :prop="col.prop"
          :label="col.label"
          :width="col.width"
        />
      </el-table>

      <!-- 文本模式 -->
      <div v-else class="text-view">
        {{ textData }}
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <div class="footer-left">
          <el-button
            v-if="config.showCopy"
            :icon="CopyDocument"
            @click="copyToClipboard"
          >
            复制
          </el-button>
          <el-button
            v-if="config.showExport"
            :icon="Download"
            @click="exportData"
          >
            导出
          </el-button>
        </div>
        <el-button @click="close">
          关闭
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.data-view-content {
  overflow: auto;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;

  .json-view {
    margin: 0;
    padding: 16px;
    background: #ffffff;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    font-family: 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #303133;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .text-view {
    padding: 16px;
    background: #ffffff;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    font-size: 14px;
    line-height: 1.8;
    color: #303133;
    white-space: pre-wrap;
    word-break: break-word;
  }
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .footer-left {
    display: flex;
    gap: 8px;
  }
}
</style>
