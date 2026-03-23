package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"
	"github.com/gorilla/mux"
)

// GetRuleDependencies 获取规则依赖关系
func (h *HTTPHandler) GetRuleDependencies(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	ruleID := r.URL.Query().Get("ruleId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var dependencies []*model.RuleDependency
	var err error

	if ruleID != "" {
		// 获取特定规则的依赖关系
		dependencies, err = h.container.GetRuleDependencyRepo().GetByRuleID(r.Context(), orgID, ruleID)
	} else {
		// 获取组织的所有依赖关系
		dependencies, err = h.container.GetRuleDependencyRepo().GetByOrgID(r.Context(), orgID)
	}

	if err != nil {
		h.logger.Error("Failed to get rule dependencies", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, map[string]interface{}{
		"dependencies": dependencies,
		"count":        len(dependencies),
	})
}

// CreateRuleDependency 创建规则依赖关系
func (h *HTTPHandler) CreateRuleDependency(w http.ResponseWriter, r *http.Request) {
	var dependency model.RuleDependency
	if err := json.NewDecoder(r.Body).Decode(&dependency); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if dependency.OrgID == "" || dependency.DependentRuleID == "" || dependency.DependentOnRuleID == "" {
		RespondBadRequest(w, "orgId, dependentRuleId, and dependentOnRuleId are required")
		return
	}

	if err := h.container.GetRuleDependencyRepo().Create(r.Context(), &dependency); err != nil {
		h.logger.Error("Failed to create rule dependency", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondCreated(w, dependency)
}

// DeleteRuleDependency 删除规则依赖关系
func (h *HTTPHandler) DeleteRuleDependency(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dependencyID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || dependencyID == "" {
		RespondBadRequest(w, "orgId and dependencyId are required")
		return
	}

	if err := h.container.GetRuleDependencyRepo().Delete(r.Context(), orgID, dependencyID); err != nil {
		h.logger.Error("Failed to delete rule dependency", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondNoContent(w)
}

// GetRuleConflicts 获取规则冲突关系
func (h *HTTPHandler) GetRuleConflicts(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	ruleID := r.URL.Query().Get("ruleId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var conflicts []*model.RuleConflict
	var err error

	if ruleID != "" {
		// 获取特定规则的冲突关系
		conflicts, err = h.container.GetRuleConflictRepo().GetByRuleID(r.Context(), orgID, ruleID)
	} else {
		// 获取组织的所有冲突关系
		conflicts, err = h.container.GetRuleConflictRepo().GetByOrgID(r.Context(), orgID)
	}

	if err != nil {
		h.logger.Error("Failed to get rule conflicts", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, map[string]interface{}{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// CreateRuleConflict 创建规则冲突关系
func (h *HTTPHandler) CreateRuleConflict(w http.ResponseWriter, r *http.Request) {
	var conflict model.RuleConflict
	if err := json.NewDecoder(r.Body).Decode(&conflict); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if conflict.OrgID == "" || conflict.RuleID1 == "" || conflict.RuleID2 == "" {
		RespondBadRequest(w, "orgId, ruleId1, and ruleId2 are required")
		return
	}

	if err := h.container.GetRuleConflictRepo().Create(r.Context(), &conflict); err != nil {
		h.logger.Error("Failed to create rule conflict", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondCreated(w, conflict)
}

// DeleteRuleConflict 删除规则冲突关系
func (h *HTTPHandler) DeleteRuleConflict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conflictID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || conflictID == "" {
		RespondBadRequest(w, "orgId and conflictId are required")
		return
	}

	if err := h.container.GetRuleConflictRepo().Delete(r.Context(), orgID, conflictID); err != nil {
		h.logger.Error("Failed to delete rule conflict", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondNoContent(w)
}

// GetRuleStatistics 获取规则统计信息
func (h *HTTPHandler) GetRuleStatistics(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	statsService := h.container.GetRuleStatisticsService()
	if statsService == nil {
		RespondInternalError(w, "Rule statistics service not available")
		return
	}

	stats, err := statsService.GetRuleStatistics(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get rule statistics", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, stats)
}
