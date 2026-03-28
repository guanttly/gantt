package employee

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmployeeNotFound             = errors.New("员工不存在")
	ErrEmployeeNoDup                = errors.New("同节点下工号已存在")
	ErrEmployeeNodeOutOfScope       = errors.New("目标组织节点不在当前管理范围内")
	ErrEmployeeNodeMustBeDepartment = errors.New("员工只能绑定到科室节点")
	ErrEmployeeSameDepartment       = errors.New("目标科室与当前科室相同，无需调动")
)

// CreateInput 创建员工的输入参数。
type CreateInput struct {
	OrgNodeID  *string `json:"org_node_id,omitempty"`
	Name       string  `json:"name"`
	EmployeeNo *string `json:"employee_no"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Position   *string `json:"position"`
	Category   *string `json:"category"`
	HireDate   *string `json:"hire_date"`
}

// UpdateInput 更新员工的输入参数。
type UpdateInput struct {
	OrgNodeID  *string `json:"org_node_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	EmployeeNo *string `json:"employee_no,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	Position   *string `json:"position,omitempty"`
	Category   *string `json:"category,omitempty"`
	Status     *string `json:"status,omitempty"`
	HireDate   *string `json:"hire_date,omitempty"`
}

type ResetPasswordResult struct {
	DefaultPassword string `json:"default_password"`
	MustResetPwd    bool   `json:"must_reset_pwd"`
}

// Service 员工业务逻辑层。
type Service struct {
	repo            *Repository
	appRoleCleaner  AppRoleCleaner
	appRoleReader   AppRoleReader
	orgNodeResolver OrgNodeResolver
	groupCleaner    GroupCleaner
}

type AppRoleCleaner interface {
	CleanupEmployeeRoles(ctx context.Context, employeeID string) error
}

type AppRoleReader interface {
	ListEmployeeRolesBatch(ctx context.Context, employeeIDs []string) (map[string][]EmployeeAppRoleInfo, error)
}

// OrgNodeResolver 查询组织节点信息的接口。
type OrgNodeResolver interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
	GetByIDs(ctx context.Context, ids []string) ([]tenant.OrgNode, error)
	GetAncestorNames(ctx context.Context, nodePath string) ([]string, error)
}

// GroupCleaner 清理员工分组关系的接口。
type GroupCleaner interface {
	RemoveEmployeeFromAllGroups(ctx context.Context, employeeID string) (int64, error)
}

// NewService 创建员工服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetAppRoleCleaner(cleaner AppRoleCleaner) {
	s.appRoleCleaner = cleaner
}

func (s *Service) SetAppRoleReader(reader AppRoleReader) {
	s.appRoleReader = reader
}

func (s *Service) SetOrgNodeResolver(resolver OrgNodeResolver) {
	s.orgNodeResolver = resolver
}

func (s *Service) SetGroupCleaner(cleaner GroupCleaner) {
	s.groupCleaner = cleaner
}

// Create 创建员工。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Employee, error) {
	orgNodeID, err := s.resolveTargetOrgNodeID(ctx, input.OrgNodeID)
	if err != nil {
		return nil, err
	}
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 检查工号唯一性
	if input.EmployeeNo != nil && *input.EmployeeNo != "" {
		existing, err := s.repo.GetByOrgNodeAndNo(ctx, orgNodeID, *input.EmployeeNo)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查工号唯一性失败: %w", err)
		}
		if existing != nil {
			return nil, ErrEmployeeNoDup
		}
	}

	emp := &Employee{
		ID:              uuid.New().String(),
		Name:            input.Name,
		EmployeeNo:      input.EmployeeNo,
		Phone:           input.Phone,
		Email:           input.Email,
		Position:        input.Position,
		Category:        input.Category,
		SchedulingRole:  SchedulingRoleEmployee,
		AppMustResetPwd: true,
		HireDate:        input.HireDate,
		Status:          StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	defaultPassword := buildDefaultAppPassword(emp)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成员工应用密码失败: %w", err)
	}
	emp.AppPasswordHash = &[]string{string(hashedPassword)}[0]
	emp.AppDefaultPassword = &defaultPassword

	if err := s.repo.Create(ctx, emp); err != nil {
		return nil, fmt.Errorf("创建员工失败: %w", err)
	}

	return emp, nil
}

// GetByID 获取员工详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Employee, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}
	return emp, nil
}

func buildDefaultAppPassword(emp *Employee) string {
	if emp.EmployeeNo != nil && strings.TrimSpace(*emp.EmployeeNo) != "" {
		return strings.TrimSpace(*emp.EmployeeNo) + "@App1"
	}
	if emp.Phone != nil && strings.TrimSpace(*emp.Phone) != "" {
		phone := strings.TrimSpace(*emp.Phone)
		if len(phone) > 4 {
			phone = phone[len(phone)-4:]
		}
		return "Emp" + phone + "@App1"
	}
	identifier := emp.ID
	if len(identifier) > 6 {
		identifier = identifier[len(identifier)-6:]
	}
	return "Emp" + identifier + "@App1"
}

// Update 更新员工信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Employee, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}

	originalOrgNodeID := emp.OrgNodeID
	if input.OrgNodeID != nil {
		targetOrgNodeID, err := s.resolveTargetOrgNodeID(ctx, input.OrgNodeID)
		if err != nil {
			return nil, err
		}
		emp.OrgNodeID = targetOrgNodeID
	}

	if input.Name != nil {
		emp.Name = *input.Name
	}
	if input.EmployeeNo != nil {
		// 检查新工号唯一性
		if *input.EmployeeNo != "" {
			existing, err := s.repo.GetByOrgNodeAndNo(ctx, emp.OrgNodeID, *input.EmployeeNo)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("检查工号唯一性失败: %w", err)
			}
			if existing != nil && existing.ID != emp.ID {
				return nil, ErrEmployeeNoDup
			}
		}
		emp.EmployeeNo = input.EmployeeNo
	}
	if input.Phone != nil {
		emp.Phone = input.Phone
	}
	if input.Email != nil {
		emp.Email = input.Email
	}
	if input.Position != nil {
		emp.Position = input.Position
	}
	if input.Category != nil {
		emp.Category = input.Category
	}
	if input.Status != nil {
		emp.Status = *input.Status
	}
	if input.HireDate != nil {
		emp.HireDate = input.HireDate
	}

	if err := s.repo.Update(ctx, emp); err != nil {
		return nil, fmt.Errorf("更新员工失败: %w", err)
	}

	if originalOrgNodeID != emp.OrgNodeID && s.appRoleCleaner != nil {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, emp.ID); err != nil {
			return nil, err
		}
	}

	if input.Status != nil && *input.Status == StatusInactive && s.appRoleCleaner != nil {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, emp.ID); err != nil {
			return nil, err
		}
	}

	return emp, nil
}

// Delete 删除员工。
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		return err
	}
	if s.appRoleCleaner != nil {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) ResetPassword(ctx context.Context, id string) (*ResetPasswordResult, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}

	defaultPassword := buildDefaultAppPassword(emp)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成员工应用密码失败: %w", err)
	}
	hash := string(hashedPassword)
	emp.AppPasswordHash = &hash
	emp.AppMustResetPwd = true
	emp.AppDefaultPassword = &defaultPassword

	if err := s.repo.Update(ctx, emp); err != nil {
		return nil, fmt.Errorf("重置员工应用密码失败: %w", err)
	}

	return &ResetPasswordResult{
		DefaultPassword: defaultPassword,
		MustResetPwd:    true,
	}, nil
}

// List 分页查询员工列表。
func (s *Service) List(ctx context.Context, opts ListOptions) ([]Employee, int64, error) {
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

func (s *Service) resolveTargetOrgNodeID(ctx context.Context, target *string) (string, error) {
	currentOrgNodeID := tenant.GetOrgNodeID(ctx)
	if currentOrgNodeID == "" {
		return "", fmt.Errorf("缺少组织节点信息")
	}
	targetOrgNodeID := currentOrgNodeID
	if target == nil || strings.TrimSpace(*target) == "" {
		targetOrgNodeID = currentOrgNodeID
	} else {
		targetOrgNodeID = strings.TrimSpace(*target)
	}

	var node tenant.OrgNode
	if err := s.repo.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", targetOrgNodeID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", tenant.ErrNodeNotFound
		}
		return "", fmt.Errorf("查询组织节点失败: %w", err)
	}

	currentPath := strings.TrimRight(tenant.GetOrgNodePath(ctx), "/")
	targetPath := strings.TrimRight(node.Path, "/")
	if currentPath != "" && targetPath != currentPath && !strings.HasPrefix(targetPath, currentPath+"/") {
		return "", ErrEmployeeNodeOutOfScope
	}
	if !node.IsActive() {
		return "", tenant.ErrNodeSuspended
	}
	if node.NodeType != tenant.NodeTypeDepartment {
		return "", ErrEmployeeNodeMustBeDepartment
	}

	return node.ID, nil
}

// EnrichResponse 将 Employee 转换为带组织路径信息的 EmployeeResponse。
func (s *Service) EnrichResponse(ctx context.Context, emp *Employee) *EmployeeResponse {
	resp := &EmployeeResponse{Employee: *emp}
	if s.orgNodeResolver == nil {
		return resp
	}

	node, err := s.orgNodeResolver.GetByID(ctx, emp.OrgNodeID)
	if err != nil {
		return resp
	}

	resp.OrgNodeName = node.Name
	resp.OrgNodeType = node.NodeType

	names, err := s.orgNodeResolver.GetAncestorNames(ctx, node.Path)
	if err == nil && len(names) > 0 {
		resp.OrgNodePathDisplay = strings.Join(names, " / ")
	} else {
		resp.OrgNodePathDisplay = node.Name
	}

	if s.appRoleReader != nil {
		if roleMap, err := s.appRoleReader.ListEmployeeRolesBatch(ctx, []string{emp.ID}); err == nil {
			resp.AppRoles = roleMap[emp.ID]
		}
	}

	return resp
}

// EnrichResponseList 批量转换员工列表为带组织路径信息的响应。
func (s *Service) EnrichResponseList(ctx context.Context, employees []Employee) []EmployeeResponse {
	return s.enrichResponseList(ctx, employees, false)
}

func (s *Service) EnrichResponseListWithRoles(ctx context.Context, employees []Employee) []EmployeeResponse {
	return s.enrichResponseList(ctx, employees, true)
}

func (s *Service) enrichResponseList(ctx context.Context, employees []Employee, includeRoles bool) []EmployeeResponse {
	results := make([]EmployeeResponse, len(employees))
	for i := range employees {
		results[i] = EmployeeResponse{Employee: employees[i]}
	}
	if s.orgNodeResolver == nil || len(employees) == 0 {
		return results
	}

	nodeIDSet := make(map[string]struct{}, len(employees))
	for _, emp := range employees {
		if emp.OrgNodeID != "" {
			nodeIDSet[emp.OrgNodeID] = struct{}{}
		}
	}
	nodeIDs := make([]string, 0, len(nodeIDSet))
	for id := range nodeIDSet {
		nodeIDs = append(nodeIDs, id)
	}
	nodes, err := s.orgNodeResolver.GetByIDs(ctx, nodeIDs)
	if err != nil {
		return results
	}
	nodeMap := make(map[string]tenant.OrgNode, len(nodes))
	pathIDSet := make(map[string]struct{}, len(nodes)*2)
	for _, node := range nodes {
		nodeMap[node.ID] = node
		for _, part := range strings.Split(strings.Trim(node.Path, "/"), "/") {
			if part != "" {
				pathIDSet[part] = struct{}{}
			}
		}
	}
	pathIDs := make([]string, 0, len(pathIDSet))
	for id := range pathIDSet {
		pathIDs = append(pathIDs, id)
	}
	pathNodes, err := s.orgNodeResolver.GetByIDs(ctx, pathIDs)
	if err != nil {
		return results
	}
	nameMap := make(map[string]string, len(pathNodes))
	for _, node := range pathNodes {
		nameMap[node.ID] = node.Name
	}

	for i := range results {
		node, ok := nodeMap[results[i].OrgNodeID]
		if !ok {
			continue
		}
		results[i].OrgNodeName = node.Name
		results[i].OrgNodeType = node.NodeType
		results[i].OrgNodePathDisplay = buildPathDisplay(node.Path, nameMap, node.Name)
	}

	if includeRoles && s.appRoleReader != nil {
		employeeIDs := make([]string, 0, len(results))
		for _, item := range results {
			employeeIDs = append(employeeIDs, item.ID)
		}
		if roleMap, err := s.appRoleReader.ListEmployeeRolesBatch(ctx, employeeIDs); err == nil {
			for i := range results {
				results[i].AppRoles = roleMap[results[i].ID]
			}
		}
	}

	return results
}

func buildPathDisplay(nodePath string, nameMap map[string]string, fallback string) string {
	parts := strings.Split(strings.Trim(nodePath, "/"), "/")
	names := make([]string, 0, len(parts))
	for _, id := range parts {
		if name, ok := nameMap[id]; ok {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return fallback
	}
	return strings.Join(names, " / ")
}

// TransferInput 员工调动的输入参数。
type TransferInput struct {
	TargetOrgNodeID string `json:"target_org_node_id"`
	Reason          string `json:"reason"`
}

// TransferResult 员工调动的结果。
type TransferResult struct {
	EmployeeID    string      `json:"employee_id"`
	FromOrgNode   OrgNodeInfo `json:"from_org_node"`
	ToOrgNode     OrgNodeInfo `json:"to_org_node"`
	RolesCleaned  int64       `json:"roles_cleaned"`
	GroupsRemoved int64       `json:"groups_removed"`
}

// OrgNodeInfo 组织节点概要信息。
type OrgNodeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PathDisplay string `json:"path_display"`
}

// Transfer 调动员工到指定科室。
func (s *Service) Transfer(ctx context.Context, id string, input TransferInput) (*TransferResult, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}

	if input.TargetOrgNodeID == "" {
		return nil, fmt.Errorf("target_org_node_id 为必填项")
	}
	if emp.OrgNodeID == input.TargetOrgNodeID {
		return nil, ErrEmployeeSameDepartment
	}

	// 验证目标节点存在且在管理范围内
	targetOrgNodeID, err := s.resolveTargetOrgNodeID(ctx, &input.TargetOrgNodeID)
	if err != nil {
		return nil, err
	}

	// 构建响应信息
	result := &TransferResult{EmployeeID: emp.ID}

	// 原科室信息
	result.FromOrgNode = s.buildOrgNodeInfo(ctx, emp.OrgNodeID)

	// 更新 org_node_id
	originalOrgNodeID := emp.OrgNodeID
	emp.OrgNodeID = targetOrgNodeID
	if err := s.repo.Update(ctx, emp); err != nil {
		return nil, fmt.Errorf("更新员工组织节点失败: %w", err)
	}

	// 清除原科室的应用角色
	if s.appRoleCleaner != nil && originalOrgNodeID != emp.OrgNodeID {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, emp.ID); err != nil {
			return nil, err
		}
	}

	// 清除原科室的分组成员关系
	if s.groupCleaner != nil {
		removed, err := s.groupCleaner.RemoveEmployeeFromAllGroups(ctx, emp.ID)
		if err != nil {
			return nil, err
		}
		result.GroupsRemoved = removed
	}

	// 目标科室信息
	result.ToOrgNode = s.buildOrgNodeInfo(ctx, targetOrgNodeID)

	return result, nil
}

// BatchTransferInput 批量调动的输入参数。
type BatchTransferInput struct {
	EmployeeIDs     []string `json:"employee_ids"`
	TargetOrgNodeID string   `json:"target_org_node_id"`
	Reason          string   `json:"reason"`
}

// BatchTransfer 批量调动员工。
func (s *Service) BatchTransfer(ctx context.Context, input BatchTransferInput) ([]TransferResult, error) {
	if len(input.EmployeeIDs) == 0 {
		return nil, fmt.Errorf("employee_ids 不能为空")
	}
	results := make([]TransferResult, 0, len(input.EmployeeIDs))
	for _, empID := range input.EmployeeIDs {
		result, err := s.Transfer(ctx, empID, TransferInput{
			TargetOrgNodeID: input.TargetOrgNodeID,
			Reason:          input.Reason,
		})
		if err != nil {
			return results, fmt.Errorf("调动员工 %s 失败: %w", empID, err)
		}
		results = append(results, *result)
	}
	return results, nil
}

func (s *Service) buildOrgNodeInfo(ctx context.Context, orgNodeID string) OrgNodeInfo {
	info := OrgNodeInfo{ID: orgNodeID}
	if s.orgNodeResolver == nil {
		return info
	}
	node, err := s.orgNodeResolver.GetByID(ctx, orgNodeID)
	if err != nil {
		return info
	}
	info.Name = node.Name
	names, err := s.orgNodeResolver.GetAncestorNames(ctx, node.Path)
	if err == nil && len(names) > 0 {
		info.PathDisplay = strings.Join(names, " / ")
	} else {
		info.PathDisplay = node.Name
	}
	return info
}
