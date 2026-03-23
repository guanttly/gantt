# 排班创建工作流 SOP (Standard Operating Procedure)

## 一、流程概览

**工作流名称**: `Workflow_Schedule_Create`  
**入口触发**: 用户点击智能排班按钮  
**架构说明**: 
- 摈弃图数据库
- 排班规则集成在 SDK 中
- 前端交互式确认流程

---

## 二、流程状态转换图

```
用户点击智能排班
    ↓
State_1_ConfirmingSchedulePeriod (确认排班周期)
    ├─→ [用户确认周期] → State_2_ConfirmingShifts
    └─→ [用户修改周期] → State_1_ConfirmingSchedulePeriod
    
State_2_ConfirmingShifts (确认排班班次)
    ├─→ [用户确认班次] → State_3_ConfirmingStaffCount
    ├─→ [用户禁用/启用班次] → State_2_ConfirmingShifts
    └─→ [用户取消] → State_Failed
    
State_3_ConfirmingStaffCount (确认班次人数)
    ├─→ [用户确认] → State_4_RetrievingStaff
    ├─→ [用户微调人数] → State_3_ConfirmingStaffCount
    └─→ [用户取消] → State_Failed
    
State_4_RetrievingStaff (检索班次相关人员)
    ↓ [人员检索完成]
State_5_RetrievingRules (检索班次相关规则)
    ↓ [规则检索完成]
    
State_6_GeneratingSchedule (遍历班次生成排班)
    ├─→ [班次循环] → State_6.1_QueryingShiftGroupRules
    └─→ [AI处理失败] → State_Failed
    
State_6.1_QueryingShiftGroupRules (查询班次所属分组规则)
    ↓
State_6.2_QueryingShiftStaffRules (查询班次相关人员规则)
    ↓
State_6.3_GeneratingShiftSchedule (AI生成班次排班)
    ├─→ [生成成功] → State_6.4_MergingDraft
    └─→ [AI失败] → State_Failed
    
State_6.4_MergingDraft (合并到总草案)
    ├─→ [还有未处理班次] → State_6_GeneratingSchedule (下一个班次)
    └─→ [所有班次完成] → State_7_ConfirmingDraft
    
State_7_ConfirmingDraft (预览并确认排班)
    ├─→ [用户确认] → State_8_SavingSchedule
    ├─→ [用户调整] → State_6_GeneratingSchedule (重新生成)
    └─→ [用户取消] → State_Failed
    
State_8_SavingSchedule (存储排班)
    ├─→ [保存成功] → State_Completed ✅
    └─→ [保存失败] → State_Failed ❌
```

---

## 三、详细流程说明

### 阶段 1: 确认排班周期

**用户交互**: 前端展示日期选择器  
**默认值**: 下周一到下周日 (7天)  
**验证规则**:
- 开始日期 ≥ 今天 - 30天
- 结束日期 ≤ 开始日期 + 90天
- 开始日期 ≤ 结束日期

**操作**: 确认 / 修改

---

### 阶段 2: 确认排班班次

**用户交互**: 前端弹出班次列表  
**数据来源**: SDK 查询所有可排班班次 (按优先级从低到高排序)  
**显示信息**: 
- 班次名称
- 班次代码
- 开始时间
- 结束时间
- 优先级
- 默认人数 (default_staff_count)

**用户操作**:
- 禁用某些班次 (不参与排班)
- 启用某些班次
- 确认班次列表

**输出**: 启用的班次列表

---

### 阶段 3: 确认班次人数

**用户交互**: 前端展示班次-日期人数配置表  
**默认值**: 每个班次的 `default_staff_count` 字段  
**显示格式**: 表格 (行=班次, 列=日期)

**用户操作**:
- 修改某天某班次的人数需求
- 批量设置某班次所有日期的人数
- 确认人数配置

**输出**: 
```go
shiftStaffRequirements := map[string]map[string]int{
    "shift_001": {
        "2025-11-18": 3,
        "2025-11-19": 5,
        ...
    },
}
```

---

### 阶段 4: 检索班次相关人员

**后端处理**: 调用 SDK 查询人员

**查询逻辑**:
1. 获取所有启用班次的关联分组
2. 查询这些分组下的所有人员
3. 过滤已请假的人员

**输出**: 
```go
staffList := []*Staff{
    {ID: "s1", Name: "张三", GroupIDs: []string{"g1", "g2"}},
    {ID: "s2", Name: "李四", GroupIDs: []string{"g1"}},
    ...
}
```

---

### 阶段 5: 检索班次相关规则

**后端处理**: 调用 SDK 查询规则

**规则类型**:
- 全局规则 (适用所有班次和人员)
- 班次规则 (特定班次的约束)
- 分组规则 (特定分组的约束)

**数据来源**: SDK 内置规则引擎

**输出**:
```go
rules := RulesContext{
    GlobalRules: []*Rule{...},
    ShiftRules: map[string][]*Rule{...},
    GroupRules: map[string][]*Rule{...},
}
```

---

### 阶段 6: 遍历班次生成排班

**核心逻辑**: 按优先级从低到高遍历所有启用的班次

#### 循环处理每个班次:

##### 6.1 查询班次所属分组规则
- 调用 SDK: `GetGroupRulesForShift(shiftID)`
- 输出: 该班次所属分组的所有约束规则

##### 6.2 查询班次相关人员规则
- 调用 SDK: `GetStaffRulesForShift(shiftID, staffIDs, startDate, endDate)`
- 包含: 请假记录、人员特殊约束
- 输出: `staffRules = map[staffID][]*Rule`

##### 6.3 汇总信息并调用 AI 生成
**输入数据**:
- 当前班次信息 (id, name, code, startTime, endTime, priority)
- 班次所属分组信息
- 可用人员列表 (已排除已排班人员)
- 班次人数需求 (shiftStaffRequirements[shiftID])
- 班次规则
- 分组规则
- 人员规则
- **上一轮排班结果和总结** (作为上下文)

**AI 输出**:
```json
{
  "schedule": {
    "2025-11-18": ["张三", "李四"],
    "2025-11-19": ["张三", "王五"],
    ...
  },
  "summary": "本轮排班情况说明，包括人员负荷、规则冲突、优化建议等"
}
```

##### 6.4 合并到总排班草案
- 策略: 同一排班周期内，一个人员只能被排到一个班次
- 更新已排班人员集合
- 累积 AI 总结 (供下一轮参考)
- 向前端推送进度: "⚙️ 正在排班 {currentIndex}/{totalShifts}: {shiftName}"

**循环直到所有班次处理完成**

---

### 阶段 7: 预览并确认排班

**用户交互**: 前端展示完整排班草案

**显示内容**:
- 每天每个班次的人员安排 (日历视图/表格视图)
- 人员统计 (每人工作天数、工作班次)
- 规则冲突提示 (如果有)
- AI 总结说明

**用户操作**:
- 预览排班
- 确认排班 → 进入保存阶段
- 调整排班 → 反馈调整意见 → 重新生成
- 取消 → 流程结束

---

### 阶段 8: 存储排班

**后端处理**: 保存排班到数据库

**流程**:
1. 转换格式: 排班草案 → 批量插入请求
2. 调用 SDK: `BatchUpsertSchedules(schedules)`
3. 冲突策略: Upsert (存在则更新)

**输出**: 
- 成功保存数量
- 失败记录 (如果有)

**完成**: 返回前端成功提示

---

## 四、核心数据结构

### ScheduleCreateContext (新版)

```go
type ScheduleCreateContext struct {
    // 排班周期
    StartDate          string                      // YYYY-MM-DD (默认下周一)
    EndDate            string                      // YYYY-MM-DD (默认下周日)
    
    // 班次配置
    AvailableShifts    []*Shift                    // SDK 查询的所有班次
    SelectedShifts     []*Shift                    // 用户选择的班次列表 (按优先级排序)
    ShiftStaffRequirements map[string]map[string]int // shiftId -> { date: count }
    
    // 数据收集
    StaffList          []*Staff                    // 可用人员列表
    GlobalRules        []*Rule                     // 全局规则
    ShiftRules         map[string][]*Rule          // shiftId -> rules
    GroupRules         map[string][]*Rule          // groupId -> rules
    
    // 生成阶段
    CurrentShiftIndex  int                         // 当前处理的班次索引
    ScheduledStaffSet  map[string]bool             // 已排班人员集合 (staffID -> true)
    DraftSchedule      *ScheduleDraft              // 排班草案
    AISummaries        []string                    // 各轮AI总结 (供下一轮参考)
    
    // 完成阶段
    FinalSchedule      *ScheduleDraft              // 最终排班
}
```

### 排班草案格式 (新版)

```json
{
  "startDate": "2025-11-18",
  "endDate": "2025-11-24",
  "shifts": {
    "早班": {
      "shiftId": "shift_001",
      "priority": 1,
      "days": {
        "2025-11-18": {
          "staff": ["张三", "李四", "王五"],
          "requiredCount": 3,
          "actualCount": 3
        },
        "2025-11-19": {
          "staff": ["张三", "赵六", "孙七"],
          "requiredCount": 5,
          "actualCount": 3
        },
        ...
      }
    },
    "中班": {
      "shiftId": "shift_002",
      "priority": 2,
      "days": { ... }
    }
  },
  "summary": "AI生成的整体排班说明",
  "staffStats": {
    "张三": { 
      "workDays": 5, 
      "shifts": ["早班", "早班", "早班", "早班", "早班"],
      "totalHours": 40
    },
    "李四": { 
      "workDays": 4, 
      "shifts": ["早班", "中班", "早班", "中班"],
      "totalHours": 32
    }
  },
  "conflicts": [
    {
      "date": "2025-11-19",
      "shift": "早班",
      "issue": "人数不足：需要5人，实际3人",
      "severity": "warning"
    }
  ]
}
```

---

## 五、SDK 接口说明

### 人员管理
```go
// 查询分组下的所有人员
ListStaffByGroups(ctx context.Context, orgID string, groupIDs []string) ([]*Staff, error)
```

### 班次管理
```go
// 查询所有班次 (按优先级排序)
ListShifts(ctx context.Context, orgID string) ([]*Shift, error)

// 查询班次所属分组
GetGroupsByShiftID(ctx context.Context, shiftID string) ([]*Group, error)
```

### 规则引擎
```go
// 查询全局规则
GetGlobalRules(ctx context.Context, orgID string) ([]*Rule, error)

// 查询班次规则
GetShiftRules(ctx context.Context, shiftID string) ([]*Rule, error)

// 查询分组规则
GetGroupRules(ctx context.Context, shiftID string) ([]*Rule, error)

// 查询人员规则 (含请假)
GetStaffRules(ctx context.Context, shiftID string, staffIDs []string, startDate, endDate string) (map[string][]*Rule, error)
```

### 排班存储
```go
// 批量保存排班
BatchUpsertSchedules(ctx context.Context, schedules []*Schedule) (*BatchResult, error)
```

---

## 六、性能指标

### 时间估算
- 阶段1 (确认周期): 用户操作，瞬时
- 阶段2 (确认班次): 查询班次 200-500ms + 用户操作
- 阶段3 (确认人数): 用户操作，瞬时
- 阶段4 (检索人员): 500-1500ms (取决于人员数量)
- 阶段5 (检索规则): 500-2000ms (SDK统一查询)
- **阶段6 (生成排班): 5-30秒** (取决于班次数量)
  - 单个班次: 1-5秒 (包含SDK规则查询 + AI生成)
  - 5个班次: 5-25秒
- 阶段7 (确认): 用户操作
- 阶段8 (保存): 1-3秒 (批量upsert)

**总耗时**: 7-37秒 (不含用户操作时间)

### 数据量估算
- 班次列表: 5-20个
- 人员列表: 10-100人
- 规则总数: 20-100条
- 排班记录: 50-700条 (7天 × 10班次 × 10人)

---

## 八、版本历史

| 版本 | 日期 | 修改内容 |
|------|------|---------|
| 2.0 | 2025-11-12 | 新版流程：摈弃图数据库，规则集成SDK，前端交互式确认 |
