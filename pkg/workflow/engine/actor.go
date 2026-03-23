// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"fmt"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"
	"runtime"
	"sync"
	"time"
)

// message 内部消息类型
type message struct {
	event   Event
	payload any
	ctx     context.Context
}

// Actor 管理单个 session 的事件循环
// 这是 FSM（有限状态机）的核心执行单元
type Actor struct {
	id    string
	inbox chan message
	stop  chan struct{}
	state State
	trans map[State]map[Event]Transition
	clock func() time.Time

	system *System // 回指 system

	startedAt time.Time
	decisions []Decision
	mu        sync.RWMutex

	terminal bool // 是否已到达终态

	// 工作流切换标志：当 SpawnSubWorkflow 或 ReturnToParent 被调用时设置
	// 用于防止 Action 执行后的状态覆盖
	workflowSwitched bool

	// 通用依赖（通过 Context 访问）
	context            Context             // 工作流上下文
	definition         *WorkflowDefinition // 工作流定义
	metrics            Metrics             // 指标收集
	subWorkflowMetrics *SubWorkflowMetrics // 子工作流指标
} // newActor 创建新的 Actor（内部使用）
func newActor(id string, definition *WorkflowDefinition, context Context, metrics Metrics, system *System) *Actor {
	// 从 System 获取子工作流指标
	var subWfMetrics *SubWorkflowMetrics
	if system != nil {
		subWfMetrics = system.subWorkflowMetrics
	}

	a := &Actor{
		id:                 id,
		inbox:              make(chan message, 32),
		stop:               make(chan struct{}),
		trans:              make(map[State]map[Event]Transition),
		clock:              time.Now,
		startedAt:          time.Now(),
		context:            context,
		definition:         definition,
		state:              definition.InitialState,
		metrics:            metrics,
		subWorkflowMetrics: subWfMetrics,
		system:             system,
	}

	a.initTransitions()
	if a.metrics != nil {
		a.metrics.IncStateActive(a.state)
	}

	return a
}

// initTransitions 初始化状态转换表
func (a *Actor) initTransitions() {
	// 执行初始化钩子
	if a.definition.OnActorInit != nil {
		_ = a.definition.OnActorInit(a)
	}

	// 清空旧的转换表，防止工作流切换时残留旧的转换
	a.trans = make(map[State]map[Event]Transition)

	for _, tr := range a.definition.Transitions {
		if a.trans[tr.From] == nil {
			a.trans[tr.From] = make(map[Event]Transition)
		}
		a.trans[tr.From][tr.Event] = tr
	}
}

// loop 事件循环
func (a *Actor) loop() {
	for {
		select {
		case <-a.stop:
			return
		case msg := <-a.inbox:
			a.handle(msg)
		}
	}
}

// handle 处理单个事件消息
func (a *Actor) handle(msg message) {
	// 调试日志：追踪事件处理入口
	if a.context != nil && a.context.Logger() != nil {
		a.context.Logger().Debug("actor handle: received event",
			"sessionID", a.id,
			"workflow", a.definition.Name,
			"currentState", a.state,
			"event", msg.event,
			"workflowSwitched", a.workflowSwitched,
		)
	}

	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("%v", r)
			if a.context != nil && a.context.Logger() != nil {
				// 获取panic堆栈信息
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stackTrace := string(buf[:n])

				a.context.Logger().Error("fsm panic",
					"error", errorMsg,
					"state", a.state,
					"event", msg.event,
					"stackTrace", stackTrace,
				)
			}

			// 通知前端发生了错误
			if a.system != nil && a.system.eventPublisher != nil {
				_ = a.system.eventPublisher.PublishWorkflowFailed(
					a.id,
					"workflow execution failed",
					errorMsg,
				)
			}
		}
	}()

	// 验证事件在当前状态下是否有效
	if err := a.definition.ValidateEvent(a.state, msg.event); err != nil {
		if a.context != nil && a.context.Logger() != nil {
			// 获取当前状态的有效事件列表
			availableEvents := a.definition.GetAvailableEvents(a.state)
			a.context.Logger().Warn("actor handle: invalid event for current state",
				"sessionID", a.id,
				"workflow", a.definition.Name,
				"currentState", a.state,
				"event", msg.event,
				"availableEvents", availableEvents,
				"error", err,
			)
		}
		return
	}

	// 查找转换
	trs := a.trans[a.state]
	tr, ok := trs[msg.event]
	if !ok {
		// 没有找到对应的转换，记录警告并忽略
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Warn("actor handle: no transition found in trans table",
				"state", a.state,
				"event", msg.event,
				"workflow", a.definition.Name,
				"transTableKeys", getTransTableKeys(a.trans),
			)
		}
		return
	}

	// Guard 检查
	if tr.Guard != nil && !tr.Guard(a.context, msg.payload) {
		return
	}

	// 执行前置副作用（BeforeAct）
	var beforeActErr error
	if tr.BeforeAct != nil {
		beforeActErr = tr.BeforeAct(msg.ctx, a, msg.payload)
	}

	// 在执行 Act 前，先清除旧的 actions
	// Act 中可以设置新的 actions，如果不设置则保持清空状态
	if a.context != nil && a.context.SessionService() != nil {
		_, _ = a.context.SessionService().UpdateWorkflowMeta(msg.ctx, a.id, func(meta *session.WorkflowMeta) error {
			if meta != nil {
				meta.Actions = nil
			}
			return nil
		})
	}

	// 执行副作用（Action）
	var actionErr error
	if tr.Act != nil {
		// 重置工作流切换标志
		a.workflowSwitched = false
		// 传递 Actor 本身而不是 a.context，因为 Actor 实现了 Context 接口
		// 这样 Action 中的 wctx.ID() 会返回 sessionID，wctx.Session() 会返回正确的 session
		actionErr = tr.Act(msg.ctx, a, msg.payload)
	}

	// 状态转换（无论 Action 是否成功，状态转换都要记录）
	// 但如果 Action 中已经切换了工作流（SpawnSubWorkflow/ReturnToParent），则跳过后续所有逻辑
	prev := a.state
	if a.workflowSwitched {
		// 工作流已切换，跳过后续状态更新和消息发送
		// 子工作流会处理自己的状态和消息
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Debug("actor handle: workflow switched in Act, skipping remaining logic",
				"sessionID", a.id,
				"event", msg.event,
				"newWorkflow", a.definition.Name,
				"newState", a.state,
			)
		}
		return
	}
	a.state = tr.To

	// 同步更新 Session 的 WorkflowMeta.Phase
	if a.context != nil && a.context.SessionService() != nil {
		_, _ = a.context.SessionService().UpdateWorkflowMeta(msg.ctx, a.id, func(meta *session.WorkflowMeta) error {
			if meta != nil {
				meta.Phase = a.state
			}
			return nil
		})
	}

	// 如果配置了状态描述，立即发送系统消息显示当前阶段（在进入状态时立即发送，不等流程完成）
	if tr.StateLabel != "" && a.context != nil && a.context.SessionService() != nil {
		_, _ = a.context.SessionService().AddSystemMessage(msg.ctx, a.id, tr.StateLabel)
	}

	// 执行状态转换后的副作用（AfterAct）
	var afterActErr error
	if tr.AfterAct != nil {
		afterActErr = tr.AfterAct(msg.ctx, a, msg.payload)
	}

	// 如果 AfterAct 中切换了工作流（SpawnSubWorkflow/ReturnToParent），跳过后续所有逻辑
	if a.workflowSwitched {
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Debug("actor handle: workflow switched in AfterAct, skipping remaining logic",
				"sessionID", a.id,
				"event", msg.event,
				"newWorkflow", a.definition.Name,
				"newState", a.state,
			)
		}
		return
	}

	// StateLabel 消息已在状态转换时立即发送，这里不再重复发送

	// 如果 WorkflowMeta 有 Description，添加为助手消息（用于承载交互按钮）
	if a.context != nil && a.context.SessionService() != nil {
		sess, err := a.context.SessionService().Get(msg.ctx, a.id)
		if err == nil && sess != nil && sess.WorkflowMeta != nil && sess.WorkflowMeta.Description != "" {
			// 添加 Description 为助手消息
			_, _ = a.context.SessionService().AddAssistantMessage(msg.ctx, a.id, sess.WorkflowMeta.Description)

			// 清空 Description，避免下次状态转换时重复添加
			_, _ = a.context.SessionService().UpdateWorkflowMeta(msg.ctx, a.id, func(meta *session.WorkflowMeta) error {
				if meta != nil {
					meta.Description = ""
				}
				return nil
			})
		}
	}

	// 如果 BeforeAct 执行失败，记录错误消息
	if beforeActErr != nil {
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Error("before-act execution failed",
				"state", a.state,
				"event", msg.event,
				"error", beforeActErr,
			)
		}

		// 添加错误消息到 Session.Messages（作为助手消息）
		if a.context != nil && a.context.SessionService() != nil {
			errorMsg := fmt.Sprintf("抱歉，处理过程中遇到错误：%s", beforeActErr.Error())
			_, _ = a.context.SessionService().AddAssistantMessage(msg.ctx, a.id, errorMsg)
		}

		// 通知前端更新（触发 session_updated 广播）
		if a.system != nil && a.system.eventPublisher != nil {
			_ = a.system.eventPublisher.PublishWorkflowFailed(
				a.id,
				"before-act execution failed",
				beforeActErr.Error(),
			)
		}
	}

	// 如果 Action 执行失败，记录错误消息到对话历史
	if actionErr != nil {
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Error("action execution failed",
				"state", a.state,
				"event", msg.event,
				"error", actionErr,
			)
		}

		// 添加带重试按钮的错误消息到 Session.Messages
		if a.context != nil && a.context.SessionService() != nil {
			errorMsg := fmt.Sprintf("❌ 处理过程中产生错误：%s\n\n请点击重试按钮重新尝试。", actionErr.Error())

			// 转换 payload 为 map[string]any（如果需要）
			var payloadMap map[string]any
			if msg.payload != nil {
				if m, ok := msg.payload.(map[string]any); ok {
					payloadMap = m
				} else {
					// 如果 payload 不是 map，创建一个新的 map
					payloadMap = make(map[string]any)
					payloadMap["data"] = msg.payload
				}
			}

			// 构建重试按钮，重试时会重新触发导致失败的事件
			retryActions := []session.WorkflowAction{
				{
					ID:      "retry",
					Type:    session.ActionTypeWorkflow,
					Label:   "🔄 重试",
					Event:   session.WorkflowEvent(msg.event), // 重新触发导致失败的事件
					Style:   session.ActionStylePrimary,
					Payload: payloadMap, // 传递原始 payload
				},
			}

			_, _ = a.context.SessionService().AddAssistantMessageWithActions(msg.ctx, a.id, errorMsg, retryActions)
		}

		// 通知前端更新（触发 session_updated 广播）
		if a.system != nil && a.system.eventPublisher != nil {
			_ = a.system.eventPublisher.PublishWorkflowFailed(
				a.id,
				"action execution failed",
				actionErr.Error(),
			)
		}
	}

	// 如果 AfterAct 执行失败，记录错误消息
	if afterActErr != nil {
		if a.context != nil && a.context.Logger() != nil {
			a.context.Logger().Error("after-act execution failed",
				"state", a.state,
				"event", msg.event,
				"error", afterActErr,
			)
		}

		// 添加带重试按钮的错误消息到 Session.Messages
		if a.context != nil && a.context.SessionService() != nil {
			errorMsg := fmt.Sprintf("❌ 处理过程中产生错误：%s\n\n请点击重试按钮重新尝试。", afterActErr.Error())

			// 转换 payload 为 map[string]any（如果需要）
			var payloadMap map[string]any
			if msg.payload != nil {
				if m, ok := msg.payload.(map[string]any); ok {
					payloadMap = m
				} else {
					// 如果 payload 不是 map，创建一个新的 map
					payloadMap = make(map[string]any)
					payloadMap["data"] = msg.payload
				}
			}

			// 构建重试按钮，重试时会重新触发导致失败的事件
			retryActions := []session.WorkflowAction{
				{
					ID:      "retry",
					Type:    session.ActionTypeWorkflow,
					Label:   "🔄 重试",
					Event:   session.WorkflowEvent(msg.event), // 重新触发导致失败的事件
					Style:   session.ActionStylePrimary,
					Payload: payloadMap, // 传递原始 payload
				},
			}

			_, _ = a.context.SessionService().AddAssistantMessageWithActions(msg.ctx, a.id, errorMsg, retryActions)
		}

		// 通知前端更新（触发 session_updated 广播）
		if a.system != nil && a.system.eventPublisher != nil {
			_ = a.system.eventPublisher.PublishWorkflowFailed(
				a.id,
				"after-act execution failed",
				afterActErr.Error(),
			)
		}
	}

	// 记录决策
	dec := Decision{
		At:    a.clock(),
		From:  prev,
		Event: msg.event,
		To:    a.state,
	}

	a.mu.Lock()
	a.decisions = append(a.decisions, dec)
	// 决策日志内存上限（保留最近 1000 条）
	if len(a.decisions) > 1000 {
		a.decisions = a.decisions[len(a.decisions)-1000:]
	}
	a.mu.Unlock()

	// 指标记录
	if a.metrics != nil {
		a.metrics.Record(prev, msg.event, a.state)
	}

	// 事件发布器：决策事件
	if a.system != nil && a.system.eventPublisher != nil {
		_ = a.system.eventPublisher.PublishDecision(a.id, dec)
	}

	// 事件发布器：状态变更事件
	if a.system != nil && a.system.eventPublisher != nil {
		_ = a.system.eventPublisher.PublishStateChange(a.id, prev, a.state, msg.event)
	}

	// 终态检查与处理
	if a.state == State("completed") || a.state == State("failed") {
		a.terminal = true
		if a.definition.OnTerminal != nil {
			a.definition.OnTerminal(a, dec)
		}

		// 发布工作流完成/失败事件
		if a.system != nil && a.system.eventPublisher != nil {
			if a.state == State("completed") {
				_ = a.system.eventPublisher.PublishWorkflowCompleted(a.id, a.context)
			} else {
				_ = a.system.eventPublisher.PublishWorkflowFailed(a.id, "workflow failed", "state reached: failed")
			}
		}
	}
}

// Send 发送事件到 Actor
func (a *Actor) Send(ctx context.Context, ev Event, payload any) error {
	if a.context != nil && a.context.Logger() != nil {
		a.context.Logger().Debug("actor Send: sending event to inbox",
			"sessionID", a.id,
			"workflow", a.definition.Name,
			"currentState", a.state,
			"event", ev,
		)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case a.inbox <- message{event: ev, payload: payload, ctx: ctx}:
		return nil
	}
}

// Stop 停止 actor 循环
func (a *Actor) Stop() {
	select {
	case <-a.stop:
		return
	default:
		close(a.stop)
		// 指标活跃计数修正
		if a.metrics != nil {
			a.metrics.DecStateActive(a.state)
		}
	}
}

// IsTerminal 返回是否到达终态
func (a *Actor) IsTerminal() bool {
	return a.terminal
}

// GetState 获取当前状态
func (a *Actor) GetState() State {
	return a.state
}

// GetID 获取 Actor ID
func (a *Actor) GetID() string {
	return a.id
}

// Context interface implementation（让 Actor 实现 Context 接口）
func (a *Actor) ID() string {
	return a.id
}

func (a *Actor) Logger() logging.ILogger {
	if a.context != nil {
		return a.context.Logger()
	}
	return nil
}

func (a *Actor) Services() IServiceRegistry {
	if a.context != nil {
		return a.context.Services()
	}
	return nil
}

func (a *Actor) Metrics() Metrics {
	return a.metrics
}

func (a *Actor) Now() time.Time {
	return a.clock()
}

func (a *Actor) SessionService() session.ISessionService {
	if a.context != nil {
		return a.context.SessionService()
	}
	return nil
}

func (a *Actor) Session() *session.Session {
	// 使用 Actor 自己的 ID (sessionID) 而不是 context 的 ID
	if a.context != nil && a.context.SessionService() != nil {
		sess, _ := a.context.SessionService().Get(context.Background(), a.id)
		return sess
	}
	return nil
}

func (a *Actor) GetMetadata(key string) (any, bool) {
	if a.context != nil {
		return a.context.GetMetadata(key)
	}
	return nil, false
}

func (a *Actor) SetMetadata(key string, value any) {
	if a.context != nil {
		a.context.SetMetadata(key, value)
	}
}

func (a *Actor) GetRetries() int {
	return 5 // 默认重试次数
}

// QueryDecisions 查询决策日志
func (a *Actor) QueryDecisions(opts DecisionQueryOptions) (list []Decision, total int) {
	a.mu.RLock()
	decisions := make([]Decision, len(a.decisions))
	copy(decisions, a.decisions)
	a.mu.RUnlock()

	// 过滤
	filtered := make([]Decision, 0, len(decisions))
	for _, d := range decisions {
		if opts.Event != "" && d.Event != opts.Event {
			continue
		}
		if opts.From != "" && d.From != opts.From {
			continue
		}
		if opts.To != "" && d.To != opts.To {
			continue
		}
		if opts.Since != nil && d.At.Before(*opts.Since) {
			continue
		}
		filtered = append(filtered, d)
	}
	total = len(filtered)

	// 排序
	if opts.Reverse {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	// Offset
	if opts.Offset > 0 && opts.Offset < len(filtered) {
		filtered = filtered[opts.Offset:]
	} else if opts.Offset >= len(filtered) {
		filtered = []Decision{}
	}

	// Limit
	lim := opts.Limit
	if lim <= 0 {
		lim = 50
	}
	if lim > 200 {
		lim = 200
	}
	if len(filtered) > lim {
		filtered = filtered[:lim]
	}
	return filtered, total
}

// Start 启动 Actor 事件循环
func (a *Actor) Start() {
	go a.loop()
}

// getTransTableKeys 获取转换表中的所有状态和事件（用于调试）
func getTransTableKeys(trans map[State]map[Event]Transition) map[string][]string {
	result := make(map[string][]string)
	for state, events := range trans {
		eventList := make([]string, 0, len(events))
		for event := range events {
			eventList = append(eventList, string(event))
		}
		result[string(state)] = eventList
	}
	return result
}
