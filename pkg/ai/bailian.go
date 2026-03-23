package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
)

type BailianProvider struct {
	ctx          context.Context
	configurator config.IServiceConfigurator
	client       *http.Client
	name         string
	logger       logging.ILogger
}

// SupportsModel implements AIProvider.
func (p *BailianProvider) SupportsModel(modelName string) bool {
	bailianConfig := p.getBailianConfig()
	if bailianConfig == nil {
		return false
	}
	if len(bailianConfig.Models) > 0 {
		for _, m := range bailianConfig.Models {
			if m.Name == modelName {
				return true
			}
		}
		return false
	}
	return false
}

func (p *BailianProvider) getBailianConfig() *config.ProviderConfig {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Bailian == nil {
		return nil
	}
	return baseCfg.AI.Bailian
}

func (p *BailianProvider) GetName() string {
	return p.name
}

func (p *BailianProvider) GetModel(modelName string) (config.AIModel, error) {
	bailianConfig := p.getBailianConfig()
	if bailianConfig == nil {
		return config.AIModel{}, fmt.Errorf("Bailian config not found")
	}
	for _, model := range bailianConfig.Models {
		if model.Name == modelName {
			return model, nil
		}
	}
	return config.AIModel{}, fmt.Errorf("Model not found: %s", modelName)
}

func (p *BailianProvider) GetAllModels() []string {
	bailianConfig := p.getBailianConfig()
	if bailianConfig == nil {
		return nil
	}
	if len(bailianConfig.Models) > 0 {
		models := make([]string, len(bailianConfig.Models))
		for i, m := range bailianConfig.Models {
			models[i] = m.Name
		}
		return models
	}
	return []string{}
}

// DashScope ChatRequest - 修正为官方文档格式
type dashscopeChatRequest struct {
	Model string `json:"model"`
	Input struct {
		Messages []AIMessage `json:"messages"`
	} `json:"input"`
	Parameters struct {
		ResultFormat string `json:"result_format"`
		Think        bool   `json:"enable_thinking,omitempty"` // 添加思考模式支持
	} `json:"parameters"`
}

// DashScope原生响应格式
type dashscopeChatResponse struct {
	StatusCode int    `json:"status_code"`
	RequestID  string `json:"request_id"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Output     struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
		Choices      []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// 使用正确的DashScope流式API格式
type dashscopeStreamRequest struct {
	Model  string `json:"model"`
	Think  bool   `json:"enable_thinking,omitempty"` // 添加think字段
	Stream bool   `json:"stream,omitempty"`          // 添加stream字段
	Input  struct {
		Messages []AIMessage `json:"messages"`
	} `json:"input"`
	Parameters struct {
		ResultFormat      string `json:"result_format"`
		IncrementalOutput bool   `json:"incremental_output"`
		Think             bool   `json:"enable_thinking,omitempty"` // 添加思考模式支持
	} `json:"parameters"`
}

// DashScope原生流式响应格式
type dashscopeStreamResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
				Role    string `json:"role"`
			} `json:"message"`
			FinishReason *string `json:"finish_reason"` // 改为指针类型，因为可能为null
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		TotalTokens             int            `json:"total_tokens"`
		OutputTokens            int            `json:"output_tokens"`
		InputTokens             int            `json:"input_tokens"`
		PromptTokensDetails     map[string]any `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails map[string]any `json:"completion_tokens_details,omitempty"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// OpenAI兼容的流式响应格式
type openaiStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role         *string `json:"role,omitempty"`
			Content      string  `json:"content,omitempty"`
			FunctionCall *string `json:"function_call,omitempty"`
			Refusal      *string `json:"refusal,omitempty"`
			ToolCalls    *string `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
		Logprobs     *string `json:"logprobs"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	ServiceTier       *string `json:"service_tier"`
	SystemFingerprint *string `json:"system_fingerprint"`
}

func (p *BailianProvider) CallModel(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	cfg := p.getBailianConfig()
	if cfg == nil {
		return AIResponse{}, fmt.Errorf("bailian config is nil")
	}
	var messages []AIMessage
	if sysPrompt != "" {
		messages = append(messages, CreateSystemMessage(sysPrompt))
	}
	if len(history) > 0 {
		messages = append(messages, history...)
	}
	messages = append(messages, CreateUserMessage(prompt))

	reqBody := dashscopeChatRequest{
		Model: modelName,
	}
	reqBody.Input.Messages = messages
	reqBody.Parameters.ResultFormat = "message"
	reqBody.Parameters.Think = think // 添加思考模式支持

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return AIResponse{}, fmt.Errorf("marshal dashscope chat request failed: %w", err)
	}
	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/v1/services/aigc/text-generation/generation"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return AIResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.logger.Error("bailian api request failed", "error", err, "url", url)
		return AIResponse{}, fmt.Errorf("bailian api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("read response body failed", "error", err)
		return AIResponse{}, fmt.Errorf("read response body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		p.logger.Error("bailian api returned error", "status_code", resp.StatusCode, "response", string(respBody))
		return AIResponse{}, fmt.Errorf("bailian api failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp dashscopeChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		p.logger.Error("unmarshal response failed", "error", err, "response", string(respBody))
		return AIResponse{}, fmt.Errorf("unmarshal dashscope chat response failed: %w", err)
	}
	if len(chatResp.Output.Choices) == 0 {
		p.logger.Error("no choices in response", "response", string(respBody))
		return AIResponse{}, fmt.Errorf("dashscope chat response no choices")
	}
	content := chatResp.Output.Choices[0].Message.Content
	return AIResponse{
		Content: content,
		Think:   "",
	}, nil
}

func (p *BailianProvider) CallModelStream(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error) {
	cfg := p.getBailianConfig()
	if cfg == nil {
		ch := make(chan AIResponse, 1)
		close(ch)
		return ch, fmt.Errorf("bailian config is nil")
	}

	var messages []AIMessage
	if sysPrompt != "" {
		messages = append(messages, CreateSystemMessage(sysPrompt))
	}
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// 如果启用think模式，修改用户消息以包含思考提示
	userPrompt := prompt
	messages = append(messages, CreateUserMessage(userPrompt))

	streamReq := dashscopeStreamRequest{
		Stream: true,
		Model:  modelName,
	}
	streamReq.Input.Messages = messages
	streamReq.Parameters.ResultFormat = "message"
	streamReq.Parameters.IncrementalOutput = true
	streamReq.Parameters.Think = think // 添加思考模式支持
	bodyBytes, err := json.Marshal(streamReq)
	if err != nil {
		ch := make(chan AIResponse, 1)
		close(ch)
		return ch, fmt.Errorf("marshal dashscope stream request failed: %w", err)
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/v1/services/aigc/text-generation/generation"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		ch := make(chan AIResponse, 1)
		close(ch)
		return ch, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-DashScope-SSE", "enable")

	ch := make(chan AIResponse, 10)

	go func() {
		defer close(ch)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			p.logger.Error("bailian stream api request failed", "error", err, "url", url)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			p.logger.Error("bailian stream api returned error", "status_code", resp.StatusCode, "response", string(respBody))
			return
		}

		var fullContent strings.Builder
		var fullThink strings.Builder

		// 处理SSE流
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// 跳过空行和注释
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// 处理data行
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)

				// 处理结束标记
				if data == "[DONE]" {
					break
				}

				var streamResp dashscopeStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					p.logger.Warn("failed to parse stream response", "error", err, "data", data)
					continue
				}

				// 检查是否有选择项
				if len(streamResp.Output.Choices) == 0 {
					continue
				}

				choice := streamResp.Output.Choices[0]
				content := choice.Message.Content

				select {
				case ch <- AIResponse{
					Content: content,
					Think:   fullThink.String(),
				}:
				case <-ctx.Done():
					return
				}

				// 检查是否完成 - 只有当finish_reason为"stop"时才真正完成
				if choice.FinishReason != nil && *choice.FinishReason == "stop" {
					break
				}
			}
		}

		if err := scanner.Err(); err != nil {
			p.logger.Error("error reading stream", "error", err)
		}

		// 确保至少发送一次响应，即使内容为空
		finalContent := fullContent.String()
		finalThink := fullThink.String()

		if finalContent == "" && finalThink == "" {
			p.logger.Warn("bailian stream completed with no content")
		} else {
			p.logger.Debug("bailian stream completed", "content_length", len(finalContent), "think_length", len(finalThink))
		}
	}()

	return ch, nil
}

// DashScope Embedding
type dashscopeEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}
type dashscopeEmbeddingResponse struct {
	Output struct {
		Embeddings [][]float32 `json:"embeddings"`
	} `json:"output"`
}

func (p *BailianProvider) Embedding(ctx context.Context, modelName string, text string) ([][]float32, error) {
	cfg := p.getBailianConfig()
	if cfg == nil {
		return nil, fmt.Errorf("Bailian configuration is not set")
	}
	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/v1/services/aigc/embedding/text-embedding/text-embedding"
	reqBody := dashscopeEmbeddingRequest{
		Model: modelName,
		Input: []string{text},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal dashscope embedding request failed: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dashscope embedding api failed: %s", string(respBody))
	}
	var embResp dashscopeEmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("unmarshal dashscope embedding response failed: %w", err)
	}
	if len(embResp.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("dashscope embedding response no data")
	}
	return embResp.Output.Embeddings, nil
}

// DashScope Rerank
type dashscopeRerankRequest struct {
	Model string `json:"model"`
	Input struct {
		Query     string   `json:"query"`
		Documents []string `json:"documents"`
	} `json:"input"`
	Parameters struct {
		TopN            int  `json:"top_n,omitempty"`
		ReturnDocuments bool `json:"return_documents,omitempty"`
	} `json:"parameters,omitempty"`
}
type dashscopeRerankResponse struct {
	Output struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float32 `json:"relevance_score"`
		} `json:"results"`
	} `json:"output"`
}

func (p *BailianProvider) Rerank(ctx context.Context, modelName string, query string, candidates []string) ([]int, []float32, error) {
	cfg := p.getBailianConfig()
	if cfg == nil {
		return nil, nil, fmt.Errorf("Bailian configuration is not set")
	}
	url := strings.TrimRight(cfg.BaseURL, "/") + "/api/v1/services/aigc/rerank/text-rerank/text-rerank"
	reqBody := dashscopeRerankRequest{
		Model: modelName,
	}
	reqBody.Input.Query = query
	reqBody.Input.Documents = candidates
	reqBody.Parameters.TopN = len(candidates)
	reqBody.Parameters.ReturnDocuments = false
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal dashscope rerank request failed: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("dashscope rerank api failed: %s", string(respBody))
	}
	var rerankResp dashscopeRerankResponse
	if err := json.Unmarshal(respBody, &rerankResp); err != nil {
		return nil, nil, fmt.Errorf("unmarshal dashscope rerank response failed: %w", err)
	}
	if len(rerankResp.Output.Results) == 0 {
		return nil, nil, fmt.Errorf("dashscope rerank response no data")
	}
	indices := make([]int, len(rerankResp.Output.Results))
	scores := make([]float32, len(rerankResp.Output.Results))
	for i, r := range rerankResp.Output.Results {
		indices[i] = r.Index
		scores[i] = r.RelevanceScore
	}
	return indices, scores, nil
}

// NewBailianProvider creates a new Bailian AI provider.
// It now takes a IServiceConfigurator to dynamically fetch Bailian-specific configurations.
func NewBailianProvider(ctx context.Context, configurator config.IServiceConfigurator, logger logging.ILogger) (AIProvider, error) {
	baseCfg := configurator.GetBaseConfig()
	if baseCfg.AI == nil {
		return nil, fmt.Errorf("bailian provider base AI config is nil")
	}
	bailianConfig := baseCfg.AI.Bailian
	if bailianConfig == nil {
		return nil, fmt.Errorf("bailian provider config is nil")
	}
	if bailianConfig.APIKey == "" {
		return nil, fmt.Errorf("bailian api_key is required")
	}
	if bailianConfig.BaseURL == "" {
		return nil, fmt.Errorf("bailian base_url is required")
	}
	return &BailianProvider{
		ctx:          ctx,
		configurator: configurator,
		client:       &http.Client{},
		name:         "bailian",
		logger:       logger.With("provider", "bailian"),
	}, nil
}
