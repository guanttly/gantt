/**
 * DataViewDialog 组件类型定义
 * 用于展示查询数据的弹框组件
 */

/**
 * 数据展示模式
 */
export type DataViewMode = 'json' | 'table' | 'text' | 'auto'

/**
 * 数据视图配置
 */
export interface DataViewConfig {
  title: string // 弹框标题
  data: any // 要展示的数据
  mode?: DataViewMode // 展示模式，默认 'auto'
  maxHeight?: string // 最大高度，默认 '500px'
  showCopy?: boolean // 是否显示复制按钮，默认 true
  showExport?: boolean // 是否显示导出按钮，默认 false
}

/**
 * 表格列定义
 */
export interface TableColumn {
  prop: string // 属性名
  label: string // 列标签
  width?: string // 列宽度
  formatter?: (value: any) => string // 格式化函数
}

/**
 * 表格数据
 */
export interface TableData {
  columns: TableColumn[]
  rows: any[]
}

/**
 * 数据视图组件对外暴露的方法
 */
export interface DataViewDialogExpose {
  open: (config: DataViewConfig) => void
  close: () => void
}
