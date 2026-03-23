package ai

import (
	"context"
	"fmt"
	"strings"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type OpenAIProvider struct {
	ctx           context.Context
	configurator  config.IServiceConfigurator
	client        openai.Client
	name          string
	logger        logging.ILogger
	historyNumber int
}

// SupportsModel implements AIProvider.
func (p *OpenAIProvider) SupportsModel(modelName string) bool {
	cfg := p.configurator.GetBaseConfig()
	if cfg.AI == nil {
		return false
	}
	openaiConfig := cfg.AI.OpenAI
	if openaiConfig == nil {
		return false
	}
	if len(openaiConfig.Models) > 0 {
		for _, m := range openaiConfig.Models {
			if m.Name == modelName {
				return true
			}
		}
		return false
	}
	return false
}

func (p *OpenAIProvider) GetName() string {
	return p.name
}

func (p *OpenAIProvider) GetModel(modelName string) (config.AIModel, error) {
	cfg := p.configurator.GetBaseConfig()
	if cfg.AI == nil {
		return config.AIModel{}, fmt.Errorf("AI config is nil")
	}
	openaiConfig := cfg.AI.OpenAI
	if openaiConfig == nil {
		return config.AIModel{}, fmt.Errorf("OpenAI config is nil")
	}
	for _, model := range openaiConfig.Models {
		if model.Name == modelName {
			return model, nil
		}
	}
	return config.AIModel{}, fmt.Errorf("model %s not found", modelName)
}

func (p *OpenAIProvider) GetAllModels() []string {
	cfg := p.configurator.GetBaseConfig()
	if cfg.AI == nil {
		return nil
	}
	openaiConfig := cfg.AI.OpenAI
	if openaiConfig == nil {
		return nil
	}
	if len(openaiConfig.Models) > 0 {
		models := make([]string, len(openaiConfig.Models))
		for i, m := range openaiConfig.Models {
			models[i] = m.Name
		}
		return models
	}
	return []string{}
}

func (p *OpenAIProvider) CallModel(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	// 构造消息历史
	var messages []openai.ChatCompletionMessageParamUnion
	if sysPrompt != "" {
		messages = append(messages, openai.SystemMessage(sysPrompt))
	}
	for _, h := range history {
		switch h.Role {
		case "user":
			messages = append(messages, openai.UserMessage(h.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(h.Content))
		case "system":
			messages = append(messages, openai.SystemMessage(h.Content))
		}
	}
	messages = append(messages, openai.UserMessage(prompt))

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    modelName,
	}

	// 如果启用深度思考，设置推理参数
	if think {
		// 使用 ReasoningEffort 参数来控制推理深度
		params.ReasoningEffort = shared.ReasoningEffort("high")
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return AIResponse{}, fmt.Errorf("openai call model failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return AIResponse{}, fmt.Errorf("openai response no choices")
	}
	return AIResponse{
		Content: resp.Choices[0].Message.Content,
		Think:   "",
	}, nil
}

func (p *OpenAIProvider) CallModelStream(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error) {
	// 构造消息历史
	var messages []openai.ChatCompletionMessageParamUnion
	if sysPrompt != "" {
		messages = append(messages, openai.SystemMessage(sysPrompt))
	}
	for _, h := range history {
		switch h.Role {
		case "user":
			messages = append(messages, openai.UserMessage(h.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(h.Content))
		case "system":
			messages = append(messages, openai.SystemMessage(h.Content))
		}
	}

	// 如果启用think模式，修改用户消息以包含思考提示
	userPrompt := prompt
	messages = append(messages, openai.UserMessage(userPrompt))

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    modelName,
	}

	// 如果启用深度思考，设置推理参数
	if think {
		// 使用 ReasoningEffort 参数来控制推理深度
		params.ReasoningEffort = shared.ReasoningEffort("high")
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	ch := make(chan AIResponse, 10)

	go func() {
		defer close(ch)
		defer stream.Close()

		var fullContent strings.Builder
		var fullThink strings.Builder
		var inThinkTag bool

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta

				// 处理常规内容
				if delta.Content != "" {
					content := delta.Content

					if think {
						// 解析think标签
						for {
							if !inThinkTag {
								// 寻找<think>标签
								if idx := strings.Index(content, "<think>"); idx != -1 {
									// 发送think标签之前的内容作为答案
									beforeThink := content[:idx]
									if beforeThink != "" {
										fullContent.WriteString(beforeThink)
									}

									// 切换到think模式
									inThinkTag = true
									content = content[idx+7:] // 移除<think>标签
									continue
								} else {
									// 没有think标签，全部作为答案内容
									fullContent.WriteString(content)
									break
								}
							} else {
								// 在think标签内，寻找</think>标签
								if idx := strings.Index(content, "</think>"); idx != -1 {
									// 发送think内容
									thinkContent := content[:idx]
									if thinkContent != "" {
										fullThink.WriteString(thinkContent)
									}

									// 切换回答案模式
									inThinkTag = false
									content = content[idx+8:] // 移除</think>标签
									continue
								} else {
									// 没有结束标签，全部作为think内容
									fullThink.WriteString(content)
									break
								}
							}
						}
					} else {
						fullContent.WriteString(content)
					}

					// 发送增量响应
					select {
					case ch <- AIResponse{
						Content: fullContent.String(),
						Think:   fullThink.String(),
					}:
					case <-ctx.Done():
						p.logger.Debug("stream context cancelled")
						return
					}
				}
			}
		}

		// 检查流处理是否出错
		if err := stream.Err(); err != nil {
			p.logger.Error("stream processing error", "error", err)
			return
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) Embedding(ctx context.Context, modelName string, text string) ([][]float32, error) {
	params := openai.EmbeddingNewParams{
		Model: modelName,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
	}
	resp, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai embedding failed: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai embedding returned no data")
	}
	embeddings := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		// openai-go 返回 float64，需转换为 float32
		vec := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			vec[j] = float32(v)
		}
		embeddings[i] = vec
	}
	return embeddings, nil
}

func (p *OpenAIProvider) Rerank(ctx context.Context, modelName string, query string, candidates []string) ([]int, []float32, error) {
	// OpenAI官方API无rerank，DashScope兼容模式可实现，若无则返回未实现
	return nil, nil, fmt.Errorf("openai rerank not implemented")
}

func NewOpenAIProvider(ctx context.Context, configurator config.IServiceConfigurator, logger logging.ILogger) (AIProvider, error) {
	cfg := configurator.GetBaseConfig()
	if cfg.AI == nil {
		return nil, fmt.Errorf("openai provider config is nil")
	}
	openaiConfig := cfg.AI.OpenAI
	if openaiConfig == nil {
		return nil, fmt.Errorf("openai provider config is nil")
	}
	if openaiConfig.APIKey == "" {
		return nil, fmt.Errorf("openai api_key is required")
	}
	if openaiConfig.BaseURL == "" {
		return nil, fmt.Errorf("openai base_url is required")
	}

	client := openai.NewClient(
		option.WithAPIKey(openaiConfig.APIKey),
		option.WithBaseURL(openaiConfig.BaseURL),
	)

	return &OpenAIProvider{
		ctx:           ctx,
		configurator:  configurator,
		client:        client,
		name:          "openai",
		historyNumber: cfg.AI.HistoryNumber,
		logger:        logger.With("provider", "openai"),
	}, nil
}
