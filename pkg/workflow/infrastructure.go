// Package workflow 提供完整的工作流基础设施
//
// # 架构设计
//
// Infrastructure 是一个"组合器"（Composer），它将多个独立组件组合成完整的工作流系统：
//
//	Infrastructure =
//	    Hub (WebSocket 连接管理)
//	  + SessionService (会话管理)
//	  + Bridge (WebSocket ↔ Session 桥接)
//	  + System (FSM Actor 管理)
//	  + ServiceRegistry (业务服务注入)
//	  + Metrics (指标收集)
//
// 各组件的职责：
//   - Hub: 管理 WebSocket 连接，分组广播
//   - SessionService: 管理 session 生命周期和持久化
//   - Bridge: 将 WebSocket 客户端绑定到 session，自动广播更新
//   - System: 管理 FSM Actor 生命周期，路由工作流事件
//   - ServiceRegistry: 注册业务服务，供 Workflow 使用
//   - Metrics: 收集工作流指标（状态转换、事件等）
//
// # 为什么需要 Bridge？
//
// Bridge 虽然被 Infrastructure 包含，但它有独立存在的价值：
//  1. 单一职责：Bridge 只负责"WebSocket ↔ Session"的绑定逻辑
//  2. 可独立使用：不需要 FSM 的系统可以只用 Bridge
//  3. 解耦设计：Infrastructure 可以替换 Bridge 实现
//
// # 使用示例
//
// 完整的工作流系统（推荐）：
//
//	infra := workflow.NewDefaultInfrastructure(logger)
//
//	// 注册业务服务
//	infra.GetServiceRegistry().Register("myService", myService)
//
//	// 注册工作流定义
//	engine.Register(&engine.WorkflowDefinition{...})
//
//	// HTTP 路由
//	http.HandleFunc("/ws", infra.HandleWebSocket)
//
//	// 触发工作流事件
//	infra.SendEvent(ctx, sessionID, "START", payload)
//
// 只使用 Bridge（不需要 FSM）：
//
//	bridge := infra.GetBridge()
//	bridge.BroadcastToSession(sessionID, "notification", data)
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/workflow/wsbridge"
	"jusha/mcp/pkg/ws"
)

// IWorkflowInfrastructure 工作流基础设施接口
type IWorkflowInfrastructure interface {
	// HandleWebSocket WebSocket HTTP 处理器
	HandleWebSocket(w http.ResponseWriter, r *http.Request)

	// GetSessionService 获取会话服务（业务层使用）
	GetSessionService() session.ISessionService

	// GetBridge 获取桥接（用于手动广播）
	GetBridge() wsbridge.IBridge

	// GetServiceRegistry 获取服务注册表（注册业务服务）
	GetServiceRegistry() engine.IServiceRegistry

	// SendEvent 发送工作流事件到指定 session
	// 这是触发状态转换的核心方法
	SendEvent(ctx context.Context, sessionID string, event engine.Event, payload any) error

	// With 依赖注入单个组件
	With(component any) IWorkflowInfrastructure
}

// Infrastructure 完整的工作流基础设施
// 包含：WebSocket + Session + Bridge + ServiceRegistry + System
type Infrastructure struct {
	// 核心组件
	Hub            ws.IHub
	SessionService session.ISessionService
	Bridge         wsbridge.IBridge
	WSServer       ws.IWSServer

	// 工作流组件
	ServiceRegistry engine.IServiceRegistry // 业务服务注册表
	Metrics         engine.Metrics          // 指标收集
	System          *engine.System          // Actor 系统

	// 日志
	logger logging.ILogger
}

// NewDefaultInfrastructure 一键创建完整的默认基础设施
// 这是最简单的使用方式，内部自动创建并连接所有组件
//
// 使用示例：
//
//	infra := workflow.NewDefaultInfrastructure(logger)
//
//	// 1. 注册业务服务
//	infra.GetServiceRegistry().Register("intentService", myIntentService)
//	infra.GetServiceRegistry().Register("rosteringService", myRosteringService)
//
//	// 2. 注册工作流定义
//	engine.Register(&engine.WorkflowDefinition{
//		Name: "rostering",
//		InitialState: "created",
//		Transitions: []engine.Transition{...},
//	})
//
//	// 3. 创建 session 时设置工作流
//	sess := session.NewSession(orgID, userID, "rostering")
//	sess.WorkflowMeta = &session.WorkflowMeta{
//		Workflow: "rostering",
//		InstanceID: fmt.Sprintf("fsm-%s", sess.ID),
//	}
//
//	// 4. 触发工作流事件
//	infra.SendEvent(ctx, sess.ID, "START", payload)
//
//	// 5. 注册 HTTP 路由
//	http.HandleFunc("/ws", infra.HandleWebSocket)
//	http.ListenAndServe(":8080", nil)
func NewDefaultInfrastructure(logger logging.ILogger, configurator config.IServiceConfigurator, wsOpts ...ws.ServerOption) IWorkflowInfrastructure {
	// 1. 创建 Hub（连接管理）
	hub := ws.NewDefaultHub()

	// 2. 创建 SessionService（会话管理）
	sessionService := session.NewDefaultSessionService(logger)

	// 3. 创建 Bridge（连接 WebSocket 和 Session）
	bridge := wsbridge.NewDefaultBridge(hub, sessionService, logger)

	// 4. 创建 ServiceRegistry（业务服务注册）
	serviceRegistry := engine.NewServiceRegistry()

	// 5. 创建 Metrics（指标收集）
	metrics := engine.NewMetrics()

	// 6. 创建 Context（工作流上下文）
	workflowContext := engine.NewWorkflowContext(
		"infrastructure", // 基础设施级别的 context ID
		logger,
		configurator,
		sessionService,
		serviceRegistry,
		metrics,
	)

	// 7. 创建 System（Actor 系统）
	system := engine.NewSystem(workflowContext, metrics)

	// 8. 将 Bridge 设置为 EventPublisher（连接 workflow 事件和 WebSocket 广播）
	system.SetEventPublisher(bridge)

	// 9. 将 Infrastructure 设置为 EventSender（让 Bridge 能够触发工作流事件）
	// 注意：这里需要先创建 Infrastructure 再设置，使用闭包延迟绑定
	var infra *Infrastructure
	bridge.SetEventSender(&infrastructureEventSender{getInfra: func() IWorkflowInfrastructure { return infra }})

	// 10. 设置 SessionService 更新回调（session 更新时自动广播）
	sessionService.SetOnUpdate(func(sess *session.Session) {
		// 广播 session_updated 消息（包含完整 session 数据）
		_ = bridge.BroadcastToSession(sess.ID, "session_updated", sess)
	})

	// 11. 设置 WorkflowMeta 更新回调（只广播 WorkflowMeta,避免重复广播）
	sessionService.SetOnWorkflowMetaUpdate(func(sessionID string, meta *session.WorkflowMeta) {
		// 广播 workflow_meta_updated 消息（只包含 WorkflowMeta）
		_ = bridge.BroadcastToSession(sessionID, "workflow_meta_updated", map[string]any{
			"sessionId":    sessionID,
			"workflowMeta": meta,
		})
	})

	// 12. 创建 WebSocket Server（HTTP 升级）
	// 添加默认的消息处理器
	finalOpts := append([]ws.ServerOption{
		ws.WithMessageHandler(defaultMessageHandler(bridge, logger)),
	}, wsOpts...)

	wsServer := ws.NewDefaultServer(logger, finalOpts...)

	infra = &Infrastructure{
		Hub:             bridge.GetHub(),
		SessionService:  sessionService,
		Bridge:          bridge,
		WSServer:        wsServer,
		ServiceRegistry: serviceRegistry,
		Metrics:         metrics,
		System:          system,
		logger:          logger,
	}

	return infra
}

// infrastructureEventSender Infrastructure 的 EventSender 适配器
// 用于让 Bridge 能够回调 Infrastructure 的 SendEvent 方法
type infrastructureEventSender struct {
	getInfra func() IWorkflowInfrastructure
}

func (s *infrastructureEventSender) SendEvent(ctx context.Context, sessionID string, event engine.Event, payload any) error {
	return s.getInfra().SendEvent(ctx, sessionID, event, payload)
}

// HandleWebSocket WebSocket HTTP 处理器
// 直接用于 http.HandleFunc
func (i *Infrastructure) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	i.WSServer.HandleWS(w, r)
}

// GetSessionService 获取会话服务（业务层使用）
func (i *Infrastructure) GetSessionService() session.ISessionService {
	return i.SessionService
}

// GetBridge 获取桥接（用于手动广播）
func (i *Infrastructure) GetBridge() wsbridge.IBridge {
	return i.Bridge
}

// GetServiceRegistry 获取服务注册表
// 业务层使用此方法注册业务服务，然后在 Workflow 中通过 Context.Services() 访问
func (i *Infrastructure) GetServiceRegistry() engine.IServiceRegistry {
	return i.ServiceRegistry
}

// ==================== 依赖注入方法 (With Pattern) ====================

// With 依赖注入单个组件（自动识别接口类型）
// 一个组件可能同时实现多个接口，会自动匹配所有支持的接口并注入
//
// 支持的接口类型：
//   - session.IIntentRecognizer: 意图识别器
//   - wsbridge.IIntentEventMapper: 意图到事件的映射器
//   - session.IStore: 会话存储
//
// 使用示例：
//
//	infra.With(myComponent)  // 自动识别并注入
func (i *Infrastructure) With(component any) IWorkflowInfrastructure {
	if component == nil {
		return i
	}

	injected := false

	// 检查是否实现 IWorkflowInitializer 接口
	if initializer, ok := component.(session.IWorkflowInitializer); ok {
		i.WithWorkflowInitializer(initializer)
		injected = true
	}

	// 检查是否实现 ICommandMapper 接口
	if mapper, ok := component.(wsbridge.ICommandMapper); ok {
		i.WithCommandMapper(mapper)
		injected = true
	}

	// 检查是否实现 IIntentRecognizer 接口
	if recognizer, ok := component.(session.IIntentRecognizer); ok {
		i.WithIntentRecognizer(recognizer)
		injected = true
	}

	// 检查是否实现 IIntentEventMapper 接口
	if mapper, ok := component.(wsbridge.IIntentEventMapper); ok {
		i.WithIntentEventMapper(mapper)
		injected = true
	}

	// 检查是否实现 IStore 接口
	if store, ok := component.(session.IStore); ok {
		i.WithSessionStore(store)
		injected = true
	}

	// 如果没有匹配任何接口，记录警告
	if !injected {
		i.logger.Warn("component does not implement any supported interface",
			"componentType", fmt.Sprintf("%T", component),
		)
	}

	return i
}

// WithIntentRecognizer 注入意图识别器到 SessionService
// 用于识别用户输入的意图
//
// 使用示例：
//
//	infra.WithIntentRecognizer(myRecognizer)
func (i *Infrastructure) WithIntentRecognizer(recognizer session.IIntentRecognizer) IWorkflowInfrastructure {
	i.SessionService.WithIntentRecognizer(recognizer)
	return i
}

// WithWorkflowInitializer 注入工作流初始化器到 SessionService
// 用于在创建 session 时根据 agentType 自动初始化 WorkflowMeta
//
// 使用示例：
//
//	infra.WithWorkflowInitializer(myInitializer)
func (i *Infrastructure) WithWorkflowInitializer(initializer session.IWorkflowInitializer) IWorkflowInfrastructure {
	i.SessionService.WithWorkflowInitializer(initializer)
	return i
}

// WithCommandMapper 注入命令映射器到 Bridge
// 用于将前端命令映射到工作流事件
//
// 使用示例：
//
//	infra.WithCommandMapper(myCommandMapper)
func (i *Infrastructure) WithCommandMapper(mapper wsbridge.ICommandMapper) IWorkflowInfrastructure {
	i.Bridge.SetCommandMapper(mapper)
	return i
}

// WithIntentEventMapper 注入意图到事件的映射器到 Bridge
// 用于将识别的意图映射到工作流事件
//
// 使用示例：
//
//	infra.WithIntentEventMapper(myMapper)
func (i *Infrastructure) WithIntentEventMapper(mapper wsbridge.IIntentEventMapper) IWorkflowInfrastructure {
	i.Bridge.SetIntentEventMapper(mapper)
	return i
}

// WithSessionStore 替换 SessionService 的存储实现
// 用于自定义会话存储（如 Redis、数据库等）
//
// 使用示例：
//
//	infra.WithSessionStore(myRedisStore)
func (i *Infrastructure) WithSessionStore(store session.IStore) IWorkflowInfrastructure {
	i.SessionService.WithStore(store)
	return i
}

// SendEvent 发送工作流事件到指定 session
// 这是触发状态转换的核心方法
//
// 使用示例：
//
//	infra.SendEvent(ctx, sessionID, "CREATE_SCHEDULE", payload)
//
// 注意：需要先使用 engine.Register() 注册工作流定义
func (i *Infrastructure) SendEvent(ctx context.Context, sessionID string, event engine.Event, payload any) error {
	// 从 session 获取工作流类型
	sess, err := i.SessionService.Get(ctx, sessionID)
	if err != nil {
		i.logger.Error("session not found", "sessionID", sessionID, "error", err)
		return err
	}

	// 从 WorkflowMeta 中获取工作流名称
	if sess.WorkflowMeta == nil || sess.WorkflowMeta.Workflow == "" {
		i.logger.Error("workflow not set in session", "sessionID", sessionID)
		return errors.NewConfigurationError("workflow not set in session", nil)
	}

	workflowName := engine.Workflow(sess.WorkflowMeta.Workflow)

	// 如果 Phase 为空，自动初始化为工作流的 InitialState
	currentState := sess.WorkflowMeta.Phase
	if currentState == "" {
		// 获取工作流定义
		workflowDef := engine.Get(workflowName)
		if workflowDef == nil {
			i.logger.Error("workflow not registered", "workflow", workflowName)
			return errors.NewConfigurationError("workflow not registered", nil)
		}

		initialState := workflowDef.InitialState
		if initialState == "" {
			i.logger.Error("workflow has no initial state", "workflow", workflowName)
			return errors.NewConfigurationError("workflow has no initial state", nil)
		}

		// 更新 session 的 Phase 为初始状态
		currentState = initialState
		_, err := i.SessionService.Update(ctx, sessionID, func(s *session.Session) error {
			if s.WorkflowMeta != nil {
				s.WorkflowMeta.Phase = initialState
			}
			return nil
		})
		if err != nil {
			i.logger.Error("failed to initialize workflow phase", "error", err)
			return err
		}

	}

	// 注意：不在这里验证事件有效性
	// 原因：
	// 1. 在子工作流切换时，Session 中的 WorkflowMeta 和 Actor 的实际状态可能短暂不一致
	// 2. Actor 的 handle() 方法会使用正确的当前状态进行验证
	// 3. 避免重复验证逻辑，让 Actor 作为事件验证的单一权威来源
	//
	// 如果需要调试，可以在日志中查看事件发送情况
	i.logger.Debug("sending event to workflow",
		"sessionID", sessionID,
		"workflow", workflowName,
		"sessionPhase", currentState,
		"event", event,
	)

	// 转发到 System（Actor 会负责实际的事件验证和状态转换）
	return i.System.SendEvent(ctx, sessionID, workflowName, event, payload)
}

// defaultMessageHandler 默认的 WebSocket 消息处理器
// 自动处理 session 绑定和基础消息
func defaultMessageHandler(bridge wsbridge.IBridge, logger logging.ILogger) ws.MessageHandler {
	return func(client *ws.Client, data []byte) error {
		// 解析基础消息格式
		var msg struct {
			Type      string `json:"type"`
			SessionID string `json:"sessionId"`
		}

		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Warn("invalid message format", "error", err)
			return err
		}

		// 自动处理 session 绑定
		if msg.Type == "bind" && msg.SessionID != "" {
			if err := bridge.BindSession(client, msg.SessionID); err != nil {
				logger.Error("failed to bind session", "sessionID", msg.SessionID, "error", err)
				return err
			}
			logger.Info("session bound", "sessionID", msg.SessionID)
			return nil
		}

		// 将消息交给 Bridge 处理（处理 Session 管理消息）
		// Bridge 会处理 get_recent_session, create_session 等消息
		// 对于不认识的消息类型，Bridge.HandleMessage 会返回 nil
		if err := bridge.HandleMessage(client, data); err != nil {
			logger.Error("bridge message handler failed", "error", err)
			return err
		}

		return nil
	}
}
