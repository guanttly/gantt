// Package engine 提供工作流引擎核心组件
package engine

import "jusha/mcp/pkg/workflow/session"

// ============================================================
// 通用状态定义
// 命名规范: _common_[state]_ 或 _[state]_（省略 common）
// 适用于所有工作流的标准终态和常见状态
// ============================================================

const (
	// 通用终态
	StateCompleted State = "_completed_" // 已完成
	StateFailed    State = "_failed_"    // 失败
	StateCancelled State = "_cancelled_" // 已取消
	StateRejected  State = "_rejected_"  // 已拒绝
	StateAborted   State = "_aborted_"   // 已中止

	// 通用初始状态
	StateCreated State = "_created_" // 已创建
	StateIdle    State = "_idle_"    // 空闲

	// 通用处理状态
	StateProcessing State = "_processing_" // 处理中
	StateWaiting    State = "_waiting_"    // 等待输入
	StatePending    State = "_pending_"    // 待处理
)

// ============================================================
// 通用事件定义
// 命名规范: _common_[event]_ 或 _[event]_（省略 common）
// 适用于所有工作流的标准事件
// ============================================================

const (
	// 生命周期事件
	EventStart    Event = "_start_"    // 启动
	EventComplete Event = "_complete_" // 完成
	EventCancel   Event = "_cancel_"   // 取消
	EventRetry    Event = "_retry_"    // 重试

	// 确认操作事件
	EventConfirm Event = "_confirm_" // 确认
	EventModify  Event = "_modify_"  // 修改
	EventReject  Event = "_reject_"  // 拒绝

	// 审批操作事件
	EventApprove Event = "_approve_" // 审批通过
	EventDeny    Event = "_deny_"    // 审批拒绝
	EventSubmit  Event = "_submit_"  // 提交审批

	// 错误处理事件
	EventError   Event = "_error_"   // 错误
	EventTimeout Event = "_timeout_" // 超时
	EventAbort   Event = "_abort_"   // 中止
)

// ============================================================
// 辅助函数
// ============================================================

// IsTerminalState 判断是否为终态
func IsTerminalState(state State) bool {
	switch state {
	case StateCompleted, StateFailed, StateCancelled, StateRejected, StateAborted:
		return true
	default:
		return false
	}
}

// GetEventStyleHint 根据事件推断按钮样式
func GetEventStyleHint(event Event) session.WorkflowActionStyle {
	// 1. 精确匹配通用事件
	switch event {
	case EventConfirm, EventApprove, EventComplete:
		return session.ActionStylePrimary
	case EventReject, EventCancel, EventDeny:
		return session.ActionStyleWarning
	case EventRetry:
		return session.ActionStyleInfo
	case EventError, EventAbort:
		return session.ActionStyleDanger
	}

	// 默认
	return session.ActionStyleSecondary
}
