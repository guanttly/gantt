package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/domain/service"
	"jusha/agent/sdk/context/domain"
	context_model "jusha/agent/sdk/context/model"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
)

const (
	// DataKeyConversationID 会话ID在 Session.Data 中的键名
	DataKeyConversationID = "conversationId"
	// DataKeyCompressedAt 最后压缩时间在 Session.Data 中的键名
	DataKeyCompressedAt = "compressedAt"
	// DataKeyMessageCount 当前消息数在 Session.Data 中的键名
	DataKeyMessageCount = "messageCount"
	// DataKeyOriginalMessageCount 原始消息数在 Session.Data 中的键名
	DataKeyOriginalMessageCount = "originalMessageCount"
)

// conversationManager 会话管理服务实现
type conversationManager struct {
	logger         logging.ILogger
	contextClient  domain.IContextClient
	sessionService session.ISessionService
	configurator   config.IRosteringConfigurator
	toolBus        mcp.IToolBus // 用于调用管理服务的 MCP 工具
}

// NewConversationManager 创建会话管理服务
func NewConversationManager(
	logger logging.ILogger,
	contextClient domain.IContextClient,
	sessionService session.ISessionService,
	cfg config.IRosteringConfigurator,
	toolBus mcp.IToolBus,
) service.IConversationService {
	return &conversationManager{
		logger:         logger.With("component", "ConversationManager"),
		contextClient:  contextClient,
		sessionService: sessionService,
		configurator:   cfg,
		toolBus:        toolBus,
	}
}

// SaveConversation 保存会话消息到上下文服务
func (c *conversationManager) SaveConversation(ctx context.Context, sessionID string, messages []session.Message) error {
	// 获取 session（每次重新获取最新的 session，避免使用过期的数据）
	sess, err := c.sessionService.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session failed: %w", err)
	}

	// 获取或创建 conversation ID（这里会再次获取 session，但会使用最新的）
	conversationID, err := c.getOrCreateConversationID(ctx, sess)
	if err != nil {
		return fmt.Errorf("get or create conversation ID failed: %w", err)
	}

	// 重新获取 session，确保获取到最新的 conversationID 和消息计数
	sess, err = c.sessionService.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session failed: %w", err)
	}

	// 获取已保存的消息ID集合（用于去重）
	savedMessageIDs := make(map[string]bool)
	if sess.Data != nil {
		// 尝试从 session.Data 中获取已保存的消息ID集合
		if savedIDsRaw, ok := sess.Data["savedMessageIDs"]; ok {
			if savedIDsMap, ok := savedIDsRaw.(map[string]any); ok {
				for msgID := range savedIDsMap {
					savedMessageIDs[msgID] = true
				}
			}
		}
	}

	// 如果 session.Data 中没有保存的消息ID集合，从 context-server 查询
	if len(savedMessageIDs) == 0 {
		// 尝试从 context-server 获取已保存的消息ID
		// 注意：这里需要调用新的方法，但为了向后兼容，我们先使用现有逻辑
		// 如果消息没有ID，则使用内容+时间戳+角色生成唯一标识（向后兼容）
		c.logger.Debug("No saved message IDs in session, will check during save", "conversationID", conversationID)
	}

	// 遍历所有消息，保存或更新（使用 ON DUPLICATE KEY UPDATE 处理去重和更新）
	savedCount := 0      // 实际成功保存的消息数量
	newMessageCount := 0 // 新插入的消息数量（用于计数）
	for _, msg := range messages {
		// 如果消息没有ID，生成一个临时ID用于去重（向后兼容）
		messageID := msg.ID
		if messageID == "" {
			// 为旧消息生成唯一标识（基于内容+时间戳+角色）
			messageID = c.generateMessageID(msg)
		}

		// 检查是否是第一次保存（用于计数和统计）
		wasNew := !savedMessageIDs[messageID]

		// 构建 Metadata，包含 Actions（如果存在）
		// 注意：即使 msg.Actions 为空，metadata 也为空 map，这样会覆盖掉旧的 metadata（用于清除 actions）
		metadata := make(map[string]any)
		if len(msg.Actions) > 0 {
			// 将 Actions 序列化为 JSON 字符串，存储在 Metadata 中
			actionsJSON, err := json.Marshal(msg.Actions)
			if err != nil {
				c.logger.Warn("Failed to marshal actions to JSON", "error", err, "messageID", messageID)
			} else {
				metadata["actions"] = string(actionsJSON)
			}
		}
		// 注意：即使 msg.Actions 为空，metadata 也为空 map，这样会覆盖掉旧的 metadata

		// 保存消息（如果已存在则更新），传递 messageID 和 Metadata（包含 Actions）
		req := context_model.ConversationAppendRequest{
			ID:        conversationID,
			MessageID: messageID,
			Role:      string(msg.Role),
			Content:   msg.Content,
			Metadata:  metadata, // 即使是空的 map，也会覆盖旧的 metadata
		}

		_, err := c.contextClient.AppendMessage(ctx, req)
		if err != nil {
			c.logger.Warn("Failed to append message to conversation", "error", err, "conversationID", conversationID, "messageID", messageID)
			// 不阻断流程，继续处理下一条消息
			continue
		}

		// 标记消息已保存
		savedMessageIDs[messageID] = true
		savedCount++
		if wasNew {
			newMessageCount++
		}
	}

	// 保存 WorkflowContext（每次保存消息时也更新 workflow context）
	if sess.Data != nil {
		workflowContext := make(map[string]any)
		if sess.WorkflowMeta != nil {
			workflowMetaMap := map[string]any{
				"workflow": sess.WorkflowMeta.Workflow,
				"phase":    sess.WorkflowMeta.Phase,
			}
			// 添加 Actions 字段
			if len(sess.WorkflowMeta.Actions) > 0 {
				actionsJSON, err := json.Marshal(sess.WorkflowMeta.Actions)
				if err != nil {
					c.logger.Warn("Failed to marshal workflow actions to JSON", "error", err)
				} else {
					workflowMetaMap["actions"] = string(actionsJSON)
				}
			}
			workflowContext["workflowMeta"] = workflowMetaMap
		}
		workflowContext["data"] = sess.Data
		workflowContext["state"] = sess.State
		workflowContext["stateDesc"] = sess.StateDesc
		workflowContext["agentType"] = sess.AgentType

		updateReq := context_model.WorkflowContextUpdateRequest{
			ConversationID: conversationID,
			Context:        workflowContext,
		}
		if err := c.contextClient.UpdateWorkflowContext(ctx, updateReq); err != nil {
			c.logger.Warn("Failed to update workflow context", "error", err)
		}
	}

	// 更新 session.Data 中的消息计数和已保存的消息ID集合
	if sess.Data == nil {
		sess.Data = make(map[string]any)
	}

	// 更新已保存的消息ID集合
	savedIDsMap := make(map[string]any)
	for msgID := range savedMessageIDs {
		savedIDsMap[msgID] = true
	}
	sess.Data["savedMessageIDs"] = savedIDsMap

	// 更新消息计数（只计算新插入的消息数量）
	currentCount := len(messages)
	lastSavedCount := 0
	if savedCount, ok := sess.Data[DataKeyMessageCount].(int); ok {
		lastSavedCount = savedCount
	}
	sess.Data[DataKeyMessageCount] = lastSavedCount + newMessageCount

	c.logger.Debug("Updated saved message IDs", "conversationID", conversationID, "savedCount", savedCount, "newMessageCount", newMessageCount, "totalMessages", currentCount, "savedMessageIDs", len(savedMessageIDs))

	// 保存更新后的 session
	_, err = c.sessionService.Update(ctx, sessionID, func(s *session.Session) error {
		s.Data = sess.Data
		return nil
	})
	if err != nil {
		c.logger.Warn("Failed to update session data", "error", err, "sessionID", sessionID)
	}

	// 调用管理服务的 MCP 工具保存对话记录基本信息
	// 统一入口：只有用户真正开始对话时才创建记录
	// 避免在只有助手欢迎消息时创建记录（用户点击助手时会自动发送欢迎消息）
	hasUserMessage := false
	for _, msg := range messages {
		if msg.Role == session.RoleUser {
			hasUserMessage = true
			break
		}
	}

	if !hasUserMessage {
		c.logger.Debug("Skipping conversation record creation: no user messages yet",
			"sessionID", sessionID, "messageCount", len(messages), "conversationID", conversationID)
		return nil
	}

	if c.toolBus != nil && len(messages) > 0 {
		// 获取最后一条消息的时间
		lastMessageAt := messages[len(messages)-1].Timestamp

		// 从 session 中提取信息
		workflowType := ""
		if sess.WorkflowMeta != nil {
			workflowType = sess.WorkflowMeta.Workflow
		}

		// 构建 meta（使用统一的提取方法从 Session 提取排班相关信息）
		meta := c.extractScheduleMetaFromSession(sess)

		// 记录meta信息（只在有内容时记录）
		if len(meta) > 0 {
			c.logger.Debug("Preparing to call management service with meta",
				"conversationID", conversationID,
				"metaKeys", func() []string {
					keys := make([]string, 0, len(meta))
					for k := range meta {
						keys = append(keys, k)
					}
					return keys
				}(),
				"scheduleStartDate", meta["scheduleStartDate"],
				"scheduleEndDate", meta["scheduleEndDate"],
				"scheduleStatus", meta["scheduleStatus"],
				"scheduleId", meta["scheduleId"])
		}

		// 调用管理服务的 MCP 工具
		// 使用实际保存的消息数量（只计算新插入的消息）
		toolInput := map[string]any{
			"conversationId": conversationID,
			"orgId":          sess.OrgID,
			"userId":         sess.UserID,
			"workflowType":   workflowType,
			"lastMessageAt":  lastMessageAt.Format(time.RFC3339),
			"messageCount":   lastSavedCount + newMessageCount, // 只计算新插入的消息数
			"meta":           meta,
		}

		resultBytes, err := c.toolBus.Execute(ctx, "management.conversation.create_or_update", toolInput)
		if err != nil {
			c.logger.Warn("Failed to call management service MCP tool", "error", err, "conversationID", conversationID)
			// 不阻断流程，只记录警告
		} else {
			c.logger.Debug("Successfully called management service MCP tool", "conversationID", conversationID, "result", string(resultBytes))
		}
	}

	// 每次保存时都更新 conversation meta，确保排班信息实时同步
	// 这可以确保即使 meta 在 toolInput 中已经传递，也会通过 UpdateConversationMeta 更新到 context-server
	c.updateConversationMetaFromSession(ctx, conversationID, sess)

	return nil
}

// GetConversationHistory 获取会话历史消息
func (c *conversationManager) GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]session.Message, error) {
	// 获取 session
	sess, err := c.sessionService.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session failed: %w", err)
	}

	// 获取 conversation ID
	conversationID, ok := sess.Data[DataKeyConversationID].(string)
	if !ok || conversationID == "" {
		// 没有 conversation ID，返回空列表
		return []session.Message{}, nil
	}

	// 调用 context client 获取历史
	req := context_model.ConversationHistoryRequest{
		ID:    conversationID,
		Limit: limit,
	}

	resp, err := c.contextClient.GetConversationHistory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get conversation history failed: %w", err)
	}

	// 转换 conversation messages 为 session messages
	messages := make([]session.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		timestamp, _ := time.Parse(time.RFC3339, msg.Timestamp)
		messages = append(messages, session.Message{
			Role:      session.MessageRole(msg.Role),
			Content:   msg.Content,
			Timestamp: timestamp,
		})
	}

	return messages, nil
}

// ListConversations 列出用户的会话列表
func (c *conversationManager) ListConversations(ctx context.Context, orgID, userID string, limit int) ([]*service.ConversationSummary, error) {
	// 构建 Meta 查询条件
	metaFilters := make(map[string]any)
	metaFilters["orgId"] = orgID
	metaFilters["userId"] = userID

	req := context_model.ConversationListRequest{
		MetaFilters: metaFilters,
		Limit:       limit,
		Offset:      0,
	}

	resp, err := c.contextClient.ListConversations(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list conversations failed: %w", err)
	}

	// 转换 conversation records 为 conversation summaries
	summaries := make([]*service.ConversationSummary, 0, len(resp.Conversations))
	for _, conv := range resp.Conversations {
		// 从 meta 中提取信息
		title := "新会话"
		lastMessageAt := ""
		messageCount := 0

		if meta, ok := conv.Meta["title"].(string); ok {
			title = meta
		}
		if lastMsgAt, ok := conv.Meta["lastMessageAt"].(string); ok {
			lastMessageAt = lastMsgAt
		}
		if msgCount, ok := conv.Meta["messageCount"].(float64); ok {
			messageCount = int(msgCount)
		}

		summaries = append(summaries, &service.ConversationSummary{
			ID:            conv.ID,
			Title:         title,
			LastMessageAt: lastMessageAt,
			MessageCount:  messageCount,
			OrgID:         orgID,
			UserID:        userID,
		})
	}

	return summaries, nil
}

// CompressConversation 压缩会话（总结旧消息）
func (c *conversationManager) CompressConversation(ctx context.Context, sessionID string) error {
	// TODO: 实现压缩逻辑
	// 1. 获取会话的所有消息
	// 2. 分离最近 N 条（保留）和更早的消息（压缩）
	// 3. 对旧消息调用 AI 生成摘要
	// 4. 将摘要保存为系统消息
	// 5. 更新上下文服务中的会话记录

	c.logger.Info("CompressConversation not implemented yet", "sessionID", sessionID)
	return nil
}

// LoadConversation 加载指定会话到当前 session（包括 WorkflowContext）
func (c *conversationManager) LoadConversation(ctx context.Context, sessionID, conversationID string) error {
	// 先尝试获取 conversation 历史（如果 conversation 不存在，这里会先报错）
	req := context_model.ConversationHistoryRequest{
		ID:    conversationID,
		Limit: 0, // 获取所有消息
	}

	resp, err := c.contextClient.GetConversationHistory(ctx, req)
	if err != nil {
		// 检查是否是 conversation 不存在的错误
		errStr := err.Error()
		if strings.Contains(errStr, "conversation not found") || strings.Contains(errStr, "record not found") {
			c.logger.Warn("Conversation not found in context-server", "conversationID", conversationID, "error", err)
			return fmt.Errorf("对话记录在上下文中不存在，可能已被删除或从未保存: %w", err)
		}
		return fmt.Errorf("get conversation history failed: %w", err)
	}

	// 获取 WorkflowContext（如果 conversation 存在但没有 workflow context，这是可选的）
	workflowContextReq := context_model.WorkflowContextGetRequest{
		ConversationID: conversationID,
	}
	workflowContextResp, err := c.contextClient.GetWorkflowContext(ctx, workflowContextReq)
	if err != nil {
		// WorkflowContext 不存在不是致命错误，记录警告但继续加载消息
		c.logger.Warn("WorkflowContext not found, loading conversation without context", "conversationID", conversationID, "error", err)
		workflowContextResp = &context_model.WorkflowContextGetResponse{
			Context: nil,
		}
	}

	// 转换 conversation messages 为 session messages，恢复消息的原始ID
	messages := make([]session.Message, 0, len(resp.Messages))
	savedMessageIDs := make(map[string]bool)
	for _, msg := range resp.Messages {
		timestamp, _ := time.Parse(time.RFC3339, msg.Timestamp)

		// 恢复消息的原始ID（如果存在）
		messageID := msg.MessageID
		if messageID == "" {
			// 对于没有 messageID 的旧消息，生成一个临时ID（向后兼容）
			// 使用内容+时间戳+角色生成唯一标识
			tempMsg := session.Message{
				Role:      session.MessageRole(msg.Role),
				Content:   msg.Content,
				Timestamp: timestamp,
			}
			messageID = c.generateMessageID(tempMsg)
		}

		// 从 Metadata 中恢复 Actions
		actions := []session.WorkflowAction{} // 初始化为空切片（向后兼容）
		if msg.Metadata != nil {
			if actionsRaw, ok := msg.Metadata["actions"]; ok {
				// actions 可能是 string 类型（JSON 字符串）或 []byte 类型
				var actionsJSON []byte
				switch v := actionsRaw.(type) {
				case string:
					actionsJSON = []byte(v)
				case []byte:
					actionsJSON = v
				default:
					// 尝试直接 unmarshal（可能是已经解析的 JSON）
					actionsJSON, _ = json.Marshal(v)
				}

				if len(actionsJSON) > 0 {
					if err := json.Unmarshal(actionsJSON, &actions); err != nil {
						c.logger.Warn("Failed to unmarshal actions from metadata", "error", err, "messageID", messageID)
						// 反序列化失败时，Actions 保持为空数组（向后兼容）
						actions = []session.WorkflowAction{}
					}
				}
			}
		}

		messages = append(messages, session.Message{
			ID:        messageID,
			Role:      session.MessageRole(msg.Role),
			Content:   msg.Content,
			Timestamp: timestamp,
			Actions:   actions, // 恢复 Actions
		})

		// 记录已保存的消息ID
		savedMessageIDs[messageID] = true
	}

	// 更新 session（恢复 WorkflowContext）
	_, err = c.sessionService.Update(ctx, sessionID, func(s *session.Session) error {
		s.Messages = messages
		if s.Data == nil {
			s.Data = make(map[string]any)
		}
		s.Data[DataKeyConversationID] = conversationID
		s.Data[DataKeyMessageCount] = len(messages)

		// 保存已保存的消息ID集合（用于去重）
		savedIDsMap := make(map[string]any)
		for msgID := range savedMessageIDs {
			savedIDsMap[msgID] = true
		}
		s.Data["savedMessageIDs"] = savedIDsMap

		// 恢复 WorkflowContext
		if workflowContextResp.Context != nil {
			// 恢复 workflowMeta
			if workflowMetaRaw, ok := workflowContextResp.Context["workflowMeta"].(map[string]any); ok {
				if s.WorkflowMeta == nil {
					s.WorkflowMeta = &session.WorkflowMeta{}
				}
				if workflow, ok := workflowMetaRaw["workflow"].(string); ok {
					s.WorkflowMeta.Workflow = workflow
				}
				if phase, ok := workflowMetaRaw["phase"].(string); ok {
					s.WorkflowMeta.Phase = session.WorkflowState(phase)
				}
				// 恢复 Actions
				if actionsRaw, ok := workflowMetaRaw["actions"]; ok {
					var actionsJSON []byte
					switch v := actionsRaw.(type) {
					case string:
						actionsJSON = []byte(v)
					case []byte:
						actionsJSON = v
					default:
						actionsJSON, _ = json.Marshal(v)
					}
					if len(actionsJSON) > 0 {
						if err := json.Unmarshal(actionsJSON, &s.WorkflowMeta.Actions); err != nil {
							c.logger.Warn("Failed to unmarshal workflow actions", "error", err)
							s.WorkflowMeta.Actions = []session.WorkflowAction{}
						}
					}
				}
			}

			// 恢复 data（合并，保留 conversationID）
			if dataRaw, ok := workflowContextResp.Context["data"].(map[string]any); ok {
				for k, v := range dataRaw {
					if k != DataKeyConversationID { // 保留当前的 conversationID
						s.Data[k] = v
					}
				}
			}

			// 恢复 state
			if state, ok := workflowContextResp.Context["state"].(string); ok {
				s.State = session.SessionState(state)
			}
			if stateDesc, ok := workflowContextResp.Context["stateDesc"].(string); ok {
				s.StateDesc = stateDesc
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update session failed: %w", err)
	}

	return nil
}

// getOrCreateConversationID 获取或创建 conversation ID
func (c *conversationManager) getOrCreateConversationID(ctx context.Context, sess *session.Session) (string, error) {
	// 重新获取最新的 session，确保获取到最新的 conversationID（避免并发问题）
	latestSess, err := c.sessionService.Get(ctx, sess.ID)
	if err != nil {
		// 如果获取失败，使用传入的 session
		latestSess = sess
	} else {
		sess = latestSess
	}

	// 检查 session.Data 中是否已有 conversation ID
	if sess.Data != nil {
		if conversationID, ok := sess.Data[DataKeyConversationID].(string); ok && conversationID != "" {
			// 如果已有 conversation ID，更新 Meta（排班信息可能在过程中变化）
			c.updateConversationMetaFromSession(ctx, conversationID, sess)
			return conversationID, nil
		}
	}

	// 从 WorkflowMeta 提取 workflowType
	workflowType := ""
	if sess.WorkflowMeta != nil {
		workflowType = sess.WorkflowMeta.Workflow // 例如: "schedule.create", "schedule.adjust"
	}

	// 从 Session.Data 提取排班相关信息
	meta := make(map[string]any)
	meta["orgId"] = sess.OrgID
	meta["userId"] = sess.UserID
	meta["agentType"] = sess.AgentType
	meta["workflowType"] = workflowType
	meta["sessionId"] = sess.ID

	// 提取排班周期信息（如果存在）
	// 注意：ScheduleCreateContext 可能是指针类型或直接类型
	if scheduleCtxRaw, ok := sess.Data[d_model.DataKeyScheduleCreateContext]; ok {
		var startDate, endDate string
		var shiftIDs []string

		// 尝试多种类型转换
		switch ctx := scheduleCtxRaw.(type) {
		case *d_model.ScheduleCreateContext:
			// 指针类型
			if ctx.StartDate != "" {
				startDate = ctx.StartDate
			}
			if ctx.EndDate != "" {
				endDate = ctx.EndDate
			}
			if len(ctx.SelectedShifts) > 0 {
				shiftIDs = make([]string, 0, len(ctx.SelectedShifts))
				for _, shift := range ctx.SelectedShifts {
					shiftIDs = append(shiftIDs, shift.ID)
				}
			}
		case map[string]any:
			// map 类型（JSON 反序列化后）
			if sd, ok := ctx["startDate"].(string); ok {
				startDate = sd
			}
			if ed, ok := ctx["endDate"].(string); ok {
				endDate = ed
			}
			if shifts, ok := ctx["selectedShifts"].([]any); ok {
				shiftIDs = extractShiftIDs(shifts)
			}
		default:
			// 尝试通过 JSON 序列化/反序列化处理
			// 如果是指针类型，可能需要特殊处理
			c.logger.Debug("ScheduleCreateContext type not recognized", "type", fmt.Sprintf("%T", ctx))
		}

		if startDate != "" {
			meta["scheduleStartDate"] = startDate
		}
		if endDate != "" {
			meta["scheduleEndDate"] = endDate
		}
		if len(shiftIDs) > 0 {
			meta["scheduleShiftIds"] = shiftIDs
		}
	}

	// 创建新的 conversation
	req := context_model.ConversationNewRequest{
		Meta: meta,
	}

	resp, err := c.contextClient.CreateConversation(ctx, req)
	if err != nil {
		return "", fmt.Errorf("create conversation failed: %w", err)
	}

	// 保存 conversation ID 到 session
	if sess.Data == nil {
		sess.Data = make(map[string]any)
	}
	sess.Data[DataKeyConversationID] = resp.ID

	// 同时保存完整的 workflow context
	workflowContext := make(map[string]any)
	if sess.WorkflowMeta != nil {
		workflowContext["workflowMeta"] = map[string]any{
			"workflow": sess.WorkflowMeta.Workflow,
			"phase":    sess.WorkflowMeta.Phase,
		}
	}
	workflowContext["data"] = sess.Data
	workflowContext["state"] = sess.State
	workflowContext["stateDesc"] = sess.StateDesc
	workflowContext["agentType"] = sess.AgentType

	updateReq := context_model.WorkflowContextUpdateRequest{
		ConversationID: resp.ID,
		Context:        workflowContext,
	}
	if err := c.contextClient.UpdateWorkflowContext(ctx, updateReq); err != nil {
		c.logger.Warn("Failed to save workflow context", "error", err)
	}

	// 更新 session
	_, err = c.sessionService.Update(ctx, sess.ID, func(s *session.Session) error {
		s.Data = sess.Data
		return nil
	})
	if err != nil {
		c.logger.Warn("Failed to save conversation ID to session", "error", err, "sessionID", sess.ID, "conversationID", resp.ID)
	}

	return resp.ID, nil
}

// extractScheduleMetaFromSession 从 Session 提取排班相关信息到 meta map
// 统一的排班信息提取方法，供 SaveConversation 和 updateConversationMetaFromSession 使用
func (c *conversationManager) extractScheduleMetaFromSession(sess *session.Session) map[string]any {
	meta := make(map[string]any)

	if sess.Data == nil {
		return meta
	}

	// 从 CreateV2Context 提取排班周期信息（schedule_v2.create 工作流）
	if createV2CtxRaw, ok := sess.Data["create_v2_context"]; ok {
		var startDate, endDate string
		var shiftIDs []string

		// 如果已经从 ScheduleCreateContext 提取了，跳过
		if _, hasStartDate := meta["scheduleStartDate"]; !hasStartDate {
			var ctxMap map[string]any

			switch ctx := createV2CtxRaw.(type) {
			case map[string]any:
				// map 类型（JSON 反序列化后）
				ctxMap = ctx
			default:
				// 尝试通过JSON序列化提取（结构体类型）
				ctxBytes, err := json.Marshal(ctx)
				if err != nil {
					c.logger.Warn("Failed to marshal CreateV2Context to JSON",
						"sessionID", sess.ID,
						"error", err,
						"ctxType", fmt.Sprintf("%T", ctx))
				} else {
					if err := json.Unmarshal(ctxBytes, &ctxMap); err != nil {
						c.logger.Warn("Failed to unmarshal CreateV2Context JSON",
							"sessionID", sess.ID,
							"error", err)
					}
				}
			}

			// 从 ctxMap 中提取字段
			if ctxMap != nil {
				// 提取 startDate
				if sd, ok := ctxMap["startDate"].(string); ok && sd != "" {
					startDate = sd
				}

				// 提取 endDate
				if ed, ok := ctxMap["endDate"].(string); ok && ed != "" {
					endDate = ed
				}

				// 提取 selectedShifts
				if shifts, ok := ctxMap["selectedShifts"].([]any); ok {
					shiftIDs = extractShiftIDs(shifts)
				}

				// 提取 savedScheduleId
				if savedID, ok := ctxMap["savedScheduleId"].(string); ok && savedID != "" {
					if _, exists := meta["scheduleId"]; !exists {
						meta["scheduleId"] = savedID
					}
				}
			}

			if startDate != "" {
				meta["scheduleStartDate"] = startDate
			}
			if endDate != "" {
				meta["scheduleEndDate"] = endDate
			}
			if len(shiftIDs) > 0 {
				meta["scheduleShiftIds"] = shiftIDs
			}
		}
	}

	// 从 save_result 中提取排班ID和状态
	if saveResultRaw, ok := sess.Data["save_result"]; ok {
		if saveResult, ok := saveResultRaw.(map[string]any); ok {
			// 提取保存成功的记录数，用于判断状态
			if upserted, ok := saveResult["upserted"].(int); ok && upserted > 0 {
				meta["scheduleStatus"] = "completed"
			} else if failed, ok := saveResult["failed"].(int); ok && failed > 0 {
				meta["scheduleStatus"] = "failed"
			}

			// 如果有 details，尝试提取第一个成功的 schedule ID
			if details, ok := saveResult["details"].([]any); ok && len(details) > 0 {
				for _, detailRaw := range details {
					if detail, ok := detailRaw.(map[string]any); ok {
						if success, ok := detail["success"].(bool); ok && success {
							if entry, ok := detail["entry"].(map[string]any); ok {
								if id, ok := entry["id"].(string); ok && id != "" {
									// 使用第一个成功的 schedule ID（如果有多个，可以考虑合并）
									if _, exists := meta["scheduleId"]; !exists {
										meta["scheduleId"] = id
									}
								}
							}
						}
					}
				}
			}
		} else if result, ok := saveResultRaw.(*d_model.BatchUpsertResult); ok {
			// 处理 BatchUpsertResult 类型
			if result.Upserted > 0 {
				meta["scheduleStatus"] = "completed"
			} else if result.Failed > 0 {
				meta["scheduleStatus"] = "failed"
			}

			// 从 Details 中提取 schedule ID
			if len(result.Details) > 0 {
				for _, detail := range result.Details {
					if detail.Success && detail.Entry != nil && detail.Entry.ID != "" {
						if _, exists := meta["scheduleId"]; !exists {
							meta["scheduleId"] = detail.Entry.ID
						}
					}
				}
			}
		}
	}

	// 如果还没有设置状态，从 Session.State 或 WorkflowMeta.Phase 推断状态
	// 优先级：save_result 状态 > Session.State > WorkflowMeta.Phase
	if _, hasStatus := meta["scheduleStatus"]; !hasStatus {
		// 从 Session.State 推断
		switch sess.State {
		case session.StateProcessing, session.StateWaiting:
			meta["scheduleStatus"] = "in_progress"
		case session.StateCompleted:
			meta["scheduleStatus"] = "completed"
		case session.StateFailed:
			meta["scheduleStatus"] = "failed"
		case session.StateIdle:
			// 空闲状态：如果有 WorkflowMeta 且在工作流中，则推断为进行中
			// 否则不设置状态（可能是刚创建还未开始）
			if sess.WorkflowMeta != nil && sess.WorkflowMeta.Phase != "" {
				phaseStr := string(sess.WorkflowMeta.Phase)
				if phaseStr == string(engine.StateCompleted) {
					meta["scheduleStatus"] = "completed"
				} else if phaseStr == string(engine.StateFailed) {
					meta["scheduleStatus"] = "failed"
				} else if phaseStr == string(engine.StateCancelled) {
					meta["scheduleStatus"] = "cancelled"
				} else {
					// 其他阶段都认为是进行中（即使Session.State是Idle，只要工作流在运行就是进行中）
					meta["scheduleStatus"] = "in_progress"
				}
			}
		default:
			// 未知状态，如果 WorkflowMeta 存在且在工作流中，则推断为进行中
			if sess.WorkflowMeta != nil && sess.WorkflowMeta.Phase != "" {
				// 如果工作流阶段不为空，说明工作流在进行中
				// 判断是否为终态（WorkflowState 和 engine.State 是同一类型）
				phaseStr := string(sess.WorkflowMeta.Phase)
				if phaseStr == string(engine.StateCompleted) {
					meta["scheduleStatus"] = "completed"
				} else if phaseStr == string(engine.StateFailed) {
					meta["scheduleStatus"] = "failed"
				} else if phaseStr == string(engine.StateCancelled) {
					meta["scheduleStatus"] = "cancelled"
				} else {
					// 其他阶段都认为是进行中
					meta["scheduleStatus"] = "in_progress"
				}
			}
		}
	}

	// 只在有实际内容时记录Debug日志，减少日志噪音
	if len(meta) > 0 {
		c.logger.Debug("Extracted schedule meta from session",
			"sessionID", sess.ID,
			"metaKeys", func() []string {
				keys := make([]string, 0, len(meta))
				for k := range meta {
					keys = append(keys, k)
				}
				return keys
			}(),
			"scheduleStartDate", meta["scheduleStartDate"],
			"scheduleEndDate", meta["scheduleEndDate"],
			"scheduleStatus", meta["scheduleStatus"],
			"scheduleId", meta["scheduleId"])
	}

	return meta
}

// updateConversationMetaFromSession 从 Session 更新 Conversation.Meta
func (c *conversationManager) updateConversationMetaFromSession(ctx context.Context, conversationID string, sess *session.Session) {
	// 使用统一的提取方法获取排班信息
	metaUpdates := c.extractScheduleMetaFromSession(sess)

	// 更新工作流状态
	if sess.WorkflowMeta != nil {
		metaUpdates["workflowPhase"] = string(sess.WorkflowMeta.Phase)
	}

	if len(metaUpdates) > 0 {
		if err := c.contextClient.UpdateConversationMeta(ctx, conversationID, metaUpdates); err != nil {
			c.logger.Warn("Failed to update conversation meta", "error", err)
		}
	}
}

// extractShiftIDs 从 selectedShifts 中提取 shift IDs
func extractShiftIDs(shifts []any) []string {
	shiftIDs := make([]string, 0, len(shifts))
	for _, shiftRaw := range shifts {
		switch shift := shiftRaw.(type) {
		case map[string]any:
			if shiftID, ok := shift["id"].(string); ok {
				shiftIDs = append(shiftIDs, shiftID)
			} else if shiftID, ok := shift["ID"].(string); ok {
				shiftIDs = append(shiftIDs, shiftID)
			}
		}
	}
	return shiftIDs
}

// generateMessageID 为没有ID的消息生成唯一标识（向后兼容）
// 基于内容+时间戳+角色生成SHA256哈希
func (c *conversationManager) generateMessageID(msg session.Message) string {
	// 使用内容、时间戳和角色生成唯一标识
	data := fmt.Sprintf("%s|%s|%s", msg.Content, msg.Timestamp.Format(time.RFC3339Nano), string(msg.Role))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:32] // 使用前32个字符作为ID
}
