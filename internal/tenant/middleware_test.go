package tenant

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_MissingOrgNodeID(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_FromHeader(t *testing.T) {
	var capturedNodeID, capturedNodePath string
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedNodeID = GetOrgNodeID(r.Context())
		capturedNodePath = GetOrgNodePath(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Node-ID", "node-123")
	req.Header.Set("X-Org-Node-Path", "/org1/node-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedNodeID != "node-123" {
		t.Errorf("nodeID = %q, want %q", capturedNodeID, "node-123")
	}
	if capturedNodePath != "/org1/node-123" {
		t.Errorf("nodePath = %q, want %q", capturedNodePath, "/org1/node-123")
	}
}

func TestMiddleware_ScopeTree(t *testing.T) {
	var scopeTree bool
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopeTree = IsScopeTree(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?scope=tree", nil)
	req.Header.Set("X-Org-Node-ID", "node-123")
	req.Header.Set("X-Org-Node-Path", "/org1/node-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !scopeTree {
		t.Error("IsScopeTree() should be true when ?scope=tree")
	}
}

func TestMiddleware_NoScopeTree(t *testing.T) {
	var scopeTree bool
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopeTree = IsScopeTree(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Node-ID", "node-123")
	req.Header.Set("X-Org-Node-Path", "/org1/node-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if scopeTree {
		t.Error("IsScopeTree() should be false without ?scope=tree")
	}
}

func TestMiddleware_FromContext(t *testing.T) {
	var capturedNodeID string
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedNodeID = GetOrgNodeID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 模拟 auth 中间件已在 Context 中设置 org_node_id
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := WithOrgNode(req.Context(), "node-from-ctx", "/org1/node-from-ctx")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedNodeID != "node-from-ctx" {
		t.Errorf("nodeID = %q, want %q", capturedNodeID, "node-from-ctx")
	}
}
