package create

import (
	"fmt"
	"strings"

	d_model "jusha/agent/rostering/domain/model"
	rule_engine "jusha/agent/rostering/internal/engine"

	"jusha/mcp/pkg/logging"
)

// ============================================================
// 规则类型中英文映射
// ============================================================

// ruleTypeCNMap 规则类型英文 → 中文映射
var ruleTypeCNMap = map[string]string{
	"workloadBalance":  "工作负载均衡",
	"workloadTotal":    "总班次负载",
	"workloadOvertime": "加班负载",
	"staffCount":       "人员配置",
	"maxCount":         "最大排班次数",
	"consecutiveMax":   "连续排班上限",
	"minRestDays":      "最少休息天数",
	"exclusive":        "互斥约束",
	"forbidden_day":    "禁止排班日",
	"preferred":        "偏好排班",
	"combinable":       "组合约束",
	"required":         "必排约束",
	"avoid":            "回避约束",
	"shortage":         "人员缺口",
	"语义规则校验":           "语义规则校验",
}

// translateRuleType 将英文规则类型翻译为中文
func translateRuleType(ruleType string) string {
	if cn, ok := ruleTypeCNMap[ruleType]; ok {
		return cn
	}
	return ruleType
}

// ============================================================
// 排班校验逻辑
// ============================================================

// validateSchedule 校验排班结果（只读，不修改排班草稿）
func validateSchedule(createCtx *CreateV4Context, logger logging.ILogger) *ValidationResult {
	result := &ValidationResult{
		IsValid:        true,
		Violations:     make([]*Violation, 0),
		Warnings:       make([]*Warning, 0),
		UncheckedRules: make([]*UncheckedRule, 0),
	}

	if createCtx.WorkingDraft == nil {
		result.IsValid = false
		result.Summary = "排班结果为空"
		return result
	}

	validator := rule_engine.NewScheduleValidator(logger)

	staffRequirements := make(map[string]map[string]int)
	for _, req := range createCtx.StaffRequirements {
		if staffRequirements[req.ShiftID] == nil {
			staffRequirements[req.ShiftID] = make(map[string]int)
		}
		staffRequirements[req.ShiftID][req.Date] = req.Count
	}

	allRules := make([]*d_model.Rule, 0, len(createCtx.Rules)+len(createCtx.TemporaryRules))
	allRules = append(allRules, createCtx.Rules...)
	allRules = append(allRules, createCtx.TemporaryRules...)

	staffList := make([]*d_model.Staff, 0, len(createCtx.AllStaff))
	for _, emp := range createCtx.AllStaff {
		staffList = append(staffList, emp)
	}

	engineResult, err := validator.ValidateFullSchedule(
		createCtx.WorkingDraft, allRules, staffList, createCtx.SelectedShifts, staffRequirements,
	)
	if err != nil {
		logger.Warn("引擎校验失败，降级为基础校验", "error", err)
		return validateScheduleBasic(createCtx)
	}

	result.IsValid = engineResult.IsValid
	result.Summary = engineResult.Summary

	for _, v := range engineResult.Violations {
		result.Violations = append(result.Violations, &Violation{
			RuleID: v.RuleID, RuleName: v.RuleName,
			Date: v.Date, ShiftID: v.ShiftID,
			StaffID: firstOrEmpty(v.StaffIDs), StaffName: firstOrEmpty(v.StaffNames),
			Description: v.Message, Severity: v.Severity,
		})
	}
	for _, w := range engineResult.Warnings {
		result.Warnings = append(result.Warnings, &Warning{
			Type:        translateRuleType(w.RuleType),
			Description: w.Message,
			Severity:    "info",
		})
	}
	for _, ur := range engineResult.UncheckedRules {
		result.UncheckedRules = append(result.UncheckedRules, &UncheckedRule{
			RuleID: ur.RuleID, RuleName: ur.RuleName,
			RuleType: ur.RuleType, Description: ur.Description, Reason: ur.Reason,
		})
	}

	return result
}

// validateScheduleBasic 基础校验（仅检查人数缺口，作为引擎校验的降级方案）
func validateScheduleBasic(createCtx *CreateV4Context) *ValidationResult {
	result := &ValidationResult{
		IsValid: true, Violations: make([]*Violation, 0), Warnings: make([]*Warning, 0),
	}

	totalRequired, totalAssigned, shortages := 0, 0, 0

	for _, req := range createCtx.StaffRequirements {
		totalRequired += req.Count
		shiftDraft := createCtx.WorkingDraft.Shifts[req.ShiftID]
		if shiftDraft == nil || shiftDraft.Days == nil {
			shortages++
			continue
		}
		dayShift := shiftDraft.Days[req.Date]
		if dayShift == nil {
			shortages++
			continue
		}
		totalAssigned += dayShift.ActualCount
		if dayShift.ActualCount < req.Count {
			shortages++
			result.Warnings = append(result.Warnings, &Warning{
				Type:        "人员缺口",
				Description: fmt.Sprintf("%s %s：需要 %d 人，实际 %d 人", req.ShiftName, req.Date, req.Count, dayShift.ActualCount),
			})
		}
	}

	if shortages > 0 {
		result.IsValid = false
	}
	result.Summary = fmt.Sprintf("需求 %d 人次，已分配 %d 人次，缺口 %d 处", totalRequired, totalAssigned, shortages)
	return result
}

// buildValidationSummary 根据校验结果构建摘要
func buildValidationSummary(result *ValidationResult) string {
	parts := make([]string, 0)
	if result.IsValid {
		parts = append(parts, "校验通过")
	} else {
		parts = append(parts, fmt.Sprintf("发现 %d 个违规项", len(result.Violations)))
	}
	semanticWarnCount, infoWarnCount := 0, 0
	for _, w := range result.Warnings {
		if w.Severity == "warning" {
			semanticWarnCount++
		} else {
			infoWarnCount++
		}
	}
	if semanticWarnCount > 0 {
		parts = append(parts, fmt.Sprintf("%d 个警告", semanticWarnCount))
	}
	if infoWarnCount > 0 {
		parts = append(parts, fmt.Sprintf("%d 条提示", infoWarnCount))
	}
	if len(result.UncheckedRules) > 0 {
		if result.LLMValidationDone {
			parts = append(parts, fmt.Sprintf("%d 条语义规则已完成 LLM 校验", len(result.UncheckedRules)))
		} else {
			parts = append(parts, fmt.Sprintf("%d 条语义规则未自动校验", len(result.UncheckedRules)))
		}
	}
	return strings.Join(parts, "，")
}

// buildValidationMessage 构建校验结果消息
func buildValidationMessage(result *ValidationResult) string {
	if result == nil {
		return "⚠️ 校验结果为空"
	}
	var message strings.Builder
	if result.IsValid {
		message.WriteString("**校验通过**\n\n")
	} else {
		message.WriteString("**校验发现问题**\n\n")
	}
	message.WriteString(result.Summary + "\n")

	if len(result.Violations) > 0 {
		message.WriteString("\n**违规项**：\n")
		for i, v := range result.Violations {
			if i >= 5 {
				message.WriteString(fmt.Sprintf("... 还有 %d 个违规项\n", len(result.Violations)-5))
				break
			}
			message.WriteString(fmt.Sprintf("- %s: %s\n", v.RuleName, v.Description))
		}
	}
	if len(result.Warnings) > 0 {
		message.WriteString("\n**警告**：\n")
		for i, w := range result.Warnings {
			if i >= 5 {
				message.WriteString(fmt.Sprintf("... 还有 %d 个警告\n", len(result.Warnings)-5))
				break
			}
			message.WriteString(fmt.Sprintf("- [%s] %s\n", w.Type, w.Description))
		}
	}
	if len(result.UncheckedRules) > 0 && result.LLMValidationDone {
		message.WriteString("\n**LLM 语义规则校验**：已完成\n")
	}
	return message.String()
}

// buildValidationResultPayload 构建校验结果的前端展示数据
func buildValidationResultPayload(result *ValidationResult) map[string]any {
	if result == nil {
		return map[string]any{"isValid": true, "violations": []any{}, "warnings": []any{}, "summary": "无校验数据"}
	}
	violations := make([]map[string]any, 0, len(result.Violations))
	for _, v := range result.Violations {
		violations = append(violations, map[string]any{
			"ruleId": v.RuleID, "ruleName": v.RuleName, "date": v.Date,
			"shiftId": v.ShiftID, "shiftName": v.ShiftName,
			"staffId": v.StaffID, "staffName": v.StaffName,
			"description": v.Description, "severity": v.Severity,
		})
	}
	warnings := make([]map[string]any, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, map[string]any{
			"type":        w.Type,
			"description": w.Description,
			"suggestion":  w.Suggestion,
			"severity":    w.Severity,
		})
	}
	return map[string]any{
		"isValid":           result.IsValid,
		"violations":        violations,
		"warnings":          warnings,
		"summary":           result.Summary,
		"llmValidationDone": result.LLMValidationDone,
	}
}
