package progressive

import (
	"context"
	"fmt"
	"time"

	"jusha/mcp/pkg/logging"
)

// IProgressiveTaskService 渐进式任务执行服务接口
type IProgressiveTaskService interface {
	// ExecuteTask 执行单个任务
	ExecuteTask(ctx context.Context, task *Task, executor ITaskExecutor, context *TaskContext) (*TaskResult, error)
	
	// ExecuteTaskPlan 执行任务计划
	ExecuteTaskPlan(ctx context.Context, plan *TaskPlan, executor ITaskExecutor, context *TaskContext) (*PlanResult, error)
}

// ProgressiveTaskService 渐进式任务执行服务实现
type ProgressiveTaskService struct {
	logger logging.ILogger
}

// NewProgressiveTaskService 创建渐进式任务执行服务
func NewProgressiveTaskService(logger logging.ILogger) IProgressiveTaskService {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}
	return &ProgressiveTaskService{
		logger: logger.With("component", "ProgressiveTaskService"),
	}
}

// ExecuteTask 执行单个任务
func (s *ProgressiveTaskService) ExecuteTask(
	ctx context.Context,
	task *Task,
	executor ITaskExecutor,
	context *TaskContext,
) (*TaskResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}
	
	if executor == nil {
		return nil, fmt.Errorf("executor cannot be nil")
	}
	
	// 检查执行器是否能处理该任务
	if !executor.CanHandle(task) {
		return nil, fmt.Errorf("executor cannot handle task %s", task.ID)
	}
	
	// 更新任务状态
	task.Status = "executing"
	task.ExecutedAt = time.Now().Format(time.RFC3339)
	
	s.logger.Info("Executing task", "taskID", task.ID, "title", task.Title)
	
	startTime := time.Now()
	
	// 执行任务
	result, err := executor.Execute(ctx, task, context)
	
	executionTime := time.Since(startTime).Seconds()
	
	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("执行失败: %v", err)
		s.logger.Error("Task execution failed", "taskID", task.ID, "error", err)
		
		return &TaskResult{
			TaskID:        task.ID,
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: executionTime,
		}, err
	}
	
	// 更新任务状态
	task.Status = "completed"
	if result != nil {
		if result.Error != "" {
			task.Result = result.Error
		} else {
			task.Result = "执行成功"
		}
		result.ExecutionTime = executionTime
	}
	
	s.logger.Info("Task execution completed", "taskID", task.ID, "success", result.Success, "time", executionTime)
	
	return result, nil
}

// ExecuteTaskPlan 执行任务计划
func (s *ProgressiveTaskService) ExecuteTaskPlan(
	ctx context.Context,
	plan *TaskPlan,
	executor ITaskExecutor,
	context *TaskContext,
) (*PlanResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("task plan cannot be nil")
	}
	
	if executor == nil {
		return nil, fmt.Errorf("executor cannot be nil")
	}
	
	if context == nil {
		context = &TaskContext{
			ContextData: make(map[string]any),
		}
	}
	
	s.logger.Info("Executing task plan", "taskCount", len(plan.Tasks))
	
	startTime := time.Now()
	result := &PlanResult{
		TaskResults:        make([]*TaskResult, 0, len(plan.Tasks)),
		CompletedCount:     0,
		FailedCount:        0,
		FinalResultData:    make(map[string]any),
	}
	
	// 按顺序执行任务
	for i, task := range plan.Tasks {
		s.logger.Info("Executing task in plan", "index", i+1, "total", len(plan.Tasks), "taskID", task.ID)
		
		taskResult, err := s.ExecuteTask(ctx, task, executor, context)
		
		if err != nil {
			result.FailedCount++
			s.logger.Warn("Task failed in plan", "taskID", task.ID, "error", err)
		} else if taskResult != nil && taskResult.Success {
			result.CompletedCount++
		} else {
			result.FailedCount++
		}
		
		if taskResult != nil {
			result.TaskResults = append(result.TaskResults, taskResult)
			
			// 合并结果数据到上下文（供后续任务使用）
			if taskResult.ResultData != nil {
				for k, v := range taskResult.ResultData {
					context.ContextData[k] = v
				}
			}
		}
		
		// 如果任务失败且需要停止，可以在这里添加逻辑
		// 当前实现继续执行后续任务
	}
	
	result.TotalExecutionTime = time.Since(startTime).Seconds()
	
	s.logger.Info("Task plan execution completed",
		"completed", result.CompletedCount,
		"failed", result.FailedCount,
		"totalTime", result.TotalExecutionTime)
	
	return result, nil
}
