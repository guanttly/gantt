# 固定人员配置 - 奇偶周功能说明

## 功能概述

固定人员配置的"按周重复"模式现在支持三种周期选项：
- **每周**：每周都执行（默认）
- **奇数周**：仅在奇数周执行（第1、3、5、7...周）
- **偶数周**：仅在偶数周执行（第2、4、6、8...周）

这个功能使得两个人可以轮流在同样的时间上班，实现AB轮换。

## 使用场景

### 场景1：两人AB轮换

**需求**：张三和李四轮流在每周一、三、五上白班

**配置**：
1. 为张三配置：
   - 模式：按周重复
   - 周期：奇数周
   - 周几：周一、三、五

2. 为李四配置：
   - 模式：按周重复
   - 周期：偶数周
   - 周几：周一、三、五

**结果**：
```
第1周（奇数周）：
  周一 - 张三
  周三 - 张三
  周五 - 张三

第2周（偶数周）：
  周一 - 李四
  周三 - 李四
  周五 - 李四

第3周（奇数周）：
  周一 - 张三
  周三 - 张三
  周五 - 张三
  
... 循环往复
```

### 场景2：分组轮换

**需求**：A组和B组每隔一周轮换夜班

**配置**：
1. 为A组所有成员配置：
   - 模式：按周重复
   - 周期：奇数周
   - 周几：周一至周日（全周）

2. 为B组所有成员配置：
   - 模式：按周重复
   - 周期：偶数周
   - 周几：周一至周日（全周）

## 周编号计算规则

### ISO 8601 周编号标准

本系统使用 ISO 8601 周编号标准来判断奇偶周：

**规则**：
- 每年的第一周是包含第一个周四的那一周
- 周从周一开始，周日结束
- 周编号从1开始（1-53）

**示例（2025年）**：

```
2025年1月
日  一  二  三  四  五  六
             1   2   3   4    ← 第1周（奇数周）
 5   6   7   8   9  10  11    ← 第2周（偶数周）
12  13  14  15  16  17  18    ← 第3周（奇数周）
19  20  21  22  23  24  25    ← 第4周（偶数周）
26  27  28  29  30  31        ← 第5周（奇数周）

2025年2月
日  一  二  三  四  五  六
                         1    ← 第5周（奇数周）
 2   3   4   5   6   7   8    ← 第6周（偶数周）
 9  10  11  12  13  14  15    ← 第7周（奇数周）
16  17  18  19  20  21  22    ← 第8周（偶数周）
23  24  25  26  27  28        ← 第9周（奇数周）
```

**重点**：
- 奇数周：1、3、5、7、9...周
- 偶数周：2、4、6、8、10...周
- 跨年时周编号会重置为1

## 数据库设计

### 字段定义

```sql
CREATE TABLE shift_fixed_assignments (
    -- ... 其他字段
    weekdays JSON COMMENT '周几上班，例如 [1,3,5] 表示周一、三、五',
    week_pattern ENUM('every', 'odd', 'even') DEFAULT 'every' 
        COMMENT '周重复模式: every=每周, odd=奇数周, even=偶数周',
    -- ... 其他字段
);
```

### 示例数据

```sql
-- 张三：奇数周的周一、三、五
INSERT INTO shift_fixed_assignments 
(id, shift_id, staff_id, pattern_type, weekdays, week_pattern, is_active) 
VALUES
('assign-1', 'shift-001', 'staff-zhang', 'weekly', JSON_ARRAY(1, 3, 5), 'odd', TRUE);

-- 李四：偶数周的周一、三、五
INSERT INTO shift_fixed_assignments 
(id, shift_id, staff_id, pattern_type, weekdays, week_pattern, is_active) 
VALUES
('assign-2', 'shift-001', 'staff-li', 'weekly', JSON_ARRAY(1, 3, 5), 'even', TRUE);
```

## 后端实现

### 模型定义

```go
// WeekPattern 周重复模式
type WeekPattern string

const (
    WeekPatternEvery WeekPattern = "every" // 每周
    WeekPatternOdd   WeekPattern = "odd"   // 奇数周
    WeekPatternEven  WeekPattern = "even"  // 偶数周
)

type ShiftFixedAssignment struct {
    // ... 其他字段
    Weekdays    []int       `json:"weekdays"`
    WeekPattern WeekPattern `json:"weekPattern"`
    // ... 其他字段
}
```

### 计算逻辑

```go
func calculateDatesForAssignment(assign *model.ShiftFixedAssignment, start, end time.Time) []time.Time {
    var dates []time.Time
    
    switch assign.PatternType {
    case model.PatternTypeWeekly:
        // ... 周几匹配逻辑
        
        // 检查周模式（奇数周/偶数周）
        if assign.WeekPattern != "" && assign.WeekPattern != model.WeekPatternEvery {
            _, weekNum := d.ISOWeek() // 获取 ISO 周编号（1-53）
            isOddWeek := weekNum%2 == 1
            
            if assign.WeekPattern == model.WeekPatternOdd && !isOddWeek {
                continue // 配置为奇数周，但当前是偶数周
            }
            if assign.WeekPattern == model.WeekPatternEven && isOddWeek {
                continue // 配置为偶数周，但当前是奇数周
            }
        }
        
        dates = append(dates, d)
    }
    
    return dates
}
```

## 前端实现

### 类型定义

```typescript
/** 周重复模式 */
type WeekPattern = 'every' | 'odd' | 'even'

interface FixedAssignment {
  weekdays?: number[]
  weekPattern?: WeekPattern
  // ... 其他字段
}
```

### UI组件

```vue
<template>
  <!-- 周期选择 -->
  <el-form-item label="周期">
    <el-radio-group v-model="form.weekPattern">
      <el-radio value="every">
        每周 <span class="tip">每周都执行</span>
      </el-radio>
      <el-radio value="odd">
        奇数周 <span class="tip">第1、3、5、7...周</span>
      </el-radio>
      <el-radio value="even">
        偶数周 <span class="tip">第2、4、6、8...周</span>
      </el-radio>
    </el-radio-group>
  </el-form-item>
  
  <!-- 周几选择 -->
  <el-form-item label="选择周几">
    <el-checkbox-group v-model="form.weekdays">
      <el-checkbox :value="1">周一</el-checkbox>
      <el-checkbox :value="2">周二</el-checkbox>
      <!-- ... -->
    </el-checkbox-group>
  </el-form-item>
</template>
```

### 显示逻辑

```typescript
function formatPattern(assignment: Shift.FixedAssignment): string {
  if (assignment.patternType === 'weekly' && assignment.weekdays) {
    const days = assignment.weekdays.map(d => `周${d}`).join('、')
    
    // 添加周期信息
    let pattern = ''
    if (assignment.weekPattern === 'odd') {
      pattern = '（奇数周）'
    }
    else if (assignment.weekPattern === 'even') {
      pattern = '（偶数周）'
    }
    
    return days + pattern
    // 示例输出：
    // "周一、周三、周五（奇数周）"
    // "周二、周四（偶数周）"
    // "周一、周三、周五"（每周，不显示标注）
  }
  // ...
}
```

## API接口

### 创建固定人员配置

**请求**：
```json
POST /api/v1/shifts/{shiftId}/fixed-assignments

{
  "shiftId": "shift-001",
  "assignments": [
    {
      "staffId": "staff-zhang",
      "patternType": "weekly",
      "weekdays": [1, 3, 5],
      "weekPattern": "odd"
    },
    {
      "staffId": "staff-li",
      "patternType": "weekly",
      "weekdays": [1, 3, 5],
      "weekPattern": "even"
    }
  ]
}
```

### 查询固定人员配置

**响应**：
```json
GET /api/v1/shifts/{shiftId}/fixed-assignments

[
  {
    "id": "assign-1",
    "staffId": "staff-zhang",
    "staffName": "张三",
    "patternType": "weekly",
    "weekdays": [1, 3, 5],
    "weekPattern": "odd",
    "isActive": true
  },
  {
    "id": "assign-2",
    "staffId": "staff-li",
    "staffName": "李四",
    "patternType": "weekly",
    "weekdays": [1, 3, 5],
    "weekPattern": "even",
    "isActive": true
  }
]
```

## 测试场景

### 测试1：奇偶周轮换

**配置**：
- 张三：奇数周的周一
- 李四：偶数周的周一

**排班周期**：2025-01-01 到 2025-01-31

**预期结果**：
```
2025-01-06（周一，第2周-偶数周）：李四
2025-01-13（周一，第3周-奇数周）：张三
2025-01-20（周一，第4周-偶数周）：李四
2025-01-27（周一，第5周-奇数周）：张三
```

### 测试2：跨年周编号

**配置**：
- 张三：奇数周的周五

**排班周期**：2024-12-27 到 2025-01-10

**预期结果**：
```
2024年:
  12-27（周五，2024年第52周-偶数周）：不排班
  
2025年:
  01-03（周五，2025年第1周-奇数周）：张三 ✓
  01-10（周五，2025年第2周-偶数周）：不排班
```

### 测试3：每周+奇偶周混合

**配置**：
- 张三：每周的周一、三（普通配置）
- 李四：奇数周的周五
- 王五：偶数周的周五

**排班周期**：任意两周

**预期结果**：
```
第N周（奇数周）：
  周一 - 张三 ✓
  周三 - 张三 ✓
  周五 - 李四 ✓

第N+1周（偶数周）：
  周一 - 张三 ✓
  周三 - 张三 ✓
  周五 - 王五 ✓
```

## 注意事项

### 1. 周编号计算

- **跨年问题**：每年的周编号会重置，第一周可能包含上一年的最后几天
- **不同标准**：确保前后端都使用 ISO 8601 标准
- **时区问题**：周编号计算基于UTC时间

### 2. 生效时间

如果配置了生效时间范围（`start_date` 和 `end_date`），奇偶周判断：
- 先判断日期是否在生效范围内
- 再判断是否匹配周几
- 最后判断是否匹配奇偶周

### 3. 默认值

- 如果不指定 `week_pattern`，默认为 `every`（每周）
- 前端UI默认选中"每周"选项
- 旧数据迁移：历史数据的 `week_pattern` 为 NULL 或空字符串时，视为 `every`

### 4. 兼容性

- 数据库字段设置为 `DEFAULT 'every'`，确保向后兼容
- API 中 `weekPattern` 为可选字段
- 旧配置在升级后仍然正常工作（被视为"每周"）

## FAQ

### Q1: 奇数周和偶数周是如何确定的？

A: 使用 ISO 8601 标准的周编号。每年第一周是包含第一个周四的那一周，周编号从1开始。奇数周就是周编号为奇数的周（1、3、5...），偶数周是周编号为偶数的周（2、4、6...）。

### Q2: 跨年时周编号会怎样？

A: 每年的周编号会重置。例如2024年的最后一周可能是第52周或53周，2025年的第一周从1开始。这意味着跨年时，奇偶周的判断会基于新年的周编号。

### Q3: 如果两个人都配置了同一天会怎样？

A: 系统会将两个人都排入该班次。固定人员配置不会互相排斥，而是累加。工作流会自动识别并优先处理所有固定人员配置。

### Q4: 可以配置"每两周"吗？

A: 当前支持的是奇数周/偶数周模式，本质上就是每两周重复一次。如果需要更灵活的周期（如每3周、每4周），需要额外开发。

### Q5: 修改周期会影响历史数据吗？

A: 不会。固定人员配置只影响未来的排班。修改配置后，新的排班会使用新配置，但已生成的历史排班不会改变。

## 总结

奇偶周功能为固定人员配置增加了重要的灵活性：
- ✅ 支持AB轮换：两人轮流在同样的时间上班
- ✅ 基于标准：使用ISO 8601周编号，清晰明确
- ✅ 易于理解：奇数周/偶数周概念直观
- ✅ 向后兼容：不影响现有配置，默认为"每周"
- ✅ 完整实现：数据库、后端、前端、UI全部支持

使用场景广泛，特别适合需要人员轮换的排班需求。

---

**文档版本**：v1.1  
**更新时间**：2025-12-17  
**功能状态**：已实现并测试通过

