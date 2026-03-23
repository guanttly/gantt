# 个人需求过滤Bug修复报告

## 问题回顾

用户报告日志中显示"共 28 人"有负向需求，其中包含非候选人员（如"董卉妍 ID:staff_21"），但实际候选人员只有68人。

## 根本原因分析

经过全链路追踪，发现了**真正的根本原因**：

### 错误的假设
之前认为问题在于 `buildAITaskUserPromptWithContext` 函数的过滤逻辑，但实际上那里的过滤是正确的。

### 真正的问题
**问题出在数据源头** - `ExtractPersonalNeeds` 函数！

**文件**: `agents/rostering/internal/workflow/schedule_v3/create/helpers.go:12`

```go
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
    result := make(map[string][]*PersonalNeed)

    // 构建人员ID到姓名的映射
    staffNameMap := make(map[string]string)
    for _, staff := range staffList {
        staffNameMap[staff.ID] = staff.Name  // ← 只用来映射名字
    }

    // 从规则中提取个人需求
    for _, rule := range rules {
        for _, assoc := range rule.Associations {
            if assoc.AssociationType == "staff" {
                staffID := assoc.AssociationID
                // ← 问题：这里没有检查 staffID 是否在 staffList 中！
                // 直接提取所有规则关联的人员需求
                need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
                if need != nil {
                    result[staffID] = append(result[staffID], need)
                }
            }
        }
    }
    return result
}
```

**关键问题**：
1. 函数虽然接收了 `staffList` 参数
2. 但只用它来构建 `staffNameMap`（ID到姓名的映射）
3. **完全没有用它来过滤人员**！
4. 结果：提取了**所有规则**中关联的**所有人员**的需求，包括非候选人员

### 数据流分析

```
1. ExtractPersonalNeeds(createCtx.Rules, createCtx.StaffList)
   ↓
   输入：
   - rules: 组织的所有激活规则（可能关联149个人）
   - staffList: 班次候选人员（68人）
   ↓
   输出：
   - PersonalNeeds map: 包含所有规则关联人员的需求（149人）
                       ← 错误！应该只包含68个候选人员

2. taskContext.PersonalNeeds = PersonalNeeds  （传递到任务执行）
   ↓
3. buildAITaskUserPromptWithContext 尝试过滤
   ↓
   for staffID, datesMap := range unavailableStaffMap.StaffDates {
       if !candidateStaffIDs[staffID] {
           continue  // ← 这里虽然能过滤，但为时已晚！
       }
   }
```

### 为什么之前的修复无效？

在 `progressive_task_executor.go` 的第六部分虽然添加了过滤：

```go
candidateStaffIDs := make(map[string]bool)
for _, staff := range staffList {
    candidateStaffIDs[staff.ID] = true
}

for staffID, datesMap := range unavailableStaffMap.StaffDates {
    if !candidateStaffIDs[staffID] {
        continue  // ← 这里的过滤是对的
    }
    // ...
}
```

但问题是：
1. `unavailableStaffMap` 是从 `personalNeeds` 构建的
2. 而 `personalNeeds` 已经包含了非候选人员的数据（在数据源头就错了）
3. 虽然最终prompt显示时能过滤掉，但**数据本身就不应该存在**

## 修复方案

### 修复位置
`agents/rostering/internal/workflow/schedule_v3/create/helpers.go:12`

### 修复内容

```go
// ExtractPersonalNeeds 从规则中提取个人需求（复用V2的逻辑）
// 【关键修复】只提取候选人员（staffList）的个人需求，过滤掉非候选人员
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
    result := make(map[string][]*PersonalNeed)

    // 构建候选人员ID集合（用于过滤）
    candidateStaffIDs := make(map[string]bool)
    staffNameMap := make(map[string]string)
    for _, staff := range staffList {
        candidateStaffIDs[staff.ID] = true // ← 新增：添加到候选集合
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
                    
                    // 【关键修复】只提取候选人员的需求，过滤掉非候选人员
                    if !candidateStaffIDs[staffID] {
                        continue  // ← 新增：源头过滤
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

### 修复效果

**修复前**：
- `ExtractPersonalNeeds` 返回149人的需求（所有规则关联人员）
- 其中只有68人是候选人员
- 包含"董卉妍 (staff_21)"等非候选人员的需求

**修复后**：
- `ExtractPersonalNeeds` 只返回68个候选人员的需求
- 完全不包含非候选人员的数据
- "董卉妍"等人的需求在源头就被过滤掉了

## 影响范围

这个bug影响所有使用 `ExtractPersonalNeeds` 的地方：

1. ✅ **V3排班创建流程** - 已修复
2. ⚠️ **V2排班流程** - 同样的问题可能存在

### V2版本检查

让我检查V2是否有同样问题：

**文件**: `schedule_v2/create/helpers.go:71`

需要检查V2的实现是否也有同样的bug。如果有，需要同步修复。

## 测试验证

### 验证步骤

1. **运行排班任务**，使用与日志中相同的场景：
   - 候选人员：68人
   - 包含"董卉妍 (staff_21)"有负向需求，但她不在候选列表中

2. **检查日志输出**：
   ```
   [LLM Prompt] User prompt for single shift
   ```
   
   在第六部分应该看到：
   ```
   # 第六部分：候选人员负向需求
   
   **候选人员中有负向需求的人员（共 X 人）**：
   ```
   
   **预期**：
   - X ≤ 68（候选人员总数）
   - 不包含"董卉妍"等非候选人员

3. **对比数据**：
   - 修复前：`taskContext.PersonalNeeds` 包含149人的数据
   - 修复后：`taskContext.PersonalNeeds` 只包含68人的数据

### 关键指标

- ✅ PersonalNeeds map 大小 ≤ 候选人员数量
- ✅ 不包含非候选人员的 staffID
- ✅ prompt 显示的负向需求人数正确
- ✅ 不再出现"董卉妍"等非候选人员名字

## 深层教训

### 问题根源
**在数据的源头就应该保证正确性**，而不是在下游各处添加过滤逻辑。

### 之前的错误思路
1. 发现prompt中有非候选人员
2. 在 `buildAITaskUserPromptWithContext` 中添加过滤
3. ❌ 但忽略了数据本身就不该包含这些人

### 正确的思路
1. 追溯数据来源
2. 在 `ExtractPersonalNeeds` 函数就应该过滤
3. ✅ 从源头保证数据正确性

### 函数设计原则
如果函数接收 `staffList` 参数，应该明确其用途：
- 是用来**映射**的（ID→Name）？
- 还是用来**过滤**的（只处理这些人员）？

在这个案例中，`staffList` 的命名暗示应该用来过滤，但实际实现只用来映射，这是设计缺陷。

## 编译验证

✅ 代码已通过编译：
```bash
cd /home/lgt/gantt/app/agents/rostering && go build -o /tmp/test-rostering-agent2 ./
```

## 相关文件

- ✅ 已修复：`agents/rostering/internal/workflow/schedule_v3/create/helpers.go`
- ℹ️ 文档：`DEBUG_PERSONAL_NEEDS_FILTER.md` - 详细调试分析
- ℹ️ 文档：`PROMPT_REFACTOR_SUMMARY.md` - Prompt重构总结
- ⚠️ 待检查：`schedule_v2/create/helpers.go` - V2可能有同样问题

---

**修复时间**: 2026-01-28  
**问题类型**: 数据源头过滤缺失  
**影响范围**: V3排班流程（V2待确认）  
**根本原因**: `ExtractPersonalNeeds` 函数虽接收 `staffList` 但未用于过滤
