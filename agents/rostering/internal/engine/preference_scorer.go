package engine

import (
	"sort"
	"strings"
	"time"

	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// PreferenceScorer 偏好评分器
type PreferenceScorer struct {
	logger logging.ILogger
}

// NewPreferenceScorer 创建偏好评分器
func NewPreferenceScorer(logger logging.ILogger) *PreferenceScorer {
	return &PreferenceScorer{logger: logger}
}

// Score 为每个候选人计算偏好得分
// shiftID 当前排班班次，用于本班次周期内已排天数均衡（能岔开就岔开）
func (s *PreferenceScorer) Score(
	candidates []*CandidateStatus,
	preferenceRules []*ClassifiedRule,
	draft *model.ScheduleDraft,
	date time.Time,
	shiftID string,
) *PreferenceScoreResult {
	result := &PreferenceScoreResult{
		Scores: make(map[string]float64),
	}

	for _, candidate := range candidates {
		score := 0.0

		// 1. 偏好规则评分
		for _, rule := range preferenceRules {
			ruleScore := s.computeRulePreferenceScore(candidate, rule, draft, date)
			// 根据规则优先级加权（优先级越高权重越大）
			score += ruleScore * float64(rule.Rule.Priority) / 10.0
		}

		// 2. 约束剩余空间评分（ConstraintScores 越高表示越宽松，更适合排班）
		constraintBonus := 0.0
		if len(candidate.ConstraintScores) > 0 {
			totalMargin := 0.0
			for _, margin := range candidate.ConstraintScores {
				totalMargin += margin
			}
			constraintBonus = totalMargin / float64(len(candidate.ConstraintScores))
		} else {
			constraintBonus = 1.0 // 无约束时满分
		}
		score += constraintBonus * 0.5 // 约束余量权重 0.5

		// 3. 负载均衡评分：整周已排班次数越少，评分越高
		workloadScore := s.computeWorkloadBalance(candidate.StaffID, draft, date)
		score += workloadScore * 0.2 // 负载均衡权重 0.2

		// 4. 当前班次本周期内已排天数均衡：该班次已排天数越少评分越高（能岔开就岔开）
		shiftBalanceScore := s.computeShiftBalance(candidate.StaffID, shiftID, draft)
		score += shiftBalanceScore * 0.3 // 班次内岔开权重 0.3

		result.Scores[candidate.StaffID] = score
	}

	return result
}

// SortCandidatesByScore 按偏好评分对候选人列表进行降序排序
func (s *PreferenceScorer) SortCandidatesByScore(
	candidates []*CandidateStatus,
	scores *PreferenceScoreResult,
) {
	if scores == nil || len(scores.Scores) == 0 {
		return
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		scoreI := scores.Scores[candidates[i].StaffID]
		scoreJ := scores.Scores[candidates[j].StaffID]
		return scoreI > scoreJ // 降序：评分高的排前面
	})
}

// computeRulePreferenceScore 计算单个规则的偏好得分
func (s *PreferenceScorer) computeRulePreferenceScore(
	candidate *CandidateStatus,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	date time.Time,
) float64 {
	if rule == nil || rule.Rule == nil {
		return 0.0
	}

	r := rule.Rule

	// 检查规则是否关联到该候选人
	isStaffAssociated := false
	isShiftAssociated := false
	for _, assoc := range r.Associations {
		if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == candidate.StaffID {
			isStaffAssociated = true
		}
		if assoc.AssociationType == model.AssociationTypeShift {
			isShiftAssociated = true
		}
	}

	switch rule.SubCategory {
	case "prefer":
		// 偏好型规则：关联到该员工则加分
		if isStaffAssociated {
			return 1.0 // 强偏好
		}
		if isShiftAssociated {
			return 0.3 // 班次偏好（弱关联）
		}
		// 全局偏好规则
		return s.evaluateGlobalPreference(candidate, rule, draft, date)

	case "suggest":
		// 建议型规则：轻微加分
		if isStaffAssociated {
			return 0.5
		}
		return 0.2

	case "forbid", "avoid":
		// 回避/禁止型偏好：关联到该员工则扣分
		if isStaffAssociated {
			return -1.0
		}
		return 0.0

	default:
		return 0.0
	}
}

// evaluateGlobalPreference 评估全局偏好规则
func (s *PreferenceScorer) evaluateGlobalPreference(
	candidate *CandidateStatus,
	rule *ClassifiedRule,
	draft *model.ScheduleDraft,
	date time.Time,
) float64 {
	r := rule.Rule
	ruleType := strings.ToLower(r.RuleType)

	switch ruleType {
	case "preferred":
		// 通用偏好：根据描述关键词推断
		desc := strings.ToLower(r.Description + r.RuleData)

		// 周末相关偏好
		if strings.Contains(desc, "周末") || strings.Contains(desc, "weekend") {
			weekday := date.Weekday()
			isWeekend := weekday == time.Saturday || weekday == time.Sunday
			if strings.Contains(desc, "休息") || strings.Contains(desc, "避免") {
				if isWeekend {
					return -0.5 // 周末偏好休息时，排周末扣分
				}
				return 0.3
			}
		}

		// 连续工作相关偏好
		if strings.Contains(desc, "连续") || strings.Contains(desc, "均衡") {
			return s.computeWorkloadBalance(candidate.StaffID, draft, date)
		}

		return 0.3 // 默认偏好得分

	case "combinable":
		// 可组合规则（例如：某些班次适合搭配）
		return 0.2

	default:
		return 0.0
	}
}

// computeWorkloadBalance 计算负载均衡评分
// 在当前排班周期内，已排班次越少，评分越高
func (s *PreferenceScorer) computeWorkloadBalance(
	staffID string,
	draft *model.ScheduleDraft,
	date time.Time,
) float64 {
	if draft == nil || draft.Shifts == nil {
		return 1.0 // 无排班记录时满分
	}

	// 统计当前周内的排班次数
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := date.AddDate(0, 0, -(weekday - 1))
	weekEnd := weekStart.AddDate(0, 0, 6)

	totalCount := 0
	for _, shiftDraft := range draft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}
		for dateStr, dayShift := range shiftDraft.Days {
			d, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			if d.Before(weekStart) || d.After(weekEnd) {
				continue
			}
			for _, sid := range dayShift.StaffIDs {
				if sid == staffID {
					totalCount++
					break
				}
			}
		}
	}

	// 一周最多 7 天，已排越多评分越低
	if totalCount >= 7 {
		return 0.0
	}
	return float64(7-totalCount) / 7.0
}

// computeShiftBalance 计算当前班次本周期内已排天数均衡评分
// 该班次已排该人员的天数越少，评分越高，实现多天岔开、尽量均匀
func (s *PreferenceScorer) computeShiftBalance(
	staffID string,
	shiftID string,
	draft *model.ScheduleDraft,
) float64 {
	if shiftID == "" || draft == nil || draft.Shifts == nil {
		return 1.0
	}
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok || shiftDraft == nil || shiftDraft.Days == nil {
		return 1.0
	}

	var startDate, endDate time.Time
	if draft.StartDate != "" && draft.EndDate != "" {
		var err error
		startDate, err = time.Parse("2006-01-02", draft.StartDate)
		if err != nil {
			return 1.0
		}
		endDate, err = time.Parse("2006-01-02", draft.EndDate)
		if err != nil {
			return 1.0
		}
		if endDate.Before(startDate) {
			return 1.0
		}
	} else {
		// 无周期时按当前周
		weekday := int(time.Now().Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = time.Now().AddDate(0, 0, -(weekday - 1))
		endDate = startDate.AddDate(0, 0, 6)
	}

	periodDays := int(endDate.Sub(startDate).Hours()/24) + 1
	if periodDays <= 0 {
		return 1.0
	}

	count := 0
	for dateStr, dayShift := range shiftDraft.Days {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if d.Before(startDate) || d.After(endDate) {
			continue
		}
		for _, sid := range dayShift.StaffIDs {
			if sid == staffID {
				count++
				break
			}
		}
	}

	if count >= periodDays {
		return 0.0
	}
	return float64(periodDays-count) / float64(periodDays)
}
