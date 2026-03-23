package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"jusha/agent/server/context/config"
	"jusha/agent/server/context/domain/model"
	domain_repo "jusha/agent/server/context/domain/repository"
	domain_service "jusha/agent/server/context/domain/service"
	"jusha/agent/server/context/internal/repository"
	"jusha/agent/server/context/internal/service"
	"jusha/mcp/pkg/adapter"
	"jusha/mcp/pkg/logging"
)

// Container 依赖注入容器
type Container struct {
	ctx    context.Context
	logger logging.ILogger
	config config.IContextConfigurator

	// 基础设施
	db *gorm.DB

	// 仓储层
	repoProvider domain_repo.IRepositoryProvider

	// 服务层
	serviceProvider domain_service.IServiceProvider
}

// NewContainer 创建依赖注入容器
func NewContainer(ctx context.Context, logger logging.ILogger, cfg config.IContextConfigurator) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	container := &Container{
		ctx:    ctx,
		logger: logger.With("component", "Container"),
		config: cfg,
	}

	// 初始化数据库
	if err := container.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 初始化仓储层
	container.initRepositories()

	// 初始化服务层
	container.initServices()

	container.logger.Info("Dependency injection container initialized successfully")
	return container, nil
}

// initDatabase 初始化数据库连接
func (c *Container) initDatabase() error {
	cfg := c.config.GetConfig()

	// 检查数据库配置
	if cfg.Config == nil || cfg.Config.Database == nil || cfg.Config.Database.MySQL == nil {
		return fmt.Errorf("database config is required")
	}

	mysqlCfg := cfg.Config.Database.MySQL
	if mysqlCfg.Host == "" {
		return fmt.Errorf("database host is required")
	}

	// 使用 adapter.NewMySQLConnection 初始化数据库连接
	dbCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	db, err := adapter.NewMySQLConnection(dbCtx, mysqlCfg, c.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	c.db = db
	c.logger.Info("Database connected successfully")

	// 自动迁移表结构
	if err := c.autoMigrate(); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	return nil
}

// autoMigrate 自动迁移数据库表结构
func (c *Container) autoMigrate() error {
	if err := c.db.AutoMigrate(
		&model.Conversation{},
		&model.ConversationMessage{},
	); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	return nil
}

// initRepositories 初始化仓储层
func (c *Container) initRepositories() {
	c.repoProvider = repository.NewRepositoryProviderBuilder().
		WithLogger(c.logger).
		WithDB(c.db).
		Build()

	c.logger.Info("All repositories initialized successfully")
}

// initServices 初始化服务层
func (c *Container) initServices() {
	c.serviceProvider = service.NewServiceProviderBuilder().
		WithLogger(c.logger).
		WithRepositoryProvider(c.repoProvider).
		Build()

	c.logger.Info("All services initialized successfully")
}

// GetServiceProvider 获取服务提供者
func (c *Container) GetServiceProvider() domain_service.IServiceProvider {
	return c.serviceProvider
}

// Close 关闭资源
func (c *Container) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
