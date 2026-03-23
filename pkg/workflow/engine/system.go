// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"jusha/mcp/pkg/errors"
)

var (
	registry = make(map[Workflow]*WorkflowDefinition)
	mu       sync.RWMutex
)

// Register 注册工作流定义
func Register(def *WorkflowDefinition) {
	if def == nil || def.Name == "" {
		panic("invalid workflow definition")
	}
	mu.Lock()
	defer mu.Unlock()
	registry[def.Name] = def
}

// Get 获取工作流定义
func Get(name Workflow) *WorkflowDefinition {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// System 管理多个 session actor 生命周期
type System struct {
	context            Context                 // 工作流上下文（包含 Logger、ServiceRegistry）
	actors             map[string]*Actor       // sessionID -> Actor
	metrics            Metrics                 // 指标收集
	subWorkflowMetrics *SubWorkflowMetrics     // 子工作流指标
	eventPublisher     IWorkflowEventPublisher // 事件发布器（可选）
	mu                 sync.RWMutex
	stopJanitor        chan struct{}
}

// NewSystem 创建 ActorSystem
func NewSystem(context Context, metrics Metrics) *System {
	// 初始化子工作流指标（复用 FSMMetrics 的 Prometheus Registry）
	var registry *prometheus.Registry
	if fsmMetrics, ok := metrics.(*FSMMetrics); ok && fsmMetrics != nil {
		registry = fsmMetrics.Registry()
	}

	s := &System{
		context:            context,
		actors:             make(map[string]*Actor),
		metrics:            metrics,
		subWorkflowMetrics: NewSubWorkflowMetrics(registry),
		stopJanitor:        make(chan struct{}),
	}
	go s.janitorLoop()
	return s
}

// SetEventPublisher 设置事件发布器（可选）
func (s *System) SetEventPublisher(publisher IWorkflowEventPublisher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventPublisher = publisher
}

// getActor 获取或创建 Actor
func (s *System) getActor(id string, workflowName Workflow) *Actor {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已存在，直接返回
	if a, ok := s.actors[id]; ok {
		return a
	}

	// 获取工作流定义
	definition := Get(workflowName)
	if definition == nil {
		if s.context != nil && s.context.Logger() != nil {
			s.context.Logger().Error("workflow definition not found", "workflow", workflowName)
		}
		return nil
	}

	// 尝试从 Session 恢复状态
	initialState := definition.InitialState
	if s.context != nil && s.context.SessionService() != nil {
		sess, err := s.context.SessionService().Get(context.Background(), id)
		if err == nil && sess != nil && sess.WorkflowMeta != nil {
			phase := sess.WorkflowMeta.Phase
			if phase != "" {
				// 验证状态是否在工作流定义中有效
				if s.isValidState(definition, State(phase)) {
					initialState = State(phase)
				
				} else {
					if s.context.Logger() != nil {
						s.context.Logger().Warn("invalid state in session, using initial state",
							"sessionID", id,
							"workflow", workflowName,
							"sessionState", phase,
							"initialState", initialState)
					}
				}
			}
		}
	}

	// 创建新 Actor
	a := newActor(id, definition, s.context, s.metrics, s)
	
	// 如果恢复的状态不是 InitialState，更新 Actor 的状态
	if initialState != definition.InitialState {
		a.mu.Lock()
		// 更新指标（减少旧状态，增加新状态）
		if a.metrics != nil {
			a.metrics.DecStateActive(a.state)
			a.metrics.IncStateActive(initialState)
		}
		a.state = initialState
		a.mu.Unlock()
	}
	
	a.Start() // 启动事件循环

	s.actors[id] = a
	return a
}

// SendEvent 发送事件到指定 session 的 actor
func (s *System) SendEvent(ctx context.Context, sessionID string, workflowName Workflow, ev Event, payload any) error {
	actor := s.getActor(sessionID, workflowName)
	if actor == nil {
		return errors.NewNotFoundError("actor not found for session", nil)
	}

	err := actor.Send(ctx, ev, payload)

	// 如果 actor 已到达终态，主动回收
	if actor.IsTerminal() {
		s.evictActor(sessionID)
	}

	return err
}

// MetricsSnapshot 返回当前 FSM 指标快照
func (s *System) MetricsSnapshot() MetricSnapshot {
	if s.metrics == nil {
		return MetricSnapshot{}
	}
	return s.metrics.Snapshot()
}

// PrometheusRegistry 暴露内部 Prometheus 注册表
func (s *System) PrometheusRegistry() *prometheus.Registry {
	if s.metrics == nil {
		return nil
	}
	// 尝试断言到 *FSMMetrics
	if fsmMetrics, ok := s.metrics.(*FSMMetrics); ok {
		return fsmMetrics.Registry()
	}
	return nil
}

// QueryDecisions 查询指定 session 的决策日志
func (s *System) QueryDecisions(sessionID string, opts DecisionQueryOptions) (list []Decision, total int) {
	s.mu.RLock()
	actor, ok := s.actors[sessionID]
	s.mu.RUnlock()

	if !ok || actor == nil {
		return nil, 0
	}
	return actor.QueryDecisions(opts)
}

// janitorLoop 定期清理到达终态或长时间无事件的 actors
func (s *System) janitorLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopJanitor:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, a := range s.actors {
				if a == nil {
					delete(s.actors, id)
					continue
				}
				if a.IsTerminal() {
					a.Stop()
					delete(s.actors, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// evictActor 立即回收指定 session 的 actor
func (s *System) evictActor(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.actors[id]; ok {
		a.Stop()
		delete(s.actors, id)
	}
}

// Stop 停止系统，关闭所有 actor 和后台协程
func (s *System) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止清理协程
	close(s.stopJanitor)

	// 停止所有 actor
	for id, a := range s.actors {
		if a != nil {
			a.Stop()
		}
		delete(s.actors, id)
	}
}

// isValidState 验证状态是否在工作流定义中有效
func (s *System) isValidState(definition *WorkflowDefinition, state State) bool {
	// 检查是否是初始状态
	if state == definition.InitialState {
		return true
	}

	// 检查是否在转换表中（作为 From 或 To）
	for _, tr := range definition.Transitions {
		if tr.From == state || tr.To == state {
			return true
		}
	}

	return false
}
