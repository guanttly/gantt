package service

import (
	"context"
	"jusha/mcp/pkg/workflow"
)

// IServiceProvider 聚合所有可用领域服务
type IServiceProvider interface {
	GetContext() context.Context
	GetSessionService() ISessionService
	GetIntentService() IIntentService
	GetDataService() IRosteringService
	GetConversationService() IConversationService
	// GetActorSystem 已废弃，使用 GetInfrastructure
	GetActorSystem() workflow.IWorkflowInfrastructure
	// GetInfrastructure 获取完整的 Workflow 基础设施
	GetInfrastructure() workflow.IWorkflowInfrastructure
}

type IServiceProviderBuilder interface {
	WithLogger() IServiceProviderBuilder
	WithConfigurator() IServiceProviderBuilder
	WithRosteringClient() IServiceProviderBuilder
	WithWorkflow() IServiceProviderBuilder
	Build() IServiceProvider
}
