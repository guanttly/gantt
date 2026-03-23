package config

import (
	"os"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"

	"github.com/spf13/viper"
)

const ConfigName = "common" // 配置文件名

type Config struct {
	Host      string           `mapstructure:"host" yaml:"host"`
	Log       *logging.Config  `mapstructure:"log" yaml:"log"`
	Database  *DatabaseConfig  `mapstructure:"database" yaml:"database"`
	Discovery *DiscoveryConfig `mapstructure:"discovery" yaml:"discovery"`
	Ports     *Ports           `mapstructure:"ports" yaml:"ports"`
	AI        *AIConfig        `mapstructure:"ai" yaml:"ai"` // 新增 AI 配置
	MCP       *MCPConfig       `mapstructure:"mcp" yaml:"mcp"`

	Timeout HttpTimeout `mapstructure:"timeout" yaml:"timeout"` // HTTP 超时配置
	License *License    `mapstructure:"license" yaml:"license"` // 许可证配置
}

type MCPConfig struct {
	DiscoveryGroupName string `mapstructure:"discovery_group_name" yaml:"discovery_group_name"` // 服务发现组名
	ClientTimeout      int    `mapstructure:"client_timeout" yaml:"client_timeout"`             // MCP客户端超时（秒）
	HealthCheckTimeout int    `mapstructure:"health_check_timeout" yaml:"health_check_timeout"` // 健康检查超时（秒）
}

type HttpTimeout struct {
	IdleTimeout       int `mapstructure:"idle_timeout" yaml:"idle_timeout"`               // 空闲连接超时时间(Seconds)
	ReadTimeout       int `mapstructure:"read_timeout" yaml:"read_timeout"`               // 读取超时时间(Seconds)
	WriteTimeout      int `mapstructure:"write_timeout" yaml:"write_timeout"`             // 写入超时时间(Seconds)
	ReadHeaderTimeout int `mapstructure:"read_header_timeout" yaml:"read_header_timeout"` // 读取请求头超时时间(Seconds)
}

type License struct {
	URL        string `mapstructure:"url" yaml:"url"`                 // 许可证验证服务的URL
	ServerName string `mapstructure:"server_name" yaml:"server_name"` // 许可证服务器名称
}

type Ports struct {
	GRPCPort int `mapstructure:"grpc_port" yaml:"grpc_port" default:"50051"`
	HTTPPort int `mapstructure:"http_port" yaml:"http_port" default:"8080"`
}

func Load(path string) (*Config, error) {
	// 判断配置文件路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err // 返回错误
	}

	// 创建一个空的 Config 实例
	var cfg Config

	// 使用 Viper 加载配置文件
	v := viper.New()            // Renamed to avoid conflict with package name
	v.AddConfigPath(path)       // 设置配置文件目录
	v.SetConfigName(ConfigName) // 设置配置文件名（不包含扩展名）
	v.SetConfigType("yaml")     // 设置配置文件类型

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, err // 返回错误
	}

	// 解析配置到 Config 结构体
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// --- 优先使用环境变量覆盖配置 ---
	if cfg.Database == nil {
		cfg.Database = &DatabaseConfig{}
	}
	if cfg.Database.MySQL == nil {
		cfg.Database.MySQL = &DBConfig{}
	}
	if cfg.Database.ObjStore == nil {
		cfg.Database.ObjStore = &ObjStore{
			MinIO:      &MinioConfig{},
			BucketName: utils.OverrideFromEnv("DEFAULT_BUCKET", "default-bucket"),
		}
	}
	if cfg.Database.ObjStore.MinIO == nil {
		cfg.Database.ObjStore.MinIO = &MinioConfig{}
	}
	if cfg.Database.VectorDB == nil {
		cfg.Database.VectorDB = &VectorDBConfig{}
	}
	if cfg.Database.VectorDB.Milvus == nil { // Ensure Milvus config is initialized
		cfg.Database.VectorDB.Milvus = &MilvusConfig{}
	}

	if cfg.Discovery == nil {
		cfg.Discovery = &DiscoveryConfig{}
	}
	if cfg.Discovery.Nacos == nil {
		cfg.Discovery.Nacos = &NacosConfig{}
	}

	if cfg.AI == nil {
		cfg.AI = &AIConfig{}
	}

	if cfg.AI.Ollama == nil {
		cfg.AI.Ollama = &ProviderConfig{}
	}

	if cfg.AI.OpenAI == nil {
		cfg.AI.OpenAI = &ProviderConfig{}
	}

	if cfg.AI.Bailian == nil {
		cfg.AI.Bailian = &ProviderConfig{}
	}

	// MCP
	if cfg.MCP == nil {
		cfg.MCP = &MCPConfig{
			DiscoveryGroupName: "mcp-server",
			ClientTimeout:      300, // 默认5分钟
			HealthCheckTimeout: 10,  // 默认10秒
		}
	}

	// Override HOST config
	cfg.Host = utils.OverrideFromEnv("KG_HOST", cfg.Host)

	// Override MySQL config
	cfg.Database.MySQL.Host = utils.OverrideFromEnv("DB_HOST", cfg.Database.MySQL.Host)
	cfg.Database.MySQL.Port = utils.OverrideIntFromEnv("DB_PORT", cfg.Database.MySQL.Port)
	cfg.Database.MySQL.Username = utils.OverrideFromEnv("DB_USER", cfg.Database.MySQL.Username)
	cfg.Database.MySQL.Password = utils.OverrideFromEnv("DB_PASSWORD", cfg.Database.MySQL.Password)
	cfg.Database.MySQL.DBName = utils.OverrideFromEnv("DB_NAME", cfg.Database.MySQL.DBName)

	// Override ObjStore config
	cfg.Database.ObjStore.MinIO.Endpoint = utils.OverrideFromEnv("MINIO_ENDPOINT", cfg.Database.ObjStore.MinIO.Endpoint)
	cfg.Database.ObjStore.MinIO.AccessKeyID = utils.OverrideFromEnv("MINIO_ACCESS_KEY", cfg.Database.ObjStore.MinIO.AccessKeyID)
	cfg.Database.ObjStore.MinIO.SecretAccessKey = utils.OverrideFromEnv("MINIO_SECRET_KEY", cfg.Database.ObjStore.MinIO.SecretAccessKey)
	cfg.Database.ObjStore.MinIO.UseSSL = utils.OverrideBoolFromEnv("MINIO_USE_SSL", cfg.Database.ObjStore.MinIO.UseSSL)

	// Override MilvusClient config
	cfg.Database.VectorDB.Milvus.Address = utils.OverrideFromEnv("MILVUS_ADDRESS", cfg.Database.VectorDB.Milvus.Address)

	// Override Nacos config
	cfg.Discovery.Nacos.Addresses = utils.OverrideFromEnv("NACOS_SERVER_ADDR", cfg.Discovery.Nacos.Addresses)
	cfg.Discovery.Nacos.NamespaceID = utils.OverrideFromEnv("NACOS_NAMESPACE_ID", cfg.Discovery.Nacos.NamespaceID)
	cfg.Discovery.Nacos.GroupName = utils.OverrideFromEnv("NACOS_GROUP_NAME", cfg.Discovery.Nacos.GroupName)
	cfg.Discovery.Nacos.Username = utils.OverrideFromEnv("NACOS_USERNAME", cfg.Discovery.Nacos.Username)
	cfg.Discovery.Nacos.Password = utils.OverrideFromEnv("NACOS_PASSWORD", cfg.Discovery.Nacos.Password)
	// --- End Environment Variable Overrides ---

	// --- 新增：覆盖 AI 配置环境变量 ---
	// Ollama
	cfg.AI.Ollama.BaseURL = utils.OverrideFromEnv("AI_OLLAMA_BASE_URL", cfg.AI.Ollama.BaseURL)

	// OpenAI
	cfg.AI.OpenAI.BaseURL = utils.OverrideFromEnv("AI_OPENAI_BASE_URL", cfg.AI.OpenAI.BaseURL)
	cfg.AI.OpenAI.APIKey = utils.OverrideFromEnv("AI_OPENAI_API_KEY", cfg.AI.OpenAI.APIKey)

	// Bailian
	cfg.AI.Bailian.BaseURL = utils.OverrideFromEnv("AI_BAILIAN_BASE_URL", cfg.AI.Bailian.BaseURL)
	cfg.AI.Bailian.APIKey = utils.OverrideFromEnv("AI_BAILIAN_API_KEY", cfg.AI.Bailian.APIKey)
	// --- AI 配置环境变量覆盖结束 ---

	// 处理DEBUG逻辑

	if mode := os.Getenv("APP_ENV"); mode == "DEBUG" {
		// host := os.Getenv("SERVER_HOST")
		host := os.Getenv("APP_HOST")
		if host != "" {
			// 修改Nacos地址
			cfg.Discovery.Nacos.Addresses = host + ":8848"
		}
	}

	// 返回解析后的配置
	return &cfg, nil
}

func CreateEmptyConfig() *Config {
	return &Config{
		Host:  "",
		Log:   &logging.Config{Level: "info"},
		Ports: &Ports{},
		AI:    &AIConfig{},
	}
}
