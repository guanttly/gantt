package executor

import (
	"context"
	"jusha/agent/rostering/domain/model"
)

// IScheduler 排班策略调度器接口
type IScheduler interface {
	// Name 调度器的名称或标识
	Name() string

	// Priority 调度器的执行优先级（数字越小优先级越高）
	// 例如：Personnel (10) -> RequiredTogether (20) -> Exclusive (30) -> Default (100)
	Priority() int

	// CanHandle 能否处理指定的 RuleType 或场景
	CanHandle(ctx context.Context, input *SchedulingExecutionInput, rule *model.Rule, shiftID string, date string) bool

	// Schedule 执行排班策略
	// 返回：
	// 1. 排好班的员工ID列表
	// 2. terminateChain 标志（true 表示责任链到此终止，不再执行后续优先级更低的调度器，例如防止兜底调度器填充仅作为客体的班次）
	// 3. 错误信息
	Schedule(ctx context.Context, input *SchedulingExecutionInput, shiftID string, date string, requiredCount int) ([]string, bool, error)
}
