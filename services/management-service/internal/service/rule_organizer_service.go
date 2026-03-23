package service

import (
	"context"
	"fmt"
	"sort"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// RuleOrganizerService 规则组织服务
type RuleOrganizerService struct {
	logger              logging.ILogger
	ruleRepo            repository.ISchedulingRuleRepository
	ruleDependencyRepo  repository.IRuleDependencyRepository
	ruleConflictRepo    repository.IRuleConflictRepository
	shiftDependencyRepo repository.IShiftDependencyRepository
	shiftRepo           repository.IShiftRepository
}

// NewRuleOrganizerService 创建规则组织服务
func NewRuleOrganizerService(
	logger logging.ILogger,
	ruleRepo repository.ISchedulingRuleRepository,
	ruleDependencyRepo repository.IRuleDependencyRepository,
	ruleConflictRepo repository.IRuleConflictRepository,
	shiftDependencyRepo repository.IShiftDependencyRepository,
	shiftRepo repository.IShiftRepository,
) domain_service.IRuleOrganizerService {
	return &RuleOrganizerService{
		logger:              logger,
		ruleRepo:            ruleRepo,
		ruleDependencyRepo:  ruleDependencyRepo,
		ruleConflictRepo:    ruleConflictRepo,
		shiftDependencyRepo: shiftDependencyRepo,
		shiftRepo:           shiftRepo,
	}
}

// OrganizeRules 组织规则
func (s *RuleOrganizerService) OrganizeRules(
	ctx context.Context,
	orgID string,
) (*domain_service.RuleOrganizationResult, error) {
	// 1. 加载所有启用的规则
	rules, err := s.ruleRepo.GetActiveRules(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("加载规则失败: %w", err)
	}

	// 2. 加载所有启用的班次
	shifts, err := s.shiftRepo.GetActiveShifts(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("加载班次失败: %w", err)
	}

	// 3. 加载依赖关系和冲突关系
	ruleDependencies, err := s.ruleDependencyRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("加载规则依赖关系失败: %w", err)
	}

	ruleConflicts, err := s.ruleConflictRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("加载规则冲突关系失败: %w", err)
	}

	shiftDependencies, err := s.shiftDependencyRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("加载班次依赖关系失败: %w", err)
	}

	// 4. 分类规则
	result := &domain_service.RuleOrganizationResult{
		ConstraintRules:     make([]*domain_service.ClassifiedRuleInfo, 0),
		PreferenceRules:     make([]*domain_service.ClassifiedRuleInfo, 0),
		DependencyRules:     make([]*domain_service.ClassifiedRuleInfo, 0),
		ShiftDependencies:   convertShiftDependencies(shiftDependencies),
		RuleDependencies:    convertRuleDependencies(ruleDependencies),
		RuleConflicts:       convertRuleConflicts(ruleConflicts),
		ShiftExecutionOrder: make([]string, 0),
		RuleExecutionOrder:  make([]string, 0),
	}

	// 构建规则ID到依赖关系的映射
	ruleDepsMap := make(map[string][]string)
	for _, dep := range ruleDependencies {
		ruleDepsMap[dep.DependentOnRuleID] = append(ruleDepsMap[dep.DependentOnRuleID], dep.DependentRuleID)
	}

	ruleConflictsMap := make(map[string][]string)
	for _, conf := range ruleConflicts {
		ruleConflictsMap[conf.RuleID1] = append(ruleConflictsMap[conf.RuleID1], conf.RuleID2)
		ruleConflictsMap[conf.RuleID2] = append(ruleConflictsMap[conf.RuleID2], conf.RuleID1)
	}

	// 分类规则
	for _, rule := range rules {
		category := rule.Category
		if category == "" {
			category = inferCategory(rule)
		}
		subCategory := rule.SubCategory
		if subCategory == "" {
			subCategory = inferSubCategory(rule)
		}

		classified := &domain_service.ClassifiedRuleInfo{
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			Category:     category,
			SubCategory:  subCategory,
			RuleType:     string(rule.RuleType),
			Dependencies: ruleDepsMap[rule.ID],
			Conflicts:    ruleConflictsMap[rule.ID],
			Priority:     rule.Priority,
			Description:  rule.Description,
		}

		switch category {
		case "constraint":
			result.ConstraintRules = append(result.ConstraintRules, classified)
		case "preference":
			result.PreferenceRules = append(result.PreferenceRules, classified)
		case "dependency":
			result.DependencyRules = append(result.DependencyRules, classified)
		}
	}

	// 5. 拓扑排序计算执行顺序
	ruleOrder, err := topologicalSortRules(rules, ruleDependencies)
	if err != nil {
		return nil, fmt.Errorf("规则拓扑排序失败: %w", err)
	}
	result.RuleExecutionOrder = ruleOrder

	shiftOrder, warnings, err := topologicalSortShiftsWithWarnings(shifts, shiftDependencies)
	if err != nil {
		return nil, fmt.Errorf("班次拓扑排序失败: %w", err)
	}
	result.ShiftExecutionOrder = shiftOrder
	result.Warnings = warnings

	return result, nil
}

func inferCategory(rule *model.SchedulingRule) string {
	switch rule.RuleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay, model.RuleTypeMaxCount,
		"consecutiveMax", "minRestDays", model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return "constraint"
	case model.RuleTypePreferred, model.RuleTypeCombinable:
		return "preference"
	default:
		return "constraint"
	}
}

func inferSubCategory(rule *model.SchedulingRule) string {
	switch rule.RuleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay:
		return "forbid"
	case model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return "must"
	case model.RuleTypeMaxCount, "consecutiveMax", "minRestDays":
		return "limit"
	case model.RuleTypePreferred:
		return "prefer"
	case model.RuleTypeCombinable:
		return "suggest"
	default:
		return "limit"
	}
}

func convertShiftDependencies(deps []*model.ShiftDependency) []*domain_service.ShiftDependencyInfo {
	result := make([]*domain_service.ShiftDependencyInfo, len(deps))
	for i, dep := range deps {
		result[i] = &domain_service.ShiftDependencyInfo{
			DependentShiftID:   dep.DependentShiftID,
			DependentOnShiftID: dep.DependentOnShiftID,
			DependencyType:     dep.DependencyType,
			RuleID:             dep.RuleID,
			Description:        dep.Description,
		}
	}
	return result
}

func convertRuleDependencies(deps []*model.RuleDependency) []*domain_service.RuleDependencyInfo {
	result := make([]*domain_service.RuleDependencyInfo, len(deps))
	for i, dep := range deps {
		result[i] = &domain_service.RuleDependencyInfo{
			DependentRuleID:   dep.DependentRuleID,
			DependentOnRuleID: dep.DependentOnRuleID,
			DependencyType:    dep.DependencyType,
			Description:       dep.Description,
		}
	}
	return result
}

func convertRuleConflicts(confs []*model.RuleConflict) []*domain_service.RuleConflictInfo {
	result := make([]*domain_service.RuleConflictInfo, len(confs))
	for i, conf := range confs {
		result[i] = &domain_service.RuleConflictInfo{
			RuleID1:            conf.RuleID1,
			RuleID2:            conf.RuleID2,
			ConflictType:       conf.ConflictType,
			Description:        conf.Description,
			ResolutionPriority: conf.ResolutionPriority,
		}
	}
	return result
}

// topologicalSortRules 拓扑排序规则
func topologicalSortRules(
	rules []*model.SchedulingRule,
	dependencies []*model.RuleDependency,
) ([]string, error) {
	// 构建入度图
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// 初始化所有节点
	for _, rule := range rules {
		inDegree[rule.ID] = 0
		graph[rule.ID] = make([]string, 0)
	}

	// 构建依赖图
	for _, dep := range dependencies {
		// dependentOnRuleID 依赖 dependentRuleID
		// 所以 dependentRuleID -> dependentOnRuleID
		graph[dep.DependentRuleID] = append(graph[dep.DependentRuleID], dep.DependentOnRuleID)
		inDegree[dep.DependentOnRuleID]++
	}

	// 找到所有入度为0的节点
	queue := make([]string, 0)
	for ruleID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, ruleID)
		}
	}

	// 拓扑排序
	result := make([]string, 0)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 检查是否有环
	if len(result) != len(rules) {
		return nil, fmt.Errorf("存在循环依赖")
	}

	return result, nil
}

// topologicalSortShiftsWithWarnings 拓扑排序班次（带冲突警告检测）
func topologicalSortShiftsWithWarnings(
	shifts []*model.Shift,
	dependencies []*model.ShiftDependency,
) ([]string, []*domain_service.OrganizationWarning, error) {
	warnings := make([]*domain_service.OrganizationWarning, 0)

	// 构建入度图
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// 初始化所有节点
	shiftIDMap := make(map[string]bool)
	shiftPriorityMap := make(map[string]int) // shiftID -> SchedulingPriority
	shiftNameMap := make(map[string]string)  // shiftID -> shiftName
	for _, shift := range shifts {
		shiftID := shift.ID
		shiftIDMap[shiftID] = true
		shiftPriorityMap[shiftID] = shift.SchedulingPriority
		shiftNameMap[shiftID] = shift.Name
		inDegree[shiftID] = 0
		graph[shiftID] = make([]string, 0)
	}

	// 构建依赖图，并检测优先级冲突
	for _, dep := range dependencies {
		// 只处理存在的班次
		if !shiftIDMap[dep.DependentShiftID] || !shiftIDMap[dep.DependentOnShiftID] {
			continue
		}

		// 检测优先级与依赖关系冲突
		// 依赖关系: DependentOnShiftID 依赖 DependentShiftID，即 DependentShiftID 应该先执行
		// 如果 DependentShiftID 的优先级数字 > DependentOnShiftID 的优先级数字，则存在冲突
		priorityDependent := shiftPriorityMap[dep.DependentShiftID]     // 被依赖的班次（应该先执行）
		priorityDependentOn := shiftPriorityMap[dep.DependentOnShiftID] // 依赖方的班次（应该后执行）

		if priorityDependent > priorityDependentOn {
			// 冲突：优先级说 DependentOnShiftID 应该先执行，但依赖关系说 DependentShiftID 应该先执行
			warnings = append(warnings, &domain_service.OrganizationWarning{
				Type: "priority_dependency_conflict",
				Message: fmt.Sprintf("班次优先级与依赖关系冲突：「%s」(优先级%d) 被「%s」(优先级%d) 依赖，应先执行，但优先级设置相反。将按依赖关系执行。",
					shiftNameMap[dep.DependentShiftID], priorityDependent,
					shiftNameMap[dep.DependentOnShiftID], priorityDependentOn),
				ShiftID1:   dep.DependentShiftID,
				ShiftName1: shiftNameMap[dep.DependentShiftID],
				Priority1:  priorityDependent,
				ShiftID2:   dep.DependentOnShiftID,
				ShiftName2: shiftNameMap[dep.DependentOnShiftID],
				Priority2:  priorityDependentOn,
				Resolution: "use_dependency",
			})
		}

		// dependentOnShiftID 依赖 dependentShiftID
		// 所以 dependentShiftID -> dependentOnShiftID
		graph[dep.DependentShiftID] = append(graph[dep.DependentShiftID], dep.DependentOnShiftID)
		inDegree[dep.DependentOnShiftID]++
	}

	// 定义排序比较函数：优先按 SchedulingPriority 排序，相同时按名称排序
	compareShifts := func(id1, id2 string) bool {
		p1, p2 := shiftPriorityMap[id1], shiftPriorityMap[id2]
		if p1 != p2 {
			return p1 < p2 // 优先级数字小的先执行
		}
		return shiftNameMap[id1] < shiftNameMap[id2] // 相同优先级按名称排序
	}

	// 找到所有入度为0的节点（按 SchedulingPriority 排序）
	queue := make([]string, 0)
	for shiftID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, shiftID)
		}
	}
	// 按排班优先级排序
	sort.Slice(queue, func(i, j int) bool {
		return compareShifts(queue[i], queue[j])
	})

	// 拓扑排序
	result := make([]string, 0)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// 收集所有新的入度为0的节点
		newNodes := make([]string, 0)
		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				newNodes = append(newNodes, neighbor)
			}
		}
		// 按排班优先级排序后加入队列
		sort.Slice(newNodes, func(i, j int) bool {
			return compareShifts(newNodes[i], newNodes[j])
		})
		queue = append(queue, newNodes...)
	}

	// 检查是否有环
	if len(result) != len(shifts) {
		return nil, warnings, fmt.Errorf("存在循环依赖")
	}

	return result, warnings, nil
}
