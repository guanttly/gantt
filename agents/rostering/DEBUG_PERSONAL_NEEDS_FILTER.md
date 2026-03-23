# 个人需求过滤调试指南

## 问题描述
日志显示有28人的负向需求，其中包含非候选人员（如"董卉妍"），但候选人员只有68人。

## 数据流追踪

### 1. PersonalNeeds 的来源
**文件**: `schedule_v3/create/actions.go:854`
```go
personalNeeds := ExtractPersonalNeeds(createCtx.Rules, createCtx.StaffList)
createCtx.PersonalNeeds = personalNeeds
```

**ExtractPersonalNeeds** (`schedule_v3/create/helpers.go:12`)
- 输入：`rules`, `staffList`
- 输出：`map[string][]*PersonalNeed`，**key 是 staffID**
- 逻辑：从规则的 Associations 中提取 staffID

**关键代码** (line 33):
```go
if assoc.AssociationType == "staff" {
    staffID := assoc.AssociationID
    // ...
    if result[staffID] == nil {
        result[staffID] = make([]*PersonalNeed, 0)
    }
    result[staffID] = append(result[staffID], need)
}
```

### 2. StaffList 的来源
**文件**: `common/schedule_context_loader.go:55`

`LoadScheduleBasicContext` 返回的 `StaffList` 是**从班次关联的分组中获取的人员**：

```go
// 167-188行
shiftStaffMap := make(map[string]*d_model.Employee)
for _, shift := range shifts {
    members, err := service.GetShiftGroupMembers(ctx, shift.ID)
    // ... 添加到 shiftStaffMap
}
```

**结论**: `StaffList` 是候选人员，使用 **staffID** 作为key。

### 3. 过滤逻辑
**文件**: `progressive_task_executor.go:2521-2536`

```go
// 构建候选人员ID集合（关键过滤）
candidateStaffIDs := make(map[string]bool)
for _, staff := range staffList {
    candidateStaffIDs[staff.ID] = true  // ← 使用 ID
}

unavailableStaffMap := e.buildUnavailableStaffMap(personalNeeds, task.TargetDates, "", "")

for staffID, datesMap := range unavailableStaffMap.StaffDates {
    // 【关键修复】只处理候选人员的不可用信息
    if !candidateStaffIDs[staffID] {  // ← 用 staffID 过滤
        continue
    }
    // ...
}
```

## 可能的问题

### 问题1: PersonalNeeds 包含了非候选人员的规则
**原因**: `ExtractPersonalNeeds(createCtx.Rules, createCtx.StaffList)` 的输入参数:
- `createCtx.Rules`: **所有规则**
- `createCtx.StaffList`: 候选人员列表

但是！`ExtractPersonalNeeds` 函数**并没有真正使用 `staffList` 来过滤规则**！

看 `helpers.go:12-44`：
```go
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
    result := make(map[string][]*PersonalNeed)

    // 构建人员ID到姓名的映射 （仅用于映射，不用于过滤！）
    staffNameMap := make(map[string]string)
    for _, staff := range staffList {
        staffNameMap[staff.ID] = staff.Name
    }

    // 从规则中提取个人需求
    for _, rule := range rules {
        // ... 遍历所有规则的所有 associations
        for _, assoc := range rule.Associations {
            if assoc.AssociationType == "staff" {
                staffID := assoc.AssociationID
                need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
                // ← 这里没有检查 staffID 是否在 staffList 中！
                if need != nil {
                    result[staffID] = append(result[staffID], need)
                }
            }
        }
    }
    return result
}
```

**问题确认**: `ExtractPersonalNeeds` 会提取**所有规则**中关联的人员需求，即使这些人员不在候选列表中！

### 问题2: Rules 的范围
`createCtx.Rules` 是如何加载的？

**文件**: `common/schedule_context_loader.go:121`
```go
rules, err := loadRules(ctx, service, orgID, selectedShifts, shiftStaffList, startDate)
```

**查看 loadRules**:
```go
// loadRules 加载规则（已去重）
func loadRules(..., shiftStaffList []*d_model.Employee, ...) ([]*d_model.Rule, error) {
    // ...实现
}
```

需要检查 `loadRules` 是否正确过滤了规则。

## 解决方案

### 方案1: 修复 ExtractPersonalNeeds 函数
在 `schedule_v3/create/helpers.go` 中添加过滤逻辑：

```go
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
    result := make(map[string][]*PersonalNeed)

    // 构建候选人员ID集合（用于过滤）
    candidateStaffIDs := make(map[string]bool)
    staffNameMap := make(map[string]string)
    for _, staff := range staffList {
        candidateStaffIDs[staff.ID] = true  // ← 添加过滤集合
        staffNameMap[staff.ID] = staff.Name
    }

    // 从规则中提取个人需求
    for _, rule := range rules {
        if rule == nil || !rule.IsActive {
            continue
        }

        if len(rule.Associations) > 0 {
            for _, assoc := range rule.Associations {
                if assoc.AssociationType == "staff" {
                    staffID := assoc.AssociationID
                    
                    // ← 添加过滤：只提取候选人员的需求
                    if !candidateStaffIDs[staffID] {
                        continue
                    }
                    
                    need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
                    if need != nil {
                        if result[staffID] == nil {
                            result[staffID] = make([]*PersonalNeed, 0)
                        }
                        result[staffID] = append(result[staffID], need)
                    }
                }
            }
        }
    }

    return result
}
```

### 方案2: 在 loadRules 时过滤
确保 `loadRules` 只加载与候选人员相关的规则。

## 验证步骤

1. **添加调试日志** - 在关键位置添加日志：
   ```go
   // 在 ExtractPersonalNeeds 中
   logger.Info("ExtractPersonalNeeds: Processing rules",
       "totalRules", len(rules),
       "candidateStaff", len(staffList))
   
   // 在 associations 循环中
   logger.Debug("Found personal need",
       "staffID", staffID,
       "isCandidate", candidateStaffIDs[staffID],
       "ruleName", rule.Name)
   ```

2. **检查日志输出** - 运行排班任务，查看：
   - PersonalNeeds map 的大小
   - 是否包含非候选人员的ID
   - 候选人员的数量

3. **验证过滤效果** - 检查最终prompt中：
   - "共 X 人" 的数量是否正确
   - 是否还包含非候选人员（如"董卉妍"）

## 预期结果

修复后：
- PersonalNeeds map 只包含候选人员的ID
- 日志显示的负向需求人数 ≤ 候选人员数量（68人）
- 不再出现非候选人员的名字（如"董卉妍"）

## 测试用例

假设：
- 候选人员：68人（staff_1 到 staff_68）
- 全部人员：149人（staff_1 到 staff_149）
- 规则关联：staff_21（董卉妍）有负向需求，但她不在候选列表中

**期望**:
- 修复前：显示"共 28 人"，包含"董卉妍 (ID:staff_21)"
- 修复后：显示"共 5 人"（或更少），不包含"董卉妍"
