package tenant

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NodeScope 仅查本节点数据。
func NodeScope(nodeID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("org_node_id = ?", nodeID)
	}
}

// NodeTreeScope 查本节点和所有后代节点数据。
func NodeTreeScope(nodeIDs []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("org_node_id IN ?", nodeIDs)
	}
}

// GetDescendantNodeIDs 获取某节点及其所有后代节点的 ID 列表。
func GetDescendantNodeIDs(db *gorm.DB, nodePath string) ([]string, error) {
	var ids []string
	err := db.Model(&OrgNode{}).
		Where("path LIKE ?", nodePath+"%").
		Where("status = ?", StatusActive).
		Pluck("id", &ids).Error
	return ids, err
}

// ApplyScope 根据 Context 自动应用 GORM 组织节点过滤。
func ApplyScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	nodeID := GetOrgNodeID(ctx)
	if nodeID == "" {
		return db
	}
	if IsScopeTree(ctx) {
		nodePath := GetOrgNodePath(ctx)
		nodeIDs, err := GetDescendantNodeIDs(db, nodePath)
		if err != nil || len(nodeIDs) == 0 {
			return db.Scopes(NodeScope(nodeID))
		}
		return db.Scopes(NodeTreeScope(nodeIDs))
	}
	return db.Scopes(NodeScope(nodeID))
}

// platformTables 平台级表白名单，不需要 org_node_id 条件。
var platformTables = map[string]bool{
	"org_nodes":        true,
	"users":            true,
	"user_node_roles":  true,
	"ai_model_configs": true,
	"subscriptions":    true,
	"audit_logs":       true,
	"system_configs":   true,
}

// RegisterTenantGuard 注册 GORM Callback，检测所有查询是否携带 org_node_id 条件。
// 未携带时拒绝执行并告警。
func RegisterTenantGuard(db *gorm.DB, logger *zap.Logger) {
	db.Callback().Query().Before("gorm:query").Register("tenant:guard:query", func(tx *gorm.DB) {
		checkTenantCondition(tx, logger)
	})
	db.Callback().Update().Before("gorm:update").Register("tenant:guard:update", func(tx *gorm.DB) {
		checkTenantCondition(tx, logger)
	})
	db.Callback().Delete().Before("gorm:delete").Register("tenant:guard:delete", func(tx *gorm.DB) {
		checkTenantCondition(tx, logger)
	})
}

func checkTenantCondition(tx *gorm.DB, logger *zap.Logger) {
	if tx.Statement == nil {
		return
	}

	table := tx.Statement.Table
	if table == "" && tx.Statement.Schema != nil {
		table = tx.Statement.Schema.Table
	}
	if table == "" {
		return
	}

	// 平台级表不检查
	if platformTables[table] {
		return
	}

	// 检查是否已通过 Context 跳过检查（内部使用）
	if isGuardSkipped(tx.Statement.Context) {
		return
	}

	// 检查 WHERE 子句是否包含 org_node_id
	if !containsOrgNodeCondition(tx) {
		err := fmt.Errorf("TENANT GUARD: query on table '%s' missing org_node_id condition", table)
		_ = tx.AddError(err)
		logger.Error("租户隔离违规",
			zap.String("table", table),
			zap.String("sql", tx.Statement.SQL.String()),
		)
	}
}

// containsOrgNodeCondition 检查 GORM 查询是否包含 org_node_id 条件。
func containsOrgNodeCondition(tx *gorm.DB) bool {
	// 检查 Statement.Clauses 中的 WHERE 条件
	for _, clause := range tx.Statement.Clauses {
		if strings.Contains(fmt.Sprintf("%v", clause.Expression), "org_node_id") {
			return true
		}
	}
	// 检查已构建的 SQL
	if strings.Contains(tx.Statement.SQL.String(), "org_node_id") {
		return true
	}
	// 检查 vars 中可能的条件
	for _, v := range tx.Statement.Vars {
		if s, ok := v.(string); ok && strings.Contains(s, "org_node_id") {
			return true
		}
	}
	return false
}

type guardSkipKey struct{}

// SkipTenantGuard 在 Context 中标记跳过租户防漏检测（仅限内部管理操作）。
func SkipTenantGuard(ctx context.Context) context.Context {
	return context.WithValue(ctx, guardSkipKey{}, true)
}

func isGuardSkipped(ctx context.Context) bool {
	v, _ := ctx.Value(guardSkipKey{}).(bool)
	return v
}
