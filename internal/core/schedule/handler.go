package schedule

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gantt-saas/internal/common/response"
	"gantt-saas/internal/core/approle"
	"gantt-saas/internal/core/schedule/step"

	"github.com/go-chi/chi/v5"
)

// Handler 排班 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建排班处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create 创建排班计划。
// POST /api/v1/schedules
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
	if input.StartDate == "" || input.EndDate == "" {
		response.BadRequest(w, "start_date 和 end_date 为必填项")
		return
	}

	sch, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, sch)
}

// GetByID 查看排班计划详情。
// GET /api/v1/schedules/:id
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sch, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, sch)
}

// List 查询排班计划列表。
// GET /api/v1/schedules
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	opts := ScheduleListOptions{
		Page:      parseIntParam(r, "page", 1),
		Size:      parseIntParam(r, "size", 20),
		Status:    r.URL.Query().Get("status"),
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
	}

	schedules, total, err := h.svc.List(r.Context(), opts)
	if err != nil {
		response.InternalError(w, "查询排班计划列表失败")
		return
	}

	response.Page(w, schedules, total, opts.Page, opts.Size)
}

// Generate 触发排班生成。
// POST /api/v1/schedules/:id/generate
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.svc.Generate(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// GetAssignments 查看排班结果。
// GET /api/v1/schedules/:id/assignments
func (h *Handler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	assignments, err := h.svc.GetAssignments(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, assignments)
}

// GetSelfAssignments 查看当前员工在日期范围内的已发布排班。
// GET /api/v1/scheduling/assignments/self
func (h *Handler) GetSelfAssignments(w http.ResponseWriter, r *http.Request) {
	employeeID := approle.CurrentEmployeeID(r.Context())
	if employeeID == "" {
		response.Forbidden(w, "权限不足")
		return
	}

	assignments, err := h.svc.GetSelfAssignments(
		r.Context(),
		employeeID,
		r.URL.Query().Get("start_date"),
		r.URL.Query().Get("end_date"),
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, assignments)
}

// AdjustAssignments 手动调整排班。
// PUT /api/v1/schedules/:id/assignments
func (h *Handler) AdjustAssignments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input step.EditInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	result, err := h.svc.AdjustAssignments(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// Validate 手动触发全规则校验。
// POST /api/v1/schedules/:id/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.svc.Validate(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, result)
}

// Publish 发布排班。
// POST /api/v1/schedules/:id/publish
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Publish(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "published"})
}

// GetChanges 查看变更记录。
// GET /api/v1/schedules/:id/changes
func (h *Handler) GetChanges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	opts := ChangeListOptions{
		Page: parseIntParam(r, "page", 1),
		Size: parseIntParam(r, "size", 20),
	}

	changes, total, err := h.svc.GetChanges(r.Context(), id, opts)
	if err != nil {
		response.InternalError(w, "查询变更记录失败")
		return
	}

	response.Page(w, changes, total, opts.Page, opts.Size)
}

// GetSummary 获取排班统计汇总。
// GET /api/v1/schedules/:id/summary
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	summary, err := h.svc.GetSummary(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, summary)
}

// Delete 删除排班计划。
// DELETE /api/v1/schedules/:id
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
	case errors.Is(err, ErrScheduleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrAssignmentNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrInvalidDateRange):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidStatus):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCannotGenerate):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCannotPublish):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCannotAdjust):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
