# 从 V1 切换到 V2 排班工作流

## 切换状态

✅ **已完成切换** - 系统现在使用 V2 工作流

## 变更内容

### 1. 意图映射更新

**文件**：`agents/rostering/domain/model/intent.go`

```diff
- IntentSchedule:       {WorkflowName: "schedule.create", Event: "start", Implemented: true},
- IntentScheduleCreate: {WorkflowName: "schedule.create", Event: "start", Implemented: true},
+ IntentSchedule:       {WorkflowName: "schedule_v2.create", Event: "start", Implemented: true},
+ IntentScheduleCreate: {WorkflowName: "schedule_v2.create", Event: "start", Implemented: true},
```

### 2. 工作流注册

V2 工作流已在 `schedule_v2/main.go` 中自动注册：

```go
import (
    _ "jusha/agent/rostering/internal/workflow/schedule_v2/core"
    _ "jusha/agent/rostering/internal/workflow/schedule_v2/create"
)
```

## V1 vs V2 主要区别

| 特性 | V1 (schedule.create) | V2 (schedule_v2.create) |
|------|---------------------|------------------------|
| 架构 | 单一工作流 | 分阶段父工作流 + 子工作流 |
| 班次处理顺序 | 按列表顺序 | 按优先级（固定→特殊→普通→科研→填充） |
| 班次类型识别 | 无 | 使用 Shift.Type 字段 |
| 个人需求 | 混合在排班中 | 独立阶段收集和确认 |
| 约束管理 | 分散 | 集中的约束累积机制 |
| 扩展性 | 需修改主流程 | 仅需扩展父工作流 |
| 固定班次 | 需要AI处理 | 自动填充，无需AI |

## V2 架构优势

### 1. 优先级驱动

```
固定班次 (最高优先级，自动填充)
    ↓
个人需求 (优先占位)
    ↓
特殊班次 (有技能要求，优先排班)
    ↓
普通班次 (常规排班)
    ↓
科研班次 (较低优先级)
    ↓
填充班次 (补充不足)
```

### 2. 约束累积

每个阶段的结果作为后续阶段的约束：
- ✅ 避免人员重复分配
- ✅ 防止时段冲突
- ✅ 确保需求优先满足

### 3. 灵活扩展

添加新班次类型：
1. 定义类型常量
2. 添加状态转换
3. 实现 Action 函数

**无需修改 Core 子工作流**！

## 班次配置要求

### 设置班次类型

V2 使用 `Shift.Type` 字段识别班次类型，请确保班次配置正确：

```go
// 示例：创建特殊班次
shift := &Shift{
    Name: "急诊值班",
    Type: "special",  // 重要！
    SchedulingPriority: 10,
    // ... 其他字段
}
```

### 班次类型对照表

| Type 值 | 说明 | 处理方式 | 优先级 |
|---------|------|----------|--------|
| `fixed` | 固定班次 | 自动填充 | 1（最高） |
| - | 个人需求阶段 | 占位约束 | 2 |
| `special` | 特殊班次 | AI排班 | 3 |
| `normal` 或空 | 普通班次 | AI排班 | 4 |
| `research` | 科研班次 | AI排班 | 5 |
| `fill` | 填充班次 | 填充处理 | 6（最低） |

### 向后兼容

- ✅ `Type` 为空的班次自动视为 `normal`（普通班次）
- ✅ 不影响现有班次数据
- ✅ 可以逐步迁移班次类型

## 数据迁移建议

### 选项 1：渐进式迁移（推荐）

现有班次保持不变，新创建的班次设置正确的 Type：

```sql
-- 可选：标记现有特殊班次
UPDATE shifts 
SET type = 'special' 
WHERE name LIKE '%值班%' OR name LIKE '%急诊%';

-- 可选：标记科研班次
UPDATE shifts 
SET type = 'research' 
WHERE name LIKE '%科研%';
```

### 选项 2：保持默认

不修改现有数据，所有班次使用默认 `normal` 类型：
- ✅ 无风险
- ⚠️ 无法利用优先级排序
- ⚠️ 固定班次仍需AI处理

## 验证步骤

### 1. 检查工作流注册

```bash
# 查看日志，确认工作流已注册
grep "Registered workflow: schedule_v2.create" logs/app.log
```

### 2. 测试排班创建

```bash
# 发送测试消息
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "orgId": "test_org",
    "userId": "test_user",
    "message": "帮我创建下周的排班"
  }'
```

### 3. 验证工作流执行

检查返回的工作流名称应该是 `schedule_v2.create`：

```json
{
  "workflowMeta": {
    "workflow": "schedule_v2.create",
    "phase": "_schedule_v2_create_init_",
    ...
  }
}
```

## 回滚方案

如果需要回滚到 V1：

### 1. 修改意图映射

恢复 `agents/rostering/domain/model/intent.go`：

```go
IntentSchedule:       {WorkflowName: "schedule.create", Event: "start", Implemented: true},
IntentScheduleCreate: {WorkflowName: "schedule.create", Event: "start", Implemented: true},
```

### 2. 重启服务

```bash
# 重启应用
systemctl restart rostering-service
```

## 监控和日志

### 关键日志

查看以下日志确认 V2 正常工作：

```
CreateV2: Starting workflow
CreateV2: Info collection completed
CreateV2: Starting personal needs phase
CreateV2: Starting fixed shift phase
CreateV2: Starting shift phase (phase=special_shift)
...
```

### 常见问题排查

**问题 1**：工作流未启动

```bash
# 检查工作流是否注册
grep "schedule_v2.create" logs/startup.log
```

**问题 2**：班次分类错误

```bash
# 检查班次 Type 字段
SELECT id, name, type FROM shifts WHERE type IS NULL OR type = '';
```

**问题 3**：意图映射失败

```bash
# 检查意图识别日志
grep "Intent mapped to workflow" logs/app.log | grep schedule
```

## 性能对比

| 指标 | V1 | V2 | 变化 |
|------|----|----|------|
| 平均响应时间 | 基准 | 待测试 | - |
| 排班质量 | 基准 | 预期提升 | +优先级排序 |
| 用户交互次数 | 基准 | 略增 | +个人需求确认 |
| 代码可维护性 | 中 | 高 | +清晰架构 |

## 支持

如有问题，请联系：
- 开发团队
- 查看详细文档：`schedule_v2/design.md`
- 查看使用指南：`schedule_v2/create/README.md`

---

**切换日期**：2025-12-17  
**版本**：V2.0.0  
**状态**：✅ 生产就绪

