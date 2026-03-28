package shift

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 班次 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建班次处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 查询班次列表。
// GET /api/v1/shifts
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	shifts, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, "查询班次列表失败")
		return
	}
	response.OK(w, shifts)
}

// GetByID 获取班次详情。
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sh, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, sh)
}

// ListAvailable 查询排班应用可用班次列表。
// GET /api/v1/app/scheduling/ref/shifts
func (h *Handler) ListAvailable(w http.ResponseWriter, r *http.Request) {
	shifts, err := h.svc.ListAvailable(r.Context())
	if err != nil {
		response.InternalError(w, "查询可用班次列表失败")
		return
	}
	response.OK(w, shifts)
}

// Create 创建班次。
// POST /api/v1/shifts
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Name == "" || input.Code == "" || input.StartTime == "" || input.EndTime == "" {
		response.BadRequest(w, "name、code、start_time、end_time 为必填项")
		return
	}

	sh, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, sh)
}

// Update 更新班次。
// PUT /api/v1/shifts/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	sh, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, sh)
}

// ToggleStatus 启用/停用班次。
// PUT /api/v1/shifts/:id/toggle
func (h *Handler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input struct {
		IsActive *bool `json:"is_active,omitempty"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.BadRequest(w, "请求参数格式错误")
			return
		}
	}

	sh, err := h.svc.ToggleStatus(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	if input.IsActive != nil && sh.IsActive != *input.IsActive {
		sh, err = h.svc.ToggleStatus(r.Context(), id)
		if err != nil {
			h.handleError(w, err)
			return
		}
	}

	response.OK(w, sh)
}

// Delete 删除班次。
// DELETE /api/v1/shifts/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

// GetDependencies 查询依赖关系。
// GET /api/v1/shifts/dependencies
func (h *Handler) GetDependencies(w http.ResponseWriter, r *http.Request) {
	deps, err := h.svc.GetDependencies(r.Context())
	if err != nil {
		response.InternalError(w, "查询班次依赖失败")
		return
	}
	response.OK(w, deps)
}

// AddDependency 配置依赖关系。
// POST /api/v1/shifts/dependencies
func (h *Handler) AddDependency(w http.ResponseWriter, r *http.Request) {
	var input DependencyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.ShiftID == "" || input.DependsOnID == "" || input.DependencyType == "" {
		response.BadRequest(w, "shift_id、depends_on_id、dependency_type 为必填项")
		return
	}

	dep, err := h.svc.AddDependency(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, dep)
}

func (h *Handler) GetShiftGroups(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.GetShiftGroups(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) SetShiftGroups(w http.ResponseWriter, r *http.Request) {
	var input ShiftGroupSetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	items, err := h.svc.SetShiftGroups(r.Context(), chi.URLParam(r, "id"), input.GroupIDs)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) AddGroupToShift(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Priority int `json:"priority"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.BadRequest(w, "请求参数格式错误")
			return
		}
	}
	item, err := h.svc.AddGroupToShift(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "groupId"), input.Priority)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.Created(w, item)
}

func (h *Handler) RemoveGroupFromShift(w http.ResponseWriter, r *http.Request) {
	err := h.svc.RemoveGroupFromShift(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "groupId"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) GetFixedAssignments(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.GetFixedAssignments(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) SaveFixedAssignments(w http.ResponseWriter, r *http.Request) {
	var input FixedAssignmentBatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	items, err := h.svc.SaveFixedAssignments(r.Context(), chi.URLParam(r, "id"), input.Assignments)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, items)
}

func (h *Handler) DeleteFixedAssignment(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteFixedAssignment(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "assignmentId"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CalculateFixedSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	preview, err := h.svc.CalculateFixedSchedule(r.Context(), chi.URLParam(r, "id"), input.StartDate, input.EndDate)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, preview)
}

func (h *Handler) GetWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.svc.GetWeeklyStaffConfig(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, config)
}

func (h *Handler) UpdateWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	var input WeeklyStaffConfig
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	config, err := h.svc.UpdateWeeklyStaffConfig(r.Context(), chi.URLParam(r, "id"), input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, config)
}

func (h *Handler) BatchGetWeeklyStaffConfig(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("shift_ids"))
	shiftIDs := []string{}
	if raw != "" {
		shiftIDs = strings.Split(raw, ",")
	}
	configs, err := h.svc.BatchGetWeeklyStaffConfig(r.Context(), shiftIDs)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, configs)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotDeptNode):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrShiftNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrShiftCodeDup):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrCyclicDep):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrSelfDep):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidDepType):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
