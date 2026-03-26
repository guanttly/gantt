package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound = errors.New("机构不存在")
	ErrInvalidOrganization  = errors.New("机构名称、编码和管理员账号信息为必填项")
	ErrPlatformUserExists   = errors.New("机构管理员用户名或邮箱已存在")
)

type CreateOrganizationInput struct {
	Name         string                  `json:"name"`
	Code         string                  `json:"code"`
	ContactName  *string                 `json:"contact_name,omitempty"`
	ContactPhone *string                 `json:"contact_phone,omitempty"`
	Admin        CreateOrganizationAdmin `json:"admin"`
}

type CreateOrganizationAdmin struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone,omitempty"`
}

type OrganizationResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Status       string  `json:"status"`
}

type CreatedAdminResponse struct {
	ID              string `json:"id"`
	Username        string `json:"username"`
	DefaultPassword string `json:"default_password"`
	MustResetPwd    bool   `json:"must_reset_pwd"`
}

type CreateOrganizationResponse struct {
	Organization OrganizationResponse `json:"organization"`
	Admin        CreatedAdminResponse `json:"admin"`
}

type OrganizationService struct {
	db *gorm.DB
}

func NewOrganizationService(db *gorm.DB) *OrganizationService {
	return &OrganizationService{db: db}
}

func (s *OrganizationService) Create(ctx context.Context, input CreateOrganizationInput) (*CreateOrganizationResponse, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Admin.Username) == "" || strings.TrimSpace(input.Admin.Email) == "" {
		return nil, ErrInvalidOrganization
	}

	var result *CreateOrganizationResponse
	err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Transaction(func(tx *gorm.DB) error {
		var usernameCount int64
		if err := tx.Model(&auth.User{}).Where("username = ? OR email = ?", input.Admin.Username, input.Admin.Email).Count(&usernameCount).Error; err != nil {
			return fmt.Errorf("检查机构管理员唯一性失败: %w", err)
		}
		if usernameCount > 0 {
			return ErrPlatformUserExists
		}

		org := &tenant.OrgNode{
			ID:           uuid.New().String(),
			NodeType:     tenant.NodeTypeOrganization,
			Name:         strings.TrimSpace(input.Name),
			Code:         strings.TrimSpace(input.Code),
			ContactName:  input.ContactName,
			ContactPhone: input.ContactPhone,
			Path:         "",
			Depth:        0,
			IsLoginPoint: true,
			Status:       tenant.StatusActive,
		}
		org.Path = "/" + org.ID

		if err := tx.Create(org).Error; err != nil {
			if isDuplicateKeyError(err) {
				return tenant.ErrCodeDuplicate
			}
			return fmt.Errorf("创建机构失败: %w", err)
		}

		defaultPassword := buildDefaultPassword(org.Code)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成机构管理员密码失败: %w", err)
		}

		adminUser := &auth.User{
			ID:           uuid.New().String(),
			Username:     strings.TrimSpace(input.Admin.Username),
			Email:        strings.TrimSpace(input.Admin.Email),
			Phone:        input.Admin.Phone,
			PasswordHash: string(hashedPassword),
			Status:       auth.UserStatusActive,
			MustResetPwd: true,
		}
		if err := tx.Create(adminUser).Error; err != nil {
			if isDuplicateKeyError(err) {
				return ErrPlatformUserExists
			}
			return fmt.Errorf("创建机构管理员失败: %w", err)
		}

		role := &auth.Role{}
		if err := tx.Where("name = ?", auth.RoleOrgAdmin).First(role).Error; err != nil {
			return fmt.Errorf("查询机构管理员角色失败: %w", err)
		}

		unr := &auth.UserNodeRole{
			ID:        uuid.New().String(),
			UserID:    adminUser.ID,
			OrgNodeID: org.ID,
			RoleID:    role.ID,
		}
		if err := tx.Create(unr).Error; err != nil {
			return fmt.Errorf("关联机构管理员角色失败: %w", err)
		}

		result = &CreateOrganizationResponse{
			Organization: OrganizationResponse{
				ID:           org.ID,
				Name:         org.Name,
				Code:         org.Code,
				ContactName:  org.ContactName,
				ContactPhone: org.ContactPhone,
				Status:       org.Status,
			},
			Admin: CreatedAdminResponse{
				ID:              adminUser.ID,
				Username:        adminUser.Username,
				DefaultPassword: defaultPassword,
				MustResetPwd:    adminUser.MustResetPwd,
			},
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *OrganizationService) List(ctx context.Context) ([]OrganizationResponse, error) {
	var nodes []tenant.OrgNode
	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).
		Where("parent_id IS NULL AND code <> ?", "platform-root").
		Order("created_at DESC").
		Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("查询机构列表失败: %w", err)
	}

	result := make([]OrganizationResponse, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, OrganizationResponse{
			ID:           node.ID,
			Name:         node.Name,
			Code:         node.Code,
			ContactName:  node.ContactName,
			ContactPhone: node.ContactPhone,
			Status:       node.Status,
		})
	}

	return result, nil
}

func (s *OrganizationService) GetByID(ctx context.Context, id string) (*OrganizationResponse, error) {
	var node tenant.OrgNode
	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ? AND parent_id IS NULL AND code <> ?", id, "platform-root").First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("查询机构详情失败: %w", err)
	}

	return &OrganizationResponse{
		ID:           node.ID,
		Name:         node.Name,
		Code:         node.Code,
		ContactName:  node.ContactName,
		ContactPhone: node.ContactPhone,
		Status:       node.Status,
	}, nil
}

func (s *OrganizationService) Suspend(ctx context.Context, id string) error {
	return s.updateStatus(ctx, id, tenant.StatusSuspended)
}

func (s *OrganizationService) Activate(ctx context.Context, id string) error {
	return s.updateStatus(ctx, id, tenant.StatusActive)
}

func (s *OrganizationService) updateStatus(ctx context.Context, id, status string) error {
	var node tenant.OrgNode
	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Where("id = ? AND parent_id IS NULL AND code <> ?", id, "platform-root").First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("查询机构失败: %w", err)
	}

	if err := s.db.WithContext(tenant.SkipTenantGuard(ctx)).Model(&tenant.OrgNode{}).Where("path LIKE ?", node.Path+"%").Update("status", status).Error; err != nil {
		return fmt.Errorf("更新机构状态失败: %w", err)
	}

	return nil
}

type OrganizationHandler struct {
	svc *OrganizationService
}

func NewOrganizationHandler(svc *OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{svc: svc}
}

func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, "查询机构列表失败")
		return
	}

	response.OK(w, items)
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateOrganizationInput
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

func (h *OrganizationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, item)
}

func (h *OrganizationHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Suspend(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "suspended"})
}

func (h *OrganizationHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Activate(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "active"})
}

func (h *OrganizationHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidOrganization):
		response.BadRequest(w, err.Error())
	case errors.Is(err, tenant.ErrCodeDuplicate):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrPlatformUserExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrOrganizationNotFound):
		response.NotFound(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}

func buildDefaultPassword(code string) string {
	normalized := strings.TrimSpace(code)
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(normalized, -1)
	b := strings.Builder{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(strings.ToLower(part[1:]))
		}
	}
	base := b.String()
	if base == "" {
		base = "Org"
	}
	return fmt.Sprintf("%s@%d", base, time.Now().Year())
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
