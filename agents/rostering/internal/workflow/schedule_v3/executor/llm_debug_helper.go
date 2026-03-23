// llm_debug_helper.go
// 封装 LLM 调试日志调用的辅助函数
package executor

import (
	"time"

	"jusha/mcp/pkg/logging"
)

// logLLMDebug 记录LLM调用到调试文件
// 封装 pkg/logging.LLMDebugLogger 的调用
func (e *ProgressiveTaskExecutor) logLLMDebug(
	taskTitle string,
	callType logging.LLMCallType,
	shiftName string,
	dateName string,
	systemPrompt string,
	userPrompt string,
	response string,
	duration time.Duration,
	err error,
) {
	debugLogger := logging.GetLLMDebugLogger()
	if debugLogger == nil || !debugLogger.IsEnabled() {
		return
	}

	debugLogger.LogLLMCall(
		taskTitle,
		callType,
		shiftName,
		dateName,
		"", // modelName 暂无（V3保留为空）
		systemPrompt,
		userPrompt,
		response,
		duration,
		err,
	)
}
