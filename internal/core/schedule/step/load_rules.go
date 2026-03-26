package step

import (
"context"

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
state.EffectiveRules = effectiveRules.Rules

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
state.ShiftOrder = filtered
} else {
state.ShiftOrder = shiftOrder
}

return nil
}
