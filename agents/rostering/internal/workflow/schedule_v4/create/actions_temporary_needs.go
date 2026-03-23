package create

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	"jusha/mcp/pkg/workflow/engine"
)

// ============================================================
// 临时需求解析与个人需求处理
// ============================================================

// ExtractPersonalNeeds 从规则中提取个人需求
func ExtractPersonalNeeds(rules []*d_model.Rule, staffList []*d_model.Employee) map[string][]*PersonalNeed {
	result := make(map[string][]*PersonalNeed)

	candidateStaffIDs := make(map[string]bool)
	staffNameMap := make(map[string]string)
	for _, staff := range staffList {
		candidateStaffIDs[staff.ID] = true
		staffNameMap[staff.ID] = staff.Name
	}

	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}
		if len(rule.Associations) > 0 {
			for _, assoc := range rule.Associations {
				if assoc.AssociationType == "staff" {
					staffID := assoc.AssociationID
					if !candidateStaffIDs[staffID] {
						continue
					}
					need := parseRuleToPersonalNeed(rule, staffID, staffNameMap)
					if need != nil {
						if result[staffID] == nil {
							result[staffID] = make([]*PersonalNeed, 0)
						}
						result[staffID] = append(result[staffID], need)
					}
				}
			}
		}
	}

	return result
}

// parseRuleToPersonalNeed 解析规则为个人需求
func parseRuleToPersonalNeed(rule *d_model.Rule, staffID string, staffNameMap map[string]string) *PersonalNeed {
	if staffID == "" {
		return nil
	}

	staffName := staffNameMap[staffID]
	if staffName == "" {
		staffName = "未知员工"
	}

	needType := "permanent"
	if rule.ValidFrom != nil && rule.ValidTo != nil {
		needType = "temporary"
	}

	requestType := "prefer"
	if rule.Priority <= 3 {
		requestType = "must"
	}

	description := rule.Description
	if description == "" {
		if rule.RuleData != "" {
			description = rule.RuleData
		} else {
			description = rule.Name
		}
	}

	return &PersonalNeed{
		StaffID: staffID, StaffName: staffName, NeedType: needType,
		RequestType: requestType, Description: description,
		Priority: rule.Priority, RuleID: rule.ID,
		Source: "rule", Confirmed: false,
	}
}

// applyTemporaryNeedsText 解析并应用临时需求文本（解析为临时规则）
func applyTemporaryNeedsText(ctx context.Context, wctx engine.Context, createCtx *CreateV4Context, requirementText string) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV4: Applying temporary needs text as rules", "sessionID", sess.ID, "textLength", len(requirementText))

	aiService, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		logger.Warn("SchedulingAI service not available, parsing manually")
		return parseTemporaryNeedsAsRulesManually(createCtx, requirementText)
	}

	parsedNeeds, err := aiService.ExtractTemporaryNeeds(
		ctx, requirementText, createCtx.AllStaff, createCtx.StartDate, createCtx.EndDate, nil,
	)
	if err != nil {
		logger.Error("Failed to parse temporary needs with LLM", "error", err)
		return parseTemporaryNeedsAsRulesManually(createCtx, requirementText)
	}

	validFrom, _ := time.Parse("2006-01-02", createCtx.StartDate)
	validTo, _ := time.Parse("2006-01-02", createCtx.EndDate)

	if createCtx.TemporaryRules == nil {
		createCtx.TemporaryRules = make([]*d_model.Rule, 0)
	}
	if createCtx.PersonalNeeds == nil {
		createCtx.PersonalNeeds = make(map[string][]*PersonalNeed)
	}

	for i, need := range parsedNeeds {
		if need.StaffID == "" {
			continue
		}

		category := "constraint"
		subCategory := "prefer"
		ruleType := "preferred"
		priority := need.Priority
		if priority == 0 {
			priority = 5
		}

		switch need.RequestType {
		case "avoid":
			subCategory = "forbid"
			ruleType = "forbidden_day"
			category = "constraint"
			if priority > 1 {
				priority = 1
			}
		case "must":
			subCategory = "must"
			ruleType = "required"
			category = "constraint"
			if priority > 2 {
				priority = 2
			}
		case "prefer":
			subCategory = "prefer"
			ruleType = "preferred"
			category = "preference"
		}

		rule := &d_model.Rule{
			ID:          fmt.Sprintf("temp_rule_%d_%d", time.Now().UnixNano(), i),
			Name:        fmt.Sprintf("临时需求: %s", need.StaffName),
			Category:    category,
			SubCategory: subCategory,
			RuleType:    ruleType,
			Description: need.Description,
			RuleData:    need.Description,
			Priority:    priority,
			IsActive:    true,
			SourceType:  "temporary",
			ValidFrom:   &validFrom,
			ValidTo:     &validTo,
		}

		rule.Associations = append(rule.Associations, d_model.RuleAssociation{
			AssociationType: "staff", AssociationID: need.StaffID,
		})
		if need.TargetShiftID != "" {
			rule.Associations = append(rule.Associations, d_model.RuleAssociation{
				AssociationType: "shift", AssociationID: need.TargetShiftID,
			})
		}
		if len(need.TargetDates) > 0 {
			datesJSON, _ := json.Marshal(map[string][]string{"targetDates": need.TargetDates})
			rule.RuleData = rule.RuleData + " | dates:" + string(datesJSON)
		}

		createCtx.TemporaryRules = append(createCtx.TemporaryRules, rule)

		personalNeed := &PersonalNeed{
			StaffID: need.StaffID, StaffName: need.StaffName,
			NeedType: "temporary", RequestType: need.RequestType,
			TargetShiftID: need.TargetShiftID, TargetShiftName: need.TargetShiftName,
			TargetDates: need.TargetDates, Description: need.Description,
			Priority: priority, RuleID: rule.ID, Source: "user", Confirmed: false,
		}
		createCtx.PersonalNeeds[need.StaffID] = append(createCtx.PersonalNeeds[need.StaffID], personalNeed)
	}

	logger.Info("CreateV4: Temporary needs parsed as rules", "sessionID", sess.ID, "rulesCount", len(createCtx.TemporaryRules))
	return nil
}

// parseTemporaryNeedsAsRulesManually 手动将临时需求解析为规则（回退方案）
func parseTemporaryNeedsAsRulesManually(createCtx *CreateV4Context, requirementText string) error {
	lines := strings.Split(requirementText, "\n")

	staffNameToID := make(map[string]string)
	for _, staff := range createCtx.AllStaff {
		staffNameToID[staff.Name] = staff.ID
	}
	shiftNameToID := make(map[string]string)
	shiftIDToName := make(map[string]string)
	for _, shift := range createCtx.SelectedShifts {
		shiftNameToID[shift.Name] = shift.ID
		shiftIDToName[shift.ID] = shift.Name
	}

	if createCtx.TemporaryRules == nil {
		createCtx.TemporaryRules = make([]*d_model.Rule, 0)
	}
	if createCtx.PersonalNeeds == nil {
		createCtx.PersonalNeeds = make(map[string][]*PersonalNeed)
	}

	validFrom, _ := time.Parse("2006-01-02", createCtx.StartDate)
	validTo, _ := time.Parse("2006-01-02", createCtx.EndDate)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var matchedStaffID, matchedStaffName string
		for name, id := range staffNameToID {
			if strings.Contains(line, name) {
				matchedStaffID = id
				matchedStaffName = name
				break
			}
		}

		category := "constraint"
		subCategory := "prefer"
		ruleType := "preferred"
		priority := 5

		if strings.Contains(line, "不能") || strings.Contains(line, "请假") || strings.Contains(line, "出差") || strings.Contains(line, "休息") {
			subCategory = "forbid"
			ruleType = "forbidden_day"
			priority = 1
		} else if strings.Contains(line, "必须") || strings.Contains(line, "一定") {
			subCategory = "must"
			ruleType = "required"
			priority = 2
		} else if strings.Contains(line, "不想") || strings.Contains(line, "不要") || strings.Contains(line, "回避") {
			subCategory = "avoid"
			ruleType = "avoid"
			priority = 3
		} else {
			category = "preference"
		}

		var matchedShiftID, matchedShiftName string
		for name, id := range shiftNameToID {
			if strings.Contains(line, name) {
				matchedShiftID = id
				matchedShiftName = name
				break
			}
		}

		rule := &d_model.Rule{
			ID: fmt.Sprintf("temp_rule_%d_%d", time.Now().UnixNano(), i),
			Name: fmt.Sprintf("临时需求: %s", matchedStaffName),
			Category: category, SubCategory: subCategory, RuleType: ruleType,
			Description: line, RuleData: line,
			Priority: priority, IsActive: true, SourceType: "temporary",
			ValidFrom: &validFrom, ValidTo: &validTo,
		}

		if matchedStaffID != "" {
			rule.Associations = append(rule.Associations, d_model.RuleAssociation{
				AssociationType: "staff", AssociationID: matchedStaffID,
			})
		}
		if matchedShiftID != "" {
			rule.Associations = append(rule.Associations, d_model.RuleAssociation{
				AssociationType: "shift", AssociationID: matchedShiftID,
			})
		}

		createCtx.TemporaryRules = append(createCtx.TemporaryRules, rule)

		if matchedStaffID != "" {
			requestType := "prefer"
			switch subCategory {
			case "forbid", "avoid":
				requestType = "avoid"
			case "must":
				requestType = "must"
			}
			need := &PersonalNeed{
				StaffID: matchedStaffID, StaffName: matchedStaffName,
				NeedType: "temporary", RequestType: requestType,
				TargetShiftID: matchedShiftID, TargetShiftName: matchedShiftName,
				Description: line, Priority: priority, RuleID: rule.ID,
				Source: "user", Confirmed: false,
			}
			createCtx.PersonalNeeds[matchedStaffID] = append(createCtx.PersonalNeeds[matchedStaffID], need)
		}
	}

	_ = shiftIDToName // suppress unused
	return nil
}

// buildTemporaryRulesPayload 构建临时规则查看按钮的 payload
func buildTemporaryRulesPayload(createCtx *CreateV4Context) map[string]any {
	constraintCount := 0
	preferenceCount := 0

	rules := make([]map[string]any, 0, len(createCtx.TemporaryRules))
	for _, rule := range createCtx.TemporaryRules {
		if rule.Category == "constraint" {
			constraintCount++
		} else {
			preferenceCount++
		}

		var staffName, shiftName string
		var targetDates []string
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == "staff" {
				for _, staff := range createCtx.AllStaff {
					if staff.ID == assoc.AssociationID {
						staffName = staff.Name
						break
					}
				}
			} else if assoc.AssociationType == "shift" {
				for _, shift := range createCtx.SelectedShifts {
					if shift.ID == assoc.AssociationID {
						shiftName = shift.Name
						break
					}
				}
			}
		}
		if rule.TimeScope != "" {
			targetDates = []string{rule.TimeScope}
		}

		ruleItem := map[string]any{
			"id": rule.ID, "name": rule.Name,
			"category": rule.Category, "subCategory": rule.SubCategory,
			"ruleType": rule.RuleType, "description": rule.Description,
			"priority": rule.Priority,
		}
		if staffName != "" {
			ruleItem["staffName"] = staffName
		}
		if shiftName != "" {
			ruleItem["shiftName"] = shiftName
		}
		if len(targetDates) > 0 {
			ruleItem["targetDates"] = targetDates
		}
		if rule.RuleData != "" {
			ruleItem["ruleData"] = rule.RuleData
		}
		if len(rule.Associations) > 0 {
			associations := make([]map[string]string, 0, len(rule.Associations))
			for _, assoc := range rule.Associations {
				associations = append(associations, map[string]string{
					"associationType": assoc.AssociationType,
					"associationId":   assoc.AssociationID,
				})
			}
			ruleItem["associations"] = associations
		}
		rules = append(rules, ruleItem)
	}

	return map[string]any{
		"totalRules":      len(createCtx.TemporaryRules),
		"constraintCount": constraintCount,
		"preferenceCount": preferenceCount,
		"rules":           rules,
	}
}

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
