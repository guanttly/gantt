// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"fmt"
	"time"

	"jusha/mcp/pkg/workflow/session"
)

// 编译时接口断言：确保 Actor 实现了 SubWorkflowContext 接口
var _ SubWorkflowContext = (*Actor)(nil)

// ============================================================
// Actor 子工作流方法实现
// ============================================================

// SpawnSubWorkflow 启动子工作流
// 这会：
// 1. 检查嵌套深度限制
// 2. 创建当前状态快照
// 3. 压栈当前工作流状态
// 4. 切换到子工作流
// 5. 触发子工作流启动事件
//
// 重要：这个方法会同步更新 Actor 的 definition、state 和 trans，
// 确保后续事件可以正确路由到子工作流
func (a *Actor) SpawnSubWorkflow(ctx context.Context, config SubWorkflowConfig) error {
	logger := a.Logger()
	sess := a.Session()
	if sess == nil {
		logger.Error("SpawnSubWorkflow: session not found", "sessionID", a.id)
		return fmt.Errorf("session not found")
	}
	if sess.WorkflowMeta == nil {
		logger.Error("SpawnSubWorkflow: workflow meta not initialized", "sessionID", a.id)
		return fmt.Errorf("workflow meta not initialized")
	}

	// 1. 检查嵌套深度
	currentDepth := sess.WorkflowMeta.GetWorkflowDepth()
	if currentDepth >= session.MaxWorkflowDepth {
		logger.Error("max workflow depth exceeded",
			"currentDepth", currentDepth,
			"maxDepth", session.MaxWorkflowDepth,
			"parentWorkflow", sess.WorkflowMeta.Workflow,
			"childWorkflow", config.WorkflowName,
		)
		return session.ErrMaxWorkflowDepthExceeded
	}

	// 2. 获取子工作流定义
	childDef := Get(config.WorkflowName)
	if childDef == nil {
		logger.Error("SpawnSubWorkflow: sub-workflow not registered",
			"sessionID", a.id,
			"childWorkflow", config.WorkflowName,
		)
		return fmt.Errorf("sub-workflow not registered: %s", config.WorkflowName)
	}

	// 2.1 验证子工作流启动事件是有效的
	if err := childDef.ValidateEvent(childDef.InitialState, EventSubWorkflowStart); err != nil {
		logger.Error("SpawnSubWorkflow: sub-workflow has no start transition",
			"sessionID", a.id,
			"childWorkflow", config.WorkflowName,
			"initialState", childDef.InitialState,
			"error", err,
		)
		// 不返回错误，仍然尝试启动
	}

	// 3. 创建状态快照
	var snapshotData map[string]any
	if len(config.SnapshotKeys) > 0 {
		snapshotData = CreateDataSnapshot(sess, config.SnapshotKeys)
	}

	// 4. 构建栈帧
	frame := session.WorkflowFrame{
		Workflow:     sess.WorkflowMeta.Workflow,
		Phase:        string(sess.WorkflowMeta.Phase),
		ReturnEvent:  string(config.OnComplete),
		ErrorEvent:   string(config.OnError),
		SnapshotKeys: config.SnapshotKeys,
		SnapshotData: snapshotData,
		StartedAt:    time.Now(),
	}

	// 5. 压栈
	if err := sess.WorkflowMeta.PushWorkflowFrame(frame); err != nil {
		return fmt.Errorf("failed to push workflow frame: %w", err)
	}

	// 6. 切换到子工作流
	previousWorkflow := sess.WorkflowMeta.Workflow
	sess.WorkflowMeta.Workflow = string(config.WorkflowName)
	sess.WorkflowMeta.Phase = childDef.InitialState

	// 7. 设置子工作流输入数据
	if config.Input != nil {
		sess.Data[DataKeySubWorkflowInput] = config.Input
	}
	sess.Data[DataKeySubWorkflowStartTime] = time.Now()

	// 8. 更新 Session
	_, err := a.SessionService().Update(ctx, a.id, func(s *session.Session) error {
		s.WorkflowMeta = sess.WorkflowMeta
		s.Data = sess.Data
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update session for sub-workflow: %w", err)
	}

	// 9. 切换 Actor 的工作流定义（关键步骤！）
	// 必须按正确顺序执行：先设置 definition，再设置 state，最后初始化 transitions
	a.definition = childDef
	a.state = childDef.InitialState
	a.workflowSwitched = true // 标记已切换工作流，防止外层 handle() 覆盖状态
	a.initTransitions()       // 必须在设置 definition 和 state 之后调用

	// 10. 记录日志
	logger.Info("SpawnSubWorkflow: switched to child workflow",
		"sessionID", a.id,
		"parentWorkflow", previousWorkflow,
		"childWorkflow", config.WorkflowName,
		"newState", a.state,
		"depth", currentDepth+1,
		"timeout", config.Timeout,
		"onComplete", config.OnComplete,
		"onError", config.OnError,
	)

	// 10.1 记录子工作流指标
	if a.subWorkflowMetrics != nil {
		a.subWorkflowMetrics.RecordSpawn(
			previousWorkflow,
			string(config.WorkflowName),
			currentDepth+1,
		)
	}

	// 11. 调用子工作流进入钩子
	if childDef.OnSubWorkflowEnter != nil {
		if err := childDef.OnSubWorkflowEnter(a, previousWorkflow); err != nil {
			logger.Warn("sub-workflow enter hook failed", "error", err)
		}
	}

	// 12. 启动超时监控（如果配置了超时且不是无限等待）
	if config.Timeout > 0 && a.system != nil {
		a.system.startSubWorkflowTimeout(a.id, config.Timeout, config.OnError)
	}

	// 13. 发送子工作流启动事件
	return a.Send(ctx, EventSubWorkflowStart, config.Input)
}

// ReturnToParent 从子工作流返回父工作流
// 这会：
// 1. 验证当前在子工作流中
// 2. 调用子工作流退出钩子
// 3. 弹栈恢复父工作流状态
// 4. 清理子工作流数据
// 5. 切换 Actor 状态和转换表
// 6. 触发父工作流的回调事件
//
// 重要：这个方法会同步更新 Actor 的 definition、state 和 trans，
// 确保后续事件可以正确路由到父工作流
func (a *Actor) ReturnToParent(ctx context.Context, result *SubWorkflowResult) error {
	logger := a.Logger()
	sess := a.Session()
	if sess == nil {
		logger.Error("ReturnToParent: session not found", "sessionID", a.id)
		return fmt.Errorf("session not found")
	}
	if sess.WorkflowMeta == nil {
		logger.Error("ReturnToParent: workflow meta not initialized", "sessionID", a.id)
		return fmt.Errorf("workflow meta not initialized")
	}

	// 1. 验证在子工作流中
	if !sess.WorkflowMeta.IsSubWorkflow() {
		logger.Error("ReturnToParent: not in sub-workflow",
			"sessionID", a.id,
			"currentWorkflow", sess.WorkflowMeta.Workflow,
		)
		return session.ErrNotInSubWorkflow
	}

	// 2. 取消超时监控
	if a.system != nil {
		a.system.cancelSubWorkflowTimeout(a.id)
	}

	// 3. 计算执行时长
	if startTime, ok := sess.Data[DataKeySubWorkflowStartTime].(time.Time); ok {
		result.Duration = time.Since(startTime)
	}

	// 4. 调用子工作流退出钩子
	if a.definition.OnSubWorkflowExit != nil {
		if err := a.definition.OnSubWorkflowExit(a, result.Success); err != nil {
			logger.Warn("sub-workflow exit hook failed", "error", err)
		}
	}

	// 5. 弹栈获取父工作流信息
	frame, err := sess.WorkflowMeta.PopWorkflowFrame()
	if err != nil {
		return fmt.Errorf("failed to pop workflow frame: %w", err)
	}

	childWorkflow := sess.WorkflowMeta.Workflow

	// 6. 恢复父工作流
	parentDef := Get(Workflow(frame.Workflow))
	if parentDef == nil {
		return fmt.Errorf("parent workflow not registered: %s", frame.Workflow)
	}

	sess.WorkflowMeta.Workflow = frame.Workflow
	sess.WorkflowMeta.Phase = session.WorkflowState(frame.Phase)

	// 7. 保存子工作流结果
	sess.Data[DataKeySubWorkflowResult] = result

	// 8. 如果失败且需要回滚，恢复快照
	if !result.Success && len(frame.SnapshotData) > 0 {
		for key, value := range frame.SnapshotData {
			sess.Data[key] = value
		}
		logger.Info("restored data snapshot after sub-workflow failure",
			"restoredKeys", frame.SnapshotKeys,
		)
	}

	// 9. 清理子工作流数据
	delete(sess.Data, DataKeySubWorkflowInput)
	delete(sess.Data, DataKeySubWorkflowStartTime)

	// 10. 更新 Session
	_, err = a.SessionService().Update(ctx, a.id, func(s *session.Session) error {
		s.WorkflowMeta = sess.WorkflowMeta
		s.Data = sess.Data
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update session for return: %w", err)
	}

	// 11. 切换 Actor 的工作流定义（关键步骤！）
	// 必须按正确顺序执行：先设置 definition，再设置 state，最后初始化 transitions
	previousDefinition := a.definition.Name
	previousState := a.state

	a.definition = parentDef
	a.state = State(frame.Phase)
	a.workflowSwitched = true // 标记已切换工作流，防止外层 handle() 覆盖状态
	a.initTransitions()       // 必须在设置 definition 和 state 之后调用

	logger.Info("ReturnToParent: switched to parent workflow",
		"sessionID", a.id,
		"previousWorkflow", previousDefinition,
		"previousState", previousState,
		"newWorkflow", a.definition.Name,
		"newState", a.state,
		"framePhase", frame.Phase,
		"returnEvent", frame.ReturnEvent,
		"errorEvent", frame.ErrorEvent,
	)

	// 12. 验证返回事件在父工作流中是有效的
	returnEvent := Event(frame.ReturnEvent)
	if !result.Success {
		returnEvent = Event(frame.ErrorEvent)
	}

	if err := a.definition.ValidateEvent(a.state, returnEvent); err != nil {
		logger.Error("ReturnToParent: return event invalid in parent workflow",
			"sessionID", a.id,
			"parentWorkflow", a.definition.Name,
			"currentState", a.state,
			"returnEvent", returnEvent,
			"error", err,
			"availableEvents", a.definition.GetAvailableEvents(a.state),
		)
		// 不返回错误，仍然尝试发送事件
	}

	// 13. 记录日志
	logger.Info("ReturnToParent: completed transition",
		"sessionID", a.id,
		"childWorkflow", childWorkflow,
		"parentWorkflow", frame.Workflow,
		"parentPhase", frame.Phase,
		"success", result.Success,
		"duration", result.Duration,
	)

	// 14. 记录子工作流指标
	if a.subWorkflowMetrics != nil {
		if result.Success {
			a.subWorkflowMetrics.RecordComplete(childWorkflow, result.Duration)
		} else {
			reason := "unknown"
			if result.ErrorMsg != "" {
				reason = result.ErrorMsg
				if len(reason) > 50 {
					reason = reason[:50] // 截断过长的错误信息
				}
			}
			a.subWorkflowMetrics.RecordFailure(childWorkflow, reason, result.Duration)
		}
	}

	// 15. 发送回调事件到父工作流
	logger.Debug("ReturnToParent: sending return event",
		"sessionID", a.id,
		"event", returnEvent,
		"currentState", a.state,
		"currentWorkflow", a.definition.Name,
	)

	return a.Send(ctx, returnEvent, result)
}

// RollbackToParent 异常回滚到父工作流
// 用于子工作流执行过程中发生不可恢复的错误时
func (a *Actor) RollbackToParent(ctx context.Context, err error) error {
	result := NewSubWorkflowError(err)
	return a.ReturnToParent(ctx, result)
}

// ============================================================
// 子工作流信息获取方法
// ============================================================

// GetWorkflowStack 获取工作流调用栈
func (a *Actor) GetWorkflowStack() []session.WorkflowFrame {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return nil
	}
	return sess.WorkflowMeta.GetWorkflowStack()
}

// GetWorkflowDepth 获取当前工作流嵌套深度
func (a *Actor) GetWorkflowDepth() int {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return 0
	}
	return sess.WorkflowMeta.GetWorkflowDepth()
}

// IsSubWorkflow 判断当前是否在子工作流中
func (a *Actor) IsSubWorkflow() bool {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return false
	}
	return sess.WorkflowMeta.IsSubWorkflow()
}

// GetParentWorkflow 获取父工作流名称
func (a *Actor) GetParentWorkflow() string {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return ""
	}
	frame := sess.WorkflowMeta.PeekWorkflowFrame()
	if frame == nil {
		return ""
	}
	return frame.Workflow
}

// GetParentPhase 获取父工作流阶段
func (a *Actor) GetParentPhase() string {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return ""
	}
	frame := sess.WorkflowMeta.PeekWorkflowFrame()
	if frame == nil {
		return ""
	}
	return frame.Phase
}

// GetParallelSubWorkflowState 获取并行子工作流状态
func (a *Actor) GetParallelSubWorkflowState() *session.ParallelSubWorkflowState {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil || sess.WorkflowMeta.Extra == nil {
		return nil
	}
	if state, ok := sess.WorkflowMeta.Extra["parallel_subworkflow_state"].(*session.ParallelSubWorkflowState); ok {
		return state
	}
	return nil
}

// ============================================================
// 并行子工作流支持
// ============================================================// SpawnParallelSubWorkflows 启动多个子工作流
// 支持两种模式：
// 1. 串行模式（默认）：依次执行每个子工作流，前一个完成后自动启动下一个
// 2. 并行模式（需要独立 Session）：真正并行执行（当前版本不支持）
//
// 使用串行模式时，子工作流会依次执行，全部完成后触发 OnAllComplete 事件
func (a *Actor) SpawnParallelSubWorkflows(ctx context.Context, config ParallelSubWorkflowConfig) error {
	logger := a.Logger()

	if len(config.Configs) == 0 {
		return fmt.Errorf("no sub-workflows configured")
	}

	// 初始化并行状态
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return fmt.Errorf("session or workflow meta not found")
	}

	// 创建并行状态跟踪器
	parallelState := &session.ParallelSubWorkflowState{
		TotalCount:    len(config.Configs),
		MergeStrategy: session.MergeStrategy(config.MergeStrategy),
		Results:       make([]session.ParallelSubWorkflowResult, len(config.Configs)),
		OnAllComplete: string(config.OnAllComplete),
		OnAnyComplete: string(config.OnAnyComplete),
		OnAllFailed:   string(config.OnAllFailed),
	}
	sess.WorkflowMeta.SetParallelState(parallelState)

	// 保存配置到 session 供后续使用
	sess.Data["_parallel_configs"] = config.Configs
	sess.Data["_parallel_current_index"] = 0

	// 更新 session
	if _, err := a.SessionService().Update(ctx, a.id, func(s *session.Session) error {
		s.WorkflowMeta = sess.WorkflowMeta
		s.Data = sess.Data
		return nil
	}); err != nil {
		return fmt.Errorf("failed to save parallel state: %w", err)
	}

	logger.Info("Starting parallel sub-workflows (serial mode)",
		"totalCount", len(config.Configs),
		"mergeStrategy", config.MergeStrategy,
	)

	// 启动第一个子工作流
	return a.SpawnSubWorkflow(ctx, config.Configs[0])
}

// ContinueParallelSubWorkflows 继续执行下一个并行子工作流
// 在子工作流返回后调用，检查是否还有未执行的子工作流
// 返回值：
// - true: 还有子工作流需要执行，已自动启动
// - false: 所有子工作流已完成
func (a *Actor) ContinueParallelSubWorkflows(ctx context.Context, lastResult *SubWorkflowResult) (bool, error) {
	logger := a.Logger()
	sess := a.Session()

	if sess == nil || sess.WorkflowMeta == nil {
		return false, fmt.Errorf("session or workflow meta not found")
	}

	// 获取并行状态
	parallelState := sess.WorkflowMeta.GetParallelState()
	if parallelState == nil {
		return false, nil // 不是并行模式
	}

	// 获取当前索引和配置
	currentIndex, ok := sess.Data["_parallel_current_index"].(int)
	if !ok {
		return false, fmt.Errorf("parallel current index not found")
	}

	configs, ok := sess.Data["_parallel_configs"].([]SubWorkflowConfig)
	if !ok {
		return false, fmt.Errorf("parallel configs not found")
	}

	// 记录当前子工作流结果
	if currentIndex < len(parallelState.Results) {
		parallelState.Results[currentIndex] = session.ParallelSubWorkflowResult{
			WorkflowName: string(configs[currentIndex].WorkflowName),
			Success:      lastResult.Success,
			Output:       lastResult.Output,
			Error:        lastResult.ErrorMsg,
			Duration:     lastResult.Duration,
			CompletedAt:  time.Now(),
		}
		parallelState.CompletedCount++
		if lastResult.Success {
			parallelState.SuccessCount++
		} else {
			parallelState.FailedCount++
		}
	}

	// 检查合并策略
	switch parallelState.MergeStrategy {
	case session.MergeStrategyAny:
		// 任一成功即返回
		if lastResult.Success {
			logger.Info("Parallel sub-workflows: any strategy - first success",
				"completedIndex", currentIndex,
			)
			a.cleanupParallelState(ctx, sess)
			return false, nil
		}
	}

	// 移动到下一个
	nextIndex := currentIndex + 1
	sess.Data["_parallel_current_index"] = nextIndex

	// 更新并行状态
	sess.WorkflowMeta.SetParallelState(parallelState)

	// 检查是否还有更多子工作流
	if nextIndex < len(configs) {
		// 更新 session
		if _, err := a.SessionService().Update(ctx, a.id, func(s *session.Session) error {
			s.WorkflowMeta = sess.WorkflowMeta
			s.Data = sess.Data
			return nil
		}); err != nil {
			return false, fmt.Errorf("failed to update parallel state: %w", err)
		}

		logger.Info("Continuing to next parallel sub-workflow",
			"nextIndex", nextIndex,
			"total", len(configs),
		)

		// 启动下一个子工作流
		if err := a.SpawnSubWorkflow(ctx, configs[nextIndex]); err != nil {
			return false, err
		}
		return true, nil
	}

	// 所有子工作流完成
	logger.Info("All parallel sub-workflows completed",
		"total", parallelState.TotalCount,
		"success", parallelState.SuccessCount,
		"failed", parallelState.FailedCount,
	)

	// 清理并行状态
	a.cleanupParallelState(ctx, sess)

	return false, nil
}

// GetParallelState 获取当前并行子工作流状态
func (a *Actor) GetParallelState() *session.ParallelSubWorkflowState {
	sess := a.Session()
	if sess == nil || sess.WorkflowMeta == nil {
		return nil
	}
	return sess.WorkflowMeta.GetParallelState()
}

// IsInParallelMode 检查是否在并行子工作流模式中
func (a *Actor) IsInParallelMode() bool {
	return a.GetParallelState() != nil
}

// HandleParallelSubWorkflowComplete 处理并行子工作流完成
// 实现 SubWorkflowContext 接口
// 当一个子工作流完成时调用，会自动继续执行下一个或合并结果
func (a *Actor) HandleParallelSubWorkflowComplete(ctx context.Context, result *SubWorkflowResult) error {
	hasMore, err := a.ContinueParallelSubWorkflows(ctx, result)
	if err != nil {
		return err
	}
	if hasMore {
		return nil // 还有更多子工作流待执行
	}

	// 所有子工作流已完成，触发合并完成事件
	parallelState := a.GetParallelState()
	if parallelState != nil {
		// 根据结果触发对应事件
		var event Event
		if parallelState.FailedCount == parallelState.TotalCount {
			event = Event(parallelState.OnAllFailed)
		} else if parallelState.SuccessCount > 0 {
			event = Event(parallelState.OnAllComplete)
		} else {
			event = Event(parallelState.OnAnyComplete)
		}

		if event != "" {
			return a.Send(ctx, event, nil)
		}
	}
	return nil
}

// cleanupParallelState 清理并行状态
func (a *Actor) cleanupParallelState(ctx context.Context, sess *session.Session) {
	sess.WorkflowMeta.ClearParallelState()
	delete(sess.Data, "_parallel_configs")
	delete(sess.Data, "_parallel_current_index")

	// 更新 session
	_, _ = a.SessionService().Update(ctx, a.id, func(s *session.Session) error {
		s.WorkflowMeta = sess.WorkflowMeta
		s.Data = sess.Data
		return nil
	})
}
