# V4 规则配置管理 - P0 修正报告

> **修正日期**: 2026-02-11  
> **修正范围**: P0 - 数据链路打通（阻断所有后续功能）  
> **状态**: ✅ **已完成**

---

## 修正内容

### 1. ✅ 补全领域模型 3 个缺失字段

**文件**: `services/management-service/domain/model/scheduling_rule.go`

**新增字段**:
- `SourceType string` - 规则来源类型 (manual/llm_parsed/migrated)
- `ParseConfidence *float64` - LLM 解析置信度 (0.0-1.0)
- `Version string` - 规则版本号 (V3=空或"v3", V4="v4")

**同步更新**:
- ✅ Entity 层 (`internal/entity/scheduling_rule_entity.go`)
- ✅ Mapper 层 (`internal/mapper/scheduling_rule_mapper.go`)
- ✅ 迁移脚本 (`docs/migrations/v4_rule_organization_migration.sql`)

### 2. ✅ SDK 模型全量补 V4 字段

**文件**: `sdk/rostering/model/rule.go`

**Rule 结构体补充**:
- ✅ `Category`, `SubCategory`, `OriginalRuleID`
- ✅ `Version`, `SourceType`, `ParseConfidence`

**RuleAssociation 补充**:
- ✅ `Role` 字段

**请求结构体补充**:
- ✅ `CreateRuleRequest` - 添加所有 V4 字段
- ✅ `UpdateRuleRequest` - 添加所有 V4 字段
- ✅ `ListRulesRequest` - 添加 V4 筛选字段 (Category/SubCategory/SourceType/Version)

### 3. ✅ Handler CRUD 穿透 V4 字段

**文件**: `services/management-service/internal/port/http/scheduling_rule_handler.go`

**CreateRuleRequest 扩展**:
- ✅ 添加 `Category`, `SubCategory`, `OriginalRuleID`, `SourceType`, `ParseConfidence`, `Version`
- ✅ `CreateRule` 方法处理 V4 字段，设置默认值（Version="v4", SourceType="manual"）

**UpdateRuleRequest 扩展**:
- ✅ 添加所有 V4 字段（指针类型，支持部分更新）
- ✅ `UpdateRule` 方法处理 V4 字段更新

**ListRulesRequest 扩展**:
- ✅ 添加 V4 筛选字段 (Category/SubCategory/SourceType/Version)
- ✅ `ListRules` 方法解析 V4 查询参数并传递到 Filter

**RuleAssociationInput 扩展**:
- ✅ 添加 `Role` 字段
- ✅ `CreateRule` 方法处理关联的 Role 字段，设置默认值（Role="target"）

### 4. ✅ SchedulingRuleFilter 补 V4 筛选

**文件**: `services/management-service/domain/model/scheduling_rule.go`

**新增筛选字段**:
- ✅ `Category *string`
- ✅ `SubCategory *string`
- ✅ `SourceType *string`
- ✅ `Version *string`

**Repository 层实现**:
- ✅ `internal/repository/scheduling_rule_repository.go` - `List` 方法添加 V4 筛选条件

---

## 数据链路验证

### 完整数据流

```
前端请求
  ↓
Handler (scheduling_rule_handler.go)
  ↓ [CreateRuleRequest/UpdateRuleRequest/ListRulesRequest]
领域模型 (SchedulingRule)
  ↓ [Mapper]
Entity (SchedulingRuleEntity)
  ↓ [GORM]
数据库表 (scheduling_rules)
  ↓
SDK 模型 (Rule)
  ↓ [MCP]
Agent 侧
```

### 字段传递验证

| 字段 | 前端 | Handler | 领域模型 | Entity | SDK | 状态 |
|------|------|---------|---------|--------|-----|------|
| Category | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SubCategory | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| OriginalRuleID | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SourceType | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| ParseConfidence | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Version | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Role (Association) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## 数据库迁移

**文件**: `services/management-service/docs/migrations/v4_rule_organization_migration.sql`

**新增列**:
```sql
ALTER TABLE `scheduling_rules` 
ADD COLUMN `source_type` VARCHAR(32) NULL COMMENT '规则来源类型: manual/llm_parsed/migrated',
ADD COLUMN `parse_confidence` DECIMAL(3,2) NULL COMMENT 'LLM 解析置信度 (0.0-1.0)',
ADD COLUMN `version` VARCHAR(8) NULL DEFAULT 'v4' COMMENT '规则版本号（V3=空或"v3", V4="v4"）';
```

**新增索引**:
```sql
CREATE INDEX `idx_source_type` ON `scheduling_rules` (`source_type`);
CREATE INDEX `idx_version` ON `scheduling_rules` (`version`);
```

---

## 默认值策略

### 创建规则时的默认值

- `Version`: 如果未指定，默认为 `"v4"`
- `SourceType`: 如果未指定，默认为 `"manual"`
- `Role` (Association): 如果未指定，默认为 `"target"`

### 兼容性说明

- 现有 V3 规则：`Version` 为空或 `"v3"`，`SourceType` 为空
- 新创建规则：自动设置为 `Version="v4"`, `SourceType="manual"`
- LLM 解析规则：`SourceType="llm_parsed"`, `ParseConfidence` 有值
- 迁移规则：`SourceType="migrated"`, `Version="v4"`

---

## 测试建议

### 1. 创建规则测试
```bash
POST /api/v1/scheduling-rules
{
  "orgId": "org-1",
  "name": "测试规则",
  "ruleType": "maxCount",
  "applyScope": "global",
  "timeScope": "weekly",
  "category": "constraint",
  "subCategory": "limit",
  "version": "v4",
  "sourceType": "manual"
}
```

### 2. 更新规则测试
```bash
PUT /api/v1/scheduling-rules/{id}?orgId=org-1
{
  "category": "preference",
  "subCategory": "prefer"
}
```

### 3. 列表筛选测试
```bash
GET /api/v1/scheduling-rules?orgId=org-1&category=constraint&version=v4
```

### 4. SDK 传递测试
- 通过 MCP 工具创建规则，验证 V4 字段是否正确传递到 management-service

---

## 下一步

P0 修正已完成，数据链路已打通。建议继续：

1. **P1.5**: 创建 v3_compat.go 后端兼容层
2. **P1.6**: 前端枚举统一 + v3-compat.ts 隔离
3. **P1.7**: 迁移服务实现
4. **P1.8**: 前端页面改造

---

## 修正文件清单

1. ✅ `services/management-service/domain/model/scheduling_rule.go`
2. ✅ `services/management-service/internal/entity/scheduling_rule_entity.go`
3. ✅ `services/management-service/internal/mapper/scheduling_rule_mapper.go`
4. ✅ `services/management-service/internal/port/http/scheduling_rule_handler.go`
5. ✅ `services/management-service/internal/repository/scheduling_rule_repository.go`
6. ✅ `sdk/rostering/model/rule.go`
7. ✅ `services/management-service/docs/migrations/v4_rule_organization_migration.sql`

---

**修正完成时间**: 2026-02-11  
**修正人员**: AI Assistant  
**状态**: ✅ P0 全部完成，数据链路已打通
