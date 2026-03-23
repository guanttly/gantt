// pkg/redis/redis.go
package adapter

import (
	"strconv"

	"jusha/mcp/pkg/config"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(cfg *config.RedisConfig) *Redis {
	addr := cfg.Host + ":" + strconv.Itoa(cfg.Port) // fixed conversion to string
	return &Redis{
		Client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: cfg.Username, // no username set
			Password: cfg.Password, // no password set
			DB:       cfg.DB,       // use default DB
		}),
	}
}
