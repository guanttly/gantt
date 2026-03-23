# 前端工作流交互实现文档

## 概述

本文档描述了前端如何与后端排班创建工作流进行交互，包括一键启动、状态显示、用户操作响应等功能。

## 架构设计

### 核心组件

1. **SchedulingToolbar** - 工具栏组件
   - 提供"一键启动排班"按钮
   - 用户点击后触发工作流

2. **ChatAssistant** - 智能助手组件
   - 显示工作流进度和状态
   - 展示AI消息和操作按钮
   - 处理用户交互（确认、修改、取消等）

3. **WorkflowProgress** - 进度显示组件
   - 可视化展示当前工作流阶段
   - 9个步骤的进度条

4. **Workflow API** - 后端接口封装
   - 触发工作流事件
   - 查询会话状态
   - 取消工作流

## 工作流事件定义

```typescript
// types/event.ts
enum WorkFlowEventType {
  // 排班创建工作流
  Schedule_Create_Start = '_start_',
  Schedule_Create_PeriodConfirmed = '_schedule_create_period_confirmed_',
  Schedule_Create_PeriodModified = '_schedule_create_period_modified_',
  Schedule_Create_ShiftsConfirmed = '_schedule_create_shifts_confirmed_',
  Schedule_Create_ShiftsModified = '_schedule_create_shifts_modified_',
  Schedule_Create_StaffCountConfirmed = '_schedule_create_staff_count_confirmed_',
  Schedule_Create_StaffCountModified = '_schedule_create_staff_count_modified_',
  Schedule_Create_DraftConfirmed = '_schedule_create_draft_confirmed_',
  Schedule_Create_DraftRejected = '_schedule_create_draft_rejected_',
  Schedule_Create_UserCancelled = '_schedule_create_user_cancelled_',
}
```

## 用户交互流程

### Phase 1: 启动和确认周期

#### 1.1 一键启动
用户点击"一键启动排班"按钮：

```typescript
// 在 scheduling/index.vue 中
function handleQuickStart() {
  showChatPanel.value = true // 打开助手面板

  nextTick(() => {
    chatAssistantRef.value.startScheduleCreationWorkflow({
      startDate: '2025-11-18', // 当前选择的日期范围
      endDate: '2025-11-24',
    })
  })
}
```

#### 1.2 后端响应
后端 `actScheduleCreateStart` 被触发：
- 提取参数（用户提供的日期或默认下周一到周日）
- 查询可用班次
- 构建确认消息
- 返回操作按钮：
  ```json
  {
    "description": "好的，我将为您安排 2025-11-18 至 2025-11-24 的排班。\n\n系统中共有 5 个班次可供选择。",
    "actions": [
      {
        "id": "confirm_period",
        "type": "workflow",
        "label": "确认周期",
        "event": "_schedule_create_period_confirmed_",
        "style": "primary",
        "payload": {
          "startDate": "2025-11-18",
          "endDate": "2025-11-24"
        }
      },
      {
        "id": "modify_period",
        "type": "workflow",
        "label": "修改周期",
        "event": "_schedule_create_period_modified_",
        "style": "secondary"
      },
      {
        "id": "cancel",
        "type": "workflow",
        "label": "取消",
        "event": "_schedule_create_user_cancelled_",
        "style": "secondary"
      }
    ]
  }
  ```

#### 1.3 前端显示
ChatAssistant 组件：
- 显示助手消息
- 渲染操作按钮
- 显示工作流进度（Phase: confirming_period）

#### 1.4 用户确认
用户点击"确认周期"按钮：

```typescript
// ChatAssistant/logic.ts
async function handleWorkflowAction(action: WorkflowAction) {
  wsClient.sendWorkflowCommand(
    sessionStore.currentSessionId,
    action.event, // '_schedule_create_period_confirmed_'
    action.payload,
  )
}
```

#### 1.5 状态转换
后端 `actScheduleCreateConfirmPeriod` 被触发：
- 验证周期参数
- 构建班次列表信息
- 转换状态：`StateConfirmingPeriod` → `StateConfirmingShifts`
- 返回新的操作按钮（确认班次、修改班次）

### Phase 2-8: 后续阶段

类似的交互模式：
1. 后端返回消息 + 操作按钮
2. 前端显示消息和按钮
3. 用户点击按钮
4. 前端发送事件到后端
5. 后端处理并转换状态
6. 循环直到完成

## API 调用示例

### 触发工作流事件

```typescript
import { triggerWorkflowEvent } from '@/api/workflow'

// 启动排班创建
await triggerWorkflowEvent({
  sessionId: 'session-123',
  event: '_start_',
  payload: {
    startDate: '2025-11-18',
    endDate: '2025-11-24',
    orgId: 'org-001',
  },
})

// 确认周期
await triggerWorkflowEvent({
  sessionId: 'session-123',
  event: '_schedule_create_period_confirmed_',
  payload: {
    startDate: '2025-11-18',
    endDate: '2025-11-24',
  },
})
```

### 查询会话状态

```typescript
import { getSessionStatus } from '@/api/workflow'

const status = await getSessionStatus('session-123')
console.log(status.workflowMeta) // { workflow, phase, actions, ... }
```

### 取消工作流

```typescript
import { cancelWorkflow } from '@/api/workflow'

await cancelWorkflow('session-123', '用户主动取消')
```

## 工作流进度可视化

WorkflowProgress 组件自动显示当前阶段：

```vue
<WorkflowProgress
  :workflow="sessionStore.workflow?.workflow"
  :current-phase="sessionStore.workflow?.phase"
/>
```

9个步骤：
1. 📅 确认周期 (confirming_period)
2. ⏰ 选择班次 (confirming_shifts)
3. 👥 确认人数 (confirming_staff_count)
4. 🔍 检索人员 (retrieving_staff)
5. 📋 加载规则 (retrieving_rules)
6. ⚙️ 生成排班 (generating_schedule)
7. 👁️ 预览草案 (previewing_draft)
8. 💾 保存排班 (saving_schedule)
9. ✅ 完成 (completed)

## 操作按钮样式映射

```typescript
const styleMapping = {
  primary: 'primary', // 主要操作（确认）
  secondary: 'default', // 次要操作（修改）
  success: 'success', // 成功操作（完成）
  danger: 'danger', // 危险操作（删除、取消）
  warning: 'warning', // 警告操作
  info: 'info', // 信息操作（查看）
  link: 'primary', // 链接样式
}
```

## WebSocket 消息格式

### 上行消息（前端→后端）

```json
{
  "type": "workflow_command",
  "sessionId": "session-123",
  "data": {
    "command": "_schedule_create_period_confirmed_",
    "payload": {
      "startDate": "2025-11-18",
      "endDate": "2025-11-24"
    }
  },
  "ts": "2025-11-13T10:00:00Z",
  "seq": 1
}
```

### 下行消息（后端→前端）

```json
{
  "type": "workflow_update",
  "sessionId": "session-123",
  "data": {
    "workflow": "schedule_create",
    "phase": "confirming_shifts",
    "description": "已确认排班周期：2025-11-18 至 2025-11-24\n\n系统中共有 5 个班次...",
    "actions": [
      {
        "id": "confirm_shifts",
        "type": "workflow",
        "label": "确认班次",
        "event": "_schedule_create_shifts_confirmed_",
        "style": "primary"
      }
    ]
  },
  "ts": "2025-11-13T10:00:01Z",
  "eventId": 100
}
```

## 调试和测试

### 开发环境配置

```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_HOST=localhost:8080
```

### 测试流程

1. 启动后端服务
   ```bash
   cd agents/rostering
   go run cmd/services/management-service/main.go
   ```

2. 启动前端开发服务器
   ```bash
   cd frontend/web
   pnpm dev
   ```

3. 打开浏览器访问 http://localhost:5173/scheduling

4. 点击"一键启动排班"按钮

5. 观察：
   - WebSocket 连接状态
   - 控制台日志
   - 工作流进度更新
   - 操作按钮显示

### 常见问题

**Q: 点击按钮后没有反应？**
- 检查 WebSocket 连接状态
- 查看浏览器控制台错误
- 确认后端服务正常运行

**Q: 工作流状态不更新？**
- 检查 sessionStore.workflow 的值
- 确认 WebSocket 消息正确接收
- 验证 phase 字段匹配

**Q: 操作按钮不显示？**
- 检查 actions 数组是否为空
- 确认 WorkflowMeta 正确传递
- 查看 ChatAssistant 组件的 watch 逻辑

## 扩展功能

### 添加自定义操作类型

```typescript
// 1. 定义新的操作类型
export type WorkflowActionType = 'workflow' | 'query' | 'command' | 'navigate' | 'custom'

// 2. 实现处理器
function handleCustomAction(action: WorkflowAction) {
  // 自定义逻辑
}

// 3. 注册处理器
const actionHandlers: ActionHandlers = {
  // ...existing handlers
  custom: handleCustomAction,
}
```

### 添加新的工作流步骤

```typescript
// WorkflowProgress.vue
const scheduleCreationSteps: WorkflowStep[] = [
  // ...existing steps
  {
    key: 'new_step',
    label: '新步骤',
    icon: '🆕',
    description: '这是一个新的工作流步骤',
  },
]
```

## 总结

前端工作流交互实现提供了：
- ✅ 一键启动排班功能
- ✅ 可视化进度显示
- ✅ 动态操作按钮
- ✅ WebSocket 实时通信
- ✅ 完整的错误处理
- ✅ 类型安全的API

用户体验流畅，代码结构清晰，易于维护和扩展。
