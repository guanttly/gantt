package tenant

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gantt-saas/internal/common/response"

	"github.com/go-chi/chi/v5"
)

// mockHandler 创建一个用于测试路由注册的 chi.Mux。
func setupTestRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		RegisterRoutes(r, h)
	})
	return r
}

func TestRegisterRoutes(t *testing.T) {
	// 仅验证路由注册不 panic
	svc := &Service{}
	h := NewHandler(svc)
	r := setupTestRouter(h)

	// 验证路由存在（Walk 不 panic 即可）
	_ = chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		return nil
	})
}

func TestHandleError_NotFound(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrNodeNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleError_Suspended(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrNodeSuspended)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleError_CodeDuplicate(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrCodeDuplicate)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleError_InvalidNodeType(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrInvalidNodeType)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleError_InvalidRootType(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrInvalidRootType)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleError_CannotDeleteRoot(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrCannotDeleteRoot)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleError_ProtectedNode(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()

	h.handleError(w, ErrProtectedNode)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCreate_BadRequest_MissingFields(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreate_BadRequest_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestResponseFormat(t *testing.T) {
	w := httptest.NewRecorder()
	response.OK(w, map[string]string{"key": "value"})

	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Data == nil {
		t.Error("response data should not be nil")
	}
}
