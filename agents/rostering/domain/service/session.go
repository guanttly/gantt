package service

import (
	"jusha/mcp/pkg/workflow/session"
)

// ISessionService 会话服务接口
// 直接使用 pkg/workflow/session 的通用能力
// 业务特定方法应该在其他服务中实现（如 IRosteringService）
type ISessionService = session.ISessionService
