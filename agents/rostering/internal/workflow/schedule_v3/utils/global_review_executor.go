package utils

import (
	"context"
	"fmt"
	"time"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/agent/rostering/internal/service"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// 全局校验执行器
// 用于在草案完成后执行规则和个人需求的逐条评审、对论迭代和批量修改
// ============================================================

// IGlobalReviewExecutor 全局校验执行器接口
type IGlobalReviewExecutor interface {
	// Execute 执行全局校验流程
	// 1. 合并规则和个人需求为评审项
	// 2. 逐条评审收集修改意见
	// 3. 检测冲突，冲突项标记人工介入
	// 4. 非冲突项进入对论迭代
	// 5. 应用通过的修改意见
	// 6. 返回结果（含需人工处理的项目）
	Execute(ctx context.Context) (*d_model.GlobalReviewResult, error)

	// SetProgressCallback 设置进度回调
	SetProgressCallback(callback d_model.GlobalReviewProgressCallback)
}

// GlobalReviewExecutor 全局校验执行器实现
type GlobalReviewExecutor struct {
	// 服务依赖
	reviewService d_service.IGlobalReviewService
	logger        logging.ILogger

	// 上下文数据
	taskContext *CoreV3TaskContext

	// 配置
	maxDebateRounds int

	// 回调
	progressCallback d_model.GlobalReviewProgressCallback
}

// GlobalReviewExecutorOption 全局校验执行器配置选项
type GlobalReviewExecutorOption func(*GlobalReviewExecutor)

// WithMaxDebateRounds 设置最大对论轮次
func WithMaxDebateRounds(rounds int) GlobalReviewExecutorOption {
	return func(e *GlobalReviewExecutor) {
		if rounds > 0 {
			e.maxDebateRounds = rounds
		}
	}
}

// WithReviewProgressCallback 设置进度回调
func WithReviewProgressCallback(callback d_model.GlobalReviewProgressCallback) GlobalReviewExecutorOption {
	return func(e *GlobalReviewExecutor) {
		e.progressCallback = callback
	}
}

// NewGlobalReviewExecutor 创建全局校验执行器
func NewGlobalReviewExecutor(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	taskContext *CoreV3TaskContext,
	opts ...GlobalReviewExecutorOption,
) (IGlobalReviewExecutor, error) {
	if taskContext == nil {
		return nil, fmt.Errorf("taskContext is required")
	}

	// 创建评审服务
	reviewService, err := service.NewGlobalReviewService(logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create review service: %w", err)
	}

	executor := &GlobalReviewExecutor{
		reviewService:   reviewService,
		logger:          logger.With("component", "GlobalReviewExecutor"),
		taskContext:     taskContext,
		maxDebateRounds: 3, // 默认最大3轮对论
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(executor)
	}

	return executor, nil
}

// SetProgressCallback 设置进度回调
func (e *GlobalReviewExecutor) SetProgressCallback(callback d_model.GlobalReviewProgressCallback) {
	e.progressCallback = callback
}

// Execute 执行全局校验流程
func (e *GlobalReviewExecutor) Execute(ctx context.Context) (*d_model.GlobalReviewResult, error) {
	startTime := time.Now()
	e.logger.Info("开始全局校验流程",
		"rulesCount", len(e.taskContext.Rules),
		"maxDebateRounds", e.maxDebateRounds,
	)

	// 收集个人需求
	personalNeeds := e.collectPersonalNeeds()
	e.logger.Info("收集个人需求",
		"personalNeedsCount", len(personalNeeds),
	)

	// 执行全局评审
	result, err := e.reviewService.ExecuteGlobalReview(
		ctx,
		e.taskContext.Rules,
		personalNeeds,
		e.taskContext.WorkingDraft,
		e.taskContext.AllStaff,
		e.taskContext.Shifts,
		e.maxDebateRounds,
		e.progressCallback,
	)
	if err != nil {
		return nil, fmt.Errorf("执行全局评审失败: %w", err)
	}

	result.ExecutionTime = time.Since(startTime).Seconds()

	e.logger.Info("全局校验流程完成",
		"totalItems", result.TotalItems,
		"violatedItems", result.ViolatedItems,
		"needsManualReview", result.NeedsManualReview,
		"executionTime", result.ExecutionTime,
	)

	return result, nil
}

// collectPersonalNeeds 从任务上下文收集个人需求
func (e *GlobalReviewExecutor) collectPersonalNeeds() []*d_model.PersonalNeed {
	if e.taskContext.PersonalNeeds == nil {
		return nil
	}

	needs := make([]*d_model.PersonalNeed, 0)
	for _, staffNeeds := range e.taskContext.PersonalNeeds {
		needs = append(needs, staffNeeds...)
	}
	return needs
}

// ============================================================
// 辅助类型：全局校验执行结果处理
// ============================================================

// GlobalReviewResultHandler 全局校验结果处理器
type GlobalReviewResultHandler struct {
	Result *d_model.GlobalReviewResult
}

// NewGlobalReviewResultHandler 创建结果处理器
func NewGlobalReviewResultHandler(result *d_model.GlobalReviewResult) *GlobalReviewResultHandler {
	return &GlobalReviewResultHandler{Result: result}
}

// HasManualReviewItems 是否有需人工处理的项目
func (h *GlobalReviewResultHandler) HasManualReviewItems() bool {
	return h.Result != nil && h.Result.NeedsManualReview
}

// GetManualReviewSummary 获取需人工处理项目的摘要
func (h *GlobalReviewResultHandler) GetManualReviewSummary() string {
	if h.Result == nil || len(h.Result.ManualReviewItems) == 0 {
		return ""
	}

	summary := fmt.Sprintf("以下%d项需要您人工处理：\n", len(h.Result.ManualReviewItems))
	for i, item := range h.Result.ManualReviewItems {
		status := "冲突"
		if item.Status == d_model.OpinionStatusManualReview {
			status = "未达共识"
		}
		summary += fmt.Sprintf("%d. [%s] %s: %s\n", i+1, status, item.ReviewItemName, item.ViolationDescription)
	}
	return summary
}

// GetModifiedDraft 获取修改后的草案
func (h *GlobalReviewResultHandler) GetModifiedDraft() *d_model.ScheduleDraft {
	if h.Result == nil {
		return nil
	}
	return h.Result.ModifiedDraft
}

// GetApprovedChangesCount 获取通过的修改数量
func (h *GlobalReviewResultHandler) GetApprovedChangesCount() int {
	if h.Result == nil || h.Result.DebateResult == nil {
		return 0
	}
	return len(h.Result.DebateResult.ApprovedOpinions)
}

// GetReviewSummaryForUser 获取面向用户的评审摘要
func (h *GlobalReviewResultHandler) GetReviewSummaryForUser() string {
	if h.Result == nil {
		return "全局评审未执行"
	}

	summary := "📋 全局评审完成\n\n"
	summary += fmt.Sprintf("• 评审项总数: %d\n", h.Result.TotalItems)
	summary += fmt.Sprintf("• 发现违规: %d项\n", h.Result.ViolatedItems)

	if h.Result.DebateResult != nil {
		summary += fmt.Sprintf("• 对论轮次: %d轮\n", h.Result.DebateResult.TotalRounds)
		summary += fmt.Sprintf("• 通过修改: %d项\n", len(h.Result.DebateResult.ApprovedOpinions))
		summary += fmt.Sprintf("• 拒绝修改: %d项\n", len(h.Result.DebateResult.RejectedOpinions))
	}

	if h.Result.NeedsManualReview {
		summary += fmt.Sprintf("\n⚠️ 有%d项需要人工处理\n", len(h.Result.ManualReviewItems))
	} else {
		summary += "\n✅ 所有评审项已自动处理完成\n"
	}

	summary += fmt.Sprintf("\n耗时: %.1f秒", h.Result.ExecutionTime)
	return summary
}
