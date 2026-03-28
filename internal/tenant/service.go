package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNodeNotFound                 = errors.New("组织节点不存在")
	ErrNodeSuspended                = errors.New("组织节点已停用")
	ErrCodeDuplicate                = errors.New("同级节点编码已存在")
	ErrInvalidNodeType              = errors.New("无效的节点类型")
	ErrInvalidRootType              = errors.New("根组织仅允许使用机构类型")
	ErrCannotDeleteRoot             = errors.New("不能删除有子节点的节点")
	ErrProtectedNode                = errors.New("平台管理根节点不允许删除")
	ErrManageScopeDenied            = errors.New("目标资源不在当前管理范围内")
	ErrCannotDeleteCurrentScopeRoot = errors.New("当前管理根节点不允许删除")
	ErrCannotMoveCurrentScopeRoot   = errors.New("当前管理根节点不允许移动")
	ErrInvalidHierarchy             = errors.New("不允许的层级关系")
)

// allowedChildTypes 定义行政架构的固定层级关系。
var allowedChildTypes = map[string][]string{
	NodeTypeOrganization: {NodeTypeCampus, NodeTypeDepartment, NodeTypeCustom},
	NodeTypeCampus:       {NodeTypeDepartment, NodeTypeCustom},
	NodeTypeCustom:       {NodeTypeDepartment, NodeTypeCustom},
	NodeTypeDepartment:   {}, // 科室是叶子节点，不能建下级
}

const protectedPlatformRootCode = "platform-root"

// validNodeTypes 允许的节点类型。
var validNodeTypes = map[string]bool{
	NodeTypeOrganization: true,
	NodeTypeCampus:       true,
	NodeTypeDepartment:   true,
	NodeTypeCustom:       true,
}

// CreateNodeInput 创建节点的输入参数。
type CreateNodeInput struct {
	ParentID     *string `json:"parent_id"`
	NodeType     string  `json:"node_type"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	IsLoginPoint bool    `json:"is_login_point"`
}

// UpdateNodeInput 更新节点的输入参数。
type UpdateNodeInput struct {
	Name         *string `json:"name,omitempty"`
	IsLoginPoint *bool   `json:"is_login_point,omitempty"`
}

// MoveNodeInput 移动节点的输入参数。
type MoveNodeInput struct {
	NewParentID string `json:"new_parent_id"`
}

// Service 组织树业务逻辑层。
type Service struct {
	repo *Repository
}

// NewService 创建组织树服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 创建组织节点。
func (s *Service) Create(ctx context.Context, input CreateNodeInput) (*OrgNode, error) {
	// 校验节点类型
	if !validNodeTypes[input.NodeType] {
		return nil, ErrInvalidNodeType
	}

	if (input.ParentID == nil || *input.ParentID == "") && input.NodeType != NodeTypeOrganization {
		return nil, ErrInvalidRootType
	}

	node := &OrgNode{
		ID:           uuid.New().String(),
		ParentID:     input.ParentID,
		NodeType:     input.NodeType,
		Name:         input.Name,
		Code:         input.Code,
		IsLoginPoint: input.IsLoginPoint,
		Status:       StatusActive,
	}

	// 计算 path 和 depth
	if input.ParentID != nil && *input.ParentID != "" {
		parent, err := s.repo.GetByID(ctx, *input.ParentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNodeNotFound
			}
			return nil, fmt.Errorf("查询父节点失败: %w", err)
		}
		if !parent.IsActive() {
			return nil, ErrNodeSuspended
		}
		if err := s.ensureNodeInScope(ctx, parent); err != nil {
			return nil, err
		}
		// 校验层级关系
		if err := validateNodeHierarchy(parent.NodeType, input.NodeType); err != nil {
			return nil, err
		}
		node.Path = parent.Path + "/" + node.ID
		node.Depth = parent.Depth + 1
	} else {
		if !hasGlobalManageScope(ctx) {
			return nil, ErrManageScopeDenied
		}
		// 顶级节点
		node.ParentID = nil
		node.Path = "/" + node.ID
		node.Depth = 0
	}

	// 检查同级 code 唯一
	existing, err := s.repo.GetByParentAndCode(ctx, input.ParentID, input.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查编码唯一性失败: %w", err)
	}
	if existing != nil {
		return nil, ErrCodeDuplicate
	}

	// 保存
	if err := s.repo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("创建节点失败: %w", err)
	}

	return node, nil
}

// GetByID 根据 ID 获取节点。
func (s *Service) GetByID(ctx context.Context, id string) (*OrgNode, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

// Update 更新节点基本信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateNodeInput) (*OrgNode, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return nil, err
	}

	if input.Name != nil {
		node.Name = *input.Name
	}
	if input.IsLoginPoint != nil {
		node.IsLoginPoint = *input.IsLoginPoint
	}

	if err := s.repo.Update(ctx, node); err != nil {
		return nil, fmt.Errorf("更新节点失败: %w", err)
	}

	return node, nil
}

// Suspend 停用节点及其所有后代。
func (s *Service) Suspend(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeNotFound
		}
		return err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return err
	}

	return s.repo.UpdateStatusByPath(ctx, node.Path, StatusSuspended)
}

// Activate 启用节点及其所有后代。
func (s *Service) Activate(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeNotFound
		}
		return err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return err
	}

	return s.repo.UpdateStatusByPath(ctx, node.Path, StatusActive)
}

// GetChildren 获取直接子节点列表。
func (s *Service) GetChildren(ctx context.Context, parentID string) ([]OrgNode, error) {
	parent, err := s.repo.GetByID(ctx, parentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, parent); err != nil {
		return nil, err
	}
	return s.repo.GetChildren(ctx, parentID)
}

// GetTree 获取以 rootID 为根的完整树结构。
func (s *Service) GetTree(ctx context.Context, rootID string) (*OrgNodeTree, error) {
	root, err := s.repo.GetByID(ctx, rootID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, root); err != nil {
		return nil, err
	}

	descendants, err := s.repo.GetDescendants(ctx, root.Path)
	if err != nil {
		return nil, fmt.Errorf("查询后代节点失败: %w", err)
	}

	return buildTree(root, descendants), nil
}

// GetRootTrees 获取所有顶级节点及其完整子树。
func (s *Service) GetRootTrees(ctx context.Context) ([]*OrgNodeTree, error) {
	if !hasGlobalManageScope(ctx) {
		scopeNodeID := GetOrgNodeID(ctx)
		if scopeNodeID == "" {
			return nil, ErrManageScopeDenied
		}
		tree, err := s.GetTree(ctx, scopeNodeID)
		if err != nil {
			return nil, err
		}
		return []*OrgNodeTree{tree}, nil
	}

	roots, err := s.repo.GetRootNodes(ctx)
	if err != nil {
		return nil, err
	}

	trees := make([]*OrgNodeTree, 0, len(roots))
	for _, root := range roots {
		descendants, err := s.repo.GetDescendants(ctx, root.Path)
		if err != nil {
			return nil, fmt.Errorf("查询根节点 %s 的组织树失败: %w", root.ID, err)
		}

		rootCopy := root
		trees = append(trees, buildTree(&rootCopy, descendants))
	}

	return trees, nil
}

// GetRootNodes 获取所有顶级组织节点。
func (s *Service) GetRootNodes(ctx context.Context) ([]OrgNode, error) {
	return s.repo.GetRootNodes(ctx)
}

// Delete 删除节点（仅允许删除叶子节点）。
func (s *Service) Delete(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeNotFound
		}
		return err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return err
	}

	if isProtectedNode(node) {
		return ErrProtectedNode
	}
	if !hasGlobalManageScope(ctx) && id == currentManageRootID(ctx) {
		return ErrCannotDeleteCurrentScopeRoot
	}

	// 检查是否有子节点
	count, err := s.repo.CountByParent(ctx, id)
	if err != nil {
		return fmt.Errorf("检查子节点失败: %w", err)
	}
	if count > 0 {
		return ErrCannotDeleteRoot
	}

	return s.repo.Delete(ctx, id)
}

func isProtectedNode(node *OrgNode) bool {
	return node != nil && node.ParentID == nil && node.Code == protectedPlatformRootCode
}

// validateNodeHierarchy 校验父子节点类型的层级关系。
func validateNodeHierarchy(parentType, childType string) error {
	allowed, ok := allowedChildTypes[parentType]
	if !ok {
		return ErrInvalidHierarchy
	}
	for _, t := range allowed {
		if t == childType {
			return nil
		}
	}
	return fmt.Errorf("%w: 不允许在 %s 下创建 %s 类型节点", ErrInvalidHierarchy, parentType, childType)
}

// IsLeafNodeType 检查节点类型是否为叶子类型（department）。
func IsLeafNodeType(nodeType string) bool {
	return nodeType == NodeTypeDepartment
}

// Move 移动节点到新的父节点下（更新节点及其所有后代的 path 和 depth）。
func (s *Service) Move(ctx context.Context, id string, input MoveNodeInput) (*OrgNode, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, node); err != nil {
		return nil, err
	}

	newParent, err := s.repo.GetByID(ctx, input.NewParentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("目标父节点不存在")
		}
		return nil, err
	}
	if err := s.ensureNodeInScope(ctx, newParent); err != nil {
		return nil, err
	}

	if !newParent.IsActive() {
		return nil, ErrNodeSuspended
	}
	if !hasGlobalManageScope(ctx) && id == currentManageRootID(ctx) {
		return nil, ErrCannotMoveCurrentScopeRoot
	}

	// 检查同级 code 唯一
	existing, err := s.repo.GetByParentAndCode(ctx, &input.NewParentID, node.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查编码唯一性失败: %w", err)
	}
	if existing != nil && existing.ID != id {
		return nil, ErrCodeDuplicate
	}

	// 计算新路径和 depth 差值
	oldPath := node.Path
	newPath := newParent.Path + "/" + node.ID
	depthDelta := (newParent.Depth + 1) - node.Depth

	// 更新父节点
	if err := s.repo.UpdateParent(ctx, id, &input.NewParentID); err != nil {
		return nil, fmt.Errorf("更新父节点失败: %w", err)
	}

	// 更新所有后代的 path
	if err := s.repo.UpdatePathPrefix(ctx, oldPath, newPath); err != nil {
		return nil, fmt.Errorf("更新路径失败: %w", err)
	}

	// 更新所有后代的 depth
	if depthDelta != 0 {
		if err := s.repo.UpdateDepthByPath(ctx, newPath, depthDelta); err != nil {
			return nil, fmt.Errorf("更新深度失败: %w", err)
		}
	}

	// 返回更新后的节点
	return s.repo.GetByID(ctx, id)
}

// GetByIDs 根据 ID 列表批量查询节点。
func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]OrgNode, error) {
	return s.repo.GetByIDs(ctx, ids)
}

// GetDescendantIDs 获取某节点及其所有活跃后代的 ID 列表。
func (s *Service) GetDescendantIDs(ctx context.Context, nodePath string) ([]string, error) {
	return s.repo.GetDescendantIDs(ctx, nodePath)
}

// GetAncestorNames 根据节点路径获取从根到该节点的所有名称（用于路径展示）。
// path 格式如 /org-id/campus-id/dept-id，返回 ["鼓楼医院", "本部院区", "放射科"]。
func (s *Service) GetAncestorNames(ctx context.Context, nodePath string) ([]string, error) {
	parts := strings.Split(strings.Trim(nodePath, "/"), "/")
	if len(parts) == 0 {
		return nil, nil
	}

	var nodes []OrgNode
	if err := s.repo.db.WithContext(SkipTenantGuard(ctx)).
		Where("id IN ?", parts).
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	nameMap := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nameMap[n.ID] = n.Name
	}

	names := make([]string, 0, len(parts))
	for _, id := range parts {
		if name, ok := nameMap[id]; ok {
			names = append(names, name)
		}
	}
	return names, nil
}

func hasGlobalManageScope(ctx context.Context) bool {
	scopeID := strings.TrimSpace(GetOrgNodeID(ctx))
	scopePath := strings.TrimRight(GetOrgNodePath(ctx), "/")
	if scopeID == "" && scopePath == "" {
		return true
	}
	return scopeID == protectedPlatformRootCode || scopePath == "/"+protectedPlatformRootCode
}

func currentManageRootID(ctx context.Context) string {
	scopeID := strings.TrimSpace(GetOrgNodeID(ctx))
	if scopeID != "" {
		return scopeID
	}
	scopePath := strings.Trim(strings.TrimSpace(GetOrgNodePath(ctx)), "/")
	if scopePath == "" {
		return ""
	}
	parts := strings.Split(scopePath, "/")
	return parts[len(parts)-1]
}

func (s *Service) ensureNodeInScope(ctx context.Context, node *OrgNode) error {
	if node == nil || hasGlobalManageScope(ctx) {
		return nil
	}
	scopePath := strings.TrimRight(GetOrgNodePath(ctx), "/")
	if scopePath == "" {
		return nil
	}
	if node.Path == scopePath || strings.HasPrefix(node.Path, scopePath+"/") {
		return nil
	}
	return ErrManageScopeDenied
}

// buildTree 将扁平列表构建为树结构。
func buildTree(root *OrgNode, nodes []OrgNode) *OrgNodeTree {
	nodeMap := make(map[string]*OrgNodeTree)
	rootTree := &OrgNodeTree{OrgNode: *root}
	nodeMap[root.ID] = rootTree

	for _, n := range nodes {
		if n.ID == root.ID {
			continue
		}
		tree := &OrgNodeTree{OrgNode: n}
		nodeMap[n.ID] = tree
	}

	for _, n := range nodes {
		if n.ID == root.ID || n.ParentID == nil {
			continue
		}
		if parent, ok := nodeMap[*n.ParentID]; ok {
			parent.Children = append(parent.Children, nodeMap[n.ID])
		}
	}

	return rootTree
}
