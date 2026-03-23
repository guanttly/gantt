package service

import (
	"context"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// ISchedulingService 排班服务接口
type ISchedulingService interface {
	// BatchAssignShifts 批量分配班次
	// 用于手动排班，一次性为多个员工分配多个日期的班次
	BatchAssignShifts(ctx context.Context, assignments []*model.ShiftAssignment) error

	// GetScheduleByDateRange 获取日期范围内的排班数据
	// 返回指定日期范围内所有员工的排班信息
	GetScheduleByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// GetEmployeeSchedule 获取员工的排班数据
	// 返回指定员工在日期范围内的排班信息
	GetEmployeeSchedule(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// UpdateScheduleAssignment 更新排班分配
	// 修改已有的排班分配（例如换班）
	UpdateScheduleAssignment(ctx context.Context, assignment *model.ShiftAssignment) error

	// DeleteScheduleAssignmentByID 通过ID删除排班分配
	// 删除指定ID的排班记录
	DeleteScheduleAssignmentByID(ctx context.Context, orgID, assignmentID string) error

	// DeleteScheduleAssignment 删除排班分配（废弃，保留向后兼容）
	// 删除指定员工在指定日期的所有排班
	DeleteScheduleAssignment(ctx context.Context, orgID, employeeID string, date time.Time) error

	// BatchDeleteScheduleAssignments 批量删除排班分配
	// 批量删除多个排班记录
	BatchDeleteScheduleAssignments(ctx context.Context, orgID string, employeeIDs []string, dates []time.Time) error

	// CheckScheduleConflict 检查排班冲突
	// 检查员工在指定日期是否已有排班
	CheckScheduleConflict(ctx context.Context, orgID, employeeID string, date time.Time) (bool, error)

	// GetScheduleSummary 获取排班汇总
	// 统计指定日期范围内的排班情况（每个班次的人数等）
	GetScheduleSummary(ctx context.Context, orgID string, startDate, endDate time.Time) (map[string]interface{}, error)
}
