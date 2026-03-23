package config

import "time"

type VectorDBConfig struct {
	Milvus *MilvusConfig `mapstructure:"milvus" yaml:"milvus"` // Milvus 配置
}

// MilvusConfig 用于在 main.go 中创建 Milvus 客户端的配置
type MilvusConfig struct {
	Address        string        `mapstructure:"address" yaml:"address"`                 // 例如 "localhost:19530"
	Username       string        `mapstructure:"username" yaml:"username"`               // Milvus 2.2.9+
	Password       string        `mapstructure:"password" yaml:"password"`               // Milvus 2.2.9+
	APIKey         string        `mapstructure:"api_key" yaml:"api_key"`                 // Milvus Cloud / Zilliz Cloud API Key
	UseSSL         bool          `mapstructure:"use_ssl" yaml:"use_ssl"`                 // 是否使用 TLS/SSL (通常在云环境或特定部署中需要)
	Database       string        `mapstructure:"database" yaml:"database"`               // Milvus 2.2.1+ 支持多数据库，指定要连接的数据库名，默认为 "" 或 "default"
	ConnectTimeout time.Duration `mapstructure:"connect_timeout" yaml:"connect_timeout"` // 连接超时时间, e.g., 5*time.Second
}
