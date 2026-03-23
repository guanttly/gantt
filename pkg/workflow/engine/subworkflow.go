// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"time"

	"jusha/mcp/pkg/workflow/session"
)

// ============================================================
// 子工作流配置和结果类型
// ============================================================

// DefaultSubWorkflowTimeout 默认子工作流超时时间（5分钟）
// 设置为 0 表示无限等待
const DefaultSubWorkflowTimeout = 5 * time.Minute

// SubWorkflowConfig 子工作流调用配置
type SubWorkflowConfig struct {
	// 子工作流标识
	WorkflowName Workflow `json:"workflowName"` // 子工作流名称

	// 输入数据
	Input any `json:"input,omitempty"` // 传递给子工作流的输入数据（支持强类型结构体或map[string]any）

	// 返回事件配置
	OnComplete Event `json:"onComplete"` // 子工作流成功完成后触发的事件
	OnError    Event `json:"onError"`    // 子工作流失败后触发的事件

	// 超时配置
	Timeout time.Duration `json:"timeout"` // 超时时间，0 表示无限等待

	// 快照配置（用于异常回滚）
	SnapshotKeys []string `json:"snapshotKeys,omitempty"` // 需要快照的 Session.Data key 列表
}

// SubWorkflowResult 子工作流执行结果
type SubWorkflowResult struct {
	Success  bool           `json:"success"`          // 是否成功
	Output   map[string]any `json:"output,omitempty"` // 输出数据
	Error    error          `json:"-"`                // 错误（不序列化）
	ErrorMsg string         `json:"error,omitempty"`  // 错误信息
	Duration time.Duration  `json:"duration"`         // 执行时长
}

// NewSubWorkflowResult 创建成功的子工作流结果
func NewSubWorkflowResult(output map[string]any) *SubWorkflowResult {
	return &SubWorkflowResult{
		Success: true,
		Output:  output,
	}
}

// NewSubWorkflowError 创建失败的子工作流结果
func NewSubWorkflowError(err error) *SubWorkflowResult {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return &SubWorkflowResult{
		Success:  false,
		Error:    err,
		ErrorMsg: errMsg,
	}
}

// ============================================================
// 并行子工作流配置
// ============================================================

// ParallelSubWorkflowConfig 并行子工作流调用配置
type ParallelSubWorkflowConfig struct {
	// 子工作流列表
	Configs []SubWorkflowConfig `json:"configs"`

	// 合并策略
	MergeStrategy string `json:"mergeStrategy"` // "all", "any", "custom"

	// 返回事件
	OnAllComplete Event `json:"onAllComplete"` // 全部完成后触发的事件
	OnAnyComplete Event `json:"onAnyComplete"` // 任一成功后触发的事件（仅 MergeStrategyAny 使用）
	OnAllFailed   Event `json:"onAllFailed"`   // 全部失败后触发的事件

	// 自定义合并函数（仅 MergeStrategyCustom 使用）
	MergeFunc func(results []session.ParallelSubWorkflowResult) *SubWorkflowResult `json:"-"`
}

// ============================================================
// 子工作流上下文接口扩展
// ============================================================

// SubWorkflowContext 子工作流上下文接口
// 扩展 Context 接口，添加子工作流相关方法
type SubWorkflowContext interface {
	Context

	// 子工作流调用
	SpawnSubWorkflow(ctx context.Context, config SubWorkflowConfig) error
	SpawnParallelSubWorkflows(ctx context.Context, config ParallelSubWorkflowConfig) error

	// 返回父工作流
	ReturnToParent(ctx context.Context, result *SubWorkflowResult) error

	// 回滚到父工作流（错误时使用）
	RollbackToParent(ctx context.Context, err error) error

	// 工作流栈信息
	GetWorkflowStack() []session.WorkflowFrame
	GetWorkflowDepth() int
	IsSubWorkflow() bool

	// 获取父工作流信息
	GetParentWorkflow() string
	GetParentPhase() string

	// 并行子工作流管理
	HandleParallelSubWorkflowComplete(ctx context.Context, result *SubWorkflowResult) error
	GetParallelSubWorkflowState() *session.ParallelSubWorkflowState
} // ============================================================
// 子工作流内部事件
// ============================================================

// 子工作流内部使用的事件
const (
	// EventSubWorkflowStart 子工作流启动事件
	EventSubWorkflowStart Event = "_subworkflow_start_"

	// EventSubWorkflowComplete 子工作流完成事件（内部使用）
	EventSubWorkflowComplete Event = "_subworkflow_complete_"

	// EventSubWorkflowFailed 子工作流失败事件（内部使用）
	EventSubWorkflowFailed Event = "_subworkflow_failed_"

	// EventSubWorkflowTimeout 子工作流超时事件（内部使用）
	EventSubWorkflowTimeout Event = "_subworkflow_timeout_"
)

// ============================================================
// 子工作流数据 Key
// ============================================================

// 子工作流相关的 Session.Data key
const (
	// DataKeySubWorkflowInput 子工作流输入数据
	DataKeySubWorkflowInput = "_subworkflow_input_"

	// DataKeySubWorkflowResult 子工作流结果数据
	DataKeySubWorkflowResult = "_subworkflow_result_"

	// DataKeySubWorkflowStartTime 子工作流开始时间
	DataKeySubWorkflowStartTime = "_subworkflow_start_time_"
)

// ============================================================
// 子工作流工具函数
// ============================================================

// GetSubWorkflowInput 从 session 获取子工作流输入
func GetSubWorkflowInput(sess *session.Session) map[string]any {
	if sess.Data == nil {
		return nil
	}
	if input, ok := sess.Data[DataKeySubWorkflowInput].(map[string]any); ok {
		return input
	}
	return nil
}

// SetSubWorkflowInput 设置子工作流输入到 session
func SetSubWorkflowInput(ctx context.Context, sessService session.ISessionService, sessionID string, input map[string]any) error {
	_, err := sessService.SetData(ctx, sessionID, DataKeySubWorkflowInput, input)
	return err
}

// GetSubWorkflowResult 从 session 获取子工作流结果
func GetSubWorkflowResult(sess *session.Session) *SubWorkflowResult {
	if sess.Data == nil {
		return nil
	}
	if result, ok := sess.Data[DataKeySubWorkflowResult].(*SubWorkflowResult); ok {
		return result
	}
	return nil
}

// SetSubWorkflowResult 设置子工作流结果到 session
func SetSubWorkflowResult(ctx context.Context, sessService session.ISessionService, sessionID string, result *SubWorkflowResult) error {
	_, err := sessService.SetData(ctx, sessionID, DataKeySubWorkflowResult, result)
	return err
}

// ClearSubWorkflowData 清理子工作流相关数据
func ClearSubWorkflowData(ctx context.Context, sessService session.ISessionService, sessionID string) error {
	sess, err := sessService.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	// 删除子工作流相关的 key
	delete(sess.Data, DataKeySubWorkflowInput)
	delete(sess.Data, DataKeySubWorkflowResult)
	delete(sess.Data, DataKeySubWorkflowStartTime)

	_, err = sessService.Update(ctx, sessionID, func(s *session.Session) error {
		s.Data = sess.Data
		return nil
	})
	return err
}

// ============================================================
// 快照工具函数
// ============================================================

// CreateDataSnapshot 创建 Session.Data 的快照
func CreateDataSnapshot(sess *session.Session, keys []string) map[string]any {
	if sess.Data == nil || len(keys) == 0 {
		return nil
	}

	snapshot := make(map[string]any)
	for _, key := range keys {
		if value, ok := sess.Data[key]; ok {
			// 注意：这里是浅拷贝，对于复杂对象需要深拷贝
			snapshot[key] = value
		}
	}
	return snapshot
}

// RestoreDataSnapshot 从快照恢复 Session.Data
func RestoreDataSnapshot(ctx context.Context, sessService session.ISessionService, sessionID string, snapshot map[string]any) error {
	if snapshot == nil {
		return nil
	}

	_, err := sessService.Update(ctx, sessionID, func(sess *session.Session) error {
		if sess.Data == nil {
			sess.Data = make(map[string]any)
		}
		for key, value := range snapshot {
			sess.Data[key] = value
		}
		return nil
	})
	return err
}

// ============================================================
// 子工作流返回辅助函数（统一调用模式）
// ============================================================

// ReturnSuccess 从子工作流成功返回父工作流
// 这是推荐的统一返回方式，替代直接调用 actor.ReturnToParent()
//
// 使用示例：
//
//	return engine.ReturnSuccess(ctx, wctx, map[string]any{
//		"result_key": resultValue,
//	})
func ReturnSuccess(ctx context.Context, wctx Context, output map[string]any) error {
	actor, ok := wctx.(*Actor)
	if !ok {
		logger := wctx.Logger()
		if logger != nil {
			logger.Warn("ReturnSuccess: context is not an Actor, cannot return to parent workflow",
				"sessionID", wctx.ID(),
			)
		}
		// 非子工作流模式，静默返回
		return nil
	}

	// 检查是否在子工作流中
	if !actor.IsSubWorkflow() {
		logger := wctx.Logger()
		if logger != nil {
			logger.Debug("ReturnSuccess: not in sub-workflow, skip return",
				"sessionID", wctx.ID(),
			)
		}
		return nil
	}

	result := NewSubWorkflowResult(output)
	return actor.ReturnToParent(ctx, result)
}

// ReturnError 从子工作流失败返回父工作流
// 这是推荐的统一返回方式，替代直接调用 actor.ReturnToParent()
//
// 使用示例：
//
//	return engine.ReturnError(ctx, wctx, fmt.Errorf("保存失败: %s", errMsg))
func ReturnError(ctx context.Context, wctx Context, err error) error {
	actor, ok := wctx.(*Actor)
	if !ok {
		logger := wctx.Logger()
		if logger != nil {
			logger.Warn("ReturnError: context is not an Actor, cannot return to parent workflow",
				"sessionID", wctx.ID(),
				"error", err,
			)
		}
		// 非子工作流模式，返回原始错误
		return err
	}

	// 检查是否在子工作流中
	if !actor.IsSubWorkflow() {
		logger := wctx.Logger()
		if logger != nil {
			logger.Debug("ReturnError: not in sub-workflow, returning original error",
				"sessionID", wctx.ID(),
				"error", err,
			)
		}
		return err
	}

	result := NewSubWorkflowError(err)
	return actor.ReturnToParent(ctx, result)
}

// ReturnCancelled 从子工作流返回父工作流（用户取消）
// 这是推荐的统一返回方式
//
// 使用示例：
//
//	return engine.ReturnCancelled(ctx, wctx, "用户主动取消")
func ReturnCancelled(ctx context.Context, wctx Context, reason string) error {
	if reason == "" {
		reason = "user cancelled"
	}
	return ReturnError(ctx, wctx, &CancelledError{Reason: reason})
}

// CancelledError 取消错误类型
type CancelledError struct {
	Reason string
}

func (e *CancelledError) Error() string {
	return e.Reason
}

// IsCancelled 检查是否是取消错误
func IsCancelled(err error) bool {
	_, ok := err.(*CancelledError)
	return ok
}

// ReturnWithOutput 从子工作流返回父工作流，带有自定义结果
// 用于需要同时传递成功状态和详细输出的场景
//
// 使用示例：
//
//	return engine.ReturnWithOutput(ctx, wctx, true, map[string]any{
//		"saved_count": 10,
//		"skipped": false,
//	})
func ReturnWithOutput(ctx context.Context, wctx Context, success bool, output map[string]any) error {
	actor, ok := wctx.(*Actor)
	if !ok {
		logger := wctx.Logger()
		if logger != nil {
			logger.Warn("ReturnWithOutput: context is not an Actor",
				"sessionID", wctx.ID(),
			)
		}
		return nil
	}

	if !actor.IsSubWorkflow() {
		return nil
	}

	result := &SubWorkflowResult{
		Success: success,
		Output:  output,
	}
	return actor.ReturnToParent(ctx, result)
}

// MustReturnToParent 强制返回父工作流，如果不在子工作流中则返回错误
// 用于必须是子工作流的场景
//
// 使用示例：
//
//	return engine.MustReturnToParent(ctx, wctx, engine.NewSubWorkflowResult(output))
func MustReturnToParent(ctx context.Context, wctx Context, result *SubWorkflowResult) error {
	actor, ok := wctx.(*Actor)
	if !ok {
		return session.ErrNotInSubWorkflow
	}
	return actor.ReturnToParent(ctx, result)
}
