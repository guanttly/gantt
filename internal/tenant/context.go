package tenant

import (
	"context"
)

type contextKey string

const (
	ctxKeyOrgNodeID   contextKey = "org_node_id"
	ctxKeyOrgNodePath contextKey = "org_node_path"
	ctxKeyScopeTree   contextKey = "scope_tree"
)

// WithOrgNode 将组织节点信息写入 Context。
func WithOrgNode(ctx context.Context, nodeID, nodePath string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyOrgNodeID, nodeID)
	ctx = context.WithValue(ctx, ctxKeyOrgNodePath, nodePath)
	return ctx
}

// WithScopeTree 标记当前请求需要查询含下级数据。
func WithScopeTree(ctx context.Context, tree bool) context.Context {
	return context.WithValue(ctx, ctxKeyScopeTree, tree)
}

// GetOrgNodeID 从 Context 中获取当前组织节点 ID。
func GetOrgNodeID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyOrgNodeID).(string)
	return v
}

// GetOrgNodePath 从 Context 中获取当前组织节点路径。
func GetOrgNodePath(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyOrgNodePath).(string)
	return v
}

// IsScopeTree 检查当前请求是否需要查询含下级数据。
func IsScopeTree(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyScopeTree).(bool)
	return v
}
