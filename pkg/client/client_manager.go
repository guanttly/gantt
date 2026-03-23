package client

import (
	"log/slog"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/logging"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
)

const (
	default_group = "DEFAULT_GROUP"
)

type ClientManager struct {
	logger logging.ILogger

	clientMap map[ServiceName]*ServiceClient
}

func NewClientManager(
	logger logging.ILogger,
	nacosClient naming_client.INamingClient,
	serviceNames ...ServiceName,
) (IClientManager, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if nacosClient == nil {
		return nil, errors.NewInitializationError("nacos client is nil", nil)
	}

	clientMap := make(map[ServiceName]*ServiceClient)
	for _, serviceName := range serviceNames {
		clientMap[serviceName] = NewServiceClient(nacosClient, serviceName, logger)
	}

	return &ClientManager{
		logger:    logger,
		clientMap: clientMap,
	}, nil
}

func (cm *ClientManager) GetClient(serviceName ServiceName) (*ServiceClient, error) {
	client, exists := cm.clientMap[serviceName]
	if !exists {
		return nil, errors.NewNotFoundError("service client not found", nil)
	}
	return client, nil
}

func (cm *ClientManager) GetAllClients() map[ServiceName]*ServiceClient {
	return cm.clientMap
}

type IClientManager interface {
	GetClient(serviceName ServiceName) (*ServiceClient, error)
	GetAllClients() map[ServiceName]*ServiceClient
}
