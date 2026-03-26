package rule

import (
	"encoding/json"
	"errors"
	"net/http"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 规则 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建规则处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 查询规则列表（含继承标记）。
// GET /api/v1/rules
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, "查询规则列表失败")
		return
	}
	response.OK(w, rules)
}

// GetEffective 计算最终生效规则集。
// GET /api/v1/rules/effective
func (h *Handler) GetEffective(w http.ResponseWriter, r *http.Request) {
	effective, err := h.svc.ListEffective(r.Context())
	if err != nil {
		response.InternalError(w, "计算生效规则失败")
		return
	}
	response.OK(w, effective)
}

// Create 创建规则。
// POST /api/v1/rules
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Name == "" || input.Category == "" || input.SubType == "" {
		response.BadRequest(w, "name、category、sub_type 为必填项")
		return
	}

	if len(input.Config) == 0 {
		response.BadRequest(w, "config 为必填项")
		return
	}

	rl, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, rl)
}

// Update 更新规则。
// PUT /api/v1/rules/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	rl, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, rl)
}

// Delete 删除规则。
// DELETE /api/v1/rules/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

// GetByID 获取规则详情。
// GET /api/v1/rules/:id
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	rl, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, rl)
}

// Validate 校验规则是否冲突。
// POST /api/v1/rules/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	// 阶段二完成后与排班管道集成
	response.OK(w, map[string]string{"status": "not_implemented"})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRuleNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrInvalidCategory):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidSubType):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidConfig):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrRuleNameDup):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrOverrideNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrCannotOverride):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
