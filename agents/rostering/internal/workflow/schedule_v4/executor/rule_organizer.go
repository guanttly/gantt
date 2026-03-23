package executor

import (
	"context"
	"fmt"
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
	"sort"
)

// RuleOrganization 规则组织结果
type RuleOrganization struct {
	// 分类后的规则
	ConstraintRules []*ClassifiedRule // 约束型规则
	PreferenceRules []*ClassifiedRule // 偏好型规则
	DependencyRules []*ClassifiedRule // 依赖型规则

	// 依赖关系
	ShiftDependencies []ShiftDependency // 班次依赖关系
	RuleDependencies  []RuleDependency  // 规则依赖关系

	// 冲突关系
	RuleConflicts []RuleConflict // 规则冲突关系

	// 执行顺序
	ShiftExecutionOrder []string // 班次执行顺序（按依赖关系排序）
	RuleExecutionOrder  []string // 规则执行顺序（按依赖关系排序）

	// 强约束班次组 (v4.2新增)
	ShiftGroups []ShiftGroup // 需要由特定 Scheduler 处理的班次组

	// PureObjectShiftIDs 纯客体班次集合
	// 无论关联的是 required_together 还是纯时间先后/数量依赖，只要这些班次仅作为客体（源）而从不作为主体触发规则，就不该在排他性以外被意外填满
	PureObjectShiftIDs map[string]bool

	// 警告信息
	Warnings []*OrganizationWarning // 组织过程中的警告（如优先级与依赖冲突）
}

// ShiftGroup 强关联/互斥的班次组，需由专门的 Scheduler 统一调度
type ShiftGroup struct {
	ShiftIDs []string
	RuleType string // e.g., RuleTypeRequiredTogether, RuleTypeExclusive
	RuleID   string // 关联的规则ID
}

// OrganizationWarning 组织警告信息
type OrganizationWarning struct {
	Type       string // 警告类型: priority_dependency_conflict
	Message    string // 警告消息
	ShiftID1   string
	ShiftName1 string
	ShiftID2   string
	ShiftName2 string
	Priority1  int
	Priority2  int
	Resolution string // 解决方式: use_dependency / use_priority
}

// ClassifiedRule 分类后的规则
type ClassifiedRule struct {
	Rule           *model.Rule
	Category       string   // constraint/preference/dependency
	SubCategory    string   // forbid/must/limit/prefer/suggest/source/resource/order
	Dependencies   []string // 依赖的其他规则ID
	Conflicts      []string // 冲突的其他规则ID
	ExecutionOrder int      // 执行顺序（数字越小越先执行）
}

// ShiftDependency 班次依赖关系
type ShiftDependency struct {
	DependentShiftID   string `json:"dependentShiftId"`   // 被依赖的班次（需要先排）
	DependentOnShiftID string `json:"dependentOnShiftId"` // 依赖的班次（后排）
	DependencyType     string `json:"dependencyType"`     // source/resource/time
	RuleID             string `json:"ruleId"`
	Description        string `json:"description"`
	TimeOffsetDays     int    `json:"timeOffsetDays"` // 客体（DependentShiftID）相较于主体（DependentOnShiftID）的时间偏移天数
}

// RuleDependency 规则依赖关系
type RuleDependency struct {
	DependentRuleID   string // 被依赖的规则（需要先执行）
	DependentOnRuleID string // 依赖的规则（后执行）
	DependencyType    string // time/source/resource/order
	Description       string // 依赖关系描述
}

// RuleConflict 规则冲突关系
type RuleConflict struct {
	RuleID1            string // 冲突的规则1
	RuleID2            string // 冲突的规则2
	ConflictType       string // exclusive/resource/time/frequency
	Description        string // 冲突描述
	ResolutionPriority int    // 解决优先级
}

// RuleOrganizer 规则组织器
type RuleOrganizer struct {
	logger   logging.ILogger
	ruleRepo RuleRepository // 规则仓储接口（可选，用于从数据库加载依赖关系）
}

// RuleRepository 规则仓储接口（可选，用于从数据库加载依赖关系）
type RuleRepository interface {
	GetRuleDependencies(ctx context.Context, orgID string) ([]*RuleDependency, error)
	GetRuleConflicts(ctx context.Context, orgID string) ([]*RuleConflict, error)
	GetShiftDependencies(ctx context.Context, orgID string) ([]*ShiftDependency, error)
}

// NewRuleOrganizer 创建规则组织器
func NewRuleOrganizer(logger logging.ILogger, ruleRepo RuleRepository) *RuleOrganizer {
	return &RuleOrganizer{
		logger:   logger,
		ruleRepo: ruleRepo,
	}
}

// OrganizeRules 组织规则
// dependencies, conflicts, shiftDeps 如果为 nil，则尝试从 ruleRepo 加载
func (o *RuleOrganizer) OrganizeRules(
	ctx context.Context,
	orgID string,
	rules []*model.Rule,
	shifts []*model.Shift,
	dependencies []*RuleDependency, // 可选：规则依赖关系
	conflicts []*RuleConflict, // 可选：规则冲突关系
	shiftDeps []*ShiftDependency, // 可选：班次依赖关系
) (*RuleOrganization, error) {
	// 1. 加载规则分类、依赖关系、冲突关系
	// 如果参数为 nil，尝试从仓储加载
	var err error
	if dependencies == nil && o.ruleRepo != nil {
		dependencies, err = o.ruleRepo.GetRuleDependencies(ctx, orgID)
		if err != nil {
			o.logger.Warn("从仓储加载规则依赖关系失败，使用空列表", "error", err)
			dependencies = make([]*RuleDependency, 0)
		}
	}
	if len(dependencies) == 0 {
		analyzer := NewDependencyAnalyzer(o.logger)
		if extractedDeps, err := analyzer.AnalyzeRuleDependencies(ctx, rules); err == nil {
			dependencies = extractedDeps
		} else {
			dependencies = make([]*RuleDependency, 0)
		}
	}

	if conflicts == nil && o.ruleRepo != nil {
		conflicts, err = o.ruleRepo.GetRuleConflicts(ctx, orgID)
		if err != nil {
			o.logger.Warn("从仓储加载规则冲突关系失败，使用空列表", "error", err)
			conflicts = make([]*RuleConflict, 0)
		}
	}
	if conflicts == nil {
		conflicts = make([]*RuleConflict, 0)
	}

	if shiftDeps == nil && o.ruleRepo != nil {
		shiftDeps, err = o.ruleRepo.GetShiftDependencies(ctx, orgID)
		if err != nil {
			o.logger.Warn("从仓储加载班次依赖关系失败，使用空列表", "error", err)
			shiftDeps = make([]*ShiftDependency, 0)
		}
	}
	if len(shiftDeps) == 0 {
		analyzer := NewDependencyAnalyzer(o.logger)
		if extractedDeps, err := analyzer.AnalyzeShiftDependencies(ctx, rules, shifts); err == nil {
			shiftDeps = extractedDeps
		} else {
			shiftDeps = make([]*ShiftDependency, 0)
		}
	}

	// 2. 构建规则分类映射
	// 转换指针切片为值切片
	shiftDepsValues := make([]ShiftDependency, len(shiftDeps))
	for i, dep := range shiftDeps {
		shiftDepsValues[i] = *dep
	}
	depsValues := make([]RuleDependency, len(dependencies))
	for i, dep := range dependencies {
		depsValues[i] = *dep
	}
	conflictsValues := make([]RuleConflict, len(conflicts))
	for i, conf := range conflicts {
		conflictsValues[i] = *conf
	}

	org := &RuleOrganization{
		ConstraintRules:   make([]*ClassifiedRule, 0),
		PreferenceRules:   make([]*ClassifiedRule, 0),
		DependencyRules:   make([]*ClassifiedRule, 0),
		ShiftDependencies: shiftDepsValues,
		RuleDependencies:  depsValues,
		RuleConflicts:     conflictsValues,
		ShiftGroups:       make([]ShiftGroup, 0),
		Warnings:          make([]*OrganizationWarning, 0),
	}

	// 3. 分类规则
	ruleIDToClassified := make(map[string]*ClassifiedRule)
	for _, rule := range rules {
		classified := o.classifyRule(rule, dependencies, conflicts)
		ruleIDToClassified[rule.ID] = classified

		switch classified.Category {
		case "constraint":
			org.ConstraintRules = append(org.ConstraintRules, classified)
		case "preference":
			org.PreferenceRules = append(org.PreferenceRules, classified)
		case "dependency":
			org.DependencyRules = append(org.DependencyRules, classified)
		}
	}

	// 4. 拓扑排序计算执行顺序
	ruleOrder, err := o.topologicalSortRules(ruleIDToClassified, dependencies)
	if err != nil {
		return nil, fmt.Errorf("规则拓扑排序失败: %w", err)
	}
	org.RuleExecutionOrder = ruleOrder

	shiftOrder, shiftWarnings, shiftGroups, err := o.topologicalSortShifts(shifts, shiftDeps)
	if err != nil {
		return nil, fmt.Errorf("班次拓扑排序失败: %w", err)
	}
	org.ShiftExecutionOrder = shiftOrder
	org.ShiftGroups = shiftGroups
	org.Warnings = append(org.Warnings, shiftWarnings...)
	org.PureObjectShiftIDs = o.buildPureObjectShiftIDs(org.ShiftDependencies)

	return org, nil
}

// buildPureObjectShiftIDs 纯客体班次（比如被依赖的前置记录）。这些班次不单独兜底填满
// 注意：required_together 的客体班次（如 CT/MRI报告上）是有独立排班需求的大班次，不能标记为纯客体。
// 仅 source/time/resource/order 等类型的被依赖班次才算"纯客体"。
func (o *RuleOrganizer) buildPureObjectShiftIDs(deps []ShiftDependency) map[string]bool {
	objectCandidates := make(map[string]bool)
	subjectSet := make(map[string]bool)
	for _, dep := range deps {
		// required_together 的客体班次有独立排班需求，不视为纯客体
		if dep.DependencyType == RuleTypeRequiredTogether {
			continue
		}
		// 其他依赖类型（source/time/resource/order），其客体是纯粹提供关联约束的
		objectCandidates[dep.DependentShiftID] = true
		subjectSet[dep.DependentOnShiftID] = true
	}
	out := make(map[string]bool)
	for shiftID := range objectCandidates {
		if !subjectSet[shiftID] {
			out[shiftID] = true
		}
	}
	return out
}

// classifyRule 分类规则
func (o *RuleOrganizer) classifyRule(
	rule *model.Rule,
	dependencies []*RuleDependency,
	conflicts []*RuleConflict,
) *ClassifiedRule {
	classified := &ClassifiedRule{
		Rule:         rule,
		Category:     "", // 从规则元数据获取，如果没有则推断
		SubCategory:  "", // 从规则元数据获取，如果没有则推断
		Dependencies: make([]string, 0),
		Conflicts:    make([]string, 0),
	}

	// 尝试从规则元数据获取分类（如果SDK model有这些字段）
	// 否则根据规则类型推断
	classified.Category = o.inferCategory(rule)
	classified.SubCategory = o.inferSubCategory(rule)

	// 收集依赖关系
	for _, dep := range dependencies {
		if dep.DependentOnRuleID == rule.ID {
			classified.Dependencies = append(classified.Dependencies, dep.DependentRuleID)
		}
	}

	// 收集冲突关系
	for _, conflict := range conflicts {
		if conflict.RuleID1 == rule.ID {
			classified.Conflicts = append(classified.Conflicts, conflict.RuleID2)
		} else if conflict.RuleID2 == rule.ID {
			classified.Conflicts = append(classified.Conflicts, conflict.RuleID1)
		}
	}

	return classified
}

// inferCategory 推断规则分类
func (o *RuleOrganizer) inferCategory(rule *model.Rule) string {
	switch rule.RuleType {
	case RuleTypeExclusive, "forbidden_day", "maxCount", "consecutiveMax", "minRestDays", RuleTypeRequiredTogether, "periodic":
		return "constraint"
	case "preferred", "combinable":
		return "preference"
	default:
		return "constraint"
	}
}

// inferSubCategory 推断规则子分类
func (o *RuleOrganizer) inferSubCategory(rule *model.Rule) string {
	switch rule.RuleType {
	case RuleTypeExclusive, "forbidden_day":
		return "forbid"
	case RuleTypeRequiredTogether, "periodic":
		return "must"
	case "maxCount", "consecutiveMax", "minRestDays":
		return "limit"
	case "preferred":
		return "prefer"
	case "combinable":
		return "suggest"
	default:
		return "limit"
	}
}

// topologicalSortRules 拓扑排序规则
func (o *RuleOrganizer) topologicalSortRules(
	rules map[string]*ClassifiedRule,
	dependencies []*RuleDependency,
) ([]string, error) {
	// 构建入度图
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// 初始化所有节点
	for ruleID := range rules {
		inDegree[ruleID] = 0
		graph[ruleID] = make([]string, 0)
	}

	// 构建依赖图
	for _, dep := range dependencies {
		// dependentRuleID 依赖 dependentOnRuleID
		// 所以 dependentOnRuleID -> dependentRuleID
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

// sortShiftsByPriority 按班次优先级排序（提取 ShiftGroups，生成冲突警告）
// 执行顺序完全由 SchedulingPriority 决定，不受依赖关系影响，
// 以避免客体班次被提前排班导致大量无法补齐的情况。
func (o *RuleOrganizer) topologicalSortShifts(
	shifts []*model.Shift,
	dependencies []*ShiftDependency,
) ([]string, []*OrganizationWarning, []ShiftGroup, error) {
	warnings := make([]*OrganizationWarning, 0)
	shiftGroups := make([]ShiftGroup, 0)

	// 初始化班次映射
	shiftIDMap := make(map[string]bool)
	shiftPriorityMap := make(map[string]int)
	shiftNameMap := make(map[string]string)
	for _, shift := range shifts {
		shiftIDMap[shift.ID] = true
		shiftPriorityMap[shift.ID] = shift.SchedulingPriority
		shiftNameMap[shift.ID] = shift.Name
	}

	// 遍历依赖关系：检测优先级冲突（仅警告）+ 收集强约束分组
	for _, dep := range dependencies {
		if !shiftIDMap[dep.DependentShiftID] || !shiftIDMap[dep.DependentOnShiftID] {
			continue
		}

		// 检测优先级与依赖关系冲突（仅生成警告，不影响排序）
		pDep := shiftPriorityMap[dep.DependentShiftID]
		pOn := shiftPriorityMap[dep.DependentOnShiftID]
		if pDep > pOn {
			warnings = append(warnings, &OrganizationWarning{
				Type: "priority_dependency_conflict",
				Message: fmt.Sprintf("班次优先级与依赖关系冲突：「%s」(优先级%d) 被「%s」(优先级%d) 依赖，但将按班次优先级执行（优先级数字小的先排）。",
					shiftNameMap[dep.DependentShiftID], pDep,
					shiftNameMap[dep.DependentOnShiftID], pOn),
				ShiftID1:   dep.DependentShiftID,
				ShiftName1: shiftNameMap[dep.DependentShiftID],
				Priority1:  pDep,
				ShiftID2:   dep.DependentOnShiftID,
				ShiftName2: shiftNameMap[dep.DependentOnShiftID],
				Priority2:  pOn,
				Resolution: "use_priority",
			})
		}

		// 收集 required_together 强约束分组
		if dep.DependencyType == RuleTypeRequiredTogether {
			groupFound := false
			for i, g := range shiftGroups {
				if g.RuleID == dep.RuleID {
					existingSet := make(map[string]bool, len(g.ShiftIDs))
					for _, id := range g.ShiftIDs {
						existingSet[id] = true
					}
					if !existingSet[dep.DependentShiftID] {
						shiftGroups[i].ShiftIDs = append(shiftGroups[i].ShiftIDs, dep.DependentShiftID)
					}
					if !existingSet[dep.DependentOnShiftID] {
						shiftGroups[i].ShiftIDs = append(shiftGroups[i].ShiftIDs, dep.DependentOnShiftID)
					}
					groupFound = true
					break
				}
			}
			if !groupFound {
				shiftGroups = append(shiftGroups, ShiftGroup{
					ShiftIDs: []string{dep.DependentShiftID, dep.DependentOnShiftID},
					RuleType: dep.DependencyType,
					RuleID:   dep.RuleID,
				})
			}
		}
	}

	// 按 SchedulingPriority 排序，相同时按名称排序
	result := make([]string, 0, len(shifts))
	for _, shift := range shifts {
		result = append(result, shift.ID)
	}
	sort.Slice(result, func(i, j int) bool {
		p1, p2 := shiftPriorityMap[result[i]], shiftPriorityMap[result[j]]
		if p1 != p2 {
			return p1 < p2
		}
		return shiftNameMap[result[i]] < shiftNameMap[result[j]]
	})

	// 组内去重
	for i := range shiftGroups {
		m := make(map[string]bool)
		unique := make([]string, 0)
		for _, id := range shiftGroups[i].ShiftIDs {
			if !m[id] {
				m[id] = true
				unique = append(unique, id)
			}
		}
		shiftGroups[i].ShiftIDs = unique
	}

	return result, warnings, shiftGroups, nil
}
