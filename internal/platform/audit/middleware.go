package audit

import (
	"context"
	"net/http"

	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// responseWrapper 包装 ResponseWriter 以捕获状态码。
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWrapper) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// Middleware 审计日志中间件，仅记录写操作（POST/PUT/DELETE/PATCH）。
func Middleware(auditLogger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 仅记录写操作
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// 包装 ResponseWriter 以捕获状态码
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			// 异步写审计日志（仅记录成功的写操作）
			statusCode := ww.Status()
			if statusCode < 400 {
				userID, orgNodeID := extractUserInfo(r.Context())
				entry := AuditEntry{
					OrgNodeID:    orgNodeID,
					UserID:       userID,
					Action:       deriveAction(r),
					ResourceType: deriveResourceType(r),
					ResourceID:   deriveResourceID(r),
					IP:           extractClientIP(r),
					UserAgent:    r.UserAgent(),
					StatusCode:   statusCode,
				}

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					auditLogger.Log(ctx, entry)
				}()
			}
		})
	}
}
