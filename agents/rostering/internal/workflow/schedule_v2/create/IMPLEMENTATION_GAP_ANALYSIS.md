# schedule_v2.create 工作流实现差距分析

## 📋 流程对比

### Design.md 定义的流程

```
1. 信息收集 (InfoCollect 子工作流)
   ↓
2. 个人需求确认
   ↓
3. 固定班次处理 (自动填充)
   ↓
4. 特殊班次排班 (循环调用 Core 子工作流)
   ↓
5. 普通班次排班 (循环调用 Core 子工作流)
   ↓
6. 科研班次排班 (循环调用 Core 子工作流)
   ↓
7. 填充班次处理
   ↓
8. 确认保存
   ↓
9. 完成
```

### 实际实现状态

| 阶段 | 设计状态 | 实现状态 | 完成度 | 问题 |
|------|---------|---------|--------|------|
| 1. 信息收集 | ✅ 设计完整 | ⚠️ 使用模拟数据 | 30% | 未实现 InfoCollect 子工作流 |
| 2. 个人需求 | ✅ 设计完整 | ✅ 基本实现 | 80% | 缺少详细展示，缺少临时需求添加 |
| 3. 固定班次 | ✅ 设计完整 | ✅ 已实现 | 95% | 缺少结果保存逻辑 |
| 4. 特殊班次 | ✅ 设计完整 | ⚠️ 框架已实现 | 60% | Core 子工作流未调用，结果未保存 |
| 5. 普通班次 | ✅ 设计完整 | ⚠️ 框架已实现 | 60% | Core 子工作流未调用，结果未保存 |
| 6. 科研班次 | ✅ 设计完整 | ⚠️ 框架已实现 | 60% | Core 子工作流未调用，结果未保存 |
| 7. 填充班次 | ✅ 设计完整 | ❌ 未实现 | 10% | 只有空函数，缺少核心逻辑 |
| 8. 确认保存 | ✅ 设计完整 | ❌ 未实现 | 5% | 只有空函数，缺少预览和保存逻辑 |
| 9. 完成 | ✅ 设计完整 | ✅ 状态定义 | 100% | - |

---

## 🔴 严重缺失（阻塞功能）

### 1. InfoCollect 子工作流未实现

**设计要求**：
- 收集排班时间范围
- 收集待排班次列表
- 收集每个班次每天的排班人数
- 收集可用人员列表
- 收集请假信息
- 收集排班规则

**当前实现**：
```go
// actions.go:53-55
// TODO: 启动 InfoCollect 子工作流
// 当前先直接触发 InfoCollected 事件（待子工作流实现后替换）

// actions.go:948-957
func populateInfoFromSubWorkflow(...) error {
    // TODO: 实际从 InfoCollect 子工作流的输出中提取数据
    // 当前使用模拟数据
    createCtx.StartDate = "2025-01-01"  // 硬编码
    createCtx.EndDate = "2025-01-07"    // 硬编码
    return nil
}
```

**影响**：工作流无法正常启动，只能使用模拟数据

---

### 2. Core 子工作流未调用

**设计要求**：
- 特殊班次、普通班次、科研班次都需要调用 `schedule.core` 子工作流
- 每个班次独立调用，循环处理

**当前实现**：
```go
// actions.go:524-533
// TODO: 调用 Core 子工作流
// config := &engine.SubWorkflowConfig{
//     WorkflowName: WorkflowSchedulingCore,
//     OnComplete:   CreateV2EventShiftCompleted,
//     OnError:      CreateV2EventSubFailed,
// }
// return wctx.SpawnSubWorkflow(config)

// 临时：直接触发完成事件（待 Core 子工作流集成）
return wctx.Send(ctx, CreateV2EventShiftCompleted, nil)
```

**影响**：所有需要AI排班的阶段都无法实际排班，只是空转

---

### 3. 班次排班结果未保存

**设计要求**：
- 每个班次完成后，需要将排班结果保存到对应阶段的 `PhaseResult.ScheduleDrafts`
- 需要更新 `OccupiedSlots` 和 `ExistingScheduleMarks`

**当前实现**：
```go
// actions.go:636-641
// TODO: 实现班次结果的解析和保存
// shiftCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, "shift_scheduling_context")
// if err == nil && found {
//     // 解析 shiftCtxData 并保存到阶段结果中
// }
```

**影响**：
- 排班结果丢失
- 后续阶段无法获取已排班信息
- 约束累积机制失效

---

### 4. 填充班次阶段未实现

**设计要求**：
- 检测排班不足的人员（`DetectUnderScheduledStaff` 已实现）
- 使用填充班次（年假、行政班等）补充排班
- 用户确认填充结果

**当前实现**：
```go
// actions.go:817-821
func startFillShiftPhase(...) error {
    // TODO: 实现填充班次阶段
    // 临时：直接触发阶段完成
    return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
}
```

**影响**：排班不足的人员无法得到补充

---

### 5. 确认保存阶段未实现

**设计要求**：
- 合并所有阶段的排班结果（`MergeScheduleDrafts` 已实现）
- 生成预览界面
- 检测冲突
- 保存到数据库
- 生成统计报表

**当前实现**：
```go
// actions.go:836-839
func actOnSaveCompleted(...) error {
    // TODO: 实现
    return nil
}
```

**影响**：排班结果无法保存，工作流无法完成

---

## ⚠️ 部分缺失（影响体验）

### 6. 个人需求详细展示

**设计要求**：
- 展示需求列表（人员、需求类型、目标班次、日期等）
- 支持用户添加临时需求

**当前实现**：
```go
// actions.go:166
// TODO: 添加详细的需求列表展示（使用 Table 或 List 类型）
```

**影响**：用户无法看到具体需求内容，体验不佳

---

### 7. 规则解析不完整

**设计要求**：
- 从规则中提取班次和日期要求
- 区分常态化需求和临时需求

**当前实现**：
```go
// helpers.go:119
// TODO: 根据 rule.TimeScope 或 rule.ValidFrom/ValidTo 判断是否为临时需求

// helpers.go:142
// TODO: 根据规则的 RuleType 和 RuleData 解析具体的班次和日期要求
```

**影响**：个人需求提取不准确，可能遗漏重要需求

---

### 8. 规则分离未实现

**设计要求**：
- 区分全局规则和班次规则
- 在排班时只传递相关规则

**当前实现**：
```go
// actions.go:565-566
shiftCtx.GlobalRules = createCtx.Rules         // TODO: 分离全局规则和班次规则
shiftCtx.ShiftRules = make([]*d_model.Rule, 0) // TODO: 从 Rules 中筛选班次规则
```

**影响**：可能传递过多无关规则，影响AI排班效率

---

### 9. 冲突检测未实现

**设计要求**：
- 在合并排班结果时检测冲突
- 报告时段冲突、人员冲突等

**当前实现**：
```go
// helpers.go:410
// TODO: 检测冲突
// detectConflicts(finalDraft)
```

**影响**：无法发现排班冲突，可能导致数据错误

---

### 10. 固定班次结果保存不完整 ⚠️ 已部分实现但有问题

**设计要求**：
- 保存固定班次的排班结果到 `FixedShiftResults.ScheduleDrafts`

**当前实现**：
```go
// actions.go:302-314 - 在 startFixedShiftPhase 中已保存到 ScheduleDrafts
createCtx.FixedShiftResults.ScheduleDrafts = fixedDrafts

// actions.go:378-385 - 但在确认时重新创建 PhaseResult，丢失了 ScheduleDrafts！
createCtx.FixedShiftResults = &PhaseResult{
    PhaseName:      PhaseFixedShift,
    // ... 但没有 ScheduleDrafts 字段
}
```

**问题**：确认时覆盖了之前保存的 `ScheduleDrafts`，导致固定排班结果丢失

**影响**：固定班次结果在确认后丢失，无法用于后续阶段和最终保存

---

## ✅ 已正确实现

1. ✅ **工作流状态机定义**：所有状态和事件都已定义
2. ✅ **上下文数据结构**：`CreateV2Context` 完整实现
3. ✅ **班次分类和排序**：`ClassifyShiftsByType` 和 `SortShiftsBySchedulingPriority` 已实现
4. ✅ **固定班次计算**：`CalculateMultipleFixedSchedules` 已集成
5. ✅ **占位管理**：`MergeOccupiedSlots` 和 `BuildOccupiedSlotsMap` 已实现
6. ✅ **人员过滤**：`filterAvailableStaffByOccupiedSlots` 已实现
7. ✅ **阶段进度管理**：`IncrementPhaseProgress` 和 `IsPhaseComplete` 已实现
8. ✅ **错误处理**：基本的错误处理已实现

---

## 📊 实现完成度统计

| 类别 | 完成度 | 说明 |
|------|--------|------|
| **架构设计** | 100% | 状态机、数据结构、流程定义完整 |
| **核心逻辑** | 60% | 分类、排序、占位管理等已实现 |
| **子工作流集成** | 0% | InfoCollect 和 Core 子工作流未调用 |
| **数据持久化** | 20% | 上下文保存已实现，排班结果未保存 |
| **用户交互** | 50% | 基本交互已实现，详细展示缺失 |
| **业务逻辑** | 40% | 固定班次已实现，其他阶段框架代码 |

**总体完成度：约 45%**

---

## 🎯 优先级修复建议

### P0（阻塞功能，必须实现）

1. **实现 InfoCollect 子工作流或替代方案**
   - 方案A：实现完整的 InfoCollect 子工作流
   - 方案B：通过 API 直接获取数据，跳过子工作流

2. **实现 Core 子工作流调用**
   - 或使用临时方案：直接调用排班服务

3. **实现班次结果保存**
   - 从 Core 子工作流结果中提取排班数据
   - 保存到对应阶段的 `PhaseResult.ScheduleDrafts`

4. **实现确认保存功能**
   - 合并所有阶段结果
   - 保存到数据库
   - 生成统计报表

### P1（重要功能，影响体验）

5. **实现填充班次逻辑**
   - 使用 `DetectUnderScheduledStaff` 检测不足
   - 实现填充策略

6. **完善个人需求展示**
   - 添加详细列表展示
   - 支持临时需求添加

7. **完善规则解析**
   - 提取班次和日期要求
   - 区分常态化/临时需求

### P2（优化功能）

8. **实现冲突检测**
9. **分离全局规则和班次规则**
10. **完善固定班次结果保存**

---

## 📝 详细TODO清单

### 信息收集阶段
- [ ] 实现 InfoCollect 子工作流
- [ ] 或实现数据获取的替代方案
- [ ] 从子工作流输出中提取数据

### 个人需求阶段
- [ ] 添加详细需求列表展示（Table/List）
- [ ] 实现临时需求添加功能
- [ ] 完善规则解析（提取班次和日期）

### 固定班次阶段
- [ ] 保存 `ScheduleDrafts` 到 `FixedShiftResults`
- [ ] 实现固定班次修改功能

### 排班阶段（特殊/普通/科研）
- [ ] 实现 Core 子工作流调用
- [ ] 实现班次结果解析和保存
- [ ] 分离全局规则和班次规则
- [ ] 更新 `OccupiedSlots` 和 `ExistingScheduleMarks`

### 填充班次阶段
- [ ] 实现填充策略
- [ ] 实现用户交互
- [ ] 保存填充结果

### 确认保存阶段
- [ ] 实现预览界面
- [ ] 实现冲突检测
- [ ] 实现数据库保存
- [ ] 实现统计报表生成

### 辅助功能
- [ ] 实现冲突检测函数
- [ ] 完善规则解析逻辑
- [ ] 计算总工作小时数

---

## 🔍 代码位置索引

### 关键TODO位置

1. **InfoCollect 子工作流**：
   - `actions.go:53-55` - 启动子工作流
   - `actions.go:77` - 提取数据
   - `actions.go:948-957` - 模拟数据填充

2. **Core 子工作流**：
   - `actions.go:524-533` - 调用子工作流
   - `helpers.go:311-322` - `SpawnCoreSubWorkflow` 函数

3. **结果保存**：
   - `actions.go:636-641` - 班次结果保存
   - `actions.go:378` - 固定班次结果保存

4. **填充班次**：
   - `actions.go:817-821` - `startFillShiftPhase`
   - `actions.go:811-814` - `actOnFillShiftComplete`

5. **确认保存**：
   - `actions.go:836-839` - `actOnSaveCompleted`
   - `actions.go:842-845` - `actModifyBeforeSave`

---

## 💡 建议

1. **分阶段实现**：先实现 P0 功能，确保工作流可以完整运行
2. **使用临时方案**：对于子工作流，可以先使用直接调用服务的方式
3. **完善测试**：每实现一个功能，添加对应的测试用例
4. **更新文档**：实现后及时更新 design.md，标记完成状态

