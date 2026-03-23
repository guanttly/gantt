package core

import (
	"context"

	"jusha/mcp/pkg/workflow/engine"

	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// ============================================================
// CoreV3 上下文管理（复用 utils 中的函数）
// ============================================================

// GetCoreV3TaskContext 从 session 获取任务上下文
// 这是 core 包对 utils 函数的包装，保持接口一致性
func GetCoreV3TaskContext(ctx context.Context, wctx engine.Context) (*utils.CoreV3TaskContext, error) {
	return utils.GetCoreV3TaskContext(ctx, wctx)
}

// SaveCoreV3TaskContext 保存任务上下文到 session
func SaveCoreV3TaskContext(ctx context.Context, wctx engine.Context, taskCtx *utils.CoreV3TaskContext) error {
	return utils.SaveCoreV3TaskContext(ctx, wctx, taskCtx)
}
