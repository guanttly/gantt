package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 包含全局 Prometheus 指标。
var Metrics = struct {
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPResponseSize     *prometheus.HistogramVec
	DBQueryDuration      *prometheus.HistogramVec
	ActiveWebSocketConns prometheus.Gauge
	// M09: 排班管道 + AI Token 指标
	SchedulingPipelineDuration prometheus.Histogram
	AITokensUsed               *prometheus.CounterVec
}{
	HTTPRequestsTotal: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	),
	HTTPRequestDuration: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求延迟（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	),
	HTTPResponseSize: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP 响应体大小（字节）",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "path"},
	),
	DBQueryDuration: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "数据库查询延迟（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	),
	ActiveWebSocketConns: promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "当前活跃 WebSocket 连接数",
		},
	),
	SchedulingPipelineDuration: promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "scheduling_pipeline_duration_seconds",
			Help:    "排班管道执行耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
	),
	AITokensUsed: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_tokens_used_total",
			Help: "AI Token 使用总量",
		},
		[]string{"provider", "purpose"},
	),
}
