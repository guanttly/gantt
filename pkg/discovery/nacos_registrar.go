// pkg/discovery/nacos_registrar.go
package discovery

import (
	"fmt"
	"jusha/mcp/pkg/logging"
	"strings" // 导入 strings 包
	"time"    // 导入 time 包

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Registrar 定义了服务注册和注销的接口 (也可以放在 discovery.go)
type IRegistrar interface {
	// Register 注册一个服务实例到服务发现中心。
	// instanceID 通常应该是唯一的，例如 host:port 或 UUID。
	// metadata 是附加的元数据。
	Register(serviceName, instanceID, host string, port int, metadata map[string]string) error

	// Deregister 从服务发现中心注销一个服务实例。
	// Nacos 通常需要 serviceName, host, port, group 来唯一确定实例。
	Deregister(serviceName, host string, port int) error // InstanceID 在 Nacos 中不直接用于注销
}

// nacosRegistrar 实现了 Registrar 接口，用于 Nacos
type nacosRegistrar struct {
	client                naming_client.INamingClient
	groupName             string // 默认或配置的服务分组
	logger                logging.ILogger
	registerRetries       int           // 注册重试次数
	registerRetryInterval time.Duration // 注册重试间隔
}

// NewNacosRegistrar 创建一个新的 Nacos Registrar 实例
// 添加了重试相关的参数
func NewNacosRegistrar(client naming_client.INamingClient, defaultGroupName string, logger logging.ILogger, retries int, retryInterval time.Duration) IRegistrar {
	if defaultGroupName == "" {
		defaultGroupName = "DEFAULT_GROUP" // Nacos 默认分组
	}
	if retries <= 0 {
		retries = 3 // 默认重试次数
	}
	if retryInterval <= 0 {
		retryInterval = 2 * time.Second // 默认重试间隔
	}
	return &nacosRegistrar{
		client:                client,
		groupName:             defaultGroupName,
		logger:                logger.With("component", "NacosRegistrar"),
		registerRetries:       retries,
		registerRetryInterval: retryInterval,
	}
}

// Register 将服务实例注册到 Nacos，包含重试逻辑
func (r *nacosRegistrar) Register(serviceName, instanceID, host string, port int, metadata map[string]string) error {
	r.logger.Debug("准备向 Nacos 注册服务实例...",
		"serviceName", serviceName,
		"instanceID", instanceID,
		"host", host,
		"port", port,
		"group", r.groupName,
		"metadata", metadata,
		"retries", r.registerRetries,
		"retryInterval", r.registerRetryInterval,
	)

	registerParam := vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceName,
		GroupName:   r.groupName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    metadata,
	}

	var lastErr error
	for attempt := 0; attempt <= r.registerRetries; attempt++ {
		if attempt > 0 {
			r.logger.Info("等待后重试注册...", "attempt", attempt, "interval", r.registerRetryInterval)
			time.Sleep(r.registerRetryInterval)
		}

		r.logger.Debug("调用 Nacos SDK RegisterInstance", "attempt", attempt, "param", fmt.Sprintf("%+v", registerParam))
		success, err := r.client.RegisterInstance(registerParam)
		lastErr = err // 保存最后一次错误

		if err == nil {
			if success {
				r.logger.Debug("Nacos 服务实例注册成功", "serviceName", serviceName, "host", host, "port", port, "attempt", attempt)
				return nil // 注册成功，直接返回
			}
			// 虽然没有错误，但注册未成功？（根据 SDK 行为可能需要检查）
			r.logger.Warn("Nacos 服务实例注册调用成功，但返回值为 false", "attempt", attempt, "param", fmt.Sprintf("%+v", registerParam))
			// 也许也需要重试？或者认为这是失败？暂时按失败处理并重试
			lastErr = fmt.Errorf("注册 Nacos 实例未成功 (success=false)")
			continue // 继续尝试
		}

		// 检查是否是连接错误，如果是则重试
		if strings.Contains(err.Error(), "client not connected") || strings.Contains(err.Error(), "STARTING") {
			r.logger.Warn("Nacos 服务实例注册失败：客户端尚未连接或仍在启动中。将进行重试...",
				"attempt", attempt,
				"error", err,
				"param", fmt.Sprintf("%+v", registerParam),
			)
			// 继续循环进行重试
		} else {
			// 其他类型的错误，直接返回失败
			r.logger.Error("Nacos 服务实例注册遇到不可恢复错误", "error", err, "param", fmt.Sprintf("%+v", registerParam))
			return fmt.Errorf("注册 Nacos 实例失败: %w", err)
		}
	}

	// 所有重试次数用尽后仍然失败
	r.logger.Error("Nacos 服务实例注册失败，已达到最大重试次数", "error", lastErr, "param", fmt.Sprintf("%+v", registerParam))
	return fmt.Errorf("注册 Nacos 实例失败，已达到最大重试次数: %w", lastErr)
}

// Deregister 从 Nacos 注销服务实例
func (r *nacosRegistrar) Deregister(serviceName, host string, port int) error {
	r.logger.Info("从 Nacos 注销服务实例...",
		"serviceName", serviceName,
		"host", host,
		"port", port,
		"group", r.groupName,
	)

	// Nacos 注销实例需要的参数
	deregisterParam := vo.DeregisterInstanceParam{
		Ip:          host,         // 实例 IP
		Port:        uint64(port), // 实例端口
		ServiceName: serviceName,  // 服务名
		GroupName:   r.groupName,  // 分组名
		Ephemeral:   true,         // **重要**：必须与注册时的 Ephemeral 值一致！
	}

	success, err := r.client.DeregisterInstance(deregisterParam)
	if err != nil {
		r.logger.Error("Nacos 服务实例注销失败", "error", err, "param", fmt.Sprintf("%+v", deregisterParam))
		return fmt.Errorf("注销 Nacos 实例失败: %w", err)
	}
	if !success {
		r.logger.Warn("Nacos 服务实例注销调用成功，但返回值为 false", "param", fmt.Sprintf("%+v", deregisterParam))
		// return fmt.Errorf("注销 Nacos 实例未成功 (success=false)")
	}

	r.logger.Info("Nacos 服务实例注销成功", "serviceName", serviceName, "host", host, "port", port)
	return nil
}
