package auth

import (
"errors"
"time"

"github.com/golang-jwt/jwt/v5"
)

var (
ErrInvalidToken = errors.New("无效的 Token")
ErrExpiredToken = errors.New("Token 已过期")
)

// Claims JWT 载荷。
type Claims struct {
jwt.RegisteredClaims
UserID      string `json:"uid"`
OrgNodeID   string `json:"nid"`
OrgNodePath string `json:"npath"`
RoleName    string `json:"role"`
}

// JWTConfig JWT 配置。
type JWTConfig struct {
Secret          string
AccessTokenTTL  time.Duration
RefreshTokenTTL time.Duration
Issuer          string
}

// DefaultJWTConfig 返回默认 JWT 配置。
func DefaultJWTConfig() JWTConfig {
return JWTConfig{
Secret:          "gantt-saas-secret-change-me",
AccessTokenTTL:  2 * time.Hour,
RefreshTokenTTL: 7 * 24 * time.Hour,
Issuer:          "gantt-saas",
}
}

// JWTManager JWT 签发与验证管理器。
type JWTManager struct {
config JWTConfig
}

// NewJWTManager 创建 JWT 管理器。
func NewJWTManager(cfg JWTConfig) *JWTManager {
if cfg.Secret == "" {
cfg.Secret = DefaultJWTConfig().Secret
}
return &JWTManager{config: cfg}
}

// GenerateAccessToken 签发 Access Token。
func (m *JWTManager) GenerateAccessToken(userID, orgNodeID, orgNodePath, roleName string) (string, error) {
return m.generateToken(userID, orgNodeID, orgNodePath, roleName, m.config.AccessTokenTTL)
}

// GenerateRefreshToken 签发 Refresh Token。
func (m *JWTManager) GenerateRefreshToken(userID, orgNodeID, orgNodePath, roleName string) (string, error) {
return m.generateToken(userID, orgNodeID, orgNodePath, roleName, m.config.RefreshTokenTTL)
}

func (m *JWTManager) generateToken(userID, orgNodeID, orgNodePath, roleName string, ttl time.Duration) (string, error) {
now := time.Now()
claims := Claims{
RegisteredClaims: jwt.RegisteredClaims{
Issuer:    m.config.Issuer,
IssuedAt:  jwt.NewNumericDate(now),
ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
},
UserID:      userID,
OrgNodeID:   orgNodeID,
OrgNodePath: orgNodePath,
RoleName:    roleName,
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
return token.SignedString([]byte(m.config.Secret))
}

// ParseToken 解析并验证 JWT。
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
return nil, ErrInvalidToken
}
return []byte(m.config.Secret), nil
})
if err != nil {
if errors.Is(err, jwt.ErrTokenExpired) {
return nil, ErrExpiredToken
}
return nil, ErrInvalidToken
}

claims, ok := token.Claims.(*Claims)
if !ok || !token.Valid {
return nil, ErrInvalidToken
}

return claims, nil
}

// AccessTokenTTLSeconds 返回 Access Token 有效期（秒）。
func (m *JWTManager) AccessTokenTTLSeconds() int {
return int(m.config.AccessTokenTTL.Seconds())
}
