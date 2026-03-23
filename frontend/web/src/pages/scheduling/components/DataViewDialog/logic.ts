import type { DataViewConfig, DataViewMode, TableColumn, TableData } from './type'
import { ElMessage } from 'element-plus'
import { computed, ref } from 'vue'

/**
 * DataViewDialog 业务逻辑 Hook
 */
export function useDataViewDialog() {
  // 状态
  const visible = ref(false)
  const config = ref<DataViewConfig>({
    title: '数据详情',
    data: null,
    mode: 'auto',
    maxHeight: '500px',
    showCopy: true,
    showExport: false,
  })

  /**
   * 自动检测数据类型并选择展示模式
   */
  function detectDataMode(data: any): DataViewMode {
    if (data === null || data === undefined)
      return 'text'

    // 数组且包含对象 -> 表格
    if (Array.isArray(data) && data.length > 0 && typeof data[0] === 'object')
      return 'table'

    // 对象 -> JSON
    if (typeof data === 'object')
      return 'json'

    // 其他 -> 文本
    return 'text'
  }

  /**
   * 计算实际使用的展示模式
   */
  const actualMode = computed<DataViewMode>(() => {
    if (config.value.mode === 'auto') {
      return detectDataMode(config.value.data)
    }
    return config.value.mode || 'text'
  })

  /**
   * 格式化 JSON 数据
   */
  const formattedJson = computed(() => {
    try {
      return JSON.stringify(config.value.data, null, 2)
    }
    catch (error) {
      console.error('JSON format error:', error)
      return String(config.value.data)
    }
  })

  /**
   * 转换数据为表格格式
   */
  const tableData = computed<TableData>(() => {
    const data = config.value.data

    if (!Array.isArray(data) || data.length === 0) {
      return {
        columns: [],
        rows: [],
      }
    }

    // 从第一行数据提取列信息
    const firstRow = data[0]
    const columns: TableColumn[] = Object.keys(firstRow).map(key => ({
      prop: key,
      label: key,
      width: 'auto',
    }))

    return {
      columns,
      rows: data,
    }
  })

  /**
   * 文本数据
   */
  const textData = computed(() => {
    if (config.value.data === null || config.value.data === undefined)
      return '无数据'

    if (typeof config.value.data === 'string')
      return config.value.data

    return String(config.value.data)
  })

  /**
   * 打开弹框
   */
  function open(newConfig: DataViewConfig) {
    config.value = {
      ...config.value,
      ...newConfig,
    }
    visible.value = true
  }

  /**
   * 关闭弹框
   */
  function close() {
    visible.value = false
  }

  /**
   * 复制到剪贴板
   */
  async function copyToClipboard() {
    try {
      let text = ''

      switch (actualMode.value) {
        case 'json':
          text = formattedJson.value
          break
        case 'table':
          // 表格转 CSV 格式
          text = tableDataToCsv(tableData.value)
          break
        case 'text':
          text = textData.value
          break
      }

      await navigator.clipboard.writeText(text)
      ElMessage.success('已复制到剪贴板')
    }
    catch (error) {
      console.error('Copy failed:', error)
      ElMessage.error('复制失败')
    }
  }

  /**
   * 导出数据
   */
  function exportData() {
    try {
      let content = ''
      let filename = 'data.txt'

      switch (actualMode.value) {
        case 'json':
          content = formattedJson.value
          filename = 'data.json'
          break
        case 'table':
          content = tableDataToCsv(tableData.value)
          filename = 'data.csv'
          break
        case 'text':
          content = textData.value
          filename = 'data.txt'
          break
      }

      // 创建下载链接
      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      link.click()
      URL.revokeObjectURL(url)

      ElMessage.success('导出成功')
    }
    catch (error) {
      console.error('Export failed:', error)
      ElMessage.error('导出失败')
    }
  }

  /**
   * 表格数据转 CSV
   */
  function tableDataToCsv(data: TableData): string {
    if (data.rows.length === 0)
      return ''

    const headers = data.columns.map(col => col.label).join(',')
    const rows = data.rows.map((row) => {
      return data.columns.map((col) => {
        const value = row[col.prop]
        // 处理包含逗号或换行的值
        if (typeof value === 'string' && (value.includes(',') || value.includes('\n'))) {
          return `"${value.replace(/"/g, '""')}"`
        }
        return value
      }).join(',')
    }).join('\n')

    return `${headers}\n${rows}`
  }

  return {
    // 状态
    visible,
    config,
    actualMode,
    formattedJson,
    tableData,
    textData,

    // 方法
    open,
    close,
    copyToClipboard,
    exportData,
  }
}
