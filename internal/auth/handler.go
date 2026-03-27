package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"gantt-saas/internal/common/response"
)

// Handler 认证 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建认证处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register 用户注册。
// POST /api/v1/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		response.BadRequest(w, "username、email、password 为必填项")
		return
	}

	result, err := h.svc.Register(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, result)
}

// Login 用户登录。
// POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Username == "" || input.Password == "" {
		response.BadRequest(w, "username、password 为必填项")
		return
	}

	result, err := h.svc.Login(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// AdminLogin 后台账号登录。
// POST /api/v1/admin/auth/login
func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Username == "" || input.Password == "" {
		response.BadRequest(w, "username、password 为必填项")
		return
	}

	result, err := h.svc.AdminLogin(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// RefreshToken 刷新 Token。
// POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if body.RefreshToken == "" {
		response.BadRequest(w, "refresh_token 为必填项")
		return
	}

	result, err := h.svc.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// SwitchNode 切换组织节点。
// POST /api/v1/auth/switch-node（需认证）
func (h *Handler) SwitchNode(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		response.Unauthorized(w, "未认证")
		return
	}

	var input SwitchNodeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.OrgNodeID == "" {
		response.BadRequest(w, "org_node_id 为必填项")
		return
	}

	result, err := h.svc.SwitchNode(r.Context(), claims, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// ResetPassword 重置密码。
// POST /api/v1/auth/password/reset（需认证）
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		response.Unauthorized(w, "未认证")
		return
	}

	var input ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.OldPassword == "" || input.NewPassword == "" {
		response.BadRequest(w, "old_password、new_password 为必填项")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), claims.UserID, input); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "password_updated"})
}

// ForceResetPassword 强制重置密码（仅首次登录时使用）。
// POST /api/v1/auth/password/force-reset（需认证）
func (h *Handler) ForceResetPassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		response.Unauthorized(w, "未认证")
		return
	}

	var input ForceResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.NewPassword == "" {
		response.BadRequest(w, "new_password 为必填项")
		return
	}

	if err := h.svc.ForceResetPassword(r.Context(), claims.UserID, input); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "password_reset_success"})
}

// GetMe 获取当前用户信息。
// GET /api/v1/auth/me（需认证）
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		response.Unauthorized(w, "未认证")
		return
	}

	result, err := h.svc.GetMe(r.Context(), claims)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// AssignRole 分配角色。
// POST /api/v1/admin/auth/assign-role（需认证+权限）
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var input AssignRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.UserID == "" || input.OrgNodeID == "" || input.RoleName == "" {
		response.BadRequest(w, "user_id、org_node_id、role_name 为必填项")
		return
	}

	result, err := h.svc.AssignRole(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, result)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrInvalidToken):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrExpiredToken):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrUserDisabled):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrAdminLoginRequired):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrAccountLocked):
		response.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", err.Error())
	case errors.Is(err, ErrNoNodePermission):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrUsernameExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrEmailExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrNodeRoleExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrPublicRegisterRole):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrWeakPassword):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrRoleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrOldPasswordMismatch):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrNotRequireReset):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
