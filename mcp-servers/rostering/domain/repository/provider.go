package repository

import (
	"jusha/gantt/mcp/rostering/config"
	"jusha/mcp/pkg/logging"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
)

type IRepositoryProvider interface {
	GetManagementRepository() IManagementRepository
}

type IRepositoryBuilder interface {
	WithLogger(logger logging.ILogger) IRepositoryBuilder
	WithConfigurator(cfg config.IRosteringConfigurator) IRepositoryBuilder
	WithNamingClient(namingClient naming_client.INamingClient) IRepositoryBuilder
	Build() IRepositoryProvider
}
