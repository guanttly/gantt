package http

import (
	"encoding/json"
	"net/http"

	domain_service "jusha/gantt/service/management/domain/service"
	"github.com/gorilla/mux"
)

// PreviewMigrationRequest 预览迁移请求
type PreviewMigrationRequest struct {
	OrgID string `json:"orgId" form:"orgId"`
}

// ExecuteMigrationRequest 执行迁移请求
type ExecuteMigrationRequest struct {
	OrgID        string   `json:"orgId" validate:"required"`
	RuleIDs      []string `json:"ruleIds" validate:"required"`
	AutoClassify bool     `json:"autoClassify"`
	FillDefaults bool     `json:"fillDefaults"`
	DryRun       bool     `json:"dryRun"`
}

// RollbackMigrationRequest 回滚迁移请求
type RollbackMigrationRequest struct {
	OrgID       string `json:"orgId" validate:"required"`
	MigrationID string `json:"migrationId" validate:"required"`
}

// GetMigrationStatusRequest 获取迁移状态请求
type GetMigrationStatusRequest struct {
	OrgID       string `json:"orgId" form:"orgId"`
	MigrationID string `json:"migrationId" form:"migrationId"`
}

// PreviewMigration 预览迁移
func (h *HTTPHandler) PreviewMigration(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 获取迁移服务
	migrationService := h.container.GetRuleMigrationService()
	if migrationService == nil {
		RespondInternalError(w, "Rule migration service not available")
		return
	}

	// 调用预览服务
	preview, err := migrationService.PreviewMigration(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to preview migration", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, preview)
}

// ExecuteMigration 执行迁移
func (h *HTTPHandler) ExecuteMigration(w http.ResponseWriter, r *http.Request) {
	var req ExecuteMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" || len(req.RuleIDs) == 0 {
		RespondBadRequest(w, "orgId and ruleIds are required")
		return
	}

	// 获取迁移服务
	migrationService := h.container.GetRuleMigrationService()
	if migrationService == nil {
		RespondInternalError(w, "Rule migration service not available")
		return
	}

	// 构建迁移计划
	plan := &domain_service.MigrationPlan{
		OrgID:        req.OrgID,
		RuleIDs:      req.RuleIDs,
		AutoClassify: req.AutoClassify,
		FillDefaults: req.FillDefaults,
		DryRun:       req.DryRun,
	}

	// 执行迁移
	result, err := migrationService.ExecuteMigration(r.Context(), req.OrgID, plan)
	if err != nil {
		h.logger.Error("Failed to execute migration", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, result)
}

// RollbackMigration 回滚迁移
func (h *HTTPHandler) RollbackMigration(w http.ResponseWriter, r *http.Request) {
	var req RollbackMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" || req.MigrationID == "" {
		RespondBadRequest(w, "orgId and migrationId are required")
		return
	}

	// 获取迁移服务
	migrationService := h.container.GetRuleMigrationService()
	if migrationService == nil {
		RespondInternalError(w, "Rule migration service not available")
		return
	}

	// 执行回滚
	if err := migrationService.RollbackMigration(r.Context(), req.OrgID, req.MigrationID); err != nil {
		h.logger.Error("Failed to rollback migration", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, map[string]string{
		"message": "Migration rolled back successfully",
	})
}

// GetMigrationStatus 获取迁移状态
func (h *HTTPHandler) GetMigrationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	migrationID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || migrationID == "" {
		RespondBadRequest(w, "orgId and migrationId are required")
		return
	}

	// 获取迁移服务
	migrationService := h.container.GetRuleMigrationService()
	if migrationService == nil {
		RespondInternalError(w, "Rule migration service not available")
		return
	}

	// 获取状态
	status, err := migrationService.GetMigrationStatus(r.Context(), orgID, migrationID)
	if err != nil {
		h.logger.Error("Failed to get migration status", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, status)
}
