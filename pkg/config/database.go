package config

import "time"

type DatabaseConfig struct {
	MySQL    *DBConfig       `mapstructure:"mysql" yaml:"mysql"`
	Redis    *RedisConfig    `mapstructure:"redis" yaml:"redis"`
	ObjStore *ObjStore       `mapstructure:"obj_store" yaml:"obj_store"` // 对象存储配置
	VectorDB *VectorDBConfig `mapstructure:"vector_db" yaml:"vector_db"`
}

// DBConfig 定义了数据库连接所需的配置参数。
// 使用 mapstructure 标签是为了方便从 Viper 等库加载配置。
type DBConfig struct {
	Host            string        `mapstructure:"host" yaml:"host"`                             // 数据库主机名或 IP (可被 DB_HOST 环境变量覆盖)
	Port            int           `mapstructure:"port" yaml:"port"`                             // 数据库端口 (可被 DB_PORT 环境变量覆盖)
	Username        string        `mapstructure:"username" yaml:"username"`                     // 用户名 (可被 DB_USER 环境变量覆盖)
	Password        string        `mapstructure:"password" yaml:"password"`                     // 密码 (可被 DB_PASSWORD 环境变量覆盖)
	DBName          string        `mapstructure:"dbname" yaml:"dbname"`                         // 数据库名称 (可被 DB_NAME 环境变量覆盖)
	Charset         string        `mapstructure:"charset" yaml:"charset"`                       // 字符集, 例如 "utf8mb4"
	ParseTime       bool          `mapstructure:"parse_time" yaml:"parse_time"`                 // 是否解析 time.Time 类型, 推荐 true
	Timeout         time.Duration `mapstructure:"timeout" yaml:"timeout"`                       // 连接尝试的超时时间, 例如 "5s"
	MaxOpenConns    int           `mapstructure:"max_open_conns" yaml:"max_open_conns"`         // 连接池最大打开连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`         // 连接池最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`   // 连接最大生存时间, 例如 "1h"
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time"` // 连接最大空闲时间, 例如 "5m"
	// DSN string `mapstructure:"dsn" yaml:"dsn"` // 可选：如果提供，则直接使用完整的 DSN 字符串
}

type RedisConfig struct {
	Host     string `mapstructure:"host" yaml:"host"`         // Redis 主机名或 IP (可被 REDIS_HOST 环境变量覆盖)
	Port     int    `mapstructure:"port" yaml:"port"`         // Redis 端口 (可被 REDIS_PORT 环境变量覆盖)
	Username string `mapstructure:"username" yaml:"username"` // // Redis 用户名 (可被 REDIS_USER 环境变量覆盖)
	Password string `mapstructure:"password" yaml:"password"` // Redis 密码 (可被 REDIS_PASSWORD 环境变量覆盖)
	DB       int    `mapstructure:"db" yaml:"db"`             // Redis 数据库索引 (可被 REDIS_DB_INDEX 环境变量覆盖)
}

type ObjStore struct {
	BucketName string       `mapstructure:"bucket_name" yaml:"bucket_name"` // 对象存储桶名称 (可被 BUCKET_NAME 环境变量覆盖)
	MinIO      *MinioConfig `mapstructure:"minio" yaml:"minio"`             // MinIO 配置
}

// MinioConfig 用于在 main.go 中创建 MinIO 客户端的配置
type MinioConfig struct {
	Endpoint        string `mapstructure:"endpoint" yaml:"endpoint"`                   // MinIO 端点 (可被 MINIO_ENDPOINT 环境变量覆盖)
	AccessKeyID     string `mapstructure:"access_key_id" yaml:"access_key_id"`         // Access Key ID (可被 MINIO_ACCESS_KEY 环境变量覆盖)
	SecretAccessKey string `mapstructure:"secret_access_key" yaml:"secret_access_key"` // Secret Access Key (可被 MINIO_SECRET_KEY 环境变量覆盖)
	UseSSL          bool   `mapstructure:"use_ssl" yaml:"use_ssl"`                     // 是否使用 SSL (可被 MINIO_USE_SSL 环境变量覆盖)
}
