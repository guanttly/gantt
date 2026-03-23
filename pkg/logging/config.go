package logging

import (
	"log/slog"
	"strings"
)

// 定义日志级别常量 (用于配置文件, 大小写不敏感).
const (
	LevelDebug = "debug" // 对应 slog.LevelDebug
	LevelInfo  = "info"  // 对应 slog.LevelInfo
	LevelWarn  = "warn"  // 对应 slog.LevelWarn
	LevelError = "error" // 对应 slog.LevelError
)

// 定义日志格式常量 (用于配置文件, 大小写不敏感).
const (
	FormatText = "text" // 文本格式
	FormatJSON = "json" // JSON 格式
)

// Config 定义了日志记录的配置参数。
type Config struct {
	Level     string `mapstructure:"level"`      // 例如 "debug", "info", "warn", "error"
	Format    string `mapstructure:"format"`     // 例如 "text", "json"
	AddSource bool   `mapstructure:"add_source"` // 是否在日志中添加源码位置信息
	Output    string `mapstructure:"output"`     // 日志输出目标: "stdout", "stderr"
	Filename  string `mapstructure:"filename"`   // 文件路径
	// 可以根据需要添加其他字段，如 OutputPath, MaxSize, MaxBackups, MaxAge, Compress 等
	// 以下是zap包配置
	MaxSize       int  `mapstructure:"max_size"`       // 单个日志文件的最大大小 (MB)
	MaxBackups    int  `mapstructure:"max_backups"`    // 保留的旧日志文件数量
	MaxAge        int  `mapstructure:"max_age"`        // 日志文件保留天数
	Compress      bool `mapstructure:"compress"`       // 是否压缩旧日志文件
	EnableConsole bool `mapstructure:"enable_console"` // 是否启用控制台输出
	EnableFile    bool `mapstructure:"enable_file"`    // 是否启用文件输出
}

// GetSlogLevel 将字符串级别的日志级别转换为 slog.Level。
func (c Config) GetLevel() slog.Level {
	switch strings.ToLower(c.Level) { // 转换为小写以进行不区分大小写的比较
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		// 如果配置了无效级别，可以记录一个默认slog警告并返回INFO级别
		// 注意：此时可能还没有完全初始化的logger实例，直接使用slog.Warn是合适的
		slog.Warn("无效的日志级别配置，将使用默认级别 INFO", "configured_level", c.Level)
		return slog.LevelInfo // 默认为 Info 级别
	}
}

// ConfigsEqual 比较两个日志配置是否相等。
func ConfigsEqual(c1, c2 Config) bool {
	return strings.EqualFold(c1.Level, c2.Level) &&
		strings.EqualFold(c1.Format, c2.Format) &&
		c1.AddSource == c2.AddSource &&
		strings.EqualFold(c1.Output, c2.Output)
}
