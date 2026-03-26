package audit

import (
	"context"
	"net/http"
	"strings"

	"gantt-saas/internal/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger 审计日志记录器。
type Logger struct {
	repo   *Repository
	logger *zap.Logger
}

// NewLogger 创建审计日志记录器。
func NewLogger(repo *Repository, logger *zap.Logger) *Logger {
	return &Logger{repo: repo, logger: logger}
}

// Log 异步写入审计日志。
func (l *Logger) Log(ctx context.Context, entry AuditEntry) {
	var orgNodeIDPtr *string
	if entry.OrgNodeID != "" {
		orgNodeIDPtr = &entry.OrgNodeID
	}
	var resourceIDPtr *string
	if entry.ResourceID != "" {
		resourceIDPtr = &entry.ResourceID
	}

	auditLog := &AuditLog{
		ID:           uuid.New().String(),
		OrgNodeID:    orgNodeIDPtr,
		UserID:       entry.UserID,
		Username:     entry.Username,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   resourceIDPtr,
		Detail:       entry.Detail,
		IP:           entry.IP,
		UserAgent:    entry.UserAgent,
		StatusCode:   entry.StatusCode,
	}

	if err := l.repo.Create(ctx, auditLog); err != nil {
		l.logger.Error("写入审计日志失败",
			zap.String("action", entry.Action),
			zap.String("user_id", entry.UserID),
			zap.Error(err),
		)
	}
}

// deriveAction 从 HTTP 请求推断操作名称。
func deriveAction(r *http.Request) string {
	path := r.URL.Path
	method := r.Method

	// 移除 /api/v1/ 前缀
	path = strings.TrimPrefix(path, "/api/v1/")
	// 分割路径
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return method + "_unknown"
	}

	var action string
	switch method {
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = strings.ToLower(method)
	}

	// 提取资源类型（去掉 ID 部分）
	resource := parts[0]
	if len(parts) > 1 {
		// 如果第二部分看起来像操作（如 suspend），附加它
		secondPart := parts[len(parts)-1]
		// 判断是否为 UUID 或动作名
		if !isUUID(secondPart) && len(parts) > 2 {
			action = secondPart
		}
	}

	return action + "_" + resource
}

// deriveResourceType 从 HTTP 请求推断资源类型。
func deriveResourceType(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// deriveResourceID 从 HTTP 请求推断资源 ID。
func deriveResourceID(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if isUUID(p) {
			return p
		}
	}
	return ""
}

// isUUID 简单判断字符串是否看起来像 UUID。
func isUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// extractClientIP 提取客户端 IP。
func extractClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	// 去掉端口号
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// extractUserInfo 从 context 中提取用户信息。
func extractUserInfo(ctx context.Context) (userID, orgNodeID string) {
	claims := auth.GetClaims(ctx)
	if claims != nil {
		userID = claims.UserID
		orgNodeID = claims.OrgNodeID
	}
	return
}
