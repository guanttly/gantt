package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"gantt-saas/internal/common/response"
)

type AppHandler struct {
	svc *AppService
}

func NewAppHandler(svc *AppService) *AppHandler {
	return &AppHandler{svc: svc}
}

func (h *AppHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input AppLoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	if input.LoginID == "" || input.Password == "" {
		response.BadRequest(w, "login_id、password 为必填项")
		return
	}
	result, err := h.svc.Login(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, result)
}

func (h *AppHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
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

func (h *AppHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

func (h *AppHandler) ForceResetPassword(w http.ResponseWriter, r *http.Request) {
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

func (h *AppHandler) GetMe(w http.ResponseWriter, r *http.Request) {
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

func (h *AppHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrInvalidToken):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrExpiredToken):
		response.Unauthorized(w, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrNoNodePermission):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrAppLoginIDAmbiguous):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrEmployeeInactive):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrWeakPassword):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrOldPasswordMismatch):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrNotRequireReset):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
