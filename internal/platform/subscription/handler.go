package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 订阅 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建订阅处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 查询订阅列表。
// GET /api/v1/admin/subscriptions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	opts := ListOptions{
		Page:   page,
		Size:   size,
		Plan:   r.URL.Query().Get("plan"),
		Status: r.URL.Query().Get("status"),
	}

	subs, total, err := h.svc.List(r.Context(), opts)
	if err != nil {
		response.InternalError(w, "查询订阅列表失败")
		return
	}

	response.Page(w, subs, total, opts.Page, opts.Size)
}

// GetByID 获取订阅详情。
// GET /api/v1/admin/subscriptions/:id
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, sub)
}

// Create 创建订阅。
// POST /api/v1/admin/subscriptions
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.OrgNodeID == "" {
		response.BadRequest(w, "org_node_id 为必填项")
		return
	}

	sub, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, sub)
}

// Update 更新订阅。
// PUT /api/v1/admin/subscriptions/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	sub, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, sub)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSubscriptionNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrSubscriptionAlreadyExists):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrInvalidPlan):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrQuotaExceeded):
		response.Error(w, http.StatusPaymentRequired, "QUOTA_EXCEEDED", err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
