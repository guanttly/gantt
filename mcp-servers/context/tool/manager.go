package tool

import (
	"context"

	"jusha/agent/server/context/config"
	"jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/tool/conversation"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// ToolManager 负责根据配置初始化并持有工具
type ToolManager struct {
	logger         logging.ILogger
	cfg            config.IContextConfigurator
	serviceProvider service.IServiceProvider
	tools          []mcp.ITool
}

func NewToolManager(logger logging.ILogger, cfg config.IContextConfigurator, serviceProvider service.IServiceProvider) *ToolManager {
	return &ToolManager{
		logger:          logger,
		cfg:             cfg,
		serviceProvider: serviceProvider,
	}
}

// GetTools 返回所有已初始化的工具
func (tm *ToolManager) GetTools() []mcp.ITool {
	return tm.tools
}

// Init 根据配置初始化工具
func (tm *ToolManager) Init(ctx context.Context) error {
	tm.tools = []mcp.ITool{}

	// 获取配置中启用的工具
	cfg := tm.cfg.GetConfig()
	enabledTools := make(map[string]bool)
	for _, toolName := range cfg.Tools.EnabledTools {
		enabledTools[toolName] = true
	}

	// 注册会话管理工具
	if enabledTools["conversation.new"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewNewConversationTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.append"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewAppendMessageTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.history"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewGetHistoryTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.list"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewListConversationsTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.update_workflow_context"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewUpdateWorkflowContextTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.get_workflow_context"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewGetWorkflowContextTool(tm.logger, tm.serviceProvider))
	}
	if enabledTools["conversation.update_meta"] || len(enabledTools) == 0 {
		tm.tools = append(tm.tools, conversation.NewUpdateMetaTool(tm.logger, tm.serviceProvider))
	}

	tm.logger.Info("Context tools initialized", "count", len(tm.tools))
	return nil
}
