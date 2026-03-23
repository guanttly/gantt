// Package engine 提供工作流引擎核心组件
package engine

import (
	"jusha/mcp/pkg/workflow/session"
)

// WorkflowHelper 工作流辅助函数
type WorkflowHelper struct {
	definition *WorkflowDefinition
}

// NewWorkflowHelper 创建工作流辅助器
func NewWorkflowHelper(workflowName Workflow) *WorkflowHelper {
	def := Get(workflowName)
	return &WorkflowHelper{
		definition: def,
	}
}

// GetAvailableActions 获取当前状态下可用的操作
// 可以直接用于前端显示按钮
func (h *WorkflowHelper) GetAvailableActions(currentState State) []session.WorkflowAction {
	if h.definition == nil {
		return nil
	}

	events := h.definition.GetAvailableEvents(currentState)
	actions := make([]session.WorkflowAction, 0, len(events))

	for _, event := range events {
		// 获取目标状态用于显示
		nextState, _ := h.definition.GetNextState(currentState, event)

		actions = append(actions, session.WorkflowAction{
			ID:    string(event),
			Type:  session.ActionTypeWorkflow,
			Label: h.getEventLabel(event),
			Event: event, // 直接使用类型安全的 Event
			Style: h.getEventStyle(event, nextState),
		})
	}

	return actions
}

// getEventLabel 获取事件的显示标签
// 业务层应该在 Action 中设置具体的 Label，这里只是降级处理
func (h *WorkflowHelper) getEventLabel(event Event) string {
	// 直接返回事件名，业务层负责设置友好的标签
	return string(event)
}

// getEventStyle 获取事件的按钮样式
// 只处理通用情况，业务层可以在 Action 定义中覆盖
func (h *WorkflowHelper) getEventStyle(event Event, nextState State) session.WorkflowActionStyle {
	// 1. 根据目标状态判断（仅处理通用终态）
	if nextState == StateCompleted {
		return session.ActionStyleSuccess
	}
	if nextState == StateFailed || nextState == StateCancelled || nextState == StateRejected {
		return session.ActionStyleDanger
	}

	// 2. 根据事件名称推断样式（通用事件）
	return GetEventStyleHint(event)
}

// ValidateTransition 验证状态转换是否有效
func (h *WorkflowHelper) ValidateTransition(from State, event Event) error {
	if h.definition == nil {
		return nil // 没有定义，跳过验证
	}
	return h.definition.ValidateEvent(from, event)
}

// GetAllStates 获取工作流中定义的所有状态
func (h *WorkflowHelper) GetAllStates() []State {
	if h.definition == nil {
		return nil
	}

	stateSet := make(map[State]bool)
	stateSet[h.definition.InitialState] = true

	for _, tr := range h.definition.Transitions {
		stateSet[tr.From] = true
		stateSet[tr.To] = true
	}

	states := make([]State, 0, len(stateSet))
	for state := range stateSet {
		states = append(states, state)
	}

	return states
}

// GetAllEvents 获取工作流中定义的所有事件
func (h *WorkflowHelper) GetAllEvents() []Event {
	if h.definition == nil {
		return nil
	}

	eventSet := make(map[Event]bool)
	for _, tr := range h.definition.Transitions {
		eventSet[tr.Event] = true
	}

	events := make([]Event, 0, len(eventSet))
	for event := range eventSet {
		events = append(events, event)
	}

	return events
}

// IsTerminalState 判断是否为终态
func (h *WorkflowHelper) IsTerminalState(state State) bool {
	if h.definition == nil {
		return false
	}

	// 终态定义：没有任何出边的状态
	for _, tr := range h.definition.Transitions {
		if tr.From == state {
			return false // 有出边，不是终态
		}
	}

	return true
}
