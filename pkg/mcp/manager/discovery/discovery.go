package discovery

import (
	"context"
)

// AgentService 智能体服务信息
type AgentService struct {
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"`
	Tools    []string `json:"tools"`
	Healthy  bool     `json:"healthy"`
}

// AgentDiscovery 智能体发现接口
type AgentDiscovery interface {
	// DiscoverAgents 发现所有可用的智能体服务
	DiscoverAgents(ctx context.Context) ([]*AgentService, error)

	// WatchAgents 监听智能体服务变化
	WatchAgents(ctx context.Context, callback func([]*AgentService)) error

	// Start 启动发现服务
	Start(ctx context.Context) error

	// Stop 停止发现服务
	Stop(ctx context.Context) error
}
