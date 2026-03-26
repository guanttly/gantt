package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type healthResponse struct {
	Status string `json:"status"`
}

type readyResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// healthHandler 返回健康检查处理器（存活检测）。
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	}
}

// readyHandler 返回就绪检查处理器（含数据库和 Redis 连通性检测）。
func readyHandler(db *gorm.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := readyResponse{
			Status:   "ok",
			Services: make(map[string]string),
		}

		// 检查 MySQL
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(r.Context()) != nil {
			resp.Status = "degraded"
			resp.Services["mysql"] = "unavailable"
		} else {
			resp.Services["mysql"] = "ok"
		}

		// 检查 Redis
		if rdb.Ping(context.Background()).Err() != nil {
			resp.Status = "degraded"
			resp.Services["redis"] = "unavailable"
		} else {
			resp.Services["redis"] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		if resp.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(resp)
	}
}
