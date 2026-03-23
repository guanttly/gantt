package discovery

import (
	"context"
	"jusha/mcp/pkg/logging"
	"net/http"
	"time"
)

// StaticAgentDiscovery 静态配置的智能体发现实现
type StaticAgentDiscovery struct {
	agents []*AgentService
	logger logging.ILogger
	client *http.Client
}

// StaticAgentConfig 静态配置格式
type StaticAgentConfig struct {
	Name     string   `yaml:"name"`
	Endpoint string   `yaml:"endpoint"`
	Tools    []string `yaml:"tools"`
}

// NewStaticAgentDiscovery 创建静态智能体发现服务
func NewStaticAgentDiscovery(configs []StaticAgentConfig, logger logging.ILogger) *StaticAgentDiscovery {
	agents := make([]*AgentService, 0, len(configs))

	for _, config := range configs {
		agent := &AgentService{
			Name:     config.Name,
			Endpoint: config.Endpoint,
			Tools:    config.Tools,
			Healthy:  true, // 默认假设健康，后续健康检查会更新
		}
		agents = append(agents, agent)
	}

	return &StaticAgentDiscovery{
		agents: agents,
		logger: logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (d *StaticAgentDiscovery) DiscoverAgents(ctx context.Context) ([]*AgentService, error) {
	// 对所有静态配置的agent进行健康检查
	for _, agent := range d.agents {
		agent.Healthy = d.checkHealth(ctx, agent.Endpoint)
	}

	d.logger.Info("Static agent discovery completed", "count", len(d.agents))
	return d.agents, nil
}

func (d *StaticAgentDiscovery) WatchAgents(ctx context.Context, callback func([]*AgentService)) error {
	// 静态发现暂时不支持实时监听，可以定期轮询
	go func() {
		ticker := time.NewTicker(60 * time.Second) // 每分钟检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				agents, err := d.DiscoverAgents(ctx)
				if err != nil {
					d.logger.Error("Static discovery health check failed", "error", err)
					continue
				}
				callback(agents)
			}
		}
	}()

	return nil
}

func (d *StaticAgentDiscovery) Start(ctx context.Context) error {
	// 初始健康检查
	_, err := d.DiscoverAgents(ctx)
	if err != nil {
		return err
	}

	d.logger.Info("Static agent discovery started")
	return nil
}

func (d *StaticAgentDiscovery) Stop(ctx context.Context) error {
	d.logger.Info("Static agent discovery stopped")
	return nil
}

// checkHealth 检查agent服务的健康状态
func (d *StaticAgentDiscovery) checkHealth(ctx context.Context, endpoint string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
	if err != nil {
		d.logger.Debug("Failed to create health check request", "endpoint", endpoint, "error", err)
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		d.logger.Debug("Health check failed", "endpoint", endpoint, "error", err)
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK
	d.logger.Debug("Health check completed", "endpoint", endpoint, "healthy", healthy)
	return healthy
}
