// Package server 提供 HTTP Server 封装，基于 chi 路由。
package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gantt-saas/internal/infra/config"
	"gantt-saas/internal/infra/observability"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server 封装 HTTP 服务器及其依赖。
type Server struct {
	Router *chi.Mux
	HTTP   *http.Server
	Logger *zap.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Config *config.Config
}

// New 创建 HTTP Server，包含全局中间件和基础路由。
func New(cfg *config.Config, logger *zap.Logger, db *gorm.DB, rdb *redis.Client) *Server {
	r := chi.NewRouter()

	// ── 全局中间件 ──
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)
	r.Use(requestLoggerMiddleware(logger))
	r.Use(metricsMiddleware)

	// ── 健康检查 ──
	r.Get("/healthz", healthHandler())
	r.Get("/readyz", readyHandler(db, rdb))

	// ── Prometheus 指标端点 ──
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &Server{
		Router: r,
		HTTP:   srv,
		Logger: logger,
		DB:     db,
		Redis:  rdb,
		Config: cfg,
	}
}

// Start 启动 HTTP 服务器（非阻塞）。
func (s *Server) Start() error {
	s.Logger.Info("HTTP 服务器启动",
		zap.String("addr", s.HTTP.Addr),
		zap.String("mode", s.Config.Server.Mode),
	)
	if err := s.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP 服务器错误: %w", err)
	}
	return nil
}

// Shutdown 优雅关闭 HTTP 服务器。
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Server.ShutdownTimeout)
	defer cancel()

	s.Logger.Info("正在关闭 HTTP 服务器...")
	return s.HTTP.Shutdown(ctx)
}

// ── 中间件实现 ──

// corsMiddleware 处理跨域请求。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, X-Org-Node-ID, X-Org-Node-Path")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestLoggerMiddleware 请求日志中间件。
func requestLoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("HTTP 请求",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", time.Since(start)),
					zap.String("request_id", chimw.GetReqID(r.Context())),
					zap.String("remote_addr", r.RemoteAddr),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// metricsMiddleware Prometheus 指标中间件。
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())
		path := r.URL.Path

		observability.Metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		observability.Metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		observability.Metrics.HTTPResponseSize.WithLabelValues(r.Method, path).Observe(float64(ww.BytesWritten()))
	})
}
