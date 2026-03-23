package engine

import (
	"time"
	"jusha/agent/rostering/domain/model"
)

// SchedulingInput 排班输入
type SchedulingInput struct {
	AllStaff          []*model.Staff
	AllRules          []*model.Rule
	PersonalNeeds     []*model.PersonalNeed
	FixedAssignments  []*model.CtxFixedShiftAssignment // 使用 CtxFixedShiftAssignment
	CurrentDraft      *model.ScheduleDraft
	ShiftID           string
	Date              time.Time
	RequiredCount     int
	AllShifts         []*model.Shift // 所有班次列表（用于时间重叠检查）
	TargetShift       *model.Shift   // 目标班次信息（用于时间重叠检查）
}

// SchedulingContext 排班上下文（传递给LLM）
type SchedulingContext struct {
	ShiftID            string
	Date               time.Time
	RequiredCount       int
	MatchedRules       *MatchedRules
	EligibleCandidates []*CandidateStatus
	ExcludedCandidates []*CandidateStatus
	ExclusionReasons   map[string]string
	ConstraintDetails   []*ConstraintDetail
	PreferenceScores    *PreferenceScoreResult
	LLMBrief            *LLMBrief
}

// MatchedRules 匹配的规则
type MatchedRules struct {
	ConstraintRules []*ClassifiedRule
	PreferenceRules []*ClassifiedRule
	DependencyRules []*ClassifiedRule
}

// ClassifiedRule 分类后的规则
type ClassifiedRule struct {
	Rule            *model.Rule
	Category        string   // constraint/preference/dependency
	SubCategory     string   // forbid/must/limit/prefer/suggest/source/resource/order
	Dependencies    []string // 依赖的其他规则ID
	Conflicts       []string // 冲突的其他规则ID
	ExecutionOrder  int      // 执行顺序（数字越小越先执行）
}

// CandidateStatus 候选人约束状态
type CandidateStatus struct {
	StaffID          string
	StaffName        string
	IsEligible       bool                  // 是否通过所有硬约束
	ViolatedRules    []*RuleViolation     // 违反的规则列表
	Warnings         []*RuleWarning       // 警告（接近限制）
	ConstraintScores map[string]float64   // 各约束的"剩余空间"评分
}

// RuleViolation 规则违反
type RuleViolation struct {
	RuleID   string
	RuleName string
	IsHard   bool
	Message  string
}

// RuleWarning 规则警告
type RuleWarning struct {
	RuleID   string
	RuleName string
	Message  string
}

// ConstraintDetail 约束详情
type ConstraintDetail struct {
	RuleID    string
	RuleName  string
	RuleType  string
	StaffID   string
	StaffName string
	Status    string // pass/violate/warning
	Message   string
}

// ConstraintCheckResult 约束检查结果
type ConstraintCheckResult struct {
	EligibleCandidates []*CandidateStatus
	ExcludedCandidates []*CandidateStatus
	Details            []*ConstraintDetail
}

// PreferenceScoreResult 偏好评分结果
type PreferenceScoreResult struct {
	Scores map[string]float64 // staffID -> score
}

// LLMBrief 传递给LLM的结构化摘要
type LLMBrief struct {
	Candidates         []*CandidateBrief
	HardConstraints    []*ConstraintBrief
	SoftPreferences    []*PreferenceBrief
	ExcludedWithReasons []*ExclusionBrief
}

// CandidateBrief 候选人摘要
type CandidateBrief struct {
	ShortID          string
	Name             string
	PreferenceScore  float64
	ConstraintMargin float64
	WeeklyCount      int
	Note             string
}

// ConstraintBrief 约束摘要
type ConstraintBrief struct {
	RuleID      string
	Description string
	Type        string
	Limit       interface{}
}

// PreferenceBrief 偏好摘要
type PreferenceBrief struct {
	Description string
	Weight      int
}

// ExclusionBrief 排除原因摘要
type ExclusionBrief struct {
	Name   string
	Reason string
}
