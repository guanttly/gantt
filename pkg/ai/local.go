package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"
)

type LocalProvider struct {
	ctx          context.Context
	configurator config.IServiceConfigurator
	client       *http.Client
	name         string
	logger       logging.ILogger
}

// ChatRequest 聊天请求结构 - 使用models.go中定义的结构
// 注意：这里重新定义是为了匹配HuggingFace API格式
type LocalChatRequest struct {
	Messages    []AIMessage `json:"messages"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
	Think       bool        `json:"think,omitempty"` // 添加think字段
	Model       string      `json:"model"`
	Stream      bool        `json:"stream,omitempty"`
}

// ChatResponse 聊天响应结构 - 包含think和content字段
type LocalChatResponse struct {
	Content string `json:"content"`
	Think   string `json:"think,omitempty"` // 添加think字段
}

// EmbeddingRequest 向量化请求结构
type EmbeddingRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model"`
}

// EmbeddingResponse 向量化响应结构
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// CallModel implements AIProvider.
func (p *LocalProvider) CallModel(ctx context.Context, modelName string, think bool, sysPrompt string, prompt string, history []AIMessage) (AIResponse, error) {
	cfg := p.getConfig()
	if cfg == nil {
		return AIResponse{}, fmt.Errorf("local config is nil")
	}

	baseUrl := cfg.BaseURL
	if baseUrl == "" {
		return AIResponse{}, fmt.Errorf("base URL is not configured for local provider")
	}

	// 构造消息列表，使用models.go中定义的AIMessage结构
	messages := make([]AIMessage, 0)

	// 添加系统消息
	if sysPrompt != "" {
		messages = append(messages, CreateSystemMessage(sysPrompt))
	}

	// 添加历史消息
	messages = append(messages, history...)

	// 添加当前用户消息
	messages = append(messages, CreateUserMessage(prompt))

	// 构造请求体
	chatReq := LocalChatRequest{
		Messages:    messages,
		MaxTokens:   32768,
		Temperature: 0.7,
		Model:       modelName,
		Think:       think,
		Stream:      false,
	}

	url := fmt.Sprintf("%s/chat", baseUrl)
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return AIResponse{}, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	// 发送HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return AIResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return AIResponse{}, fmt.Errorf("failed to send chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AIResponse{}, fmt.Errorf("chat request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var chatResp LocalChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return AIResponse{}, fmt.Errorf("failed to decode chat response: %w", err)
	}

	if think && chatResp.Think == "" {
		// 如果需要think但没有提供，尝试从内容中提取
		p.logger.Warn("Think requested but not provided in response, extracting from content")

		eThink, eContent := utils.ParseAIContent(chatResp.Content)
		return AIResponse{
			Content: eContent,
			Think:   eThink,
		}, nil
	}

	return AIResponse{
		Content: chatResp.Content,
		Think:   chatResp.Think,
	}, nil
}

// CallModelStream implements AIProvider.
func (p *LocalProvider) CallModelStream(ctx context.Context, modelName string, think bool, sysPrompt string, prompt string, history []AIMessage) (chan AIResponse, error) {
	respChan := make(chan AIResponse, 10)

	go func() {
		defer close(respChan)

		// 对于流式调用，我们需要处理分块响应
		cfg := p.getConfig()
		if cfg == nil {
			p.logger.Error("Stream call model failed: local config is nil")
			return
		}

		baseUrl := cfg.BaseURL
		if baseUrl == "" {
			p.logger.Error("Stream call model failed: base URL is not configured")
			return
		}

		// 构造消息列表
		messages := make([]AIMessage, 0)
		if sysPrompt != "" {
			messages = append(messages, CreateSystemMessage(sysPrompt))
		}
		messages = append(messages, history...)
		messages = append(messages, CreateUserMessage(prompt))

		// 构造流式请求体
		chatReq := LocalChatRequest{
			Messages:    messages,
			MaxTokens:   2048,
			Temperature: 0.7,
			Model:       modelName,
			Think:       think,
			Stream:      true, // 启用流式
		}

		url := fmt.Sprintf("%s/chat", baseUrl)
		reqBody, err := json.Marshal(chatReq)
		if err != nil {
			p.logger.Error("Stream call model failed to marshal request", "error", err)
			return
		}

		// 发送HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			p.logger.Error("Stream call model failed to create request", "error", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := p.client.Do(httpReq)
		if err != nil {
			p.logger.Error("Stream call model failed to send request", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			p.logger.Error("Stream call model failed", "status", resp.StatusCode)
			return
		}

		// 处理流式响应
		var fullContent strings.Builder
		var extractedThink string

		decoder := json.NewDecoder(resp.Body)
	streamLoop:
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var streamResp struct {
					Content string `json:"content"`
					Think   string `json:"think,omitempty"`
					Done    bool   `json:"done,omitempty"`
				}

				if err := decoder.Decode(&streamResp); err != nil {
					// 流结束或解析错误
					break streamLoop
				}

				if streamResp.Think != "" && extractedThink == "" {
					extractedThink = streamResp.Think
				}

				fullContent.WriteString(streamResp.Content)

				// 发送增量响应
				select {
				case respChan <- AIResponse{
					Content: fullContent.String(),
					Think:   extractedThink,
				}:
				case <-ctx.Done():
					return
				}

				if streamResp.Done {
					break streamLoop
				}
			}
		}
	}()

	return respChan, nil
}

// Embedding implements AIProvider.
func (p *LocalProvider) Embedding(ctx context.Context, modelName string, prompt string) ([][]float32, error) {
	cfg := p.getConfig()
	if cfg == nil {
		return nil, fmt.Errorf("local config is nil")
	}

	baseUrl := cfg.BaseURL
	if baseUrl == "" {
		return nil, fmt.Errorf("base URL is not configured for local provider")
	}

	url := fmt.Sprintf("%s/embedding", baseUrl)

	// 构造请求体
	embeddingReq := EmbeddingRequest{
		Texts: []string{prompt},
		Model: modelName,
	}

	reqBody, err := json.Marshal(embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// 发送HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return embeddingResp.Embeddings, nil
}

func (p *LocalProvider) GetModel(modelName string) (config.AIModel, error) {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Local == nil {
		return config.AIModel{}, fmt.Errorf("local config is nil")
	}
	for _, model := range baseCfg.AI.Local.Models {
		if model.Name == modelName {
			return model, nil
		}
	}
	return config.AIModel{}, fmt.Errorf("model not found: %s", modelName)
}

// GetAllModels implements AIProvider.
func (p *LocalProvider) GetAllModels() []string {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Local == nil {
		return nil
	}
	if len(baseCfg.AI.Local.Models) > 0 {
		models := make([]string, len(baseCfg.AI.Local.Models))
		for i, m := range baseCfg.AI.Local.Models {
			models[i] = m.Name
		}
		return models
	}
	return nil
}

// GetName implements AIProvider.
func (p *LocalProvider) GetName() string {
	return p.name
}

// SupportsModel implements AIProvider.
func (p *LocalProvider) SupportsModel(modelName string) bool {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Local == nil {
		return false
	}
	if len(baseCfg.AI.Local.Models) > 0 {
		for _, m := range baseCfg.AI.Local.Models {
			if m.Name == modelName {
				return true
			}
		}
		return false
	}
	return false
}

type RerankRequest struct {
	Query    string   `json:"query"`
	Passages []string `json:"passages"`
	TopK     *int     `json:"top_k,omitempty"`
	Model    string   `json:"model"`
}

type RerankResponse struct {
	Results []map[string]any `json:"results"`
}

func (p *LocalProvider) Rerank(ctx context.Context, modelName string, query string, candidates []string) ([]int, []float32, error) {
	// 本地模型调用本地部署的hagging face模型
	cfg := p.getConfig()
	if cfg == nil {
		return nil, nil, fmt.Errorf("local config is nil")
	}
	if len(cfg.Models) == 0 {
		return nil, nil, fmt.Errorf("no models configured for local provider")
	}

	baseUrl := cfg.BaseURL
	if baseUrl == "" {
		return nil, nil, fmt.Errorf("base URL is not configured for local provider")
	}
	url := fmt.Sprintf("%s/rerank", baseUrl)

	// 构造请求体
	rerankReq := RerankRequest{
		Query:    query,
		Passages: candidates,
		Model:    modelName,
	}

	reqBody, err := json.Marshal(rerankReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal rerank request: %w", err)
	}

	// 发送HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send rerank request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("rerank request failed with status: %d", resp.StatusCode)
	}

	// 解析响应
	var rerankResp RerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode rerank response: %w", err)
	}

	// 提取索引和分数
	indices := make([]int, len(rerankResp.Results))
	scores := make([]float32, len(rerankResp.Results))

	for i, result := range rerankResp.Results {
		if idx, ok := result["index"].(float64); ok {
			indices[i] = int(idx)
		}
		if score, ok := result["score"].(float64); ok {
			scores[i] = float32(score)
		}
	}

	return indices, scores, nil
}

func (p *LocalProvider) getConfig() *config.ProviderConfig {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Local == nil {
		return nil
	}
	return baseCfg.AI.Local
}

// NewLocalProvider creates a new Local AI provider.
// It now takes a IServiceConfigurator to dynamically fetch Local-specific configurations.
func NewLocalProvider(ctx context.Context, configurator config.IServiceConfigurator, logger logging.ILogger) (AIProvider, error) {

	provider := &LocalProvider{
		ctx:          ctx,
		configurator: configurator,
		client:       &http.Client{},
		name:         "LocalProvider",
		logger:       logger,
	}
	return provider, nil
}
