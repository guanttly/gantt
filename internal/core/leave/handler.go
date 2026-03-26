package leave

import (
"encoding/json"
"errors"
"net/http"
"strconv"

"gantt-saas/internal/common/response"

"github.com/go-chi/chi/v5"
)

// Handler 请假 HTTP 处理器。
type Handler struct {
svc *Service
}

// NewHandler 创建请假处理器。
func NewHandler(svc *Service) *Handler {
return &Handler{svc: svc}
}

// List 查询请假列表（分页 + 过滤）。
// GET /api/v1/leaves
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
size, _ := strconv.Atoi(r.URL.Query().Get("size"))

opts := ListOptions{
Page:       page,
Size:       size,
EmployeeID: r.URL.Query().Get("employee_id"),
Status:     r.URL.Query().Get("status"),
StartDate:  r.URL.Query().Get("start_date"),
EndDate:    r.URL.Query().Get("end_date"),
}

leaves, total, err := h.svc.List(r.Context(), opts)
if err != nil {
response.InternalError(w, "查询请假列表失败")
return
}

response.Page(w, leaves, total, opts.Page, opts.Size)
}

// Create 创建请假记录。
// POST /api/v1/leaves
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
var input CreateInput
if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
response.BadRequest(w, "请求参数格式错误")
return
}

if input.EmployeeID == "" || input.LeaveType == "" || input.StartDate == "" || input.EndDate == "" {
response.BadRequest(w, "employee_id、leave_type、start_date、end_date 为必填项")
return
}

l, err := h.svc.Create(r.Context(), input)
if err != nil {
h.handleError(w, err)
return
}

response.Created(w, l)
}

// Update 更新请假记录。
// PUT /api/v1/leaves/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

var input UpdateInput
if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
response.BadRequest(w, "请求参数格式错误")
return
}

l, err := h.svc.Update(r.Context(), id, input)
if err != nil {
h.handleError(w, err)
return
}

response.OK(w, l)
}

// Delete 删除请假记录。
// DELETE /api/v1/leaves/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

if err := h.svc.Delete(r.Context(), id); err != nil {
h.handleError(w, err)
return
}

response.NoContent(w)
}

// Approve 审批请假。
// PUT /api/v1/leaves/:id/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
id := chi.URLParam(r, "id")

var body struct {
Approved bool `json:"approved"`
}
if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
response.BadRequest(w, "请求参数格式错误")
return
}

l, err := h.svc.Approve(r.Context(), id, body.Approved)
if err != nil {
h.handleError(w, err)
return
}

response.OK(w, l)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
switch {
case errors.Is(err, ErrLeaveNotFound):
response.NotFound(w, err.Error())
case errors.Is(err, ErrLeaveNotPending):
response.BadRequest(w, err.Error())
default:
response.InternalError(w, "内部错误")
}
}
