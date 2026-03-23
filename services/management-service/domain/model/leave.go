package model

import (
	"time"
)

// LeaveRecord 假期记录领域模型
// 记录已成事实的请假情况，不包含审批流
type LeaveRecord struct {
	ID           string     `json:"id"`
	OrgID        string     `json:"orgId"`
	EmployeeID   string     `json:"employeeId"`
	EmployeeName string     `json:"employeeName,omitempty"` // 员工姓名（用于列表显示）
	Type         LeaveType  `json:"type"`
	StartDate    time.Time  `json:"startDate"`
	EndDate      time.Time  `json:"endDate"`
	Days         float64    `json:"days"`
	StartTime    *string    `json:"startTime,omitempty"`
	EndTime      *string    `json:"endTime,omitempty"`
	Reason       string     `json:"reason"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

// LeaveType 假期类型
type LeaveType string

const (
	LeaveTypeAnnual       LeaveType = "annual"       // 年假
	LeaveTypeSick         LeaveType = "sick"         // 病假
	LeaveTypePersonal     LeaveType = "personal"     // 事假
	LeaveTypeMaternity    LeaveType = "maternity"    // 产假
	LeaveTypePaternity    LeaveType = "paternity"    // 陪产假
	LeaveTypeMarriage     LeaveType = "marriage"     // 婚假
	LeaveTypeBereavement  LeaveType = "bereavement"  // 丧假
	LeaveTypeCompensatory LeaveType = "compensatory" // 调休
	LeaveTypeOther        LeaveType = "other"        // 其他
)

// IsActive 假期是否有效（当前日期是否在假期范围内）
func (l *LeaveRecord) IsActive() bool {
	now := time.Now()
	return now.After(l.StartDate) && now.Before(l.EndDate.AddDate(0, 0, 1))
}

// CalculateDays 计算请假天数
// 注意：此方法仅用于简单场景。真实业务应使用 ILeaveService.CalculateLeaveDays 方法
// 该方法会考虑工作日、节假日、组织日历等因素
func (l *LeaveRecord) CalculateDays() {
	if l.Days == 0 {
		// 简单计算：结束日期 - 开始日期 + 1
		// 不考虑周末、节假日、小时级请假等复杂场景
		duration := l.EndDate.Sub(l.StartDate)
		l.Days = duration.Hours()/24 + 1
	}
}

// LeaveFilter 假期查询过滤器
type LeaveFilter struct {
	OrgID      string
	EmployeeID *string
	Keyword    string // 员工姓名或工号搜索
	Type       *LeaveType
	StartDate  *time.Time // 查询开始日期之后的假期
	EndDate    *time.Time // 查询结束日期之前的假期
	Page       int
	PageSize   int
}

// LeaveListResult 假期列表结果
type LeaveListResult struct {
	Items    []*LeaveRecord `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// LeaveBalance 假期余额领域模型
type LeaveBalance struct {
	ID         uint64    `json:"id"`
	OrgID      string    `json:"orgId"`
	EmployeeID string    `json:"employeeId"`
	Type       LeaveType `json:"type"`
	Year       int       `json:"year"`
	Total      float64   `json:"total"`
	Used       float64   `json:"used"`
	Remaining  float64   `json:"remaining"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
