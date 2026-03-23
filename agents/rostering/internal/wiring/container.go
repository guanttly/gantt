package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/agent/rostering/config"
	"jusha/agent/rostering/internal/adapter"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/ai/toolcalling"
	common_config "jusha/mcp/pkg/config"
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
	"jusha/mcp/pkg/workflow"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/progressive"
	"jusha/mcp/pkg/workflow/session"

	// 导入 clients
	d_service "jusha/agent/rostering/domain/service"
	i_service "jusha/agent/rostering/internal/service"
	context_client "jusha/agent/sdk/context/client"
	context_domain "jusha/agent/sdk/context/domain"
	scheduling_client "jusha/agent/sdk/rostering/client"
	scheduling_domain "jusha/agent/sdk/rostering/domain"

	// 导入工作流定义包，触发 init() 函数进行工作流注册
	_ "jusha/agent/rostering/internal/workflow/schedule"
	_ "jusha/agent/rostering/internal/workflow/schedule_v2"
	_ "jusha/agent/rostering/internal/workflow/schedule_v3"
	_ "jusha/agent/rostering/internal/workflow/schedule_v4/create"

	// 导入 schedule_v3 utils
	schedule_v3_utils "jusha/agent/rostering/internal/workflow/schedule_v3/utils"

	// V4 规则引擎
	rule_engine "jusha/agent/rostering/internal/engine"
)

// Container 依赖注入容器
type Container struct {
	ctx          context.Context
	logger       logging.ILogger
	configurator config.IRosteringConfigurator

	// 基础设施组件
	namingClient naming_client.INamingClient
	mcpManager   mcp.MCPServerManager
	toolBus      mcp.IToolBus

	// 客户端
	rosteringClient scheduling_domain.IClient
	contextClient   context_domain.IContextClient

	// 服务层
	sessionService               d_service.ISessionService
	intentService                d_service.IIntentService
	schedulingAIService          d_service.ISchedulingAIService
	rosteringService             d_service.IRosteringService
	conversationService          d_service.IConversationService
	progressiveSchedulingService interface{} // IProgressiveSchedulingService from schedule_v3/utils

	// 通用服务（用于渐进式任务执行）
	toolCallingService     interface{}           // toolcalling.IToolCallingService
	progressiveTaskService interface{}           // progressive.IProgressiveTaskService
	aiFactoryForTasks      *ai.AIProviderFactory // AI 工厂（用于任务执行）

	// V4 确定性规则引擎
	ruleEngine *rule_engine.RuleEngine

	// Workflow 基础设施（替代 store 和 actorSystem）
	infrastructure workflow.IWorkflowInfrastructure

	// 生命周期
	started bool
}

const _mcp_group = "mcp-server"

// NewContainer 创建依赖注入容器
func NewContainer(ctx context.Context, logger logging.ILogger, cfg config.IRosteringConfigurator) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	container := &Container{
		logger:       logger.With("component", "Container"),
		configurator: cfg,
		ctx:          ctx,
	}

	// 初始化基础设施
	if err := container.initInfrastructure(ctx); err != nil {
		return nil, fmt.Errorf("init infrastructure: %w", err)
	}

	// 初始化客户端（需要在 contextGateway 之前）
	if err := container.initClients(); err != nil {
		return nil, fmt.Errorf("init clients: %w", err)
	}

	// 初始化服务层
	if err := container.initServices(ctx); err != nil {
		return nil, fmt.Errorf("init services: %w", err)
	}

	// 初始化工作流
	if err := container.initWorkflow(); err != nil {
		return nil, fmt.Errorf("init workflow: %w", err)
	}

	container.started = true
	container.logger.Info("Dependency injection container initialized successfully")

	return container, nil
}

// initInfrastructure 初始化基础设施组件
func (c *Container) initInfrastructure(ctx context.Context) error {
	c.logger.Info("Initializing infrastructure components")

	// 1. 创建命名服务客户端
	if c.configurator != nil {
		baseConfig := c.configurator.GetBaseConfig()
		if baseConfig.Discovery != nil && baseConfig.Discovery.Nacos != nil {
			client, err := discovery.NewNacosNamingClient(baseConfig.Discovery.Nacos, c.logger)
			if err != nil {
				c.logger.Warn("Failed to create naming client", "error", err)
			} else {
				c.namingClient = client
			}
		}
	}

	// 2. 创建MCP管理器
	if c.namingClient != nil {
		manager := mcp.NewDefaultMCPServerManager(
			c.namingClient,
			_mcp_group, // 服务组名
			c.logger,
		)

		// 启动MCP管理器
		if err := manager.Start(ctx); err != nil {
			c.logger.Error("Failed to start MCP server manager", "error", err)
			return fmt.Errorf("start MCP manager: %w", err)
		}

		c.mcpManager = manager
	}

	// 3. 创建工具总线
	if c.mcpManager != nil {
		toolBusConfig := mcp.DefaultToolBusConfig()

		// 从配置中读取超时参数
		if c.configurator != nil {
			cfg := c.configurator.GetConfig()
			if cfg.MCP != nil && cfg.MCP.ClientTimeout > 0 {
				toolBusConfig.DefaultTimeout = time.Duration(cfg.MCP.ClientTimeout) * time.Second
			}
		}

		c.toolBus = mcp.NewMCPToolBus(c.mcpManager, toolBusConfig, c.logger)
	}

	return nil
}

// initClients 初始化客户端
func (c *Container) initClients() error {
	if c.toolBus == nil {
		return fmt.Errorf("tool bus not available")
	}

	c.rosteringClient = scheduling_client.NewClient(c.toolBus, c.logger)
	c.contextClient = context_client.NewClient(c.toolBus, c.logger)
	return nil
}

// initServices 初始化服务层
func (c *Container) initServices(_ context.Context) error {
	c.logger.Info("Initializing service layer")

	// 1. 创建 Workflow Infrastructure
	c.infrastructure = workflow.NewDefaultInfrastructure(c.logger, c.configurator)
	c.sessionService = c.infrastructure.GetSessionService()

	// 2. 创建 RosteringService（需要在 WorkflowInitializer 之前创建）
	c.rosteringService = i_service.NewRosteringService(c.logger, c.rosteringClient, c.toolBus)

	// 3. 注入业务层的 Mapper 到 Infrastructure
	workflowInitializer := adapter.NewWorkflowInitializer(c.rosteringService)
	commandMapper := adapter.NewCommandMapper()
	c.infrastructure.
		With(workflowInitializer).
		With(commandMapper)

	// 4. 创建意图服务
	if intentSvc, err := i_service.NewIntentService(c.logger, c.sessionService, c.configurator); err != nil {
		c.logger.Error("Failed to create intent service", "error", err)
		return fmt.Errorf("create intent service: %w", err)
	} else {
		c.intentService = intentSvc
		c.infrastructure.With(intentSvc)
	}

	// 5. 创建 AI 排班服务
	if schedulingAISvc, err := i_service.NewSchedulingAIService(c.logger, c.configurator); err != nil {
		c.logger.Error("Failed to create scheduling AI service", "error", err)
		return fmt.Errorf("create scheduling AI service: %w", err)
	} else {
		c.schedulingAIService = schedulingAISvc
	}

	// 5.5. 创建渐进式排班服务 (V3 工作流)
	if c.configurator != nil {
		// 需要创建一个独立的 AI Factory 用于渐进式排班服务
		baseConfigurator := c.configurator.(common_config.IServiceConfigurator)
		aiFactory := ai.NewAIModelFactory(context.Background(), baseConfigurator, c.logger)
		c.aiFactoryForTasks = aiFactory
		c.progressiveSchedulingService = schedule_v3_utils.NewProgressiveSchedulingService(c.logger, aiFactory, c.configurator)
	}

	// 5.6. 创建通用服务（工具调用和渐进式任务执行）
	c.toolCallingService = toolcalling.NewToolCallingService(c.logger)
	c.progressiveTaskService = progressive.NewProgressiveTaskService(c.logger)

	// 5.7. 创建 V4 确定性规则引擎
	c.ruleEngine = rule_engine.NewRuleEngine(c.logger)

	// 6. 创建 ConversationService
	if c.contextClient == nil {
		c.logger.Warn("ContextClient not available, ConversationService will not be created")
	} else {
		c.conversationService = i_service.NewConversationManager(c.logger, c.contextClient, c.sessionService, c.configurator, c.toolBus)
		c.setupAutoSaveCallback()
	}

	c.logger.Info("Service layer initialized successfully")
	return nil
}

// setupAutoSaveCallback 设置自动保存回调
// 在 session 更新时，如果是消息更新，则异步保存到 context service
func (c *Container) setupAutoSaveCallback() {
	if c.conversationService == nil || c.sessionService == nil {
		return
	}

	// 获取原有的 onUpdate 回调（由 Infrastructure 设置）
	// 由于 Infrastructure 在 initServices 之前已经设置了回调，
	// 我们需要包装它，既保留原有功能，又添加自动保存功能
	//
	// 注意：这里无法直接获取原有回调，所以我们需要在 Infrastructure 设置回调之后
	// 重新设置一个包装的回调。但由于 Infrastructure 是在 initServices 中创建的，
	// 而我们的 container 是在 initServices 之后初始化的，所以我们可以安全地
	// 重新设置回调，但需要确保 Infrastructure 的回调仍然有效。
	//
	// 实际方案：由于 Infrastructure 的回调是在 Infrastructure 创建时设置的，
	// 而我们的 container 是在 Infrastructure 创建之后初始化的，
	// 我们可以通过 Infrastructure 获取 Bridge，然后获取原有的广播逻辑。
	// 但为了简化，我们采用另一种方式：
	// 在 onUpdate 回调中，检查消息数量，如果增加则异步保存所有消息

	// 保存上一次的消息数量（用于检测消息是否增加）
	lastMessageCounts := make(map[string]int)

	// 设置包装的回调
	c.sessionService.SetOnUpdate(func(sess *session.Session) {
		// 1. 先执行原有的 WebSocket 广播（通过 Infrastructure 的 Bridge）
		// 注意：由于 Infrastructure 已经设置了回调，我们需要确保它仍然有效
		// 但实际上，SetOnUpdate 会替换回调，所以我们需要手动调用 Bridge 的广播
		// 或者，我们可以通过 Infrastructure 获取 Bridge 并手动广播
		if c.infrastructure != nil && c.infrastructure.GetBridge() != nil {
			_ = c.infrastructure.GetBridge().BroadcastToSession(sess.ID, "session_updated", sess)
		}

		// 2. 检查消息数量是否增加（表示有新消息）
		lastCount, exists := lastMessageCounts[sess.ID]
		currentCount := len(sess.Messages)

		// 只有在消息数量 > 0 且消息数量增加时才保存
		// 避免在会话创建时（消息数量为0）创建空的对话记录
		if currentCount > 0 && (!exists || currentCount > lastCount) {
			// 消息数量增加，异步保存
			lastMessageCounts[sess.ID] = currentCount

			// 异步保存，不阻塞主流程
			go func() {
				ctx := context.Background()
				if err := c.conversationService.SaveConversation(ctx, sess.ID, sess.Messages); err != nil {
					c.logger.Warn("Failed to auto-save conversation", "error", err, "sessionID", sess.ID)
				} else {
					c.logger.Debug("Conversation auto-saved successfully", "sessionID", sess.ID, "messageCount", currentCount)
				}
			}()
		}
	})
}

// initWorkflow 初始化工作流
func (c *Container) initWorkflow() error {
	c.logger.Info("Initializing workflow components")

	// 注册业务服务到 Infrastructure 的 ServiceRegistry
	registry := c.infrastructure.GetServiceRegistry()

	if c.configurator != nil {
		engine.RegisterService(registry, "configurator", c.configurator)
	}
	if c.intentService != nil {
		engine.RegisterService(registry, engine.ServiceKeyIntent, c.intentService)
	}
	if c.schedulingAIService != nil {
		engine.RegisterService(registry, engine.ServiceKeySchedulingAI, c.schedulingAIService)
	}
	engine.RegisterService(registry, engine.ServiceKeyRostering, c.rosteringService)
	if c.conversationService != nil {
		engine.RegisterService(registry, "conversation", c.conversationService)
	}
	if c.progressiveSchedulingService != nil {
		engine.RegisterService(registry, engine.ServiceKeyProgressiveScheduling, c.progressiveSchedulingService)
	}
	if c.toolCallingService != nil {
		engine.RegisterService(registry, engine.ServiceKeyToolCalling, c.toolCallingService)
	}
	if c.progressiveTaskService != nil {
		engine.RegisterService(registry, engine.ServiceKeyProgressiveTask, c.progressiveTaskService)
	}
	if c.aiFactoryForTasks != nil {
		engine.RegisterService(registry, engine.ServiceKeyAIFactory, c.aiFactoryForTasks)
	}
	if c.ruleEngine != nil {
		engine.RegisterService(registry, "ruleEngine", c.ruleEngine)
	}
	if c.infrastructure != nil && c.infrastructure.GetBridge() != nil {
		engine.RegisterService(registry, engine.ServiceKeyBridge, c.infrastructure.GetBridge())
	}

	// 设置 Bridge 的对话加载器（用于处理 load_conversation WebSocket 消息）
	if c.conversationService != nil && c.infrastructure != nil {
		if bridge := c.infrastructure.GetBridge(); bridge != nil {
			bridge.SetConversationLoader(c.conversationService)
		}
	}

	c.logger.Info("Workflow components initialized successfully")
	return nil
}

// GetService 获取服务提供者
func (c *Container) GetService(ctx context.Context) d_service.IServiceProvider {
	return &serviceProvider{
		container: c,
		ctx:       ctx,
	}
}

// Shutdown 关闭容器和清理资源
func (c *Container) Shutdown(ctx context.Context) error {
	if !c.started {
		return nil
	}

	c.logger.Info("Shutting down container")

	// Workflow Infrastructure 会自动清理资源
	// 不需要手动停止

	// 关闭MCP管理器
	if c.mcpManager != nil {
		if err := c.mcpManager.Stop(ctx); err != nil {
			c.logger.Error("Failed to stop MCP manager", "error", err)
		}
	}

	c.started = false
	c.logger.Info("Container shutdown completed")
	return nil
}

// IsStarted 检查容器是否已启动
func (c *Container) IsStarted() bool {
	return c.started
}

// Health 健康检查
func (c *Container) Health() error {
	if !c.started {
		return fmt.Errorf("container not started")
	}

	// 检查工具总线健康状态
	if c.toolBus != nil {
		if err := c.toolBus.Health(); err != nil {
			return fmt.Errorf("tool bus health check failed: %w", err)
		}
	}

	return nil
}

// serviceProvider 服务提供者实现
type serviceProvider struct {
	ctx       context.Context
	container *Container
}

func (p *serviceProvider) GetContext() context.Context {
	return p.ctx
}

func (p *serviceProvider) GetSessionService() d_service.ISessionService {
	return p.container.sessionService
}

func (p *serviceProvider) GetIntentService() d_service.IIntentService {
	return p.container.intentService
}

func (p *serviceProvider) GetDataService() d_service.IRosteringService {
	return p.container.rosteringService
}

func (p *serviceProvider) GetConversationService() d_service.IConversationService {
	return p.container.conversationService
}

func (p *serviceProvider) GetActorSystem() workflow.IWorkflowInfrastructure {
	return p.container.infrastructure
}

// GetInfrastructure 获取 Workflow Infrastructure
func (p *serviceProvider) GetInfrastructure() workflow.IWorkflowInfrastructure {
	return p.container.infrastructure
}

// NewServiceProvider 创建服务提供者（使用新的依赖注入容器）
func NewServiceProvider(ctx context.Context, logger logging.ILogger, cfg config.IRosteringConfigurator) (d_service.IServiceProvider, error) {
	// 创建依赖注入容器
	container, err := NewContainer(ctx, logger, cfg)
	if err != nil {
		logger.Error("Failed to create dependency injection container", "error", err)
		return nil, err
	}

	// 返回服务提供者
	return container.GetService(ctx), nil
}
