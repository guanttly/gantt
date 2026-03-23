package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// 关联班次分组识别
// ============================================================

// identifyShiftGroups 识别关联班次分组
// 调用LLM分析班次之间的关联关系，将需要一起排班的班次分到同一组
// 返回分组结果，同组班次将共享规则和人员信息
func (e *ProgressiveTaskExecutor) identifyShiftGroups(
	ctx context.Context,
	shiftSpecs []ShiftTaskSpec,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
) ([]ShiftGroup, string, error) {
	// 如果只有一个或零个班次，无需分组
	if len(shiftSpecs) <= 1 {
		if len(shiftSpecs) == 1 {
			group := ShiftGroup{
				GroupID:       "group_1",
				Shifts:        shiftSpecs,
				RelatedReason: "单个班次，无需分组",
				SharedRuleIDs: []string{},
			}
			return []ShiftGroup{group}, "只有一个班次，直接作为独立分组", nil
		}
		return []ShiftGroup{}, "无班次需要分组", nil
	}

	// 构建提示词（返回 shortID 映射表）
	userPrompt, shiftForwardMappings, ruleForwardMappings := e.buildShiftGroupingPrompt(shiftSpecs, shifts, rules)
	systemPrompt := e.buildShiftGroupingSystemPrompt()

	// 构建反向映射（shortID -> realID）
	shiftReverseMappings := make(map[string]string)
	for realID, shortID := range shiftForwardMappings {
		shiftReverseMappings[shortID] = realID
	}
	ruleReverseMappings := make(map[string]string)
	for realID, shortID := range ruleForwardMappings {
		ruleReverseMappings[shortID] = realID
	}

	// 调用LLM
	llmCallStart := time.Now()
	resp, err := e.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	llmCallDuration := time.Since(llmCallStart)

	// 记录到调试文件
	e.logLLMDebug("shift_grouping", logging.LLMCallShiftGrouping, "", "", systemPrompt, userPrompt, resp.Content, llmCallDuration, err)

	if err != nil {
		e.logger.Error("Shift grouping LLM call failed", "error", err)
		// 失败时回退到保守策略：所有班次放入同一组
		return e.fallbackToSingleGroup(shiftSpecs, "LLM调用失败，采用保守分组策略")
	}

	// 解析LLM响应（传入反向映射以将 shortID 转回真实ID）
	groups, reasoning, parseErr := e.parseShiftGroupingResponse(resp.Content, shiftSpecs, shiftReverseMappings, ruleReverseMappings)
	if parseErr != nil {
		e.logger.Warn("Failed to parse shift grouping response, using fallback", "error", parseErr)
		return e.fallbackToSingleGroup(shiftSpecs, fmt.Sprintf("解析失败(%v)，采用保守分组策略", parseErr))
	}

	return groups, reasoning, nil
}

// parseShiftGroupingResponse 解析LLM返回的分组结果
// shiftReverseMappings: shortID -> realID 的班次映射
// ruleReverseMappings: shortID -> realID 的规则映射
func (e *ProgressiveTaskExecutor) parseShiftGroupingResponse(
	content string,
	originalSpecs []ShiftTaskSpec,
	shiftReverseMappings map[string]string,
	ruleReverseMappings map[string]string,
) ([]ShiftGroup, string, error) {
	// 提取JSON部分
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, "", fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	// 解析JSON
	var result struct {
		Groups []struct {
			GroupID       string   `json:"groupId"`
			ShiftIDs      []string `json:"shiftIds"`
			RelatedReason string   `json:"relatedReason"`
			SharedRuleIDs []string `json:"sharedRuleIds"`
		} `json:"groups"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// 构建shiftID到ShiftTaskSpec的映射（使用真实ID）
	specMap := make(map[string]ShiftTaskSpec)
	for _, spec := range originalSpecs {
		specMap[spec.ShiftID] = spec
	}

	// 转换为ShiftGroup结构
	groups := make([]ShiftGroup, 0, len(result.Groups))
	coveredShiftIDs := make(map[string]bool)

	for _, g := range result.Groups {
		// 转换规则ID：shortID -> realID
		realRuleIDs := make([]string, 0, len(g.SharedRuleIDs))
		for _, ruleShortID := range g.SharedRuleIDs {
			if realID, ok := ruleReverseMappings[ruleShortID]; ok {
				realRuleIDs = append(realRuleIDs, realID)
			} else {
				// 如果不在映射中，可能已经是真实ID
				realRuleIDs = append(realRuleIDs, ruleShortID)
			}
		}

		group := ShiftGroup{
			GroupID:       g.GroupID,
			Shifts:        make([]ShiftTaskSpec, 0, len(g.ShiftIDs)),
			RelatedReason: g.RelatedReason,
			SharedRuleIDs: realRuleIDs,
		}

		for _, shiftShortID := range g.ShiftIDs {
			// 转换班次ID：shortID -> realID
			realShiftID := shiftShortID
			if mappedID, ok := shiftReverseMappings[shiftShortID]; ok {
				realShiftID = mappedID
			}

			if spec, ok := specMap[realShiftID]; ok {
				group.Shifts = append(group.Shifts, spec)
				coveredShiftIDs[realShiftID] = true
			} else {
				e.logger.Warn("Unknown shift ID in grouping result", "shortID", shiftShortID, "realID", realShiftID)
			}
		}

		if len(group.Shifts) > 0 {
			groups = append(groups, group)
		}
	}

	// 检查是否所有班次都被覆盖
	for _, spec := range originalSpecs {
		if !coveredShiftIDs[spec.ShiftID] {
			e.logger.Warn("Shift not covered by any group, creating single group",
				"shiftID", spec.ShiftID,
				"shiftName", spec.ShiftName)
			// 创建单独的分组
			groups = append(groups, ShiftGroup{
				GroupID:       fmt.Sprintf("group_orphan_%s", spec.ShiftID),
				Shifts:        []ShiftTaskSpec{spec},
				RelatedReason: "未被分组的班次，独立处理",
				SharedRuleIDs: []string{},
			})
		}
	}

	return groups, result.Reasoning, nil
}

// fallbackToSingleGroup 回退策略：将所有班次放入同一组
// 这是最保守的策略，确保规则不会被遗漏
func (e *ProgressiveTaskExecutor) fallbackToSingleGroup(
	shiftSpecs []ShiftTaskSpec,
	reason string,
) ([]ShiftGroup, string, error) {
	group := ShiftGroup{
		GroupID:       "group_all",
		Shifts:        shiftSpecs,
		RelatedReason: reason,
		SharedRuleIDs: []string{},
	}

	e.logger.Info("Using fallback single group strategy",
		"reason", reason,
		"shiftCount", len(shiftSpecs))

	return []ShiftGroup{group}, reason, nil
}

// getGroupRelatedRules 获取分组相关的所有规则
// 用于在组内所有班次的提示词中包含这些规则
func (e *ProgressiveTaskExecutor) getGroupRelatedRules(
	group ShiftGroup,
	allRules []*d_model.Rule,
) []*d_model.Rule {
	// 构建组内班次ID集合
	groupShiftIDs := make(map[string]bool)
	for _, s := range group.Shifts {
		groupShiftIDs[s.ShiftID] = true
	}

	// 过滤与组内任意班次相关的规则
	relatedRules := make([]*d_model.Rule, 0)
	for _, rule := range allRules {
		isRelated := false

		// 检查规则的Associations字段，查找类型为shift的关联
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == "shift" && groupShiftIDs[assoc.AssociationID] {
				isRelated = true
				break
			}
		}

		if isRelated {
			relatedRules = append(relatedRules, rule)
		}
	}

	return relatedRules
}

// getGroupRelatedStaff 获取分组相关的所有人员
// 用于在组内所有班次的提示词中包含这些人员
func (e *ProgressiveTaskExecutor) getGroupRelatedStaff(
	ctx context.Context,
	group ShiftGroup,
	allStaff []*d_model.Employee,
) []*d_model.Employee {
	// 收集组内所有班次的人员
	staffSet := make(map[string]*d_model.Employee)

	for _, s := range group.Shifts {
		// 获取班次关联的人员
		shiftStaff, err := e.rosteringService.GetShiftGroupMembers(ctx, s.ShiftID)
		if err != nil {
			e.logger.Warn("Failed to get shift group members, using all staff",
				"shiftID", s.ShiftID,
				"error", err)
			// 失败时使用全员
			for _, emp := range allStaff {
				staffSet[emp.ID] = emp
			}
			continue
		}

		for _, emp := range shiftStaff {
			staffSet[emp.ID] = emp
		}
	}

	// 转换为列表
	result := make([]*d_model.Employee, 0, len(staffSet))
	for _, emp := range staffSet {
		result = append(result, emp)
	}

	return result
}
