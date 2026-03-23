package progressive

import (
	"context"
)

// Task 通用任务模型（与领域模型解耦）
type Task struct {
	// ID 任务ID
	ID string `json:"id"`
	
	// Order 执行顺序
	Order int `json:"order"`
	
	// Title 任务标题
	Title string `json:"title"`
	
	// Description 任务详细说明
	Description string `json:"description"`
	
	// TargetShifts 涉及的班次ID列表（可选，领域特定）
	TargetShifts []string `json:"targetShifts,omitempty"`
	
	// TargetDates 涉及的日期列表（可选，领域特定）
	TargetDates []string `json:"targetDates,omitempty"`
	
	// TargetStaff 涉及的人员ID列表（可选，领域特定）
	TargetStaff []string `json:"targetStaff,omitempty"`
	
	// RuleIDs 相关规则ID列表（可选，领域特定）
	RuleIDs []string `json:"ruleIds,omitempty"`
	
	// Priority 优先级 (1-高，2-中，3-低)
	Priority int `json:"priority"`
	
	// Status 状态: "pending", "executing", "completed", "failed"
	Status string `json:"status"`
	
	// Result 执行结果说明
	Result string `json:"result,omitempty"`
	
	// ExecutedAt 执行时间
	ExecutedAt string `json:"executedAt,omitempty"`
	
	// Metadata 元数据（领域特定扩展）
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TaskPlan 任务计划
type TaskPlan struct {
	// Tasks 有序的任务列表
	Tasks []*Task `json:"tasks"`
	
	// Summary 整体规划说明
	Summary string `json:"summary"`
	
	// Reasoning AI的思考过程
	Reasoning string `json:"reasoning,omitempty"`
}

// TaskContext 任务执行上下文（领域特定数据）
type TaskContext struct {
	// ContextData 上下文数据（领域特定，由执行器填充）
	ContextData map[string]any `json:"contextData"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	// TaskID 任务ID
	TaskID string `json:"taskId"`
	
	// Success 是否成功
	Success bool `json:"success"`
	
	// ResultData 结果数据（领域特定，由执行器填充）
	ResultData map[string]any `json:"resultData,omitempty"`
	
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
	
	// ExecutionTime 执行时间（秒）
	ExecutionTime float64 `json:"executionTime"`
	
	// Metadata 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PlanResult 计划执行结果
type PlanResult struct {
	// TaskResults 任务执行结果列表
	TaskResults []*TaskResult `json:"taskResults"`
	
	// CompletedCount 完成任务数
	CompletedCount int `json:"completedCount"`
	
	// FailedCount 失败任务数
	FailedCount int `json:"failedCount"`
	
	// TotalExecutionTime 总执行时间（秒）
	TotalExecutionTime float64 `json:"totalExecutionTime"`
	
	// FinalResultData 最终结果数据（领域特定）
	FinalResultData map[string]any `json:"finalResultData,omitempty"`
}

// ITaskExecutor 任务执行器接口
type ITaskExecutor interface {
	// Execute 执行任务
	Execute(ctx context.Context, task *Task, context *TaskContext) (*TaskResult, error)
	
	// CanHandle 判断是否能处理该任务
	CanHandle(task *Task) bool
}
