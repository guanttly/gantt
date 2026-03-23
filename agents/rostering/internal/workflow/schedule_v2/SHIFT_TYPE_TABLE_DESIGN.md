# 班次类型独立表设计总结

## 📋 设计方案概述

根据您的建议，我们将班次类型从硬编码字符串改为独立的数据库表管理，实现更灵活和可配置的架构。

## 🏗️ 核心架构

```
┌─────────────────────────┐
│   Frontend (管理端)     │
│  - 常规班次              │
│  - 加班班次              │
│  - 备班班次              │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  API Layer              │
│  GET /shift-types       │ ← 获取类型列表
│  POST /shifts           │ ← 创建班次（带 shift_type_id）
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Service Layer          │
│  - ShiftTypeService     │
│  - ShiftService         │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Database               │
│  ┌─────────────────┐    │
│  │  shift_types    │    │
│  │  - id (PK)      │    │
│  │  - code         │    │
│  │  - name         │    │
│  │  - priority     │    │
│  │  - phase        │    │
│  └────────┬────────┘    │
│           │ 1:N         │
│  ┌────────▼────────┐    │
│  │  shifts         │    │
│  │  - shift_type_id│    │
│  │  - name         │    │
│  └─────────────────┘    │
└─────────────────────────┘
```

## 📊 数据库设计

### 主表：shift_types

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(36) | 主键 |
| org_id | VARCHAR(36) | 组织ID |
| code | VARCHAR(50) | 类型编码（regular, overtime） |
| name | VARCHAR(100) | 显示名称（常规班次、加班班次） |
| scheduling_priority | INT | 排班优先级（1-100，越小越优先） |
| workflow_phase | VARCHAR(50) | 工作流阶段（normal/special/fixed/fill） |
| color | VARCHAR(20) | 显示颜色（hex） |
| icon | VARCHAR(50) | 图标名称 |
| is_ai_scheduling | BOOLEAN | 是否需要AI排班 |
| is_fixed_schedule | BOOLEAN | 是否固定排班 |
| is_overtime | BOOLEAN | 是否算加班 |
| requires_special_skill | BOOLEAN | 是否需要特殊技能 |
| is_active | BOOLEAN | 是否启用 |
| is_system | BOOLEAN | 是否系统内置（不可删除） |

### 关联表：shifts

```sql
ALTER TABLE shifts 
    ADD COLUMN shift_type_id VARCHAR(36),
    ADD INDEX idx_shift_type (shift_type_id);
```

## 🎯 系统内置类型

| Code | Name | Priority | Phase | AI排班 | 说明 |
|------|------|----------|-------|--------|------|
| fixed | 固定班次 | 10 | fixed | ✗ | 每周固定人员 |
| special | 特殊班次 | 30 | special | ✓ | 有技能要求 |
| overtime | 加班班次 | 31 | special | ✓ | 节假日加班 |
| standby | 备班班次 | 32 | special | ✓ | 待命或应急 |
| regular | 常规班次 | 50 | normal | ✓ | 日常工作 |
| research | 科研班次 | 70 | research | ✓ | 科研学习 |
| fill | 填充班次 | 90 | fill | ✗ | 补充不足 |

## 🔄 前后端对接方案

### 方案一：前端直接使用 code

```typescript
// 前端
const shiftTypeCodes = ['regular', 'overtime', 'standby'];

// 创建班次时，先查询 shift_type_id
const shiftType = await fetchShiftTypeByCode('regular');
const newShift = {
  name: '白班',
  shiftTypeId: shiftType.id,  // 传递ID
  startTime: '08:00',
  endTime: '16:00'
};
```

### 方案二：前端获取类型列表（推荐）

```typescript
// 1. 页面加载时获取类型列表
const shiftTypes = await fetchShiftTypes({
  orgId: currentOrg.id,
  includeSystem: true
});

// 返回：
// [
//   {id: 'uuid-1', code: 'regular', name: '常规班次', color: '#409EFF', priority: 50},
//   {id: 'uuid-2', code: 'overtime', name: '加班班次', color: '#F56C6C', priority: 31},
//   {id: 'uuid-3', code: 'standby', name: '备班班次', color: '#C71585', priority: 32}
// ]

// 2. 下拉框展示
<Select v-model="newShift.shiftTypeId">
  <Option 
    v-for="type in shiftTypes" 
    :key="type.id" 
    :value="type.id"
    :label="type.name"
  >
    <span :style="{color: type.color}">{{ type.name }}</span>
  </Option>
</Select>

// 3. 创建班次
const newShift = {
  name: '白班',
  shiftTypeId: selectedTypeId,  // 直接使用ID
  startTime: '08:00',
  endTime: '16:00'
};
```

### 兼容层（可选）

如果前端暂时无法修改，后端可提供兼容层：

```go
// API 接收
type CreateShiftRequest struct {
    Name        string `json:"name"`
    Type        string `json:"type"`        // 旧字段（可选）
    ShiftTypeID string `json:"shiftTypeId"` // 新字段（可选）
}

// 转换逻辑
func (r *CreateShiftRequest) ToCommand(ctx context.Context, repo repository.ShiftTypeRepository) {
    // 优先使用 shiftTypeId
    if r.ShiftTypeID != "" {
        return r.ShiftTypeID
    }
    
    // 如果只有 type，从数据库查询对应的 shift_type_id
    if r.Type != "" {
        shiftType, _ := repo.GetByCode(ctx, orgID, r.Type)
        if shiftType != nil {
            r.ShiftTypeID = shiftType.ID
        }
    }
}
```

## 💻 代码实现

### 已创建文件

1. ✅ **数据库脚本**
   - `agents/rostering/docs/database/shift_type_schema.sql`
   - 包含表创建、数据迁移、初始化数据

2. ✅ **SDK 模型**
   - `sdk/rostering/model/shift_type.go`
   - `ShiftType` 结构体定义
   - 请求/响应模型

3. ✅ **Repository 接口**
   - `agents/rostering/domain/repository/shift_type_repository.go`
   - 数据访问层接口定义

4. ✅ **Service 层**
   - `agents/rostering/domain/service/shift_type_service.go`
   - 业务逻辑实现
   - CRUD 操作
   - 统计查询

5. ✅ **API Handler**
   - `agents/rostering/api/http/handler/shift_type_handler.go`
   - RESTful API 实现
   - 路由定义

6. ✅ **迁移指南**
   - `SHIFT_TYPE_MIGRATION_GUIDE.md`
   - 详细迁移步骤
   - 代码示例

7. ✅ **设计文档更新**
   - `design.md` 已更新 V2.1 章节

## 🔌 API 接口

### 获取班次类型列表

```http
GET /api/v1/shift-types
Query Parameters:
  - org_id: string (required)
  - workflow_phase: string (optional)
  - is_active: boolean (optional)
  - include_system: boolean (default: true)

Response:
{
  "code": 200,
  "data": [
    {
      "id": "uuid-1",
      "code": "regular",
      "name": "常规班次",
      "schedulingPriority": 50,
      "workflowPhase": "normal",
      "color": "#409EFF",
      "isAiScheduling": true
    }
  ]
}
```

### 创建班次类型

```http
POST /api/v1/shift-types
Body:
{
  "orgId": "org-uuid",
  "code": "custom_shift",
  "name": "自定义班次",
  "schedulingPriority": 40,
  "workflowPhase": "normal",
  "color": "#67C23A",
  "isAiScheduling": true
}
```

### 创建班次（使用类型）

```http
POST /api/v1/shifts
Body:
{
  "name": "白班",
  "shiftTypeId": "uuid-1",  // 关联班次类型
  "startTime": "08:00",
  "endTime": "16:00"
}
```

## 🔧 工作流集成

### 动态类型查询

```go
// 旧方式（硬编码）
if shift.Type == "special" {
    // 特殊班次处理
}

// 新方式（动态查询）
shiftType, _ := shiftTypeService.GetShiftType(ctx, shift.ShiftTypeID)
if shiftType.WorkflowPhase == "special" {
    // 特殊班次处理
}
```

### 按优先级排序

```go
// 获取组织的所有班次类型（按优先级排序）
shiftTypes, _ := shiftTypeService.GetShiftTypesByPriority(ctx, orgID)

// 按类型分组并排序班次
phaseGroups, _ := ClassifyShiftsByTypeWithSort(ctx, shifts, shiftTypeService)
for _, group := range phaseGroups {
    log.Printf("处理 %s 阶段，优先级: %d", group.Phase, group.Priority)
    for _, shift := range group.Shifts {
        // 处理班次...
    }
}
```

## 📈 优势总结

### 1. 灵活性
- ✅ 无需修改代码即可调整类型和优先级
- ✅ 支持组织自定义班次类型
- ✅ 可动态添加新类型属性

### 2. 可维护性
- ✅ 类型配置集中管理
- ✅ 减少硬编码
- ✅ 易于扩展

### 3. 用户体验
- ✅ 可视化类型配置界面
- ✅ 丰富的类型展示（颜色、图标）
- ✅ 统一的前后端类型定义

### 4. 可扩展性
- ✅ 可添加更多业务属性
- ✅ 支持多租户隔离
- ✅ 便于集成外部系统

## 🚀 实施计划

### Phase 1: 数据库层（1-2天）
- [ ] 执行 SQL 脚本创建表
- [ ] 插入系统内置类型
- [ ] 数据迁移（现有班次）
- [ ] 验证数据完整性

### Phase 2: 后端实现（3-4天）
- [x] SDK 模型定义
- [x] Repository 接口定义
- [ ] Repository MySQL 实现
- [x] Service 层实现
- [x] API Handler 实现
- [ ] 单元测试
- [ ] 集成测试

### Phase 3: 工作流更新（2-3天）
- [ ] 更新 helpers.go（动态类型查询）
- [ ] 更新 actions.go（使用 ShiftTypeService）
- [ ] 更新 Context（添加服务依赖）
- [ ] 测试工作流

### Phase 4: 前端集成（3-4天）
- [ ] 创建班次类型管理页面
- [ ] 班次表单集成类型选择器
- [ ] 班次列表展示类型信息
- [ ] UI/UX 调整
- [ ] 前端测试

### Phase 5: 测试与上线（2-3天）
- [ ] 功能测试
- [ ] 回归测试
- [ ] 性能测试
- [ ] 文档完善
- [ ] 灰度发布
- [ ] 全量上线

**总计：11-16 工作日**

## 📝 注意事项

1. **数据迁移**
   - 确保现有班次数据完整迁移到新表
   - 保留旧字段一段时间用于回滚

2. **性能优化**
   - 批量查询班次类型，避免 N+1 问题
   - 考虑添加缓存层

3. **权限控制**
   - 系统内置类型不可删除和修改
   - 限制组织自定义类型数量

4. **向后兼容**
   - API 提供兼容层支持旧版前端
   - 逐步迁移前端代码

5. **监控与日志**
   - 记录类型变更操作
   - 监控工作流性能

## 📚 相关文档

- 详细迁移指南：`SHIFT_TYPE_MIGRATION_GUIDE.md`
- 数据库脚本：`agents/rostering/docs/database/shift_type_schema.sql`
- API 文档：自动生成的 Swagger 文档
- 工作流设计：`design.md`

---

**设计完成时间**：2025-12-17  
**设计人员**：AI Assistant  
**版本**：V2.1

