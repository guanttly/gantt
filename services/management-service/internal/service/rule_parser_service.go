package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

// RuleParserServiceImpl 规则解析服务实现
type RuleParserServiceImpl struct {
	logger             logging.ILogger
	aiFactory          *ai.AIProviderFactory
	ruleRepo           repository.ISchedulingRuleRepository
	employeeRepo       repository.IEmployeeRepository
	shiftRepo          repository.IShiftRepository
	groupRepo          repository.IGroupRepository
	scopeRepo          repository.IRuleApplyScopeRepository
	ruleDependencyRepo repository.IRuleDependencyRepository
	ruleConflictRepo   repository.IRuleConflictRepository
	validator          *RuleParserValidator
	nameMatcher        *NameMatcher
}

// NewRuleParserService 创建规则解析服务实例
func NewRuleParserService(
	logger logging.ILogger,
	aiFactory *ai.AIProviderFactory,
	ruleRepo repository.ISchedulingRuleRepository,
	employeeRepo repository.IEmployeeRepository,
	shiftRepo repository.IShiftRepository,
	groupRepo repository.IGroupRepository,
	scopeRepo repository.IRuleApplyScopeRepository,
	ruleDependencyRepo repository.IRuleDependencyRepository,
	ruleConflictRepo repository.IRuleConflictRepository,
) service.IRuleParserService {
	validator := NewRuleParserValidator(logger, ruleRepo, employeeRepo, shiftRepo, groupRepo)
	nameMatcher := NewNameMatcher(logger, aiFactory, employeeRepo, shiftRepo, groupRepo)

	return &RuleParserServiceImpl{
		logger:             logger,
		aiFactory:          aiFactory,
		ruleRepo:           ruleRepo,
		employeeRepo:       employeeRepo,
		shiftRepo:          shiftRepo,
		groupRepo:          groupRepo,
		scopeRepo:          scopeRepo,
		ruleDependencyRepo: ruleDependencyRepo,
		ruleConflictRepo:   ruleConflictRepo,
		validator:          validator,
		nameMatcher:        nameMatcher,
	}
}

// ParseRule 解析语义化规则
func (s *RuleParserServiceImpl) ParseRule(ctx context.Context, req *service.ParseRuleRequest) (*service.ParseRuleResponse, error) {
	// 兼容处理：如果提供了 ruleText，使用 ruleText；否则使用 ruleDescription
	ruleText := req.RuleText
	if ruleText == "" && req.RuleDescription != "" {
		ruleText = req.RuleDescription
	}
	if ruleText == "" {
		return nil, fmt.Errorf("ruleText 或 ruleDescription 必须提供")
	}

	// 如果未提供 shiftNames/groupNames，自动获取
	shiftNames := req.ShiftNames
	groupNames := req.GroupNames
	if len(shiftNames) == 0 || len(groupNames) == 0 {
		shiftNamesAuto, groupNamesAuto := s.getShiftAndGroupNames(ctx, req.OrgID)
		if len(shiftNames) == 0 {
			shiftNames = shiftNamesAuto
		}
		if len(groupNames) == 0 {
			groupNames = groupNamesAuto
		}
	}

	// 1. 构建LLM提示词
	systemPrompt := s.buildParseSystemPrompt()
	userPrompt := s.buildParseUserPrompt(ruleText, shiftNames, groupNames, req)

	// 2. 调用LLM解析
	resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM解析失败: %w", err)
	}

	// 3. 解析LLM响应
	parseResult, err := s.parseLLMResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	// 4. 名称匹配（将 LLM 返回的名称转换为 ID）
	if err := s.matchNames(ctx, req.OrgID, parseResult.ParsedRules); err != nil {
		s.logger.Warn("名称匹配失败，部分关联可能未匹配", "error", err)
		// 名称匹配失败不阻断，只记录警告
	}

	// 5. 三层验证
	validationResult, err := s.validator.ValidateThreeLayers(ctx, req.OrgID, parseResult.ParsedRules)
	if err != nil {
		return nil, fmt.Errorf("验证失败: %w", err)
	}
	if len(validationResult.Layer1Errors) > 0 {
		return nil, fmt.Errorf("结构完整性验证失败: %v", validationResult.Layer1Errors)
	}

	// 6. 检查与现有规则的冲突
	conflicts, err := s.checkConflictsWithExisting(ctx, req.OrgID, parseResult.ParsedRules)
	if err != nil {
		return nil, fmt.Errorf("检查冲突失败: %w", err)
	}

	// 合并冲突，并过滤无效数据（如自己和自己冲突）
	allConflicts := make([]*service.RuleConflict, 0)
	for _, c := range parseResult.Conflicts {
		// 过滤掉自己和自己冲突的情况（标准化后比较，处理空格等差异）
		name1 := strings.TrimSpace(c.RuleName1)
		name2 := strings.TrimSpace(c.RuleName2)
		if name1 != name2 && name1 != "" && name2 != "" {
			allConflicts = append(allConflicts, c)
		}
		// 自己和自己冲突是LLM的常见错误输出，静默过滤即可
	}
	allConflicts = append(allConflicts, conflicts...)

	// 生成回译文本（如果 LLM 未返回，则生成简单回译）
	backTranslation := parseResult.BackTranslation
	if backTranslation == "" {
		backTranslation = s.generateBackTranslation(parseResult.ParsedRules)
	}

	// 为每个解析后的规则添加置信度
	for _, rule := range parseResult.ParsedRules {
		// 置信度计算（简单实现：基于验证结果）
		confidence := s.calculateConfidence(validationResult, []*service.ParsedRule{rule})
		// 注意：ParsedRule 结构体中没有 ParseConfidence 字段，需要在保存时设置到 SchedulingRule
		_ = confidence // 暂时未使用，后续可以在保存时使用
	}

	return &service.ParseRuleResponse{
		OriginalRule:    ruleText,
		ParsedRules:     parseResult.ParsedRules,
		Dependencies:    parseResult.Dependencies,
		Conflicts:       allConflicts,
		Reasoning:       parseResult.Reasoning,
		BackTranslation: backTranslation,
	}, nil
}

// BatchParse 批量解析规则
func (s *RuleParserServiceImpl) BatchParse(ctx context.Context, req *service.BatchParseRequest) (*service.BatchParseResponse, error) {
	results := make([]*service.ParseRuleResponse, 0, len(req.RuleTexts))
	errors := make([]*service.ParseError, 0)

	// 如果未提供 shiftNames/groupNames，自动获取
	shiftNames := req.ShiftNames
	groupNames := req.GroupNames
	if len(shiftNames) == 0 || len(groupNames) == 0 {
		shiftNamesAuto, groupNamesAuto := s.getShiftAndGroupNames(ctx, req.OrgID)
		if len(shiftNames) == 0 {
			shiftNames = shiftNamesAuto
		}
		if len(groupNames) == 0 {
			groupNames = groupNamesAuto
		}
	}

	// 逐个解析
	for _, ruleText := range req.RuleTexts {
		parseReq := &service.ParseRuleRequest{
			OrgID:      req.OrgID,
			RuleText:   ruleText,
			ShiftNames: shiftNames,
			GroupNames: groupNames,
		}
		result, err := s.ParseRule(ctx, parseReq)
		if err != nil {
			errors = append(errors, &service.ParseError{
				RuleText: ruleText,
				Error:    err.Error(),
			})
			continue
		}
		results = append(results, result)
	}

	return &service.BatchParseResponse{
		Results: results,
		Errors:  errors,
	}, nil
}

// SaveParsedRules 保存解析后的规则
func (s *RuleParserServiceImpl) SaveParsedRules(
	ctx context.Context,
	orgID string,
	parsedRules []*service.ParsedRule,
	dependencies []*service.RuleDependency,
	conflicts []*service.RuleConflict,
) ([]*model.SchedulingRule, error) {
	savedRules := make([]*model.SchedulingRule, 0, len(parsedRules))
	ruleNameToID := make(map[string]string) // 规则名称到ID的映射

	// 构建班次、员工、分组的名称到ID映射
	shiftNameToID, err := s.buildShiftNameToIDMap(ctx, orgID)
	if err != nil {
		s.logger.Warn("构建班次名称映射失败", "error", err)
		// 不阻断，继续处理
	}
	employeeNameToID, err := s.buildEmployeeNameToIDMap(ctx, orgID)
	if err != nil {
		s.logger.Warn("构建员工名称映射失败", "error", err)
	}
	groupNameToID, err := s.buildGroupNameToIDMap(ctx, orgID)
	if err != nil {
		s.logger.Warn("构建分组名称映射失败", "error", err)
	}

	// 1. 保存所有规则
	for _, parsed := range parsedRules {
		rule := &model.SchedulingRule{
			ID:             uuid.New().String(),
			OrgID:          orgID,
			Name:           parsed.Name,
			Description:    parsed.Description,
			RuleType:       parsed.RuleType,
			ApplyScope:     parsed.ApplyScope,
			TimeScope:      parsed.TimeScope,
			TimeOffsetDays: parsed.TimeOffsetDays,
			RuleData:       parsed.RuleData,
			MaxCount:       parsed.MaxCount,
			ConsecutiveMax: parsed.ConsecutiveMax,
			IntervalDays:   parsed.IntervalDays,
			MinRestDays:    parsed.MinRestDays,
			Priority:       parsed.Priority,
			ValidFrom:      parsed.ValidFrom,
			ValidTo:        parsed.ValidTo,
			IsActive:       true,
			Category:       parsed.Category,
			SubCategory:    parsed.SubCategory,
			SourceType:     "llm_parsed",
			Version:        "v4",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// V4.1: 构建班次关联（直接构建 Associations）
		shiftAssociations := s.buildShiftAssociations(parsed, shiftNameToID)
		if len(shiftAssociations) > 0 {
			rule.Associations = append(rule.Associations, shiftAssociations...)
		}

		// V4.1: 构建适用范围
		applyScopes := s.buildApplyScopes(parsed, employeeNameToID, groupNameToID)
		if len(applyScopes) > 0 {
			rule.ApplyScopes = applyScopes
		} else {
			// 默认全局
			rule.ApplyScopes = []model.RuleApplyScope{{ScopeType: model.ScopeTypeAll}}
		}

		// 保存规则
		if err := s.ruleRepo.Create(ctx, rule); err != nil {
			return nil, fmt.Errorf("保存规则失败: %w", err)
		}

		// 保存班次关联到关联表
		if len(shiftAssociations) > 0 {
			if err := s.ruleRepo.AddAssociations(ctx, orgID, rule.ID, shiftAssociations); err != nil {
				s.logger.Warn("保存班次关联失败", "ruleID", rule.ID, "error", err)
			}
		}

		// V4.1新增：保存适用范围（如果有员工/分组范围）
		if len(rule.ApplyScopes) > 0 {
			if err := s.scopeRepo.ReplaceScopes(ctx, orgID, rule.ID, rule.ApplyScopes); err != nil {
				s.logger.Warn("保存适用范围失败", "ruleID", rule.ID, "error", err)
			}
		}

		// 保存旧版关联（向后兼容）
		if len(parsed.Associations) > 0 {
			if err := s.ruleRepo.AddAssociations(ctx, orgID, rule.ID, parsed.Associations); err != nil {
				s.logger.Warn("保存规则关联失败", "ruleID", rule.ID, "error", err)
			}
		}

		ruleNameToID[parsed.Name] = rule.ID
		savedRules = append(savedRules, rule)
	}

	// 2. 保存依赖关系
	if len(dependencies) > 0 {
		ruleDeps := make([]*model.RuleDependency, 0, len(dependencies))
		for _, dep := range dependencies {
			// 将规则名称转换为ID
			dependentRuleID := ruleNameToID[dep.DependentRuleName]
			dependentOnRuleID := ruleNameToID[dep.DependentOnRuleName]
			if dependentRuleID != "" && dependentOnRuleID != "" {
				ruleDeps = append(ruleDeps, &model.RuleDependency{
					ID:                uuid.New().String(),
					OrgID:             orgID,
					DependentRuleID:   dependentRuleID,
					DependentOnRuleID: dependentOnRuleID,
					DependencyType:    dep.DependencyType,
					Description:       dep.Description,
					CreatedAt:         time.Now(),
				})
			}
		}
		if len(ruleDeps) > 0 {
			if s.ruleDependencyRepo != nil {
				if err := s.ruleDependencyRepo.BatchCreate(ctx, ruleDeps); err != nil {
					return nil, fmt.Errorf("保存规则依赖关系失败: %w", err)
				}
			}
		}
	}

	// 3. 保存冲突关系
	if len(conflicts) > 0 {
		ruleConfs := make([]*model.RuleConflict, 0, len(conflicts))
		for _, conf := range conflicts {
			ruleID1 := ruleNameToID[conf.RuleName1]
			ruleID2 := ruleNameToID[conf.RuleName2]
			if ruleID1 != "" && ruleID2 != "" {
				ruleConfs = append(ruleConfs, &model.RuleConflict{
					ID:                 uuid.New().String(),
					OrgID:              orgID,
					RuleID1:            ruleID1,
					RuleID2:            ruleID2,
					ConflictType:       conf.ConflictType,
					Description:        conf.Description,
					ResolutionPriority: 0, // 默认优先级
					CreatedAt:          time.Now(),
				})
			}
		}
		if len(ruleConfs) > 0 {
			if s.ruleConflictRepo != nil {
				if err := s.ruleConflictRepo.BatchCreate(ctx, ruleConfs); err != nil {
					return nil, fmt.Errorf("保存规则冲突关系失败: %w", err)
				}
			}
		}
	}

	return savedRules, nil
}

// buildShiftNameToIDMap 构建班次名称到ID的映射
func (s *RuleParserServiceImpl) buildShiftNameToIDMap(ctx context.Context, orgID string) (map[string]string, error) {
	result := make(map[string]string)
	shifts, err := s.shiftRepo.List(ctx, &model.ShiftFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}
	if shifts != nil {
		for _, shift := range shifts.Items {
			result[shift.Name] = shift.ID
		}
	}
	return result, nil
}

// buildEmployeeNameToIDMap 构建员工名称到ID的映射
func (s *RuleParserServiceImpl) buildEmployeeNameToIDMap(ctx context.Context, orgID string) (map[string]string, error) {
	result := make(map[string]string)
	employees, err := s.employeeRepo.List(ctx, &model.EmployeeFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, err
	}
	if employees != nil {
		for _, emp := range employees.Items {
			result[emp.Name] = emp.ID
		}
	}
	return result, nil
}

// buildGroupNameToIDMap 构建分组名称到ID的映射
func (s *RuleParserServiceImpl) buildGroupNameToIDMap(ctx context.Context, orgID string) (map[string]string, error) {
	result := make(map[string]string)
	groups, err := s.groupRepo.List(ctx, &model.GroupFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}
	if groups != nil {
		for _, group := range groups.Items {
			result[group.Name] = group.ID
		}
	}
	return result, nil
}

// buildShiftAssociations 根据解析结果构建班次关联（直接生成 RuleAssociation）
func (s *RuleParserServiceImpl) buildShiftAssociations(parsed *service.ParsedRule, shiftNameToID map[string]string) []model.RuleAssociation {
	var associations []model.RuleAssociation

	// 日志：显示 LLM 返回的班次字段
	s.logger.Info("构建班次关联",
		"ruleName", parsed.Name,
		"subjectShifts", parsed.SubjectShifts,
		"objectShifts", parsed.ObjectShifts,
		"targetShifts", parsed.TargetShifts,
		"shiftNameToIDCount", len(shiftNameToID),
	)

	// 处理主体班次
	for _, shiftName := range parsed.SubjectShifts {
		if shiftID, ok := shiftNameToID[shiftName]; ok {
			associations = append(associations, model.RuleAssociation{
				AssociationType: model.AssociationTypeShift,
				AssociationID:   shiftID,
				Role:            model.RelationRoleSubject,
			})
			s.logger.Info("添加主体班次关联", "shiftName", shiftName, "shiftID", shiftID)
		} else {
			s.logger.Warn("班次名称未找到", "shiftName", shiftName, "role", "subject")
		}
	}

	// 处理客体班次
	for _, shiftName := range parsed.ObjectShifts {
		if shiftID, ok := shiftNameToID[shiftName]; ok {
			associations = append(associations, model.RuleAssociation{
				AssociationType: model.AssociationTypeShift,
				AssociationID:   shiftID,
				Role:            model.RelationRoleObject,
			})
		} else {
			s.logger.Warn("班次名称未找到", "shiftName", shiftName, "role", "object")
		}
	}

	// 处理目标班次（单目标规则）
	for _, shiftName := range parsed.TargetShifts {
		if shiftID, ok := shiftNameToID[shiftName]; ok {
			associations = append(associations, model.RuleAssociation{
				AssociationType: model.AssociationTypeShift,
				AssociationID:   shiftID,
				Role:            model.RelationRoleTarget,
			})
		} else {
			s.logger.Warn("班次名称未找到", "shiftName", shiftName, "role", "target")
		}
	}

	return associations
}

// buildApplyScopes 根据解析结果构建适用范围
func (s *RuleParserServiceImpl) buildApplyScopes(parsed *service.ParsedRule, employeeNameToID, groupNameToID map[string]string) []model.RuleApplyScope {
	var scopes []model.RuleApplyScope

	scopeType := parsed.ScopeType
	if scopeType == "" {
		scopeType = model.ScopeTypeAll
	}

	switch scopeType {
	case model.ScopeTypeAll:
		scopes = append(scopes, model.RuleApplyScope{ScopeType: model.ScopeTypeAll})

	case model.ScopeTypeEmployee, model.ScopeTypeExcludeEmployee:
		for _, empName := range parsed.ScopeEmployees {
			if empID, ok := employeeNameToID[empName]; ok {
				scopes = append(scopes, model.RuleApplyScope{
					ScopeType: scopeType,
					ScopeID:   empID,
					ScopeName: empName,
				})
			} else {
				s.logger.Warn("员工名称未找到", "employeeName", empName)
			}
		}

	case model.ScopeTypeGroup, model.ScopeTypeExcludeGroup:
		for _, groupName := range parsed.ScopeGroups {
			if groupID, ok := groupNameToID[groupName]; ok {
				scopes = append(scopes, model.RuleApplyScope{
					ScopeType: scopeType,
					ScopeID:   groupID,
					ScopeName: groupName,
				})
			} else {
				s.logger.Warn("分组名称未找到", "groupName", groupName)
			}
		}
	}

	return scopes
}

// buildParseSystemPrompt 构建解析系统提示词
func (s *RuleParserServiceImpl) buildParseSystemPrompt() string {
	return `你是排班规则解析专家。你的任务是将用户输入的自然语言规则描述，解析并拆解为多条结构化的排班规则。

## 规则分类体系

### 1. 约束型规则（Constraint Rules）
- **禁止型**：exclusive（排他）、forbidden_day（禁止日期）
- **限制型**：maxCount（最大次数）、consecutiveMax（连续天数限制）、MinRestDays（最少休息天数）
- **必须型**：required_together（必须同时）、periodic（周期性要求）

### 2. 偏好型规则（Preference Rules）
- **优先型**：preferred（偏好）
- **可合并型**：combinable（可合并）

### 3. 依赖型规则（Dependency Rules）
- **来源依赖**：人员必须来自前一日某班次
- **资源预留**：当日班次人员需保留给次日
- **顺序依赖**：规则A必须在规则B之前执行

## V4.1 班次关系模型（重要）

每条规则都可以用"主体-动作-客体"的结构来描述：
- **主体班次（subjectShifts）**：规则约束的发起方/来源班次
- **客体班次（objectShifts）**：规则约束的作用对象/目标班次
- **目标班次（targetShifts）**：单目标规则（如maxCount）的目标班次

### 班次角色判断规则：
1. **exclusive（排他）**：A班和B班不能同时排
   - subjectShifts: [A班] - 作为约束发起方
   - objectShifts: [B班] - 作为被排斥的对象
   
2. **required_together（必须同时）**：排A班必须同时排B班
   - subjectShifts: [A班] - 触发条件
   - objectShifts: [B班] - 必须一起安排的班次

3. **maxCount（最大次数）**：某班次每周最多排X次
   - targetShifts: [该班次] - 被限制次数的班次

4. **periodic（周期性）**：某班次必须每隔N天安排一次
   - targetShifts: [该班次] - 周期性安排的班次

5. **combinable（可合并）**：A班和B班可以合并排
   - subjectShifts: [A班]
   - objectShifts: [B班]

## 适用范围说明

- **scopeType**：范围类型
  - all：全局适用
  - employee：仅适用指定员工
  - group：仅适用指定分组
  - exclude_employee：排除指定员工
  - exclude_group：排除指定分组
- **scopeEmployees**：员工名称列表（当scopeType为employee/exclude_employee时填写）
- **scopeGroups**：分组名称列表（当scopeType为group/exclude_group时填写）

## 解析要求

1. **识别规则类型**：根据语义判断规则类型（exclusive/maxCount/periodic/required_together/combinable等）。注意：如果在某天排了A班次，要求某天（如前一天）必须排B班次，这种类型属于"required_together"（必须同时规则），绝不可以使用未定义的资源依赖等类型！
2. **提取数值参数**：识别并提取数值型参数（如"最多3次"中的3）。
3. **识别跨日依赖偏移**：如果规则涉及到跨日（例如：“前一天”、“昨天”、“大前天”、“明天”），计算其客体班次发生日距离主体班次发生日的偏移天数，并填入 timeOffsetDays。例如：昨天是 -1，前天是 -2，明天是 1。如果没有跨日关系，则忽略该字段。（比如：“下夜班的人员必须是前一天安排了本部夜班的人”，这里下夜班是主体，本部夜班是客体，客体发生在主体的昨天，所以 timeOffsetDays = -1）。
4. **识别班次角色**：根据规则语义，正确划分主体班次、客体班次、目标班次。例如A要求昨天的B，则A是主体，B是客体。
5. **识别适用范围**：判断是全局规则还是针对特定员工/分组
6. **识别时间范围**：判断是same_day/same_week/same_month/custom
7. **识别依赖关系**：识别规则间的依赖关系
8. **识别冲突关系**：识别规则间的冲突关系

## 输出格式

请返回JSON格式：
{
  "parsedRules": [
    {
      "name": "规则名称",
      "category": "constraint/preference/dependency",
      "subCategory": "forbid/must/limit/prefer/suggest/source/resource/order",
      "ruleType": "exclusive/maxCount/periodic/required_together/...",
      "applyScope": "global/specific",
      "timeScope": "same_day/same_week/same_month/custom",
      "timeOffsetDays": -1,
      "description": "规则说明",
      "ruleData": "规则数据（语义化描述）",
      "maxCount": 3,
      "consecutiveMax": 2,
      "intervalDays": 7,
      "minRestDays": 1,
      "priority": 5,
      "subjectShifts": ["班次A名称"],
      "objectShifts": ["班次B名称"],
      "targetShifts": ["目标班次名称"],
      "scopeType": "all/employee/group/exclude_employee/exclude_group",
      "scopeEmployees": ["员工姓名"],
      "scopeGroups": ["分组名称"]
    }
  ],
  "dependencies": [
    {
      "dependentRuleName": "被依赖的规则名",
      "dependentOnRuleName": "依赖的规则名",
      "dependencyType": "time/source/resource/order",
      "description": "依赖关系描述"
    }
  ],
  "conflicts": [
    {
      "ruleName1": "规则1名称",
      "ruleName2": "规则2名称（必须与规则1不同）",
      "conflictType": "exclusive/resource/time/frequency",
      "description": "冲突描述"
    }
  ],
  "reasoning": "解析思路说明"
}

**重要说明：**
1. 请只返回上述 JSON 格式，不要添加任何额外的解释、说明或代码块标记。直接输出 JSON 对象。
2. 班次名称必须与用户提供的可用班次列表中的名称完全匹配。
3. 员工/分组名称必须与用户提供的列表中的名称完全匹配。
4. 根据规则类型选择正确的班次角色：二元关系规则使用subjectShifts+objectShifts，单目标规则使用targetShifts。
5. 如果规则是全局适用的，scopeType设为"all"，不需要填写scopeEmployees和scopeGroups。
6. conflicts 数组只用于描述**不同规则之间**的冲突关系。`
}

// buildParseUserPrompt 构建用户提示词
func (s *RuleParserServiceImpl) buildParseUserPrompt(ruleText string, shiftNames, groupNames []string, req *service.ParseRuleRequest) string {
	var b strings.Builder
	b.WriteString("请解析以下排班规则：\n\n")
	b.WriteString(fmt.Sprintf("规则描述：%s\n", ruleText))

	// 向后兼容：如果提供了 Name/ApplyScope/Priority，也加入提示词
	if req.Name != "" {
		b.WriteString(fmt.Sprintf("规则名称：%s\n", req.Name))
	}
	if req.ApplyScope != "" {
		b.WriteString(fmt.Sprintf("应用范围：%s\n", req.ApplyScope))
	}
	if req.Priority > 0 {
		b.WriteString(fmt.Sprintf("优先级：%d\n", req.Priority))
	}

	// 注入班次和分组名称列表（帮助 LLM 识别）
	if len(shiftNames) > 0 {
		b.WriteString(fmt.Sprintf("\n可用班次列表：%s\n", strings.Join(shiftNames, "、")))
	}
	if len(groupNames) > 0 {
		b.WriteString(fmt.Sprintf("可用分组列表：%s\n", strings.Join(groupNames, "、")))
	}

	return b.String()
}

// getShiftAndGroupNames 获取班次和分组名称列表
func (s *RuleParserServiceImpl) getShiftAndGroupNames(ctx context.Context, orgID string) ([]string, []string) {
	shiftNames := make([]string, 0)
	groupNames := make([]string, 0)

	// 获取班次列表
	shifts, err := s.shiftRepo.List(ctx, &model.ShiftFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err == nil && shifts != nil {
		for _, shift := range shifts.Items {
			shiftNames = append(shiftNames, shift.Name)
		}
	}

	// 获取分组列表
	groups, err := s.groupRepo.List(ctx, &model.GroupFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err == nil && groups != nil {
		for _, group := range groups.Items {
			groupNames = append(groupNames, group.Name)
		}
	}

	return shiftNames, groupNames
}

// matchNames 名称匹配（将 LLM 返回的名称转换为 ID）
func (s *RuleParserServiceImpl) matchNames(ctx context.Context, orgID string, parsedRules []*service.ParsedRule) error {
	for _, rule := range parsedRules {
		for i := range rule.Associations {
			assoc := &rule.Associations[i]
			// 如果 AssociationID 看起来像名称而不是 ID（简单判断：不包含连字符或长度较短）
			if len(assoc.AssociationID) < 32 && !strings.Contains(assoc.AssociationID, "-") {
				var matchedID string
				var err error

				switch assoc.AssociationType {
				case model.AssociationTypeEmployee:
					matchedID, err = s.nameMatcher.MatchEmployeeName(ctx, orgID, assoc.AssociationID)
				case model.AssociationTypeShift:
					matchedID, err = s.nameMatcher.MatchShiftName(ctx, orgID, assoc.AssociationID)
				case model.AssociationTypeGroup:
					matchedID, err = s.nameMatcher.MatchGroupName(ctx, orgID, assoc.AssociationID)
				}

				if err == nil {
					assoc.AssociationID = matchedID
					s.logger.Info("名称匹配成功", "type", assoc.AssociationType, "name", assoc.AssociationID, "id", matchedID)
				} else {
					s.logger.Warn("名称匹配失败", "type", assoc.AssociationType, "name", assoc.AssociationID, "error", err)
				}
			}
		}
	}
	return nil
}

// LLMResponse LLM响应结构
type LLMResponse struct {
	ParsedRules     []*service.ParsedRule     `json:"parsedRules"`
	Dependencies    []*service.RuleDependency `json:"dependencies"`
	Conflicts       []*service.RuleConflict   `json:"conflicts"`
	Reasoning       string                    `json:"reasoning"`
	BackTranslation string                    `json:"backTranslation,omitempty"` // 回译文本
}

// calculateConfidence 计算解析置信度
func (s *RuleParserServiceImpl) calculateConfidence(validationResult *ValidationResult, parsedRules []*service.ParsedRule) float64 {
	if len(parsedRules) == 0 {
		return 0.0
	}

	// 基础置信度
	confidence := 0.8

	// 如果有验证错误，降低置信度
	if len(validationResult.Layer1Errors) > 0 {
		confidence -= 0.3
	}
	if len(validationResult.Layer2Errors) > 0 {
		confidence -= 0.1
	}
	if len(validationResult.Layer3Errors) > 0 {
		confidence -= 0.1
	}

	// 确保在 0.0-1.0 范围内
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateBackTranslation 生成回译文本（如果 LLM 未返回）
func (s *RuleParserServiceImpl) generateBackTranslation(parsedRules []*service.ParsedRule) string {
	if len(parsedRules) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("解析后的规则：\n")
	for i, rule := range parsedRules {
		b.WriteString(fmt.Sprintf("%d. %s (%s/%s): %s\n", i+1, rule.Name, rule.Category, rule.SubCategory, rule.Description))
	}
	return b.String()
}

// parseLLMResponse 解析LLM响应
func (s *RuleParserServiceImpl) parseLLMResponse(content string) (*LLMResponse, error) {
	originalContent := content
	content = strings.TrimSpace(content)

	// 尝试多种方式提取 JSON

	// 方式1: 处理 markdown 代码块 (```json ... ``` 或 ``` ... ```)
	if strings.Contains(content, "```") {
		// 提取代码块中的JSON
		lines := strings.Split(content, "\n")
		var jsonLines []string
		inCodeBlock := false
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}
			if inCodeBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		if len(jsonLines) > 0 {
			content = strings.Join(jsonLines, "\n")
		}
	}

	// 方式2: 尝试找到 JSON 对象的开始和结束
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") {
		// 尝试找到第一个 { 和最后一个 }
		startIdx := strings.Index(content, "{")
		endIdx := strings.LastIndex(content, "}")
		if startIdx >= 0 && endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	// 方式3: 清理可能的尾部噪音（有些 LLM 会在 JSON 后添加解释文字）
	content = strings.TrimSpace(content)

	var result LLMResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 记录详细日志以便调试
		logContent := content
		if len(logContent) > 500 {
			logContent = logContent[:500]
		}
		logOriginal := originalContent
		if len(logOriginal) > 500 {
			logOriginal = logOriginal[:500]
		}
		s.logger.Error("JSON解析失败",
			"error", err,
			"extractedContent", logContent,
			"originalContent", logOriginal,
		)
		errContent := content
		if len(errContent) > 200 {
			errContent = errContent[:200]
		}
		return nil, fmt.Errorf("JSON解析失败: %w, content: %s", err, errContent)
	}

	// 验证解析结果不为空
	if len(result.ParsedRules) == 0 {
		logContent := content
		if len(logContent) > 500 {
			logContent = logContent[:500]
		}
		s.logger.Warn("LLM返回空规则列表", "content", logContent)
	}

	return &result, nil
}

// checkConflictsWithExisting 检查与现有规则的冲突
func (s *RuleParserServiceImpl) checkConflictsWithExisting(
	ctx context.Context,
	orgID string,
	parsedRules []*service.ParsedRule,
) ([]*service.RuleConflict, error) {
	conflicts := make([]*service.RuleConflict, 0)

	// 1. 加载现有规则
	filter := &model.SchedulingRuleFilter{
		OrgID:    orgID,
		IsActive: boolPtr(true),
		Page:     1,
		PageSize: 10000,
	}
	existingRules, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取现有规则失败: %w", err)
	}

	// 2. 比较规则名称（重复名称冲突）
	existingNames := make(map[string]bool)
	for _, existing := range existingRules.Items {
		existingNames[existing.Name] = true
	}
	for _, parsed := range parsedRules {
		if existingNames[parsed.Name] {
			conflicts = append(conflicts, &service.RuleConflict{
				RuleName1:    parsed.Name,
				RuleName2:    "(已存在的同名规则)",
				ConflictType: "duplicate",
				Description:  fmt.Sprintf("规则名称 \"%s\" 已存在，保存时将覆盖或需要修改名称", parsed.Name),
			})
		}
	}

	// 3. 比较规则类型和参数（逻辑冲突）
	for _, parsed := range parsedRules {
		for _, existing := range existingRules.Items {
			// 检查互斥规则
			if s.isExclusiveConflict(parsed, existing) {
				conflicts = append(conflicts, &service.RuleConflict{
					RuleName1:    parsed.Name,
					RuleName2:    existing.Name,
					ConflictType: "exclusive",
					Description:  fmt.Sprintf("规则 %s 与 %s 互斥", parsed.Name, existing.Name),
				})
			}

			// 检查资源冲突
			if s.isResourceConflict(parsed, existing) {
				conflicts = append(conflicts, &service.RuleConflict{
					RuleName1:    parsed.Name,
					RuleName2:    existing.Name,
					ConflictType: "resource",
					Description:  fmt.Sprintf("规则 %s 与 %s 存在资源冲突", parsed.Name, existing.Name),
				})
			}
		}
	}

	return conflicts, nil
}

// isExclusiveConflict 检查是否为互斥冲突
func (s *RuleParserServiceImpl) isExclusiveConflict(parsed *service.ParsedRule, existing *model.SchedulingRule) bool {
	// 如果两个规则都是 exclusive 类型，且关联的对象有重叠，则互斥
	if parsed.RuleType == model.RuleTypeExclusive && existing.RuleType == model.RuleTypeExclusive {
		return s.hasOverlappingAssociations(parsed.Associations, existing.Associations)
	}
	return false
}

// isResourceConflict 检查是否为资源冲突
func (s *RuleParserServiceImpl) isResourceConflict(parsed *service.ParsedRule, existing *model.SchedulingRule) bool {
	// 如果两个规则都限制了同一资源（如最大次数），且参数冲突
	if parsed.RuleType == model.RuleTypeMaxCount && existing.RuleType == model.RuleTypeMaxCount {
		if parsed.MaxCount != nil && existing.MaxCount != nil {
			// 如果两个规则都限制同一对象，且限制值冲突
			if s.hasOverlappingAssociations(parsed.Associations, existing.Associations) {
				// 例如：一个规则说最多3次，另一个说最多5次，这是冲突的
				return *parsed.MaxCount != *existing.MaxCount
			}
		}
	}
	return false
}

// hasOverlappingAssociations 检查关联对象是否有重叠
func (s *RuleParserServiceImpl) hasOverlappingAssociations(assocs1 []model.RuleAssociation, assocs2 []model.RuleAssociation) bool {
	assocSet1 := make(map[string]bool)
	for _, assoc := range assocs1 {
		key := string(assoc.AssociationType) + ":" + assoc.AssociationID
		assocSet1[key] = true
	}

	for _, assoc := range assocs2 {
		key := string(assoc.AssociationType) + ":" + assoc.AssociationID
		if assocSet1[key] {
			return true
		}
	}
	return false
}
