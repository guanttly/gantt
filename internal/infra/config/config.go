// Package config 提供配置加载功能，支持 YAML 文件 + 环境变量覆盖。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用的顶层配置结构体。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	AI       AIConfig       `mapstructure:"ai"`
	Admin    AdminConfig    `mapstructure:"admin"`
}

// ServerConfig HTTP 服务器配置。
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Mode            string        `mapstructure:"mode"` // development / production
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig MySQL 数据库配置。
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
}

// DSN 返回 MySQL 连接字符串。
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

// RedisConfig Redis 配置。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig 日志配置。
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // json / console
	Output string `mapstructure:"output"` // stdout / 文件路径
}

// AIConfig AI 全局配置。
type AIConfig struct {
	Enabled         bool              `mapstructure:"enabled"`
	DefaultProvider string            `mapstructure:"default_provider"`
	OpenAI          *AIProviderConfig `mapstructure:"openai"`
	Bailian         *AIProviderConfig `mapstructure:"bailian"`
	Ollama          *AIProviderConfig `mapstructure:"ollama"`
	Quota           AIQuotaConfig     `mapstructure:"quota"`
}

// AIProviderConfig 单个 AI Provider 的配置。
type AIProviderConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	APIKey  string        `mapstructure:"api_key"`
	BaseURL string        `mapstructure:"base_url"`
	Model   string        `mapstructure:"model"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// AIQuotaConfig AI 配额配置。
type AIQuotaConfig struct {
	DefaultMonthlyTokens int `mapstructure:"default_monthly_tokens"`
}

// JWTConfig JWT 认证配置。
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

// AdminConfig 默认平台管理员配置。
type AdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

// Load 加载配置文件并绑定环境变量。
// 配置文件搜索顺序：./config/ → ./ → /etc/gantt/
// 环境变量前缀：GANTT_，例如 GANTT_SERVER_PORT=9090
func Load() (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/gantt")

	// 环境变量覆盖
	v.SetEnvPrefix("GANTT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件（文件不存在时使用默认值 + 环境变量）
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 文件不存在，使用默认值 + 环境变量
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "development")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "10s")

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.name", "gantt_saas")
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 3600)

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Log
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	// AI
	v.SetDefault("ai.enabled", false)

	// JWT
	v.SetDefault("jwt.secret", "gantt-saas-secret-change-me")
	v.SetDefault("jwt.access_token_ttl", "2h")
	v.SetDefault("jwt.refresh_token_ttl", "168h")
	v.SetDefault("jwt.issuer", "gantt-saas")

	// Admin
	v.SetDefault("admin.username", "admin")
	v.SetDefault("admin.email", "admin@gantt.local")
	v.SetDefault("admin.password", "Admin@123456")
}
