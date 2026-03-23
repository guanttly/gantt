package toolcalling

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
)

// IToolCallingService AI工具调用服务接口
type IToolCallingService interface {
	// RegisterTool 注册工具
	RegisterTool(tool ITool) error
	
	// UnregisterTool 注销工具
	UnregisterTool(name string) error
	
	// GetTool 获取工具
	GetTool(name string) (ITool, error)
	
	// ListTools 列出所有工具
	ListTools() []ITool
	
	// CallTool 直接调用工具
	CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
	
	// ExecuteWithTools 使用工具执行AI任务（支持多轮工具调用）
	ExecuteWithTools(
		ctx context.Context,
		aiFactory *ai.AIProviderFactory,
		modelConfig *config.AIModelProvider,
		sysPrompt, userPrompt string,
		tools []ITool,
		maxIterations int,
	) (*AIExecutionResult, error)
}

// AIExecutionResult AI执行结果（包含工具调用）
type AIExecutionResult struct {
	// FinalResponse 最终AI响应
	FinalResponse string `json:"finalResponse"`
	
	// ToolCalls 工具调用历史
	ToolCalls []ToolCallRecord `json:"toolCalls,omitempty"`
	
	// Iterations 迭代次数
	Iterations int `json:"iterations"`
}

// ToolCallRecord 工具调用记录
type ToolCallRecord struct {
	// ToolName 工具名称
	ToolName string `json:"toolName"`
	
	// Arguments 调用参数
	Arguments map[string]any `json:"arguments"`
	
	// Result 执行结果
	Result *ToolResult `json:"result"`
	
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}

// ToolCallingService AI工具调用服务实现
type ToolCallingService struct {
	logger    logging.ILogger
	tools     map[string]ITool
	toolsMu   sync.RWMutex
	maxIterations int // 最大迭代次数（防止无限循环）
}

// NewToolCallingService 创建工具调用服务
func NewToolCallingService(logger logging.ILogger) IToolCallingService {
	if logger == nil {
		// 如果没有提供 logger，创建一个默认的
		logger = logging.NewDefaultLogger()
	}
	return &ToolCallingService{
		logger:        logger.With("component", "ToolCallingService"),
		tools:         make(map[string]ITool),
		maxIterations: 10, // 默认最大10轮
	}
}

// RegisterTool 注册工具
func (s *ToolCallingService) RegisterTool(tool ITool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}
	
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()
	
	if _, exists := s.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	s.tools[name] = tool
	s.logger.Info("Tool registered", "name", name)
	return nil
}

// UnregisterTool 注销工具
func (s *ToolCallingService) UnregisterTool(name string) error {
	s.toolsMu.Lock()
	defer s.toolsMu.Unlock()
	
	if _, exists := s.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}
	
	delete(s.tools, name)
	s.logger.Info("Tool unregistered", "name", name)
	return nil
}

// GetTool 获取工具
func (s *ToolCallingService) GetTool(name string) (ITool, error) {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	
	tool, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	
	return tool, nil
}

// ListTools 列出所有工具
func (s *ToolCallingService) ListTools() []ITool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	
	tools := make([]ITool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	
	return tools
}

// CallTool 直接调用工具
func (s *ToolCallingService) CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error) {
	tool, err := s.GetTool(name)
	if err != nil {
		return nil, err
	}
	
	s.logger.Debug("Calling tool", "name", name, "args", args)
	result, err := tool.Execute(ctx, args)
	if err != nil {
		s.logger.Error("Tool execution failed", "name", name, "error", err)
		return NewErrorResult(err), err
	}
	
	s.logger.Debug("Tool executed successfully", "name", name)
	return result, nil
}

// ExecuteWithTools 使用工具执行AI任务（支持多轮工具调用）
// 实现基于文本解析的工具调用机制：AI在响应中以JSON格式返回工具调用请求
func (s *ToolCallingService) ExecuteWithTools(
	ctx context.Context,
	aiFactory *ai.AIProviderFactory,
	modelConfig *config.AIModelProvider,
	sysPrompt, userPrompt string,
	tools []ITool,
	maxIterations int,
) (*AIExecutionResult, error) {
	if maxIterations <= 0 {
		maxIterations = s.maxIterations
	}

	// 构建工具描述（用于AI理解可用工具）
	toolsDescription := s.buildToolsDescription(tools)
	
	// 构建工具调用格式说明
	toolCallFormat := s.buildToolCallFormat(tools)
	
	// 增强系统提示词，包含工具说明和调用格式
	enhancedSysPrompt := sysPrompt + "\n\n**可用工具**：\n" + toolsDescription
	enhancedSysPrompt += "\n\n**工具调用格式**：\n" + toolCallFormat
	enhancedSysPrompt += "\n\n**重要**：如果需要使用工具，请在响应中以JSON格式返回工具调用请求。"
	enhancedSysPrompt += "如果不需要工具或工具调用已完成，直接返回最终答案。"

	// 构建工具名称映射，用于快速查找
	toolMap := make(map[string]ITool)
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	// 构建对话历史
	history := make([]ai.AIMessage, 0)
	currentPrompt := userPrompt

	result := &AIExecutionResult{
		ToolCalls: make([]ToolCallRecord, 0),
		Iterations: 0,
	}

	// 多轮迭代：直到AI不再调用工具或达到最大迭代次数
	for iteration := 0; iteration < maxIterations; iteration++ {
		result.Iterations++

		// 调用AI
		response, err := aiFactory.CallWithModel(ctx, modelConfig, enhancedSysPrompt, currentPrompt, history)
		if err != nil {
			return nil, fmt.Errorf("AI call failed (iteration %d): %w", iteration+1, err)
		}

		// 尝试解析工具调用请求
		toolCalls, hasToolCalls, finalAnswer := s.parseToolCallsFromResponse(response.Content, toolMap)
		
		if !hasToolCalls {
			// 没有工具调用，AI返回最终答案
			result.FinalResponse = response.Content
			s.logger.Info("AI returned final answer", "iteration", iteration+1)
			break
		}

		// 执行工具调用
		toolResults := make([]string, 0)
		for _, toolCall := range toolCalls {
			toolRecord := ToolCallRecord{
				ToolName:  toolCall.ToolName,
				Arguments: toolCall.Arguments,
			}

			// 执行工具
			tool, exists := toolMap[toolCall.ToolName]
			if !exists {
				err := fmt.Errorf("tool %s not found", toolCall.ToolName)
				toolRecord.Error = err.Error()
				errorMsg := fmt.Sprintf("工具 %s 不存在或未注册。可用工具：%v", 
					toolCall.ToolName, s.getAvailableToolNames(toolMap))
				toolResults = append(toolResults, errorMsg)
				s.logger.Warn("Tool not found", 
					"toolName", toolCall.ToolName, 
					"availableTools", s.getAvailableToolNames(toolMap),
					"arguments", toolCall.Arguments)
				result.ToolCalls = append(result.ToolCalls, toolRecord)
				continue
			}

			s.logger.Debug("Executing tool", 
				"tool", toolCall.ToolName, 
				"arguments", toolCall.Arguments,
				"iteration", iteration+1)
			
			toolResult, err := tool.Execute(ctx, toolCall.Arguments)
			if err != nil {
				toolRecord.Error = err.Error()
				toolRecord.Result = NewErrorResult(err)
				errorMsg := fmt.Sprintf("工具 %s 执行失败：%s\n参数：%v", 
					toolCall.ToolName, err.Error(), toolCall.Arguments)
				toolResults = append(toolResults, errorMsg)
				s.logger.Error("Tool execution failed", 
					"tool", toolCall.ToolName, 
					"error", err,
					"arguments", toolCall.Arguments,
					"errorType", fmt.Sprintf("%T", err))
			} else {
				toolRecord.Result = toolResult
				// 格式化工具结果用于反馈给AI
				resultStr := s.formatToolResult(toolCall.ToolName, toolResult)
				toolResults = append(toolResults, resultStr)
				s.logger.Info("Tool executed successfully", 
					"tool", toolCall.ToolName, 
					"iteration", iteration+1,
					"hasResult", toolResult != nil)
			}

			result.ToolCalls = append(result.ToolCalls, toolRecord)
		}

		// 将工具结果反馈给AI，继续对话
		if finalAnswer != "" {
			currentPrompt = finalAnswer + "\n\n工具调用结果：\n" + strings.Join(toolResults, "\n\n")
		} else {
			currentPrompt = "工具调用结果：\n" + strings.Join(toolResults, "\n\n") + "\n\n请根据工具调用结果继续处理任务。"
		}

		// 添加到对话历史
		history = append(history, ai.AIMessage{
			Role:    "assistant",
			Content: response.Content,
		})
		history = append(history, ai.AIMessage{
			Role:    "user",
			Content: currentPrompt,
		})
	}

	// 如果达到最大迭代次数仍未得到最终答案，使用最后一轮的响应
	if result.FinalResponse == "" {
		response, err := aiFactory.CallWithModel(ctx, modelConfig, enhancedSysPrompt, currentPrompt, history)
		if err == nil {
			result.FinalResponse = response.Content
		} else {
			result.FinalResponse = "任务执行超时或失败"
		}
	}

	s.logger.Info("Tool calling execution completed", 
		"iterations", result.Iterations,
		"toolCalls", len(result.ToolCalls))

	return result, nil
}

// buildToolsDescription 构建工具描述（用于AI理解）
func (s *ToolCallingService) buildToolsDescription(tools []ITool) string {
	var desc strings.Builder
	for i, tool := range tools {
		if i > 0 {
			desc.WriteString("\n")
		}
		desc.WriteString(fmt.Sprintf("- **%s**: %s", tool.Name(), tool.Description()))
		
		// 添加参数模式
		if schema := tool.InputSchema(); schema != nil {
			if schemaJSON, err := json.MarshalIndent(schema, "", "  "); err == nil {
				desc.WriteString(fmt.Sprintf("\n  参数模式：\n```json\n%s\n```", string(schemaJSON)))
			}
		}
	}
	return desc.String()
}

// buildToolCallFormat 构建工具调用格式说明
func (s *ToolCallingService) buildToolCallFormat(tools []ITool) string {
	return `如果需要调用工具，请在响应中包含以下格式的JSON：

` + "```json" + `
{
  "toolCalls": [
    {
      "toolName": "工具名称",
      "arguments": {
        "参数名": "参数值"
      }
    }
  ],
  "finalAnswer": "可选的最终答案（如果有部分答案可先返回）"
}
` + "```" + `

如果不需要调用工具，直接返回最终答案即可。`
}

// ToolCallRequest 工具调用请求（从AI响应中解析）
type ToolCallRequest struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
}

// ToolCallResponse AI响应中的工具调用格式
type ToolCallResponse struct {
	ToolCalls  []ToolCallRequest `json:"toolCalls"`
	FinalAnswer string           `json:"finalAnswer,omitempty"`
}

// parseToolCallsFromResponse 从AI响应中解析工具调用请求
// 返回：工具调用列表、是否有工具调用、最终答案（如果有）
func (s *ToolCallingService) parseToolCallsFromResponse(
	response string,
	toolMap map[string]ITool,
) ([]ToolCallRequest, bool, string) {
	var jsonStr string
	
	// 方法1: 尝试提取 ```json 代码块中的完整内容
	codeBlockPattern := regexp.MustCompile(`(?s)` + "```json" + `\s*(.+?)` + "```")
	codeBlockMatches := codeBlockPattern.FindStringSubmatch(response)
	if len(codeBlockMatches) > 1 {
		jsonStr = strings.TrimSpace(codeBlockMatches[1])
		s.logger.Debug("Extracted JSON from code block", "length", len(jsonStr))
	}
	
	// 方法2: 如果没找到代码块，尝试查找 JSON 对象（支持嵌套）
	if jsonStr == "" {
		jsonStr = s.extractJSONObject(response)
	}
	
	if jsonStr == "" {
		s.logger.Debug("No JSON found in response")
		return nil, false, ""
	}

	// 解析JSON
	var toolCallResp ToolCallResponse
	if err := json.Unmarshal([]byte(jsonStr), &toolCallResp); err != nil {
		s.logger.Debug("Failed to parse tool call JSON", "error", err, "json", jsonStr)
		// 尝试清理 JSON 字符串（移除可能的 markdown 格式）
		cleanedJSON := s.cleanJSONString(jsonStr)
		if err2 := json.Unmarshal([]byte(cleanedJSON), &toolCallResp); err2 != nil {
			s.logger.Warn("Failed to parse tool call JSON after cleaning", "error", err2)
			return nil, false, ""
		}
	}

	if len(toolCallResp.ToolCalls) == 0 {
		return nil, false, toolCallResp.FinalAnswer
	}

	// 验证工具是否存在
	validToolCalls := make([]ToolCallRequest, 0)
	for _, toolCall := range toolCallResp.ToolCalls {
		if _, exists := toolMap[toolCall.ToolName]; exists {
			validToolCalls = append(validToolCalls, toolCall)
		} else {
			s.logger.Warn("Tool not found in tool map", "toolName", toolCall.ToolName)
		}
	}

	if len(validToolCalls) == 0 {
		return nil, false, toolCallResp.FinalAnswer
	}

	return validToolCalls, true, toolCallResp.FinalAnswer
}

// extractJSONObject 从文本中提取完整的 JSON 对象（支持嵌套）
func (s *ToolCallingService) extractJSONObject(text string) string {
	// 查找包含 "toolCalls" 的 JSON 对象
	startIdx := strings.Index(text, `{"toolCalls"`)
	if startIdx == -1 {
		startIdx = strings.Index(text, `{"toolcalls"`)
	}
	if startIdx == -1 {
		return ""
	}
	
	// 从找到的位置开始，查找完整的 JSON 对象
	// 使用括号计数来找到匹配的 }
	depth := 0
	for i := startIdx; i < len(text); i++ {
		if text[i] == '{' {
			depth++
		} else if text[i] == '}' {
			depth--
			if depth == 0 {
				// 找到了完整的 JSON 对象
				return text[startIdx : i+1]
			}
		}
	}
	
	return ""
}

// cleanJSONString 清理 JSON 字符串，移除可能的 markdown 格式
func (s *ToolCallingService) cleanJSONString(jsonStr string) string {
	// 移除可能的 markdown 代码块标记
	cleaned := strings.TrimSpace(jsonStr)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

// formatToolResult 格式化工具执行结果（用于反馈给AI）
func (s *ToolCallingService) formatToolResult(toolName string, result *ToolResult) string {
	if result == nil {
		return fmt.Sprintf("工具 %s 返回空结果", toolName)
	}

	if result.IsError {
		return fmt.Sprintf("工具 %s 执行失败：%s", toolName, result.Content)
	}

	return fmt.Sprintf("工具 %s 执行成功：\n%s", toolName, result.Content)
}

// getAvailableToolNames 获取可用工具名称列表
func (s *ToolCallingService) getAvailableToolNames(toolMap map[string]ITool) []string {
	names := make([]string, 0, len(toolMap))
	for name := range toolMap {
		names = append(names, name)
	}
	return names
}
