package session

import (
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================
// 子工作流栈帧
// ============================================================

// MaxWorkflowDepth 最大工作流嵌套深度
const MaxWorkflowDepth = 5

// WorkflowFrame 工作流栈帧，记录父工作流调用点信息
type WorkflowFrame struct {
	Workflow     string         `json:"workflow"`      // 父工作流名称
	Phase        string         `json:"phase"`         // 父工作流调用时的阶段
	ReturnEvent  string         `json:"return_event"`  // 子工作流成功后返回的事件
	ErrorEvent   string         `json:"error_event"`   // 子工作流失败后返回的事件
	SnapshotKeys []string       `json:"snapshot_keys"` // 需要回滚的数据键
	SnapshotData map[string]any `json:"snapshot_data"` // 快照数据（用于回滚）
	StartedAt    time.Time      `json:"started_at"`    // 子工作流启动时间
}

// ============================================================
// 并行子工作流状态
// ============================================================

// MergeStrategy 并行子工作流结果合并策略
type MergeStrategy string

const (
	MergeStrategyAll    MergeStrategy = "all"    // 等待所有子工作流完成
	MergeStrategyAny    MergeStrategy = "any"    // 任一子工作流完成即返回
	MergeStrategyCustom MergeStrategy = "custom" // 自定义合并逻辑
)

// ParallelSubWorkflowResult 单个并行子工作流结果
type ParallelSubWorkflowResult struct {
	WorkflowName string        `json:"workflow_name"`
	Success      bool          `json:"success"`
	Output       any           `json:"output,omitempty"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	CompletedAt  time.Time     `json:"completed_at"`
}

// ParallelSubWorkflowState 并行子工作流状态
type ParallelSubWorkflowState struct {
	TotalCount     int                         `json:"total_count"`     // 总数
	CompletedCount int                         `json:"completed_count"` // 已完成数
	SuccessCount   int                         `json:"success_count"`   // 成功数
	FailedCount    int                         `json:"failed_count"`    // 失败数
	MergeStrategy  MergeStrategy               `json:"merge_strategy"`  // 合并策略
	Results        []ParallelSubWorkflowResult `json:"results"`         // 各子工作流结果
	OnAllComplete  string                      `json:"on_all_complete"` // 全部完成事件
	OnAnyComplete  string                      `json:"on_any_complete"` // 任一完成事件
	OnAllFailed    string                      `json:"on_all_failed"`   // 全部失败事件
}

// ============================================================
// WorkflowMeta Extra 字段扩展
// ============================================================

const (
	KeyWorkflowStack = "workflow_stack" // 工作流调用栈
	KeyParallelState = "parallel_state" // 并行子工作流状态
)

// ensureExtra 确保 Extra 字段已初始化
func (m *WorkflowMeta) ensureExtra() {
	if m.Extra == nil {
		m.Extra = make(map[string]any)
	}
}

// GetWorkflowStack 获取工作流调用栈
func (m *WorkflowMeta) GetWorkflowStack() []WorkflowFrame {
	if m.Extra == nil {
		return nil
	}
	raw, ok := m.Extra[KeyWorkflowStack]
	if !ok {
		return nil
	}

	// 处理反序列化后可能是 []interface{} 的情况
	switch stack := raw.(type) {
	case []WorkflowFrame:
		return stack
	case []interface{}:
		result := make([]WorkflowFrame, 0, len(stack))
		for _, item := range stack {
			if frame, ok := item.(WorkflowFrame); ok {
				result = append(result, frame)
			} else if frameMap, ok := item.(map[string]interface{}); ok {
				// JSON 反序列化后的情况
				frame := WorkflowFrame{}
				if workflow, ok := frameMap["workflow"].(string); ok {
					frame.Workflow = workflow
				}
				if phase, ok := frameMap["phase"].(string); ok {
					frame.Phase = phase
				}
				if returnEvent, ok := frameMap["return_event"].(string); ok {
					frame.ReturnEvent = returnEvent
				}
				if errorEvent, ok := frameMap["error_event"].(string); ok {
					frame.ErrorEvent = errorEvent
				}
				if keys, ok := frameMap["snapshot_keys"].([]any); ok {
					frame.SnapshotKeys = make([]string, 0, len(keys))
					for _, k := range keys {
						if ks, ok := k.(string); ok {
							frame.SnapshotKeys = append(frame.SnapshotKeys, ks)
						}
					}
				}
				if data, ok := frameMap["snapshot_data"].(map[string]any); ok {
					frame.SnapshotData = data
				}
				if startedAt, ok := frameMap["started_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
						frame.StartedAt = t
					}
				}
				result = append(result, frame)
			}
		}
		return result
	default:
		// 尝试 JSON 反序列化
		jsonBytes, _ := json.Marshal(raw)
		var result []WorkflowFrame
		if json.Unmarshal(jsonBytes, &result) == nil {
			return result
		}
		return nil
	}
}

// PushWorkflowFrame 压栈工作流帧
func (m *WorkflowMeta) PushWorkflowFrame(frame WorkflowFrame) error {
	m.ensureExtra()
	stack := m.GetWorkflowStack()
	if len(stack) >= MaxWorkflowDepth {
		return ErrMaxWorkflowDepthExceeded
	}
	stack = append(stack, frame)
	m.Extra[KeyWorkflowStack] = stack
	return nil
}

// PopWorkflowFrame 弹栈工作流帧
func (m *WorkflowMeta) PopWorkflowFrame() (*WorkflowFrame, error) {
	stack := m.GetWorkflowStack()
	if len(stack) == 0 {
		return nil, ErrWorkflowStackEmpty
	}
	frame := stack[len(stack)-1]
	m.Extra[KeyWorkflowStack] = stack[:len(stack)-1]
	return &frame, nil
}

// PeekWorkflowFrame 查看栈顶帧但不弹出
func (m *WorkflowMeta) PeekWorkflowFrame() *WorkflowFrame {
	stack := m.GetWorkflowStack()
	if len(stack) == 0 {
		return nil
	}
	frame := stack[len(stack)-1]
	return &frame
}

// GetWorkflowDepth 获取当前工作流嵌套深度
func (m *WorkflowMeta) GetWorkflowDepth() int {
	return len(m.GetWorkflowStack())
}

// IsSubWorkflow 判断当前是否在子工作流中
func (m *WorkflowMeta) IsSubWorkflow() bool {
	return m.GetWorkflowDepth() > 0
}

// SetParallelState 设置并行子工作流状态
func (m *WorkflowMeta) SetParallelState(state *ParallelSubWorkflowState) {
	m.ensureExtra()
	m.Extra[KeyParallelState] = state
}

// GetParallelState 获取并行子工作流状态
func (m *WorkflowMeta) GetParallelState() *ParallelSubWorkflowState {
	if m.Extra == nil {
		return nil
	}
	raw, ok := m.Extra[KeyParallelState]
	if !ok {
		return nil
	}
	if state, ok := raw.(*ParallelSubWorkflowState); ok {
		return state
	}
	return nil
}

// ClearParallelState 清除并行子工作流状态
func (m *WorkflowMeta) ClearParallelState() {
	if m.Extra != nil {
		delete(m.Extra, KeyParallelState)
	}
}

// ============================================================
// 子工作流错误类型
// ============================================================

// 预定义错误
var (
	ErrMaxWorkflowDepthExceeded = fmt.Errorf("max workflow depth exceeded (limit: %d)", MaxWorkflowDepth)
	ErrWorkflowStackEmpty       = fmt.Errorf("workflow stack is empty")
	ErrNotInSubWorkflow         = fmt.Errorf("not in a sub-workflow")
	ErrSubWorkflowTimeout       = fmt.Errorf("sub-workflow execution timeout")
)
