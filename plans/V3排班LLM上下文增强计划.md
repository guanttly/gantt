# V3排班LLM上下文增强计划

**版本**: v2.0 - 基于现有班次数据  
**创建日期**: 2026-01-26  
**状态**: 实施中  
**预计工期**: 8小时  

---

## 一、问题背景

### 1.1 当前问题
LLM在执行排班任务时缺少关键上下文信息：
- ❌ 不知道班次的时间段（几点到几点）
- ❌ 不知道人员已被安排的其他班次
- ❌ 无法判断时间冲突
- ❌ 导致同一人被安排到7-8个班次

### 1.2 现有优势
✅ **班次数据已包含时间信息**（从MCP获取）：
```go
type Shift struct {
    StartTime   string `json:"startTime"`   // HH:MM格式，如"08:00"
    EndTime     string `json:"endTime"`     // HH:MM格式，如"18:00"
    Duration    int    `json:"duration"`    // 分钟数
    IsOvernight bool   `json:"isOvernight"` // 是否跨夜
}
```

---

## 二、解决方案设计

### 2.1 核心数据结构

#### 结构1：已分配班次信息
```go
// AssignedShiftInfo 已分配的班次信息（包含完整时间）
type AssignedShiftInfo struct {
    ShiftID     string  `json:"shiftId"`
    ShiftName   string  `json:"shiftName"`
    StartTime   string  `json:"startTime"`   // 直接从Shift复制
    EndTime     string  `json:"endTime"`     // 直接从Shift复制
    Duration    float64 `json:"duration"`    // 小时数
    IsOvernight bool    `json:"isOvernight"` // 直接从Shift复制
    IsFixed     bool    `json:"isFixed"`     // 是否固定排班
    Source      string  `json:"source"`      // "fixed" | "task_1" | "task_2"
}
```

#### 结构2：人员当前排班状态
```go
// StaffCurrentSchedule 人员当前排班状态（某一天）
type StaffCurrentSchedule struct {
    StaffID    string               `json:"staffId"`
    StaffName  string               `json:"staffName"`
    Date       string               `json:"date"`
    Shifts     []*AssignedShiftInfo `json:"shifts"`     // 已安排的班次
    TotalHours float64              `json:"totalHours"` // 总工时
    Warnings   []string             `json:"warnings"`   // 警告（如超时、冲突）
}
```

#### 结构3：排班上下文
```go
// SchedulingContext 排班上下文（传给LLM的完整信息）
type SchedulingContext struct {
    // 当前任务信息
    TargetDate      string `json:"targetDate"`
    TargetShiftID   string `json:"targetShiftId"`
    TargetShiftName string `json:"targetShiftName"`
    TargetShiftTime string `json:"targetShiftTime"` // "08:00-16:00"
    RequiredCount   int    `json:"requiredCount"`
    
    // 所有班次时间表（直接使用Shift对象）
    AllShifts []*Shift `json:"allShifts"`
    
    // 人员当前排班状态
    StaffSchedules []*StaffCurrentSchedule `json:"staffSchedules"`
    
    // 约束配置
    MaxDailyHours float64 `json:"maxDailyHours"` // 12小时
    MinRestHours  float64 `json:"minRestHours"`  // 12小时
}
```

### 2.2 核心算法

#### 算法1：时间重叠检测
```go
func CheckTimeOverlap(shift1, shift2 *Shift) bool {
    s1 := timeToMinutes(shift1.StartTime)
    e1 := timeToMinutes(shift1.EndTime)
    s2 := timeToMinutes(shift2.StartTime)
    e2 := timeToMinutes(shift2.EndTime)
    
    // 处理跨夜
    if shift1.IsOvernight { e1 += 24 * 60 }
    if shift2.IsOvernight { e2 += 24 * 60 }
    
    // 检查重叠
    return !(e1 <= s2 || e2 <= s1)
}
```

#### 算法2：构建人员排班状态
```go
func BuildStaffCurrentSchedules(
    date string,
    workingDraft *ScheduleDraft,
    staffList []*Employee,
    allShifts []*Shift,
) []*StaffCurrentSchedule {
    // 1. 构建班次映射
    shiftMap := buildShiftMap(allShifts)
    
    // 2. 遍历WorkingDraft收集排班
    staffScheduleMap := make(map[string]*StaffCurrentSchedule)
    for shiftID, shiftDraft := range workingDraft.Shifts {
        if dayShift := shiftDraft.Days[date]; dayShift != nil {
            shift := shiftMap[shiftID]
            for i, staffID := range dayShift.StaffIDs {
                // 添加班次到人员记录
                addShiftToStaff(staffScheduleMap, staffID, shift, dayShift, i)
            }
        }
    }
    
    // 3. 检测警告
    for _, schedule := range staffScheduleMap {
        detectWarnings(schedule)
    }
    
    return mapToSlice(staffScheduleMap)
}
```

### 2.3 Prompt增强示例

**改进前**：
```
为早班安排5人。
可用人员：张三、李四、王五...
```

**改进后**：
```
【排班任务】为 早班 (08:00-16:00) 安排人员，需要 5 人

【班次时间表】
- 早班: 08:00-16:00 (8.0小时)
- 中班: 14:00-22:00 (8.0小时)
- 晚班: 22:00-06:00 (8.0小时) (跨夜)

【人员当前排班】(2025-01-20)
- 张三: 已安排 晚班(22:00-06:00)[固定] (共8.0小时)
  ⚠️ 如安排早班将休息不足(<12小时)
- 李四: 尚未安排 ✅ 优先考虑
- 王五: 已安排 待命(00:00-24:00) (共0.0小时) ✅ 可叠加其他班次

【约束条件】
- 单日最大工作时长: 12.0小时
- 班次间最小休息: 12.0小时
- 避免有实际工作时间重叠的班次组合
- 优先选择尚未排班的人员

【可用人员】
（列表...）
```

---

## 三、实施计划

### Phase 1: 数据结构（1.5小时）✅ 已完成

**文件**: `agents/rostering/domain/model/schedule.go`

- [x] 添加 `AssignedShiftInfo` 结构体
- [x] 添加 `StaffCurrentSchedule` 结构体
- [x] 添加 `SchedulingContext` 结构体

### Phase 2: 时间计算工具（2小时）✅ 已完成

**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/time_calculator.go`

- [x] `timeToMinutes(timeStr string) int` - HH:MM转分钟
- [x] `CheckTimeOverlap(shift1, shift2 *Shift) bool` - 检测时间重叠
- [x] `HasTimeOverlap(shifts []*AssignedShiftInfo) bool` - 检测班次数组冲突
- [x] `CalculateRestHours(shift1End, shift2Start string) float64` - 计算休息时长

**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/scheduling_context_builder.go`

- [x] `BuildStaffCurrentSchedules()` - 构建人员排班状态
- [x] `BuildSchedulingContext()` - 构建完整上下文
- [x] 辅助函数：`buildShiftMap()`, `getStaffName()`, `determineSource()`
- [x] 修复日期计算bug（使用time包精确处理）

### Phase 3: 集成到任务执行（2小时）✅ 已完成

**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/progressive_task_executor.go`

- [x] 修改子任务执行逻辑
- [x] 构建SchedulingContext并传递给AI

**文件**: `agents/rostering/internal/services/scheduling_ai.go`

- [x] 更新 `ExecuteTodoTask` 方法签名（新增allShifts和workingDraft参数）
- [x] 重构Prompt构造，使用新上下文
- [x] 添加V3增强的班次时间表展示
- [x] 添加V3增强的人员排班状态展示
- [x] 强化约束条件展示

### Phase 4: 验证增强（1小时）✅ 已完成

**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/change_applier.go`

- [x] 添加时间冲突验证
- [x] 添加超时验证
- [x] 拒绝不符合约束的AI输出
- [x] 实现ValidateScheduleDraft函数
- [x] 实现ValidateScheduleChanges函数

### Phase 5: 配置支持（0.5小时）✅ 已完成

**文件**: `config/agents/rostering-agent.yml`

- [x] 添加scheduling_constraints配置项
- [x] max_daily_hours: 12.0
- [x] min_rest_hours: 12.0
- [x] allow_oncall_overlap: true
- [x] strict_time_check: true
- [x] check_consecutive_shifts: true

### Phase 6: 测试验证（待实施）

- [ ] 单元测试：时间计算函数
- [ ] 单元测试：重叠检测
- [ ] 集成测试：避免冲突场景
- [ ] 集成测试：超时保护

---

## 四、验收标准

### 4.1 功能验收
- [ ] LLM能够看到所有班次的时间信息
- [ ] LLM能够看到人员当日已安排的班次
- [ ] 系统拒绝导致时间冲突的排班
- [ ] 系统拒绝导致超时的排班
- [ ] 同一人员同一天不再被安排7-8个冲突班次

### 4.2 性能验收
- [ ] 上下文构建 < 50ms
- [ ] 时间重叠检测 < 5ms
- [ ] 不影响任务执行性能

### 4.3 代码质量
- [ ] 所有新增函数有单元测试
- [ ] 测试覆盖率 > 80%
- [ ] 无编译警告
- [ ] 代码符合规范

---

## 五、风险与应对

| 风险 | 概率 | 影响 | 应对措施 |
|-----|------|------|---------|
| Prompt过长超Token限制 | 中 | 高 | 只传递当前日期的排班状态，简化输出格式 |
| LLM仍不遵守约束 | 中 | 高 | 后端强制验证，拒绝不合规输出 |
| 跨夜班次计算错误 | 低 | 中 | 详尽单元测试覆盖 |
| 性能影响 | 低 | 低 | 缓存班次映射，避免重复计算 |

---

## 六、配置项

建议在 `config/agents/rostering-agent.yml` 添加：

```yaml
scheduling_constraints:
  max_daily_hours: 12.0      # 单日最大工作时长
  min_rest_hours: 12.0       # 班次间最小休息
  allow_oncall_overlap: true # 允许待命班叠加
  strict_time_check: true    # 启用严格时间检查
```

---

## 七、预期效果

### 改进前
```
张三: 早班、中班、晚班、待命、支援、培训、值班
      (7个班次，存在时间冲突，总计32小时)
```

### 改进后
```
张三: 早班 (08:00-16:00, 8h)
李四: 中班 (14:00-22:00, 8h)
王五: 待命 (全天, 0h实际工时) + 支援 (10:00-12:00, 2h) ← 合理叠加
```

---

**文档维护者**: GitHub Copilot  
**最后更新**: 2026-01-27

---

## 八、修复记录（2026-01-27）

### 🔧 核心问题修复

#### 1. 修复日期计算Bug ✅
**问题**: `calculatePreviousDate`函数使用简化实现，2月份会错误计算为30天  
**修复**: 使用Go标准库`time.Time`精确处理日期计算  
**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/scheduling_context_builder.go`

#### 2. 集成V3上下文到ExecuteTodoTask ✅
**问题**: `ExecuteTodoTask`方法完全没有使用V3的排班上下文，导致LLM无法看到班次时间和时间冲突  
**修复**:
- 修改方法签名，新增`allShifts`和`workingDraft`参数
- 在方法内部调用`buildSchedulingContextForTodo`构建V3上下文
- 将上下文传递给`buildTodoExecutionUserPrompt`

**文件**: `agents/rostering/internal/service/scheduling_ai.go`

#### 3. 增强Prompt展示V3信息 ✅
**问题**: Prompt中缺少班次时间表和人员排班状态的详细展示  
**修复**:
- 修改`buildTodoExecutionUserPrompt`方法签名，新增`allShifts`和`schedulingContext`参数
- 在Prompt开头添加【⚠️ 强制约束】区块，突出显示硬性限制
- 添加【📋 班次时间表】区块，展示所有班次的时间安排
- 添加【👥 人员当日排班状态】区块，分组显示已排班/未排班人员
- 对已排班人员，详细展示每个班次的时间、工时、错误和警告

**效果**: LLM现在可以清楚看到：
- 所有班次的时间段（08:00-16:00等）
- 人员已安排的其他班次及时间
- 时间冲突情况（标记为❌）
- 超时警告（标记为⚠️）

#### 4. 实现后端强制校验 ✅
**问题**: 缺少后端强制校验，即使LLM生成违规排班也会被接受  
**修复**:
- 在`change_applier.go`中实现`ValidateScheduleDraft`函数
- 实现`ValidateScheduleChanges`函数用于应用前校验
- 校验内容包括：
  - 时间冲突检测（同一人同一天的班次时间重叠）
  - 超时检测（当日总工时超过限制）
  - 跨日休息检测（连续工作日的休息时间不足）
- 返回详细的ValidationResult，包含错误列表、警告列表和统计信息

**文件**: `agents/rostering/internal/workflow/schedule_v3/utils/change_applier.go`

#### 5. 添加配置项支持 ✅
**问题**: 约束值（12小时工时、12小时休息）硬编码在代码中  
**修复**:
- 在`rostering-agent.yml`中添加`scheduling_constraints`配置节
- 支持配置项：
  - `max_daily_hours`: 每日最大工时（默认12.0）
  - `min_rest_hours`: 最小休息时间（默认12.0）
  - `allow_oncall_overlap`: 是否允许待命班叠加（默认true）
  - `strict_time_check`: 是否启用严格校验（默认true）
  - `check_consecutive_shifts`: 是否检查跨日连班（默认true）

**文件**: `config/agents/rostering-agent.yml`

### 📊 修复效果

修复前：
```
问题：同一人被安排7-8个冲突班次
原因：LLM看不到班次时间信息和冲突情况
```

修复后：
```
✅ LLM可以看到所有班次的时间段
✅ LLM可以看到人员已排班情况和时间冲突
✅ 后端强制校验拒绝违规排班
✅ 约束条件可配置化
```

### 🎯 待后续工作

1. ~~**调用链修改**: 需要修改所有调用`ExecuteTodoTask`的地方，传递新增的参数~~ ✅ 已完成
2. ~~**配置读取**: 需要在代码中读取配置文件中的约束值，替换硬编码的12.0~~ ✅ 已完成
3. **单元测试**: 为新增的校验功能添加单元测试
4. **集成测试**: 验证完整链路的正确性

### ✅ 已完成工作

#### 配置读取实现 (2026-01-27)

**修改文件**:
- `agents/rostering/config/config.go`: 添加`SchedulingConstraintsConfig`结构体
- `agents/rostering/internal/workflow/schedule_v3/utils/scheduling_context_builder.go`: 接受配置参数
- `agents/rostering/internal/workflow/schedule_v3/utils/progressive_task_executor.go`: 读取配置并传递
- `agents/rostering/internal/workflow/schedule_v3/core/actions.go`: 注入configurator
- `agents/rostering/internal/wiring/container.go`: 注册configurator为服务
- `agents/rostering/internal/workflow/schedule_v2/core/actions.go`: V2兼容性修复（传递nil）

**实现细节**:
```go
// 1. 配置结构体定义
type SchedulingConstraintsConfig struct {
    MaxDailyHours          float64
    MinRestHours           float64
    AllowOncallOverlap     bool
    StrictTimeCheck        bool
    CheckConsecutiveShifts bool
}

// 2. V3读取配置
cfg := e.configurator.GetConfig()
maxDailyHours := cfg.SchedulingConstraints.MaxDailyHours
minRestHours := cfg.SchedulingConstraints.MinRestHours

// 3. V2保持兼容
schedulingAIService.ExecuteTodoTask(..., nil, nil)
```

---

**修复执行者**: GitHub Copilot  
**修复日期**: 2026-01-27
