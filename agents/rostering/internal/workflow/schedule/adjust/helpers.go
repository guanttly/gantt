package adjust

import (
	"fmt"

	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
)

// ========== 工具函数 ==========

// GetScheduleAdjustContext 获取排班调整上下文
func GetScheduleAdjustContext(sess *session.Session) (*d_model.ScheduleAdjustContext, error) {
	if sess == nil {
		return nil, fmt.Errorf("session is nil")
	}

	ctxRaw, ok := sess.Data[d_model.DataKeyScheduleAdjustContext]
	if !ok {
		return nil, fmt.Errorf("adjust context not found in session")
	}

	adjustCtx, ok := ctxRaw.(*d_model.ScheduleAdjustContext)
	if !ok {
		return nil, fmt.Errorf("invalid adjust context type")
	}

	return adjustCtx, nil
}

// GetOrCreateScheduleAdjustContext 获取或创建排班调整上下文
func GetOrCreateScheduleAdjustContext(sess *session.Session) *d_model.ScheduleAdjustContext {
	if sess == nil {
		return d_model.NewScheduleAdjustContext()
	}

	if ctxRaw, ok := sess.Data[d_model.DataKeyScheduleAdjustContext]; ok {
		if adjustCtx, ok := ctxRaw.(*d_model.ScheduleAdjustContext); ok {
			return adjustCtx
		}
	}

	return d_model.NewScheduleAdjustContext()
}
