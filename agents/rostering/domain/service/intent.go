package service

import (
	"context"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
)

// IIntentService 意图识别服务
// 整合了意图识别和事件映射功能
type IIntentService interface {
	// ========== 意图识别接口 (实现 session.IIntentRecognizer) ==========

	// Recognize 识别用户意图
	Recognize(ctx context.Context, req session.IntentRecognizeRequest) (*session.IntentRecognizeResponse, error)

	// ValidateIntent 验证意图的有效性
	ValidateIntent(ctx context.Context, intent *session.Intent) ([]string, error)

	// SupportedIntents 返回支持的意图类型列表
	SupportedIntents() []string

	// ========== 事件映射接口 (实现 wsbridge.IIntentEventMapper) ==========

	// MapIntentToEvent 将意图类型映射到工作流事件
	MapIntentToEvent(intentType string) engine.Event
}
