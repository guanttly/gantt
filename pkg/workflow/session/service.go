package session

import (
	"context"
	"strings"
	"time"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"
)

// WorkflowInitResult 工作流初始化结果
type WorkflowInitResult struct {
	WorkflowName string // 工作流名称
	Version      string // 工作流版本（如 "v2", "v3"）
}

// IWorkflowInitializer 工作流初始化器接口（业务层实现）
// 业务系统需要实现此接口来定义 agentType 到 workflow 的映射
type IWorkflowInitializer interface {
	// InitializeWorkflow 根据 agentType 初始化工作流
	// 返回 workflow 名称和版本，如果该 agentType 不需要工作流则返回空字符串
	// orgID 和 userID 用于获取用户特定的工作流偏好（如版本选择）
	// 为了向后兼容，如果返回空字符串，表示不需要工作流
	// 如果返回非空字符串，表示工作流名称（旧接口）
	// 新实现应该返回 WorkflowInitResult
	InitializeWorkflow(agentType, orgID, userID string) string
	// InitializeWorkflowWithVersion 根据 agentType 初始化工作流并返回版本信息
	// 如果实现类支持此方法，优先使用此方法
	InitializeWorkflowWithVersion(agentType, orgID, userID string) (WorkflowInitResult, bool)
}

// Service 会话服务接口
// 提供会话生命周期管理、消息管理、状态管理等能力
type ISessionService interface {
	// 会话生命周期
	Create(ctx context.Context, req CreateSessionRequest) (*Session, error)
	Get(ctx context.Context, id string) (*Session, error)
	Update(ctx context.Context, id string, mutate func(*Session) error) (*Session, error)
	Delete(ctx context.Context, id string) error

	// 消息管理
	AddMessage(ctx context.Context, sessionID string, msg Message) (*Session, error)
	AddUserMessage(ctx context.Context, sessionID string, content string) (*Session, error)
	AddAssistantMessage(ctx context.Context, sessionID string, content string) (*Session, error)
	AddAssistantMessageWithActions(ctx context.Context, sessionID string, content string, actions []WorkflowAction) (*Session, error)
	AddSystemMessage(ctx context.Context, sessionID string, content string) (*Session, error)
	GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)

	// 意图识别
	RecognizeIntent(ctx context.Context, sessionID string, userMessage string) (*IntentRecognizeResponse, error)

	// 状态管理
	SetState(ctx context.Context, sessionID string, state SessionState, desc string) (*Session, error)
	SetError(ctx context.Context, sessionID string, errMsg string) (*Session, error)

	// 工作流元数据
	SetWorkflowMeta(ctx context.Context, sessionID string, meta *WorkflowMeta) (*Session, error)
	UpdateWorkflowMeta(ctx context.Context, sessionID string, mutate func(*WorkflowMeta) error) (*Session, error)

	// 业务数据
	SetData(ctx context.Context, sessionID string, key string, value any) (*Session, error)
	GetData(ctx context.Context, sessionID string, key string) (any, bool, error)

	// 查询
	List(ctx context.Context, filter SessionFilter) ([]*Session, error)
	Count(ctx context.Context, filter SessionFilter) (int, error)

	// 完成会话
	Finalize(ctx context.Context, sessionID string) (*Session, error)

	// 事件回调（由 Infrastructure 注入）
	SetOnUpdate(fn func(*Session))
	SetOnWorkflowMetaUpdate(fn func(sessionID string, meta *WorkflowMeta))

	// 依赖注入方法
	WithStore(store IStore) ISessionService
	WithLogger(logger logging.ILogger) ISessionService
	WithIntentRecognizer(recognizer IIntentRecognizer) ISessionService
	WithWorkflowInitializer(initializer IWorkflowInitializer) ISessionService
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	OrgID       string
	UserID      string
	AgentType   string
	InitialData map[string]any
}

// sessionService 会话服务实现
type sessionService struct {
	store                IStore
	logger               logging.ILogger
	intentRecognizer     IIntentRecognizer                          // 可选，通过 WithIntentRecognizer 注入
	workflowInitializer  IWorkflowInitializer                       // 可选，通过 WithWorkflowInitializer 注入
	onUpdate             func(*Session)                             // session 更新回调（用于 WebSocket 广播）
	onWorkflowMetaUpdate func(sessionID string, meta *WorkflowMeta) // WorkflowMeta 更新回调
}

// newSessionService 创建会话服务
func newSessionService(store IStore, logger logging.ILogger) ISessionService {
	return &sessionService{
		store:  store,
		logger: logger,
	}
}

// SetOnUpdate 设置 session 更新回调（由 Infrastructure 注入）
func (s *sessionService) SetOnUpdate(fn func(*Session)) {
	s.onUpdate = fn
}

// SetOnWorkflowMetaUpdate 设置 WorkflowMeta 更新回调（由 Infrastructure 注入）
func (s *sessionService) SetOnWorkflowMetaUpdate(fn func(sessionID string, meta *WorkflowMeta)) {
	s.onWorkflowMetaUpdate = fn
}

// Create 创建会话
func (s *sessionService) Create(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	session := NewSession(req.OrgID, req.UserID, req.AgentType)

	// 设置初始数据
	if req.InitialData != nil {
		for k, v := range req.InitialData {
			session.Data[k] = v
		}
	}

	// 如果注入了工作流初始化器，则初始化工作流
	if s.workflowInitializer != nil {
		var workflowName string
		var version string

		// 优先使用新接口（支持版本信息）
		if initializerWithVersion, ok := s.workflowInitializer.(interface {
			InitializeWorkflowWithVersion(agentType, orgID, userID string) (WorkflowInitResult, bool)
		}); ok {
			if result, ok := initializerWithVersion.InitializeWorkflowWithVersion(req.AgentType, req.OrgID, req.UserID); ok {
				workflowName = result.WorkflowName
				version = result.Version
			}
		}

		// 如果新接口不可用或返回空，使用旧接口
		if workflowName == "" {
			workflowName = s.workflowInitializer.InitializeWorkflow(req.AgentType, req.OrgID, req.UserID)
			if workflowName != "" {
				// 从工作流名称中解析版本（向后兼容）
				version = extractWorkflowVersion(workflowName)
			}
		}

		if workflowName != "" {
			session.WorkflowMeta = &WorkflowMeta{
				Workflow: workflowName,
				Version:  version,
				Phase:    "", // 初始状态为空，将在第一次 SendEvent 时由 FSM 设置
			}
			s.logger.Info("workflow initialized for session", "sessionId", session.ID, "workflow", workflowName, "version", version)
		}
	}

	if err := s.store.Set(session); err != nil {
		s.logger.Error("failed to create session", "error", err)
		return nil, errors.WrapInfrastructureError(err, "create session failed")
	}

	s.logger.Info("session created", "id", session.ID, "agentType", session.AgentType)
	return session, nil
}

// Get 获取会话
func (s *sessionService) Get(ctx context.Context, id string) (*Session, error) {
	session, err := s.store.Get(id)
	if err != nil {
		return nil, errors.WrapInfrastructureError(err, "get session failed")
	}
	if session == nil {
		return nil, errors.NewNotFoundError("session not found", nil)
	}
	return session, nil
}

// Delete 删除会话
func (s *sessionService) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(id); err != nil {
		s.logger.Error("failed to delete session", "id", id, "error", err)
		return errors.WrapInfrastructureError(err, "delete session failed")
	}
	s.logger.Info("session deleted", "id", id)
	return nil
}

// Update 更新会话
func (s *sessionService) Update(ctx context.Context, id string, mutate func(*Session) error) (*Session, error) {
	const maxRetries = 5

	for i := 0; i < maxRetries; i++ {
		session, err := s.store.Get(id)
		if err != nil {
			return nil, errors.WrapInfrastructureError(err, "get session failed")
		}
		if session == nil {
			return nil, errors.NewNotFoundError("session not found", nil)
		}

		expectedVersion := session.Version

		ok, newVersion, err := s.store.Update(id, expectedVersion, mutate)
		if err != nil {
			return nil, errors.WrapInfrastructureError(err, "update session failed")
		}

		if ok {
			// 成功，获取更新后的会话
			updated, _ := s.store.Get(id)

			// 触发更新回调（WebSocket 广播）
			if s.onUpdate != nil && updated != nil {
				s.onUpdate(updated)
			}
			// 注意：s.onUpdate 可能被 UpdateWorkflowMeta 临时禁用，此时不需要警告

			return updated, nil
		}

		// CAS 失败，重试
		s.logger.Debug("session update conflict, retrying",
			"id", id,
			"expectedVersion", expectedVersion,
			"currentVersion", newVersion,
			"attempt", i+1)
	}

	return nil, errors.NewConflictError("update session failed after retries", nil)
}

// AddMessage 添加消息
func (s *sessionService) AddMessage(ctx context.Context, sessionID string, msg Message) (*Session, error) {
	updated, err := s.Update(ctx, sessionID, func(session *Session) error {
		if msg.ID == "" {
			msg.ID = generateID()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}
		session.Messages = append(session.Messages, msg)
		session.UpdatedAt = time.Now()
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updated, nil
}

// AddUserMessage 添加用户消息
func (s *sessionService) AddUserMessage(ctx context.Context, sessionID string, content string) (*Session, error) {
	return s.AddMessage(ctx, sessionID, Message{
		Role:    RoleUser,
		Content: content,
	})
}

// AddAssistantMessage 添加助手消息
func (s *sessionService) AddAssistantMessage(ctx context.Context, sessionID string, content string) (*Session, error) {
	return s.AddMessage(ctx, sessionID, Message{
		Role:    RoleAssistant,
		Content: content,
	})
}

// AddAssistantMessageWithActions 添加带操作按钮的助手消息
func (s *sessionService) AddAssistantMessageWithActions(ctx context.Context, sessionID string, content string, actions []WorkflowAction) (*Session, error) {
	return s.AddMessage(ctx, sessionID, Message{
		Role:    RoleAssistant,
		Content: content,
		Actions: actions,
	})
}

// AddSystemMessage 添加系统消息
func (s *sessionService) AddSystemMessage(ctx context.Context, sessionID string, content string) (*Session, error) {
	return s.AddMessage(ctx, sessionID, Message{
		Role:    RoleSystem,
		Content: content,
	})
}

// GetMessages 获取消息列表
func (s *sessionService) GetMessages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages := session.Messages
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// RecognizeIntent 识别用户消息的意图
func (s *sessionService) RecognizeIntent(ctx context.Context, sessionID string, userMessage string) (*IntentRecognizeResponse, error) {
	// 如果没有配置意图识别器，返回 nil
	if s.intentRecognizer == nil {
		s.logger.Warn("intent recognizer not configured", "sessionID", sessionID)
		return nil, nil
	}

	// 获取 session 以提供上下文
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		s.logger.Error("failed to get session for intent recognition", "sessionID", sessionID, "error", err)
		return nil, err
	}

	// 构建基本上下文信息
	contextData := IntentRecognizeContext{
		OrgID:        session.OrgID,
		UserID:       session.UserID,
		AgentType:    session.AgentType,
		CurrentState: session.State,
		Messages:     session.Messages,
	}

	// 调用意图识别器
	req := IntentRecognizeRequest{
		SessionID:   sessionID,
		UserMessage: userMessage,
		Context:     contextData,
	}

	response, err := s.intentRecognizer.Recognize(ctx, req)
	if err != nil {
		s.logger.Error("intent recognition failed", "sessionID", sessionID, "error", err)
		return nil, err
	}

	return response, nil
}

// SetState 设置状态
func (s *sessionService) SetState(ctx context.Context, sessionID string, state SessionState, desc string) (*Session, error) {
	return s.Update(ctx, sessionID, func(session *Session) error {
		session.SetState(state, desc)
		return nil
	})
}

// SetError 设置错误
func (s *sessionService) SetError(ctx context.Context, sessionID string, errMsg string) (*Session, error) {
	return s.Update(ctx, sessionID, func(session *Session) error {
		session.SetError(errMsg)
		return nil
	})
}

// SetWorkflowMeta 设置工作流元数据
func (s *sessionService) SetWorkflowMeta(ctx context.Context, sessionID string, meta *WorkflowMeta) (*Session, error) {
	return s.Update(ctx, sessionID, func(session *Session) error {
		session.WorkflowMeta = meta
		session.UpdatedAt = time.Now()
		return nil
	})
}

// UpdateWorkflowMeta 更新工作流元数据
// 注意：此方法只触发 onWorkflowMetaUpdate,不触发 onUpdate,避免重复广播
func (s *sessionService) UpdateWorkflowMeta(ctx context.Context, sessionID string, mutate func(*WorkflowMeta) error) (*Session, error) {
	// 临时禁用 onUpdate 回调
	originalOnUpdate := s.onUpdate
	s.onUpdate = nil
	defer func() {
		s.onUpdate = originalOnUpdate
	}()

	updated, err := s.Update(ctx, sessionID, func(session *Session) error {
		if session.WorkflowMeta == nil {
			session.WorkflowMeta = &WorkflowMeta{
				Extra: make(map[string]any),
			}
		}
		if err := mutate(session.WorkflowMeta); err != nil {
			return err
		}
		session.UpdatedAt = time.Now()
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 触发 WorkflowMeta 专属回调
	if s.onWorkflowMetaUpdate != nil && updated != nil && updated.WorkflowMeta != nil {
		s.onWorkflowMetaUpdate(sessionID, updated.WorkflowMeta)
	}

	return updated, nil
}

// SetData 设置业务数据
func (s *sessionService) SetData(ctx context.Context, sessionID string, key string, value any) (*Session, error) {
	return s.Update(ctx, sessionID, func(session *Session) error {
		if session.Data == nil {
			session.Data = make(map[string]any)
		}
		session.Data[key] = value
		session.UpdatedAt = time.Now()
		return nil
	})
}

// GetData 获取业务数据
func (s *sessionService) GetData(ctx context.Context, sessionID string, key string) (any, bool, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, false, err
	}

	value, ok := session.Data[key]
	return value, ok, nil
}

// List 查询会话列表
func (s *sessionService) List(ctx context.Context, filter SessionFilter) ([]*Session, error) {
	return s.store.List(filter)
}

// Count 统计会话数量
func (s *sessionService) Count(ctx context.Context, filter SessionFilter) (int, error) {
	return s.store.Count(filter)
}

// Finalize 完成会话
func (s *sessionService) Finalize(ctx context.Context, sessionID string) (*Session, error) {
	session, err := s.Update(ctx, sessionID, func(session *Session) error {
		session.State = StateCompleted
		session.StateDesc = "completed"
		session.UpdatedAt = time.Now()
		return nil
	})

	if err == nil {
		s.logger.Info("session finalized", "id", sessionID)
	}

	return session, err
}

// ==================== 依赖注入方法 (With Pattern) ====================

// WithStore 替换 Store 实现
func (s *sessionService) WithStore(store IStore) ISessionService {
	s.store = store
	return s
}

// WithLogger 替换 Logger 实现
func (s *sessionService) WithLogger(logger logging.ILogger) ISessionService {
	s.logger = logger
	return s
}

// WithIntentRecognizer 注入或替换意图识别器
func (s *sessionService) WithIntentRecognizer(recognizer IIntentRecognizer) ISessionService {
	s.intentRecognizer = recognizer
	return s
}

// WithWorkflowInitializer 注入工作流初始化器
func (s *sessionService) WithWorkflowInitializer(initializer IWorkflowInitializer) ISessionService {
	s.workflowInitializer = initializer
	return s
}

// ========== WorkflowMeta 辅助方法 ==========

// SetWorkflowActions 设置工作流交互按钮的辅助方法
// 统一使用 UpdateWorkflowMeta 来持久化修改,会自动触发 onUpdate 回调
// transitionID 用于标记 actions 所属的状态转换，引擎会根据此判断是否需要自动清除
func SetWorkflowActions(ctx context.Context, service ISessionService, sessionID string, actions []WorkflowAction) error {
	_, err := service.UpdateWorkflowMeta(ctx, sessionID, func(meta *WorkflowMeta) error {
		meta.Actions = actions
		// 保留当前的 transitionID，表示这些 actions 属于当前转换
		return nil
	})
	return err
}

// SetWorkflowMetaWithActions 设置工作流描述和交互按钮的辅助方法
// transitionID 用于标记 actions 所属的状态转换，引擎会根据此判断是否需要自动清除
func SetWorkflowMetaWithActions(ctx context.Context, service ISessionService, sessionID string, description string, actions []WorkflowAction) error {
	_, err := service.UpdateWorkflowMeta(ctx, sessionID, func(meta *WorkflowMeta) error {
		meta.Description = description
		meta.Actions = actions
		// 保留当前的 transitionID，表示这些 actions 属于当前转换
		return nil
	})
	return err
}

// extractWorkflowVersion 从工作流名称中提取版本号
func extractWorkflowVersion(workflowName string) string {
	if strings.Contains(workflowName, "schedule_v2") {
		return "v2"
	}
	if strings.Contains(workflowName, "schedule_v3") {
		return "v3"
	}
	return ""
}
