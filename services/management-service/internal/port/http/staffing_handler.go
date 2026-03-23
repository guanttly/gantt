package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateStaffingRuleRequest 创建计算规则请求
type CreateStaffingRuleRequest struct {
	ShiftID         string             `json:"shiftId" validate:"required"`
	ModalityRoomIDs []string           `json:"modalityRoomIds" validate:"required"`
	TimePeriodID    string             `json:"timePeriodId" validate:"required"`
	AvgReportLimit  int                `json:"avgReportLimit"`
	RoundingMode    model.RoundingMode `json:"roundingMode"` // ceil/floor
	Description     string             `json:"description"`
}

// UpdateStaffingRuleRequest 更新计算规则请求
type UpdateStaffingRuleRequest struct {
	ModalityRoomIDs []string           `json:"modalityRoomIds"`
	TimePeriodID    string             `json:"timePeriodId"`
	AvgReportLimit  int                `json:"avgReportLimit"`
	RoundingMode    model.RoundingMode `json:"roundingMode"`
	Description     string             `json:"description"`
	IsActive        *bool              `json:"isActive,omitempty"`
}

// CalculateStaffing 计算排班人数
func (h *HTTPHandler) CalculateStaffing(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		ShiftID string `json:"shiftId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.ShiftID == "" {
		RespondBadRequest(w, "shiftId is required")
		return
	}

	preview, err := h.container.StaffingCalculationService().CalculateStaffCount(r.Context(), orgID, req.ShiftID)
	if err != nil {
		h.logger.Error("Failed to calculate staffing", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, preview)
}

// ApplyStaffing 应用排班人数
func (h *HTTPHandler) ApplyStaffing(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req model.ApplyStaffCountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.ShiftID == "" {
		RespondBadRequest(w, "shiftId is required")
		return
	}

	result, err := h.container.StaffingCalculationService().ApplyStaffCount(r.Context(), orgID, &req)
	if err != nil {
		h.logger.Error("Failed to apply staffing", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, result)
}

// CreateStaffingRule 创建计算规则
func (h *HTTPHandler) CreateStaffingRule(w http.ResponseWriter, r *http.Request) {
	var req CreateStaffingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	rule := &model.ShiftStaffingRule{
		ShiftID:         req.ShiftID,
		ModalityRoomIDs: req.ModalityRoomIDs,
		TimePeriodID:    req.TimePeriodID,
		AvgReportLimit:  req.AvgReportLimit,
		RoundingMode:    req.RoundingMode,
		Description:     req.Description,
	}

	if err := h.container.StaffingCalculationService().CreateOrUpdateStaffingRule(r.Context(), rule); err != nil {
		h.logger.Error("Failed to create staffing rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, rule)
}

// GetStaffingRule 获取计算规则
func (h *HTTPHandler) GetStaffingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	rule, err := h.container.StaffingCalculationService().GetStaffingRuleByID(r.Context(), ruleID)
	if err != nil {
		h.logger.Error("Failed to get staffing rule", "error", err)
		RespondNotFound(w, "计算规则不存在")
		return
	}

	RespondSuccess(w, rule)
}

// UpdateStaffingRule 更新计算规则
func (h *HTTPHandler) UpdateStaffingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var req UpdateStaffingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 先获取现有规则
	existing, err := h.container.StaffingCalculationService().GetStaffingRuleByID(r.Context(), ruleID)
	if err != nil {
		RespondNotFound(w, "计算规则不存在")
		return
	}

	// 更新字段
	if len(req.ModalityRoomIDs) > 0 {
		existing.ModalityRoomIDs = req.ModalityRoomIDs
	}
	if req.TimePeriodID != "" {
		existing.TimePeriodID = req.TimePeriodID
	}
	existing.AvgReportLimit = req.AvgReportLimit
	if req.RoundingMode != "" {
		existing.RoundingMode = req.RoundingMode
	}
	existing.Description = req.Description
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.container.StaffingCalculationService().CreateOrUpdateStaffingRule(r.Context(), existing); err != nil {
		h.logger.Error("Failed to update staffing rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, existing)
}

// DeleteStaffingRule 删除计算规则
func (h *HTTPHandler) DeleteStaffingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.container.StaffingCalculationService().DeleteStaffingRule(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete staffing rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "删除成功"})
}

// ListStaffingRulesResponse 计算规则列表响应
type ListStaffingRulesResponse struct {
	Items []*model.ShiftStaffingRule `json:"items"`
	Total int                        `json:"total"`
}

// ListStaffingRules 查询所有计算规则
func (h *HTTPHandler) ListStaffingRules(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	rules, err := h.container.StaffingCalculationService().ListStaffingRules(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to list staffing rules", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	// 包装成分页格式返回
	response := ListStaffingRulesResponse{
		Items: rules,
		Total: len(rules),
	}
	RespondSuccess(w, response)
}

// GetWeeklyStaffConfig 获取班次周默认人数配置
func (h *HTTPHandler) GetWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	config, err := h.container.StaffingCalculationService().GetWeeklyStaffConfig(r.Context(), orgID, shiftID)
	if err != nil {
		h.logger.Error("Failed to get weekly staff config", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, config)
}

// SetWeeklyStaffConfig 设置班次周默认人数配置
func (h *HTTPHandler) SetWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	var req struct {
		WeeklyConfig []model.WeekdayStaff `json:"weeklyConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.StaffingCalculationService().SetWeeklyStaffConfig(r.Context(), shiftID, req.WeeklyConfig); err != nil {
		h.logger.Error("Failed to set weekly staff config", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "设置成功"})
}

// BatchGetWeeklyStaffConfig 批量获取多个班次的周人数配置
func (h *HTTPHandler) BatchGetWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	shiftIDsStr := r.URL.Query().Get("shiftIds")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if shiftIDsStr == "" {
		RespondBadRequest(w, "shiftIds is required")
		return
	}

	// 解析班次ID列表
	shiftIDs := strings.Split(shiftIDsStr, ",")
	if len(shiftIDs) == 0 {
		RespondBadRequest(w, "shiftIds is empty")
		return
	}

	// 批量获取周人数配置
	weeklyStaffMap, err := h.container.ShiftWeeklyStaffRepository().GetByShiftIDs(r.Context(), shiftIDs)
	if err != nil {
		h.logger.Error("Failed to batch get weekly staff configs", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	// 构建响应
	result := make(map[string]*model.WeeklyStaffConfig)
	for _, shiftID := range shiftIDs {
		weeklyStaffs := weeklyStaffMap[shiftID]
		// 构建完整的7天配置
		weeklyConfig := make([]model.WeekdayStaff, 7)
		weekdayMap := make(map[int]int)
		for _, ws := range weeklyStaffs {
			weekdayMap[ws.Weekday] = ws.StaffCount
		}
		for i := 0; i < 7; i++ {
			count := 0
			isCustom := false
			if c, ok := weekdayMap[i]; ok {
				count = c
				isCustom = true
			}
			weeklyConfig[i] = model.WeekdayStaff{
				Weekday:     i,
				WeekdayName: model.GetWeekdayName(i),
				StaffCount:  count,
				IsCustom:    isCustom,
			}
		}
		result[shiftID] = &model.WeeklyStaffConfig{
			ShiftID:      shiftID,
			WeeklyConfig: weeklyConfig,
		}
	}

	RespondSuccess(w, result)
}
