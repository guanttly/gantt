package engine

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ============================================================
// 子工作流指标
// ============================================================

// SubWorkflowMetrics 子工作流指标
type SubWorkflowMetrics struct {
	mu sync.RWMutex

	// 内部统计
	spawnTotal    int64            // 启动总数
	completeTotal int64            // 完成总数
	failureTotal  int64            // 失败总数
	timeoutTotal  int64            // 超时总数
	rollbackTotal int64            // 回滚总数
	depthMax      int              // 最大嵌套深度
	durationSum   time.Duration    // 总执行时长
	durationCount int64            // 执行次数（用于计算平均值）
	byWorkflow    map[string]int64 // 按工作流名称统计

	// Prometheus 指标
	registry *prometheus.Registry

	spawnCounter    *prometheus.CounterVec
	completeCounter *prometheus.CounterVec
	failureCounter  *prometheus.CounterVec
	timeoutCounter  prometheus.Counter
	rollbackCounter prometheus.Counter
	depthGauge      prometheus.Gauge
	durationHist    *prometheus.HistogramVec
}

// NewSubWorkflowMetrics 创建子工作流指标实例
func NewSubWorkflowMetrics(registry *prometheus.Registry) *SubWorkflowMetrics {
	m := &SubWorkflowMetrics{
		byWorkflow: make(map[string]int64),
		registry:   registry,
	}

	if registry != nil {
		m.initPrometheusMetrics()
	}

	return m
}

// initPrometheusMetrics 初始化 Prometheus 指标
func (m *SubWorkflowMetrics) initPrometheusMetrics() {
	m.spawnCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_subworkflow_spawn_total",
			Help: "Total number of sub-workflows spawned",
		},
		[]string{"parent_workflow", "child_workflow"},
	)

	m.completeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_subworkflow_complete_total",
			Help: "Total number of sub-workflows completed successfully",
		},
		[]string{"workflow"},
	)

	m.failureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_subworkflow_failure_total",
			Help: "Total number of sub-workflows failed",
		},
		[]string{"workflow", "reason"},
	)

	m.timeoutCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "workflow_subworkflow_timeout_total",
			Help: "Total number of sub-workflow timeouts",
		},
	)

	m.rollbackCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "workflow_subworkflow_rollback_total",
			Help: "Total number of sub-workflow rollbacks",
		},
	)

	m.depthGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "workflow_subworkflow_depth_max",
			Help: "Maximum observed sub-workflow nesting depth",
		},
	)

	m.durationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workflow_subworkflow_duration_seconds",
			Help:    "Sub-workflow execution duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"workflow"},
	)

	// 注册指标
	m.registry.MustRegister(m.spawnCounter)
	m.registry.MustRegister(m.completeCounter)
	m.registry.MustRegister(m.failureCounter)
	m.registry.MustRegister(m.timeoutCounter)
	m.registry.MustRegister(m.rollbackCounter)
	m.registry.MustRegister(m.depthGauge)
	m.registry.MustRegister(m.durationHist)
}

// RecordSpawn 记录子工作流启动
func (m *SubWorkflowMetrics) RecordSpawn(parentWorkflow, childWorkflow string, depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.spawnTotal++
	m.byWorkflow[childWorkflow]++

	if depth > m.depthMax {
		m.depthMax = depth
	}

	if m.spawnCounter != nil {
		m.spawnCounter.WithLabelValues(parentWorkflow, childWorkflow).Inc()
	}
	if m.depthGauge != nil {
		m.depthGauge.Set(float64(m.depthMax))
	}
}

// RecordComplete 记录子工作流完成
func (m *SubWorkflowMetrics) RecordComplete(workflow string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.completeTotal++
	m.durationSum += duration
	m.durationCount++

	if m.completeCounter != nil {
		m.completeCounter.WithLabelValues(workflow).Inc()
	}
	if m.durationHist != nil {
		m.durationHist.WithLabelValues(workflow).Observe(duration.Seconds())
	}
}

// RecordFailure 记录子工作流失败
func (m *SubWorkflowMetrics) RecordFailure(workflow, reason string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failureTotal++
	m.durationSum += duration
	m.durationCount++

	if m.failureCounter != nil {
		m.failureCounter.WithLabelValues(workflow, reason).Inc()
	}
	if m.durationHist != nil {
		m.durationHist.WithLabelValues(workflow).Observe(duration.Seconds())
	}
}

// RecordTimeout 记录子工作流超时
func (m *SubWorkflowMetrics) RecordTimeout() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.timeoutTotal++
	m.failureTotal++

	if m.timeoutCounter != nil {
		m.timeoutCounter.Inc()
	}
}

// RecordRollback 记录子工作流回滚
func (m *SubWorkflowMetrics) RecordRollback() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rollbackTotal++

	if m.rollbackCounter != nil {
		m.rollbackCounter.Inc()
	}
}

// Snapshot 获取子工作流指标快照
func (m *SubWorkflowMetrics) Snapshot() SubWorkflowMetricSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	byWorkflow := make(map[string]int64)
	for k, v := range m.byWorkflow {
		byWorkflow[k] = v
	}

	var avgDuration time.Duration
	if m.durationCount > 0 {
		avgDuration = m.durationSum / time.Duration(m.durationCount)
	}

	return SubWorkflowMetricSnapshot{
		SpawnTotal:      m.spawnTotal,
		CompleteTotal:   m.completeTotal,
		FailureTotal:    m.failureTotal,
		TimeoutTotal:    m.timeoutTotal,
		RollbackTotal:   m.rollbackTotal,
		DepthMax:        m.depthMax,
		AverageDuration: avgDuration,
		ByWorkflow:      byWorkflow,
	}
}

// SubWorkflowMetricSnapshot 子工作流指标快照
type SubWorkflowMetricSnapshot struct {
	SpawnTotal      int64            `json:"spawn_total"`
	CompleteTotal   int64            `json:"complete_total"`
	FailureTotal    int64            `json:"failure_total"`
	TimeoutTotal    int64            `json:"timeout_total"`
	RollbackTotal   int64            `json:"rollback_total"`
	DepthMax        int              `json:"depth_max"`
	AverageDuration time.Duration    `json:"average_duration"`
	ByWorkflow      map[string]int64 `json:"by_workflow"`
}
