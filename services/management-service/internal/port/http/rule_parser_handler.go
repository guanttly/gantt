package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/service"
)

// ParseRule 解析语义化规则
func (h *HTTPHandler) ParseRule(w http.ResponseWriter, r *http.Request) {
	var req service.ParseRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 兼容处理：ruleText 或 ruleDescription 必须提供
	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if req.RuleText == "" && req.RuleDescription == "" {
		RespondBadRequest(w, "ruleText or ruleDescription is required")
		return
	}

	parserService := h.container.GetRuleParserService()
	if parserService == nil {
		RespondInternalError(w, "Rule parser service not available")
		return
	}

	result, err := parserService.ParseRule(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to parse rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, result)
}

// BatchParse 批量解析规则
func (h *HTTPHandler) BatchParse(w http.ResponseWriter, r *http.Request) {
	var req service.BatchParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if len(req.RuleTexts) == 0 {
		RespondBadRequest(w, "ruleTexts is required and cannot be empty")
		return
	}

	parserService := h.container.GetRuleParserService()
	if parserService == nil {
		RespondInternalError(w, "Rule parser service not available")
		return
	}

	result, err := parserService.BatchParse(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to batch parse rules", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, result)
}
