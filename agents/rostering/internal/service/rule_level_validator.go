package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"jusha/mcp/pkg/logging"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
)

// ============================================================
// 规则级校验器实现
// ============================================================

// RuleLevelValidator 规则级校验器实现
type RuleLevelValidator struct {
	logger logging.ILogger
}

// NewRuleLevelValidator 创建规则级校验器
func NewRuleLevelValidator(logger logging.ILogger) d_service.IRuleLevelValidator {
	return &RuleLevelValidator{
		logger: logger.With("component", "RuleLevelValidator"),
	}
}

// ValidateStaffCount 人数校验（仅超配检测，用于渐进式中途校验）
// 只检查是否超配，允许人数不足（渐进填充过程中正常现象）
func (v *RuleLevelValidator) ValidateStaffCount(ctx context.Context, scheduleDraft *d_model.ShiftScheduleDraft, staffRequirements map[string]int) (*d_model.RuleValidationResult, error) {
	result := &d_model.RuleValidationResult{
		Passed:           true,
		StaffCountIssues: make([]*d_model.ValidationIssue, 0),
	}

	if scheduleDraft == nil || scheduleDraft.Schedule == nil {
		// 空草案在渐进式校验中是允许的
		result.Summary = "人数校验通过（草案为空，渐进式允许）"
		return result, nil
	}

	// 【渐进式校验】只检查超配，不检查不足
	for date, requiredCount := range staffRequirements {
		if requiredCount <= 0 {
			continue // 不需要排班的日期跳过
		}

		actualCount := 0
		if staffIDs, ok := scheduleDraft.Schedule[date]; ok {
			actualCount = len(staffIDs)
		}

		// 只检查超配（严重错误）
		if actualCount > requiredCount {
			result.Passed = false
			result.StaffCountIssues = append(result.StaffCountIssues, &d_model.ValidationIssue{
				Type:          "staff_count_overflow",
				Severity:      "critical",
				Description:   fmt.Sprintf("日期 %s 超配：需要 %d 人，实际安排了 %d 人，超出 %d 人", date, requiredCount, actualCount, actualCount-requiredCount),
				AffectedDates: []string{date},
			})
		}
		// 人数不足的情况不再标记为失败（渐进式允许）
	}

	if len(result.StaffCountIssues) == 0 {
		result.Summary = "人数校验通过：无超配问题"
	} else {
		result.Summary = fmt.Sprintf("人数校验失败：发现 %d 个超配问题", len(result.StaffCountIssues))
	}

	return result, nil
}

// ValidateStaffCountStrict 严格人数校验（用于最终校验）
// 检查排班草案中每日的人数是否严格等于需求
// 返回详细的缺员信息（班次、日期、缺员数）
func (v *RuleLevelValidator) ValidateStaffCountStrict(ctx context.Context, scheduleDraft *d_model.ShiftScheduleDraft, staffRequirements map[string]int) (*d_model.RuleValidationResult, error) {
	result := &d_model.RuleValidationResult{
		Passed:           true,
		StaffCountIssues: make([]*d_model.ValidationIssue, 0),
	}

	if scheduleDraft == nil || scheduleDraft.Schedule == nil {
		result.Passed = false
		result.Summary = "排班草案为空"
		return result, nil
	}

	// 严格检查：人数必须等于需求
	for date, requiredCount := range staffRequirements {
		if requiredCount <= 0 {
			continue // 不需要排班的日期跳过
		}

		actualCount := 0
		if staffIDs, ok := scheduleDraft.Schedule[date]; ok {
			actualCount = len(staffIDs)
		}

		if actualCount < requiredCount {
			// 人数不足（严格模式下是严重错误）
			result.Passed = false
			result.StaffCountIssues = append(result.StaffCountIssues, &d_model.ValidationIssue{
				Type:          "staff_count_shortage",
				Severity:      "critical",
				Description:   fmt.Sprintf("日期 %s 缺员：需要 %d 人，实际只有 %d 人，缺少 %d 人", date, requiredCount, actualCount, requiredCount-actualCount),
				AffectedDates: []string{date},
			})
		} else if actualCount > requiredCount {
			// 人数超过上限（也是错误）
			result.Passed = false
			result.StaffCountIssues = append(result.StaffCountIssues, &d_model.ValidationIssue{
				Type:          "staff_count_overflow",
				Severity:      "critical",
				Description:   fmt.Sprintf("日期 %s 超配：需要 %d 人，实际安排了 %d 人，超出 %d 人", date, requiredCount, actualCount, actualCount-requiredCount),
				AffectedDates: []string{date},
			})
		}
	}

	// 检查是否有未在需求中的日期被排班
	if len(staffRequirements) > 0 {
		for date := range scheduleDraft.Schedule {
			if _, ok := staffRequirements[date]; !ok {
				// 该日期不在需求中，但被排班了（警告）
				result.StaffCountIssues = append(result.StaffCountIssues, &d_model.ValidationIssue{
					Type:          "staff_count_unexpected",
					Severity:      "low",
					Description:   fmt.Sprintf("日期 %s 不在需求中，但被安排了排班", date),
					AffectedDates: []string{date},
				})
			}
		}
	}

	if len(result.StaffCountIssues) == 0 {
		result.Summary = "严格人数校验通过：所有日期的人数都严格满足需求"
	} else {
		shortageCount := 0
		overflowCount := 0
		for _, issue := range result.StaffCountIssues {
			if issue.Type == "staff_count_shortage" {
				shortageCount++
			} else if issue.Type == "staff_count_overflow" {
				overflowCount++
			}
		}
		result.Summary = fmt.Sprintf("严格人数校验失败：%d 个缺员，%d 个超配", shortageCount, overflowCount)
	}

	return result, nil
}

// ValidateShiftRules 班次规则校验
// 注意：时间冲突检查已在单班次校验中完成（使用真正的时间重叠检查），
// 这里只做基础的占位检查，不重复判断时间冲突
func (v *RuleLevelValidator) ValidateShiftRules(ctx context.Context, scheduleDraft *d_model.ShiftScheduleDraft, shifts []*d_model.Shift, occupiedSlots map[string]map[string]string) (*d_model.RuleValidationResult, error) {
	result := &d_model.RuleValidationResult{
		Passed:          true,
		ShiftRuleIssues: make([]*d_model.ValidationIssue, 0),
	}

	if scheduleDraft == nil || scheduleDraft.Schedule == nil {
		result.Passed = false
		result.Summary = "排班草案为空"
		return result, nil
	}

	// 构建班次ID到班次信息的映射（用于时间重叠检查）
	shiftMap := make(map[string]*d_model.Shift)
	shiftIDToName := make(map[string]string)
	for _, shift := range shifts {
		if shift != nil && shift.ID != "" {
			shiftMap[shift.ID] = shift
			if shift.Name != "" {
				shiftIDToName[shift.ID] = shift.Name
			}
		}
	}

	// 获取当前正在校验的班次（shifts[0] 是当前班次）
	var currentShift *d_model.Shift
	if len(shifts) > 0 {
		currentShift = shifts[0]
	}

	// 辅助函数：获取班次显示名称
	getShiftDisplayName := func(shiftID string) string {
		if name, ok := shiftIDToName[shiftID]; ok {
			return name
		}
		if len(shiftID) > 8 {
			return fmt.Sprintf("班次[%s...]", shiftID[:8])
		}
		return fmt.Sprintf("班次[%s]", shiftID)
	}

	// 辅助函数：获取人员显示名称
	getStaffDisplayName := func(staffID string) string {
		if len(staffID) > 8 {
			return fmt.Sprintf("人员[%s...]", staffID[:8])
		}
		return fmt.Sprintf("人员[%s]", staffID)
	}

	// 辅助函数：检查两个班次的时间是否真的重叠（超过1小时）
	checkTimeOverlap := func(shift1, shift2 *d_model.Shift) bool {
		if shift1 == nil || shift2 == nil {
			return false // 无法判断时不认为冲突
		}

		// 解析时间
		parseTime := func(timeStr string) int {
			parts := strings.Split(timeStr, ":")
			if len(parts) >= 2 {
				h, _ := strconv.Atoi(parts[0])
				m, _ := strconv.Atoi(parts[1])
				return h*60 + m
			}
			return 0
		}

		s1 := parseTime(shift1.StartTime)
		e1 := parseTime(shift1.EndTime)
		s2 := parseTime(shift2.StartTime)
		e2 := parseTime(shift2.EndTime)

		// 处理结束时间为00:00的情况（当作24:00）
		if e1 == 0 {
			e1 = 24 * 60
		}
		if e2 == 0 {
			e2 = 24 * 60
		}

		// 计算重叠时间
		overlapStart := s1
		if s2 > s1 {
			overlapStart = s2
		}
		overlapEnd := e1
		if e2 < e1 {
			overlapEnd = e2
		}
		overlap := overlapEnd - overlapStart
		if overlap < 0 {
			overlap = 0
		}

		// 重叠超过60分钟才算冲突
		return overlap > 60
	}

	// 检查已占位信息中的冲突（只有真正时间重叠才算冲突）
	for staffID, dates := range occupiedSlots {
		for date, occupiedShiftID := range dates {
			// 如果是同一个班次，不算冲突
			if currentShift != nil && occupiedShiftID == currentShift.ID {
				continue
			}

			// 检查该人员在该日期是否在排班草案中
			if staffIDs, ok := scheduleDraft.Schedule[date]; ok {
				for _, scheduledStaffID := range staffIDs {
					if scheduledStaffID == staffID {
						// 获取已占用的班次信息
						occupiedShift := shiftMap[occupiedShiftID]

						// 检查时间是否真的重叠
						hasRealOverlap := checkTimeOverlap(currentShift, occupiedShift)
						if !hasRealOverlap {
							// 时间不重叠，不算冲突（如下夜班00:00-08:00和夜班20:00-24:00）
							continue
						}

						// 有真实的时间重叠，报告冲突
						staffDisplayName := getStaffDisplayName(staffID)
						shiftDisplayName := getShiftDisplayName(occupiedShiftID)
						result.Passed = false
						result.ShiftRuleIssues = append(result.ShiftRuleIssues, &d_model.ValidationIssue{
							Type:           "shift_rule",
							Severity:       "high",
							Description:    fmt.Sprintf("人员 %s 在日期 %s 已被班次 %s 占位（时间冲突），不能重复分配", staffDisplayName, date, shiftDisplayName),
							AffectedDates:  []string{date},
							AffectedStaff:  []string{staffID},
							AffectedShifts: []string{occupiedShiftID},
						})
					}
				}
			}
		}
	}

	if len(result.ShiftRuleIssues) == 0 {
		result.Summary = "班次规则校验通过：未发现人员冲突"
	} else {
		result.Summary = fmt.Sprintf("班次规则校验失败：发现 %d 个冲突问题", len(result.ShiftRuleIssues))
	}

	return result, nil
}

// ValidateRuleCompliance 规则合规性校验
func (v *RuleLevelValidator) ValidateRuleCompliance(ctx context.Context, scheduleDraft *d_model.ShiftScheduleDraft, rules []*d_model.Rule, staffList []*d_model.Employee, fixedShiftAssignments map[string][]string) (*d_model.RuleValidationResult, error) {
	result := &d_model.RuleValidationResult{
		Passed:               true,
		RuleComplianceIssues: make([]*d_model.ValidationIssue, 0),
	}

	if scheduleDraft == nil || scheduleDraft.Schedule == nil {
		result.Passed = false
		result.Summary = "排班草案为空"
		return result, nil
	}

	// 构建人员ID到人员对象的映射
	staffMap := make(map[string]*d_model.Employee)
	staffIDToName := make(map[string]string)
	for _, staff := range staffList {
		if staff != nil {
			staffMap[staff.ID] = staff
			if staff.ID != "" && staff.Name != "" {
				staffIDToName[staff.ID] = staff.Name
			}
		}
	}

	// 辅助函数：获取人员显示名称
	getStaffDisplayName := func(staffID string) string {
		if name, ok := staffIDToName[staffID]; ok {
			return name
		}
		// 如果找不到姓名，显示UUID的简化格式（前8位）
		if len(staffID) > 8 {
			return fmt.Sprintf("人员[%s...]", staffID[:8])
		}
		return fmt.Sprintf("人员[%s]", staffID)
	}

	// 【P1修复】构建固定排班人员日期集合，用于豁免检查
	fixedStaffDates := make(map[string]map[string]bool)
	for date, staffIDs := range fixedShiftAssignments {
		for _, staffID := range staffIDs {
			if fixedStaffDates[staffID] == nil {
				fixedStaffDates[staffID] = make(map[string]bool)
			}
			fixedStaffDates[staffID][date] = true
		}
	}

	// 简化实现：检查基本规则合规性
	// 实际应该根据规则类型进行更详细的校验
	for _, rule := range rules {
		if rule == nil || !rule.IsActive {
			continue
		}

		// 根据规则类型进行校验
		switch rule.RuleType {
		case "maxCount":
			// 检查最大次数规则
			if rule.MaxCount != nil {
				// 统计每个人员在排班周期内的排班次数
				staffCountMap := make(map[string]int)
				// 【P1修复】同时记录非固定排班次数（固定排班不计入）
				staffDynamicCountMap := make(map[string]int)
				for date, staffIDs := range scheduleDraft.Schedule {
					for _, staffID := range staffIDs {
						staffCountMap[staffID]++
						// 检查是否为固定排班
						if _, isFixed := fixedStaffDates[staffID][date]; !isFixed {
							staffDynamicCountMap[staffID]++
						}
					}
				}

				for staffID, count := range staffCountMap {
					// 【P1修复】对固定排班人员使用更宽松的检查：只检查动态排班部分
					dynamicCount := staffDynamicCountMap[staffID]
					if dynamicCount > *rule.MaxCount {
						staffDisplayName := getStaffDisplayName(staffID)
						result.Passed = false
						result.RuleComplianceIssues = append(result.RuleComplianceIssues, &d_model.ValidationIssue{
							Type:          "rule_compliance",
							Severity:      "high",
							Description:   fmt.Sprintf("规则 %s：人员 %s 动态排班次数 %d 超过最大限制 %d（固定排班 %d 次，总计 %d 次）", rule.Name, staffDisplayName, dynamicCount, *rule.MaxCount, count-dynamicCount, count),
							AffectedStaff: []string{staffID},
						})
					}
				}
			}

		case "consecutiveMax":
			// 检查连续天数规则
			if rule.ConsecutiveMax != nil {
				// 统计每个人员的连续排班天数
				// 【P1修复】固定排班不计入连续天数统计
				// 这里简化实现，实际需要按日期顺序检查
				// TODO: 实现更详细的连续天数检查
			}

		case "forbidden_day":
			// 检查禁止日期规则
			// 【P1修复】固定排班豁免禁止日期检查
			// TODO: 实现禁止日期检查

		default:
			// 其他规则类型的校验
			// TODO: 根据具体规则类型实现
		}
	}

	if len(result.RuleComplianceIssues) == 0 {
		result.Summary = "规则合规性校验通过：未发现规则违反"
	} else {
		result.Summary = fmt.Sprintf("规则合规性校验失败：发现 %d 个规则违反", len(result.RuleComplianceIssues))
	}

	return result, nil
}

// ValidateAll 综合校验（人数、班次、规则合规性）
func (v *RuleLevelValidator) ValidateAll(ctx context.Context, scheduleDraft *d_model.ShiftScheduleDraft, staffRequirements map[string]int, shifts []*d_model.Shift, rules []*d_model.Rule, staffList []*d_model.Employee, occupiedSlots map[string]map[string]string, fixedShiftAssignments map[string][]string) (*d_model.RuleValidationResult, error) {
	startTime := time.Now()

	// 执行各项校验
	staffCountResult, err := v.ValidateStaffCount(ctx, scheduleDraft, staffRequirements)
	if err != nil {
		return nil, fmt.Errorf("人数校验失败: %w", err)
	}

	shiftRuleResult, err := v.ValidateShiftRules(ctx, scheduleDraft, shifts, occupiedSlots)
	if err != nil {
		return nil, fmt.Errorf("班次规则校验失败: %w", err)
	}

	ruleComplianceResult, err := v.ValidateRuleCompliance(ctx, scheduleDraft, rules, staffList, fixedShiftAssignments)
	if err != nil {
		return nil, fmt.Errorf("规则合规性校验失败: %w", err)
	}

	// 合并结果
	result := &d_model.RuleValidationResult{
		Passed:               staffCountResult.Passed && shiftRuleResult.Passed && ruleComplianceResult.Passed,
		StaffCountIssues:     staffCountResult.StaffCountIssues,
		ShiftRuleIssues:      shiftRuleResult.ShiftRuleIssues,
		RuleComplianceIssues: ruleComplianceResult.RuleComplianceIssues,
	}

	// 生成综合总结
	totalIssues := len(result.StaffCountIssues) + len(result.ShiftRuleIssues) + len(result.RuleComplianceIssues)
	highSeverityCount := 0
	for _, issue := range result.StaffCountIssues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}
	for _, issue := range result.ShiftRuleIssues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}
	for _, issue := range result.RuleComplianceIssues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}

	duration := time.Since(startTime).Seconds()
	if result.Passed {
		result.Summary = fmt.Sprintf("综合校验通过（耗时 %.2f 秒）：所有校验项均通过", duration)
	} else {
		result.Summary = fmt.Sprintf("综合校验失败（耗时 %.2f 秒）：发现 %d 个问题（%d 个严重问题）", duration, totalIssues, highSeverityCount)
	}

	v.logger.Info("规则级校验完成",
		"passed", result.Passed,
		"totalIssues", totalIssues,
		"highSeverityCount", highSeverityCount,
		"duration", duration)

	return result, nil
}
