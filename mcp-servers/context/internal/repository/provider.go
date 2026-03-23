package repository

import (
	"gorm.io/gorm"

	"jusha/agent/server/context/domain/repository"
	"jusha/mcp/pkg/logging"
)

type baseRepositoryProvider struct {
	logger logging.ILogger
	db     *gorm.DB

	conversationRepo repository.IConversationRepository
}

func (p *baseRepositoryProvider) Conversation() repository.IConversationRepository {
	return p.conversationRepo
}

type repositoryBuilder struct {
	logger logging.ILogger
	db     *gorm.DB
}

func NewRepositoryProviderBuilder() repository.IRepositoryBuilder {
	return &repositoryBuilder{}
}

func (b *repositoryBuilder) WithLogger(logger interface{}) repository.IRepositoryBuilder {
	if l, ok := logger.(logging.ILogger); ok {
		b.logger = l
	}
	return b
}

func (b *repositoryBuilder) WithDB(db interface{}) repository.IRepositoryBuilder {
	if d, ok := db.(*gorm.DB); ok {
		b.db = d
	}
	return b
}

func (b *repositoryBuilder) Build() repository.IRepositoryProvider {
	if b.logger == nil {
		panic("logger is required to build RepositoryProvider")
	}

	if b.db == nil {
		panic("db is required to build RepositoryProvider")
	}

	return &baseRepositoryProvider{
		logger:          b.logger,
		db:              b.db,
		conversationRepo: NewConversationRepository(b.db, b.logger),
	}
}
