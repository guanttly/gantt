# ChatAssistant 组件

智能排班助手聊天组件，支持多种交互类型。

## 文件结构

```
ChatAssistant/
├── index.vue       # Vue 组件（UI 和模板）
├── logic.ts        # 业务逻辑（Hooks）
├── type.ts         # 类型定义
└── README.md       # 组件文档
```

## 功能特性

### 四种 Action 类型支持

1. **workflow** - 工作流状态变更
   - 添加灰色系统消息显示状态迁移
   - 触发后端工作流事件

2. **query** - 查询操作
   - 弹框显示查询结果数据
   - 不改变工作流状态

3. **command** - 命令交互
   - 可选用户输入
   - 发送命令和消息到后端

4. **navigate** - 页面导航
   - 跳转到指定 URL

## 使用示例

```vue
<script setup lang="ts">
import ChatAssistant from './components/ChatAssistant/index.vue'

const showChat = ref(false)

function toggleChat() {
  showChat.value = !showChat.value
}
</script>

<template>
  <ChatAssistant v-if="showChat" @close="toggleChat" />
</template>
```

## 与后端集成

### 消息格式

```typescript
interface WorkflowAction {
  id: string
  type: 'workflow' | 'query' | 'command' | 'navigate'
  label: string
  event: string
  style?: 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' | 'link'
  payload?: Record<string, any>
}
```

### WebSocket 消息

组件通过 `schedulingSession` store 与后端 WebSocket 通信：

- 发送用户消息
- 接收助手响应
- 监听工作流状态变更
- 更新可用操作按钮

## 开发说明

### 添加新的 Action 类型

1. 在 `type.ts` 中扩展 `ActionHandlers` 接口
2. 在 `logic.ts` 中实现对应的 handler 函数
3. 在 `actionHandlers` 对象中注册 handler

### 自定义样式

修改 `index.vue` 中的 `<style>` 部分，支持：
- 消息气泡样式
- 按钮样式
- 系统通知样式

## TODO

- [ ] 集成真实的 WebSocket 连接
- [ ] 优化查询数据展示（使用专用弹框组件）
- [ ] 添加消息已读/未读状态
- [ ] 支持富文本消息内容
- [ ] 添加历史消息加载
