package tenant

import (
	"net/http"

	"gantt-saas/internal/common/response"
)

// Middleware 组织节点 HTTP 中间件。
// 从请求 Context 中提取 org_node_id（由 auth 中间件写入），并处理 scope=tree 参数。
// auth 中间件在解析 JWT 后已将 OrgNodeID / OrgNodePath 写入 Context。
func Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 从 Context 中获取 org_node_id（由 auth 中间件设置）
			nodeID := GetOrgNodeID(ctx)
			if nodeID == "" {
				// 开发阶段备选：从 Header 中获取
				nodeID = r.Header.Get("X-Org-Node-ID")
				nodePath := r.Header.Get("X-Org-Node-Path")
				if nodeID != "" {
					ctx = WithOrgNode(ctx, nodeID, nodePath)
				}
			}

			if nodeID == "" {
				response.Unauthorized(w, "缺少组织节点信息")
				return
			}

			// 检查 ?scope=tree 参数
			if r.URL.Query().Get("scope") == "tree" {
				ctx = WithScopeTree(ctx, true)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
