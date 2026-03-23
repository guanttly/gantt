package config

import "time"

type DiscoveryConfig struct {
	Enabled               bool          `mapstructure:"enabled" yaml:"enabled" default:"true"`
	Type                  string        `mapstructure:"type" yaml:"type" default:"nacos"` // 服务发现类型
	Nacos                 *NacosConfig  `mapstructure:"nacos" yaml:"nacos"`               // Nacos 配置
	RegisterRetries       int           `mapstructure:"register_retries" yaml:"register_retries" default:"3"`
	RegisterRetryInterval time.Duration `mapstructure:"register_retry_interval" yaml:"register_retry_interval" default:"2"`
}

type NacosConfig struct {
	Addresses       string `mapstructure:"addresses" yaml:"addresses"`                // Nacos 服务器地址
	NamespaceID     string `mapstructure:"namespaceId" yaml:"namespaceId"`            // Namespace ID
	GroupName       string `mapstructure:"groupName" yaml:"groupName"`                // 服务分组
	Username        string `mapstructure:"username" yaml:"username"`                  // 用户名
	Password        string `mapstructure:"password" yaml:"password"`                  // 密码
	TimeoutMs       uint64 `mapstructure:"timeoutMs" yaml:"timeoutMs" default:"5000"` // 超时时间
	LogDir          string `mapstructure:"logDir" yaml:"logDir"`                      // 日志目录
	CacheDir        string `mapstructure:"cacheDir" yaml:"cacheDir"`                  // 缓存目录
	LogLevel        string `mapstructure:"logLevel" yaml:"logLevel" default:"info"`   // 日志级别
	UpdateThreadNum int    `mapstructure:"updateThreadNum" yaml:"updateThreadNum"`    // 更新线程数
	NotLoadCache    bool   `mapstructure:"notLoadCache" yaml:"notLoadCache"`          // 是否加载缓存
	// ConnectTimeout time.Duration `mapstructure:"connectTimeout" yaml:"connectTimeout"` // 连接超时时间
}
