# 排班创建工作流 V2

优先级驱动的分阶段排班创建工作流，支持灵活的班次类型扩展。

## 快速开始

### 1. 导入工作流

```go
import (
    _ "jusha/agent/rostering/internal/workflow/schedule_v2/create"
)
```

工作流会自动注册为 `schedule_v2.create`。

### 2. 准备班次数据

确保班次的 `Type` 字段设置正确：

```go
// 创建或更新班次时设置类型
shift := &Shift{
    Name: "急诊班",
    Type: "special",  // 特殊班次
    SchedulingPriority: 10,
    // ... 其他字段
}
```

**班次类型说明**：

| Type 值 | 说明 | 排班方式 | 优先级 |
|---------|------|----------|--------|
| `fixed` | 固定班次 | 自动填充 | 1（最高） |
| `special` | 特殊班次 | AI排班 | 3（高） |
| `normal` | 普通班次 | AI排班 | 4（中） |
| `research` | 科研班次 | AI排班 | 5（较低） |
| `fill` | 填充班次 | 填充处理 | 6（最低） |
| 空值 | 默认普通班次 | AI排班 | 4（中） |

**前端类型自动映射**：

| 前端类型 | 前端显示 | 自动映射到 | 优先级 |
|---------|---------|-----------|--------|
| `regular` | 常规班次 | `normal` | 4（中） |
| `overtime` | 加班班次 | `special` | 3（高） |
| `standby` | 备班班次 | `special` | 3（高） |

> 📝 **说明**：前端使用 `regular`/`overtime`/`standby`，系统会自动映射到对应的工作流类型。无需手动转换，已有数据无需修改。详见 [前后端类型对齐说明](../FRONTEND_TYPE_MAPPING.md)。

### 3. 启动工作流

```go
// 获取会话
session := getSession(sessionID)

// 创建工作流上下文
ctx := engine.NewContext(session, services)

// 启动工作流
err := ctx.Send(context.Background(), CreateV2EventStart, nil)
if err != nil {
    log.Error("Failed to start workflow", "error", err)
}
```

## 工作流阶段

### 阶段 1：信息收集

自动收集：
- 排班周期（开始日期、结束日期）
- 可用班次列表
- 人员列表
- 排班规则
- 人数需求

**用户交互**：确认收集的信息

### 阶段 2：个人需求

从规则中提取个人需求（常态化和临时）：
- 需求类型：`permanent` (常态化) / `temporary` (临时)
- 请求类型：`prefer` (偏好) / `avoid` (回避) / `must` (必须)

**用户交互**：确认需求列表，可补充临时需求

### 阶段 3：固定班次

自动填充固定班次（从历史数据或配置）。

**用户交互**：确认固定班次安排

### 阶段 4：特殊班次排班

循环处理每个特殊班次：
1. 按 `SchedulingPriority` 排序
2. 调用 Core 子工作流进行AI排班
3. 更新已占位信息

**约束条件**：
- 个人需求
- 固定班次占位
- 已排班人员不可重复分配

### 阶段 5：普通班次排班

与特殊班次类似，处理普通班次列表。

**约束条件**：累积前面所有阶段的占位信息

### 阶段 6：科研班次排班

处理科研班次（优先级较低）。

### 阶段 7：填充班次

检测排班不足的人员，使用填充班次补充：
- 年假班次
- 行政班次
- 培训班次等

### 阶段 8：确认保存

汇总所有阶段结果，展示完整排班表。

**用户交互**：最终确认并保存

## 核心概念

### 约束累积

每个阶段的排班结果会累积为约束条件：

```go
// 已占位映射
OccupiedSlots: map[staffID]map[date]shiftID

// 示例
{
    "staff_001": {
        "2025-01-01": "shift_fixed_001",  // 固定班次占位
        "2025-01-02": "shift_special_001" // 特殊班次占位
    }
}
```

后续阶段排班时，这些人员-日期组合不可再被分配。

### 个人需求作为约束

个人需求在阶段2收集后，作为约束传递给所有排班阶段：

```go
type PersonalNeed struct {
    StaffID       string   // 人员ID
    RequestType   string   // "prefer" | "avoid" | "must"
    TargetShiftID string   // 希望/回避的班次
    TargetDates   []string // 特定日期
    Priority      int      // 优先级
}
```

Core 子工作流在生成排班时会遵守这些需求。

### 优先级排序

同类型班次内部，按 `SchedulingPriority` 升序排序：

```go
shifts := SortShiftsBySchedulingPriority(shifts)
// Priority 值越小，越先被处理
```

## 扩展指南

### 添加新的班次类型

#### 1. 定义常量

在 `state/schedule/create_v2.go` 中：

```go
const (
    // ... 现有类型
    ShiftTypeTraining = "training" // 培训班次
)
```

#### 2. 添加状态

```go
const (
    // ... 现有状态
    CreateV2StateTrainingShift engine.State = "_schedule_v2_create_training_shift_"
)
```

#### 3. 定义转换

在 `create/definition.go` 的 `buildCreateV2Transitions()` 中添加：

```go
// 科研班次 -> 培训班次
{
    From:       CreateV2StateResearchShift,
    Event:      CreateV2EventShiftPhaseComplete,
    To:         CreateV2StateTrainingShift,
    StateLabel: "科研班次完成，正在排培训班次...",
    Act:        actOnPhaseComplete,
},

// 培训班次排班
{
    From:       CreateV2StateTrainingShift,
    Event:      CreateV2EventShiftCompleted,
    To:         CreateV2StateTrainingShift,
    StateLabel: "培训班次排班中...",
    Act:        actOnShiftCompleted,
    AfterAct:   actSpawnNextShiftOrComplete,
},

// 培训班次 -> 填充班次
{
    From:       CreateV2StateTrainingShift,
    Event:      CreateV2EventShiftPhaseComplete,
    To:         CreateV2StateFillShift,
    StateLabel: "培训班次完成，正在处理填充班次...",
    Act:        actOnPhaseComplete,
},
```

#### 4. 更新 Actions

在 `create/actions.go` 的 `startNextPhase()` 中添加：

```go
func startNextPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context, currentPhase string) error {
    switch currentPhase {
    // ... 现有阶段
    case PhaseResearchShift:
        return startShiftPhase(ctx, wctx, createCtx, PhaseTrainingShift, ShiftTypeTraining, CreateV2StateTrainingShift)
    case PhaseTrainingShift:
        return startFillShiftPhase(ctx, wctx, createCtx)
    // ...
    }
}
```

#### 5. 添加阶段常量

在 `state/schedule/create_v2.go` 中：

```go
const (
    // ... 现有阶段
    PhaseTrainingShift = "training_shift" // 培训班次
)
```

#### 6. 更新辅助函数

在 `create/helpers.go` 的 `getPhaseNameCN()` 中添加：

```go
phaseNames := map[string]string{
    // ... 现有映射
    PhaseTrainingShift: "培训班次",
}
```

完成！新类型的班次会在正确的优先级位置被处理。

### 自定义排班逻辑

如果某种班次类型需要特殊的排班逻辑（不使用 Core 子工作流），可以在 Actions 中实现专门的处理函数：

```go
// startTrainingShiftPhase 培训班次特殊处理
func startTrainingShiftPhase(ctx context.Context, wctx engine.Context, createCtx *CreateV2Context) error {
    // 实现特殊逻辑
    // ...
    
    // 完成后触发阶段完成事件
    return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
}
```

## 调试技巧

### 1. 查看当前阶段

```go
createCtx, _ := loadCreateV2Context(ctx, wctx)
fmt.Printf("当前阶段: %s\n", createCtx.CurrentPhase)
fmt.Printf("当前班次索引: %d/%d\n", 
    createCtx.CurrentShiftIndex, 
    createCtx.TotalShiftsInPhase)
```

### 2. 检查已占位信息

```go
for staffID, dates := range createCtx.OccupiedSlots {
    for date, shiftID := range dates {
        fmt.Printf("人员 %s 在 %s 已分配到班次 %s\n", 
            staffID, date, shiftID)
    }
}
```

### 3. 查看分类结果

```go
for shiftType, shifts := range createCtx.ClassifiedShifts {
    fmt.Printf("类型 %s: %d 个班次\n", shiftType, len(shifts))
}
```

### 4. 启用详细日志

```go
logger := wctx.Logger()
logger.Info("详细信息", 
    "phase", createCtx.CurrentPhase,
    "completedShifts", createCtx.CompletedShiftCount,
    "skippedShifts", createCtx.SkippedShiftCount)
```

## 常见问题

### Q: 班次没有按预期优先级处理？

**A**: 检查两点：
1. 班次的 `Type` 字段是否设置正确
2. 同类型班次的 `SchedulingPriority` 值（越小越优先）

### Q: 如何跳过某个阶段？

**A**: 在用户交互时选择"跳过"按钮，或在代码中触发 `CreateV2EventSkipPhase` 事件。

### Q: 个人需求如何生效？

**A**: 个人需求在阶段2收集后，会在 `prepareShiftSchedulingContext()` 中传递给 Core 子工作流，AI排班时会遵守这些约束。

### Q: 如何处理固定班次？

**A**: 固定班次（`Type="fixed"`）不需要AI排班，系统会自动从历史数据或配置中填充，只需用户确认。

### Q: 填充班次如何工作？

**A**: 填充班次在所有正式班次排完后触发，系统检测每个人员的排班天数，对未达标人员使用填充班次（年假、行政班等）补充。

## 最佳实践

1. **合理设置班次类型**：确保每个班次的 `Type` 字段准确反映其性质
2. **设置优先级**：使用 `SchedulingPriority` 控制同类型班次内的处理顺序
3. **规则清晰化**：确保个人需求规则格式规范，便于自动提取
4. **阶段性确认**：不要跳过重要的用户确认节点
5. **保留日志**：记录每个阶段的执行结果，便于问题追踪

## 性能优化

1. **班次数量**：建议每次排班的班次数量不超过 50 个
2. **人员规模**：系统可支持数百人的排班
3. **并发处理**：未来可考虑同类型班次的并行处理（需要引擎支持）
4. **缓存优化**：重复查询的规则和人员信息可以缓存

## 参与贡献

遇到问题或有改进建议，请联系开发团队。

## 许可

内部项目，仅供公司内部使用。

