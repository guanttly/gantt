package executor

import (
	"context"
	"fmt"
	"time"

	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/internal/engine"
	"jusha/mcp/pkg/logging"
)

// ============================================================
// SingleShiftScheduler — 单班次原子排班模块
//
// 职责单一：为班次 A 在日期 D 按规则排 X 人。
// 所有排人操作（正排、补排、置换、LLM 调整后重排）统一走此模块。
// 不包含任何依赖解析逻辑——依赖协调由 DependencyResolver 负责。
// ============================================================

// SingleShiftInput 单班次原子排班输入
type SingleShiftInput struct {
	ShiftID       string                 // 目标班次ID
	Date          string                 // 目标日期 YYYY-MM-DD
	RequiredCount int                    // 需要排入的人数
	Source        model.AssignmentSource // 分配来源标记（fixed/rule/dependency/default）

	// 候选人约束（可选）
	Whitelist []string // 白名单：仅从这些人中选（为空则使用班次成员池/全员池）
	Blacklist []string // 黑名单：排除这些人

	// 置换模式（可选）
	// 当目标班次已满但需要为依赖补排时，传入可被置换的低优先级人员ID
	// SingleShiftScheduler 会先移除这些人员腾出名额，再选入新人
	Replaceable []string
}

// SingleShiftResult 单班次原子排班结果
type SingleShiftResult struct {
	Assignments []*model.StaffAssignment // 选中的人员分配列表
	Replaced    []*model.StaffAssignment // 被置换的人员列表（仅置换模式下有值）
}

// SchedulingError 排班失败的结构化错误
type SchedulingError struct {
	ShiftID      string   // 班次ID
	ShiftName    string   // 班次名称
	Date         string   // 日期
	Required     int      // 需要人数
	Available    int      // 可用人数
	Deficit      int      // 缺少人数
	Reason       string   // 失败原因
	RelatedRules []string // 关联规则ID
	Detail       string   // 诊断详情（主体/客体班次人员快照）
}

func (e *SchedulingError) Error() string {
	return fmt.Sprintf("排班失败 [%s/%s] %s: 需要%d人, 可用%d人, 缺少%d人 - %s",
		e.ShiftName, e.Date, e.ShiftID, e.Required, e.Available, e.Deficit, e.Reason)
}

// SingleShiftScheduler 单班次原子排班器
type SingleShiftScheduler struct {
	logger     logging.ILogger
	ruleEngine *engine.RuleEngine
}

// NewSingleShiftScheduler 创建单班次原子排班器
func NewSingleShiftScheduler(logger logging.ILogger, ruleEngine *engine.RuleEngine) *SingleShiftScheduler {
	return &SingleShiftScheduler{
		logger:     logger,
		ruleEngine: ruleEngine,
	}
}

// Schedule 为指定班次-日期排入所需人数（原子操作）
//
// 核心流程：
//  1. 确定候选池（白名单优先，否则班次成员池/全员池）
//  2. 排除黑名单
//  3. 置换模式：先移除 Replaceable 中的低优先级人员腾出名额
//  4. RuleEngine.PrepareSchedulingContext 过滤 + 评分 + 约束校验
//  5. 选取 top-N 候选人
//  6. 候选不足时返回结构化 SchedulingError
func (s *SingleShiftScheduler) Schedule(
	ctx context.Context,
	input *SingleShiftInput,
	execInput *SchedulingExecutionInput,
) (*SingleShiftResult, error) {
	if input.RequiredCount <= 0 {
		return &SingleShiftResult{}, nil
	}

	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return nil, fmt.Errorf("解析日期失败: %w", err)
	}

	shift := findShiftByID(execInput.Shifts, input.ShiftID)
	shiftName := input.ShiftID
	if shift != nil {
		shiftName = shift.Name
	}

	result := &SingleShiftResult{}

	// ── 1. 置换模式预处理：从 draft 中移除可置换的低优先级人员 ──
	if len(input.Replaceable) > 0 && execInput.CurrentDraft != nil {
		result.Replaced = s.executeReplacement(execInput.CurrentDraft, input.ShiftID, input.Date, input.Replaceable)
	}

	// ── 2. 确定候选池 ──
	baseStaff := s.buildCandidatePool(input, execInput)

	// ── 3. 排除黑名单 & 已排班 & 禁排 & 不可用 ──
	filteredStaff := s.filterCandidates(baseStaff, input, execInput)

	// ── 4. 准备固定排班（用于 RuleEngine 感知） ──
	var fixedAssignments []*model.CtxFixedShiftAssignment
	for _, fa := range execInput.FixedAssignments {
		if fa.Date == input.Date {
			fixedAssignments = append(fixedAssignments, fa)
		}
	}

	// ── 5. 调用 RuleEngine 进行过滤 + 评分 + 约束校验 ──
	schedulingInput := &engine.SchedulingInput{
		AllStaff:         filteredStaff,
		AllRules:         execInput.Rules,
		PersonalNeeds:    execInput.PersonalNeeds,
		FixedAssignments: fixedAssignments,
		CurrentDraft:     execInput.CurrentDraft,
		ShiftID:          input.ShiftID,
		Date:             date,
		RequiredCount:    input.RequiredCount,
		AllShifts:        execInput.Shifts,
		TargetShift:      shift,
	}

	schedulingCtx, err := s.ruleEngine.PrepareSchedulingContext(ctx, schedulingInput)
	if err != nil {
		return nil, fmt.Errorf("规则引擎预处理失败 [%s/%s]: %w", shiftName, input.Date, err)
	}

	// ── 6. 选取 top-N 候选人（去重保护）──
	selected := make([]*model.StaffAssignment, 0, input.RequiredCount)
	selectedSet := make(map[string]bool, input.RequiredCount)
	priority := model.SourceToPriority(input.Source)

	for i := 0; len(selected) < input.RequiredCount && i < len(schedulingCtx.EligibleCandidates); i++ {
		candidate := schedulingCtx.EligibleCandidates[i]
		if !selectedSet[candidate.StaffID] {
			selected = append(selected, &model.StaffAssignment{
				StaffID:   candidate.StaffID,
				StaffName: candidate.StaffName,
				Source:    input.Source,
				Priority:  priority,
			})
			selectedSet[candidate.StaffID] = true
		}
	}

	// ── 7. 检查是否满足需求 ──
	if len(selected) < input.RequiredCount {
		deficit := input.RequiredCount - len(selected)

		// 收集排除原因
		reason := s.buildExclusionSummary(schedulingCtx, len(filteredStaff))

		s.logger.Warn("SingleShiftScheduler: 候选人不足",
			"shiftID", input.ShiftID,
			"shiftName", shiftName,
			"date", input.Date,
			"required", input.RequiredCount,
			"available", len(selected),
			"deficit", deficit,
			"source", input.Source,
			"reason", reason,
		)

		// 对于依赖补排（dependency），人数不足是硬错误，必须报告
		if input.Source == model.AssignmentSourceDependency {
			return nil, &SchedulingError{
				ShiftID:   input.ShiftID,
				ShiftName: shiftName,
				Date:      input.Date,
				Required:  input.RequiredCount,
				Available: len(selected),
				Deficit:   deficit,
				Reason:    reason,
			}
		}
		// 对于其他来源（default/rule），人数不足记录 warning 但不报错
		// 调用方可根据 len(Assignments) < RequiredCount 自行判断
	}

	result.Assignments = selected
	return result, nil
}

// buildCandidatePool 构建候选人池
func (s *SingleShiftScheduler) buildCandidatePool(
	input *SingleShiftInput,
	execInput *SchedulingExecutionInput,
) []*model.Staff {
	// 白名单模式：仅从白名单中选人
	if len(input.Whitelist) > 0 {
		whiteSet := make(map[string]bool, len(input.Whitelist))
		for _, id := range input.Whitelist {
			whiteSet[id] = true
		}
		// 从 AllStaff 中提取白名单匹配的 Staff 对象
		var pool []*model.Staff
		for _, staff := range execInput.AllStaff {
			if whiteSet[staff.ID] {
				pool = append(pool, staff)
			}
		}
		return pool
	}

	// 班次成员池模式
	if members, ok := execInput.ShiftMembersMap[input.ShiftID]; ok && len(members) > 0 {
		return members
	}

	// 全员池
	return execInput.AllStaff
}

// filterCandidates 过滤候选人（黑名单 + 已排班 + 禁排 + 不可用）
func (s *SingleShiftScheduler) filterCandidates(
	baseStaff []*model.Staff,
	input *SingleShiftInput,
	execInput *SchedulingExecutionInput,
) []*model.Staff {
	blackSet := make(map[string]bool, len(input.Blacklist))
	for _, id := range input.Blacklist {
		blackSet[id] = true
	}

	var filtered []*model.Staff
	for _, staff := range baseStaff {
		// 黑名单排除
		if blackSet[staff.ID] {
			continue
		}

		// 已排在同班次同日期
		if isAlreadyScheduledOnShift(staff.ID, input.ShiftID, input.Date, execInput) {
			continue
		}

		// 局部重排黑名单（UnavailableStaffMap）
		if execInput.UnavailableStaffMap != nil &&
			execInput.UnavailableStaffMap[input.Date] != nil &&
			execInput.UnavailableStaffMap[input.Date][input.ShiftID] != nil &&
			execInput.UnavailableStaffMap[input.Date][input.ShiftID][staff.ID] {
			continue
		}

		// 个人/规则层面的"禁止排班"
		if isForbiddenOnShift(staff.ID, input.ShiftID, input.Date, execInput) {
			continue
		}

		filtered = append(filtered, staff)
	}
	return filtered
}

// executeReplacement 执行置换：从 draft 中移除 Replaceable 的低优先级人员
func (s *SingleShiftScheduler) executeReplacement(
	draft *model.ScheduleDraft,
	shiftID string,
	date string,
	replaceable []string,
) []*model.StaffAssignment {
	if draft == nil || draft.Shifts == nil {
		return nil
	}
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok || shiftDraft == nil || shiftDraft.Days == nil {
		return nil
	}
	dayShift, ok := shiftDraft.Days[date]
	if !ok || dayShift == nil {
		return nil
	}

	var replaced []*model.StaffAssignment
	for _, staffID := range replaceable {
		// 如果有 Assignments，优先通过 Assignments 操作
		if len(dayShift.Assignments) > 0 {
			for _, a := range dayShift.Assignments {
				if a.StaffID == staffID && a.Priority == model.AssignmentPriorityLow {
					if dayShift.RemoveStaffByID(staffID) {
						replaced = append(replaced, a)
						s.logger.Info("SingleShiftScheduler: 置换低优先级人员",
							"shiftID", shiftID, "date", date,
							"replacedStaffID", staffID, "replacedStaffName", a.StaffName)
					}
					break
				}
			}
		} else {
			// 兼容模式：Assignments 为空时直接操作 StaffIDs
			for i, sid := range dayShift.StaffIDs {
				if sid == staffID {
					name := ""
					if i < len(dayShift.Staff) {
						name = dayShift.Staff[i]
					}
					dayShift.StaffIDs = append(dayShift.StaffIDs[:i], dayShift.StaffIDs[i+1:]...)
					if i < len(dayShift.Staff) {
						dayShift.Staff = append(dayShift.Staff[:i], dayShift.Staff[i+1:]...)
					}
					dayShift.ActualCount = len(dayShift.StaffIDs)
					replaced = append(replaced, &model.StaffAssignment{
						StaffID:   staffID,
						StaffName: name,
						Source:    model.AssignmentSourceDefault,
						Priority:  model.AssignmentPriorityLow,
					})
					s.logger.Info("SingleShiftScheduler: 置换低优先级人员(兼容模式)",
						"shiftID", shiftID, "date", date,
						"replacedStaffID", staffID)
					break
				}
			}
		}
	}
	return replaced
}

// buildExclusionSummary 构建候选人排除原因摘要
func (s *SingleShiftScheduler) buildExclusionSummary(
	schedulingCtx *engine.SchedulingContext,
	poolSize int,
) string {
	if poolSize == 0 {
		return "候选人池为空，请检查班次成员配置"
	}

	exclusionCount := make(map[string]int)
	for _, reason := range schedulingCtx.ExclusionReasons {
		exclusionCount[reason]++
	}
	for _, excluded := range schedulingCtx.ExcludedCandidates {
		for _, v := range excluded.ViolatedRules {
			key := "约束违反:" + v.RuleName + "(" + v.Message + ")"
			exclusionCount[key]++
		}
	}

	if len(exclusionCount) == 0 {
		return fmt.Sprintf("候选人池%d人经规则引擎过滤后无合格人选", poolSize)
	}

	parts := make([]string, 0, len(exclusionCount))
	for reason, count := range exclusionCount {
		parts = append(parts, fmt.Sprintf("%s(%d人)", reason, count))
	}
	return fmt.Sprintf("候选人池%d人, 排除原因: %s", poolSize, joinStrings(parts, "; "))
}

// joinStrings 辅助函数：连接字符串切片
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
