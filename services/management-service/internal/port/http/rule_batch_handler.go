package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/service"
)

// BatchSaveRulesRequest 批量保存规则请求
type BatchSaveRulesRequest struct {
	OrgID        string                    `json:"orgId"`
	Rules        []*model.SchedulingRule   `json:"rules"`        // 直接保存的规则
	ParsedRules  []*service.ParsedRule     `json:"parsedRules"`  // 解析后的规则（V4）
	Dependencies []*service.RuleDependency `json:"dependencies"` // 依赖关系
	Conflicts    []*service.RuleConflict   `json:"conflicts"`    // 冲突关系
}

// BatchSaveRules 批量保存规则（V4）
func (h *HTTPHandler) BatchSaveRules(w http.ResponseWriter, r *http.Request) {
	var req BatchSaveRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 支持两种方式：直接 rules 或解析后的 parsedRules
	if len(req.Rules) == 0 && len(req.ParsedRules) == 0 {
		RespondBadRequest(w, "rules or parsedRules is required")
		return
	}

	// 如果是 parsedRules，调用 RuleParserService 保存
	if len(req.ParsedRules) > 0 {
		parserService := h.container.GetRuleParserService()
		if parserService == nil {
			RespondInternalError(w, "Rule parser service not available")
			return
		}

		savedRules, err := parserService.SaveParsedRules(r.Context(), req.OrgID, req.ParsedRules, req.Dependencies, req.Conflicts)
		if err != nil {
			h.logger.Error("Failed to save parsed rules", "error", err)
			RespondInternalError(w, "保存规则失败: "+err.Error())
			return
		}

		RespondOK(w, map[string]any{
			"created": len(savedRules),
			"failed":  0,
			"rules":   savedRules,
		})
		return
	}

	// 否则直接保存 rules
	ruleService := h.container.GetSchedulingRuleService()
	if ruleService == nil {
		RespondInternalError(w, "Rule service not available")
		return
	}

	// 批量创建规则
	created := 0
	failed := 0
	errors := make([]string, 0)

	for _, rule := range req.Rules {
		rule.OrgID = req.OrgID
		if err := ruleService.CreateRule(r.Context(), rule); err != nil {
			h.logger.Error("Failed to create rule in batch", "ruleName", rule.Name, "error", err)
			failed++
			errors = append(errors, rule.Name+": "+err.Error())
		} else {
			created++
		}
	}

	RespondOK(w, map[string]interface{}{
		"created": created,
		"failed":  failed,
		"errors":  errors,
	})
}

// OrganizeRulesRequest 组织规则请求
type OrganizeRulesRequest struct {
	OrgID string `json:"orgId" binding:"required"`
}

// OrganizeRules 组织规则（V4）
func (h *HTTPHandler) OrganizeRules(w http.ResponseWriter, r *http.Request) {
	var req OrganizeRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	organizerService := h.container.GetRuleOrganizerService()
	if organizerService == nil {
		RespondInternalError(w, "Rule organizer service not available")
		return
	}

	result, err := organizerService.OrganizeRules(r.Context(), req.OrgID)
	if err != nil {
		h.logger.Error("Failed to organize rules", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, result)
}
