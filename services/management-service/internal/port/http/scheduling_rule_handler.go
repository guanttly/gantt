package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	OrgID          string                 `json:"orgId" validate:"required"`
	Name           string                 `json:"name" validate:"required"`
	Description    string                 `json:"description"`
	RuleType       model.RuleType         `json:"ruleType" validate:"required"`
	ApplyScope     model.ApplyScope       `json:"applyScope" validate:"required"`
	TimeScope      model.TimeScope        `json:"timeScope" validate:"required"`
	RuleData       string                 `json:"ruleData"`
	MaxCount       *int                   `json:"maxCount,omitempty"`
	ConsecutiveMax *int                   `json:"consecutiveMax,omitempty"`
	IntervalDays   *int                   `json:"intervalDays,omitempty"`
	MinRestDays    *int                   `json:"minRestDays,omitempty"`
	Priority       int                    `json:"priority"`
	IsActive       bool                   `json:"isActive"`
	ValidFrom      *string                `json:"validFrom,omitempty"`
	ValidTo        *string                `json:"validTo,omitempty"`
	Associations   []RuleAssociationInput `json:"associations,omitempty"`
	// V4新增字段
	Category        string   `json:"category,omitempty"`
	SubCategory     string   `json:"subCategory,omitempty"`
	OriginalRuleID  string   `json:"originalRuleId,omitempty"`
	SourceType      string   `json:"sourceType,omitempty"`
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`
	Version         string   `json:"version,omitempty"`
	// V4.1新增字段：适用范围
	ApplyScopes []ApplyScopeInput `json:"applyScopes,omitempty"` // 适用范围列表
}

// ApplyScopeInput 适用范围输入（V4.1新增）
type ApplyScopeInput struct {
	ScopeType string `json:"scopeType" validate:"required"` // 范围类型: all/employee/group/exclude_employee/exclude_group
	ScopeID   string `json:"scopeId,omitempty"`             // 范围对象ID（当scopeType不为all时必填）
	ScopeName string `json:"scopeName,omitempty"`           // 范围对象名称（冗余，便于展示）
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	RuleType       *model.RuleType  `json:"ruleType"`
	TimeScope      *model.TimeScope `json:"timeScope"`
	RuleData       string           `json:"ruleData"`
	MaxCount       *int             `json:"maxCount,omitempty"`
	ConsecutiveMax *int             `json:"consecutiveMax,omitempty"`
	IntervalDays   *int             `json:"intervalDays,omitempty"`
	MinRestDays    *int             `json:"minRestDays,omitempty"`
	Priority       *int             `json:"priority"`
	IsActive       *bool            `json:"isActive"`
	ValidFrom      *string          `json:"validFrom,omitempty"`
	ValidTo        *string          `json:"validTo,omitempty"`
	// V4新增字段
	Category        *string  `json:"category,omitempty"`
	SubCategory     *string  `json:"subCategory,omitempty"`
	OriginalRuleID  *string  `json:"originalRuleId,omitempty"`
	SourceType      *string  `json:"sourceType,omitempty"`
	ParseConfidence *float64 `json:"parseConfidence,omitempty"`
	Version         *string  `json:"version,omitempty"`
	// V4.1新增字段：结构化的班次关系和适用范围
	Associations []RuleAssociationInput `json:"associations,omitempty"` // 班次关联列表
	ApplyScopes  []ApplyScopeInput      `json:"applyScopes,omitempty"`  // 适用范围列表
}

// ListRulesRequest 查询规则列表请求
type ListRulesRequest struct {
	PageRequest
	OrgID      string            `json:"orgId" form:"orgId"`
	RuleType   *model.RuleType   `json:"ruleType" form:"ruleType"`
	ApplyScope *model.ApplyScope `json:"applyScope" form:"applyScope"`
	TimeScope  *model.TimeScope  `json:"timeScope" form:"timeScope"`
	IsActive   *bool             `json:"isActive" form:"isActive"`
	Keyword    string            `json:"keyword" form:"keyword"`
	// V4新增筛选字段
	Category    *string `json:"category" form:"category"`
	SubCategory *string `json:"subCategory" form:"subCategory"`
	SourceType  *string `json:"sourceType" form:"sourceType"`
	Version     *string `json:"version" form:"version"`
}

// RuleAssociationInput 规则关联输入 (V4.1: 仅用于创建规则时的内部转换)
type RuleAssociationInput struct {
	AssociationType model.AssociationType `json:"associationType" validate:"required"`
	AssociationID   string                `json:"associationId" validate:"required"`
	Role            string                `json:"role,omitempty"` // V4新增：关联角色 target/source/reference
}

// RuleListResponse 规则列表响应（包含关联统计）
type RuleListResponse struct {
	*model.SchedulingRule
	AssociationCount int `json:"associationCount"` // 关联数量
	EmployeeCount    int `json:"employeeCount"`    // 关联的员工数量
	ShiftCount       int `json:"shiftCount"`       // 关联的班次数量
	GroupCount       int `json:"groupCount"`       // 关联的分组数量
}

// CreateRule 创建排班规则
func (h *HTTPHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" || req.Name == "" {
		RespondBadRequest(w, "orgId and name are required")
		return
	}

	// @deprecated V3: 规范化 V3 枚举值
	normalizedRuleType := normalizeRuleType(string(req.RuleType))
	normalizedApplyScope := normalizeApplyScope(string(req.ApplyScope))
	normalizedTimeScope := normalizeTimeScope(string(req.TimeScope))

	rule := &model.SchedulingRule{
		OrgID:          req.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		RuleType:       normalizedRuleType,
		ApplyScope:     normalizedApplyScope,
		TimeScope:      normalizedTimeScope,
		RuleData:       req.RuleData,
		MaxCount:       req.MaxCount,
		ConsecutiveMax: req.ConsecutiveMax,
		IntervalDays:   req.IntervalDays,
		MinRestDays:    req.MinRestDays,
		Priority:       req.Priority,
		IsActive:       req.IsActive,
		// V4新增字段
		Category:        req.Category,
		SubCategory:     req.SubCategory,
		OriginalRuleID:  req.OriginalRuleID,
		SourceType:      req.SourceType,
		ParseConfidence: req.ParseConfidence,
		Version:         req.Version,
	}
	// 如果未指定 Version，默认为 v4
	if rule.Version == "" {
		rule.Version = "v4"
	}
	// 如果未指定 SourceType，默认为 manual
	if rule.SourceType == "" {
		rule.SourceType = "manual"
	}

	// @deprecated V3: 为 V3 规则填充默认值
	FillV3Defaults(rule)

	if req.ValidFrom != nil {
		validFrom, err := parseDate(*req.ValidFrom)
		if err != nil {
			RespondBadRequest(w, "Invalid validFrom date format")
			return
		}
		rule.ValidFrom = &validFrom
	}
	if req.ValidTo != nil {
		validTo, err := parseDate(*req.ValidTo)
		if err != nil {
			RespondBadRequest(w, "Invalid validTo date format")
			return
		}
		rule.ValidTo = &validTo
	}

	if len(req.Associations) > 0 {
		rule.Associations = make([]model.RuleAssociation, len(req.Associations))
		for i, assoc := range req.Associations {
			rule.Associations[i] = model.RuleAssociation{
				AssociationType: assoc.AssociationType,
				AssociationID:   assoc.AssociationID,
				Role:            assoc.Role, // V4新增：关联角色
			}
			// 如果未指定 Role，默认为 target
			if rule.Associations[i].Role == "" {
				rule.Associations[i].Role = "target"
			}
		}
	}

	// V4.1新增：处理适用范围
	if len(req.ApplyScopes) > 0 {
		rule.ApplyScopes = make([]model.RuleApplyScope, len(req.ApplyScopes))
		for i, scope := range req.ApplyScopes {
			rule.ApplyScopes[i] = model.RuleApplyScope{
				ScopeType: scope.ScopeType,
				ScopeID:   scope.ScopeID,
				ScopeName: scope.ScopeName,
			}
		}
	} else {
		// 如果没有指定范围，默认为全局
		rule.ApplyScopes = []model.RuleApplyScope{
			{ScopeType: model.ScopeTypeAll},
		}
	}

	// V4.1新增：验证班次关联完整性
	if err := rule.ValidateShiftAssociations(); err != nil {
		RespondBadRequest(w, err.Error())
		return
	}

	if err := h.container.GetSchedulingRuleService().CreateRule(r.Context(), rule); err != nil {
		h.logger.Error("Failed to create rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondCreated(w, rule)
}

// UpdateRule 更新排班规则
func (h *HTTPHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || ruleID == "" {
		RespondBadRequest(w, "orgId and ruleId are required")
		return
	}

	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	rule, err := h.container.GetSchedulingRuleService().GetRule(r.Context(), orgID, ruleID)
	if err != nil {
		h.logger.Error("Failed to get rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}
	if rule == nil {
		RespondNotFound(w, "Rule not found")
		return
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.RuleType != nil {
		// @deprecated V3: 规范化 V3 枚举值
		normalizedRuleType := normalizeRuleType(string(*req.RuleType))
		rule.RuleType = normalizedRuleType
	}
	if req.TimeScope != nil {
		// @deprecated V3: 规范化 V3 枚举值
		normalizedTimeScope := normalizeTimeScope(string(*req.TimeScope))
		rule.TimeScope = normalizedTimeScope
	}
	if req.RuleData != "" {
		rule.RuleData = req.RuleData
	}
	if req.MaxCount != nil {
		rule.MaxCount = req.MaxCount
	}
	if req.ConsecutiveMax != nil {
		rule.ConsecutiveMax = req.ConsecutiveMax
	}
	if req.IntervalDays != nil {
		rule.IntervalDays = req.IntervalDays
	}
	if req.MinRestDays != nil {
		rule.MinRestDays = req.MinRestDays
	}

	// 若规则类型不使用数值参数，清除 DB 中可能残留的旧值，避免校验误拦
	effectiveType := rule.RuleType
	if req.RuleType != nil {
		effectiveType = normalizeRuleType(string(*req.RuleType))
	}
	if effectiveType != model.RuleTypeMaxCount {
		rule.MaxCount = nil
		rule.ConsecutiveMax = nil
	}
	if effectiveType != model.RuleTypePeriodic && effectiveType != model.RuleTypeMaxCount {
		rule.IntervalDays = nil
		rule.MinRestDays = nil
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if req.ValidFrom != nil {
		validFrom, err := parseDate(*req.ValidFrom)
		if err != nil {
			RespondBadRequest(w, "Invalid validFrom date format")
			return
		}
		rule.ValidFrom = &validFrom
	}
	if req.ValidTo != nil {
		validTo, err := parseDate(*req.ValidTo)
		if err != nil {
			RespondBadRequest(w, "Invalid validTo date format")
			return
		}
		rule.ValidTo = &validTo
	}

	// V4新增字段更新
	if req.Category != nil {
		rule.Category = *req.Category
	}
	if req.SubCategory != nil {
		rule.SubCategory = *req.SubCategory
	}
	if req.OriginalRuleID != nil {
		rule.OriginalRuleID = *req.OriginalRuleID
	}
	if req.SourceType != nil {
		rule.SourceType = *req.SourceType
	}
	if req.ParseConfidence != nil {
		rule.ParseConfidence = req.ParseConfidence
	}
	if req.Version != nil {
		rule.Version = *req.Version
	}

	// V4.1新增：处理适用范围
	if req.ApplyScopes != nil {
		rule.ApplyScopes = make([]model.RuleApplyScope, len(req.ApplyScopes))
		for i, scope := range req.ApplyScopes {
			rule.ApplyScopes[i] = model.RuleApplyScope{
				ScopeType: scope.ScopeType,
				ScopeID:   scope.ScopeID,
				ScopeName: scope.ScopeName,
			}
		}
	}

	// V4.1新增：处理班次关联（覆盖旧数据）
	if len(req.Associations) > 0 {
		rule.Associations = make([]model.RuleAssociation, len(req.Associations))
		for i, assoc := range req.Associations {
			rule.Associations[i] = model.RuleAssociation{
				AssociationType: assoc.AssociationType,
				AssociationID:   assoc.AssociationID,
				Role:            assoc.Role,
			}
			if rule.Associations[i].Role == "" {
				rule.Associations[i].Role = "target"
			}
		}
	}

	// V4.1新增：验证班次关联完整性
	if err := rule.ValidateShiftAssociations(); err != nil {
		RespondBadRequest(w, err.Error())
		return
	}

	if err := h.container.GetSchedulingRuleService().UpdateRule(r.Context(), rule); err != nil {
		h.logger.Error("Failed to update rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rule)
}

// DeleteRule 删除排班规则
func (h *HTTPHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || ruleID == "" {
		RespondBadRequest(w, "orgId and ruleId are required")
		return
	}

	if err := h.container.GetSchedulingRuleService().DeleteRule(r.Context(), orgID, ruleID); err != nil {
		h.logger.Error("Failed to delete rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondNoContent(w)
}

// GetRule 获取规则详情
func (h *HTTPHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || ruleID == "" {
		RespondBadRequest(w, "orgId and ruleId are required")
		return
	}

	// V4.1: 使用 GetRuleWithRelations 加载完整规则（包含班次关系和适用范围）
	rule, err := h.container.GetSchedulingRuleService().GetRuleWithRelations(r.Context(), orgID, ruleID)
	if err != nil {
		h.logger.Error("Failed to get rule", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	if rule == nil {
		RespondNotFound(w, "Rule not found")
		return
	}

	RespondOK(w, rule)
}

// ListRules 查询规则列表
func (h *HTTPHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	var req ListRulesRequest

	req.OrgID = r.URL.Query().Get("orgId")

	if ruleTypeStr := r.URL.Query().Get("ruleType"); ruleTypeStr != "" {
		// @deprecated V3: 规范化 V3 枚举值
		normalizedRuleType := normalizeRuleType(ruleTypeStr)
		req.RuleType = &normalizedRuleType
	}

	if applyScopeStr := r.URL.Query().Get("applyScope"); applyScopeStr != "" {
		// @deprecated V3: 规范化 V3 枚举值
		normalizedApplyScope := normalizeApplyScope(applyScopeStr)
		req.ApplyScope = &normalizedApplyScope
	}

	if timeScopeStr := r.URL.Query().Get("timeScope"); timeScopeStr != "" {
		// @deprecated V3: 规范化 V3 枚举值
		normalizedTimeScope := normalizeTimeScope(timeScopeStr)
		req.TimeScope = &normalizedTimeScope
	}

	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			req.IsActive = &isActive
		}
	}

	req.Keyword = r.URL.Query().Get("keyword")

	// V4新增筛选字段
	if category := r.URL.Query().Get("category"); category != "" {
		req.Category = &category
	}
	if subCategory := r.URL.Query().Get("subCategory"); subCategory != "" {
		req.SubCategory = &subCategory
	}
	if sourceType := r.URL.Query().Get("sourceType"); sourceType != "" {
		req.SourceType = &sourceType
	}
	if version := r.URL.Query().Get("version"); version != "" {
		req.Version = &version
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if size := r.URL.Query().Get("size"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			req.Size = s
		}
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 20
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.SchedulingRuleFilter{
		OrgID:      req.OrgID,
		RuleType:   req.RuleType,
		ApplyScope: req.ApplyScope,
		TimeScope:  req.TimeScope,
		IsActive:   req.IsActive,
		Keyword:    req.Keyword,
		// V4新增筛选字段
		Category:    req.Category,
		SubCategory: req.SubCategory,
		SourceType:  req.SourceType,
		Version:     req.Version,
		Page:        req.Page,
		PageSize:    req.Size,
	}

	result, err := h.container.GetSchedulingRuleService().ListRules(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list rules", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	// 为每个规则添加关联统计
	enrichedItems := make([]*RuleListResponse, len(result.Items))
	for i, rule := range result.Items {
		// 获取规则的关联信息
		associations, err := h.container.GetSchedulingRuleService().GetRuleAssociations(r.Context(), req.OrgID, rule.ID)

		employeeCount := 0
		shiftCount := 0
		groupCount := 0
		if err == nil {
			for _, assoc := range associations {
				switch assoc.AssociationType {
				case model.AssociationTypeEmployee:
					employeeCount++
				case model.AssociationTypeShift:
					shiftCount++
				case model.AssociationTypeGroup:
					groupCount++
				}
			}
		}

		enrichedItems[i] = &RuleListResponse{
			SchedulingRule:   rule,
			AssociationCount: len(associations),
			EmployeeCount:    employeeCount,
			ShiftCount:       shiftCount,
			GroupCount:       groupCount,
		}
	}

	RespondOK(w, map[string]interface{}{
		"items": enrichedItems,
		"total": result.Total,
	})
}

// GetRulesForEmployee 获取员工相关的规则
func (h *HTTPHandler) GetRulesForEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	employeeID := vars["employeeId"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || employeeID == "" {
		RespondBadRequest(w, "orgId and employeeId are required")
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForEmployee(r.Context(), orgID, employeeID)
	if err != nil {
		h.logger.Error("Failed to get rules for employee", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// GetRulesForShift 获取班次相关的规则
func (h *HTTPHandler) GetRulesForShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["shiftId"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || shiftID == "" {
		RespondBadRequest(w, "orgId and shiftId are required")
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForShift(r.Context(), orgID, shiftID)
	if err != nil {
		h.logger.Error("Failed to get rules for shift", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// GetRulesForGroup 获取分组相关的规则
func (h *HTTPHandler) GetRulesForGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" || groupID == "" {
		RespondBadRequest(w, "orgId and groupId are required")
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForGroup(r.Context(), orgID, groupID)
	if err != nil {
		h.logger.Error("Failed to get rules for group", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// GetRulesForEmployees 批量获取多个员工相关的规则
func (h *HTTPHandler) GetRulesForEmployees(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		EmployeeIDs []string `json:"employeeIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if len(req.EmployeeIDs) == 0 {
		RespondOK(w, make(map[string][]*model.SchedulingRule))
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForEmployees(r.Context(), orgID, req.EmployeeIDs)
	if err != nil {
		h.logger.Error("Failed to get rules for employees", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// GetRulesForShifts 批量获取多个班次相关的规则
func (h *HTTPHandler) GetRulesForShifts(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		ShiftIDs []string `json:"shiftIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if len(req.ShiftIDs) == 0 {
		RespondOK(w, make(map[string][]*model.SchedulingRule))
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForShifts(r.Context(), orgID, req.ShiftIDs)
	if err != nil {
		h.logger.Error("Failed to get rules for shifts", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// GetRulesForGroups 批量获取多个分组相关的规则
func (h *HTTPHandler) GetRulesForGroups(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		GroupIDs []string `json:"groupIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if len(req.GroupIDs) == 0 {
		RespondOK(w, make(map[string][]*model.SchedulingRule))
		return
	}

	rules, err := h.container.GetSchedulingRuleService().GetRulesForGroups(r.Context(), orgID, req.GroupIDs)
	if err != nil {
		h.logger.Error("Failed to get rules for groups", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondOK(w, rules)
}

// ToggleSchedulingRuleStatus 切换排班规则启用/禁用状态
func (h *HTTPHandler) ToggleSchedulingRuleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetSchedulingRuleService().ToggleRuleStatus(r.Context(), orgID, ruleID, req.IsActive); err != nil {
		h.logger.Error("Failed to toggle scheduling rule status", "error", err)
		RespondInternalError(w, "Failed to toggle scheduling rule status")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Scheduling rule status updated successfully",
		"ruleId":   ruleID,
		"isActive": req.IsActive,
	})
}
