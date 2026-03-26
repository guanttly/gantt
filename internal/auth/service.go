package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"

	appconfig "gantt-saas/internal/infra/config"
	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound        = errors.New("用户不存在")
	ErrUserDisabled        = errors.New("用户已禁用")
	ErrInvalidCredentials  = errors.New("用户名或密码错误")
	ErrUsernameExists      = errors.New("用户名已存在")
	ErrEmailExists         = errors.New("邮箱已存在")
	ErrPublicRegisterRole  = errors.New("公开注册仅允许创建 employee 角色")
	ErrWeakPassword        = errors.New("密码强度不足：至少 8 位，需包含大写、小写和数字")
	ErrNoNodePermission    = errors.New("用户无该组织节点的访问权限")
	ErrAccountLocked       = errors.New("账户已锁定，请 15 分钟后再试")
	ErrRoleNotFound        = errors.New("角色不存在")
	ErrNodeRoleExists      = errors.New("用户在该节点已拥有此角色")
	ErrNodeRoleNotFound    = errors.New("用户节点角色关联不存在")
	ErrOldPasswordMismatch = errors.New("旧密码不正确")
	ErrNotRequireReset     = errors.New("当前用户无需强制重置密码")
)

const (
	loginLockPrefix  = "login:lock:"
	loginCountPrefix = "login:fail:"
	maxLoginAttempts = 5
	lockDuration     = 15 * time.Minute
	failCountWindow  = 5 * time.Minute
)

// isStrongPassword 校验密码强度：至少 8 位，包含大写、小写和数字。
func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// RegisterInput 注册输入。
type RegisterInput struct {
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	Password  string  `json:"password"`
	OrgNodeID string  `json:"org_node_id"`
	RoleName  string  `json:"role_name"`
}

// LoginInput 登录输入。
type LoginInput struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	OrgNodeID string `json:"org_node_id"`
}

// LoginResponse 登录响应。
type LoginResponse struct {
	AccessToken    string         `json:"access_token"`
	RefreshToken   string         `json:"refresh_token"`
	ExpiresIn      int            `json:"expires_in"`
	User           UserInfo       `json:"user"`
	CurrentNode    *NodeRoleInfo  `json:"current_node"`
	AvailableNodes []NodeRoleInfo `json:"available_nodes"`
	MustResetPwd   bool           `json:"must_reset_pwd"`
}

// SwitchNodeInput 切换节点输入。
type SwitchNodeInput struct {
	OrgNodeID string `json:"org_node_id"`
}

// ResetPasswordInput 密码重置输入。
type ResetPasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ForceResetPasswordInput 强制重置密码输入（首次登录后使用）。
type ForceResetPasswordInput struct {
	NewPassword string `json:"new_password"`
}

// AssignRoleInput 分配角色输入。
type AssignRoleInput struct {
	UserID    string `json:"user_id"`
	OrgNodeID string `json:"org_node_id"`
	RoleName  string `json:"role_name"`
}

// MeResponse 当前用户信息响应。
type MeResponse struct {
	User           UserInfo       `json:"user"`
	CurrentNode    *NodeRoleInfo  `json:"current_node,omitempty"`
	AvailableNodes []NodeRoleInfo `json:"available_nodes"`
}

// Service 认证业务逻辑层。
type Service struct {
	repo       *Repository
	tenantRepo *tenant.Repository
	jwt        *JWTManager
	rdb        *redis.Client
}

// NewService 创建认证服务。
func NewService(repo *Repository, tenantRepo *tenant.Repository, jwt *JWTManager, rdb *redis.Client) *Service {
	return &Service{
		repo:       repo,
		tenantRepo: tenantRepo,
		jwt:        jwt,
		rdb:        rdb,
	}
}

// Register 用户注册。
func (s *Service) Register(ctx context.Context, input RegisterInput) (*LoginResponse, error) {
	requestedRoleName := input.RoleName
	if requestedRoleName != "" && requestedRoleName != string(RoleEmployee) {
		return nil, ErrPublicRegisterRole
	}

	// 校验密码强度
	if !isStrongPassword(input.Password) {
		return nil, ErrWeakPassword
	}

	// 检查用户名唯一
	if _, err := s.repo.GetUserByUsername(ctx, input.Username); err == nil {
		return nil, ErrUsernameExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}

	// 检查邮箱唯一
	if _, err := s.repo.GetUserByEmail(ctx, input.Email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	user := &User{
		ID:           uuid.New().String(),
		Username:     input.Username,
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: string(hashedPassword),
		Status:       UserStatusActive,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 公开注册仅允许落到最低权限 employee 角色。
	if input.OrgNodeID != "" {
		if _, err := s.tenantRepo.GetByID(ctx, input.OrgNodeID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNoNodePermission
			}
			return nil, fmt.Errorf("查询组织节点失败: %w", err)
		}

		if err := s.assignRoleInternal(ctx, user.ID, input.OrgNodeID, string(RoleEmployee)); err != nil {
			return nil, fmt.Errorf("分配角色失败: %w", err)
		}

		// 直接签发 Token 返回
		return s.buildLoginResponse(ctx, user, input.OrgNodeID)
	}

	// 无节点关联时仅返回用户信息
	return &LoginResponse{
		User:           user.ToInfo(),
		AvailableNodes: []NodeRoleInfo{},
		MustResetPwd:   user.MustResetPwd,
	}, nil
}

// Login 用户登录。
func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginResponse, error) {
	// 检查账户锁定
	locked, err := s.isAccountLocked(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("检查锁定状态失败: %w", err)
	}
	if locked {
		return nil, ErrAccountLocked
	}

	// 查询用户
	user, err := s.repo.GetUserByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordLoginFailure(ctx, input.Username)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	if user.Status == UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		s.recordLoginFailure(ctx, input.Username)
		return nil, ErrInvalidCredentials
	}

	// 登录成功，清除失败计数
	s.clearLoginFailure(ctx, input.Username)

	// 如果未指定节点，取用户第一个可用节点
	orgNodeID := input.OrgNodeID
	if orgNodeID == "" {
		unrs, err := s.repo.GetUserNodeRoles(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("查询用户节点失败: %w", err)
		}
		if len(unrs) > 0 {
			orgNodeID = s.pickPreferredOrgNodeID(unrs)
		}
	}

	if orgNodeID == "" {
		// 用户没有任何节点关联，返回空的登录响应
		return &LoginResponse{
			User:           user.ToInfo(),
			AvailableNodes: []NodeRoleInfo{},
			MustResetPwd:   user.MustResetPwd,
		}, nil
	}

	return s.buildLoginResponse(ctx, user, orgNodeID)
}

// RefreshToken 刷新 Access Token。
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	claims, err := s.jwt.ParseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.Status == UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	return s.buildLoginResponse(ctx, user, claims.OrgNodeID)
}

// SwitchNode 切换当前组织节点。
func (s *Service) SwitchNode(ctx context.Context, claims *Claims, input SwitchNodeInput) (*LoginResponse, error) {
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return s.buildLoginResponse(ctx, user, input.OrgNodeID)
}

// ResetPassword 重置密码。
func (s *Service) ResetPassword(ctx context.Context, userID string, input ResetPasswordInput) error {
	// 校验新密码强度
	if !isStrongPassword(input.NewPassword) {
		return ErrWeakPassword
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 校验旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)); err != nil {
		return ErrOldPasswordMismatch
	}

	// 哈希新密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	user.PasswordHash = string(hashed)
	return s.repo.UpdateUser(ctx, user)
}

// GetMe 获取当前用户信息和可切换节点列表。
func (s *Service) GetMe(ctx context.Context, claims *Claims) (*MeResponse, error) {
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	available, err := s.getUserAvailableNodes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// 找到当前节点信息
	var currentNode *NodeRoleInfo
	for _, n := range available {
		if n.NodeID == claims.OrgNodeID {
			cn := n
			currentNode = &cn
			break
		}
	}

	return &MeResponse{
		User:           user.ToInfo(),
		CurrentNode:    currentNode,
		AvailableNodes: available,
	}, nil
}

// AssignRole 分配用户到组织节点的角色。
func (s *Service) AssignRole(ctx context.Context, input AssignRoleInput) (*UserNodeRole, error) {
	// 检查用户存在
	if _, err := s.repo.GetUserByID(ctx, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 检查节点存在
	if _, err := s.tenantRepo.GetByID(ctx, input.OrgNodeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoNodePermission
		}
		return nil, err
	}

	return s.assignRoleAndReturn(ctx, input.UserID, input.OrgNodeID, input.RoleName)
}

// RemoveRole 移除用户在节点的角色关联。
func (s *Service) RemoveRole(ctx context.Context, id string) error {
	return s.repo.DeleteUserNodeRole(ctx, id)
}

// SeedSystemRoles 初始化系统预置角色。
func (s *Service) SeedSystemRoles(ctx context.Context) error {
	for _, role := range SystemRoles {
		r := role // 避免闭包引用问题
		if err := s.repo.UpsertRole(ctx, &r); err != nil {
			return fmt.Errorf("初始化角色 %s 失败: %w", role.Name, err)
		}
	}
	return nil
}

// SeedDefaultAdmin 初始化默认平台管理员（幂等）。
func (s *Service) SeedDefaultAdmin(ctx context.Context, cfg appconfig.AdminConfig) error {
	// 检查用户是否已存在
	if _, err := s.repo.GetUserByUsername(ctx, cfg.Username); err == nil {
		return nil // 已存在，跳过
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("检查默认管理员失败: %w", err)
	}

	// 创建“平台管理”根组织节点（幂等）
	var platformNode *tenant.OrgNode
	existing, err := s.tenantRepo.GetByParentAndCode(ctx, nil, "platform-root")
	if err == nil {
		platformNode = existing
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		nodeID := uuid.New().String()
		platformNode = &tenant.OrgNode{
			ID:           nodeID,
			ParentID:     nil,
			NodeType:     tenant.NodeTypeOrganization,
			Name:         "平台管理",
			Code:         "platform-root",
			Path:         "/" + nodeID,
			Depth:        0,
			IsLoginPoint: true,
			Status:       tenant.StatusActive,
		}
		if err := s.tenantRepo.Create(ctx, platformNode); err != nil {
			return fmt.Errorf("创建平台管理节点失败: %w", err)
		}
	} else {
		return fmt.Errorf("检查平台管理节点失败: %w", err)
	}

	// 创建管理员用户
	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	adminUser := &User{
		ID:           uuid.New().String(),
		Username:     cfg.Username,
		Email:        cfg.Email,
		PasswordHash: string(hashed),
		Status:       UserStatusActive,
		MustResetPwd: true,
	}

	if err := s.repo.CreateUser(ctx, adminUser); err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}

	// 分配 platform_admin 角色
	if err := s.assignRoleInternal(ctx, adminUser.ID, platformNode.ID, string(RolePlatformAdmin)); err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	return nil
}

// ForceResetPassword 强制重置密码（仅对 must_reset_pwd=true 的用户有效）。
func (s *Service) ForceResetPassword(ctx context.Context, userID string, input ForceResetPasswordInput) error {
	if !isStrongPassword(input.NewPassword) {
		return ErrWeakPassword
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if !user.MustResetPwd {
		return ErrNotRequireReset
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	user.PasswordHash = string(hashed)
	user.MustResetPwd = false
	return s.repo.UpdateUser(ctx, user)
}

// ── 内部方法 ──

func (s *Service) buildLoginResponse(ctx context.Context, user *User, orgNodeID string) (*LoginResponse, error) {
	// 查询用户在目标节点的角色
	unr, err := s.repo.GetUserNodeRole(ctx, user.ID, orgNodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoNodePermission
		}
		return nil, fmt.Errorf("查询节点角色失败: %w", err)
	}

	// 查询节点信息
	node, err := s.tenantRepo.GetByID(ctx, orgNodeID)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %w", err)
	}

	roleName := ""
	if unr.Role != nil {
		roleName = unr.Role.Name
	}

	// 签发 Token
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, orgNodeID, node.Path, roleName)
	if err != nil {
		return nil, fmt.Errorf("签发 Access Token 失败: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, orgNodeID, node.Path, roleName)
	if err != nil {
		return nil, fmt.Errorf("签发 Refresh Token 失败: %w", err)
	}

	// 查询所有可用节点
	available, err := s.getUserAvailableNodes(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	currentNode := &NodeRoleInfo{
		NodeID:   node.ID,
		NodeName: node.Name,
		NodePath: node.Path,
		RoleName: roleName,
	}

	return &LoginResponse{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresIn:      s.jwt.AccessTokenTTLSeconds(),
		User:           user.ToInfo(),
		CurrentNode:    currentNode,
		AvailableNodes: available,
		MustResetPwd:   user.MustResetPwd,
	}, nil
}

func (s *Service) getUserAvailableNodes(ctx context.Context, userID string) ([]NodeRoleInfo, error) {
	unrs, err := s.repo.GetUserNodeRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户节点角色失败: %w", err)
	}

	nodes := make([]NodeRoleInfo, 0, len(unrs))
	for _, unr := range unrs {
		node, err := s.tenantRepo.GetByID(ctx, unr.OrgNodeID)
		if err != nil {
			continue // 节点可能已被删除
		}
		roleName := ""
		if unr.Role != nil {
			roleName = unr.Role.Name
		}
		nodes = append(nodes, NodeRoleInfo{
			NodeID:   node.ID,
			NodeName: node.Name,
			NodePath: node.Path,
			RoleName: roleName,
		})
	}

	return nodes, nil
}

func (s *Service) pickPreferredOrgNodeID(unrs []UserNodeRole) string {
	if len(unrs) == 0 {
		return ""
	}

	best := unrs[0]
	bestScore := rolePriority(best)

	for _, unr := range unrs[1:] {
		score := rolePriority(unr)
		if score > bestScore {
			best = unr
			bestScore = score
		}
	}

	return best.OrgNodeID
}

func rolePriority(unr UserNodeRole) int {
	if unr.Role == nil {
		return 0
	}

	switch RoleName(unr.Role.Name) {
	case RolePlatformAdmin:
		return 500
	case RoleOrgAdmin:
		return 400
	case RoleDeptAdmin:
		return 300
	case RoleScheduler:
		return 200
	case RoleEmployee:
		return 100
	default:
		return 0
	}
}

func (s *Service) assignRoleInternal(ctx context.Context, userID, orgNodeID, roleName string) error {
	_, err := s.assignRoleAndReturn(ctx, userID, orgNodeID, roleName)
	return err
}

func (s *Service) assignRoleAndReturn(ctx context.Context, userID, orgNodeID, roleName string) (*UserNodeRole, error) {
	role, err := s.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	// 检查是否已存在
	existing, err := s.repo.GetUserNodeRole(ctx, userID, orgNodeID)
	if err == nil && existing != nil {
		return nil, ErrNodeRoleExists
	}

	unr := &UserNodeRole{
		ID:        uuid.New().String(),
		UserID:    userID,
		OrgNodeID: orgNodeID,
		RoleID:    role.ID,
	}

	if err := s.repo.CreateUserNodeRole(ctx, unr); err != nil {
		return nil, fmt.Errorf("创建用户节点角色关联失败: %w", err)
	}

	unr.Role = role
	return unr, nil
}

// ── 登录限流 ──

func (s *Service) isAccountLocked(ctx context.Context, username string) (bool, error) {
	if s.rdb == nil {
		return false, nil
	}
	key := loginLockPrefix + username
	result, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (s *Service) recordLoginFailure(ctx context.Context, username string) {
	if s.rdb == nil {
		return
	}
	key := loginCountPrefix + username
	count, _ := s.rdb.Incr(ctx, key).Result()
	s.rdb.Expire(ctx, key, failCountWindow)

	if count >= int64(maxLoginAttempts) {
		lockKey := loginLockPrefix + username
		s.rdb.Set(ctx, lockKey, "1", lockDuration)
		s.rdb.Del(ctx, key) // 清除计数
	}
}

func (s *Service) clearLoginFailure(ctx context.Context, username string) {
	if s.rdb == nil {
		return
	}
	s.rdb.Del(ctx, loginCountPrefix+username, loginLockPrefix+username)
}
