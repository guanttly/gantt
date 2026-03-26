package observability

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"

	"time"
)

// GormLogger 将 GORM 日志集成到 zap。
type GormLogger struct {
	zapLogger *zap.Logger
	level     logger.LogLevel
}

// NewGormLogger 创建 GORM 用的 zap 日志适配器。
func NewGormLogger(zapLogger *zap.Logger) *GormLogger {
	return &GormLogger{
		zapLogger: zapLogger.Named("gorm"),
		level:     logger.Info,
	}
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *GormLogger) Info(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		l.zapLogger.Sugar().Infof(msg, data...)
	}
}

func (l *GormLogger) Warn(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		l.zapLogger.Sugar().Warnf(msg, data...)
	}
}

func (l *GormLogger) Error(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		l.zapLogger.Sugar().Errorf(msg, data...)
	}
}

func (l *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// 记录 Prometheus 指标
	Metrics.DBQueryDuration.WithLabelValues("query").Observe(elapsed.Seconds())

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	if err != nil {
		l.zapLogger.Error("SQL Error", append(fields, zap.Error(err))...)
		return
	}

	if elapsed > 200*time.Millisecond {
		l.zapLogger.Warn("Slow SQL", fields...)
		return
	}

	l.zapLogger.Debug("SQL", fields...)
}
