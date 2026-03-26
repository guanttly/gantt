/**
 * DataViewDialog 组件类型定义
 */

/** 数据展示模式 */
export type DataViewMode = 'json' | 'table' | 'text' | 'auto'

/** 数据视图配置 */
export interface DataViewConfig {
  title: string
  data: any
  mode?: DataViewMode
  maxHeight?: string
  showCopy?: boolean
  showExport?: boolean
}

/** 表格列定义 */
export interface TableColumn {
  prop: string
  label: string
  width?: string
  formatter?: (value: any) => string
}

/** 表格数据 */
export interface TableData {
  columns: TableColumn[]
  rows: any[]
}

/** 组件对外暴露方法 */
export interface DataViewDialogExpose {
  open: (config: DataViewConfig) => void
  close: () => void
}
