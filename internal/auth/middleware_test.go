package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test"})
	handler := AuthMiddleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test"})
	handler := AuthMiddleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test"})
	handler := AuthMiddleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test", AccessTokenTTL: -1 * time.Second})
	token, _ := mgr.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RolePlatformAdmin))

	handler := AuthMiddleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test", AccessTokenTTL: time.Hour})
	token, _ := mgr.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RoleScheduler))

	var capturedClaims *Claims
	handler := AuthMiddleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = GetClaims(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedClaims == nil {
		t.Fatal("claims should not be nil")
	}
	if capturedClaims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", capturedClaims.UserID, "user-1")
	}
	if capturedClaims.OrgNodeID != "node-1" {
		t.Errorf("OrgNodeID = %q, want %q", capturedClaims.OrgNodeID, "node-1")
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	handler := RequirePermission("schedule:create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := &Claims{RoleName: string(RolePlatformAdmin)}
	ctx := context.WithValue(context.Background(), claimsContextKey{}, claims)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	handler := RequirePermission("schedule:create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := &Claims{RoleName: string(RoleEmployee)}
	ctx := context.WithValue(context.Background(), claimsContextKey{}, claims)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequirePermission_NoClaims(t *testing.T) {
	handler := RequirePermission("schedule:create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetClaims_Nil(t *testing.T) {
	ctx := context.Background()
	claims := GetClaims(ctx)
	if claims != nil {
		t.Error("GetClaims() on empty context should return nil")
	}
}
