// pkg/logging/logging.go
package logging

import (
	"context" // 导入 context 包
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time" // 导入 time 包用于 ReplaceAttr 示例

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	GlobalLogger = NewDefaultLogger() // 默认 logger 实例
)

type ILogger interface {
	With(args ...any) *slog.Logger

	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)

	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// NewLogger 根据提供的本包内的 Config 和 context 创建并配置一个新的 slog.Logger 实例。
// 它允许配置日志级别、格式（Text 或 JSON）以及是否添加源码位置。
// 如果日志配置为输出到文件，则会启动一个 goroutine，在提供的 context 完成时关闭该文件。
func NewLogger(ctx context.Context, cfg Config) ILogger {
	// 1. 解析日志级别 (使用 Config 的方法)
	level := cfg.GetLevel() // 使用 GetSlogLevel

	// 2. 确定日志输出目标
	var output io.Writer = os.Stdout // 默认为标准输出
	outputTarget := "stdout"         // 用于日志记录

	outputLower := strings.ToLower(cfg.Output)
	if outputLower == "stderr" {
		output = os.Stderr
		outputTarget = "stderr"
	}
	filename := dealFileName(cfg.Filename)
	// 尝试作为文件路径处理
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// 如果打开文件失败，记录警告并回退到 stdout
		slog.Warn("无法打开日志文件，将使用 stdout", "file_path", filename, "error", err)
		output = os.Stdout // 回退
		outputTarget = "stdout (fallback from file error)"
	} else {
		output = file
		outputTarget = cfg.Output

		// 启动一个 goroutine 来监听上下文的取消信号，以便关闭文件。
		go func(f *os.File, path string) {
			<-ctx.Done() // 等待上下文完成
			err := f.Close()
			if err != nil {
				GlobalLogger.Error("关闭日志文件失败（上下文取消）", "file_path", path, "error", err)
			} else {
				GlobalLogger.Info("日志文件已关闭（上下文取消）", "file_path", path)
			}
		}(file, filename)
		// 注意：如果 NewLogger 因配置热更新被多次调用，且每次都指定了不同的文件路径，
		// 旧文件对应的 goroutine 仍会持有旧文件的句柄，直到最初的上下文被取消。
		// 如果日志输出路径在热更新中频繁更改，这可能导致短期内打开较多文件句柄。
		// 理想情况下，配置热更新逻辑应负责关闭旧的日志文件句柄（如果 NewLogger 返回了它）。
		// 但根据当前要求（不返回 closer），关闭责任完全由上下文驱动。

		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100 // 默认100MB
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 3 // 默认保留3个备份
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 28 // 默认保留28天
		}
		compress := cfg.Compress
		// 根据配置决定输出目标
		outToCon := cfg.EnableConsole
		outToFile := cfg.EnableFile

		var lumberjackLogger = &lumberjack.Logger{
			Filename:   filename,   // 日志文件路径
			MaxSize:    maxSize,    // 每个日志文件最大尺寸（MB）
			MaxBackups: maxBackups, // 保留旧文件的最大个数
			MaxAge:     maxAge,     // 保留旧文件的最大天数
			Compress:   compress,   // 是否压缩/归档旧文件
		}

		// 添加控制台输出
		if outToCon {
			if outToFile {
				// 同时输出到控制台和文件
				output = io.MultiWriter(os.Stdout, lumberjackLogger)
				outputTarget = "stdout and file"
			} else {
				// 仅输出到控制台
				output = os.Stdout
				outputTarget = "stdout"
			}
		} else if outToFile {
			// 仅输出到文件
			output = lumberjackLogger
			outputTarget = "file"
		}
	}

	// 3. 配置 Handler 选项
	handlerOpts := &slog.HandlerOptions{
		AddSource: cfg.AddSource, // 是否添加源码位置
		Level:     level,         // 设置最低日志级别

		// 可选：自定义属性替换，例如修改时间戳格式
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// 将时间格式化为 RFC3339Nano
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.RFC3339Nano))
			}
			// 可以添加其他替换逻辑，比如隐藏敏感信息等
			return a
		},
	}

	// 4. 根据配置选择并创建 Handler
	var handler slog.Handler
	formatLower := strings.ToLower(cfg.Format) // 转换为小写以进行不区分大小写的比较

	// 使用 Debug 级别直接通过 slog 记录初始化信息，因为 logger 实例还没完全创建
	logFields := []any{"level", level.String(), "add_source", cfg.AddSource, "output", outputTarget}

	if formatLower == FormatJSON { // 使用本包内定义的常量
		handler = slog.NewJSONHandler(output, handlerOpts)
		slog.Debug("使用 JSON 格式初始化日志记录器", logFields...)
	} else { // 默认为 Text 格式
		if formatLower != FormatText && formatLower != "" { // 使用本包内定义的常量
			slog.Warn("无效的日志格式配置，将使用默认格式 TEXT", "configured_format", cfg.Format)
		}
		handler = slog.NewTextHandler(output, handlerOpts)
		slog.Debug("使用 TEXT 格式初始化日志记录器", logFields...)
	}

	// 5. 创建 Logger 实例
	logger := slog.New(handler)

	// 6. 返回创建的 Logger 实例
	// 注意：这里不调用 slog.SetDefault(logger)，让调用者（通常是 main.go）
	// 决定是否将其设置为全局默认 logger。
	return logger
}

// NewDefaultLogger 提供一个快速获取默认配置 Logger 的方法 (INFO, Text, No Source, stdout)
// 在配置加载完成前或简单场景下可以使用。
// 推荐使用 NewLogger(cfg) 进行标准初始化。
func NewDefaultLogger() ILogger {
	var output io.Writer = os.Stdout // 默认为标准输出
	handlerOpts := &slog.HandlerOptions{
		AddSource: false,          // 是否添加源码位置
		Level:     slog.LevelInfo, // 设置最低日志级别

		// 可选：自定义属性替换，例如修改时间戳格式
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// 将时间格式化为 RFC3339Nano
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.RFC3339Nano))
			}
			// 可以添加其他替换逻辑，比如隐藏敏感信息等
			return a
		},
	}

	// 4. 根据配置选择并创建 Handler
	var handler slog.Handler
	formatLower := strings.ToLower(FormatText) // 转换为小写以进行不区分大小写的比较

	// 使用 Debug 级别直接通过 slog 记录初始化信息，因为 logger 实例还没完全创建
	logFields := []any{"level", slog.LevelInfo.String(), "add_source", false, "output", "stdout"}

	if formatLower == FormatJSON { // 使用本包内定义的常量
		handler = slog.NewJSONHandler(output, handlerOpts)
		slog.Debug("使用 JSON 格式初始化日志记录器", logFields...)
	} else { // 默认为 Text 格式
		if formatLower != FormatText && formatLower != "" { // 使用本包内定义的常量
			slog.Warn("无效的日志格式配置，将使用默认格式 TEXT", "configured_format", FormatText)
		}
		handler = slog.NewTextHandler(output, handlerOpts)
		slog.Debug("使用 TEXT 格式初始化日志记录器", logFields...)
	}

	// 5. 创建 Logger 实例
	logger := slog.New(handler)

	// 6. 返回创建的 Logger 实例
	// 注意：这里不调用 slog.SetDefault(logger)，让调用者（通常是 main.go）
	// 决定是否将其设置为全局默认 logger。
	return logger
}

func dealFileName(filename string) string {
	if filename == "" || strings.HasSuffix(filename, "/") || strings.HasSuffix(filename, "\\") {
		filename = filename + "app.log" // 默认日志文件名
	}
	// 如果没有目录分隔符，默认放在 logs/ 目录下
	if !strings.Contains(filename, "/") && !strings.Contains(filename, "\\") {
		filename = filepath.Join("logs", filename)
	}
	// 确保日志目录存在
	logDir := filepath.Dir(filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		GlobalLogger.Error("创建日志目录失败", "dir", logDir, "error", err)
		filename = filepath.Base(filename) // 回退到当前目录
	}
	// 如果文件名没有扩展名，默认添加 .log
	if filepath.Ext(filename) == "" {
		filename = filename + ".log"
	}
	return filename
}
