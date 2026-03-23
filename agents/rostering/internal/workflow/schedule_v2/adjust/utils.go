package adjust

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
)

// ============================================================
// 上下文管理
// ============================================================

const (
	// DataKeyAdjustV2Context 调整上下文在 session 中的 key
	DataKeyAdjustV2Context = "adjust_v2_context"
)

// LoadAdjustV2Context 从 session 加载调整上下文
func LoadAdjustV2Context(sess *session.Session) (*AdjustV2Context, error) {
	if ctxData, ok := sess.Data[DataKeyAdjustV2Context]; ok {
		// 尝试直接类型断言
		if adjustCtx, ok := ctxData.(*AdjustV2Context); ok {
			return adjustCtx, nil
		}
		// 尝试从 JSON 反序列化
		if ctxMap, ok := ctxData.(map[string]any); ok {
			jsonBytes, err := json.Marshal(ctxMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal context: %w", err)
			}
			var adjustCtx AdjustV2Context
			if err := json.Unmarshal(jsonBytes, &adjustCtx); err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
			return &adjustCtx, nil
		}
	}
	return nil, fmt.Errorf("adjust V2 context not found in session")
}

// SaveAdjustV2Context 保存调整上下文到 session
func SaveAdjustV2Context(ctx context.Context, wctx engine.Context, adjustCtx *AdjustV2Context) error {
	sess := wctx.Session()
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, DataKeyAdjustV2Context, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust V2 context: %w", err)
	}
	return nil
}
