// Package engine 提供工作流引擎核心组件
package engine

import (
	"context"
	"sync"
	"time"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"
)

// workflowContext 工作流上下文实现
type workflowContext struct {
	id             string
	logger         logging.ILogger
	configurator   config.IServiceConfigurator
	sessionService session.ISessionService
	services       IServiceRegistry
	metrics        Metrics
	metadata       *sync.Map // 使用指针避免复制锁
}

// NewWorkflowContext 创建工作流上下文
func NewWorkflowContext(
	id string,
	logger logging.ILogger,
	configurator config.IServiceConfigurator,
	sessionService session.ISessionService,
	services IServiceRegistry,
	metrics Metrics,
) Context {
	return &workflowContext{
		id:             id,
		logger:         logger,
		configurator:   configurator,
		sessionService: sessionService,
		services:       services,
		metrics:        metrics,
		metadata:       &sync.Map{}, // 初始化为指针
	}
}

func (c *workflowContext) ID() string {
	return c.id
}

func (c *workflowContext) Logger() logging.ILogger {
	return c.logger
}

func (c *workflowContext) Now() time.Time {
	return time.Now()
}

func (c *workflowContext) Send(ctx context.Context, event Event, payload any) error {
	// 这个方法由 Actor 重写实现
	return nil
}

func (c *workflowContext) Session() *session.Session {
	if c.sessionService == nil {
		return nil
	}
	sess, _ := c.sessionService.Get(context.Background(), c.id)
	return sess
}

func (c *workflowContext) SessionService() session.ISessionService {
	return c.sessionService
}

func (c *workflowContext) Services() IServiceRegistry {
	return c.services
}

func (c *workflowContext) Metrics() Metrics {
	return c.metrics
}

func (c *workflowContext) GetMetadata(key string) (any, bool) {
	return c.metadata.Load(key)
}

func (c *workflowContext) SetMetadata(key string, value any) {
	c.metadata.Store(key, value)
}

// ==================== 依赖注入方法 (With Pattern) ====================

// WithLogger 替换 Logger 实现
func (c *workflowContext) WithLogger(logger logging.ILogger) Context {
	c.logger = logger
	return c
}

// WithConfigurator 替换配置器实现
func (c *workflowContext) WithConfigurator(configurator config.IServiceConfigurator) Context {
	c.configurator = configurator
	return c
}

// WithSessionService 替换 SessionService 实现
func (c *workflowContext) WithSessionService(sessionService session.ISessionService) Context {
	c.sessionService = sessionService
	return c
}

// WithServices 替换 ServiceRegistry 实现
func (c *workflowContext) WithServices(services IServiceRegistry) Context {
	c.services = services
	return c
}

// WithMetrics 替换 Metrics 实现
func (c *workflowContext) WithMetrics(metrics Metrics) Context {
	c.metrics = metrics
	return c
}
