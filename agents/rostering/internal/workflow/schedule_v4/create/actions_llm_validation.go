package create

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
)

// ============================================================
// LLM 语义规则校验（只读，不修改排班）
// ============================================================

// llmRuleValidationResult LLM 单条规则校验结果
type llmRuleValidationResult struct {
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	IsCompliant bool   `json:"isCompliant"` // 是否符合规则
	Reason      string `json:"reason"`      // 校验原因说明
	Suggestion  string `json:"suggestion"`  // 修改建议（仅供参考，不主动执行）
}

// llmSingleRuleResponse LLM 单条规则校验响应
type llmSingleRuleResponse struct {
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	IsCompliant bool   `json:"isCompliant"`
	Reason      string `json:"reason"`
	Suggestion  string `json:"suggestion"`
}

// llmValidationResponse LLM 校验响应（批量，保留兼容）
type llmValidationResponse struct {
	Results []llmRuleValidationResult `json:"results"`
	Summary string                    `json:"summary"`
}

const singleRuleSystemPrompt = `你是一个排班校验助手。请根据排班结果和规则描述，判断排班是否满足该条语义规则。

请以 JSON 格式返回结果（只返回 JSON，不要有多余文字）：
{
  "ruleId": "规则ID",
  "ruleName": "规则名称",
  "isCompliant": true或false,
  "reason": "判断原因",
  "suggestion": "如不符合，给出修改建议（符合则留空）"
}

注意：
1. 仅根据排班数据和规则描述进行判断
2. 如果无法确定是否符合，标记为符合并说明原因
3. 建议内容仅供参考，不会被自动执行`

// performLLMSemanticValidation 对无法确定性校验的语义规则执行 LLM 辅助校验（只读）
// 并发逐条校验，每条规则独立调用 LLM，结果追加到 ValidationResult 中
func performLLMSemanticValidation(
	ctx context.Context,
	wctx engine.Context,
	createCtx *CreateV4Context,
	validationResult *ValidationResult,
	logger logging.ILogger,
) {
	if len(validationResult.UncheckedRules) == 0 {
		return
	}

	// 获取 AI 服务
	aiFactory, ok := engine.GetService[*ai.AIProviderFactory](wctx, engine.ServiceKeyAIFactory)
	if !ok {
		logger.Warn("AI 服务不可用，跳过 LLM 语义规则校验")
		return
	}

	debugLogger := logging.GetLLMDebugLogger()
	taskTitle := fmt.Sprintf("V4排班_%s_%s", createCtx.StartDate, createCtx.EndDate)

	// 构建排班摘要（所有规则共用）
	scheduleSummary := buildScheduleSummaryForLLM(createCtx)

	// 最大并发数：避免同时发起过多 LLM 请求触发限流
	const maxConcurrency = 5

	rules := validationResult.UncheckedRules
	logger.Info("LLM 语义规则校验开始（并发）", "规则数", len(rules), "最大并发", maxConcurrency)

	type ruleResult struct {
		idx    int
		result *llmSingleRuleResponse
		err    error
	}

	resultCh := make(chan ruleResult, len(rules))
	// semaphore：用带缓冲 channel 限制同时运行的 goroutine 数量
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, rule *UncheckedRule) {
			defer wg.Done()

			// 获取信号量（超过 maxConcurrency 时在此阻塞）
			sem <- struct{}{}
			defer func() { <-sem }()

			// 如果外层 ctx 已取消（整体超时/用户取消），直接跳过
			if ctx.Err() != nil {
				resultCh <- ruleResult{idx: idx, err: ctx.Err()}
				return
			}

			userPrompt := fmt.Sprintf(`## 排班摘要

%s

## 待校验规则

规则ID: %s
规则名称: %s
规则描述: %s
无法确定性校验的原因: %s

请判断排班是否满足上述规则，返回 JSON 格式校验结果。`,
				scheduleSummary,
				rule.RuleID, rule.RuleName, rule.Description, rule.Reason,
			)

			start := time.Now()
			resp, err := aiFactory.CallDefault(ctx, singleRuleSystemPrompt, userPrompt, nil)
			duration := time.Since(start)

			// 记录到 debugllm
			if debugLogger != nil && debugLogger.IsEnabled() {
				respContent := ""
				if err == nil {
					respContent = resp.Content
				}
				debugLogger.LogLLMCall(
					taskTitle,
					logging.LLMCallV4Validation,
					rule.RuleName, // shiftName 复用存放规则名
					"",
					"",
					singleRuleSystemPrompt,
					userPrompt,
					respContent,
					duration,
					err,
				)
			}

			if err != nil {
				logger.Warn("LLM 语义规则校验调用失败", "rule", rule.RuleName, "error", err)
				resultCh <- ruleResult{idx: idx, err: err}
				return
			}

			// 解析单条响应
			r := parseSingleRuleResponse(resp.Content, rule, logger)
			resultCh <- ruleResult{idx: idx, result: r}
		}(i, rule)
	}

	// 等待所有 goroutine 完成后关闭 channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	nonCompliantCount := 0
	for res := range resultCh {
		if res.err != nil || res.result == nil {
			continue
		}
		r := res.result
		if !r.IsCompliant {
			nonCompliantCount++
			validationResult.Warnings = append(validationResult.Warnings, &Warning{
				Type:        "语义规则校验",
				Description: fmt.Sprintf("[%s] %s", r.RuleName, r.Reason),
				Suggestion:  r.Suggestion,
				Severity:    "warning",
			})
		}
	}

	validationResult.LLMValidationDone = true
	createCtx.LLMCallCount += len(rules)

	logger.Info("LLM 语义规则校验完成",
		"规则数", len(rules),
		"不合规数", nonCompliantCount)
}

// buildScheduleSummaryForLLM 构建用于 LLM 校验的排班摘要
func buildScheduleSummaryForLLM(createCtx *CreateV4Context) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("排班周期: %s ~ %s\n", createCtx.StartDate, createCtx.EndDate))
	sb.WriteString(fmt.Sprintf("参与班次: %d 个\n", len(createCtx.SelectedShifts)))
	sb.WriteString(fmt.Sprintf("参与人员: %d 人\n\n", len(createCtx.AllStaff)))

	// 人员名单
	sb.WriteString("**人员名单:** ")
	staffNames := make([]string, 0, len(createCtx.AllStaff))
	for _, s := range createCtx.AllStaff {
		staffNames = append(staffNames, s.Name)
	}
	sb.WriteString(strings.Join(staffNames, "、"))
	sb.WriteString("\n\n")

	// 班次排班详情
	if createCtx.WorkingDraft != nil && createCtx.WorkingDraft.Shifts != nil {
		// 构建班次名称映射
		shiftNameMap := make(map[string]string)
		for _, shift := range createCtx.SelectedShifts {
			shiftNameMap[shift.ID] = shift.Name
		}

		for shiftID, shiftDraft := range createCtx.WorkingDraft.Shifts {
			shiftName := shiftNameMap[shiftID]
			if shiftName == "" {
				shiftName = shiftID
			}
			sb.WriteString(fmt.Sprintf("**%s:**\n", shiftName))

			if shiftDraft.Days == nil {
				sb.WriteString("  无排班数据\n")
				continue
			}

			dates := make([]string, 0, len(shiftDraft.Days))
			for date := range shiftDraft.Days {
				dates = append(dates, date)
			}
			sortStrings(dates)

			for _, date := range dates {
				dayShift := shiftDraft.Days[date]
				if dayShift == nil {
					continue
				}
				names := strings.Join(dayShift.Staff, "、")
				if names == "" {
					names = "(空)"
				}
				sb.WriteString(fmt.Sprintf("  %s: %s (%d/%d人)\n",
					date, names, dayShift.ActualCount, dayShift.RequiredCount))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// parseSingleRuleResponse 解析单条规则 LLM 响应
// 如果解析失败，回退为"符合"（保守策略，不误报）
func parseSingleRuleResponse(content string, rule *UncheckedRule, logger logging.ILogger) *llmSingleRuleResponse {
	jsonStr := extractJSONBlock(content)
	if jsonStr == "" {
		jsonStr = content
	}

	var result llmSingleRuleResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		logger.Warn("解析 LLM 单条规则响应失败，视为符合",
			"rule", rule.RuleName, "error", err, "content", truncateString(content, 200))
		// 解析失败时回退：标记为符合，不产生误报
		return &llmSingleRuleResponse{
			RuleID:      rule.RuleID,
			RuleName:    rule.RuleName,
			IsCompliant: true,
			Reason:      "LLM 响应解析失败，保守标记为符合",
		}
	}

	// 补全规则 ID / 名称（防止 LLM 漏填）
	if result.RuleID == "" {
		result.RuleID = rule.RuleID
	}
	if result.RuleName == "" {
		result.RuleName = rule.RuleName
	}
	return &result
}

// extractJSONBlock 从文本中提取 JSON 代码块
func extractJSONBlock(content string) string {
	// 查找 ```json ... ``` 格式
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + len("```json")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	// 查找 ``` ... ``` 格式
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + len("```")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	// 尝试直接找 { ... } 块
	if start := strings.Index(content, "{"); start != -1 {
		if end := strings.LastIndex(content, "}"); end > start {
			return content[start : end+1]
		}
	}
	return ""
}
