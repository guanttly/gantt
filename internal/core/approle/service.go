package approle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEmployeeNotFound         = errors.New("员工不存在")
	ErrGroupNotFound            = errors.New("分组不存在")
	ErrInvalidAppRole           = errors.New("无效的应用角色")
	ErrAppRoleExists            = errors.New("应用角色已存在")
	ErrAppRoleNotFound          = errors.New("应用角色不存在")
	ErrDefaultGroupRoleExists   = errors.New("分组默认应用角色已存在")
	ErrDefaultGroupRoleNotFound = errors.New("分组默认应用角色不存在")
	ErrNodeOutOfScope           = errors.New("目标节点不在当前管理范围内")
	ErrRoleNodeMismatch         = errors.New("应用角色节点必须与员工或分组所属节点一致")
	ErrInvalidGrantedBy         = errors.New("缺少授权人信息")
	ErrEmployeeBindingRequired  = errors.New("当前账号未绑定员工")
)

var appRolePermissions = map[string][]string{
	RoleScheduleAdmin: {
		"schedule:create",
		"schedule:execute",
		"schedule:adjust",
		"schedule:publish",
		"schedule:view:all",
		"leave:approve",
		"leave:view:node",
	},
	RoleScheduler: {
		"schedule:create",
		"schedule:execute",
		"schedule:adjust",
		"schedule:view:node",
	},
	RoleLeaveApprover: {
		"leave:approve",
		"leave:view:node",
	},
	RoleEmployee: {
		"schedule:view:self",
		"leave:create:self",
		"preference:edit:self",
	},
}

type AssignEmployeeRoleInput struct {
	AppRole   string     `json:"app_role"`
	OrgNodeID string     `json:"org_node_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type BatchAssignEmployeeRoleInput struct {
	EmployeeIDs []string   `json:"employee_ids"`
	AppRole     string     `json:"app_role"`
	OrgNodeID   string     `json:"org_node_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type BatchAssignResult struct {
	Created []EmployeeAppRoleResponse `json:"created"`
	Skipped []string                  `json:"skipped_employee_ids"`
}

type AssignGroupDefaultRoleInput struct {
	AppRole   string `json:"app_role"`
	OrgNodeID string `json:"org_node_id"`
}

type Service struct {
	repo     *Repository
	nodeRepo *tenant.Repository
}

func NewService(repo *Repository, nodeRepo *tenant.Repository) *Service {
	return &Service{repo: repo, nodeRepo: nodeRepo}
}

func (s *Service) ListEmployeeRoles(ctx context.Context, employeeID string) ([]EmployeeAppRoleResponse, error) {
	emp, err := s.requireEmployeeInScope(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListEmployeeRoles(ctx, emp.ID)
	if err != nil {
		return nil, fmt.Errorf("查询员工应用角色失败: %w", err)
	}
	return mapEmployeeRoleRows(rows), nil
}

func (s *Service) AssignEmployeeRole(ctx context.Context, employeeID string, input AssignEmployeeRoleInput, grantedBy string) (*EmployeeAppRoleResponse, error) {
	if strings.TrimSpace(grantedBy) == "" {
		return nil, ErrInvalidGrantedBy
	}
	if !validAppRoles[input.AppRole] {
		return nil, ErrInvalidAppRole
	}
	emp, err := s.requireEmployeeInScope(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, input.OrgNodeID); err != nil {
		return nil, err
	}
	if emp.OrgNodeID != input.OrgNodeID {
		return nil, ErrRoleNodeMismatch
	}

	if existing, err := s.repo.FindEmployeeRole(ctx, employeeID, input.OrgNodeID, input.AppRole); err == nil {
		_ = existing
		return nil, ErrAppRoleExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查应用角色唯一性失败: %w", err)
	}

	role := &EmployeeAppRole{
		ID:         uuid.New().String(),
		EmployeeID: employeeID,
		OrgNodeID:  input.OrgNodeID,
		AppRole:    input.AppRole,
		Source:     SourceManual,
		GrantedBy:  grantedBy,
		ExpiresAt:  input.ExpiresAt,
	}
	if err := s.repo.CreateEmployeeRole(ctx, role); err != nil {
		return nil, fmt.Errorf("创建员工应用角色失败: %w", err)
	}

	items, err := s.ListEmployeeRoles(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == role.ID {
			return &item, nil
		}
	}
	return nil, ErrAppRoleNotFound
}

func (s *Service) RemoveEmployeeRole(ctx context.Context, employeeID, roleID string) error {
	if _, err := s.requireEmployeeInScope(ctx, employeeID); err != nil {
		return err
	}
	role, err := s.repo.GetEmployeeRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAppRoleNotFound
		}
		return fmt.Errorf("查询员工应用角色失败: %w", err)
	}
	if role.EmployeeID != employeeID {
		return ErrAppRoleNotFound
	}
	if err := s.repo.DeleteEmployeeRole(ctx, roleID); err != nil {
		return fmt.Errorf("删除员工应用角色失败: %w", err)
	}
	return nil
}

func (s *Service) BatchAssignEmployeeRoles(ctx context.Context, input BatchAssignEmployeeRoleInput, grantedBy string) (*BatchAssignResult, error) {
	result := &BatchAssignResult{}
	for _, employeeID := range input.EmployeeIDs {
		item, err := s.AssignEmployeeRole(ctx, employeeID, AssignEmployeeRoleInput{
			AppRole:   input.AppRole,
			OrgNodeID: input.OrgNodeID,
			ExpiresAt: input.ExpiresAt,
		}, grantedBy)
		if err != nil {
			if errors.Is(err, ErrAppRoleExists) {
				result.Skipped = append(result.Skipped, employeeID)
				continue
			}
			return nil, err
		}
		result.Created = append(result.Created, *item)
	}
	return result, nil
}

func (s *Service) ListGroupDefaultRoles(ctx context.Context, groupID string) ([]GroupDefaultAppRoleResponse, error) {
	if _, err := s.requireGroupInScope(ctx, groupID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListGroupDefaultRoles(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("查询分组默认应用角色失败: %w", err)
	}
	return items, nil
}

func (s *Service) AssignGroupDefaultRole(ctx context.Context, groupID string, input AssignGroupDefaultRoleInput, createdBy string) (*GroupDefaultAppRoleResponse, error) {
	if strings.TrimSpace(createdBy) == "" {
		return nil, ErrInvalidGrantedBy
	}
	if !validAppRoles[input.AppRole] {
		return nil, ErrInvalidAppRole
	}
	grp, err := s.requireGroupInScope(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, input.OrgNodeID); err != nil {
		return nil, err
	}
	if grp.OrgNodeID != input.OrgNodeID {
		return nil, ErrRoleNodeMismatch
	}

	if existing, err := s.repo.FindGroupDefaultRole(ctx, groupID, input.OrgNodeID, input.AppRole); err == nil {
		_ = existing
		return nil, ErrDefaultGroupRoleExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查分组默认角色唯一性失败: %w", err)
	}

	role := &GroupDefaultAppRole{
		ID:        uuid.New().String(),
		GroupID:   groupID,
		OrgNodeID: input.OrgNodeID,
		AppRole:   input.AppRole,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreateGroupDefaultRole(ctx, role); err != nil {
		return nil, fmt.Errorf("创建分组默认应用角色失败: %w", err)
	}

	if err := s.grantDefaultRoleToCurrentMembers(ctx, role, createdBy); err != nil {
		return nil, err
	}

	items, err := s.ListGroupDefaultRoles(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == role.ID {
			return &item, nil
		}
	}
	return nil, ErrDefaultGroupRoleNotFound
}

func (s *Service) RemoveGroupDefaultRole(ctx context.Context, groupID, roleID string) error {
	if _, err := s.requireGroupInScope(ctx, groupID); err != nil {
		return err
	}
	role, err := s.repo.GetGroupDefaultRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDefaultGroupRoleNotFound
		}
		return fmt.Errorf("查询分组默认应用角色失败: %w", err)
	}
	if role.GroupID != groupID {
		return ErrDefaultGroupRoleNotFound
	}
	if err := s.repo.DeleteGroupDefaultRole(ctx, roleID); err != nil {
		return fmt.Errorf("删除分组默认应用角色失败: %w", err)
	}
	if err := s.repo.DeleteEmployeeRolesByGroup(ctx, groupID); err != nil {
		return fmt.Errorf("回收分组授予的应用角色失败: %w", err)
	}
	return nil
}

func (s *Service) Summary(ctx context.Context) ([]AppRoleSummaryItem, error) {
	rows, err := s.repo.Summaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询应用角色汇总失败: %w", err)
	}
	items := make([]AppRoleSummaryItem, 0, len(rows))
	for _, row := range rows {
		if !s.isNodeInScope(ctx, row.OrgNodeID) {
			continue
		}
		items = append(items, AppRoleSummaryItem(row))
	}
	return items, nil
}

func (s *Service) Expiring(ctx context.Context, withinDays int) ([]ExpiringRoleItem, error) {
	if withinDays <= 0 {
		withinDays = 30
	}
	rows, err := s.repo.Expiring(ctx, time.Now().AddDate(0, 0, withinDays))
	if err != nil {
		return nil, fmt.Errorf("查询即将过期的应用角色失败: %w", err)
	}
	items := make([]ExpiringRoleItem, 0, len(rows))
	for _, row := range rows {
		if !s.isNodeInScope(ctx, row.OrgNodeID) {
			continue
		}
		items = append(items, ExpiringRoleItem{
			EmployeeAppRoleResponse: mapEmployeeRoleRow(row),
			EmployeeName:            row.EmployeeName,
		})
	}
	return items, nil
}

func (s *Service) SyncRolesForGroupMember(ctx context.Context, groupID, employeeID, grantedBy string) error {
	if strings.TrimSpace(grantedBy) == "" {
		return ErrInvalidGrantedBy
	}
	grp, err := s.requireGroupInScope(ctx, groupID)
	if err != nil {
		return err
	}
	emp, err := s.requireEmployeeInScope(ctx, employeeID)
	if err != nil {
		return err
	}
	if emp.OrgNodeID != grp.OrgNodeID {
		return ErrRoleNodeMismatch
	}
	defaultRoles, err := s.repo.ListGroupDefaultRoleModels(ctx, groupID)
	if err != nil {
		return fmt.Errorf("查询分组默认应用角色失败: %w", err)
	}
	for _, role := range defaultRoles {
		if existing, err := s.repo.FindEmployeeRole(ctx, employeeID, role.OrgNodeID, role.AppRole); err == nil {
			_ = existing
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查员工应用角色失败: %w", err)
		}
		sourceGroupID := groupID
		if err := s.repo.CreateEmployeeRole(ctx, &EmployeeAppRole{
			ID:            uuid.New().String(),
			EmployeeID:    employeeID,
			OrgNodeID:     role.OrgNodeID,
			AppRole:       role.AppRole,
			Source:        SourceGroup,
			SourceGroupID: &sourceGroupID,
			GrantedBy:     grantedBy,
		}); err != nil {
			return fmt.Errorf("同步分组默认应用角色失败: %w", err)
		}
	}
	return nil
}

func (s *Service) RevokeRolesForGroupMember(ctx context.Context, groupID, employeeID string) error {
	if err := s.repo.DeleteEmployeeRoleByGroupMember(ctx, groupID, employeeID); err != nil {
		return fmt.Errorf("回收分组成员应用角色失败: %w", err)
	}
	return nil
}

func (s *Service) CleanupEmployeeRoles(ctx context.Context, employeeID string) error {
	if _, err := s.requireEmployeeInScope(ctx, employeeID); err != nil {
		return err
	}
	if err := s.repo.DeleteAllEmployeeRoles(ctx, employeeID); err != nil {
		return fmt.Errorf("清理员工应用角色失败: %w", err)
	}
	return nil
}

func (s *Service) CleanExpiredRoles(ctx context.Context) (int64, error) {
	rowsAffected, err := s.repo.DeleteExpiredRoles(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("清理过期应用角色失败: %w", err)
	}
	return rowsAffected, nil
}

func (s *Service) CleanupGroup(ctx context.Context, groupID string) error {
	if err := s.repo.DeleteEmployeeRolesByGroup(ctx, groupID); err != nil {
		return fmt.Errorf("清理分组来源应用角色失败: %w", err)
	}
	if err := s.repo.DeleteGroupDefaultRolesByGroup(ctx, groupID); err != nil {
		return fmt.Errorf("清理分组默认应用角色失败: %w", err)
	}
	return nil
}

func (s *Service) MyRoles(ctx context.Context, actorUserID string) (*MyRolesResponse, error) {
	emp, err := s.resolveActorEmployee(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	currentNodeID := tenant.GetOrgNodeID(ctx)
	if currentNodeID == "" {
		currentNodeID = emp.OrgNodeID
	}
	node, err := s.nodeRepo.GetByID(ctx, currentNodeID)
	if err != nil {
		return nil, fmt.Errorf("查询当前节点失败: %w", err)
	}

	rows, err := s.repo.ListEmployeeRoles(ctx, emp.ID)
	if err != nil {
		return nil, fmt.Errorf("查询当前员工应用角色失败: %w", err)
	}
	items := make([]EmployeeAppRoleResponse, 0, len(rows)+1)
	for _, row := range rows {
		if row.OrgNodeID != currentNodeID {
			continue
		}
		items = append(items, mapEmployeeRoleRow(row))
	}
	items = append(items, EmployeeAppRoleResponse{
		EmployeeID:  emp.ID,
		OrgNodeID:   currentNodeID,
		OrgNodeName: node.Name,
		AppRole:     RoleEmployee,
		Source:      SourceSystem,
	})

	return &MyRolesResponse{
		EmployeeID:  emp.ID,
		OrgNodeID:   currentNodeID,
		OrgNodeName: node.Name,
		AppRoles:    dedupeRoles(items),
	}, nil
}

func (s *Service) MyPermissions(ctx context.Context, actorUserID string) (*MyPermissionsResponse, error) {
	roles, err := s.MyRoles(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	permissionSet := make(map[string]struct{})
	for _, role := range roles.AppRoles {
		for _, permission := range appRolePermissions[role.AppRole] {
			permissionSet[permission] = struct{}{}
		}
	}
	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return &MyPermissionsResponse{
		EmployeeID:  roles.EmployeeID,
		OrgNodeID:   roles.OrgNodeID,
		OrgNodeName: roles.OrgNodeName,
		Permissions: permissions,
	}, nil
}

func (s *Service) requireEmployeeInScope(ctx context.Context, employeeID string) (*employeeRecord, error) {
	emp, err := s.repo.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("查询员工失败: %w", err)
	}
	if !s.isNodePathInScope(ctx, emp.Path) {
		return nil, ErrNodeOutOfScope
	}
	return emp, nil
}

func (s *Service) requireGroupInScope(ctx context.Context, groupID string) (*groupRecord, error) {
	grp, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("查询分组失败: %w", err)
	}
	if !s.isNodePathInScope(ctx, grp.Path) {
		return nil, ErrNodeOutOfScope
	}
	return grp, nil
}

func (s *Service) ensureNodeInScope(ctx context.Context, nodeID string) error {
	if !s.isNodeInScope(ctx, nodeID) {
		return ErrNodeOutOfScope
	}
	return nil
}

func (s *Service) isNodeInScope(ctx context.Context, nodeID string) bool {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return false
	}
	return s.isNodePathInScope(ctx, node.Path)
}

func (s *Service) isNodePathInScope(ctx context.Context, targetPath string) bool {
	currentPath := strings.TrimRight(tenant.GetOrgNodePath(ctx), "/")
	if currentPath == "" {
		return false
	}
	targetPath = strings.TrimRight(targetPath, "/")
	return targetPath == currentPath || strings.HasPrefix(targetPath, currentPath+"/")
}

func (s *Service) grantDefaultRoleToCurrentMembers(ctx context.Context, role *GroupDefaultAppRole, grantedBy string) error {
	members, err := s.repo.ListGroupMembers(ctx, role.GroupID)
	if err != nil {
		return fmt.Errorf("查询分组成员失败: %w", err)
	}
	for _, member := range members {
		if member.Status != "active" {
			continue
		}
		if member.OrgNodeID != role.OrgNodeID {
			continue
		}
		if existing, err := s.repo.FindEmployeeRole(ctx, member.EmployeeID, role.OrgNodeID, role.AppRole); err == nil {
			_ = existing
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查分组成员应用角色失败: %w", err)
		}

		sourceGroupID := role.GroupID
		if err := s.repo.CreateEmployeeRole(ctx, &EmployeeAppRole{
			ID:            uuid.New().String(),
			EmployeeID:    member.EmployeeID,
			OrgNodeID:     role.OrgNodeID,
			AppRole:       role.AppRole,
			Source:        SourceGroup,
			SourceGroupID: &sourceGroupID,
			GrantedBy:     grantedBy,
		}); err != nil {
			return fmt.Errorf("为分组成员授予默认应用角色失败: %w", err)
		}
	}
	return nil
}

func mapEmployeeRoleRows(rows []employeeRoleRow) []EmployeeAppRoleResponse {
	items := make([]EmployeeAppRoleResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapEmployeeRoleRow(row))
	}
	return items
}

func mapEmployeeRoleRow(row employeeRoleRow) EmployeeAppRoleResponse {
	return EmployeeAppRoleResponse{
		ID:              row.ID,
		EmployeeID:      row.EmployeeID,
		OrgNodeID:       row.OrgNodeID,
		OrgNodeName:     row.OrgNodeName,
		AppRole:         row.AppRole,
		Source:          row.Source,
		SourceGroupID:   row.SourceGroupID,
		SourceGroupName: row.SourceGroupName,
		GrantedBy:       row.GrantedBy,
		GrantedAt:       row.GrantedAt,
		ExpiresAt:       row.ExpiresAt,
	}
}

func (s *Service) resolveActorEmployee(ctx context.Context, actorUserID string) (*employeeRecord, error) {
	if strings.TrimSpace(actorUserID) == "" {
		return nil, ErrEmployeeBindingRequired
	}
	emp, err := s.repo.GetEmployeeByID(ctx, actorUserID)
	if err == nil {
		if emp.Status != "active" {
			return nil, ErrEmployeeNotFound
		}
		if !s.isNodePathInScope(ctx, emp.Path) {
			return nil, ErrNodeOutOfScope
		}
		return emp, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询当前员工失败: %w", err)
	}
	emp, err = s.repo.GetBoundEmployeeByUserID(ctx, actorUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeBindingRequired
		}
		return nil, fmt.Errorf("查询绑定员工失败: %w", err)
	}
	if !s.isNodePathInScope(ctx, emp.Path) {
		return nil, ErrNodeOutOfScope
	}
	return emp, nil
}

func dedupeRoles(items []EmployeeAppRoleResponse) []EmployeeAppRoleResponse {
	seen := make(map[string]struct{}, len(items))
	result := make([]EmployeeAppRoleResponse, 0, len(items))
	for _, item := range items {
		key := item.OrgNodeID + "|" + item.AppRole
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}
