package approle

import (
	"context"
	"errors"
	"net/http"

	"gantt-saas/internal/auth"
	"gantt-saas/internal/common/response"
)

type permissionsContextKey struct{}
type employeeContextKey struct{}

func CurrentPermissions(ctx context.Context) []string {
	permissions, _ := ctx.Value(permissionsContextKey{}).([]string)
	return permissions
}

func CurrentEmployeeID(ctx context.Context) string {
	employeeID, _ := ctx.Value(employeeContextKey{}).(string)
	return employeeID
}

func RequireAnyPermission(svc *Service, required ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.GetClaims(r.Context())
			if claims == nil {
				response.Unauthorized(w, "未认证")
				return
			}

			result, err := svc.MyPermissions(r.Context(), claims.UserID)
			if err != nil {
				switch {
				case errors.Is(err, ErrEmployeeBindingRequired), errors.Is(err, ErrEmployeeNotFound), errors.Is(err, ErrNodeOutOfScope):
					response.Forbidden(w, "权限不足")
				default:
					response.InternalError(w, "内部错误")
				}
				return
			}

			if !hasAnyPermission(result.Permissions, required) {
				response.Forbidden(w, "权限不足")
				return
			}

			ctx := context.WithValue(r.Context(), permissionsContextKey{}, result.Permissions)
			ctx = context.WithValue(ctx, employeeContextKey{}, result.EmployeeID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func hasAnyPermission(granted []string, required []string) bool {
	set := make(map[string]struct{}, len(granted))
	for _, permission := range granted {
		set[permission] = struct{}{}
	}
	for _, permission := range required {
		if _, ok := set[permission]; ok {
			return true
		}
	}
	return false
}
