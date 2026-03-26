// Package cache 提供 Redis 客户端封装。
package cache

import (
	"context"
	"fmt"

	"gantt-saas/internal/infra/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRedis 根据配置初始化 Redis 客户端。
func NewRedis(cfg *config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 验证连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	logger.Info("Redis 连接成功", zap.String("addr", cfg.Addr))

	return rdb, nil
}
