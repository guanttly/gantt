package toolcalling

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"
)

// ToolCallingProvider AI Provider 包装器，支持 Function Calling
type ToolCallingProvider struct {
	provider     ai.AIProvider
	toolService  IToolCallingService
	logger       logging.ILogger
	maxIterations int
}

// NewToolCallingProvider 创建支持工具调用的 AI Provider 包装器
func NewToolCallingProvider(
	provider ai.AIProvider,
	toolService IToolCallingService,
	logger logging.ILogger,
) *ToolCallingProvider {
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}
	return &ToolCallingProvider{
		provider:      provider,
		toolService:   toolService,
		logger:        logger.With("component", "ToolCallingProvider"),
		maxIterations: 10,
	}
}

// CallModelWithTools 使用工具调用模型
// 注意：这是一个简化实现，实际需要根据具体的 AI Provider 的 Function Calling API 来实现
func (p *ToolCallingProvider) CallModelWithTools(
	ctx context.Context,
	modelName string,
	think bool,
	sysPrompt, userPrompt string,
	history []ai.AIMessage,
	tools []ITool,
) (*AIExecutionResult, error) {
	// 构建工具描述
	toolsDescription := p.buildToolsDescription(tools)
	
	// 增强系统提示词
	enhancedSysPrompt := sysPrompt + "\n\n**可用工具**：\n" + toolsDescription
	enhancedSysPrompt += "\n\n**工具使用说明**：如果需要使用工具，请在响应中明确说明要调用的工具名称和参数。"
	
	// 执行多轮对话
	// 注意：真正的 Function Calling 逻辑在 toolcalling.Service.ExecuteWithTools 中实现
	// 这里只是简单的模型调用包装器，实际的工具调用解析和执行由 Service 层处理
	
	response, err := p.provider.CallModel(ctx, modelName, think, enhancedSysPrompt, userPrompt, history)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}
	
	result := &AIExecutionResult{
		FinalResponse: response.Content,
		ToolCalls:     []ToolCallRecord{},
		Iterations:    1,
	}
	
	return result, nil
}

// buildToolsDescription 构建工具描述
func (p *ToolCallingProvider) buildToolsDescription(tools []ITool) string {
	var desc string
	for i, tool := range tools {
		if i > 0 {
			desc += "\n"
		}
		desc += fmt.Sprintf("- **%s**: %s", tool.Name(), tool.Description())
		
		// 添加参数说明
		if schema := tool.InputSchema(); schema != nil {
			if schemaJSON, err := json.MarshalIndent(schema, "", "  "); err == nil {
				desc += fmt.Sprintf("\n  参数模式：\n```json\n%s\n```", string(schemaJSON))
			}
		}
	}
	return desc
}

// GetProvider 获取底层 Provider
func (p *ToolCallingProvider) GetProvider() ai.AIProvider {
	return p.provider
}
