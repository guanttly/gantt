package repository

// IRepositoryProvider 仓储提供者接口
type IRepositoryProvider interface {
	Conversation() IConversationRepository
}

// IRepositoryBuilder 仓储构建器接口
type IRepositoryBuilder interface {
	WithLogger(logger interface{}) IRepositoryBuilder
	WithDB(db interface{}) IRepositoryBuilder
	Build() IRepositoryProvider
}
