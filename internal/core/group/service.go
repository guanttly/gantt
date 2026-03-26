package group

import (
"context"
"errors"
"fmt"

"gantt-saas/internal/tenant"

"github.com/google/uuid"
"gorm.io/gorm"
)

var (
ErrGroupNotFound   = errors.New("分组不存在")
ErrMemberExists    = errors.New("成员已在分组中")
ErrMemberNotFound  = errors.New("成员不在分组中")
)

// CreateInput 创建分组的输入参数。
type CreateInput struct {
Name        string  `json:"name"`
Description *string `json:"description"`
}

// UpdateInput 更新分组的输入参数。
type UpdateInput struct {
Name        *string `json:"name,omitempty"`
Description *string `json:"description,omitempty"`
}

// Service 分组业务逻辑层。
type Service struct {
repo *Repository
}

// NewService 创建分组服务。
func NewService(repo *Repository) *Service {
return &Service{repo: repo}
}

// Create 创建分组。
func (s *Service) Create(ctx context.Context, input CreateInput) (*EmployeeGroup, error) {
orgNodeID := tenant.GetOrgNodeID(ctx)
if orgNodeID == "" {
return nil, fmt.Errorf("缺少组织节点信息")
}

g := &EmployeeGroup{
ID:          uuid.New().String(),
Name:        input.Name,
Description: input.Description,
TenantModel: tenant.TenantModel{
OrgNodeID: orgNodeID,
},
}

if err := s.repo.Create(ctx, g); err != nil {
return nil, fmt.Errorf("创建分组失败: %w", err)
}

return g, nil
}

// GetByID 获取分组详情。
func (s *Service) GetByID(ctx context.Context, id string) (*EmployeeGroup, error) {
g, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrGroupNotFound
}
return nil, err
}
return g, nil
}

// Update 更新分组信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*EmployeeGroup, error) {
g, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrGroupNotFound
}
return nil, err
}

if input.Name != nil {
g.Name = *input.Name
}
if input.Description != nil {
g.Description = input.Description
}

if err := s.repo.Update(ctx, g); err != nil {
return nil, fmt.Errorf("更新分组失败: %w", err)
}

return g, nil
}

// Delete 删除分组（同时删除成员关联）。
func (s *Service) Delete(ctx context.Context, id string) error {
_, err := s.repo.GetByID(ctx, id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ErrGroupNotFound
}
return err
}

// 先删除成员关联
if err := s.repo.DeleteMembersByGroup(ctx, id); err != nil {
return fmt.Errorf("删除分组成员失败: %w", err)
}

return s.repo.Delete(ctx, id)
}

// List 查询分组列表。
func (s *Service) List(ctx context.Context) ([]EmployeeGroup, error) {
return s.repo.List(ctx)
}

// GetMembers 获取分组成员列表。
func (s *Service) GetMembers(ctx context.Context, groupID string) ([]GroupMember, error) {
// 验证分组存在
_, err := s.repo.GetByID(ctx, groupID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrGroupNotFound
}
return nil, err
}

return s.repo.GetMembers(ctx, groupID)
}

// AddMember 添加成员到分组。
func (s *Service) AddMember(ctx context.Context, groupID, employeeID string) (*GroupMember, error) {
orgNodeID := tenant.GetOrgNodeID(ctx)
if orgNodeID == "" {
return nil, fmt.Errorf("缺少组织节点信息")
}

// 验证分组存在
_, err := s.repo.GetByID(ctx, groupID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return nil, ErrGroupNotFound
}
return nil, err
}

// 检查是否已存在
existing, err := s.repo.GetMember(ctx, groupID, employeeID)
if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
return nil, err
}
if existing != nil {
return nil, ErrMemberExists
}

m := &GroupMember{
ID:         uuid.New().String(),
GroupID:    groupID,
EmployeeID: employeeID,
TenantModel: tenant.TenantModel{
OrgNodeID: orgNodeID,
},
}

if err := s.repo.AddMember(ctx, m); err != nil {
return nil, fmt.Errorf("添加成员失败: %w", err)
}

return m, nil
}

// RemoveMember 从分组中移除成员。
func (s *Service) RemoveMember(ctx context.Context, groupID, employeeID string) error {
// 验证成员存在
_, err := s.repo.GetMember(ctx, groupID, employeeID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return ErrMemberNotFound
}
return err
}

return s.repo.RemoveMember(ctx, groupID, employeeID)
}
