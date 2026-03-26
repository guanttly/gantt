package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		RegisterPublicRoutes(r, h)

		r.Group(func(r chi.Router) {
			RegisterProtectedRoutes(r, h)
			RegisterAdminRoutes(r, h)
		})
	})

	// Walk routes 不 panic
	_ = chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		return nil
	})
}

func TestHandler_Login_BadRequest(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"username":""}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_Register_BadRequest(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"username":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_RefreshToken_MissingField(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.RefreshToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_SwitchNode_NoClaims(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{"org_node_id":"node-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.SwitchNode(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_ResetPassword_NoClaims(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.ResetPassword(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandler_GetMe_NoClaims(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.GetMe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleError_AllCases(t *testing.T) {
	h := &Handler{}
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"InvalidCredentials", ErrInvalidCredentials, http.StatusUnauthorized},
		{"InvalidToken", ErrInvalidToken, http.StatusUnauthorized},
		{"ExpiredToken", ErrExpiredToken, http.StatusUnauthorized},
		{"UserNotFound", ErrUserNotFound, http.StatusNotFound},
		{"UserDisabled", ErrUserDisabled, http.StatusForbidden},
		{"AccountLocked", ErrAccountLocked, http.StatusTooManyRequests},
		{"NoNodePermission", ErrNoNodePermission, http.StatusForbidden},
		{"UsernameExists", ErrUsernameExists, http.StatusConflict},
		{"EmailExists", ErrEmailExists, http.StatusConflict},
		{"NodeRoleExists", ErrNodeRoleExists, http.StatusConflict},
		{"WeakPassword", ErrWeakPassword, http.StatusBadRequest},
		{"RoleNotFound", ErrRoleNotFound, http.StatusNotFound},
		{"OldPasswordMismatch", ErrOldPasswordMismatch, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.handleError(w, tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("handleError(%v) status = %d, want %d", tt.err, w.Code, tt.wantStatus)
			}
		})
	}
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"Abc12345", true},
		{"StrongPass1", true},
		{"abcdefgh", false}, // 无大写和数字
		{"ABCDEFGH", false}, // 无小写和数字
		{"12345678", false}, // 无字母
		{"Abc1", false},     // 太短
		{"abcABC12", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := isStrongPassword(tt.password); got != tt.valid {
				t.Errorf("isStrongPassword(%q) = %v, want %v", tt.password, got, tt.valid)
			}
		})
	}
}

func TestUserToInfo(t *testing.T) {
	phone := "+8613800000000"
	user := &User{
		ID:       "user-1",
		Username: "zhangsan",
		Email:    "zs@example.com",
		Phone:    &phone,
	}

	info := user.ToInfo()
	if info.ID != "user-1" {
		t.Errorf("ID = %q, want %q", info.ID, "user-1")
	}
	if info.Username != "zhangsan" {
		t.Errorf("Username = %q, want %q", info.Username, "zhangsan")
	}
	if info.Email != "zs@example.com" {
		t.Errorf("Email = %q, want %q", info.Email, "zs@example.com")
	}
	if info.Phone == nil || *info.Phone != phone {
		t.Errorf("Phone = %v, want %q", info.Phone, phone)
	}
}

func TestJSONArray_ScanValue(t *testing.T) {
	perms := JSONArray{"employee:read", "schedule:*"}

	// Value
	val, err := perms.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	str, ok := val.(string)
	if !ok {
		t.Fatalf("Value() type = %T, want string", val)
	}

	// Scan back
	var perms2 JSONArray
	if err := perms2.Scan([]byte(str)); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(perms2) != 2 || perms2[0] != "employee:read" || perms2[1] != "schedule:*" {
		t.Errorf("Scan result = %v, want %v", perms2, perms)
	}
}

func TestJSONArray_ScanNil(t *testing.T) {
	var perms JSONArray
	if err := perms.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error = %v", err)
	}
	if perms != nil {
		t.Errorf("Scan(nil) = %v, want nil", perms)
	}
}

func TestJSONArray_ValueNil(t *testing.T) {
	var perms JSONArray
	val, err := perms.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if val != "[]" {
		t.Errorf("Value() = %v, want %q", val, "[]")
	}
}

func TestAssignRoleInput_JSON(t *testing.T) {
	input := AssignRoleInput{
		UserID:    "user-1",
		OrgNodeID: "node-1",
		RoleName:  string(RoleScheduler),
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	var out AssignRoleInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if out.UserID != input.UserID || out.OrgNodeID != input.OrgNodeID || out.RoleName != input.RoleName {
		t.Errorf("roundtrip mismatch: %+v != %+v", out, input)
	}
}
