// pkg/discovery/nacos_client.go
package discovery

import (
	"fmt"
	"strconv"
	"strings"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client" // 显式导入 naming_client
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NewNacosNamingClient 根据配置创建 Nacos 命名服务客户端 (naming_client.INamingClient)
func NewNacosNamingClient(cfg *config.NacosConfig, logger logging.ILogger) (naming_client.INamingClient, error) {
	logger = logger.With("component", "NacosClient") // 添加组件标签

	// 1. 解析服务器地址
	serverConfigs := []constant.ServerConfig{}
	addrs := strings.Split(cfg.Addresses, ",")
	if len(addrs) == 0 || addrs[0] == "" {
		return nil, fmt.Errorf("Nacos 配置错误：缺少服务器地址 (addresses)")
	}
	for _, addr := range addrs {
		parts := strings.Split(strings.TrimSpace(addr), ":")
		if len(parts) != 2 {
			logger.Warn("无效的 Nacos 服务器地址格式 (应为 host:port)", "address", addr)
			continue
		}
		host := parts[0]
		port, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			logger.Warn("无效的 Nacos 服务器端口", "address", addr, "error", err)
			continue
		}
		serverConfig := *constant.NewServerConfig(host, port)
		serverConfig.GrpcPort = port + 1000 // Nacos 默认 gRPC 端口为 HTTP 端口 + 1000
		serverConfigs = append(serverConfigs, serverConfig)
	}
	if len(serverConfigs) == 0 {
		return nil, fmt.Errorf("nacos 配置错误：没有有效的服务器地址")
	}

	// 2. 创建客户端配置
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(cfg.NamespaceID),
		constant.WithTimeoutMs(cfg.TimeoutMs),
		constant.WithNotLoadCacheAtStart(!cfg.NotLoadCache), // 注意 Nacos SDK 的 NotLoadCacheAtStart 逻辑相反
		constant.WithLogDir(cfg.LogDir),
		constant.WithCacheDir(cfg.CacheDir),
		constant.WithLogLevel(cfg.LogLevel),
		constant.WithUpdateThreadNum(cfg.UpdateThreadNum),
		// 添加认证信息 (如果配置了)
		constant.WithUsername(cfg.Username),
		constant.WithPassword(cfg.Password),
	)

	// 3. 创建 NamingClient
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		logger.Error("创建 Nacos NamingClient 失败", "error", err)
		return nil, fmt.Errorf("无法创建 Nacos 命名客户端: %w", err)
	}

	return namingClient, nil
}
