// Package engine 提供通用的工作流引擎核心组件
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"
)

// Event 工作流事件
// 使用 session.WorkflowEvent 作为底层类型，保持类型统一
type Event = session.WorkflowEvent

// State 工作流状态
// 使用 session.WorkflowState 作为底层类型，保持类型统一
type State = session.WorkflowState

// ActionStyle 行为样式
// 使用 WorkflowActionStyle 作为底层类型，保持类型统一
type ActionStyle = session.WorkflowActionStyle

// ActionStyle 行为类型
// 使用 WorkflowActionType 作为底层类型，保持类型统一
type ActionType = session.WorkflowActionType

// Workflow 工作流类型标识
type Workflow string

// Context 工作流执行上下文
// 为业务 Action 提供访问能力
type Context interface {
	// 基础信息
	ID() string
	Logger() logging.ILogger
	Now() time.Time

	// 事件发送
	Send(ctx context.Context, event Event, payload any) error

	// 会话访问（通用）
	Session() *session.Session
	SessionService() session.ISessionService

	// 业务服务访问（可插拔）
	Services() IServiceRegistry

	// 指标
	Metrics() Metrics

	// 元数据（扩展字段）
	GetMetadata(key string) (any, bool)
	SetMetadata(key string, value any)
}

// ServiceRegistry 业务服务注册表
// 用于注入和访问业务特定服务
type IServiceRegistry interface {
	Register(key string, service any)
	Get(key string) (any, bool)
	MustGet(key string) any
}

// serviceRegistry 默认实现
type serviceRegistry struct {
	services sync.Map
}

// NewServiceRegistry 创建服务注册表
func NewServiceRegistry() IServiceRegistry {
	return &serviceRegistry{}
}

func (r *serviceRegistry) Register(key string, service any) {
	r.services.Store(key, service)
}

func (r *serviceRegistry) Get(key string) (any, bool) {
	return r.services.Load(key)
}

func (r *serviceRegistry) MustGet(key string) any {
	if svc, ok := r.Get(key); ok {
		return svc
	}
	panic(fmt.Sprintf("service not found: %s", key))
}

// Transition 状态转换定义
type Transition struct {
	From       State
	Event      Event
	To         State
	StateLabel string                                    // 目标状态的中文描述（可选）
	Guard      func(Context, any) bool                   // 可选守卫条件
	BeforeAct  func(context.Context, Context, any) error // 转换副作用（状态转换前、Act之前执行）
	Act        func(context.Context, Context, any) error // 转换副作用（状态转换前执行）
	AfterAct   func(context.Context, Context, any) error // 转换副作用（状态转换后执行）
}

// Decision 状态转换记录
type Decision struct {
	At        time.Time `json:"at"`
	From      State     `json:"from"`
	Event     Event     `json:"event"`
	To        State     `json:"to"`
	Info      string    `json:"info,omitempty"`
	DraftVer  int64     `json:"draftVersion,omitempty"`
	ResultVer int64     `json:"resultVersion,omitempty"`
}

// WorkflowDefinition 工作流定义
type WorkflowDefinition struct {
	Name         Workflow
	InitialState State
	Transitions  []Transition
	OnActorInit  func(ctx Context) error         // 可选初始化钩子
	OnTerminal   func(ctx Context, dec Decision) // 可选终态钩子

	// 子工作流支持
	IsSubWorkflow      bool                                           // 标记是否为子工作流
	OnSubWorkflowEnter func(ctx Context, parentWorkflow string) error // 进入子工作流时的钩子
	OnSubWorkflowExit  func(ctx Context, success bool) error          // 退出子工作流时的钩子
}

// ValidateEvent 验证事件在当前状态下是否有效
func (def *WorkflowDefinition) ValidateEvent(currentState State, event Event) error {
	for _, tr := range def.Transitions {
		if tr.From == currentState && tr.Event == event {
			return nil // 找到有效转换
		}
	}
	return fmt.Errorf("invalid event '%s' for state '%s' in workflow '%s'", event, currentState, def.Name)
}

// GetAvailableEvents 获取当前状态下可用的事件列表
func (def *WorkflowDefinition) GetAvailableEvents(currentState State) []Event {
	events := make([]Event, 0)
	seen := make(map[Event]bool)

	for _, tr := range def.Transitions {
		if tr.From == currentState {
			if !seen[tr.Event] {
				events = append(events, tr.Event)
				seen[tr.Event] = true
			}
		}
	}

	return events
}

// GetNextState 获取在当前状态下触发事件后的目标状态
func (def *WorkflowDefinition) GetNextState(currentState State, event Event) (State, bool) {
	for _, tr := range def.Transitions {
		if tr.From == currentState && tr.Event == event {
			return tr.To, true
		}
	}
	return "", false
}

// WorkflowEventPublisher 工作流事件发布器（可选）
// 用于发布工作流执行过程中的各种事件
type IWorkflowEventPublisher interface {
	// 发布决策事件
	PublishDecision(sessionID string, dec Decision) error

	// 发布状态变更事件
	PublishStateChange(sessionID string, from State, to State, event Event) error

	// 发布工作流完成事件
	PublishWorkflowCompleted(sessionID string, ctx Context) error

	// 发布工作流失败事件
	PublishWorkflowFailed(sessionID string, reason string, errorMsg string) error
}

// DecisionQueryOptions 决策查询选项
type DecisionQueryOptions struct {
	Event   Event
	From    State
	To      State
	Since   *time.Time
	Offset  int
	Limit   int
	Reverse bool
}

// Metrics 指标接口
type Metrics interface {
	// 状态转换记录
	Record(from State, event Event, to State)

	// 活跃状态计数
	IncStateActive(state State)
	DecStateActive(state State)

	// 获取快照
	Snapshot() MetricSnapshot
}

// MetricSnapshot 指标快照
type MetricSnapshot struct {
	StateActive    map[State]int64  `json:"stateActive"`
	TransitionHist map[string]int64 `json:"transitionHist"`
	TotalEvents    int64            `json:"totalEvents"`
}
