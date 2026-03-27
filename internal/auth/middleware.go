package auth

import (
	"context"
	"net/http"
	"strings"

	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"
)

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// AuthMiddleware JWT 认证中间件。
// 从 Authorization Header 提取 Bearer Token，验证后将 Claims 写入 Context。
// 同时将 OrgNodeID / OrgNodePath 写入 tenant Context，供后续中间件/handler 使用。
func AuthMiddleware(jwtMgr *JWTManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从 Header 提取 Token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "缺少 Authorization 头")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Unauthorized(w, "Authorization 格式错误，应为 Bearer <token>")
				return
			}

			tokenStr := parts[1]
			claims, err := jwtMgr.ParseToken(tokenStr)
			if err != nil {
				if err == ErrExpiredToken {
					response.Unauthorized(w, "Token 已过期")
				} else {
					response.Unauthorized(w, "无效的 Token")
				}
				return
			}

			// 将 Claims 写入 Context
			ctx := WithClaims(r.Context(), claims)

			// 将组织节点信息写入 tenant Context
			ctx = tenant.WithOrgNode(ctx, claims.OrgNodeID, claims.OrgNodePath)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission 权限检查中间件。
// 检查当前用户角色是否拥有指定权限，无权限返回 403。
func RequirePermission(permission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				response.Unauthorized(w, "未认证")
				return
			}

			if !HasPermission(claims.RoleName, permission) {
				response.Forbidden(w, "权限不足")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims 从 Context 中获取 JWT Claims。
func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey{}).(*Claims)
	return claims
}
