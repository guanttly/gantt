package service

// IServiceProvider 服务提供者接口
type IServiceProvider interface {
	Conversation() IConversationService
}

// IServiceBuilder 服务构建器接口
type IServiceBuilder interface {
	WithLogger(logger interface{}) IServiceBuilder
	WithRepositoryProvider(repoProvider interface{}) IServiceBuilder
	Build() IServiceProvider
}
