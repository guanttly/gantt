package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
	"time"
)

// ISchedulingRepository 排班仓储接口
type ISchedulingRepository interface {
	// CreateAssignment 创建排班分配
	CreateAssignment(ctx context.Context, assignment *model.ShiftAssignment) error

	// BatchCreateAssignments 批量创建排班分配
	BatchCreateAssignments(ctx context.Context, assignments []*model.ShiftAssignment) error

	// UpdateAssignment 更新排班分配
	UpdateAssignment(ctx context.Context, assignment *model.ShiftAssignment) error

	// DeleteAssignment 删除排班分配（通过ID）
	DeleteAssignment(ctx context.Context, orgID, assignmentID string) error

	// DeleteByEmployeeAndDate 删除员工在指定日期的排班
	DeleteByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) error

	// GetAssignmentByID 根据ID获取排班分配
	GetAssignmentByID(ctx context.Context, orgID, assignmentID string) (*model.ShiftAssignment, error)

	// GetAssignmentByEmployeeAndDate 获取员工在指定日期的排班
	GetAssignmentByEmployeeAndDate(ctx context.Context, orgID, employeeID string, date time.Time) (*model.ShiftAssignment, error)

	// ListAssignmentsByEmployee 查询员工的排班列表
	ListAssignmentsByEmployee(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// ListAssignmentsByDateRange 查询日期范围内的所有排班
	ListAssignmentsByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// ListAssignmentsByShift 查询特定班次的排班列表
	ListAssignmentsByShift(ctx context.Context, orgID, shiftID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error)

	// CountAssignmentsByDate 统计指定日期的排班数量
	CountAssignmentsByDate(ctx context.Context, orgID string, date time.Time) (int64, error)

	// CountAssignmentsByShift 统计指定班次在日期范围内的排班数量
	CountAssignmentsByShift(ctx context.Context, orgID, shiftID string, startDate, endDate time.Time) (int64, error)
}
