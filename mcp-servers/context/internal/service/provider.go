package service

import (
	"jusha/agent/server/context/domain/repository"
	"jusha/agent/server/context/domain/service"
	"jusha/mcp/pkg/logging"
)

type baseServiceProvider struct {
	logger logging.ILogger

	conversationService service.IConversationService
}

func (p *baseServiceProvider) Conversation() service.IConversationService {
	return p.conversationService
}

type serviceBuilder struct {
	logger       logging.ILogger
	repoProvider repository.IRepositoryProvider
}

func NewServiceProviderBuilder() service.IServiceBuilder {
	return &serviceBuilder{}
}

func (b *serviceBuilder) WithLogger(logger interface{}) service.IServiceBuilder {
	if l, ok := logger.(logging.ILogger); ok {
		b.logger = l
	}
	return b
}

func (b *serviceBuilder) WithRepositoryProvider(repoProvider interface{}) service.IServiceBuilder {
	if rp, ok := repoProvider.(repository.IRepositoryProvider); ok {
		b.repoProvider = rp
	}
	return b
}

func (b *serviceBuilder) Build() service.IServiceProvider {
	if b.logger == nil {
		panic("logger is required to build ServiceProvider")
	}

	if b.repoProvider == nil {
		panic("repository provider is required to build ServiceProvider")
	}

	return &baseServiceProvider{
		logger:            b.logger,
		conversationService: NewConversationService(b.logger, b.repoProvider.Conversation()),
	}
}
