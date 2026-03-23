package middleware

import (
	"context"
	"fmt"
	"jusha/mcp/pkg/logging"
	"log/slog"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// LoggingMiddleware 创建一个记录请求和响应的端点中间件
func LoggingMiddleware(logger logging.ILogger, method string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request any) (response any, err error) {
			logger.Debug("method %s event request_received request %s received", method, fmt.Sprintf("%+v", request)) // 记录请求详情 (注意生产环境可能需要脱敏)
			defer func(begin time.Time) {
				logger.Error(
					"method", method,
					"event", "request_completed",
					slog.Int64("duration_ms", time.Since(begin).Milliseconds()),
					"error", err, // err 为 nil 时不会记录 error 键
				)
			}(time.Now())
			response, err = next(ctx, request)
			return
		}
	}
}
