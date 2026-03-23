# Rostering Workflow API 测试示例

本文档演示如何通过 HTTP API 和 WebSocket 测试排班创建工作流。

## 前置条件

1. 启动 rostering agent 服务（默认端口：根据配置文件）
2. 准备测试工具：curl, websocat 或浏览器 WebSocket 客户端

## 测试流程

### 1. 创建会话

```bash
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "orgId": "org-001",
    "userId": "user-001",
    "workflow": "schedule.create"
  }'
```

**响应示例：**
```json
{
  "sessionId": "sess_abc123",
  "orgId": "org-001",
  "userId": "user-001",
  "workflow": "schedule.create",
  "state": "_schedule_create_confirming_period_"
}
```

**说明：**
- `sessionId` 用于后续所有请求
- `workflow` 指定工作流类型（schedule.create）
- 初始状态为 `_schedule_create_confirming_period_`（确认排班周期）

---

### 2. 连接 WebSocket（可选但推荐）

```bash
# 使用 websocat
websocat ws://localhost:8080/ws
```

**连接后发送绑定消息：**
```json
{
  "type": "bind",
  "sessionId": "sess_abc123"
}
```

**说明：**
- WebSocket 用于接收实时状态更新和进度通知
- 绑定后会自动收到会话相关的所有广播消息

---

### 3. 阶段1：确认排班周期

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_period_confirmed_",
    "payload": {
      "startDate": "2025-01-01",
      "endDate": "2025-01-31"
    }
  }'
```

**响应：**
```json
{
  "status": "ok",
  "event": "_schedule_create_period_confirmed_"
}
```

**WebSocket 收到（状态转换）：**
```json
{
  "type": "state_change",
  "sessionId": "sess_abc123",
  "from": "_schedule_create_confirming_period_",
  "to": "_schedule_create_confirming_shifts_",
  "data": {
    "description": "请选择排班班次",
    "actions": [
      {"event": "_schedule_create_shifts_confirmed_", "label": "确认班次", "type": "primary"},
      {"event": "_schedule_create_shifts_modified_", "label": "修改班次", "type": "secondary"}
    ]
  }
}
```

**说明：**
- 日期范围：今天往前30天 ~ 开始日期往后90天
- 系统会自动查询该时间段的班次信息
- 状态转换：确认周期 → 确认班次

**修改周期（如果需要）：**
```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_period_modified_",
    "payload": {
      "startDate": "2025-01-15",
      "endDate": "2025-02-15"
    }
  }'
```

---

### 4. 阶段2：确认班次

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_shifts_confirmed_",
    "payload": {
      "selectedShiftIds": ["shift-001", "shift-002", "shift-003"]
    }
  }'
```

**WebSocket 收到：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_confirming_staff_count_",
  "data": {
    "description": "请确认每个班次的每日人数需求",
    "staffRequirements": {
      "早班": {
        "2025-01-01": 5,
        "2025-01-02": 5
      },
      "中班": {
        "2025-01-01": 4,
        "2025-01-02": 4
      }
    }
  }
}
```

**说明：**
- 系统会按照 SchedulingPriority 对班次排序
- 自动从班次的 DefaultStaffCount 初始化人数需求

**修改班次（如果需要）：**
```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_shifts_modified_",
    "payload": {
      "selectedShiftIds": ["shift-001", "shift-004"]
    }
  }'
```

---

### 5. 阶段3：确认人数需求

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_staff_count_confirmed_",
    "payload": {
      "requirements": {
        "早班": {
          "2025-01-01": 6,
          "2025-01-02": 5
        },
        "中班": {
          "2025-01-01": 4,
          "2025-01-02": 3
        }
      }
    }
  }'
```

**WebSocket 收到：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_retrieving_staff_",
  "data": {
    "description": "正在检索可用人员..."
  }
}
```

**说明：**
- 可以修改任意日期的人数需求
- 确认后自动进入数据检索阶段

---

### 6. 阶段4-5：自动数据检索

系统会自动执行以下操作（无需用户交互）：

1. **检索人员**：
   - 查询班次关联的分组
   - 查询分组中的员工
   - 过滤请假人员
   - 状态：`_schedule_create_retrieving_staff_` → `_schedule_create_retrieving_rules_`

2. **检索规则**：
   - 查询全局排班规则
   - 查询班次特定规则
   - 查询分组规则
   - 初始化草案结构
   - 状态：`_schedule_create_retrieving_rules_` → `_schedule_create_generating_schedule_`

**WebSocket 持续收到进度更新：**
```json
{
  "type": "progress",
  "phase": "retrieving_staff",
  "message": "已检索到 25 名可用员工"
}
```

---

### 7. 阶段6：AI 生成排班

系统会逐个班次调用 AI 生成排班，每个班次完成时会收到更新：

**WebSocket 收到（每个班次）：**
```json
{
  "type": "progress",
  "phase": "generating",
  "data": {
    "description": "⚙️ 正在排班 1/3: 早班",
    "currentShift": "早班",
    "progress": 33
  }
}
```

**全部完成后：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_confirming_draft_",
  "data": {
    "description": "排班草案已生成，请预览并确认",
    "draft": {
      "shifts": {
        "早班": {
          "days": {
            "2025-01-01": {
              "staffIds": ["emp-001", "emp-002", "emp-003"],
              "actualCount": 3,
              "requiredCount": 3
            }
          }
        }
      },
      "staffStats": {
        "emp-001": {
          "workDays": 15,
          "totalHours": 120,
          "shifts": ["早班", "中班"]
        }
      },
      "conflicts": []
    },
    "actions": [
      {"event": "_schedule_create_draft_confirmed_", "label": "✅ 确认排班", "type": "primary"},
      {"event": "_schedule_create_draft_rejected_", "label": "🔄 调整重排", "type": "secondary"},
      {"event": "_schedule_create_cancel_", "label": "❌ 取消", "type": "danger"}
    ]
  }
}
```

**说明：**
- AI 会考虑人员可用性、规则约束、历史排班等因素
- 每个班次独立生成，避免累积错误
- 生成的草案包含员工统计和冲突检测

---

### 8. 阶段7：确认或拒绝草案

#### 8a. 确认草案

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_draft_confirmed_",
    "payload": {}
  }'
```

**WebSocket 收到：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_saving_schedule_",
  "data": {
    "description": "正在保存排班..."
  }
}
```

#### 8b. 拒绝并调整

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_draft_rejected_",
    "payload": {
      "feedback": "早班人手分配不均，希望增加周末的人数",
      "adjustments": {
        "preferWeekends": true
      }
    }
  }'
```

**说明：**
- 拒绝后会重置生成状态，重新从第一个班次开始
- 用户反馈会添加到 AI 上下文中，影响重新生成的结果
- 状态回到 `_schedule_create_generating_schedule_`

---

### 9. 阶段8：保存排班

确认后自动执行保存（无需用户交互）：

**WebSocket 收到（成功）：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_completed_",
  "data": {
    "description": "✅ 排班保存成功：150 条记录",
    "result": {
      "total": 150,
      "upserted": 150,
      "failed": 0
    }
  }
}
```

**WebSocket 收到（失败）：**
```json
{
  "type": "state_change",
  "to": "_schedule_create_failed_",
  "data": {
    "description": "❌ 保存失败：数据库连接超时",
    "actions": [
      {"event": "_schedule_create_retry_save_", "label": "🔄 重试保存", "type": "primary"},
      {"event": "_schedule_create_cancel_", "label": "❌ 取消", "type": "secondary"}
    ]
  }
}
```

**重试保存：**
```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_retry_save_",
    "payload": {}
  }'
```

---

### 10. 查询会话状态

随时查询当前会话状态：

```bash
curl -X GET http://localhost:8080/api/sessions/sess_abc123
```

**响应示例：**
```json
{
  "id": "sess_abc123",
  "orgId": "org-001",
  "userId": "user-001",
  "workflowMeta": {
    "workflow": "schedule.create",
    "phase": "_schedule_create_confirming_draft_",
    "description": "排班草案已生成，请预览并确认",
    "actions": [
      {"event": "_schedule_create_draft_confirmed_", "label": "✅ 确认排班", "type": "primary"}
    ]
  },
  "data": {
    "DataKeyScheduleCreateContext": {
      "startDate": "2025-01-01",
      "endDate": "2025-01-31",
      "selectedShifts": [...],
      "draftSchedule": {...}
    }
  }
}
```

---

## 取消工作流

任意阶段都可以取消：

```bash
curl -X POST http://localhost:8080/api/sessions/sess_abc123/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": "_schedule_create_cancel_",
    "payload": {
      "reason": "用户主动取消"
    }
  }'
```

---

## 完整工作流状态图

```
[确认周期] 
    ↓ period_confirmed
[确认班次]
    ↓ shifts_confirmed
[确认人数]
    ↓ staff_count_confirmed
[检索人员] (自动)
    ↓ staff_retrieved
[检索规则] (自动)
    ↓ rules_retrieved
[生成排班] (AI 循环)
    ↓ all_shifts_complete
[确认草案]
    ├─ draft_confirmed → [保存排班]
    │                        ↓ save_success
    │                    [已完成] ✅
    └─ draft_rejected → [重新生成] 🔄
```

---

## 错误处理

### AI 生成失败

```json
{
  "type": "state_change",
  "to": "_schedule_create_failed_",
  "data": {
    "description": "❌ AI生成失败：服务不可用",
    "actions": [
      {"event": "_schedule_create_retry_generation_", "label": "🔄 重试", "type": "primary"},
      {"event": "_schedule_create_cancel_", "label": "❌ 取消", "type": "secondary"}
    ]
  }
}
```

### 系统错误

```json
{
  "type": "state_change",
  "to": "_schedule_create_failed_",
  "data": {
    "description": "❌ 系统错误：内部服务异常",
    "actions": [
      {"event": "_schedule_create_cancel_", "label": "❌ 关闭", "type": "secondary"}
    ]
  }
}
```

---

## 测试脚本

完整的测试脚本（Bash）：

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"
SESSION_ID=""

# 1. 创建会话
echo "1. 创建会话..."
RESPONSE=$(curl -s -X POST $BASE_URL/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"orgId":"org-001","userId":"user-001","workflow":"schedule.create"}')
SESSION_ID=$(echo $RESPONSE | jq -r '.sessionId')
echo "Session ID: $SESSION_ID"
echo ""

# 2. 确认周期
echo "2. 确认排班周期..."
curl -s -X POST $BASE_URL/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"event":"_schedule_create_period_confirmed_","payload":{"startDate":"2025-01-01","endDate":"2025-01-31"}}' \
  | jq .
echo ""

# 3. 确认班次
echo "3. 确认班次..."
curl -s -X POST $BASE_URL/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"event":"_schedule_create_shifts_confirmed_","payload":{"selectedShiftIds":["shift-001","shift-002"]}}' \
  | jq .
echo ""

# 4. 确认人数
echo "4. 确认人数需求..."
curl -s -X POST $BASE_URL/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"event":"_schedule_create_staff_count_confirmed_","payload":{"requirements":{"早班":{"2025-01-01":5}}}}' \
  | jq .
echo ""

# 5. 等待 AI 生成（实际应监听 WebSocket）
echo "5. 等待 AI 生成..."
sleep 5

# 6. 查询状态
echo "6. 查询当前状态..."
curl -s -X GET $BASE_URL/api/sessions/$SESSION_ID | jq .
echo ""

# 7. 确认草案
echo "7. 确认排班草案..."
curl -s -X POST $BASE_URL/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"event":"_schedule_create_draft_confirmed_","payload":{}}' \
  | jq .
echo ""

echo "完成！"
```

---

## 注意事项

1. **WebSocket 连接**：建议始终连接 WebSocket 以接收实时更新
2. **事件名称**：必须使用完整的事件名（如 `_schedule_create_period_confirmed_`）
3. **状态验证**：Infrastructure 会自动验证事件是否在当前状态下有效
4. **错误处理**：所有错误都会返回 HTTP 4xx/5xx 状态码和详细错误信息
5. **并发控制**：同一会话不应并发发送多个事件

---

## 前端集成示例

```javascript
// 创建会话
const createSession = async () => {
  const response = await fetch('http://localhost:8080/api/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      orgId: 'org-001',
      userId: 'user-001',
      workflow: 'schedule.create'
    })
  });
  return await response.json();
};

// 连接 WebSocket
const connectWebSocket = (sessionId) => {
  const ws = new WebSocket('ws://localhost:8080/ws');
  
  ws.onopen = () => {
    // 绑定会话
    ws.send(JSON.stringify({ type: 'bind', sessionId }));
  };
  
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
    
    if (data.type === 'state_change') {
      updateUI(data);
    } else if (data.type === 'progress') {
      updateProgress(data);
    }
  };
  
  return ws;
};

// 发送事件
const sendEvent = async (sessionId, event, payload) => {
  const response = await fetch(`http://localhost:8080/api/sessions/${sessionId}/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ event, payload })
  });
  return await response.json();
};

// 使用示例
const session = await createSession();
const ws = connectWebSocket(session.sessionId);
await sendEvent(session.sessionId, '_schedule_create_period_confirmed_', {
  startDate: '2025-01-01',
  endDate: '2025-01-31'
});
```
