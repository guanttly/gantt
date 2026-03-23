package repository

import (
	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/mcp/pkg/logging"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
)

const (
	managementServiceName    = "management-service"
	managementServiceBaseURL = "api/v1"
)

type baseRepositoryProvider struct {
	logger logging.ILogger
	cfg    config.IRosteringConfigurator

	managementRepo repository.IManagementRepository
}

func newBaseRepositoryProvider(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	namingClient naming_client.INamingClient,
	serviceName string,
	baseURL string,
) *baseRepositoryProvider {
	return &baseRepositoryProvider{
		logger: logger,
		cfg:    cfg,

		managementRepo: newManagementClient(logger, cfg, namingClient, serviceName, baseURL),
	}
}

type repositoryBuilder struct {
	logger       logging.ILogger
	cfg          config.IRosteringConfigurator
	namingClient naming_client.INamingClient
}

func NewRepositoryProviderBuilder() repository.IRepositoryBuilder {
	return &repositoryBuilder{}
}
func (b *repositoryBuilder) WithLogger(logger logging.ILogger) repository.IRepositoryBuilder {
	b.logger = logger
	return b
}

func (b *repositoryBuilder) WithConfigurator(cfg config.IRosteringConfigurator) repository.IRepositoryBuilder {
	b.cfg = cfg
	return b
}

func (b *repositoryBuilder) WithNamingClient(namingClient naming_client.INamingClient) repository.IRepositoryBuilder {
	b.namingClient = namingClient
	return b
}

func (b *repositoryBuilder) Build() repository.IRepositoryProvider {
	if b.logger == nil {
		panic("logger is required to build RepositoryProvider")
	}

	if b.cfg == nil {
		panic("configurator is required to build RepositoryProvider")
	}

	if b.namingClient == nil {
		panic("naming client is required to build RepositoryProvider")
	}

	return newBaseRepositoryProvider(
		b.logger,
		b.cfg,
		b.namingClient,
		managementServiceName,
		managementServiceBaseURL,
	)
}

func (p *baseRepositoryProvider) GetManagementRepository() repository.IManagementRepository {
	return p.managementRepo
}
