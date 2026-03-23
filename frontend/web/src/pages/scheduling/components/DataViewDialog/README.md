# DataViewDialog 组件

数据查看弹框组件，用于展示查询数据，支持多种展示模式。

## 功能特性

- 🎨 **多种展示模式**: JSON、表格、文本三种展示方式
- 🤖 **自动识别**: 根据数据类型自动选择最佳展示模式
- 📋 **复制功能**: 一键复制数据到剪贴板
- 💾 **导出功能**: 支持导出为 JSON/CSV/TXT 文件
- 🎯 **响应式**: 自适应不同屏幕尺寸

## 使用方法

### 基础使用

```vue
<script setup lang="ts">
import type { DataViewDialogExpose } from './components/DataViewDialog/type'
import { ref } from 'vue'
import DataViewDialog from './components/DataViewDialog/index.vue'

const dataViewDialogRef = ref<DataViewDialogExpose | null>(null)

function showData() {
  dataViewDialogRef.value?.open({
    title: '数据详情',
    data: { name: 'John', age: 30 },
    mode: 'auto', // 自动检测
    showCopy: true,
    showExport: false,
  })
}
</script>

<template>
  <button @click="showData">
    查看数据
  </button>
  <DataViewDialog ref="dataViewDialogRef" />
</template>
```

### 展示模式

#### 1. JSON 模式 (`mode: 'json'`)
适合展示对象、嵌套数据结构：
```typescript
dataViewDialogRef.value?.open({
  title: 'JSON 数据',
  data: { user: { name: 'Alice', roles: ['admin', 'user'] } },
  mode: 'json',
})
```

#### 2. 表格模式 (`mode: 'table'`)
适合展示数组数据：
```typescript
dataViewDialogRef.value?.open({
  title: '用户列表',
  data: [
    { id: 1, name: 'Alice', age: 25 },
    { id: 2, name: 'Bob', age: 30 },
  ],
  mode: 'table',
})
```

#### 3. 文本模式 (`mode: 'text'`)
适合展示纯文本、字符串：
```typescript
dataViewDialogRef.value?.open({
  title: '文本内容',
  data: 'Hello, World!',
  mode: 'text',
})
```

#### 4. 自动模式 (`mode: 'auto'`)（推荐）
根据数据类型自动选择：
- 数组对象 → 表格模式
- 对象 → JSON 模式
- 其他 → 文本模式

```typescript
dataViewDialogRef.value?.open({
  title: '数据详情',
  data: someData,
  mode: 'auto', // 默认值
})
```

### 配置选项

```typescript
interface DataViewConfig {
  title: string // 弹框标题
  data: any // 要展示的数据
  mode?: DataViewMode // 展示模式: 'json' | 'table' | 'text' | 'auto'
  maxHeight?: string // 最大高度，默认 '500px'
  showCopy?: boolean // 是否显示复制按钮，默认 true
  showExport?: boolean // 是否显示导出按钮，默认 false
}
```

## 在 ChatAssistant 中的集成

ChatAssistant 组件已集成 DataViewDialog，用于处理 `query` 类型的操作：

```typescript
// logic.ts
function handleQueryAction(action: WorkflowAction) {
  if (dataViewDialogRef?.value) {
    dataViewDialogRef.value.open({
      title: action.label,
      data: action.payload || {},
      mode: 'auto',
      showCopy: true,
      showExport: false,
    })
  }
}
```

当后端返回 WorkflowAction 类型为 `query` 时，点击按钮会自动弹出 DataViewDialog 展示数据。

## 导出功能

导出功能会根据当前展示模式选择文件格式：

- JSON 模式 → 导出为 `.json` 文件
- 表格模式 → 导出为 `.csv` 文件（CSV 格式）
- 文本模式 → 导出为 `.txt` 文件

## 样式定制

组件使用 Element Plus 设计风格，可通过 CSS 变量进行主题定制：

```scss
.data-view-content {
  .json-view {
    font-family: 'Courier New', monospace;
    font-size: 13px;
    // 自定义样式...
  }
}
```

## 注意事项

1. **大数据量**: 表格模式限制最大高度为 400px，超出会显示滚动条
2. **复制功能**: 需要浏览器支持 Clipboard API（现代浏览器均支持）
3. **CSV 导出**: 处理包含逗号或换行的字段时自动添加引号转义
4. **降级方案**: 如果 DataViewDialog 未初始化，会降级使用 ElMessageBox
