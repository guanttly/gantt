package audit

import (
	"net/http"
	"strconv"
	"time"

	"gantt-saas/internal/common/response"
)

// Handler 审计日志查询 HTTP 处理器。
type Handler struct {
	repo *Repository
}

// NewHandler 创建审计日志处理器。
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// List 查询审计日志列表。
// GET /api/v1/admin/audit-logs
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	opts := ListOptions{
		Page:         page,
		Size:         size,
		OrgNodeID:    r.URL.Query().Get("org_node_id"),
		UserID:       r.URL.Query().Get("user_id"),
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
	}

	// 解析时间范围
	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			opts.StartTime = t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			// 包含当天结束时间
			opts.EndTime = t.Add(24*time.Hour - time.Second)
		}
	}

	logs, total, err := h.repo.List(r.Context(), opts)
	if err != nil {
		response.InternalError(w, "查询审计日志失败")
		return
	}

	response.Page(w, logs, total, opts.Page, opts.Size)
}
