# V4 规则组织排班系统实现总结

## 概述

本文档总结了 V4 版本排班系统的完整实现，包括后端规则解析、规则组织、确定性规则引擎，以及前端页面改造和可视化功能。

## 实现时间线

- **数据库模型扩展** ✅
- **规则解析服务** ✅
- **确定性规则引擎** ✅
- **规则组织器** ✅
- **V4 排班工作流** ✅
- **后端 API** ✅
- **依赖关系仓储** ✅
- **前端 API 接口** ✅
- **前端页面改造** ✅
- **依赖关系可视化** ✅

## 后端实现

### 1. 数据库模型扩展

**文件位置：**
- `services/management-service/domain/model/scheduling_rule.go`
- `services/management-service/internal/entity/scheduling_rule_entity.go`
- `services/management-service/docs/migrations/v4_rule_organization_migration.sql`

**新增字段：**
- `Category`: 规则分类（constraint/preference/dependency）
- `SubCategory`: 规则子分类（forbid/must/limit/prefer/suggest/source/resource/order）
- `OriginalRuleID`: 原始规则ID（如果是从语义化规则解析出来的）
- `Role`: 关联角色（target/source/reference）

**新增表：**
- `rule_dependencies`: 规则依赖关系表
- `rule_conflicts`: 规则冲突关系表
- `shift_dependencies`: 班次依赖关系表

### 2. 规则解析服务

**文件位置：**
- `services/management-service/domain/service/rule_parser.go`
- `services/management-service/internal/service/rule_parser_service.go`

**功能：**
- 使用 LLM 解析自然语言规则描述
- 三层验证机制：结构验证、语义一致性验证、模拟验证
- 自动识别规则分类、依赖关系和冲突关系
- 批量保存解析后的规则

**API 端点：**
- `POST /api/v1/scheduling-rules/parse` - 解析语义化规则
- `POST /api/v1/scheduling-rules/batch` - 批量保存解析后的规则

### 3. 确定性规则引擎

**文件位置：**
- `agents/rostering/internal/engine/`

**核心组件：**
- `CandidateFilter`: 候选人过滤器（替代 LLM-1）
- `RuleMatcher`: 规则匹配器（替代 LLM-2）
- `ConstraintChecker`: 约束检查器（替代 LLM-3）
- `PreferenceScorer`: 偏好评分器
- `ScheduleValidator`: 排班验证器（替代 LLM-5）

**优势：**
- 减少 LLM 调用次数（从 5 次减少到 1 次）
- 提高稳定性和一致性
- 降低 LLM 成本
- 提供结构化的排班上下文

### 4. 规则组织器

**文件位置：**
- `agents/rostering/internal/workflow/schedule_v4/executor/rule_organizer.go`
- `agents/rostering/internal/workflow/schedule_v4/executor/dependency_analyzer.go`
- `services/management-service/internal/service/rule_organizer_service.go`

**功能：**
- 规则分类（constraint/preference/dependency）
- 依赖关系分析（source/resource/order）
- 冲突关系检测
- 拓扑排序确定执行顺序

**API 端点：**
- `POST /api/v1/scheduling-rules/organize` - 组织规则

### 5. V4 排班工作流

**文件位置：**
- `agents/rostering/internal/workflow/schedule_v4/`

**工作流定义：**
- `state/schedule/create_v4.go` - 状态定义
- `create/definition.go` - 工作流定义
- `executor/v4_executor.go` - 执行器

**特点：**
- 依赖驱动的排班顺序
- 确定性规则引擎预处理
- 最小化 LLM 调用

### 6. 依赖关系仓储

**文件位置：**
- `services/management-service/domain/repository/rule_dependency_repository.go`
- `services/management-service/internal/repository/rule_dependency_repository.go`

**仓储接口：**
- `IRuleDependencyRepository`: 规则依赖关系仓储
- `IRuleConflictRepository`: 规则冲突关系仓储
- `IShiftDependencyRepository`: 班次依赖关系仓储

## 前端实现

### 1. API 接口和类型定义

**文件位置：**
- `frontend/web/src/api/scheduling-rule/index.ts`
- `frontend/web/src/api/scheduling-rule/model.d.ts`

**新增 API：**
- `parseRule()`: 解析语义化规则
- `batchSaveRules()`: 批量保存解析后的规则
- `organizeRules()`: 组织规则

**新增类型：**
- `ParseRuleRequest/Response`: 规则解析请求/响应
- `RuleOrganizationResult`: 规则组织结果
- `ClassifiedRuleInfo`: 分类后的规则信息
- `RuleDependencyInfo`: 规则依赖关系信息
- `RuleConflictInfo`: 规则冲突关系信息

### 2. 语义化规则解析组件

**文件位置：**
- `frontend/web/src/pages/management/scheduling-rule/components/RuleParseDialog.vue`

**功能：**
- 语义化规则输入界面
- 实时解析和预览
- 显示解析后的规则列表
- 显示依赖关系和冲突关系
- 批量保存解析结果

### 3. 规则列表页面改造

**文件位置：**
- `frontend/web/src/pages/management/scheduling-rule/index.vue`

**新增功能：**
- 显示规则分类列
- "语义化规则（V4）"按钮
- "组织规则（V4）"按钮
- 规则组织结果对话框（多标签页展示）

### 4. 依赖关系可视化组件

**文件位置：**
- `frontend/web/src/pages/management/scheduling-rule/components/RuleDependencyGraph.vue`

**功能：**
- 使用 ECharts 绘制规则依赖关系图
- 节点颜色区分规则分类（约束/偏好/依赖）
- 箭头表示依赖方向
- 虚线表示冲突关系
- 支持交互（悬停显示详情、拖拽、缩放）

### 5. 工具函数扩展

**文件位置：**
- `frontend/web/src/pages/management/scheduling-rule/logic.ts`

**新增函数：**
- `getCategoryText()`: 获取规则分类文本
- `getCategoryTagType()`: 获取规则分类标签类型

## 核心特性

### 1. 语义化规则输入

用户可以使用自然语言描述规则，系统自动解析为结构化规则：

**示例：**
- "张三每周最多工作5天"
- "夜班后至少休息1天"
- "优先安排周末休息"

### 2. 规则自动分类

系统自动将规则分类为：
- **约束规则（Constraint）**: 硬约束，必须遵守
- **偏好规则（Preference）**: 软约束，尽量满足
- **依赖规则（Dependency）**: 定义规则间的依赖关系

### 3. 依赖关系识别

系统自动识别规则间的依赖关系：
- **时间依赖（time）**: 基于时间顺序的依赖
- **数据依赖（source）**: 基于数据来源的依赖
- **资源依赖（resource）**: 基于资源分配的依赖
- **顺序依赖（order）**: 基于执行顺序的依赖

### 4. 冲突关系检测

系统自动检测规则冲突：
- **排他冲突（exclusive）**: 规则互斥
- **资源冲突（resource）**: 资源竞争
- **时间冲突（time）**: 时间冲突
- **频率冲突（frequency）**: 频率冲突

### 5. 拓扑排序

系统使用拓扑排序算法确定规则和班次的执行顺序，确保依赖关系得到正确处理。

### 6. 确定性规则引擎

通过代码实现的确定性规则引擎，替代了 V3 中的多个 LLM 调用：
- **LLM-1 → CandidateFilter**: 候选人过滤
- **LLM-2 → RuleMatcher**: 规则匹配
- **LLM-3 → ConstraintChecker**: 约束检查
- **LLM-5 → ScheduleValidator**: 排班验证

**保留的 LLM 调用：**
- **LLM-4**: 核心排班决策（基于结构化上下文）

## 技术亮点

1. **三层验证机制**: 确保解析后的规则质量
2. **依赖驱动调度**: 基于依赖关系的智能排班顺序
3. **可视化展示**: 直观展示规则依赖关系和冲突
4. **类型安全**: 完整的 TypeScript 类型定义
5. **可扩展性**: 模块化设计，易于扩展

## 使用指南

### 后端使用

1. **解析语义化规则**:
```go
resp, err := ruleParserService.ParseRule(ctx, &ParseRuleRequest{
    OrgID: "org-1",
    Name: "规则名称",
    RuleDescription: "张三每周最多工作5天",
    Priority: 5,
})
```

2. **组织规则**:
```go
result, err := ruleOrganizerService.OrganizeRules(ctx, "org-1")
```

### 前端使用

1. **打开语义化规则解析对话框**:
   - 点击"语义化规则（V4）"按钮
   - 输入规则描述
   - 点击"解析规则"
   - 查看解析结果
   - 点击"保存规则"

2. **组织规则**:
   - 点击"组织规则（V4）"按钮
   - 查看规则组织结果
   - 在"依赖关系图"标签页查看可视化图表

## 后续优化建议

1. **性能优化**:
   - 规则解析结果缓存
   - 依赖关系图增量更新

2. **用户体验**:
   - 规则解析历史记录
   - 规则模板库
   - 批量导入规则

3. **功能扩展**:
   - 规则版本管理
   - 规则测试工具
   - 规则影响分析

## 总结

V4 版本排班系统通过引入规则组织、依赖分析和确定性规则引擎，显著提升了排班的稳定性、一致性和效率。语义化规则输入降低了使用门槛，可视化展示提高了可理解性。整个系统已经完成实现，可以投入使用。
