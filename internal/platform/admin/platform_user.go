package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrPlatformUserNotFound    = errors.New("平台账号不存在")
	ErrInvalidPlatformUser     = errors.New("平台账号用户名、邮箱、组织节点和角色为必填项")
	ErrInvalidPlatformRole     = errors.New("平台账号只允许分配机构管理员或科室管理员角色")
	ErrCannotDisableSelf       = errors.New("不能禁用当前登录账号")
	ErrCannotResetUnknownScope = errors.New("平台账号未绑定组织节点，无法重置默认密码")
	ErrManageScopeDenied       = errors.New("目标资源不在当前管理范围内")
)

type CreatePlatformUserInput struct {
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	OrgNodeID string  `json:"org_node_id"`
	RoleName  string  `json:"role_name"`
}

type PlatformUserNodeRole struct {
	OrgNodeID   string `json:"org_node_id"`
	OrgNodeName string `json:"org_node_name"`
	RoleName    string `json:"role_name"`
}

type PlatformUserResponse struct {
	ID              string                 `json:"id"`
	Username        string                 `json:"username"`
	Email           string                 `json:"email"`
	Phone           *string                `json:"phone,omitempty"`
	Status          string                 `json:"status"`
	MustResetPwd    bool                   `json:"must_reset_pwd"`
	BoundEmployeeID *string                `json:"bound_employee_id,omitempty"`
	Roles           []PlatformUserNodeRole `json:"roles"`
}

type CreatePlatformUserResponse struct {
	User            PlatformUserResponse `json:"user"`
	DefaultPassword string               `json:"default_password"`
}

type ResetPlatformUserPasswordResponse struct {
	DefaultPassword string `json:"default_password"`
	MustResetPwd    bool   `json:"must_reset_pwd"`
}

type platformUserRoleRow struct {
	UserID      string
	OrgNodeID   string
	OrgNodePath string
	OrgNodeName string
	RoleName    string
	NodeCode    string
}

type PlatformUserService struct {
	db *gorm.DB
}

var manageablePlatformRoleNames = []string{
	string(auth.RolePlatformAdmin),
	string(auth.RoleOrgAdmin),
	string(auth.RoleDeptAdmin),
}

func NewPlatformUserService(db *gorm.DB) *PlatformUserService {
	return &PlatformUserService{db: db}
}

func (s *PlatformUserService) List(ctx context.Context, orgNodeID string) ([]PlatformUserResponse, error) {
	if orgNodeID != "" {
		if err := s.ensureNodeInScope(ctx, orgNodeID); err != nil {
			return nil, err
		}
	}

	query := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Model(&auth.User{})
	query = query.Joins("JOIN user_node_roles ON user_node_roles.user_id = platform_users.id").
		Joins("JOIN org_nodes ON org_nodes.id = user_node_roles.org_node_id").
		Joins("JOIN roles ON roles.id = user_node_roles.role_id").
		Where("roles.name IN ?", manageablePlatformRoleNames)

	if scopePath := currentScopePath(ctx); scopePath != "" {
		query = query.Where("org_nodes.path LIKE ?", scopePath+"%")
	}
	if orgNodeID != "" {
		query = query.Where("user_node_roles.org_node_id = ?", orgNodeID)
	}

	var users []auth.User
	if err := query.Distinct("platform_users.*").Order("platform_users.created_at DESC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询平台账号列表失败: %w", err)
	}

	return s.buildPlatformUserResponses(ctx, users)
}

func (s *PlatformUserService) GetByID(ctx context.Context, id string) (*PlatformUserResponse, error) {
	var user auth.User
	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformUserNotFound
		}
		return nil, fmt.Errorf("查询平台账号失败: %w", err)
	}

	items, err := s.buildPlatformUserResponses(ctx, []auth.User{user})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrManageScopeDenied
	}

	return &items[0], nil
}

func (s *PlatformUserService) Create(ctx context.Context, input CreatePlatformUserInput) (*CreatePlatformUserResponse, error) {
	if strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.OrgNodeID) == "" || strings.TrimSpace(input.RoleName) == "" {
		return nil, ErrInvalidPlatformUser
	}
	if !isAllowedPlatformRole(input.RoleName) {
		return nil, ErrInvalidPlatformRole
	}

	var result *CreatePlatformUserResponse
	err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Transaction(func(tx *gorm.DB) error {
		var node tenant.OrgNode
		if err := tx.Where("id = ?", input.OrgNodeID).First(&node).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tenant.ErrNodeNotFound
			}
			return fmt.Errorf("查询组织节点失败: %w", err)
		}
		if !isPathInScope(ctx, node.Path) {
			return ErrManageScopeDenied
		}

		var existing int64
		if err := tx.Model(&auth.User{}).Where("username = ? OR email = ?", strings.TrimSpace(input.Username), strings.TrimSpace(input.Email)).Count(&existing).Error; err != nil {
			return fmt.Errorf("检查平台账号唯一性失败: %w", err)
		}
		if existing > 0 {
			return ErrPlatformUserExists
		}

		defaultPassword := buildDefaultPassword(node.Code)
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成平台账号密码失败: %w", err)
		}

		user := &auth.User{
			ID:           uuid.New().String(),
			Username:     strings.TrimSpace(input.Username),
			Email:        strings.TrimSpace(input.Email),
			Phone:        input.Phone,
			PasswordHash: string(passwordHash),
			Status:       auth.UserStatusActive,
			MustResetPwd: true,
		}
		if err := tx.Create(user).Error; err != nil {
			if isDuplicateKeyError(err) {
				return ErrPlatformUserExists
			}
			return fmt.Errorf("创建平台账号失败: %w", err)
		}

		var role auth.Role
		if err := tx.Where("name = ?", input.RoleName).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return auth.ErrRoleNotFound
			}
			return fmt.Errorf("查询平台账号角色失败: %w", err)
		}

		if err := tx.Create(&auth.UserNodeRole{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			OrgNodeID: node.ID,
			RoleID:    role.ID,
		}).Error; err != nil {
			return fmt.Errorf("绑定平台账号角色失败: %w", err)
		}

		result = &CreatePlatformUserResponse{
			User: PlatformUserResponse{
				ID:           user.ID,
				Username:     user.Username,
				Email:        user.Email,
				Phone:        user.Phone,
				Status:       user.Status,
				MustResetPwd: user.MustResetPwd,
				Roles: []PlatformUserNodeRole{{
					OrgNodeID:   node.ID,
					OrgNodeName: node.Name,
					RoleName:    role.Name,
				}},
			},
			DefaultPassword: defaultPassword,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *PlatformUserService) ResetPassword(ctx context.Context, id string) (*ResetPlatformUserPasswordResponse, error) {
	var result *ResetPlatformUserPasswordResponse
	err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Transaction(func(tx *gorm.DB) error {
		var user auth.User
		if err := tx.Where("id = ?", id).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlatformUserNotFound
			}
			return fmt.Errorf("查询平台账号失败: %w", err)
		}

		roleRows, err := s.loadRoleRows(tx, []string{id})
		if err != nil {
			return err
		}
		rows := roleRows[id]
		if len(rows) == 0 {
			return ErrCannotResetUnknownScope
		}
		rows = filterRoleRowsByScope(ctx, rows)
		if len(rows) == 0 {
			return ErrManageScopeDenied
		}

		defaultPassword := buildDefaultPassword(rows[0].NodeCode)
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成默认密码失败: %w", err)
		}

		user.PasswordHash = string(passwordHash)
		user.MustResetPwd = true
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("重置平台账号密码失败: %w", err)
		}

		result = &ResetPlatformUserPasswordResponse{
			DefaultPassword: defaultPassword,
			MustResetPwd:    true,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *PlatformUserService) Disable(ctx context.Context, id, actorUserID string) error {
	if actorUserID != "" && actorUserID == id {
		return ErrCannotDisableSelf
	}

	item, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrPlatformUserNotFound
	}

	result := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Model(&auth.User{}).Where("id = ?", id).Update("status", auth.UserStatusDisabled)
	if result.Error != nil {
		return fmt.Errorf("禁用平台账号失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPlatformUserNotFound
	}

	return nil
}

func (s *PlatformUserService) buildPlatformUserResponses(ctx context.Context, users []auth.User) ([]PlatformUserResponse, error) {
	if len(users) == 0 {
		return []PlatformUserResponse{}, nil
	}

	userIDs := make([]string, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	roleRows, err := s.loadRoleRows(s.db.WithContext(tenant.SkipTenantGuard(ctx)), userIDs)
	if err != nil {
		return nil, err
	}

	result := make([]PlatformUserResponse, 0, len(users))
	for _, user := range users {
		roles := filterRoleRowsByScope(ctx, roleRows[user.ID])
		if len(roles) == 0 {
			continue
		}
		sort.Slice(roles, func(i, j int) bool {
			if roles[i].RoleName == roles[j].RoleName {
				return roles[i].OrgNodeName < roles[j].OrgNodeName
			}
			return roles[i].RoleName < roles[j].RoleName
		})

		mappedRoles := make([]PlatformUserNodeRole, 0, len(roles))
		for _, role := range roles {
			mappedRoles = append(mappedRoles, PlatformUserNodeRole{
				OrgNodeID:   role.OrgNodeID,
				OrgNodeName: role.OrgNodeName,
				RoleName:    role.RoleName,
			})
		}

		result = append(result, PlatformUserResponse{
			ID:              user.ID,
			Username:        user.Username,
			Email:           user.Email,
			Phone:           user.Phone,
			Status:          user.Status,
			MustResetPwd:    user.MustResetPwd,
			BoundEmployeeID: user.BoundEmployeeID,
			Roles:           mappedRoles,
		})
	}

	return result, nil
}

func (s *PlatformUserService) loadRoleRows(db *gorm.DB, userIDs []string) (map[string][]platformUserRoleRow, error) {
	var rows []platformUserRoleRow
	err := db.Table("user_node_roles").
		Select("user_node_roles.user_id, user_node_roles.org_node_id, org_nodes.path AS org_node_path, org_nodes.name AS org_node_name, org_nodes.code AS node_code, roles.name AS role_name").
		Joins("JOIN org_nodes ON org_nodes.id = user_node_roles.org_node_id").
		Joins("JOIN roles ON roles.id = user_node_roles.role_id").
		Where("roles.name IN ?", manageablePlatformRoleNames).
		Where("user_node_roles.user_id IN ?", userIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("查询平台账号角色失败: %w", err)
	}

	result := make(map[string][]platformUserRoleRow, len(userIDs))
	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row)
	}

	return result, nil
}

func isAllowedPlatformRole(roleName string) bool {
	return roleName == string(auth.RoleOrgAdmin) || roleName == string(auth.RoleDeptAdmin)
}

func currentScopePath(ctx context.Context) string {
	return strings.TrimRight(tenant.GetOrgNodePath(ctx), "/")
}

func isPathInScope(ctx context.Context, path string) bool {
	scopePath := currentScopePath(ctx)
	if scopePath == "" {
		return true
	}
	path = strings.TrimRight(path, "/")
	return path == scopePath || strings.HasPrefix(path, scopePath+"/")
}

func filterRoleRowsByScope(ctx context.Context, rows []platformUserRoleRow) []platformUserRoleRow {
	if currentScopePath(ctx) == "" {
		return rows
	}
	filtered := make([]platformUserRoleRow, 0, len(rows))
	for _, row := range rows {
		if isPathInScope(ctx, row.OrgNodePath) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (s *PlatformUserService) ensureNodeInScope(ctx context.Context, nodeID string) error {
	var node tenant.OrgNode
	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ?", nodeID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tenant.ErrNodeNotFound
		}
		return fmt.Errorf("查询组织节点失败: %w", err)
	}
	if !isPathInScope(ctx, node.Path) {
		return ErrManageScopeDenied
	}
	return nil
}

type PlatformUserHandler struct {
	svc *PlatformUserService
}

func NewPlatformUserHandler(svc *PlatformUserService) *PlatformUserHandler {
	return &PlatformUserHandler{svc: svc}
}

func (h *PlatformUserHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context(), r.URL.Query().Get("org_node_id"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, items)
}

func (h *PlatformUserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, item)
}

func (h *PlatformUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreatePlatformUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	result, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, result)
}

func (h *PlatformUserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.ResetPassword(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

func (h *PlatformUserHandler) Disable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	actorID := ""
	if claims != nil {
		actorID = claims.UserID
	}

	if err := h.svc.Disable(r.Context(), chi.URLParam(r, "id"), actorID); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "disabled"})
}

func (h *PlatformUserHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidPlatformUser):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidPlatformRole):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrPlatformUserExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrPlatformUserNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrCannotDisableSelf):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCannotResetUnknownScope):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrManageScopeDenied):
		response.Forbidden(w, err.Error())
	case errors.Is(err, tenant.ErrNodeNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, auth.ErrRoleNotFound):
		response.NotFound(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
