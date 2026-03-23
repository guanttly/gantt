package engine

import (
	"context"
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// RuleEngine 确定性规则引擎
// 替代 V3 中的 LLM-1/LLM-2/LLM-3/LLM-5，用代码实现确定性规则计算
type RuleEngine struct {
	logger            logging.ILogger
	candidateFilter   *CandidateFilter   // 替代 LLM-1
	ruleMatcher       *RuleMatcher       // 替代 LLM-2
	constraintChecker *ConstraintChecker // 替代 LLM-3
	validator         *ScheduleValidator // 替代 LLM-5
	preferenceScorer  *PreferenceScorer  // 偏好评分
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(logger logging.ILogger) *RuleEngine {
	return &RuleEngine{
		logger:            logger,
		candidateFilter:   NewCandidateFilter(logger),
		ruleMatcher:       NewRuleMatcher(logger),
		constraintChecker: NewConstraintChecker(logger),
		validator:         NewScheduleValidator(logger),
		preferenceScorer:  NewPreferenceScorer(logger),
	}
}

// PrepareSchedulingContext 为单个班次单个日期准备排班上下文
// 替代 V3 的 3-LLM 预分析，全部用代码完成
func (e *RuleEngine) PrepareSchedulingContext(
	ctx context.Context,
	input *SchedulingInput,
) (*SchedulingContext, error) {

	// 1. 规则匹配（替代 LLM-2）：根据 Associations + Category 精确匹配
	matchedRules := e.ruleMatcher.MatchRules(input.AllRules, input.ShiftID, input.Date)

	// 2. 候选人过滤（替代 LLM-1）：根据请假、固定排班、已排班状态等确定性数据过滤
	candidates, exclusionReasons := e.candidateFilter.Filter(
		input.AllStaff,
		input.PersonalNeeds,
		input.FixedAssignments,
		input.CurrentDraft,
		input.ShiftID,
		input.Date,
		input.AllShifts,
		input.TargetShift,
	)

	// 3. 约束检查（替代 LLM-3）：根据结构化规则参数计算每个候选人的约束状态
	constraintResults := e.constraintChecker.CheckAll(
		candidates,
		matchedRules,
		input.CurrentDraft,
		input.ShiftID,
		input.Date,
		input.AllStaff,
	)

	// 4. 偏好评分：为每个候选人计算偏好得分（含当前班次本周期已排天数均衡，实现岔开）
	preferenceScores := e.preferenceScorer.Score(
		constraintResults.EligibleCandidates,
		matchedRules.PreferenceRules,
		input.CurrentDraft,
		input.Date,
		input.ShiftID,
	)

	// 5. 按偏好评分降序排序候选人（V4核心：确保 selectStaff 能取到最优候选人）
	e.preferenceScorer.SortCandidatesByScore(
		constraintResults.EligibleCandidates,
		preferenceScores,
	)

	return &SchedulingContext{
		ShiftID:            input.ShiftID,
		Date:               input.Date,
		RequiredCount:      input.RequiredCount,
		MatchedRules:       matchedRules,
		EligibleCandidates: constraintResults.EligibleCandidates,
		ExcludedCandidates: constraintResults.ExcludedCandidates,
		ExclusionReasons:   exclusionReasons,
		ConstraintDetails:  constraintResults.Details,
		PreferenceScores:   preferenceScores,
		LLMBrief:           e.buildLLMBrief(constraintResults, preferenceScores, matchedRules),
	}, nil
}

// ValidateSchedule 确定性校验排班结果（替代 LLM-5）
func (e *RuleEngine) ValidateSchedule(
	ctx context.Context,
	schedule *model.ScheduleDraft,
	rules *MatchedRules,
	allDraft *model.ScheduleDraft,
) (*ValidationResult, error) {
	return e.validator.Validate(schedule, rules, allDraft)
}

// buildLLMBrief 构建传递给 LLM 的结构化摘要
func (e *RuleEngine) buildLLMBrief(
	constraints *ConstraintCheckResult,
	preferences *PreferenceScoreResult,
	rules *MatchedRules,
) *LLMBrief {
	return &LLMBrief{
		Candidates:          buildCandidateBriefs(constraints.EligibleCandidates, preferences),
		HardConstraints:     buildConstraintBriefs(rules.ConstraintRules),
		SoftPreferences:     buildPreferenceBriefs(rules.PreferenceRules),
		ExcludedWithReasons: buildExclusionBriefs(constraints.ExcludedCandidates),
	}
}

// buildCandidateBriefs 构建候选人摘要
func buildCandidateBriefs(candidates []*CandidateStatus, preferences *PreferenceScoreResult) []*CandidateBrief {
	briefs := make([]*CandidateBrief, 0, len(candidates))
	for _, c := range candidates {
		score := 0.0
		if preferences != nil {
			score = preferences.Scores[c.StaffID]
		}
		briefs = append(briefs, &CandidateBrief{
			ShortID:          c.StaffID[:8],
			Name:             c.StaffName,
			PreferenceScore:  score,
			ConstraintMargin: computeAvgConstraintMargin(c.ConstraintScores),
			Note:             buildNote(c),
		})
	}
	return briefs
}

// buildConstraintBriefs 构建约束摘要
func buildConstraintBriefs(rules []*ClassifiedRule) []*ConstraintBrief {
	briefs := make([]*ConstraintBrief, 0, len(rules))
	for _, r := range rules {
		briefs = append(briefs, &ConstraintBrief{
			RuleID:      r.Rule.ID,
			Description: r.Rule.Description,
			Type:        string(r.Rule.RuleType),
			Limit:       getRuleLimit(r.Rule),
		})
	}
	return briefs
}

// buildPreferenceBriefs 构建偏好摘要
func buildPreferenceBriefs(rules []*ClassifiedRule) []*PreferenceBrief {
	briefs := make([]*PreferenceBrief, 0, len(rules))
	for _, r := range rules {
		briefs = append(briefs, &PreferenceBrief{
			Description: r.Rule.Description,
			Weight:      r.Rule.Priority,
		})
	}
	return briefs
}

// buildExclusionBriefs 构建排除原因摘要
func buildExclusionBriefs(excluded []*CandidateStatus) []*ExclusionBrief {
	briefs := make([]*ExclusionBrief, 0, len(excluded))
	for _, c := range excluded {
		reason := "违反约束"
		if len(c.ViolatedRules) > 0 {
			reason = c.ViolatedRules[0].Message
		}
		briefs = append(briefs, &ExclusionBrief{
			Name:   c.StaffName,
			Reason: reason,
		})
	}
	return briefs
}

// 辅助函数
func computeAvgConstraintMargin(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 1.0
	}
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}

func buildNote(c *CandidateStatus) string {
	if len(c.Warnings) > 0 {
		return c.Warnings[0].Message
	}
	return ""
}

func getRuleLimit(rule *model.Rule) interface{} {
	if rule.MaxCount != nil {
		return *rule.MaxCount
	}
	if rule.ConsecutiveMax != nil {
		return *rule.ConsecutiveMax
	}
	if rule.MinRestDays != nil {
		return *rule.MinRestDays
	}
	return nil
}
