package step

import (
	"context"
	"time"

	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/shift"
)

// Step 排班管道步骤接口。
type Step interface {
	Name() string
	Execute(ctx context.Context, state *ScheduleState) error
}

// ScheduleState 排班管道共享状态，在 Pipeline 各 Step 之间流转。
type ScheduleState struct {
	// ── 输入 ──
	ScheduleID     string
	OrgNodeID      string
	GroupID        string // 排班分组ID，为空时取全节点员工
	StartDate      string
	EndDate        string
	CreatedBy      string
	Config         *ScheduleConfig
	EffectiveRules []rule.Rule

	// ── 中间状态 ──
	ShiftOrder []shift.Shift             // 拓扑排序后的班次执行顺序
	Candidates map[string][]string       // "shiftID|date" → 候选员工 ID 列表
	Scores     map[string]map[string]int // "shiftID|date" → employeeID → 偏好评分

	// ── 编辑输入（调整 Pipeline 使用）──
	EditInput *EditInput

	// ── 输出 ──
	Assignments []Assignment
	Violations  []Violation

	// ── 进度回调 ──
	OnProgress func(step string, progress float64, message string)
}

// NewScheduleState 创建排班管道状态。
func NewScheduleState(scheduleID, orgNodeID, groupID, startDate, endDate, createdBy string, config *ScheduleConfig) *ScheduleState {
	return &ScheduleState{
		ScheduleID:  scheduleID,
		OrgNodeID:   orgNodeID,
		GroupID:     groupID,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedBy:   createdBy,
		Config:      config,
		Candidates:  make(map[string][]string),
		Scores:      make(map[string]map[string]int),
		Assignments: make([]Assignment, 0),
		Violations:  make([]Violation, 0),
	}
}

// Assignment 排班分配（Pipeline 内使用的轻量结构）。
type Assignment struct {
	ID         string
	ScheduleID string
	EmployeeID string
	ShiftID    string
	Date       string
	Source     string
	OrgNodeID  string
}

// Violation 规则违反记录。
type Violation struct {
	AssignmentID string `json:"assignment_id"`
	EmployeeID   string `json:"employee_id"`
	ShiftID      string `json:"shift_id"`
	Date         string `json:"date"`
	RuleID       string `json:"rule_id"`
	RuleName     string `json:"rule_name"`
	Reason       string `json:"reason"`
}

// ScheduleConfig 排班配置。
type ScheduleConfig struct {
	ShiftIDs     []string                  `json:"shift_ids"`
	Requirements map[string]map[string]int `json:"requirements"`
	Preferences  []PersonalPreference      `json:"preferences,omitempty"`
}

// PersonalPreference 个人排班偏好。
type PersonalPreference struct {
	EmployeeID string `json:"employee_id"`
	ShiftID    string `json:"shift_id"`
	Date       string `json:"date"`
	Weight     int    `json:"weight"`
}

// EditInput 排班调整输入。
type EditInput struct {
	Adds     []EditAddItem    `json:"adds,omitempty"`
	Removes  []string         `json:"removes,omitempty"`
	Modifies []EditModifyItem `json:"modifies,omitempty"`
}

// EditAddItem 新增排班项。
type EditAddItem struct {
	EmployeeID string `json:"employee_id"`
	ShiftID    string `json:"shift_id"`
	Date       string `json:"date"`
}

// EditModifyItem 修改排班项。
type EditModifyItem struct {
	AssignmentID string `json:"assignment_id"`
	EmployeeID   string `json:"employee_id,omitempty"`
	ShiftID      string `json:"shift_id,omitempty"`
	Date         string `json:"date,omitempty"`
}

// ── 排班来源常量 ──

const (
	SourceFixed  = "fixed"
	SourceRule   = "rule"
	SourceFill   = "fill"
	SourceAI     = "ai"
	SourceManual = "manual"
)

// ── 辅助方法 ──

// IsOccupied 检查某员工某天是否已被排班。
func (s *ScheduleState) IsOccupied(employeeID, date string) bool {
	for _, a := range s.Assignments {
		if a.EmployeeID == employeeID && a.Date == date {
			return true
		}
	}
	return false
}

// IsOccupiedForShift 检查某员工某天某班次是否已被排班。
func (s *ScheduleState) IsOccupiedForShift(employeeID, shiftID, date string) bool {
	for _, a := range s.Assignments {
		if a.EmployeeID == employeeID && a.ShiftID == shiftID && a.Date == date {
			return true
		}
	}
	return false
}

// CountAssigned 统计某班次某天已分配的人数。
func (s *ScheduleState) CountAssigned(shiftID, date string) int {
	count := 0
	for _, a := range s.Assignments {
		if a.ShiftID == shiftID && a.Date == date {
			count++
		}
	}
	return count
}

// ParseDate 解析日期字符串。
func ParseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}
