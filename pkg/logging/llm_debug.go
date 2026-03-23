// pkg/logging/llm_debug.go
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// LLMDebugLogger 用于将LLM调用的输入输出保存到单独文件
// 按任务分类，每个LLM调用保存为独立文件
type LLMDebugLogger struct {
	baseDir     string         // 基础目录，默认为 ./debugllm
	enabled     bool           // 是否启用
	mu          sync.Mutex     // 保护文件写入
	callCounter map[string]int // 每个任务的调用计数器
}

var (
	globalLLMDebugLogger *LLMDebugLogger
	llmDebugOnce         sync.Once
)

// LLMCallType 定义LLM调用类型
type LLMCallType string

const (
	LLMCallTaskParsing           LLMCallType = "task_parsing"           // 任务解析
	LLMCallShiftGrouping         LLMCallType = "shift_grouping"         // 班次分组
	LLMCallScheduling            LLMCallType = "scheduling"             // 排班生成
	LLMCallValidation            LLMCallType = "validation"             // 校验
	LLMCallFailureAnalysis       LLMCallType = "failure_analysis"       // 失败分析
	LLMCallRuleMatching          LLMCallType = "rule_matching"          // 规则匹配
	LLMCallDayValidation         LLMCallType = "day_validation"         // 单日校验
	LLMCallBatchScheduling       LLMCallType = "batch_scheduling"       // 批量排班
	LLMCallBatchValidation       LLMCallType = "batch_validation"       // 批量校验
	LLMCallProgressiveDay        LLMCallType = "progressive_day"        // 渐进式单日
	LLMCallRequirementAssessment LLMCallType = "requirement_assessment" // 需求评估生成计划
	LLMCallPersonalNeedsFilter   LLMCallType = "personal_needs_filter"  // 个人需求过滤
	LLMCallRulesFilter           LLMCallType = "rules_filter"           // 规则过滤
	LLMCallScheduleAdjust        LLMCallType = "schedule_adjust"        // 排班调整(LLM4)
	LLMCallRuleConflict          LLMCallType = "rule_conflict"          // 规则冲突分析(LLM3)
	LLMCallDayCorrection         LLMCallType = "day_correction"         // 单日纠正
	LLMCallBatchFix              LLMCallType = "batch_fix"              // 批量修复
	LLMCallV4RuleAdjustment      LLMCallType = "v4_rule_adjustment"     // V4语义规则调整
	LLMCallV4Validation          LLMCallType = "v4_validation"          // V4 LLM辅助校验
)

// callTypeStepMap 定义调用类型到执行步骤的映射（按执行顺序）
var callTypeStepMap = map[LLMCallType]int{
	LLMCallTaskParsing:           1,  // Step 1: 任务解析
	LLMCallShiftGrouping:         2,  // Step 2: 班次分组
	LLMCallRequirementAssessment: 3,  // Step 3: 需求评估
	LLMCallPersonalNeedsFilter:   4,  // Step 4: 个人需求过滤
	LLMCallRulesFilter:           5,  // Step 5: 规则过滤
	LLMCallRuleConflict:          6,  // Step 6: 规则冲突分析
	LLMCallProgressiveDay:        7,  // Step 7: 渐进式单日排班
	LLMCallDayValidation:         8,  // Step 8: 单日校验
	LLMCallDayCorrection:         9,  // Step 9: 单日纠正
	LLMCallScheduleAdjust:        10, // Step 10: 排班调整
	LLMCallBatchScheduling:       11, // Step 11: 批量排班
	LLMCallBatchValidation:       12, // Step 12: 批量校验
	LLMCallBatchFix:              13, // Step 13: 批量修复
	LLMCallFailureAnalysis:       14, // Step 14: 失败分析
	LLMCallRuleMatching:          15, // Step 15: 规则匹配
	LLMCallScheduling:            16, // Step 16: 排班生成（通用）
	LLMCallValidation:            17, // Step 17: 校验（通用）
	LLMCallV4RuleAdjustment:      18, // Step 18: V4语义规则调整
	LLMCallV4Validation:          19, // Step 19: V4 LLM辅助校验
}

// callTypeNameMap 定义调用类型的中文名称
var callTypeNameMap = map[LLMCallType]string{
	LLMCallTaskParsing:           "任务解析",
	LLMCallShiftGrouping:         "班次分组",
	LLMCallRequirementAssessment: "需求评估",
	LLMCallPersonalNeedsFilter:   "个人需求过滤",
	LLMCallRulesFilter:           "规则过滤",
	LLMCallRuleConflict:          "规则冲突分析",
	LLMCallProgressiveDay:        "渐进式单日排班",
	LLMCallDayValidation:         "单日校验",
	LLMCallDayCorrection:         "单日纠正",
	LLMCallScheduleAdjust:        "排班调整",
	LLMCallBatchScheduling:       "批量排班",
	LLMCallBatchValidation:       "批量校验",
	LLMCallBatchFix:              "批量修复",
	LLMCallFailureAnalysis:       "失败分析",
	LLMCallRuleMatching:          "规则匹配",
	LLMCallScheduling:            "排班生成",
	LLMCallValidation:            "校验",
	LLMCallV4RuleAdjustment:      "V4语义规则调整",
	LLMCallV4Validation:          "V4辅助校验",
}

// getStepNumber 获取调用类型对应的步骤编号
func getStepNumber(callType LLMCallType) int {
	if step, ok := callTypeStepMap[callType]; ok {
		return step
	}
	// 如果未定义，返回一个较大的数字，放在最后
	return 99
}

// getStepDirName 获取步骤目录名称（格式：stepXX_调用类型）
func getStepDirName(callType LLMCallType) string {
	step := getStepNumber(callType)
	return fmt.Sprintf("step%02d_%s", step, string(callType))
}

// getCallTypeName 获取调用类型的中文名称
func getCallTypeName(callType LLMCallType) string {
	if name, ok := callTypeNameMap[callType]; ok {
		return name
	}
	return string(callType)
}

// GetLLMDebugLogger 获取全局LLM调试日志实例
func GetLLMDebugLogger() *LLMDebugLogger {
	llmDebugOnce.Do(func() {
		globalLLMDebugLogger = NewLLMDebugLogger("./debugllm", true)
	})
	return globalLLMDebugLogger
}

// NewLLMDebugLogger 创建新的LLM调试日志实例
func NewLLMDebugLogger(baseDir string, enabled bool) *LLMDebugLogger {
	logger := &LLMDebugLogger{
		baseDir:     baseDir,
		enabled:     enabled,
		callCounter: make(map[string]int),
	}

	// 确保基础目录存在
	if enabled {
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			GlobalLogger.Error("Failed to create LLM debug directory", "dir", baseDir, "error", err)
		}
	}

	return logger
}

// SetEnabled 设置是否启用
func (l *LLMDebugLogger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// IsEnabled 检查是否启用
func (l *LLMDebugLogger) IsEnabled() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enabled
}

// LogLLMCall 记录LLM调用
// taskTitle: 任务标题（用于文件名）
// callType: 调用类型
// shiftName: 班次名称（可选）
// dateName: 日期（可选）
// modelName: 使用的模型名称（可选，如 qwen3-max）
// systemPrompt: 系统提示词
// userPrompt: 用户提示词
// response: LLM响应内容
// duration: 调用耗时
// err: 错误信息（可选）
func (l *LLMDebugLogger) LogLLMCall(
	taskTitle string,
	callType LLMCallType,
	shiftName string,
	dateName string,
	modelName string,
	systemPrompt string,
	userPrompt string,
	response string,
	duration time.Duration,
	err error,
) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 生成序号
	counterKey := fmt.Sprintf("%s_%s", taskTitle, callType)
	l.callCounter[counterKey]++
	seqNum := l.callCounter[counterKey]

	// 构建文件名
	fileName := l.buildFileName(taskTitle, callType, shiftName, dateName, seqNum)

	// 构建文件路径（按日期和步骤分目录）
	dateDir := time.Now().Format("2006-01-02")
	stepDir := getStepDirName(callType) // 使用step格式：step01_task_parsing
	fullDir := filepath.Join(l.baseDir, dateDir, stepDir)

	// 确保目录存在
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		GlobalLogger.Error("Failed to create LLM debug subdirectory", "dir", fullDir, "error", err)
		return
	}

	filePath := filepath.Join(fullDir, fileName)

	// 构建文件内容
	content := l.buildContent(taskTitle, callType, shiftName, dateName, modelName, seqNum, systemPrompt, userPrompt, response, duration, err)

	// 写入文件
	if writeErr := os.WriteFile(filePath, []byte(content), 0644); writeErr != nil {
		GlobalLogger.Error("Failed to write LLM debug file", "path", filePath, "error", writeErr)
	}
}

// buildFileName 构建文件名
func (l *LLMDebugLogger) buildFileName(taskTitle string, callType LLMCallType, shiftName string, dateName string, seqNum int) string {
	// 清理文件名中的非法字符
	safeTitle := sanitizeFileName(taskTitle)
	safeShift := sanitizeFileName(shiftName)
	safeDate := sanitizeFileName(dateName)

	// 限制长度（按字符数，不是字节数，避免截断多字节UTF-8字符）
	if utf8.RuneCountInString(safeTitle) > 30 {
		runes := []rune(safeTitle)
		safeTitle = string(runes[:30])
	}
	if utf8.RuneCountInString(safeShift) > 20 {
		runes := []rune(safeShift)
		safeShift = string(runes[:20])
	}
	if utf8.RuneCountInString(safeDate) > 15 {
		runes := []rune(safeDate)
		safeDate = string(runes[:15])
	}

	// 构建文件名
	var parts []string
	parts = append(parts, fmt.Sprintf("%03d", seqNum))
	if safeTitle != "" {
		parts = append(parts, safeTitle)
	}
	if safeShift != "" {
		parts = append(parts, safeShift)
	}
	if safeDate != "" {
		parts = append(parts, safeDate)
	}

	return strings.Join(parts, "_") + ".md"
}

// buildContent 构建文件内容（Markdown格式）
func (l *LLMDebugLogger) buildContent(
	taskTitle string,
	callType LLMCallType,
	shiftName string,
	dateName string,
	modelName string,
	seqNum int,
	systemPrompt string,
	userPrompt string,
	response string,
	duration time.Duration,
	err error,
) string {
	var sb strings.Builder

	// 元信息
	sb.WriteString("# LLM调用记录\n\n")
	sb.WriteString("## 元信息\n\n")

	// 添加步骤信息
	stepNum := getStepNumber(callType)
	stepName := getCallTypeName(callType)
	sb.WriteString(fmt.Sprintf("- **步骤**: Step %02d - %s\n", stepNum, stepName))
	sb.WriteString(fmt.Sprintf("- **时间**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **任务**: %s\n", taskTitle))
	sb.WriteString(fmt.Sprintf("- **调用类型**: %s\n", callType))
	if shiftName != "" {
		sb.WriteString(fmt.Sprintf("- **班次**: %s\n", shiftName))
	}
	if dateName != "" {
		sb.WriteString(fmt.Sprintf("- **日期**: %s\n", dateName))
	}
	if modelName != "" {
		sb.WriteString(fmt.Sprintf("- **模型**: %s\n", modelName))
	}
	sb.WriteString(fmt.Sprintf("- **序号**: %d\n", seqNum))
	sb.WriteString(fmt.Sprintf("- **耗时**: %.2fs\n", duration.Seconds()))
	if err != nil {
		sb.WriteString(fmt.Sprintf("- **错误**: %s\n", err.Error()))
	}
	sb.WriteString("\n---\n\n")

	// System Prompt
	sb.WriteString("## System Prompt\n\n")
	sb.WriteString("```\n")
	sb.WriteString(systemPrompt)
	sb.WriteString("\n```\n\n")

	// User Prompt
	sb.WriteString("## User Prompt\n\n")
	sb.WriteString("```\n")
	sb.WriteString(userPrompt)
	sb.WriteString("\n```\n\n")

	// Response
	sb.WriteString("## Response\n\n")
	if err != nil {
		sb.WriteString("**调用失败**\n\n")
	}
	sb.WriteString("```\n")
	sb.WriteString(response)
	sb.WriteString("\n```\n")

	return sb.String()
}

// sanitizeFileName 清理文件名中的非法字符
func sanitizeFileName(name string) string {
	if name == "" {
		return ""
	}
	// 移除或替换非法字符（包括Windows/Linux文件系统不支持的字符）
	// 注意：括号()在某些系统上可能有问题，也替换掉
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f()（）]`)
	safe := re.ReplaceAllString(name, "_")
	// 移除多余空格
	safe = strings.TrimSpace(safe)
	// 替换空格为下划线
	safe = strings.ReplaceAll(safe, " ", "_")
	// 移除连续的下划线
	for strings.Contains(safe, "__") {
		safe = strings.ReplaceAll(safe, "__", "_")
	}
	// 移除开头和结尾的下划线
	safe = strings.Trim(safe, "_")
	return safe
}

// ClearOldLogs 清理旧的调试日志（保留最近N天）
func (l *LLMDebugLogger) ClearOldLogs(keepDays int) error {
	if !l.enabled {
		return nil
	}

	entries, err := os.ReadDir(l.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read debug directory: %w", err)
	}

	cutoffDate := time.Now().AddDate(0, 0, -keepDays)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 尝试解析目录名为日期
		dirDate, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue // 不是日期格式的目录，跳过
		}

		if dirDate.Before(cutoffDate) {
			dirPath := filepath.Join(l.baseDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				GlobalLogger.Warn("Failed to remove old debug log directory", "dir", dirPath, "error", err)
			} else {
				GlobalLogger.Info("Removed old debug log directory", "dir", dirPath)
			}
		}
	}

	return nil
}

// ResetCounters 重置调用计数器（通常在新任务开始时调用）
func (l *LLMDebugLogger) ResetCounters() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callCounter = make(map[string]int)
}
