// Package wsbridge 提供 workflow 和 ws 的集成桥接
// 将 WebSocket 连接与工作流会话关联
//
// # 设计说明
//
// Bridge 是一个轻量级的桥接层，专注于"WebSocket 客户端"和"Workflow Session"的关联管理。
// 它的职责是：
//   - 将 WebSocket 连接绑定到特定 session（按 sessionID 分组）
//   - 向 session 的所有客户端广播消息
//   - 响应 session 更新事件
//
// # 使用场景
//
//  1. **通过 Infrastructure 使用（推荐）**：
//     大多数情况下，应该使用 workflow.Infrastructure，它内部集成了 Bridge
//
//     infra := workflow.NewDefaultInfrastructure(logger)
//     bridge := infra.GetBridge()  // 访问底层 bridge
//
//  2. **独立使用（特殊场景）**：
//     如果你的系统只需要 WebSocket + Session（不需要 FSM 工作流），可以单独使用 Bridge
//
//     hub := ws.NewDefaultHub()
//     sessionService := session.NewDefaultSessionService(logger)
//     bridge := wsbridge.NewDefaultBridge(hub, sessionService, logger)
//
// # 与 Infrastructure 的关系
//
//   - Bridge 是 Infrastructure 的组成部分（依赖关系）
//   - Infrastructure = Hub + SessionService + Bridge + System + ServiceRegistry
//   - Bridge 可以独立存在，但 Infrastructure 必须包含 Bridge
package wsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/ws"
)

// 消息类型常量
const (
	// Client -> Server (Session Management)
	MsgTypeGetRecentSession = "get_recent_session"
	MsgTypeCreateSession    = "create_session"

	// Client -> Server (Workflow)
	MsgTypeUserMessage      = "user_message"
	MsgTypeWorkflowCommand  = "workflow_command"
	MsgTypeLoadConversation = "load_conversation" // 加载历史对话

	// Server -> Client
	MsgTypeSessionSnapshot  = "session_snapshot"
	MsgTypeSessionCreated   = "session_created"
	MsgTypeAssistantMessage = "assistant_message"
	MsgTypeSystemMessage    = "system_message"
	MsgTypeError            = "error"
)

// IEventSender 事件发送接口
// 用于解耦 Bridge 和 Infrastructure
type IEventSender interface {
	SendEvent(ctx context.Context, sessionID string, event engine.Event, payload any) error
}

// ICommandMapper 命令映射器接口（业务层实现）
// 业务系统需要实现此接口来定义前端命令如何映射到工作流事件
type ICommandMapper interface {
	// MapCommandToEvent 将前端命令映射为工作流事件
	// command: 前端发送的命令字符串（如 "_start_"）
	// 返回对应的工作流事件，如果无法映射则返回空字符串
	MapCommandToEvent(command string) engine.Event
}

// IntentWorkflowMapping 意图到工作流的映射结果
type IntentWorkflowMapping struct {
	WorkflowName string       // 工作流名称（如 "schedule.create"）
	Event        engine.Event // 触发事件
	Implemented  bool         // 是否已实现
}

// IIntentEventMapper 意图到事件的映射接口
// 业务系统需要实现此接口来定义意图如何映射到工作流事件
type IIntentEventMapper interface {
	// MapIntentToEvent 将意图类型映射为工作流事件
	// intentType: 意图类型（如 "schedule.create"）
	// 返回对应的工作流事件，如果无法映射则返回空字符串
	MapIntentToEvent(intentType string) engine.Event

	// MapIntentToWorkflow 将意图类型映射为完整的工作流信息
	// intentType: 意图类型（如 "schedule.create"）
	// 返回包含工作流名称、事件和实现状态的映射结果，如果无法映射则返回 nil
	MapIntentToWorkflow(intentType string) *IntentWorkflowMapping
}

// IConversationLoader 对话加载器接口
// 业务系统需要实现此接口来加载历史对话
type IConversationLoader interface {
	// LoadConversation 加载指定对话到当前 session
	LoadConversation(ctx context.Context, sessionID, conversationID string) error
}

// IBridge workflow 和 WebSocket 的桥接接口
type IBridge interface {
	// BindSession 将客户端绑定到 session
	BindSession(client *ws.Client, sessionID string) error

	// BroadcastToSession 向指定 session 的所有客户端广播消息
	BroadcastToSession(sessionID string, messageType string, data any) error

	// HandleMessage 处理客户端消息（Session 管理相关）
	HandleMessage(client *ws.Client, data []byte) error

	// SetEventSender 设置事件发送器（用于处理工作流事件）
	SetEventSender(sender IEventSender)

	// SetCommandMapper 设置命令映射器
	SetCommandMapper(mapper ICommandMapper)

	// SetIntentEventMapper 设置意图到事件的映射器
	SetIntentEventMapper(mapper IIntentEventMapper)

	// SetConversationLoader 设置对话加载器（用于加载历史对话）
	SetConversationLoader(loader IConversationLoader)

	// GetHub 获取 WebSocket Hub
	GetHub() ws.IHub

	// GetSessionClients 获取指定 session 的所有客户端
	GetSessionClients(sessionID string) []*ws.Client

	// GetSessionClientCount 获取指定 session 的客户端数量
	GetSessionClientCount(sessionID string) int

	// ==================== WorkflowEventPublisher 接口实现 ====================

	// PublishDecision 发布决策事件
	PublishDecision(sessionID string, dec engine.Decision) error

	// PublishStateChange 发布状态变更事件
	PublishStateChange(sessionID string, from engine.State, to engine.State, event engine.Event) error

	// PublishWorkflowCompleted 发布工作流完成事件
	PublishWorkflowCompleted(sessionID string, ctx engine.Context) error

	// PublishWorkflowFailed 发布工作流失败事件
	PublishWorkflowFailed(sessionID string, reason string, errorMsg string) error
}

// Bridge workflow 和 WebSocket 的桥接层默认实现
// 实现 IBridge 接口
type Bridge struct {
	hub                ws.IHub
	sessionService     session.ISessionService
	eventSender        IEventSender        // 事件发送器（用于触发工作流）
	commandMapper      ICommandMapper      // 命令到事件的映射器（业务实现）
	intentEventMapper  IIntentEventMapper  // 意图到事件的映射器（业务实现）
	conversationLoader IConversationLoader // 对话加载器（业务实现）
	logger             logging.ILogger
}

// NewBridge 创建桥接层
// 接受接口类型参数，允许使用自定义实现
func newBridge(hub ws.IHub, sessionService session.ISessionService, logger logging.ILogger) *Bridge {
	b := &Bridge{
		hub:            hub,
		sessionService: sessionService,
		logger:         logger,
	}

	return b
}

// SetEventSender 设置事件发送器
func (b *Bridge) SetEventSender(sender IEventSender) {
	b.eventSender = sender
}

// SetCommandMapper 设置命令映射器
func (b *Bridge) SetCommandMapper(mapper ICommandMapper) {
	b.commandMapper = mapper
}

// SetIntentEventMapper 设置意图到事件的映射器
func (b *Bridge) SetIntentEventMapper(mapper IIntentEventMapper) {
	b.intentEventMapper = mapper
}

// SetConversationLoader 设置对话加载器
func (b *Bridge) SetConversationLoader(loader IConversationLoader) {
	b.conversationLoader = loader
}

// HandleMessage 处理客户端消息（Session 管理相关）
// 此方法已集成到 Infrastructure 的 defaultMessageHandler 中
// 支持的消息类型：
//   - get_recent_session: 获取最近的会话
//   - create_session: 创建新会话
//
// 对于不认识的消息类型，返回 nil（由其他处理器处理）
func (b *Bridge) HandleMessage(client *ws.Client, data []byte) error {
	var msg struct {
		Type string         `json:"type"`
		Data map[string]any `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		b.logger.Error("failed to unmarshal message", "error", err)
		return b.sendError(client, "invalid message format")
	}

	ctx := client.Context()

	// 处理消息
	switch msg.Type {
	case MsgTypeGetRecentSession:
		return b.handleGetRecentSession(ctx, client, msg.Data)
	case MsgTypeCreateSession:
		return b.handleCreateSession(ctx, client, msg.Data)
	case MsgTypeUserMessage:
		return b.handleUserMessage(ctx, client, msg.Data)
	case MsgTypeWorkflowCommand:
		return b.handleWorkflowCommand(ctx, client, msg.Data)
	case MsgTypeLoadConversation:
		return b.handleLoadConversation(ctx, client, msg.Data)
	default:
		// 其他消息类型不处理，返回 nil（可能由其他处理器处理）
		return nil
	}
}

// BindSession 将客户端绑定到 session
// 客户端会被加入以 sessionID 为 key 的分组
func (b *Bridge) BindSession(client *ws.Client, sessionID string) error {
	if sessionID == "" {
		return errors.NewInvalidArgumentError("sessionID is empty")
	}

	// 验证 session 是否存在
	_, err := b.sessionService.Get(context.Background(), sessionID)
	if err != nil {
		return errors.WrapInfrastructureError(err, "session not found")
	}

	// 注册到 hub
	b.hub.Register(client, sessionID)
	client.SetID(sessionID)

	return nil
}

// BroadcastToSession 向指定 session 的所有客户端广播消息
func (b *Bridge) BroadcastToSession(sessionID string, messageType string, data any) error {
	msg := Message{
		Type:      messageType,
		SessionID: sessionID,
		Data:      data,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return errors.WrapInfrastructureError(err, "marshal message failed")
	}

	count := b.hub.Broadcast(sessionID, payload)
	b.logger.Debug("broadcast to session", "sessionID", sessionID, "type", messageType, "clients", count)

	return nil
}

// GetHub 获取 WebSocket Hub
func (b *Bridge) GetHub() ws.IHub {
	return b.hub
}

// GetSessionClients 获取指定 session 的所有客户端
func (b *Bridge) GetSessionClients(sessionID string) []*ws.Client {
	return b.hub.GetGroupClients(sessionID)
}

// GetSessionClientCount 获取指定 session 的客户端数量
func (b *Bridge) GetSessionClientCount(sessionID string) int {
	return b.hub.GroupClientCount(sessionID)
}

// ==================== WorkflowEventPublisher 实现 ====================

// PublishDecision 发布决策事件
func (b *Bridge) PublishDecision(sessionID string, dec engine.Decision) error {
	return b.BroadcastToSession(sessionID, "decision_log_append", map[string]interface{}{
		"list": []engine.Decision{dec},
	})
}

// PublishStateChange 发布状态变更事件
func (b *Bridge) PublishStateChange(sessionID string, from engine.State, to engine.State, event engine.Event) error {
	// 状态变更已经通过 AddSystemMessage 在 session.Messages 中记录
	// 这里广播 workflow_update 事件
	return b.BroadcastToSession(sessionID, "workflow_update", map[string]interface{}{
		"phase": to,
		"from":  from,
		"event": event,
	})
}

// PublishWorkflowCompleted 发布工作流完成事件
func (b *Bridge) PublishWorkflowCompleted(sessionID string, ctx engine.Context) error {
	return b.BroadcastToSession(sessionID, "finalize_completed", map[string]interface{}{
		"sessionId": sessionID,
		"status":    "completed",
	})
}

// PublishWorkflowFailed 发布工作流失败事件
func (b *Bridge) PublishWorkflowFailed(sessionID string, reason string, errorMsg string) error {
	return b.BroadcastToSession(sessionID, "error", map[string]interface{}{
		"reason":  reason,
		"message": errorMsg,
	})
}

// Message 通用消息格式
type Message struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Data      any    `json:"data"`
}

// ==================== Session 消息处理 ====================

// handleGetRecentSession 处理获取最近会话请求
func (b *Bridge) handleGetRecentSession(ctx context.Context, client *ws.Client, data map[string]interface{}) error {
	orgID, _ := data["orgId"].(string)
	userID, _ := data["userId"].(string)

	if orgID == "" || userID == "" {
		b.logger.Warn("get_recent_session: missing orgId or userId")
		return b.sendError(client, "orgId and userId are required")
	}

	b.logger.Info("handling get_recent_session", "orgId", orgID, "userId", userID)

	// 查询最近的 Session
	sessions, err := b.sessionService.List(ctx, session.SessionFilter{
		OrgID:  orgID,
		UserID: userID,
		Limit:  1, // 只取最近一条（已按 UpdatedAt DESC 排序）
	})

	if err != nil {
		b.logger.Error("failed to list sessions", "error", err, "orgId", orgID, "userId", userID)
		return b.sendError(client, "failed to get recent session")
	}

	var recentSession *session.Session
	if len(sessions) > 0 {
		recentSession = sessions[0]
		b.logger.Info("found recent session", "sessionId", recentSession.ID)

		// 绑定客户端到会话（关键：切换标签页重新连接时需要重新绑定）
		if err := b.BindSession(client, recentSession.ID); err != nil {
			b.logger.Warn("failed to bind client to recent session", "sessionId", recentSession.ID, "error", err)
			// 绑定失败不影响返回会话数据，但可能导致消息发送失败
		}

		// 发送响应
		return b.sendMessage(client, MsgTypeSessionSnapshot, recentSession)
	} else {
		b.logger.Info("no recent session found", "orgId", orgID, "userId", userID)
		// 找不到会话，发送错误消息和操作按钮
		return b.sendErrorWithActions(client, "找不到会话，请选择操作：", []map[string]interface{}{
			// {
			// 	"id":    "reconnect",
			// 	"type":  "workflow",
			// 	"label": "重连",
			// 	"event": "reconnect_session",
			// 	"style": "primary",
			// },
			{
				"id":    "create_new_and_disconnect",
				"type":  "workflow",
				"label": "发起新会话",
				"event": "create_new_session_and_disconnect",
				"style": "secondary",
			},
		})
	}
}

// handleCreateSession 处理创建会话请求
func (b *Bridge) handleCreateSession(ctx context.Context, client *ws.Client, data map[string]interface{}) error {
	orgID, _ := data["orgId"].(string)
	userID, _ := data["userId"].(string)
	agentType, _ := data["agentType"].(string)

	if orgID == "" || userID == "" || agentType == "" {
		b.logger.Warn("create_session: missing required fields")
		return b.sendError(client, "orgId, userId and agentType are required")
	}

	b.logger.Info("handling create_session", "orgId", orgID, "userId", userID, "agentType", agentType)

	// 创建新 Session
	sess, err := b.sessionService.Create(ctx, session.CreateSessionRequest{
		OrgID:     orgID,
		UserID:    userID,
		AgentType: agentType,
	})

	if err != nil {
		b.logger.Error("failed to create session", "error", err)
		return b.sendError(client, "failed to create session")
	}

	// 立即绑定客户端到 session（关键！）
	if err := b.BindSession(client, sess.ID); err != nil {
		b.logger.Error("failed to bind client to session", "sessionId", sess.ID, "error", err)
		// 绑定失败但 session 已创建，不删除 session，返回错误让客户端重试
		return b.sendError(client, "session created but binding failed, please reconnect")
	}

	// 发送欢迎消息
	welcomeMsg := `你好！我是智能排班助手。

我可以帮你：
• 快速生成排班计划
• 优化排班安排
• 处理排班冲突

请告诉我你需要什么帮助。
你可以说："开始排班"进行一次新的排班。
`

	updatedSess, err := b.sessionService.AddAssistantMessage(ctx, sess.ID, welcomeMsg)
	if err != nil {
		b.logger.Warn("failed to add welcome message", "error", err)
	} else {
		// 广播欢迎消息
		_ = b.BroadcastToSession(sess.ID, MsgTypeAssistantMessage, map[string]interface{}{
			"message":   welcomeMsg,
			"timestamp": updatedSess.Messages[len(updatedSess.Messages)-1].Timestamp,
		})
	}

	// 发送工作流版本系统消息（在欢迎消息之后同步发送）
	// 重新获取会话以获取最新的WorkflowMeta（包含版本信息）
	updatedSessForVersion, err := b.sessionService.Get(ctx, sess.ID)
	if err == nil && updatedSessForVersion != nil && updatedSessForVersion.WorkflowMeta != nil && updatedSessForVersion.WorkflowMeta.Version != "" {
		versionMsg := fmt.Sprintf("当前工作流版本：%s", strings.ToUpper(updatedSessForVersion.WorkflowMeta.Version))
		updatedSessWithMsg, err := b.sessionService.AddSystemMessage(ctx, sess.ID, versionMsg)
		if err != nil {
			b.logger.Warn("failed to add workflow version message", "error", err)
		} else {
			// 广播系统消息
			_ = b.BroadcastToSession(sess.ID, MsgTypeSystemMessage, map[string]interface{}{
				"message":   versionMsg,
				"timestamp": updatedSessWithMsg.Messages[len(updatedSessWithMsg.Messages)-1].Timestamp,
			})
		}
	}

	// 发送响应
	return b.sendMessage(client, MsgTypeSessionCreated, map[string]interface{}{
		"sessionId": sess.ID,
		"createdAt": sess.CreatedAt,
	})
}

// handleUserMessage 处理用户文本消息
func (b *Bridge) handleUserMessage(ctx context.Context, client *ws.Client, data map[string]interface{}) error {
	sessionID, _ := data["sessionId"].(string)
	message, _ := data["message"].(string)

	if sessionID == "" || message == "" {
		b.logger.Warn("user_message: missing sessionId or message")
		return b.sendError(client, "sessionId and message are required")
	}

	b.logger.Info("handling user_message", "sessionId", sessionID, "message", message)

	// 验证 session 是否存在
	_, err := b.sessionService.Get(ctx, sessionID)
	if err != nil {
		b.logger.Error("session not found", "sessionId", sessionID, "error", err)
		return b.sendError(client, "session not found")
	}

	// 添加用户消息到 session
	if _, err := b.sessionService.AddUserMessage(ctx, sessionID, message); err != nil {
		b.logger.Error("failed to add user message", "error", err)
		return b.sendError(client, "failed to save message")
	}

	// 调用意图识别
	intentResp, err := b.sessionService.RecognizeIntent(ctx, sessionID, message)

	// 根据意图识别结果决定响应
	var assistantMsg string
	var intentType string // 用于日志和前端判断

	if err != nil {
		// 意图识别出错（如 AI 调用失败）
		b.logger.Error("intent recognition error", "sessionId", sessionID, "error", err)
		assistantMsg = "抱歉，我暂时无法理解您的需求。请稍后再试或联系管理员。"
		intentType = "error"
	} else if intentResp == nil || intentResp.Intent == nil {
		// 未识别到意图
		b.logger.Info("no intent recognized", "sessionId", sessionID)
		assistantMsg = "收到您的消息，但是我不太明白您想做什么，您可以更详细地描述您的需求吗？"
		intentType = "unknown"
	} else {
		// 识别到意图
		intent := intentResp.Intent
		intentType = intent.Type

		// 检查是否需要更多信息
		if intentResp.NeedsMoreInfo && len(intentResp.MissingFields) > 0 {
			assistantMsg = fmt.Sprintf("请提供以下信息：%v", intentResp.MissingFields)
		} else {
			// 信息完整，尝试启动工作流
			assistantMsg = b.tryStartWorkflow(ctx, sessionID, intent)
		}
	}

	// 添加助手消息
	sess, err := b.sessionService.AddAssistantMessage(ctx, sessionID, assistantMsg)
	if err != nil {
		b.logger.Error("failed to add assistant message", "error", err)
		return b.sendError(client, "failed to save response")
	}

	// 通过 BroadcastToSession 发送助手消息
	_ = b.BroadcastToSession(sessionID, MsgTypeAssistantMessage, map[string]interface{}{
		"message":    assistantMsg,
		"timestamp":  sess.Messages[len(sess.Messages)-1].Timestamp,
		"intentType": intentType, // 添加意图类型，便于前端判断状态
	})

	return nil
}

// tryStartWorkflow 尝试启动工作流，返回助手消息
func (b *Bridge) tryStartWorkflow(ctx context.Context, sessionID string, intent *session.Intent) string {
	if b.eventSender == nil {
		b.logger.Error("eventSender not configured", "sessionId", sessionID)
		return "抱歉，工作流系统未初始化，无法处理您的请求。请联系管理员。"
	}

	if b.intentEventMapper == nil {
		b.logger.Error("intentEventMapper not configured", "sessionId", sessionID)
		return "抱歉，意图映射器未配置，无法处理您的请求。请联系管理员。"
	}

	// 先获取会话，检查是否已设置工作流
	sess, err := b.sessionService.Get(ctx, sessionID)
	if err != nil {
		b.logger.Error("failed to get session", "error", err, "sessionId", sessionID)
		return "抱歉，系统出现问题，无法启动工作流。请稍后重试。"
	}

	var workflowName string
	var workflowEvent engine.Event

	// 优先使用会话中已设置的工作流
	if sess.WorkflowMeta != nil && sess.WorkflowMeta.Workflow != "" {
		workflowName = sess.WorkflowMeta.Workflow
		// 从映射表获取事件（工作流名称已确定，只需要事件）
		mapping := b.intentEventMapper.MapIntentToWorkflow(intent.Type)
		if mapping == nil || mapping.Event == "" {
			b.logger.Warn("intent type not mapped for event", "intentType", intent.Type, "sessionId", sessionID, "workflow", workflowName)
			return fmt.Sprintf("抱歉，我无法处理「%s」类型的请求。", intent.Type)
		}
		workflowEvent = mapping.Event
	} else {
		// 如果会话中没有工作流，使用映射表
	mapping := b.intentEventMapper.MapIntentToWorkflow(intent.Type)
	if mapping == nil || mapping.Event == "" {
		b.logger.Warn("intent type not mapped", "intentType", intent.Type, "sessionId", sessionID)
		return fmt.Sprintf("抱歉，我无法处理「%s」类型的请求。", intent.Type)
	}

	if !mapping.Implemented {
		b.logger.Warn("workflow not implemented", "intentType", intent.Type, "workflow", mapping.WorkflowName)
		return fmt.Sprintf("抱歉，「%s」功能暂未开放，敬请期待。", intent.Type)
	}

		workflowName = mapping.WorkflowName
		workflowEvent = mapping.Event

	// 设置 WorkflowMeta
	_, updateErr := b.sessionService.Update(ctx, sessionID, func(s *session.Session) error {
		if s.WorkflowMeta == nil {
			s.WorkflowMeta = &session.WorkflowMeta{}
		}
			s.WorkflowMeta.Workflow = workflowName
		return nil
	})

	if updateErr != nil {
		b.logger.Error("failed to update workflow meta", "error", updateErr, "sessionId", sessionID)
		return "抱歉，系统出现问题，无法启动工作流。请稍后重试。"
		}

	}

	// 异步触发工作流
	b.logger.Info("starting workflow",
		"sessionId", sessionID,
		"workflow", workflowName,
		"event", workflowEvent)

	go func() {
		if err := b.eventSender.SendEvent(ctx, sessionID, workflowEvent, intent); err != nil {
			b.logger.Error("failed to start workflow", "error", err)
			_ = b.PublishWorkflowFailed(sessionID, "workflow_start_failed", err.Error())
		}
	}()

	return "已识别您的需求，正在为您准备..."
}

// handleWorkflowCommand 处理工作流命令
func (b *Bridge) handleWorkflowCommand(ctx context.Context, client *ws.Client, data map[string]interface{}) error {
	sessionID, _ := data["sessionId"].(string)
	command, _ := data["command"].(string)
	payload := data["payload"]

	if sessionID == "" || command == "" {
		b.logger.Warn("workflow_command: missing sessionId or command")
		return b.sendError(client, "sessionId and command are required")
	}

	b.logger.Info("handling workflow_command", "sessionId", sessionID, "command", command)

	// 检查是否设置了事件发送器
	if b.eventSender == nil {
		b.logger.Error("eventSender not set, cannot process workflow command")
		return b.sendError(client, "workflow system not initialized")
	}

	// 检查是否设置了命令映射器
	if b.commandMapper == nil {
		b.logger.Error("commandMapper not set, cannot map command to event")
		return b.sendError(client, "command mapper not configured")
	}

	// 将命令映射为 FSM 事件
	event := b.commandMapper.MapCommandToEvent(command)
	if event == "" {
		b.logger.Warn("unknown command", "command", command)
		return b.sendError(client, fmt.Sprintf("unknown command: %s", command))
	}

	// 验证 session 是否存在（避免发送到不存在的会话）
	if _, err := b.sessionService.Get(ctx, sessionID); err != nil {
		b.logger.Error("session not found for workflow command", "sessionId", sessionID, "error", err)
		return b.sendErrorWithActions(client, "会话已失效，请重新连接或新建会话。", []map[string]interface{}{
			{
				"id":    "reconnect",
				"type":  "command",
				"label": "重新连接",
				"event": "reconnect_session",
				"style": "primary",
			},
			{
				"id":    "create_new_and_disconnect",
				"type":  "command",
				"label": "新建会话",
				"event": "create_new_session_and_disconnect",
				"style": "secondary",
			},
		})
	}

	// 发送事件到工作流系统
	if err := b.eventSender.SendEvent(ctx, sessionID, event, payload); err != nil {
		b.logger.Error("failed to send event", "event", event, "error", err)
		return b.sendError(client, fmt.Sprintf("failed to process command: %v", err))
	}

	return nil
}

// handleLoadConversation 处理加载历史对话消息
func (b *Bridge) handleLoadConversation(ctx context.Context, client *ws.Client, data map[string]interface{}) error {
	sessionID, _ := data["sessionId"].(string)
	conversationID, _ := data["conversationId"].(string)

	if sessionID == "" || conversationID == "" {
		b.logger.Warn("load_conversation: missing sessionId or conversationId")
		return b.sendError(client, "sessionId and conversationId are required")
	}

	b.logger.Info("handling load_conversation", "sessionId", sessionID, "conversationId", conversationID)

	// 检查是否设置了对话加载器
	if b.conversationLoader == nil {
		b.logger.Error("conversationLoader not set, cannot load conversation")
		return b.sendError(client, "conversation loader not configured")
	}

	// 加载对话（这会更新 session，OnSessionUpdate 回调会自动广播 session_updated）
	if err := b.conversationLoader.LoadConversation(ctx, sessionID, conversationID); err != nil {
		b.logger.Error("failed to load conversation", "error", err, "sessionId", sessionID, "conversationId", conversationID)
		return b.sendError(client, fmt.Sprintf("failed to load conversation: %v", err))
	}

	b.logger.Info("conversation loaded successfully", "sessionId", sessionID, "conversationId", conversationID)
	return nil
}

// sendMessage 发送消息给客户端
func (b *Bridge) sendMessage(client *ws.Client, msgType string, data interface{}) error {
	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		b.logger.Error("failed to marshal message", "error", err)
		return err
	}

	client.Send(msgBytes)
	b.logger.Debug("message sent", "type", msgType, "client", client.ID())
	return nil
}

// sendError 发送错误消息给客户端
func (b *Bridge) sendError(client *ws.Client, errMsg string) error {
	return b.sendMessage(client, MsgTypeError, map[string]any{
		"message": errMsg,
	})
}

// sendErrorWithActions 发送带操作按钮的错误消息
func (b *Bridge) sendErrorWithActions(client *ws.Client, errMsg string, actions []map[string]interface{}) error {
	return b.sendMessage(client, MsgTypeError, map[string]any{
		"message": errMsg,
		"actions": actions,
	})
}
