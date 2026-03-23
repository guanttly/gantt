package executor

import (
	"context"
	"fmt"
	"strings"

	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// DependencyAnalyzer 依赖关系分析器
type DependencyAnalyzer struct {
	logger logging.ILogger
}

// NewDependencyAnalyzer 创建依赖关系分析器
func NewDependencyAnalyzer(logger logging.ILogger) *DependencyAnalyzer {
	return &DependencyAnalyzer{logger: logger}
}

// AnalyzeRuleDependencies 分析规则依赖关系
func (a *DependencyAnalyzer) AnalyzeRuleDependencies(
	ctx context.Context,
	rules []*model.Rule,
) ([]*RuleDependency, error) {
	dependencies := make([]*RuleDependency, 0)

	for i, rule1 := range rules {
		for j, rule2 := range rules {
			if i == j {
				continue
			}

			// 检测来源依赖
			if dep := a.detectSourceDependency(rule1, rule2); dep != nil {
				dependencies = append(dependencies, dep)
			}

			// 检测资源预留
			if dep := a.detectResourceReservation(rule1, rule2); dep != nil {
				dependencies = append(dependencies, dep)
			}

			// 检测顺序依赖
			if dep := a.detectOrderDependency(rule1, rule2); dep != nil {
				dependencies = append(dependencies, dep)
			}
		}
	}

	return dependencies, nil
}

// AnalyzeShiftDependencies 分析班次依赖关系
func (a *DependencyAnalyzer) AnalyzeShiftDependencies(
	ctx context.Context,
	rules []*model.Rule,
	shifts []*model.Shift,
) ([]*ShiftDependency, error) {
	dependencies := make([]*ShiftDependency, 0)
	shiftIDMap := make(map[string]*model.Shift)
	for _, shift := range shifts {
		shiftIDMap[shift.ID] = shift
	}

	for _, rule := range rules {
		// V4 强制要求 RuleType 必填，空值规则不予生效
		if rule.RuleType == "" {
			continue
		}

		// 排他规则不进入依赖分析管道，由 ExclusiveScheduler 和底层 ConstraintChecker 独立处理
		if rule.RuleType == RuleTypeExclusive {
			continue
		}

		// 先从规则的Associations中提取班次依赖关系
		// 使用 Role 字段区分 target/source
		var targetShiftIDs, sourceShiftIDs []string
		hasExplicitRole := false

		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeShift {
				// 优先使用 Role 字段
				switch assoc.Role {
				case model.RelationRoleTarget, model.RelationRoleSubject:
					// target/主体（subject）：受约束方 → 后排（DependentOnShiftID）
					targetShiftIDs = append(targetShiftIDs, assoc.AssociationID)
					hasExplicitRole = true
				case model.RelationRoleSource, model.RelationRoleObject:
					// source/客体（object）：参考方 → 先排（DependentShiftID）
					sourceShiftIDs = append(sourceShiftIDs, assoc.AssociationID)
					hasExplicitRole = true
				default:
					// 如果没有 Role 字段，按顺序推断
					if len(targetShiftIDs) == 0 {
						targetShiftIDs = append(targetShiftIDs, assoc.AssociationID)
					} else {
						sourceShiftIDs = append(sourceShiftIDs, assoc.AssociationID)
					}
				}
			}
		}

		// 根据结构化字段判断是否为依赖型规则（不再使用描述文本模糊推断）
		isDependency := false
		if rule.Category == "dependency" {
			isDependency = true
		} else if rule.RuleType == RuleTypeRequiredTogether {
			isDependency = true
		} else if hasExplicitRole && len(targetShiftIDs) > 0 && len(sourceShiftIDs) > 0 {
			// 管理后台的主客体班次角色标注明确，视为班次依赖规则
			isDependency = true
		}

		if !isDependency {
			continue
		}

		if len(targetShiftIDs) > 0 && len(sourceShiftIDs) > 0 {
			// 直接使用 RuleType 确定依赖类型（不再使用描述文本模糊推断）
			depType := DependencyTypeSource
			switch rule.RuleType {
			case RuleTypeRequiredTogether:
				depType = RuleTypeRequiredTogether
			default:
				// 有显式主客体角色标注时，兜底为 required_together
				if hasExplicitRole {
					depType = RuleTypeRequiredTogether
				}
			}

			for _, targetShiftID := range targetShiftIDs {
				if _, ok := shiftIDMap[targetShiftID]; !ok {
					continue
				}
				for _, sourceShiftID := range sourceShiftIDs {
					if _, ok := shiftIDMap[sourceShiftID]; !ok {
						continue
					}

					dependencies = append(dependencies, &ShiftDependency{
						DependentShiftID:   sourceShiftID, // 被依赖的班次（需要先排）
						DependentOnShiftID: targetShiftID, // 依赖的班次（后排）
						DependencyType:     depType,
						RuleID:             rule.ID,
						Description:        rule.Description,
						TimeOffsetDays:     getTimeOffsetDays(rule),
					})
				}
			}
		}
	}

	return dependencies, nil
}

func getTimeOffsetDays(rule *model.Rule) int {
	if rule.TimeOffsetDays != nil {
		return *rule.TimeOffsetDays
	}
	return 0
}

// detectSourceDependency 检测来源依赖
func (a *DependencyAnalyzer) detectSourceDependency(rule1, rule2 *model.Rule) *RuleDependency {
	// 检查rule2是否依赖rule1的数据来源
	// 例如："下夜班人员必须来自前一日的上半夜班"

	// 检查rule2是否是来源依赖规则
	if !contains(rule2.Description, "来源") {
		return nil
	}

	// 检查rule2的Associations中是否有source指向rule1的target
	for _, assoc2 := range rule2.Associations {
		// 优先使用 Role 字段判断是否为 source
		isSource := assoc2.Role == model.RelationRoleSource || (assoc2.Role == "" && contains(rule2.Description, "来源"))
		if !isSource {
			continue
		}

		// 检查rule1是否有对应的target
		for _, assoc1 := range rule1.Associations {
			isTarget := assoc1.Role == model.RelationRoleTarget || assoc1.Role == ""
			if isTarget && assoc1.AssociationID == assoc2.AssociationID {
				return &RuleDependency{
					DependentRuleID:   rule1.ID,
					DependentOnRuleID: rule2.ID,
					DependencyType:    DependencyTypeSource,
					Description:       fmt.Sprintf("%s的人员必须来自%s", rule2.Name, rule1.Name),
				}
			}
		}
	}

	return nil
}

// detectResourceReservation 检测资源预留
func (a *DependencyAnalyzer) detectResourceReservation(rule1, rule2 *model.Rule) *RuleDependency {
	// 检查rule1是否需要为rule2预留资源
	// 例如："当日上半夜班人员需保留给次日下夜班"

	// 检查rule1是否是资源预留规则
	if !contains(rule1.Description, "保留") && !contains(rule1.Description, "预留") {
		return nil
	}

	// 检查rule1的Associations中是否有resource指向rule2
	for _, assoc1 := range rule1.Associations {
		// 优先使用 Role 字段判断是否为 resource
		isResource := assoc1.Role == "resource" || (assoc1.Role == "" && (contains(rule1.Description, "保留") || contains(rule1.Description, "预留")))
		if !isResource {
			continue
		}

		// 检查rule2是否有对应的target
		for _, assoc2 := range rule2.Associations {
			if assoc2.AssociationID == assoc1.AssociationID {
				return &RuleDependency{
					DependentRuleID:   rule1.ID,
					DependentOnRuleID: rule2.ID,
					DependencyType:    DependencyTypeResource,
					Description:       fmt.Sprintf("%s的人员需保留给%s", rule1.Name, rule2.Name),
				}
			}
		}
	}

	return nil
}

// detectOrderDependency 检测顺序依赖
func (a *DependencyAnalyzer) detectOrderDependency(rule1, rule2 *model.Rule) *RuleDependency {
	// 检查rule2是否必须在rule1之后执行

	// 检查rule2是否是顺序依赖规则
	if !contains(rule2.Description, "顺序") && !contains(rule2.Description, "之后") {
		return nil
	}

	// 简单的顺序依赖检测：如果rule2的描述中提到rule1
	if contains(rule2.Description, rule1.Name) {
		return &RuleDependency{
			DependentRuleID:   rule1.ID,
			DependentOnRuleID: rule2.ID,
			DependencyType:    DependencyTypeOrder,
			Description:       fmt.Sprintf("%s必须在%s之后执行", rule2.Name, rule1.Name),
		}
	}

	return nil
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(s, substr)
}
