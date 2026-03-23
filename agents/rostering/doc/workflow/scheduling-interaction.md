# 排班工作流前后端交互完整示例

本文档以**排班创建工作流**为例，详细说明 FSM 如何处理前后端交互，包括消息流转、状态管理、按钮交互等完整流程。

## 📖 目录

1. [工作流概述](#工作流概述)
2. [状态流转图](#状态流转图)
3. [前后端交互协议](#前后端交互协议)
4. [完整交互流程示例](#完整交互流程示例)
5. [数据结构说明](#数据结构说明)
6. [错误处理](#错误处理)

---

## 工作流概述

### 排班创建工作流简介

排班创建工作流（`Workflow_Schedule_Create`）用于处理用户创建新排班的请求。该工作流涉及多个步骤：

1. **信息验证**：检查排班创建所需的必要信息（如开始日期、结束日期等）
2. **数据收集**：查询人员信息、班次列表、排班规则、历史排班等
3. **AI 生成**：基于收集的数据和规则，AI 生成排班初稿
4. **用户确认**：展示初稿给用户，等待确认或调整
5. **执行创建**：用户确认后，执行实际的排班创建操作
6. **完成/失败**：返回最终结果

### 关键特性

- ✅ **多步骤异步处理**：每个步骤异步执行，不阻塞用户
- ✅ **智能信息补充**：缺少信息时主动询问用户
- ✅ **多轮草案调整**：支持用户多次调整排班方案
- ✅ **版本控制**：通过 `DraftVersion` 防止并发冲突
- ✅ **完整日志追踪**：决策日志记录每次状态变化

---

## 状态流转图

### 完整状态图

```
┌──────────────────┐
│  用户发起排班请求  │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────┐
│ State_Schedule_Executing │ ← 通用执行状态
└────────┬────────────────┘
         │ Event_Schedule_Create
         ▼
┌──────────────────────────────┐
│ State_Create_ValidatingInfo  │ 验证信息完整性
└────────┬─────────────────────┘
         │
         ├─ ✅ 信息完整 (Event_InfoValidated)
         │     │
         │     ▼
         │  ┌──────────────────────────┐
         │  │ State_Create_QueryingStaff│ Step 1: 查询人员
         │  └────────┬─────────────────┘
         │           │ Event_StaffQueried
         │           ▼
         │  ┌───────────────────────────┐
         │  │ State_Create_QueryingShifts│ Step 1.5: 查询班次
         │  └────────┬──────────────────┘
         │           │ Event_ShiftsQueried
         │           ▼
         │  ┌──────────────────────────┐
         │  │ State_Create_QueryingRules│ Step 2: 查询规则
         │  └────────┬─────────────────┘
         │           │ Event_RulesQueried
         │           ▼
         │  ┌────────────────────────────┐
         │  │State_Create_QueryingHistory│ Step 3: 查询历史
         │  └────────┬───────────────────┘
         │           │ Event_HistoryQueried
         │           ▼
         │  ┌─────────────────────────────────┐
         │  │State_Create_QueryingStaffRules  │ Step 4: 查询人员规则
         │  └────────┬────────────────────────┘
         │           │ Event_StaffRulesQueried
         │           ▼
         │  ┌─────────────────────────────┐
         │  │State_Create_GeneratingDraft │ Step 5: AI 生成初稿
         │  └────────┬────────────────────┘
         │           │ Event_DraftGenerated
         │           ▼
         │  ┌──────────────────────────────┐
         │  │State_Create_ConfirmingDraft  │◄─────┐ 等待用户确认
         │  └────────┬─────────────────────┘      │
         │           │                             │
         │           ├─ ✅ 用户确认 (Event_UserConfirmed)
         │           │     │
         │           │     ▼
         │           │  ┌──────────────────────────┐
         │           │  │ State_Create_Executing   │ 执行创建
         │           │  └────────┬─────────────────┘
         │           │           │
         │           │           ├─ ✅ Event_Success
         │           │           │     │
         │           │           │     ▼
         │           │           │  ┌─────────────────────┐
         │           │           │  │ State_Create_Completed│ 完成 ✅
         │           │           │  └─────────────────────┘
         │           │           │
         │           │           └─ ❌ Event_Failure
         │           │                 │
         │           │                 ▼
         │           │              ┌──────────────────┐
         │           │              │State_Create_Failed│ 失败 ❌
         │           │              └──────────────────┘
         │           │
         │           ├─ 🔄 用户调整 (Event_UserRequestAdjust)
         │           │     │
         │           │     ▼
         │           │  ┌────────────────────────────┐
         │           │  │State_Create_AdjustingDraft │ 调整草案
         │           │  └────────┬───────────────────┘
         │           │           │ Event_DraftAdjusted
         │           │           └─────────────────────┘
         │           │                (重新生成)
         │           │
         │           └─ ❌ 用户取消 (Event_UserCancelled)
         │                 │
         │                 ▼
         │              ┌──────────────────┐
         │              │State_Create_Failed│ 取消 ❌
         │              └──────────────────┘
         │
         └─ ❌ 信息不完整 (Event_InfoIncomplete)
               │
               ▼
         ┌────────────────────────────────┐
         │State_Create_WaitingSupplementary│ 等待用户补充
         └────────┬───────────────────────┘
                  │ Event_UserSupplementary
                  │ (循环处理补充信息)
                  └─────────────────────────┐
                        Event_UserSupplemented
                        (重新验证)
```

### 关键状态说明

| 状态 | 说明 | 前端展示 |
|------|------|----------|
| `State_Create_ValidatingInfo` | 验证信息完整性 | "正在验证排班信息..." |
| `State_Create_WaitingSupplementary` | 等待用户补充信息 | "请补充以下信息：..." |
| `State_Create_QueryingStaff` | 查询部门人员 | "正在查询人员信息...（步骤 1/5）" |
| `State_Create_QueryingShifts` | 查询班次列表 | "正在查询班次信息...（步骤 1.5/5）" |
| `State_Create_QueryingRules` | 查询排班规则 | "正在查询排班规则...（步骤 2/5）" |
| `State_Create_QueryingHistory` | 查询历史排班 | "正在查询历史排班...（步骤 3/5）" |
| `State_Create_QueryingStaffRules` | 查询人员规则 | "正在查询人员规则...（步骤 4/5）" |
| `State_Create_GeneratingDraft` | AI 生成排班初稿 | "正在生成排班方案...（步骤 5/5）" |
| `State_Create_ConfirmingDraft` | 等待用户确认 | 展示初稿 + 按钮（确认/调整/取消） |
| `State_Create_AdjustingDraft` | 调整排班草案 | "正在调整排班方案..." |
| `State_Create_Executing` | 执行排班创建 | "正在创建排班..." |
| `State_Create_Completed` | 创建完成 | "排班创建成功！" |
| `State_Create_Failed` | 创建失败 | "排班创建失败：{原因}" |

---

## 前后端交互协议

### WebSocket 消息类型

#### 1. 前端 → 后端

##### 1.1 用户文本消息

```typescript
// 用户输入自然语言
{
  type: "user_message",
  sessionId: "session_123",
  message: "帮我创建下周的排班"
}
```

**后端处理**：
- 通过 AI 意图识别解析用户意图
- 发送 `Event_Schedule_Create` 事件到 FSM

##### 1.2 工作流命令

```typescript
// 用户点击按钮
{
  type: "workflow_command",
  sessionId: "session_123",
  command: "confirm",  // 或 "cancel", "adjust"
  payload: {
    expectedDraftVersion: 3,  // 可选：版本号
    adjustments: {            // 可选：调整参数
      addShift: { date: "2025-10-25", staffId: "staff_1", shiftId: "shift_morning" },
      removeShiftDates: ["2025-10-26"]
    }
  }
}
```

**后端处理**：
- 将命令映射为 FSM 事件：
  - `confirm` → `Event_Schedule_Create_UserConfirmed`
  - `cancel` → `Event_Schedule_Create_UserCancelled`
  - `adjust` → `Event_Schedule_Create_UserRequestAdjust`

##### 1.3 补充信息

```typescript
// 用户补充缺失的信息
{
  type: "supplementary_info",
  sessionId: "session_123",
  data: {
    startDate: "2025-10-27",
    endDate: "2025-11-02"
  }
}
```

**后端处理**：
- 发送 `Event_Action_UserSupplementary` 事件
- 更新 `ScheduleCreateContext.Intent.Arguments`
- 重新验证信息完整性

#### 2. 后端 → 前端

##### 2.1 助手消息

```typescript
{
  type: "assistant_message",
  sessionId: "session_123",
  message: "正在查询人员信息...（步骤 1/5）",
  timestamp: "2025-10-24T10:30:00Z"
}
```

**前端处理**：
- 在聊天界面展示消息
- 可选：展示进度条（如果消息包含步骤信息）

##### 2.2 工作流状态更新

```typescript
{
  type: "workflow_update",
  sessionId: "session_123",
  workflowMeta: {
    workflow: "scheduling",
    state: "_state_schedule_create_confirming_draft_",
    phase: "create_confirming_draft",
    phaseDesc: "待确认排班初稿",
    actions: [
      { action: "confirm", label: "确认", type: "primary" },
      { action: "cancel", label: "取消", type: "default" },
      { action: "adjust", label: "调整", type: "link" }
    ],
    actionsVersion: 5,
    extra: {
      draftVersion: 3,
      currentStep: 5,
      totalSteps: 5,
      scheduleDraft: {
        startDate: "2025-10-27",
        endDate: "2025-11-02",
        shifts: [
          { date: "2025-10-27", staffId: "staff_1", staffName: "张三", shiftId: "shift_morning", shiftName: "早班" },
          // ... 更多排班数据
        ]
      }
    }
  }
}
```

**前端处理**：
- 根据 `phase` 更新 UI 状态
- 根据 `actions` 渲染按钮
- 根据 `extra` 展示具体数据（如排班表格）
- 根据 `actionsVersion` 判断是否需要更新按钮

##### 2.3 决策日志追加

```typescript
{
  type: "decision_log_append",
  sessionId: "session_123",
  decision: {
    at: "2025-10-24T10:30:15Z",
    from: "_state_schedule_create_querying_staff_",
    event: "_event_schedule_create_staff_queried_",
    to: "_state_schedule_create_querying_shifts_",
    info: "查询到 15 名人员",
    draftVersion: 0,
    resultVersion: 0
  }
}
```

**前端处理**：
- 追加到决策日志列表
- 用于调试或展示详细执行流程

##### 2.4 工作流完成

```typescript
{
  type: "workflow_completed",
  sessionId: "session_123",
  result: {
    scheduleId: "schedule_789",
    message: "排班创建成功！"
  }
}
```

**前端处理**：
- 展示成功消息
- 可选：跳转到排班详情页

##### 2.5 工作流失败

```typescript
{
  type: "workflow_failed",
  sessionId: "session_123",
  reason: "user_cancelled",
  errorMessage: "用户已取消排班创建"
}
```

**前端处理**：
- 展示错误消息
- 清空按钮（工作流已终止）

---

## 完整交互流程示例

下面以一个完整的排班创建流程为例，展示前后端的详细交互。

### 场景：用户创建下周排班

**用户目标**：创建 2025-10-27 ~ 2025-11-02 的排班

---

### Step 0: 用户发起请求

#### 前端发送

```json
{
  "type": "user_message",
  "sessionId": "session_123",
  "message": "帮我创建下周的排班"
}
```

#### 后端处理流程

1. **AI 意图识别**：
   ```json
   {
     "intent": "schedule_create",
     "arguments": {
       "startDate": "2025-10-27",
       "endDate": "2025-11-02"
     }
   }
   ```

2. **发送 FSM 事件**：
   ```go
   fsmSystem.SendEvent(ctx, "session_123", Workflow_Schedule, Event_Schedule_Create, intentResult)
   ```

3. **状态转换**：
   - `State_Schedule_Executing` → `State_Create_ValidatingInfo`

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "正在验证排班信息..."
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "state": "_state_schedule_create_validating_info_",
    "phase": "create_validating_info",
    "phaseDesc": "验证排班信息",
    "actions": null,
    "extra": {}
  }
}
```

---

### Step 0.5: 信息不完整，请求补充（可选）

**如果缺少必要信息**（如日期范围），FSM 会进入等待补充状态。

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "创建排班需要以下信息，请补充：\n- 开始日期\n- 结束日期"
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "state": "_state_schedule_create_waiting_supplementary_",
    "phase": "create_waiting_supplementary",
    "phaseDesc": "等待补充信息",
    "actions": [
      { "action": "supplement", "label": "补充信息", "type": "primary" }
    ],
    "extra": {
      "missingFields": ["startDate", "endDate"],
      "promptMessage": "请提供排班的开始和结束日期"
    }
  }
}
```

#### 前端发送补充信息

```json
{
  "type": "supplementary_info",
  "sessionId": "session_123",
  "data": {
    "startDate": "2025-10-27",
    "endDate": "2025-11-02"
  }
}
```

#### 后端处理

- 发送 `Event_Action_UserSupplementary` → 更新 Intent.Arguments
- 发送 `Event_Schedule_Create_UserSupplemented` → 重新验证信息

---

### Step 1: 查询人员信息

#### 后端自动执行

信息验证通过后，FSM 自动进入数据收集阶段。

#### 状态转换

- `State_Create_ValidatingInfo` → `State_Create_QueryingStaff`

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "正在查询人员信息...（步骤 1/5）"
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "state": "_state_schedule_create_querying_staff_",
    "phase": "create_querying_staff",
    "phaseDesc": "查询部门人员",
    "actions": null,
    "extra": {
      "currentStep": 1,
      "totalSteps": 5
    }
  }
}
```

#### 异步查询完成

```go
// 查询人员服务
staffList, err := dataService.QueryStaff(ctx, deptId)

// 发送完成事件
actor.Send(ctx, Event_Schedule_Create_StaffQueried, staffList)
```

#### 决策日志

```json
{
  "type": "decision_log_append",
  "sessionId": "session_123",
  "decision": {
    "at": "2025-10-24T10:30:15Z",
    "from": "_state_schedule_create_validating_info_",
    "event": "_event_schedule_create_info_validated_",
    "to": "_state_schedule_create_querying_staff_",
    "info": "信息验证通过"
  }
}
```

---

### Step 1.5 ~ 4: 继续查询（班次、规则、历史、人员规则）

**类似流程**，每个步骤都会：

1. 发送助手消息：`"正在查询XXX...（步骤 N/5）"`
2. 更新 `workflow_update`：更新 `currentStep`
3. 异步查询完成后，发送下一个事件
4. 记录决策日志

#### 进度展示示例

```json
// Step 2: 查询规则
{
  "type": "assistant_message",
  "message": "正在查询排班规则...（步骤 2/5）"
}

// Step 3: 查询历史
{
  "type": "assistant_message",
  "message": "正在查询历史排班...（步骤 3/5）"
}

// Step 4: 查询人员规则
{
  "type": "assistant_message",
  "message": "正在查询人员规则...（步骤 4/5）"
}
```

---

### Step 5: AI 生成排班初稿

#### 状态转换

- `State_Create_QueryingStaffRules` → `State_Create_GeneratingDraft`

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "正在生成排班方案...（步骤 5/5）"
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "phase": "create_generating_draft",
    "phaseDesc": "生成排班初稿",
    "extra": {
      "currentStep": 5,
      "totalSteps": 5
    }
  }
}
```

#### AI 生成完成

```go
// 调用 AI 服务
draft, err := aiService.GenerateScheduleDraft(ctx, context)

// 更新草案版本
dto.DraftVersion++

// 发送完成事件
actor.Send(ctx, Event_Schedule_Create_DraftGenerated, draft)
```

---

### Step 6: 展示初稿，等待用户确认

#### 状态转换

- `State_Create_GeneratingDraft` → `State_Create_ConfirmingDraft`

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "排班初稿已生成，请确认："
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "state": "_state_schedule_create_confirming_draft_",
    "phase": "create_confirming_draft",
    "phaseDesc": "待确认排班初稿",
    "actions": [
      { "action": "confirm", "label": "确认", "type": "primary" },
      { "action": "cancel", "label": "取消", "type": "default" },
      { "action": "adjust", "label": "调整", "type": "link" }
    ],
    "actionsVersion": 5,
    "extra": {
      "draftVersion": 1,
      "scheduleDraft": {
        "startDate": "2025-10-27",
        "endDate": "2025-11-02",
        "shifts": [
          {
            "date": "2025-10-27",
            "staffId": "staff_1",
            "staffName": "张三",
            "shiftId": "shift_morning",
            "shiftName": "早班",
            "startTime": "08:00",
            "endTime": "16:00"
          },
          {
            "date": "2025-10-27",
            "staffId": "staff_2",
            "staffName": "李四",
            "shiftId": "shift_afternoon",
            "shiftName": "中班",
            "startTime": "16:00",
            "endTime": "24:00"
          },
          // ... 更多排班
        ],
        "statistics": {
          "totalShifts": 35,
          "staffCount": 15,
          "averageShiftsPerStaff": 2.3
        }
      }
    }
  }
}
```

#### 前端展示

```tsx
// React 示例
function ScheduleDraftConfirmation({ workflowMeta }) {
  const { scheduleDraft, draftVersion } = workflowMeta.extra;
  const { actions } = workflowMeta;

  return (
    <div>
      <h3>排班初稿（版本 {draftVersion}）</h3>
      
      {/* 排班表格 */}
      <ScheduleTable data={scheduleDraft.shifts} />
      
      {/* 统计信息 */}
      <Statistics data={scheduleDraft.statistics} />
      
      {/* 操作按钮 */}
      <div className="actions">
        {actions.map(action => (
          <Button
            key={action.action}
            type={action.type}
            onClick={() => handleAction(action.action, draftVersion)}
          >
            {action.label}
          </Button>
        ))}
      </div>
    </div>
  );
}

function handleAction(action, draftVersion) {
  const payload = {
    expectedDraftVersion: draftVersion
  };
  
  sendWorkflowCommand({
    command: action,
    payload: payload
  });
}
```

---

### Step 7a: 用户确认

#### 前端发送

```json
{
  "type": "workflow_command",
  "sessionId": "session_123",
  "command": "confirm",
  "payload": {
    "expectedDraftVersion": 1
  }
}
```

#### 后端处理

1. **映射为 FSM 事件**：`Event_Schedule_Create_UserConfirmed`

2. **Guard 检查**：
   ```go
   func guardDraftVersion(dto *SessionDTO, actor *Actor, payload any) bool {
       expected := payload.(map[string]any)["expectedDraftVersion"].(int64)
       return dto.DraftVersion == expected  // 防止并发冲突
   }
   ```

3. **状态转换**：
   - `State_Create_ConfirmingDraft` → `State_Create_Executing`

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "正在创建排班..."
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "phase": "create_executing",
    "phaseDesc": "执行创建",
    "actions": null
  }
}
```

#### 执行创建

```go
// 调用排班服务
scheduleId, err := scheduleService.CreateSchedule(ctx, draft)

if err != nil {
    actor.Send(ctx, Event_Schedule_Create_Failure, err)
} else {
    actor.Send(ctx, Event_Schedule_Create_Success, scheduleId)
}
```

#### 创建成功

```json
{
  "type": "workflow_completed",
  "sessionId": "session_123",
  "result": {
    "scheduleId": "schedule_789",
    "message": "排班创建成功！"
  }
}
```

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "✅ 排班创建成功！排班 ID: schedule_789"
}
```

---

### Step 7b: 用户调整（可选）

#### 前端发送

```json
{
  "type": "workflow_command",
  "sessionId": "session_123",
  "command": "adjust",
  "payload": {
    "expectedDraftVersion": 1,
    "adjustments": {
      "addShift": {
        "date": "2025-10-28",
        "staffId": "staff_3",
        "shiftId": "shift_morning"
      },
      "removeShiftDates": ["2025-10-29"]
    }
  }
}
```

#### 后端处理

1. **映射为 FSM 事件**：`Event_Schedule_Create_UserRequestAdjust`

2. **状态转换**：
   - `State_Create_ConfirmingDraft` → `State_Create_AdjustingDraft`

3. **AI 调整草案**：
   ```go
   // 应用用户的调整建议
   adjustedDraft, err := aiService.AdjustDraft(ctx, currentDraft, adjustments)
   
   // 发送调整完成事件
   actor.Send(ctx, Event_Schedule_Create_DraftAdjusted, adjustedDraft)
   ```

4. **重新生成**：
   - `State_Create_AdjustingDraft` → `State_Create_GeneratingDraft`

5. **再次确认**：
   - `State_Create_GeneratingDraft` → `State_Create_ConfirmingDraft`
   - `DraftVersion` 递增为 2

#### 后端响应

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "正在调整排班方案..."
}
```

**调整完成后**：

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "调整后的排班方案已生成，请确认："
}
```

```json
{
  "type": "workflow_update",
  "sessionId": "session_123",
  "workflowMeta": {
    "workflow": "scheduling",
    "phase": "create_confirming_draft",
    "phaseDesc": "待确认排班初稿",
    "actions": [
      { "action": "confirm", "label": "确认", "type": "primary" },
      { "action": "cancel", "label": "取消", "type": "default" },
      { "action": "adjust", "label": "继续调整", "type": "link" }
    ],
    "actionsVersion": 6,
    "extra": {
      "draftVersion": 2,  // 版本递增
      "scheduleDraft": {
        // ... 调整后的排班数据
      }
    }
  }
}
```

**用户可以继续调整或确认**，每次调整都会递增 `draftVersion`。

---

### Step 7c: 用户取消（可选）

#### 前端发送

```json
{
  "type": "workflow_command",
  "sessionId": "session_123",
  "command": "cancel",
  "payload": {
    "expectedDraftVersion": 1
  }
}
```

#### 后端处理

1. **映射为 FSM 事件**：`Event_Schedule_Create_UserCancelled`

2. **状态转换**：
   - `State_Create_ConfirmingDraft` → `State_Create_Failed`

#### 后端响应

```json
{
  "type": "workflow_failed",
  "sessionId": "session_123",
  "reason": "user_cancelled",
  "errorMessage": "用户已取消排班创建"
}
```

```json
{
  "type": "assistant_message",
  "sessionId": "session_123",
  "message": "已取消排班创建"
}
```

---

## 数据结构说明

### SessionDTO（会话数据）

```go
type SessionDTO struct {
    SessionID       string           `json:"sessionId"`
    State           State            `json:"state"`
    StateDesc       string           `json:"stateDesc"`
    WorkflowMeta    *WorkflowMeta    `json:"workflowMeta"`
    Context         *SchedulingContext `json:"context"`
    Messages        []Message        `json:"messages"`
    DraftVersion    int64            `json:"draftVersion"`
    ResultVersion   int64            `json:"resultVersion"`
    CreatedAt       time.Time        `json:"createdAt"`
    UpdatedAt       time.Time        `json:"updatedAt"`
}
```

### WorkflowMeta（工作流元数据）

```go
type WorkflowMeta struct {
    Workflow       string            `json:"workflow"`        // 工作流名称
    State          string            `json:"state"`           // 当前状态
    Phase          string            `json:"phase"`           // 当前阶段
    PhaseDesc      string            `json:"phaseDesc"`       // 阶段描述
    Actions        []WorkflowAction  `json:"actions"`         // 可用操作
    ActionsVersion int64             `json:"actionsVersion"`  // 操作版本号
    Extra          map[string]any    `json:"extra"`           // 额外数据
}
```

### WorkflowAction（工作流操作）

```go
type WorkflowAction struct {
    Action  string `json:"action"`  // 操作标识（confirm, cancel, adjust）
    Label   string `json:"label"`   // 显示文本
    Type    string `json:"type"`    // 按钮类型（primary, default, link）
    Enabled bool   `json:"enabled"` // 是否可用
}
```

### ScheduleCreateContext（排班创建上下文）

```go
type ScheduleCreateContext struct {
    Intent             *IntentResult      `json:"intent"`             // 意图识别结果
    StaffList          []Staff            `json:"staffList"`          // 人员列表
    ShiftList          []Shift            `json:"shiftList"`          // 班次列表
    SchedulingRules    []Rule             `json:"schedulingRules"`    // 排班规则
    HistorySchedules   []Schedule         `json:"historySchedules"`   // 历史排班
    StaffRules         map[string][]Rule  `json:"staffRules"`         // 人员规则
    ScheduleDraft      *ScheduleDraft     `json:"scheduleDraft"`      // 排班初稿
    ValidationAttempts int                `json:"validationAttempts"` // 验证尝试次数
    MissingFields      []string           `json:"missingFields"`      // 缺失字段
}
```

### Decision（决策日志）

```go
type Decision struct {
    At        time.Time `json:"at"`        // 决策时间
    From      State     `json:"from"`      // 起始状态
    Event     Event     `json:"event"`     // 触发事件
    To        State     `json:"to"`        // 目标状态
    Info      string    `json:"info"`      // 附加信息
    DraftVer  int64     `json:"draftVersion"`  // 草案版本
    ResultVer int64     `json:"resultVersion"` // 结果版本
}
```

---

## 错误处理

### 1. AI 处理失败

#### 触发条件

- AI 服务调用失败
- AI 返回的结果格式错误
- AI 处理超时

#### FSM 处理

```go
// 任何 AI 处理步骤失败
actor.Send(ctx, Event_Schedule_Create_AIFailed, err)

// 状态转换
State_Create_GeneratingDraft → State_Create_Failed
```

#### 前端响应

```json
{
  "type": "workflow_failed",
  "sessionId": "session_123",
  "reason": "ai_processing_failed",
  "errorMessage": "AI 处理失败：服务不可用"
}
```

### 2. 版本冲突

#### 触发条件

用户提交的 `expectedDraftVersion` 与后端实际的 `DraftVersion` 不匹配。

#### FSM 处理

```go
// Guard 拒绝转换
func guardDraftVersion(dto *SessionDTO, actor *Actor, payload any) bool {
    expected := payload.(map[string]any)["expectedDraftVersion"].(int64)
    if dto.DraftVersion != expected {
        // 发送版本冲突消息
        actor.ISessionService().AddAssistantMessage(
            ctx,
            dto.SessionID,
            fmt.Sprintf("版本冲突：期望 %d，实际 %d。请刷新后重试。", expected, dto.DraftVersion),
        )
        return false  // 拒绝转换
    }
    return true
}
```

#### 前端处理

- 接收到 `assistant_message` 后，提示用户刷新
- 重新拉取最新的 `workflowMeta`

### 3. 数据查询失败

#### 触发条件

查询人员、班次、规则等数据时失败。

#### FSM 处理

```go
// 查询失败
data, err := dataService.Query(ctx, params)
if err != nil {
    // 记录错误并发送失败事件
    actor.Logger().Error("query failed", "error", err)
    actor.Send(ctx, Event_Schedule_Create_Failure, err)
}
```

#### 前端响应

```json
{
  "type": "workflow_failed",
  "sessionId": "session_123",
  "reason": "data_query_failed",
  "errorMessage": "数据查询失败：无法连接到数据服务"
}
```

### 4. 超时处理

#### 触发条件

用户在确认阶段超过一定时间未操作。

#### FSM 处理

```go
// 启动超时定时器
go func() {
    time.Sleep(10 * time.Minute)
    actor.Send(ctx, Event_Schedule_Create_Timeout, nil)
}()

// 超时转换
Transition{
    From:  State_Create_ConfirmingDraft,
    Event: Event_Schedule_Create_Timeout,
    To:    State_Create_Failed,
    Act:   actHandleTimeout,
}
```

#### 前端响应

```json
{
  "type": "workflow_failed",
  "sessionId": "session_123",
  "reason": "timeout",
  "errorMessage": "操作超时，请重新发起排班创建"
}
```

---

## 总结

通过 FSM 工作流引擎，排班创建流程实现了：

✅ **清晰的状态管理**：每个阶段都有明确的状态和事件  
✅ **灵活的用户交互**：支持确认、调整、取消等多种操作  
✅ **智能的信息补充**：自动识别缺失信息并引导用户补充  
✅ **版本控制**：通过 `DraftVersion` 防止并发冲突  
✅ **完整的日志追踪**：决策日志记录每次状态变化  
✅ **优雅的错误处理**：各种异常情况都有明确的处理路径  

**前端开发要点**：
- 监听 `workflow_update` 消息，根据 `phase` 和 `actions` 更新 UI
- 发送命令时携带 `expectedDraftVersion`，防止版本冲突
- 根据 `extra` 数据展示具体内容（排班表、统计等）
- 处理各种错误消息，提供友好的用户提示

**后端开发要点**：
- 在 `Act` 中执行业务逻辑，异步操作通过回调事件驱动
- 使用 `Guard` 进行条件检查，如版本校验
- 及时更新 `WorkflowMeta`，保持前端状态同步
- 记录详细的决策日志，便于调试和审计

---

**相关文档**：
- [FSM 工作流引擎完全指南](../fsm-guide.md)
- [FSM 工作流重构说明](./README.md)

**文档版本**：v1.0  
**最后更新**：2025-10-24  
**维护者**：Scheduling Service Team
