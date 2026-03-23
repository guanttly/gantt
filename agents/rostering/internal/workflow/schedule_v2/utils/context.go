package utils

import (
	"context"
	"fmt"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
)

// GetShiftSchedulingContext 从 session 获取共享排班上下文
func GetShiftSchedulingContext(sess *session.Session) (*d_model.ShiftSchedulingContext, error) {
	if ctx, ok := sess.Data[d_model.DataKeyShiftSchedulingContext]; ok {
		if shiftCtx, ok := ctx.(*d_model.ShiftSchedulingContext); ok {
			return shiftCtx, nil
		}
	}
	return nil, fmt.Errorf("shift scheduling context not found in session")
}

// SaveShiftSchedulingContext 保存共享排班上下文到 session
func SaveShiftSchedulingContext(ctx context.Context, wctx engine.Context, shiftCtx *d_model.ShiftSchedulingContext) error {
	sess := wctx.Session()
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyShiftSchedulingContext, shiftCtx); err != nil {
		return fmt.Errorf("failed to save shift scheduling context: %w", err)
	}
	return nil
}
