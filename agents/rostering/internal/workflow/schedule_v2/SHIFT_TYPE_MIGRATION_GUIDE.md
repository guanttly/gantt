# 班次类型独立表迁移指南

## 📋 概述

将班次类型从硬编码字符串改为数据库表管理，实现更灵活的配置和扩展能力。

## 🎯 迁移目标

### 现有方案（硬编码）
```go
// 硬编码的班次类型常量
const (
    ShiftTypeFixed    = "fixed"
    ShiftTypeSpecial  = "special"
    ShiftTypeNormal   = "normal"
)

// 在 Shift 表的 Type 字段中存储字符串
shift.Type = "normal"
```

### 新方案（数据库表）
```go
// 班次类型独立表 shift_types
- 可配置的类型定义
- 动态优先级管理
- 工作流阶段映射
- 前后端统一类型定义

// Shift 表通过外键关联
shift.ShiftTypeID = "type-uuid"
```

## 📊 数据库变更

### 1. 创建班次类型表

```sql
-- 见 agents/rostering/docs/database/shift_type_schema.sql
CREATE TABLE shift_types (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    scheduling_priority INT NOT NULL,
    workflow_phase VARCHAR(50) NOT NULL,
    -- 更多字段...
);
```

### 2. 修改 Shifts 表

```sql
-- 添加外键字段
ALTER TABLE shifts 
    ADD COLUMN shift_type_id VARCHAR(36) COMMENT '班次类型ID',
    ADD INDEX idx_shift_type (shift_type_id);

-- 可选：保留旧字段用于兼容
-- ALTER TABLE shifts ADD COLUMN type_legacy VARCHAR(50);
```

### 3. 数据迁移

```sql
-- 步骤1：插入系统内置类型（见 SQL 文件）
INSERT INTO shift_types VALUES (...);

-- 步骤2：迁移现有班次的 type 到 shift_type_id
UPDATE shifts s
INNER JOIN shift_types st ON st.code = s.type
SET s.shift_type_id = st.id
WHERE s.shift_type_id IS NULL;
```

## 🔧 代码改动

### 1. Domain Model 更新

**SDK 模型**：`sdk/rostering/model/shift_type.go`

```go
type ShiftType struct {
    ID                   string
    Code                 string  // 类型编码
    Name                 string  // 显示名称
    SchedulingPriority   int     // 排班优先级
    WorkflowPhase        string  // 工作流阶段
    IsAIScheduling       bool    // 是否需要AI排班
    // ... 更多配置字段
}
```

**Shift 模型更新**：

```go
// 旧方式
type Shift struct {
    Type string `json:"type"` // 硬编码字符串
}

// 新方式
type Shift struct {
    ShiftTypeID string     `json:"shiftTypeId"` // 外键
    ShiftType   *ShiftType `json:"shiftType,omitempty"` // 关联对象（可选）
}
```

### 2. Repository 层

**新增接口**：`domain/repository/shift_type_repository.go`

```go
type ShiftTypeRepository interface {
    GetByID(ctx context.Context, id string) (*model.ShiftType, error)
    ListByOrgID(ctx context.Context, orgID string) ([]*model.ShiftType, error)
    ListByPriority(ctx context.Context, orgID string) ([]*model.ShiftType, error)
    // ... 更多方法
}
```

### 3. Service 层

**新增服务**：`domain/service/shift_type_service.go`

```go
type ShiftTypeService interface {
    CreateShiftType(ctx context.Context, req *model.CreateShiftTypeRequest) (*model.ShiftType, error)
    GetShiftTypesByPriority(ctx context.Context, orgID string) ([]*model.ShiftType, error)
    // ... 更多方法
}
```

### 4. Workflow 代码更新

**旧方式（硬编码）**：

```go
// 使用 Shift.Type 字段直接判断
func ClassifyShiftsByType(shifts []*Shift) map[string][]*Shift {
    result := make(map[string][]*Shift)
    for _, shift := range shifts {
        shiftType := shift.Type  // 直接使用字符串
        if shiftType == "" {
            shiftType = "normal"
        }
        result[shiftType] = append(result[shiftType], shift)
    }
    return result
}
```

**新方式（动态查询）**：

```go
// 从数据库查询类型信息
func ClassifyShiftsByType(
    ctx context.Context, 
    shifts []*Shift,
    shiftTypeService service.ShiftTypeService,
) (map[string][]*Shift, error) {
    result := make(map[string][]*Shift)
    
    // 批量获取班次类型信息
    typeMap := make(map[string]*ShiftType)
    for _, shift := range shifts {
        if shift.ShiftTypeID != "" {
            if _, ok := typeMap[shift.ShiftTypeID]; !ok {
                shiftType, err := shiftTypeService.GetShiftType(ctx, shift.ShiftTypeID)
                if err == nil && shiftType != nil {
                    typeMap[shift.ShiftTypeID] = shiftType
                }
            }
        }
    }
    
    // 按 WorkflowPhase 分类
    for _, shift := range shifts {
        phase := "normal" // 默认值
        if shiftType, ok := typeMap[shift.ShiftTypeID]; ok {
            phase = shiftType.WorkflowPhase
        }
        result[phase] = append(result[phase], shift)
    }
    
    return result, nil
}
```

**按优先级排序（新方式）**：

```go
type ShiftPhaseGroup struct {
    Phase    string
    Priority int
    Shifts   []*Shift
}

func ClassifyShiftsByTypeWithSort(
    ctx context.Context,
    shifts []*Shift,
    shiftTypeService service.ShiftTypeService,
) ([]*ShiftPhaseGroup, error) {
    // 获取类型信息
    typeMap := fetchShiftTypes(ctx, shifts, shiftTypeService)
    
    // 按阶段分组
    phaseGroups := make(map[string]*ShiftPhaseGroup)
    for _, shift := range shifts {
        phase := "normal"
        priority := 50
        
        if shiftType, ok := typeMap[shift.ShiftTypeID]; ok {
            phase = shiftType.WorkflowPhase
            priority = shiftType.SchedulingPriority
        }
        
        if _, ok := phaseGroups[phase]; !ok {
            phaseGroups[phase] = &ShiftPhaseGroup{
                Phase:    phase,
                Priority: priority,
                Shifts:   make([]*Shift, 0),
            }
        }
        
        phaseGroups[phase].Shifts = append(phaseGroups[phase].Shifts, shift)
    }
    
    // 按优先级排序
    result := make([]*ShiftPhaseGroup, 0, len(phaseGroups))
    for _, group := range phaseGroups {
        result = append(result, group)
    }
    
    sort.Slice(result, func(i, j int) bool {
        return result[i].Priority < result[j].Priority
    })
    
    return result, nil
}
```

### 5. Actions 代码更新示例

**旧方式**：

```go
func actStartSpecialShiftPhase(ctx context.Context, wctx engine.Context, payload any) error {
    v2ctx := wctx.Data().(*CreateV2Context)
    
    // 硬编码筛选
    phaseShifts := make([]*Shift, 0)
    for _, shift := range v2ctx.SelectedShifts {
        if shift.Type == "special" {  // 硬编码判断
            phaseShifts = append(phaseShifts, shift)
        }
    }
    
    // ...
}
```

**新方式**：

```go
func actStartSpecialShiftPhase(ctx context.Context, wctx engine.Context, payload any) error {
    v2ctx := wctx.Data().(*CreateV2Context)
    
    // 动态查询和分类
    shiftTypeService := getShiftTypeService(wctx) // 从依赖注入获取
    phaseGroups, err := ClassifyShiftsByTypeWithSort(ctx, v2ctx.SelectedShifts, shiftTypeService)
    if err != nil {
        return fmt.Errorf("分类班次失败: %w", err)
    }
    
    // 找到 special 阶段的班次
    var phaseShifts []*Shift
    for _, group := range phaseGroups {
        if group.Phase == "special" {
            phaseShifts = group.Shifts
            break
        }
    }
    
    if len(phaseShifts) == 0 {
        // 没有特殊班次，跳过该阶段
        return wctx.Send(ctx, CreateV2EventShiftPhaseComplete, nil)
    }
    
    // 保存到上下文
    v2ctx.CurrentPhase = "special"
    v2ctx.PhaseShiftList = phaseShifts
    v2ctx.CurrentShiftIndex = 0
    
    // 开始第一个班次的排班
    return startNextShiftInPhase(ctx, wctx, v2ctx)
}
```

## 🔄 前后端对接

### 前端发送数据

```typescript
// 创建班次时，前端发送 shift_type_id
interface CreateShiftRequest {
  name: string;
  shiftTypeId: string;  // 从下拉列表选择的类型ID
  startTime: string;
  endTime: string;
  // ...
}

// 前端需要先获取可用的班次类型列表
const shiftTypes = await fetchShiftTypes(orgId);
// [{id: "xxx", code: "regular", name: "常规班次", priority: 50}, ...]
```

### 后端返回数据

```go
// API 返回班次时，关联查询类型信息
type ShiftResponse struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    ShiftTypeID string            `json:"shiftTypeId"`
    ShiftType   *ShiftTypeInfo    `json:"shiftType"` // 关联的类型信息
}

type ShiftTypeInfo struct {
    ID       string `json:"id"`
    Code     string `json:"code"`
    Name     string `json:"name"`
    Priority int    `json:"priority"`
    Color    string `json:"color"`
}
```

## ✅ 迁移步骤

### 步骤 1：数据库变更（DevOps）
1. 执行 `shift_type_schema.sql` 创建表
2. 插入系统内置班次类型
3. 执行数据迁移脚本
4. 验证数据迁移结果

### 步骤 2：后端代码更新（Backend）
1. 创建 ShiftType 模型（SDK）
2. 实现 Repository 层
3. 实现 Service 层
4. 更新 Shift 相关 API
5. 添加 ShiftType 管理 API
6. 更新 Workflow 代码
7. 单元测试

### 步骤 3：前端代码更新（Frontend）
1. 更新班次类型选择组件
2. 调用班次类型列表 API
3. 提交时传递 `shiftTypeId`
4. 更新班次显示（颜色、图标）
5. 集成测试

### 步骤 4：验证与上线
1. 功能测试
2. 回归测试
3. 性能测试
4. 灰度发布
5. 全量上线

## 📝 配置管理

### 系统管理员功能
- 创建自定义班次类型
- 调整类型优先级
- 配置类型属性（颜色、图标、是否需要AI排班）
- 禁用/启用类型

### 界面示例
```
班次类型管理
┌────────────────────────────────────────────────────────┐
│ 名称        编码      优先级  工作流阶段  AI排班  操作  │
├────────────────────────────────────────────────────────┤
│ 固定班次    fixed     10      fixed       ✗      [编辑]│
│ 特殊班次    special   30      special     ✓      [编辑]│
│ 加班班次    overtime  31      special     ✓      [编辑]│
│ 常规班次    regular   50      normal      ✓      [编辑]│
│ 科研班次    research  70      research    ✓      [编辑]│
│ 填充班次    fill      90      fill        ✗      [编辑]│
│ + 新建班次类型                                         │
└────────────────────────────────────────────────────────┘
```

## 🎨 前端类型映射（兼容方案）

如果前端暂时无法修改，可以在后端提供兼容层：

```go
// API 层兼容前端的旧类型字段
type CreateShiftRequestCompat struct {
    Name         string `json:"name"`
    Type         string `json:"type"`         // 前端发送的旧字段
    ShiftTypeID  string `json:"shiftTypeId"`  // 新字段（可选）
    // ...
}

// 转换逻辑
func (r *CreateShiftRequestCompat) ToCommand() *CreateShiftCommand {
    shiftTypeID := r.ShiftTypeID
    
    // 如果前端没有传 shiftTypeId，从 type 映射
    if shiftTypeID == "" && r.Type != "" {
        shiftTypeID = mapFrontendTypeToID(r.Type, orgID)
    }
    
    return &CreateShiftCommand{
        ShiftTypeID: shiftTypeID,
        // ...
    }
}

func mapFrontendTypeToID(frontendType, orgID string) string {
    // 查询数据库，找到对应的 shift_type_id
    mapping := map[string]string{
        "regular":  "常规班次的ID",
        "overtime": "加班班次的ID",
        "standby":  "备班班次的ID",
    }
    return mapping[frontendType]
}
```

## 📈 优势与收益

### 灵活性提升
- ✅ 无需修改代码即可调整类型和优先级
- ✅ 支持组织自定义班次类型
- ✅ 前后端统一类型定义

### 可维护性提升
- ✅ 类型配置集中管理
- ✅ 减少硬编码
- ✅ 易于扩展新类型

### 用户体验提升
- ✅ 可视化类型配置界面
- ✅ 动态优先级调整
- ✅ 丰富的类型展示（颜色、图标）

## ⚠️ 注意事项

1. **向后兼容**：确保数据迁移完整，避免现有功能受影响
2. **性能优化**：批量查询班次类型，避免 N+1 查询问题
3. **缓存策略**：班次类型变更不频繁，可以考虑缓存
4. **权限控制**：系统内置类型不可删除和修改
5. **数据一致性**：外键约束保证数据完整性

## 🔗 相关文件

- 数据库脚本：`agents/rostering/docs/database/shift_type_schema.sql`
- SDK 模型：`sdk/rostering/model/shift_type.go`
- Repository：`agents/rostering/domain/repository/shift_type_repository.go`
- Service：`agents/rostering/domain/service/shift_type_service.go`
- 原工作流设计：`agents/rostering/internal/workflow/schedule_v2/design.md`

