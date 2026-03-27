package tenant

import (
	"encoding/json"
	"errors"
	"net/http"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// Handler 组织树 HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 创建组织树处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetTree 获取组织树。
// GET /api/v1/admin/org-nodes?root_id=xxx
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	rootID := r.URL.Query().Get("root_id")

	if rootID != "" {
		tree, err := h.svc.GetTree(r.Context(), rootID)
		if err != nil {
			h.handleError(w, err)
			return
		}
		response.OK(w, tree)
		return
	}

	// 无 root_id 返回所有顶级节点及其完整子树
	nodes, err := h.svc.GetRootTrees(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.OK(w, nodes)
}

// Create 创建组织节点。
// POST /api/v1/admin/org-nodes
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateNodeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.Name == "" || input.Code == "" || input.NodeType == "" {
		response.BadRequest(w, "name、code、node_type 为必填项")
		return
	}

	node, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.Created(w, node)
}

// Update 更新组织节点。
// PUT /api/v1/admin/org-nodes/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input UpdateNodeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	node, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, node)
}

// Suspend 停用组织节点。
// PUT /api/v1/admin/org-nodes/:id/suspend
func (h *Handler) Suspend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Suspend(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "suspended"})
}

// Activate 启用组织节点。
// PUT /api/v1/admin/org-nodes/:id/activate
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Activate(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, map[string]string{"status": "active"})
}

// GetChildren 获取子节点列表。
// GET /api/v1/admin/org-nodes/:id/children
func (h *Handler) GetChildren(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	children, err := h.svc.GetChildren(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, children)
}

// Delete 删除组织节点。
// DELETE /api/v1/admin/org-nodes/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	response.NoContent(w)
}

// Move 移动组织节点到新的父节点。
// PUT /api/v1/admin/org-nodes/:id/move
func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input MoveNodeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	if input.NewParentID == "" {
		response.BadRequest(w, "new_parent_id 为必填项")
		return
	}

	node, err := h.svc.Move(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, node)
}

// GetByID 获取单个节点详情。
// GET /api/v1/admin/org-nodes/:id
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	node, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.OK(w, node)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNodeNotFound):
		response.NotFound(w, err.Error())
	case errors.Is(err, ErrNodeSuspended):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCodeDuplicate):
		response.Conflict(w, err.Error())
	case errors.Is(err, ErrInvalidNodeType):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidRootType):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrInvalidHierarchy):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrCannotDeleteRoot):
		response.BadRequest(w, err.Error())
	case errors.Is(err, ErrProtectedNode):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrManageScopeDenied):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrCannotDeleteCurrentScopeRoot):
		response.Forbidden(w, err.Error())
	case errors.Is(err, ErrCannotMoveCurrentScopeRoot):
		response.Forbidden(w, err.Error())
	default:
		response.InternalError(w, "内部错误")
	}
}
