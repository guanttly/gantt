// pkg/adapter/mysql.go
package adapter

import (
	"context"
	"fmt"
	"time"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	// 导入 gorm 库 和 mysql 驱动
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewMySQLConnection 根据提供的配置，建立一个新的到 MySQL 数据库的连接池。
// 它使用 context 来控制连接和初始 PING 的超时。
func NewMySQLConnection(ctx context.Context, cfg *config.DBConfig, logger logging.ILogger) (*gorm.DB, error) {
	// 1. 构造数据源名称 (DSN)
	// 格式: username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	var dsn string
	if cfg.Host == "" || cfg.Port == 0 || cfg.Username == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("数据库配置不完整：缺少 Host (%s), Port (%d), Username (%s), 或 DBName (%s)", cfg.Host, cfg.Port, cfg.Username, cfg.DBName)
	}

	// 设置默认值
	charset := "utf8mb4"
	if cfg.Charset != "" {
		charset = cfg.Charset
	}
	parseTime := "True"
	if !cfg.ParseTime {
		parseTime = "False" // 通常应该总是 True
	}

	// 拼接 DSN
	// 注意：loc=Local 假定应用服务器的时区与数据库服务器一致。如果不同，应指定正确的时区，如 &loc=UTC
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=Local",
		cfg.Username,
		cfg.Password, // 密码可以为空字符串
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		charset,
		parseTime,
	)

	logger.InfoContext(ctx, "正在连接到 MySQL 数据库...", "host", cfg.Host, "port", cfg.Port, "dbname", cfg.DBName, "user", cfg.Username)

	// 2. 使用 gorm.Open 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 可在此处配置 GORM 选项，例如日志记录器
		// Logger: logger.Default.LogMode(logger.Info), // 根据需要配置 GORM 日志
	})
	if err != nil {
		logger.ErrorContext(ctx, "数据库连接失败", "error", err)
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	// GORM 建议使用 PingContext 验证连接
	sqlDB, err := db.DB()
	if err != nil {
		logger.ErrorContext(ctx, "获取底层 sql.DB 失败", "error", err)
		return nil, fmt.Errorf("无法获取数据库连接实例: %w", err)
	}

	// 使用传入的 context 控制 Ping 的超时
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // 设置 Ping 的超时时间
	defer cancel()
	if err = sqlDB.PingContext(pingCtx); err != nil {
		logger.ErrorContext(ctx, "数据库 Ping 失败", "error", err)
		// 尝试关闭已打开但无法 Ping 通的连接
		_ = sqlDB.Close()
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	logger.InfoContext(ctx, "数据库连接成功并通过 Ping 验证")

	// 3. 配置连接池参数 (通过底层的 *sql.DB)
	maxOpen := 25 // 默认最大打开连接数
	if cfg.MaxOpenConns > 0 {
		maxOpen = cfg.MaxOpenConns
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	logger.DebugContext(ctx, "设置数据库 MaxOpenConns", "value", maxOpen)

	maxIdle := 5 // 默认最大空闲连接数
	if cfg.MaxIdleConns > 0 {
		maxIdle = cfg.MaxIdleConns
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	logger.DebugContext(ctx, "设置数据库 MaxIdleConns", "value", maxIdle)

	maxLifetime := time.Hour // 默认连接最大生存时间
	if cfg.ConnMaxLifetime > 0 {
		maxLifetime = cfg.ConnMaxLifetime
	}
	sqlDB.SetConnMaxLifetime(maxLifetime)
	logger.DebugContext(ctx, "设置数据库 ConnMaxLifetime", "value", maxLifetime)

	maxIdleTime := 5 * time.Minute // 默认连接最大空闲时间
	if cfg.ConnMaxIdleTime > 0 {
		maxIdleTime = cfg.ConnMaxIdleTime
	}
	sqlDB.SetConnMaxIdleTime(maxIdleTime)
	logger.DebugContext(ctx, "设置数据库 ConnMaxIdleTime", "value", maxIdleTime)

	// 4. 返回 GORM DB 对象
	return db, nil
}

func CloseDB(db *gorm.DB, logger logging.ILogger) {
	if db == nil {
		logger.Warn("尝试关闭一个 nil 的数据库连接")
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("获取底层 sql.DB 失败", "error", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error("关闭数据库连接失败", "error", err)
	} else {
		logger.Info("数据库连接已成功关闭")
	}
}
