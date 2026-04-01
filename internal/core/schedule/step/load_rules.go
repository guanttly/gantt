package step

import (
	"context"
	"sort"

	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/tenant"
)

// LoadRulesStep 加载生效规则和班次拓扑排序。
type LoadRulesStep struct {
	RuleService  *rule.Service
	ShiftService *shift.Service
}

// Name 返回步骤名称。
func (s *LoadRulesStep) Name() string { return "LoadRules" }

// Execute 执行加载规则步骤。
func (s *LoadRulesStep) Execute(ctx context.Context, state *ScheduleState) error {
	nodeID := tenant.GetOrgNodeID(ctx)

	// 1. 计算生效规则集
	effectiveRules, err := s.RuleService.ComputeEffectiveRules(ctx, nodeID)
	if err != nil {
		return err
	}
	state.EffectiveRules = rule.OrderRulesForExecution(effectiveRules.Rules)

	// 2. 获取班次拓扑排序
	shiftOrder, err := s.ShiftService.GetTopologicalOrder(ctx)
	if err != nil {
		return err
	}

	// 如果配置了参与班次，只保留配置中的班次
	if state.Config != nil && len(state.Config.ShiftIDs) > 0 {
		shiftIDSet := make(map[string]bool)
		for _, id := range state.Config.ShiftIDs {
			shiftIDSet[id] = true
		}
		var filtered []shift.Shift
		for _, sh := range shiftOrder {
			if shiftIDSet[sh.ID] {
				filtered = append(filtered, sh)
			}
		}
		state.ShiftOrder = sortShiftOrderByRuleDependencies(filtered, state.EffectiveRules)
	} else {
		state.ShiftOrder = sortShiftOrderByRuleDependencies(shiftOrder, state.EffectiveRules)
	}

	return nil
}

func sortShiftOrderByRuleDependencies(shiftOrder []shift.Shift, rules []rule.Rule) []shift.Shift {
	if len(shiftOrder) <= 1 || len(rules) == 0 {
		return shiftOrder
	}

	shiftMap := make(map[string]shift.Shift, len(shiftOrder))
	orderIndex := make(map[string]int, len(shiftOrder))
	inDegree := make(map[string]int, len(shiftOrder))
	graph := make(map[string]map[string]struct{})

	for idx, sh := range shiftOrder {
		shiftMap[sh.ID] = sh
		orderIndex[sh.ID] = idx
		inDegree[sh.ID] = 0
	}

	addEdge := func(fromID, toID string) {
		if fromID == "" || toID == "" || fromID == toID {
			return
		}
		if _, ok := shiftMap[fromID]; !ok {
			return
		}
		if _, ok := shiftMap[toID]; !ok {
			return
		}
		if graph[fromID] == nil {
			graph[fromID] = make(map[string]struct{})
		}
		if _, exists := graph[fromID][toID]; exists {
			return
		}
		graph[fromID][toID] = struct{}{}
		inDegree[toID]++
	}

	for _, currentRule := range rules {
		switch currentRule.SubType {
		case rule.SubTypeSource:
			semantics := currentRule.SourceSemantics()
			for _, sourceID := range semantics.SourceShiftIDs {
				for _, targetID := range semantics.TargetShiftIDs {
					addEdge(sourceID, targetID)
				}
			}
		case rule.SubTypeMust:
			semantics := currentRule.RequiredTogetherSemantics()
			for _, requiredID := range semantics.RequiredShiftIDs {
				for _, targetID := range semantics.TargetShiftIDs {
					addEdge(requiredID, targetID)
				}
			}
		case rule.SubTypeOrder:
			semantics := currentRule.OrderSemantics()
			for _, beforeID := range semantics.BeforeShiftIDs {
				for _, targetID := range semantics.TargetShiftIDs {
					addEdge(beforeID, targetID)
				}
			}
		}
	}

	queue := make([]string, 0, len(shiftOrder))
	for _, sh := range shiftOrder {
		if inDegree[sh.ID] == 0 {
			queue = append(queue, sh.ID)
		}
	}

	result := make([]shift.Shift, 0, len(shiftOrder))
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		result = append(result, shiftMap[currentID])

		nextIDs := make([]string, 0, len(graph[currentID]))
		for nextID := range graph[currentID] {
			nextIDs = append(nextIDs, nextID)
		}
		sort.Slice(nextIDs, func(i, j int) bool {
			return orderIndex[nextIDs[i]] < orderIndex[nextIDs[j]]
		})

		for _, nextID := range nextIDs {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = insertShiftQueue(queue, nextID, orderIndex)
			}
		}
	}

	if len(result) != len(shiftOrder) {
		return shiftOrder
	}

	return result
}

func insertShiftQueue(queue []string, shiftID string, orderIndex map[string]int) []string {
	insertAt := len(queue)
	for idx, currentID := range queue {
		if orderIndex[shiftID] < orderIndex[currentID] {
			insertAt = idx
			break
		}
	}
	queue = append(queue, "")
	copy(queue[insertAt+1:], queue[insertAt:])
	queue[insertAt] = shiftID
	return queue
}
