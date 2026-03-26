package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/ai/chat"
	"gantt-saas/internal/ai/quota"
	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"go.uber.org/zap"
)

// Handler AI 模块的 HTTP Handler。
type Handler struct {
	chatHandler *chat.Handler
	ruleParser  *ruleparse.Parser
	quotaMgr    *quota.Manager
	factory     *ai.Factory
	logger      *zap.Logger
}

// NewHandler 创建 AI HTTP Handler。
func NewHandler(chatHandler *chat.Handler, ruleParser *ruleparse.Parser, quotaMgr *quota.Manager, factory *ai.Factory, logger *zap.Logger) *Handler {
	return &Handler{
		chatHandler: chatHandler,
		ruleParser:  ruleParser,
		quotaMgr:    quotaMgr,
		factory:     factory,
		logger:      logger.Named("ai-handler"),
	}
}

// Chat POST /api/v1/ai/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var msg chat.UserMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	if msg.Content == "" {
		response.BadRequest(w, "message 不能为空")
		return
	}

	// 检查 AI 是否可用
	if !h.factory.HasProvider() {
		response.BadRequest(w, "AI 服务未启用")
		return
	}

	// 检查配额
	orgNodeID := tenant.GetOrgNodeID(r.Context())
	if orgNodeID != "" {
		provider, _ := h.factory.Default()
		if provider != nil {
			err := h.quotaMgr.CheckAndDeduct(r.Context(), orgNodeID, provider.Name(), ai.TokenUsage{TotalTokens: 0})
			if errors.Is(err, quota.ErrQuotaExceeded) {
				response.Error(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "AI 配额已用完")
				return
			}
		}
	}

	botResp, err := h.chatHandler.Handle(r.Context(), msg)
	if err != nil {
		h.logger.Error("chat failed", zap.Error(err))
		response.InternalError(w, "AI 对话失败")
		return
	}

	// 记录使用
	if orgNodeID != "" && botResp.Usage.TotalTokens > 0 {
		provider, _ := h.factory.Default()
		if provider != nil {
			_ = h.quotaMgr.RecordUsage(r.Context(), provider.Name(), "", "chat", botResp.Usage)
			_ = h.quotaMgr.CheckAndDeduct(r.Context(), orgNodeID, provider.Name(), botResp.Usage)
		}
	}

	response.OK(w, botResp)
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
