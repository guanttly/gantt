package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type parseRulesBatchInput struct {
	Description  string                       `json:"description"`
	ShiftCatalog []ruleparse.ShiftCatalogItem `json:"shift_catalog,omitempty"`
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
