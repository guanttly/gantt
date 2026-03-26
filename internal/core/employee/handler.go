package employee

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 员工 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建员工处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 查询员工列表（分页 + 搜索）。
// GET /api/v1/employees
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	opts := ListOptions{
		Page:     page,
		Size:     size,
		Keyword:  r.URL.Query().Get("keyword"),
		Status:   r.URL.Query().Get("status"),
		Position: r.URL.Query().Get("position"),
		Category: r.URL.Query().Get("category"),
	}

	employees, total, err := h.svc.List(r.Context(), opts)
	if err != nil {
		response.InternalError(w, "查询员工列表失败")
		return
	}

	response.Page(w, employees, total, opts.Page, opts.Size)
}

// Create 创建员工。
// POST /api/v1/employees
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

	emp, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, emp)
}

// GetByID 获取员工详情。
// GET /api/v1/employees/:id
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	emp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, emp)
}

// Update 更新员工。
// PUT /api/v1/employees/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	emp, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, emp)
}

// Delete 删除员工。
// DELETE /api/v1/employees/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmployeeNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrEmployeeNoDup):
		response.Conflict(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
