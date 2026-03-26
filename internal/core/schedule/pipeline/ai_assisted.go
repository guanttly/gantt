package pipeline

import (
	"gantt-saas/internal/ai"
	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/leave"
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/schedule/step"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/infra/websocket"

	"go.uber.org/zap"
)

// AIAssistedDeps AI 辅助排班 Pipeline 的依赖。
type AIAssistedDeps struct {
	RuleService  *rule.Service
	ShiftService *shift.Service
	EmployeeRepo *employee.Repository
	LeaveRepo    *leave.Repository
	DraftSaver   step.DraftSaver
	AIProvider   ai.Provider
	Broadcaster  websocket.Broadcaster // 可选
	Logger       *zap.Logger
}

// NewAIAssistedPipeline 创建 AI 辅助排班 Pipeline。
// 流程：加载规则 → 过滤候选人 → 固定占位 → 规则占位 → AI智能选人 → 兜底填充 → 全规则校验 → 保存草稿 → 通知
func NewAIAssistedPipeline(deps *AIAssistedDeps) *Pipeline {
	return NewPipeline("ai_assisted", deps.Logger,
		&step.LoadRulesStep{
			RuleService:  deps.RuleService,
			ShiftService: deps.ShiftService,
		},
		&step.FilterCandidatesStep{
			EmployeeRepo: deps.EmployeeRepo,
			LeaveRepo:    deps.LeaveRepo,
		},
		&step.PhaseZeroStep{},
		&step.PhaseOneStep{},
		&step.AISelectStep{
			Provider: deps.AIProvider,
			Logger:   deps.Logger,
		},
		&step.PhaseTwoStep{}, // 兜底填充 AI 未覆盖的空缺
		&step.FullValidationStep{},
		&step.SaveDraftStep{
			Repo: deps.DraftSaver,
		},
		&step.NotifyWSStep{Broadcaster: deps.Broadcaster},
	)
}
