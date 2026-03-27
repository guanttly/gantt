package group

import (
	"encoding/json"
	"errors"
	"net/http"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 分组 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建分组处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 查询分组列表。
// GET /api/v1/groups
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, "查询分组列表失败")
		return
	}
	response.OK(w, groups)
}

// Create 创建分组。
// POST /api/v1/groups
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Name == "" {
		response.BadRequest(w, "name 为必填项")
		return
	}

	g, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, g)
}

// Update 更新分组。
// PUT /api/v1/groups/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	g, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, g)
}

// Delete 删除分组。
// DELETE /api/v1/groups/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

// GetMembers 获取分组成员列表。
// GET /api/v1/groups/:id/members
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	members, err := h.svc.GetMembers(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, members)
}

// AddMember 添加成员到分组。
// POST /api/v1/groups/:id/members
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")

	var body struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if body.EmployeeID == "" {
		response.BadRequest(w, "employee_id 为必填项")
		return
	}

	actorID := ""
	if claims := auth.GetClaims(r.Context()); claims != nil {
		actorID = claims.UserID
	}

	m, err := h.svc.AddMember(r.Context(), groupID, body.EmployeeID, actorID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, m)
}

// RemoveMember 从分组中移除成员。
// DELETE /api/v1/groups/:id/members/:eid
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	employeeID := chi.URLParam(r, "eid")

	if err := h.svc.RemoveMember(r.Context(), groupID, employeeID); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotDeptNode):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrGroupNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrMemberExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrMemberNotFound):
		response.NotFound(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
