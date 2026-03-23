package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"jusha/mcp/pkg/logging"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

const default_cluster = "DEFAULT"

// NacosAgentDiscovery 基于Nacos的智能体发现实现
type NacosAgentDiscovery struct {
	namingClient naming_client.INamingClient
	logger       logging.ILogger
	groupName    string

	mu        sync.RWMutex
	agents    []*AgentService
	callbacks []func([]*AgentService)
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewNacosAgentDiscovery 创建Nacos智能体发现服务
func NewNacosAgentDiscovery(namingClient naming_client.INamingClient, logger logging.ILogger, groupName string) *NacosAgentDiscovery {
	if groupName == "" {
		groupName = "mcp-server"
	}

	return &NacosAgentDiscovery{
		namingClient: namingClient,
		logger:       logger,
		groupName:    groupName,
		stopChan:     make(chan struct{}),
	}
}

func (d *NacosAgentDiscovery) DiscoverAgents(ctx context.Context) ([]*AgentService, error) {
	// 获取所有agent服务实例
	services, err := d.namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: d.groupName,
		PageNo:    1,
		PageSize:  100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get services from Nacos: %w", err)
	}

	var agents []*AgentService

	// 遍历所有服务
	for _, serviceName := range services.Doms {
		// d.namingClient.Subscribe(&vo.SubscribeParam{
		// 	ServiceName:       serviceName,
		// 	SubscribeCallback: sc.subscribeCallback,
		// 	GroupName:         d.groupName,
		// })

		instances, err := d.namingClient.SelectInstances(vo.SelectInstancesParam{
			Clusters:    []string{default_cluster},
			ServiceName: serviceName,
			GroupName:   d.groupName,
			HealthyOnly: true,
		})
		if err != nil {
			d.logger.Warn("Failed to get healthy instances", "service", serviceName, "error", err)
			continue
		}

		// 处理每个健康实例
		for _, instance := range instances {
			agent, err := d.getToolsFromInstance(serviceName, &instance)
			if err != nil {
				d.logger.Warn("Failed to parse agent instance", "service", serviceName, "error", err)
				continue
			}

			agents = append(agents, agent)
		}
	}

	// 更新本地缓存
	d.mu.Lock()
	d.agents = agents
	d.mu.Unlock()

	d.logger.Debug("Discovered agents via Nacos", "count", len(agents))
	return agents, nil
}

func (d *NacosAgentDiscovery) WatchAgents(ctx context.Context, callback func([]*AgentService)) error {
	d.mu.Lock()
	d.callbacks = append(d.callbacks, callback)
	d.mu.Unlock()

	// 启动监听服务变化的goroutine
	d.wg.Add(1)
	go d.watchLoop(ctx)

	return nil
}

func (d *NacosAgentDiscovery) Start(ctx context.Context) error {
	// 首次发现
	_, err := d.DiscoverAgents(ctx)
	if err != nil {
		return fmt.Errorf("initial discovery failed: %w", err)
	}

	return nil
}

func (d *NacosAgentDiscovery) Stop(ctx context.Context) error {
	close(d.stopChan)
	d.wg.Wait()

	d.logger.Info("Nacos agent discovery stopped")
	return nil
}

// parseAgentFromInstance 从Nacos实例解析智能体信息
func (d *NacosAgentDiscovery) getToolsFromInstance(serviceName string, instance *model.Instance) (*AgentService, error) {
	agent := &AgentService{
		Name:     serviceName,
		Endpoint: fmt.Sprintf("http://%s:%d", instance.Ip, instance.Port),
		Healthy:  instance.Healthy,
	}

	// 从metadata中解析工具列表
	if toolsStr, exists := instance.Metadata["tools"]; exists {
		var tools []string
		if err := json.Unmarshal([]byte(toolsStr), &tools); err == nil {
			agent.Tools = tools
		} else {
			// 如果JSON解析失败，尝试按逗号分割
			agent.Tools = strings.Split(toolsStr, ",")
			for i, tool := range agent.Tools {
				agent.Tools[i] = strings.TrimSpace(tool)
			}
		}
	}

	return agent, nil
}

// watchLoop 监听服务变化的后台循环
func (d *NacosAgentDiscovery) watchLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			agents, err := d.DiscoverAgents(ctx)
			if err != nil {
				d.logger.Error("Failed to discover agents in watch loop", "error", err)
				continue
			}

			// 通知所有回调函数
			d.mu.RLock()
			callbacks := make([]func([]*AgentService), len(d.callbacks))
			copy(callbacks, d.callbacks)
			d.mu.RUnlock()

			for _, callback := range callbacks {
				go callback(agents)
			}
		}
	}
}
