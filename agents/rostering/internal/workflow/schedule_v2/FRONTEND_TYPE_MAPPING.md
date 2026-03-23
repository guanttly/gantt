# 前后端班次类型对齐说明

## 概述

前端管理界面和后端排班工作流使用了不同的班次分类体系。本文档说明两者的对应关系和自动映射机制。

## 类型对照表

| 前端类型 | 前端显示名称 | 后端工作流类型 | 处理方式 | 优先级 |
|---------|------------|--------------|---------|--------|
| `regular` | 常规班次 | `normal` | AI排班 | 4（中） |
| `overtime` | 加班班次 | `special` | AI排班，优先处理 | 3（高） |
| `standby` | 备班班次 | `special` | AI排班，优先处理 | 3（高） |
| （新增）- | 固定班次 | `fixed` | 自动填充 | 1（最高） |
| （新增）- | 科研班次 | `research` | AI排班 | 5（较低） |
| （新增）- | 填充班次 | `fill` | 填充处理 | 6（最低） |
| 空值 | 未设置 | `normal` | AI排班（默认） | 4（中） |

## 映射逻辑

### 自动映射

在 `ClassifyShiftsByType()` 函数中，系统会自动将前端类型映射到工作流类型：

```go
// 前端 -> 后端映射
ShiftTypeFrontendMapping = map[string]string{
    "regular":  "normal",   // 常规班次 -> 普通班次
    "overtime": "special",  // 加班班次 -> 特殊班次
    "standby":  "special",  // 备班班次 -> 特殊班次
}
```

### 映射原理

1. **常规班次** (`regular`) → **普通班次** (`normal`)
   - 这是最常见的班次类型
   - 按标准流程AI排班
   - 优先级：中等

2. **加班班次** (`overtime`) → **特殊班次** (`special`)
   - 加班通常有额外要求或限制
   - 需要优先安排
   - 优先级：高

3. **备班班次** (`standby`) → **特殊班次** (`special`)
   - 备班人员通常需要特定资质
   - 需要优先安排
   - 优先级：高

## 实际效果

### 排班顺序

当创建排班时，系统按以下顺序处理：

```
1. 固定班次 (fixed) - 如果有
2. 个人需求占位
3. 特殊班次 (special)
   ├─ 加班班次 (overtime) ───┐
   └─ 备班班次 (standby)  ───┤ → 统一作为 special 处理
4. 普通班次 (normal)
   └─ 常规班次 (regular) ────→ 映射为 normal
5. 科研班次 (research) - 如果有
6. 填充班次 (fill) - 如果有
```

### 示例场景

假设有以下班次：

```javascript
// 前端创建的班次
[
  { name: "白班", type: "regular" },      // → normal (优先级4)
  { name: "夜班", type: "regular" },      // → normal (优先级4)
  { name: "周末加班", type: "overtime" }, // → special (优先级3)
  { name: "备班", type: "standby" },      // → special (优先级3)
]
```

**排班处理顺序**：
1. 先处理"周末加班"和"备班"（special类型，优先级3）
2. 再处理"白班"和"夜班"（normal类型，优先级4）

## 向后兼容

### 已有数据

- ✅ 已有班次的 `type` 字段不需要修改
- ✅ 前端继续使用 `regular`/`overtime`/`standby`
- ✅ 工作流自动映射，无需干预

### 新增类型

如需在前端支持更多类型（如固定班次、科研班次），需要：

1. **前端添加类型选项**：

```typescript
// frontend/web/src/pages/management/shift/logic.ts
export const typeOptions: TypeOption[] = [
  { label: '常规班次', value: 'regular' },
  { label: '加班班次', value: 'overtime' },
  { label: '备班班次', value: 'standby' },
  { label: '固定班次', value: 'fixed' },    // 新增
  { label: '科研班次', value: 'research' },  // 新增
]
```

2. **后端无需修改**：
   - `fixed` 和 `research` 在工作流中已定义
   - 自动按正确优先级处理

## 配置建议

### 推荐设置

根据班次性质设置合适的类型：

| 班次名称 | 推荐类型 | 原因 |
|---------|---------|------|
| 白班、夜班、中班 | `regular` | 标准工作班次 |
| 周末值班、节假日加班 | `overtime` | 需要优先安排 |
| 应急备班、待命班 | `standby` | 需要特定人员 |
| 门诊固定班 | `fixed` | 每周固定人员 |
| 科研日、学习日 | `research` | 优先级较低 |

### 不推荐的做法

❌ **将所有班次都设为 `overtime`**
- 会导致所有班次优先级相同，失去优先级排序的意义

❌ **频繁更改班次类型**
- 影响历史数据的一致性
- 建议在创建班次时就设置正确

## 开发指南

### 添加新的前端类型

如果需要添加新的前端类型，在 `create_v2.go` 中更新映射：

```go
// 1. 定义前端类型常量
const (
    ShiftTypeEmergency = "emergency" // 应急班次
)

// 2. 添加映射关系
var ShiftTypeFrontendMapping = map[string]string{
    ShiftTypeRegular:   ShiftTypeNormal,
    ShiftTypeOvertime:  ShiftTypeSpecial,
    ShiftTypeStandby:   ShiftTypeSpecial,
    ShiftTypeEmergency: ShiftTypeSpecial, // 新增
}
```

### 调试映射

查看班次映射结果：

```go
classified := ClassifyShiftsByType(shifts)
for workflowType, shiftList := range classified {
    fmt.Printf("工作流类型 %s: %d 个班次\n", workflowType, len(shiftList))
    for _, shift := range shiftList {
        fmt.Printf("  - %s (原类型: %s)\n", shift.Name, shift.Type)
    }
}
```

## 常见问题

### Q: 为什么加班班次和备班班次都映射到 special？

**A**: 两者都有共同特点：
- 需要优先安排（人员可用性有限）
- 通常有额外要求（资质、经验等）
- 在排班逻辑中应该优先于常规班次

可以通过 `SchedulingPriority` 字段进一步区分它们的相对优先级。

### Q: 我想让某个常规班次优先排班，怎么办？

**A**: 有两个方法：
1. **改变类型**：将其设为 `overtime` 或 `standby`
2. **设置优先级**：保持类型为 `regular`，但设置较小的 `SchedulingPriority` 值

### Q: 前端类型改变会影响已有排班吗？

**A**: 不会。
- 已生成的排班数据不会改变
- 只影响新创建的排班
- 班次类型映射是实时的

### Q: 可以跳过类型映射，直接使用后端类型吗？

**A**: 可以。
- 前端直接使用 `normal`/`special`/`fixed` 等
- 这些类型不需要映射，直接识别
- 但需要前端配合修改

## 技术细节

### 映射执行位置

```go
// agents/rostering/internal/workflow/schedule_v2/create/helpers.go

func ClassifyShiftsByType(shifts []*d_model.Shift) map[string][]*d_model.Shift {
    for _, shift := range shifts {
        shiftType := shift.Type
        
        // 映射逻辑
        if mappedType, ok := ShiftTypeFrontendMapping[shiftType]; ok {
            shiftType = mappedType
        }
        
        result[shiftType] = append(result[shiftType], shift)
    }
    return result
}
```

### 映射表位置

```go
// agents/rostering/internal/workflow/state/schedule/create_v2.go

var ShiftTypeFrontendMapping = map[string]string{
    ShiftTypeRegular:  ShiftTypeNormal,
    ShiftTypeOvertime: ShiftTypeSpecial,
    ShiftTypeStandby:  ShiftTypeSpecial,
}
```

## 总结

✅ **前端继续使用现有类型**（`regular`/`overtime`/`standby`）  
✅ **后端自动映射处理**  
✅ **无需修改已有数据**  
✅ **优先级排序生效**  
✅ **向后兼容**  

---

**最后更新**：2025-12-17  
**版本**：V2.0.0  
**状态**：✅ 已实现

