package approle

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListEmployeeRoles(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListEmployeeRoles(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) ListEmployeeRolesBatch(w http.ResponseWriter, r *http.Request) {
	rawIDs := strings.TrimSpace(r.URL.Query().Get("employee_ids"))
	if rawIDs == "" {
		response.OK(w, map[string][]EmployeeAppRoleResponse{})
		return
	}
	parts := strings.Split(rawIDs, ",")
	employeeIDs := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		employeeIDs = append(employeeIDs, id)
	}
	items, err := h.svc.ListEmployeeRolesBatch(r.Context(), employeeIDs)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) AssignEmployeeRole(w http.ResponseWriter, r *http.Request) {
	var input AssignEmployeeRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	actorID := h.currentUserID(r)
	item, err := h.svc.AssignEmployeeRole(r.Context(), chi.URLParam(r, "id"), input, actorID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.Created(w, item)
}

func (h *Handler) RemoveEmployeeRole(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveEmployeeRole(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "roleId")); err != nil {
		h.handleError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) BatchAssignEmployeeRoles(w http.ResponseWriter, r *http.Request) {
	var input BatchAssignEmployeeRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	result, err := h.svc.BatchAssignEmployeeRoles(r.Context(), input, h.currentUserID(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, result)
}

func (h *Handler) ListGroupDefaultRoles(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListGroupDefaultRoles(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) AssignGroupDefaultRole(w http.ResponseWriter, r *http.Request) {
	var input AssignGroupDefaultRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	item, err := h.svc.AssignGroupDefaultRole(r.Context(), chi.URLParam(r, "id"), input, h.currentUserID(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.Created(w, item)
}

func (h *Handler) RemoveGroupDefaultRole(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveGroupDefaultRole(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "roleId")); err != nil {
		h.handleError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Summary(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) Expiring(w http.ResponseWriter, r *http.Request) {
	withinDays, _ := strconv.Atoi(r.URL.Query().Get("within_days"))
	items, err := h.svc.Expiring(r.Context(), withinDays)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) MyRoles(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.MyRoles(r.Context(), h.currentUserID(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) MyPermissions(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.MyPermissions(r.Context(), h.currentUserID(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) currentUserID(r *http.Request) string {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		return ""
	}
	return claims.UserID
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmployeeNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrGroupNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrInvalidAppRole):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrAppRoleExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrAppRoleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrDefaultGroupRoleExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrDefaultGroupRoleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrNodeOutOfScope):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrRoleNodeMismatch):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidGrantedBy):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrEmployeeBindingRequired):
		response.Forbidden(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
