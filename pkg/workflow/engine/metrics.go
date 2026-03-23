package engine

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// FSMMetrics FSM 指标实现
type FSMMetrics struct {
	mu             sync.RWMutex
	stateActive    map[State]int64
	transitionHist map[string]int64
	totalEvents    int64

	// Prometheus 指标
	registry          *prometheus.Registry
	transitionCounter *prometheus.CounterVec
	stateGauge        *prometheus.GaugeVec
}

// NewMetrics 创建指标实例
func NewMetrics() *FSMMetrics {
	m := &FSMMetrics{
		stateActive:    make(map[State]int64),
		transitionHist: make(map[string]int64),
		registry:       prometheus.NewRegistry(),
	}

	// 创建 Prometheus 指标
	m.transitionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_transitions_total",
			Help: "Total number of workflow transitions",
		},
		[]string{"from", "event", "to"},
	)

	m.stateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workflow_state_active",
			Help: "Number of workflows in each state",
		},
		[]string{"state"},
	)

	m.registry.MustRegister(m.transitionCounter)
	m.registry.MustRegister(m.stateGauge)

	return m
}

// Record 记录状态转换
func (m *FSMMetrics) Record(from State, event Event, to State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s-%s->%s", from, event, to)
	m.transitionHist[key]++
	m.totalEvents++

	// 更新 Prometheus 指标
	m.transitionCounter.WithLabelValues(string(from), string(event), string(to)).Inc()
}

// IncStateActive 增加活跃状态计数
func (m *FSMMetrics) IncStateActive(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stateActive[state]++
	m.stateGauge.WithLabelValues(string(state)).Set(float64(m.stateActive[state]))
}

// DecStateActive 减少活跃状态计数
func (m *FSMMetrics) DecStateActive(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stateActive[state] > 0 {
		m.stateActive[state]--
	}
	m.stateGauge.WithLabelValues(string(state)).Set(float64(m.stateActive[state]))
}

// Snapshot 获取指标快照
func (m *FSMMetrics) Snapshot() MetricSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stateActive := make(map[State]int64)
	for k, v := range m.stateActive {
		stateActive[k] = v
	}

	transitionHist := make(map[string]int64)
	for k, v := range m.transitionHist {
		transitionHist[k] = v
	}

	return MetricSnapshot{
		StateActive:    stateActive,
		TransitionHist: transitionHist,
		TotalEvents:    m.totalEvents,
	}
}

// Registry 获取 Prometheus 注册表
func (m *FSMMetrics) Registry() *prometheus.Registry {
	return m.registry
}
