package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/ai/chat"
	"gantt-saas/internal/ai/quota"
	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler AI 模块的 HTTP Handler。
type Handler struct {
	chatHandler       *chat.Handler
	conversationStore *chat.ConversationStore
	ruleParser        *ruleparse.Parser
	quotaMgr          *quota.Manager
	factory           *ai.Factory
	logger            *zap.Logger
}

type parseRulesBatchInput struct {
	Description  string                       `json:"description"`
	ShiftCatalog []ruleparse.ShiftCatalogItem `json:"shift_catalog,omitempty"`
}

// NewHandler 创建 AI HTTP Handler。
func NewHandler(chatHandler *chat.Handler, conversationStore *chat.ConversationStore, ruleParser *ruleparse.Parser, quotaMgr *quota.Manager, factory *ai.Factory, logger *zap.Logger) *Handler {
	return &Handler{
		chatHandler:       chatHandler,
		conversationStore: conversationStore,
		ruleParser:        ruleParser,
		quotaMgr:          quotaMgr,
		factory:           factory,
		logger:            logger.Named("ai-handler"),
	}
}

type updateConversationInput struct {
	Title string `json:"title"`
}

// Chat POST /api/v1/ai/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	deadline := time.Now().Add(2 * time.Minute)
	_ = http.NewResponseController(w).SetWriteDeadline(deadline)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	var msg chat.UserMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	msg.Content = strings.TrimSpace(msg.Content)
	if msg.Content == "" {
		response.BadRequest(w, "message 不能为空")
		return
	}
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	msg.UserID = userID
	msg.OrgNodeID = orgNodeID

	// 检查 AI 是否可用
	if !h.factory.HasProvider() {
		response.BadRequest(w, "AI 服务未启用")
		return
	}

	// 检查配额
	if orgNodeID != "" {
		provider, _ := h.factory.Default()
		if provider != nil {
			err := h.quotaMgr.CheckAndDeduct(ctx, orgNodeID, provider.Name(), ai.TokenUsage{TotalTokens: 0})
			if errors.Is(err, quota.ErrQuotaExceeded) {
				response.Error(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "AI 配额已用完")
				return
			}
		}
	}

	botResp, err := h.chatHandler.Handle(ctx, msg)
	if err != nil {
		h.logger.Error("chat failed", zap.Error(err))
		response.InternalError(w, "AI 对话失败")
		return
	}

	// 记录使用
	if orgNodeID != "" && botResp.Usage.TotalTokens > 0 {
		provider, _ := h.factory.Default()
		if provider != nil {
			_ = h.quotaMgr.RecordUsage(ctx, provider.Name(), "", "chat", botResp.Usage)
			_ = h.quotaMgr.CheckAndDeduct(ctx, orgNodeID, provider.Name(), botResp.Usage)
		}
	}

	response.OK(w, botResp)
}

// ListConversations GET /api/v1/ai/conversations
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	if h.conversationStore == nil {
		response.InternalError(w, "会话存储未初始化")
		return
	}

	conversations, err := h.conversationStore.ListConversations(r.Context(), userID, orgNodeID)
	if err != nil {
		h.logger.Error("list conversations failed", zap.Error(err))
		response.InternalError(w, "获取会话列表失败")
		return
	}
	response.OK(w, conversations)
}

// CreateConversation POST /api/v1/ai/conversations
func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	if h.conversationStore == nil {
		response.InternalError(w, "会话存储未初始化")
		return
	}

	conversation, err := h.conversationStore.CreateConversation(r.Context(), userID, orgNodeID)
	if err != nil {
		h.logger.Error("create conversation failed", zap.Error(err))
		response.InternalError(w, "创建会话失败")
		return
	}
	response.Created(w, conversation)
}

// GetConversation GET /api/v1/ai/conversations/{conversationID}
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	if h.conversationStore == nil {
		response.InternalError(w, "会话存储未初始化")
		return
	}

	conversationID := chi.URLParam(r, "conversationID")
	detail, err := h.conversationStore.GetConversationDetail(r.Context(), conversationID, userID, orgNodeID)
	if err != nil {
		h.handleConversationStoreError(w, err, "获取会话详情失败")
		return
	}
	response.OK(w, detail)
}

// UpdateConversation PATCH /api/v1/ai/conversations/{conversationID}
func (h *Handler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	if h.conversationStore == nil {
		response.InternalError(w, "会话存储未初始化")
		return
	}

	var input updateConversationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		response.BadRequest(w, "title 不能为空")
		return
	}

	conversationID := chi.URLParam(r, "conversationID")
	conversation, err := h.conversationStore.UpdateConversationTitle(r.Context(), conversationID, userID, orgNodeID, input.Title)
	if err != nil {
		h.handleConversationStoreError(w, err, "更新会话失败")
		return
	}
	response.OK(w, conversation)
}

// DeleteConversation DELETE /api/v1/ai/conversations/{conversationID}
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID, orgNodeID, ok := h.currentActor(r)
	if !ok {
		response.Unauthorized(w, "未认证")
		return
	}
	if h.conversationStore == nil {
		response.InternalError(w, "会话存储未初始化")
		return
	}

	conversationID := chi.URLParam(r, "conversationID")
	if err := h.conversationStore.DeleteConversation(r.Context(), conversationID, userID, orgNodeID); err != nil {
		h.handleConversationStoreError(w, err, "删除会话失败")
		return
	}
	response.NoContent(w)
}

// ParseRule POST /api/v1/ai/parse-rule
func (h *Handler) ParseRule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	if input.Description == "" {
		response.BadRequest(w, "description 不能为空")
		return
	}

	if !h.factory.HasProvider() {
		response.BadRequest(w, "AI 服务未启用")
		return
	}

	cfg, err := h.ruleParser.Parse(r.Context(), input.Description)
	if err != nil {
		h.logger.Error("parse rule failed", zap.Error(err))
		response.InternalError(w, "规则解析失败")
		return
	}

	response.OK(w, cfg)
}

// ParseRulesBatch POST /api/v1/ai/parse-rules
func (h *Handler) ParseRulesBatch(w http.ResponseWriter, r *http.Request) {
	var input parseRulesBatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	if input.Description == "" {
		response.BadRequest(w, "description 不能为空")
		return
	}

	if !h.factory.HasProvider() {
		response.BadRequest(w, "AI 服务未启用")
		return
	}

	result, err := h.ruleParser.ParseBatch(r.Context(), input.Description, ruleparse.ParseOptions{ShiftCatalog: input.ShiftCatalog})
	if err != nil {
		h.logger.Error("parse rules batch failed", zap.Error(err))
		response.InternalError(w, "规则批量解析失败")
		return
	}

	response.OK(w, result)
}

// ParseRulesBatchStream POST /api/v1/ai/parse-rules-stream — SSE 流式批量解析规则。
func (h *Handler) ParseRulesBatchStream(w http.ResponseWriter, r *http.Request) {
	var input parseRulesBatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	if input.Description == "" {
		response.BadRequest(w, "description 不能为空")
		return
	}
	if !h.factory.HasProvider() {
		response.BadRequest(w, "AI 服务未启用")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.InternalError(w, "streaming not supported")
		return
	}

	// 延长 write deadline，SSE 流可能持续数分钟
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(5 * time.Minute))

	// 开启流式调用
	ch, err := h.ruleParser.ParseBatchStream(r.Context(), input.Description, ruleparse.ParseOptions{ShiftCatalog: input.ShiftCatalog})
	if err != nil {
		h.logger.Error("parse rules batch stream failed", zap.Error(err))
		response.InternalError(w, "规则批量解析失败")
		return
	}

	// 设置 SSE 头（在第一次写入之前）
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	var fullContent strings.Builder

	for chunk := range ch {
		if chunk.Done {
			break
		}
		fullContent.WriteString(chunk.Content)

		// 发送 reasoning（思考过程）或 chunk（最终内容）
		if chunk.Reasoning != "" {
			data, _ := json.Marshal(map[string]string{"reasoning": chunk.Reasoning})
			fmt.Fprintf(w, "event: reasoning\ndata: %s\n\n", data)
			flusher.Flush()
		}
		if chunk.Content != "" {
			data, _ := json.Marshal(map[string]string{"content": chunk.Content})
			fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}

	content := fullContent.String()
	if content == "" {
		h.logger.Error("parse rules batch stream: empty LLM response")
		errData, _ := json.Marshal(map[string]string{"message": "AI 未返回任何内容，请重试"})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
		flusher.Flush()
		return
	}

	// 流结束，解析完整 JSON
	result, err := h.ruleParser.ParseBatchFromContent(content)
	if err != nil {
		h.logger.Error("parse rules batch JSON failed", zap.Error(err))
		errData, _ := json.Marshal(map[string]string{"message": "AI 返回内容解析失败，请重试"})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
		flusher.Flush()
		return
	}

	// 发送结构化结果
	doneData, _ := json.Marshal(result)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", doneData)
	flusher.Flush()
}

func (h *Handler) currentActor(r *http.Request) (string, string, bool) {
	claims := auth.GetClaims(r.Context())
	if claims == nil || claims.UserID == "" {
		return "", "", false
	}
	orgNodeID := tenant.GetOrgNodeID(r.Context())
	if orgNodeID == "" {
		orgNodeID = claims.OrgNodeID
	}
	if orgNodeID == "" {
		return "", "", false
	}
	return claims.UserID, orgNodeID, true
}

func (h *Handler) handleConversationStoreError(w http.ResponseWriter, err error, fallbackMessage string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.NotFound(w, "会话不存在")
		return
	}
	h.logger.Error(fallbackMessage, zap.Error(err))
	response.InternalError(w, fallbackMessage)
}

// GetQuota GET /api/v1/ai/quota
func (h *Handler) GetQuota(w http.ResponseWriter, r *http.Request) {
	orgNodeID := tenant.GetOrgNodeID(r.Context())
	if orgNodeID == "" {
		response.BadRequest(w, "缺少组织节点信息")
		return
	}

	status, err := h.quotaMgr.GetQuotaStatus(r.Context(), orgNodeID)
	if err != nil {
		response.InternalError(w, "查询配额失败")
		return
	}

	response.OK(w, status)
}

// GetUsage GET /api/v1/ai/usage
func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgNodeID := tenant.GetOrgNodeID(r.Context())
	if orgNodeID == "" {
		response.BadRequest(w, "缺少组织节点信息")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	logs, total, err := h.quotaMgr.GetUsageLogs(r.Context(), orgNodeID, page, size)
	if err != nil {
		response.InternalError(w, "查询使用记录失败")
		return
	}

	response.Page(w, logs, total, page, size)
}
