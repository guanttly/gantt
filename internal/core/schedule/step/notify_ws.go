package step

import (
	"context"

	"gantt-saas/internal/infra/websocket"
)

// NotifyWSStep 通过 WebSocket 通知前端排班进度与完成。
type NotifyWSStep struct {
	Broadcaster websocket.Broadcaster // 可为 nil，兼容无 WS 环境
}

// Name 返回步骤名称。
func (s *NotifyWSStep) Name() string { return "NotifyWS" }

// Execute 执行 WebSocket 通知。
func (s *NotifyWSStep) Execute(ctx context.Context, state *ScheduleState) error {
	// 回调通知（内部日志/调试用）
	if state.OnProgress != nil {
		state.OnProgress("NotifyWS", 1.0, "排班完成，通知前端")
	}

	// 如果没有 Broadcaster 实例，跳过推送
	if s.Broadcaster == nil {
		return nil
	}

	// 构造排班完成消息，向 schedule 对应的分组广播
	msg := websocket.NewCompleteMessage(
		state.ScheduleID,
		len(state.Assignments),
		len(state.Violations),
	)

	// 用 scheduleID 作为 groupID，前端订阅时加入同一分组
	_ = s.Broadcaster.BroadcastToGroup(state.ScheduleID, msg)
	return nil
}
