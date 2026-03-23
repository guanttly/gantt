package executor

import (
	"context"
	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
)

// RequiredTogetherScheduler 必须同时排班调度器（精简版）
//
// 重构后简化为 DependencyResolver 的薄代理：
//   - CanHandle 仅判断班次是否属于 required_together 组
//   - Schedule 委托给 DependencyResolver.Resolve()
//
// 注意：在新架构中，V4Executor.executePhaseOne 已直接调用 DependencyResolver，
// 此调度器保留仅为向后兼容（如 ExecuteSingleShiftDate 等旧路径）。
type RequiredTogetherScheduler struct {
	logger     logging.ILogger
	ruleEngine *engine.RuleEngine
}

func NewRequiredTogetherScheduler(
	logger logging.ILogger,
	ruleEngine *engine.RuleEngine,
) *RequiredTogetherScheduler {
	return &RequiredTogetherScheduler{
		logger:     logger,
		ruleEngine: ruleEngine,
	}
}

func (s *RequiredTogetherScheduler) Name() string {
	return "RequiredTogetherScheduler"
}

func (s *RequiredTogetherScheduler) Priority() int {
	return 20 // 优先级低于 Personnel，高于 Exclusive 和 Default
}

func (s *RequiredTogetherScheduler) CanHandle(
	ctx context.Context,
	input *SchedulingExecutionInput,
	rule *model.Rule,
	shiftID string,
	date string,
) bool {
	// 检查当前班次是否属于必同组
	for _, group := range input.RuleOrganization.ShiftGroups {
		if group.RuleType == RuleTypeRequiredTogether {
			for _, id := range group.ShiftIDs {
				if id == shiftID {
					return true
				}
			}
		}
	}
	return false
}

func (s *RequiredTogetherScheduler) Schedule(
	ctx context.Context,
	input *SchedulingExecutionInput,
	shiftID string,
	dateStr string,
	requiredCount int,
) ([]string, bool, error) {
	if requiredCount <= 0 {
		return []string{}, false, nil
	}

	s.logger.Info("[RequiredTogether] 委托给 DependencyResolver",
		"shiftID", shiftID, "date", dateStr, "requiredCount", requiredCount)

	// 委托给 DependencyResolver（通过 SingleShiftScheduler）
	singleScheduler := NewSingleShiftScheduler(s.logger, s.ruleEngine)
	resolver := NewDependencyResolver(s.logger, singleScheduler)

	resolveResult, err := resolver.Resolve(
		ctx, shiftID, dateStr, requiredCount,
		model.AssignmentSourceRule, input,
	)
	if err != nil {
		return nil, false, err
	}

	// 提取人员ID列表
	staffIDs := make([]string, 0, len(resolveResult.Assignments))
	for _, a := range resolveResult.Assignments {
		staffIDs = append(staffIDs, a.StaffID)
	}

	// required_together 处理完毕后终止调度器链（避免后续调度器重复处理）
	return staffIDs, true, nil
}
