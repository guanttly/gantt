package leave

import (
"context"
"errors"
"fmt"

"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

var (
ErrLeaveNotFound   = errors.New("请假记录不存在")
ErrLeaveNotPending = errors.New("请假记录不在待审批状态")
)

// CreateInput 创建请假的输入参数。
type CreateInput struct {
EmployeeID string  `json:"employee_id"`
LeaveType  string  `json:"leave_type"`
StartDate  string  `json:"start_date"`
EndDate    string  `json:"end_date"`
Reason     *string `json:"reason"`
}

// UpdateInput 更新请假的输入参数。
type UpdateInput struct {
LeaveType *string `json:"leave_type,omitempty"`
StartDate *string `json:"start_date,omitempty"`
EndDate   *string `json:"end_date,omitempty"`
Reason    *string `json:"reason,omitempty"`
}

// Service 请假业务逻辑层。
type Service struct {
repo *Repository
}

// NewService 创建请假服务。
func NewService(repo *Repository) *Service {
return &Service{repo: repo}
}

// Create 创建请假记录。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Leave, error) {
orgNodeID := tenant.GetOrgNodeID(ctx)
if orgNodeID == "" {
return nil, fmt.Errorf("缺少组织节点信息")
}

l := &Leave{
ID:         uuid.New().String(),
EmployeeID: input.EmployeeID,
LeaveType:  input.LeaveType,
StartDate:  input.StartDate,
EndDate:    input.EndDate,
Reason:     input.Reason,
Status:     StatusPending,
TenantModel: tenant.TenantModel{
OrgNodeID: orgNodeID,
},
}

if err := s.repo.Create(ctx, l); err != nil {
return nil, fmt.Errorf("创建请假记录失败: %w", err)
}

return l, nil
}

// GetByID 获取请假详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Leave, error) {
l, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrLeaveNotFound
}
return nil, err
}
return l, nil
}

// Update 更新请假记录（仅 pending 状态可更新）。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Leave, error) {
l, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrLeaveNotFound
}
return nil, err
}

if !l.IsPending() {
return nil, ErrLeaveNotPending
}

if input.LeaveType != nil {
l.LeaveType = *input.LeaveType
}
if input.StartDate != nil {
l.StartDate = *input.StartDate
}
if input.EndDate != nil {
l.EndDate = *input.EndDate
}
if input.Reason != nil {
l.Reason = input.Reason
}

if err := s.repo.Update(ctx, l); err != nil {
return nil, fmt.Errorf("更新请假记录失败: %w", err)
}

return l, nil
}

// Delete 删除请假记录（仅 pending 状态可删除）。
func (s *Service) Delete(ctx context.Context, id string) error {
l, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ErrLeaveNotFound
}
return err
}

if !l.IsPending() {
return ErrLeaveNotPending
}

return s.repo.Delete(ctx, id)
}

// Approve 审批请假（通过/拒绝）。
func (s *Service) Approve(ctx context.Context, id string, approved bool) (*Leave, error) {
l, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrLeaveNotFound
}
return nil, err
}

if !l.IsPending() {
return nil, ErrLeaveNotPending
}

if approved {
l.Status = StatusApproved
} else {
l.Status = StatusRejected
}

if err := s.repo.Update(ctx, l); err != nil {
return nil, fmt.Errorf("审批请假失败: %w", err)
}

return l, nil
}

// List 分页查询请假列表。
func (s *Service) List(ctx context.Context, opts ListOptions) ([]Leave, int64, error) {
if opts.Page <= 0 {
opts.Page = 1
}
if opts.Size <= 0 {
opts.Size = 20
}
if opts.Size > 100 {
opts.Size = 100
}
return s.repo.List(ctx, opts)
}

// IsEmployeeOnLeave 检查员工在指定日期是否请假（供排班引擎调用）。
func (s *Service) IsEmployeeOnLeave(ctx context.Context, employeeID, date string) (bool, error) {
leaves, err := s.repo.GetByEmployeeAndDateRange(ctx, employeeID, date, date)
if err != nil {
return false, err
}
return len(leaves) > 0, nil
}
