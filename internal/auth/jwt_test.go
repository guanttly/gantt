package auth

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAndParse(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		Issuer:          "test",
	})

	token, err := mgr.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RoleScheduler))
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.OrgNodeID != "node-1" {
		t.Errorf("OrgNodeID = %q, want %q", claims.OrgNodeID, "node-1")
	}
	if claims.OrgNodePath != "/org/node-1" {
		t.Errorf("OrgNodePath = %q, want %q", claims.OrgNodePath, "/org/node-1")
	}
	if claims.RoleName != string(RoleScheduler) {
		t.Errorf("RoleName = %q, want %q", claims.RoleName, string(RoleScheduler))
	}
	if claims.Issuer != "test" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "test")
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "test-secret"})

	_, err := mgr.ParseToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("ParseToken(invalid) error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestJWTManager_WrongSecret(t *testing.T) {
	mgr1 := NewJWTManager(JWTConfig{Secret: "secret-1", AccessTokenTTL: time.Hour})
	mgr2 := NewJWTManager(JWTConfig{Secret: "secret-2"})

	token, _ := mgr1.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RolePlatformAdmin))
	_, err := mgr2.ParseToken(token)
	if err != ErrInvalidToken {
		t.Errorf("ParseToken(wrong secret) error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{
		Secret:         "test-secret",
		AccessTokenTTL: -1 * time.Second,
	})

	token, _ := mgr.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RolePlatformAdmin))
	_, err := mgr.ParseToken(token)
	if err != ErrExpiredToken {
		t.Errorf("ParseToken(expired) error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestJWTManager_RefreshToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})

	token, err := mgr.GenerateRefreshToken("user-1", "node-1", "/org/node-1", string(RoleScheduler))
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken(refresh) error = %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
}

func TestJWTManager_DefaultConfig(t *testing.T) {
	cfg := DefaultJWTConfig()
	if cfg.Secret == "" {
		t.Error("default secret should not be empty")
	}
	if cfg.AccessTokenTTL != 2*time.Hour {
		t.Errorf("default AccessTokenTTL = %v, want 2h", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("default RefreshTokenTTL = %v, want 168h", cfg.RefreshTokenTTL)
	}
}

func TestJWTManager_EmptySecretUsesDefault(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "", AccessTokenTTL: time.Hour})
	token, err := mgr.GenerateAccessToken("user-1", "node-1", "/org/node-1", string(RolePlatformAdmin))
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
}

func TestAccessTokenTTLSeconds(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{Secret: "s", AccessTokenTTL: 2 * time.Hour})
	if got := mgr.AccessTokenTTLSeconds(); got != 7200 {
		t.Errorf("AccessTokenTTLSeconds() = %d, want 7200", got)
	}
}
