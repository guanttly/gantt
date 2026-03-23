// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"sync"
	"time"
)

// ============================================================
// System 子工作流超时监控
// ============================================================

// subWorkflowTimer 子工作流超时计时器
type subWorkflowTimer struct {
	timer      *time.Timer
	errorEvent Event
}

// 子工作流超时监控 map（sessionID -> timer）
var (
	subWorkflowTimers   = make(map[string]*subWorkflowTimer)
	subWorkflowTimersMu sync.RWMutex
)

// startSubWorkflowTimeout 启动子工作流超时监控
// timeout <= 0 表示不启用超时
func (s *System) startSubWorkflowTimeout(sessionID string, timeout time.Duration, errorEvent Event) {
	if timeout <= 0 {
		return
	}

	// 先取消可能存在的旧计时器
	s.cancelSubWorkflowTimeout(sessionID)

	subWorkflowTimersMu.Lock()
	defer subWorkflowTimersMu.Unlock()

	timer := time.AfterFunc(timeout, func() {
		s.handleSubWorkflowTimeout(sessionID, errorEvent)
	})

	subWorkflowTimers[sessionID] = &subWorkflowTimer{
		timer:      timer,
		errorEvent: errorEvent,
	}

	if s.context != nil && s.context.Logger() != nil {
		s.context.Logger().Debug("started sub-workflow timeout",
			"sessionID", sessionID,
			"timeout", timeout,
			"errorEvent", errorEvent,
		)
	}
}

// cancelSubWorkflowTimeout 取消子工作流超时监控
func (s *System) cancelSubWorkflowTimeout(sessionID string) {
	subWorkflowTimersMu.Lock()
	defer subWorkflowTimersMu.Unlock()

	if t, ok := subWorkflowTimers[sessionID]; ok {
		t.timer.Stop()
		delete(subWorkflowTimers, sessionID)

		if s.context != nil && s.context.Logger() != nil {
			s.context.Logger().Debug("cancelled sub-workflow timeout",
				"sessionID", sessionID,
			)
		}
	}
}

// handleSubWorkflowTimeout 处理子工作流超时
func (s *System) handleSubWorkflowTimeout(sessionID string, errorEvent Event) {
	subWorkflowTimersMu.Lock()
	if _, ok := subWorkflowTimers[sessionID]; !ok {
		// 已被取消
		subWorkflowTimersMu.Unlock()
		return
	}
	delete(subWorkflowTimers, sessionID)
	subWorkflowTimersMu.Unlock()

	// 空指针检查
	if s.context == nil {
		return
	}

	logger := s.context.Logger()
	if logger != nil {
		logger.Warn("sub-workflow timeout",
			"sessionID", sessionID,
			"errorEvent", errorEvent,
		)
	}

	// 获取 Actor
	s.mu.RLock()
	actor, ok := s.actors[sessionID]
	s.mu.RUnlock()

	if !ok || actor == nil {
		if logger != nil {
			logger.Error("actor not found for timeout handling",
				"sessionID", sessionID,
			)
		}
		return
	}

	// 强制回滚到父工作流
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := &SubWorkflowResult{
		Success:  false,
		ErrorMsg: "sub-workflow execution timeout",
	}

	if err := actor.ReturnToParent(ctx, result); err != nil {
		if logger != nil {
			logger.Error("failed to rollback on timeout",
				"sessionID", sessionID,
				"error", err,
			)
		}
	}

	// 记录超时指标
	if s.subWorkflowMetrics != nil {
		s.subWorkflowMetrics.RecordTimeout()
	}
}

// ============================================================
// System 并行子工作流支持
// ============================================================

// SpawnParallelSubWorkflowsAsync 异步启动多个并行子工作流
// 这是一个高级功能，允许在不同的 goroutine 中并行执行多个子工作流
// 注意：当前实现为简化版，真正的并行执行需要更复杂的编排
func (s *System) SpawnParallelSubWorkflowsAsync(ctx context.Context, sessionID string, config ParallelSubWorkflowConfig) error {
	if len(config.Configs) == 0 {
		return nil
	}

	logger := s.context.Logger()

	// 获取主 Actor
	s.mu.RLock()
	actor, ok := s.actors[sessionID]
	s.mu.RUnlock()

	if !ok || actor == nil {
		return ErrActorNotFound
	}

	// 使用主 Actor 的 SpawnParallelSubWorkflows 方法
	if err := actor.SpawnParallelSubWorkflows(ctx, config); err != nil {
		if logger != nil {
			logger.Error("failed to spawn parallel sub-workflows",
				"sessionID", sessionID,
				"count", len(config.Configs),
				"error", err,
			)
		}
		return err
	}

	return nil
}

// ============================================================
// 子工作流诊断信息
// ============================================================

// SubWorkflowDiagnostics 子工作流诊断信息
type SubWorkflowDiagnostics struct {
	SessionID     string        `json:"session_id"`
	CurrentDepth  int           `json:"current_depth"`
	IsSubWorkflow bool          `json:"is_sub_workflow"`
	HasTimeout    bool          `json:"has_timeout"`
	TimeoutLeft   time.Duration `json:"timeout_left,omitempty"`
	ParentInfo    *ParentInfo   `json:"parent_info,omitempty"`
}

// ParentInfo 父工作流信息
type ParentInfo struct {
	Workflow string `json:"workflow"`
	Phase    string `json:"phase"`
}

// GetSubWorkflowDiagnostics 获取子工作流诊断信息
func (s *System) GetSubWorkflowDiagnostics(sessionID string) *SubWorkflowDiagnostics {
	s.mu.RLock()
	actor, ok := s.actors[sessionID]
	s.mu.RUnlock()

	if !ok || actor == nil {
		return nil
	}

	diag := &SubWorkflowDiagnostics{
		SessionID:     sessionID,
		CurrentDepth:  actor.GetWorkflowDepth(),
		IsSubWorkflow: actor.IsSubWorkflow(),
	}

	// 检查是否有超时
	subWorkflowTimersMu.RLock()
	if t, ok := subWorkflowTimers[sessionID]; ok {
		diag.HasTimeout = true
		// 注意：这里无法精确获取剩余时间，只能标记存在
		_ = t // 避免 unused variable
	}
	subWorkflowTimersMu.RUnlock()

	// 获取父工作流信息
	if diag.IsSubWorkflow {
		parentWorkflow := actor.GetParentWorkflow()
		parentPhase := actor.GetParentPhase()
		if parentWorkflow != "" {
			diag.ParentInfo = &ParentInfo{
				Workflow: parentWorkflow,
				Phase:    parentPhase,
			}
		}
	}

	return diag
}

// ============================================================
// 错误定义
// ============================================================

// 子工作流相关错误
var (
	ErrActorNotFound = &ActorNotFoundError{}
)

// ActorNotFoundError Actor 未找到错误
type ActorNotFoundError struct{}

func (e *ActorNotFoundError) Error() string {
	return "actor not found"
}
