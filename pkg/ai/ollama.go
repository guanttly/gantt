package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"
)

type OllamaProvider struct {
	ctx          context.Context
	configurator config.IServiceConfigurator
	name         string
	logger       logging.ILogger
	httpClient   *http.Client
}

// HTTP请求和响应结构体
type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Think    bool            `json:"think,omitempty"`
	Options  map[string]any  `json:"options,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt time.Time     `json:"created_at"`
	Message   OllamaMessage `json:"message"`
	Done      bool          `json:"done"`
}

type OllamaEmbedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"`
}

type OllamaEmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
}

type OllamaRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type OllamaRerankResponse struct {
	Model   string `json:"model"`
	Results []struct {
		Index int     `json:"index"`
		Score float32 `json:"relevance_score"`
	} `json:"results"`
}

// SupportsModel implements AIProvider.
func (p *OllamaProvider) SupportsModel(modelName string) bool {
	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil {
		return false
	}
	if len(ollamaConfig.Models) > 0 {
		for _, m := range ollamaConfig.Models {
			if m.Name == modelName {
				return true
			}
		}
		return false
	}
	return false
}

func (p *OllamaProvider) GetName() string {
	return p.name
}

func (p *OllamaProvider) getOllamaConfig() *config.ProviderConfig {
	baseCfg := p.configurator.GetBaseConfig()
	if baseCfg.AI == nil || baseCfg.AI.Ollama == nil {
		return nil
	}
	return baseCfg.AI.Ollama
}

func (p *OllamaProvider) GetModel(modelName string) (config.AIModel, error) {
	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil {
		return config.AIModel{}, fmt.Errorf("ollama config is nil")
	}
	for _, model := range ollamaConfig.Models {
		if model.Name == modelName {
			return model, nil
		}
	}
	return config.AIModel{}, fmt.Errorf("model not found: %s", modelName)
}

func (p *OllamaProvider) GetAllModels() []string {
	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil {
		return nil
	}
	if len(ollamaConfig.Models) > 0 {
		models := make([]string, len(ollamaConfig.Models))
		for i, m := range ollamaConfig.Models {
			models[i] = m.Name
		}
		return models
	}
	return []string{}
}

func (p *OllamaProvider) CallModel(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil || ollamaConfig.BaseURL == "" {
		return AIResponse{}, fmt.Errorf("ollama configuration or base URL is missing")
	}

	// 构建消息列表
	var messages []OllamaMessage

	// 添加系统消息
	if sysPrompt != "" {
		messages = append(messages, OllamaMessage{
			Role:    "system",
			Content: sysPrompt,
		})
	}

	// 添加历史消息
	for _, h := range history {
		messages = append(messages, OllamaMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	if !think {
		prompt = prompt + "\n\n \\no_think"
	}

	// 添加当前用户消息
	messages = append(messages, OllamaMessage{
		Role:    "user",
		Content: prompt,
	})

	req := OllamaChatRequest{
		Model:    modelName,
		Think:    think,
		Stream:   false,
		Messages: messages,
		Options:  map[string]any{"keep_alive": "0s"},
	}

	var aiResp AIResponse
	var fullContent strings.Builder

	// 发送HTTP请求
	resp, err := p.sendChatRequest(ctx, ollamaConfig.BaseURL, &req)
	if err != nil {
		return AIResponse{}, fmt.Errorf("ollama call model failed: %w", err)
	}

	fullContent.WriteString(resp.Message.Content)

	// 解析 think 和 content
	responseText := fullContent.String()
	thinkContent, answerContent := utils.ParseAIContent(responseText)
	aiResp.Content = answerContent
	aiResp.Think = thinkContent

	return aiResp, nil
}

func (p *OllamaProvider) CallModelStream(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error) {
	streamChan := make(chan AIResponse, 10)

	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil || ollamaConfig.BaseURL == "" {
		close(streamChan)
		return streamChan, fmt.Errorf("ollama configuration or base URL is missing")
	}

	// 构建消息列表
	var messages []OllamaMessage

	// 添加系统消息
	if sysPrompt != "" {
		messages = append(messages, OllamaMessage{
			Role:    "system",
			Content: sysPrompt,
		})
	}

	// 添加历史消息
	for _, h := range history {
		messages = append(messages, OllamaMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// 添加当前用户消息
	messages = append(messages, OllamaMessage{
		Role:    "user",
		Content: prompt,
	})

	req := OllamaChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   true,
		Think:    think,
		Options:  map[string]any{"keep_alive": "0s"},
	}

	go func() {
		defer close(streamChan)
		var fullContent strings.Builder

		err := p.sendChatStreamRequest(ctx, ollamaConfig.BaseURL, &req, func(resp *OllamaChatResponse) error {
			fullContent.WriteString(resp.Message.Content)
			select {
			case streamChan <- AIResponse{Content: fullContent.String()}:
			case <-ctx.Done():
				return fmt.Errorf("context cancelled")
			}
			return nil
		})
		if err != nil {
			p.logger.Error("Ollama stream error", "error", err)
		}
	}()

	return streamChan, nil
}

func (p *OllamaProvider) Embedding(ctx context.Context, modelName string, text string) ([][]float32, error) {
	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil {
		return nil, fmt.Errorf("Ollama configuration is not set")
	}
	if ollamaConfig.BaseURL == "" {
		return nil, fmt.Errorf("Ollama base URL is required")
	}

	// 参数校验
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("Ollama embedding: modelName is empty")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("Ollama embedding: text is empty")
	}

	req := OllamaEmbedRequest{
		Model: modelName,
		Input: text,
	}

	resp, err := p.sendEmbedRequest(ctx, ollamaConfig.BaseURL, &req)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding failed: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embedding returned no data")
	}
	return resp.Embeddings, nil
}

// HTTP请求方法
func (p *OllamaProvider) sendChatRequest(ctx context.Context, baseURL string, req *OllamaChatRequest) (*OllamaChatResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}

	// 如果是流式响应，需要逐行读取直到完成
	var finalResp *OllamaChatResponse
	var fullContent strings.Builder

	decoder := json.NewDecoder(httpResp.Body)
	for {
		var resp OllamaChatResponse
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode response: %w", err)
		}

		// 累积消息内容
		fullContent.WriteString(resp.Message.Content)

		// 保存最后一个响应作为基础
		finalResp = &resp

		// 如果标记为完成，退出循环
		if resp.Done {
			break
		}
	}

	// 如果没有收到任何响应
	if finalResp == nil {
		return nil, fmt.Errorf("no response received")
	}

	// 更新最终响应的消息内容为完整内容
	finalResp.Message.Content = fullContent.String()

	return finalResp, nil
}

func (p *OllamaProvider) sendChatStreamRequest(ctx context.Context, baseURL string, req *OllamaChatRequest, callback func(*OllamaChatResponse) error) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	p.logger.Debug("Sending chat request to Ollama",
		slog.String("baseURL", baseURL),
		slog.String("req", string(jsonData)),
	)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(httpResp.Body)
	for {
		var resp OllamaChatResponse
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode response: %w", err)
		}

		if err := callback(&resp); err != nil {
			return err
		}

		if resp.Done {
			break
		}
	}

	return nil
}

func (p *OllamaProvider) sendEmbedRequest(ctx context.Context, baseURL string, req *OllamaEmbedRequest) (*OllamaEmbedResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}

	var resp OllamaEmbedResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &resp, nil
}

func (p *OllamaProvider) sendRerankRequest(ctx context.Context, baseURL string, req *OllamaRerankRequest) (*OllamaRerankResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal rerank request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/rerank", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create rerank request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send rerank request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(body))
	}

	temp, _ := io.ReadAll(httpResp.Body)
	fmt.Println("Rerank Response:", string(temp))

	var resp OllamaRerankResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}

	return &resp, nil
}

// Rerank 对 candidates 进行重排序，返回排序后的下标和分数（分数越高越相关）
func (p *OllamaProvider) Rerank(ctx context.Context, modelName string, query string, candidates []string) ([]int, []float32, error) {
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("rerank: candidates is empty")
	}

	// 参数校验
	if strings.TrimSpace(modelName) == "" {
		return nil, nil, fmt.Errorf("Ollama rerank: modelName is empty")
	}

	ollamaConfig := p.getOllamaConfig()
	if ollamaConfig == nil || ollamaConfig.BaseURL == "" {
		return nil, nil, fmt.Errorf("ollama configuration or base URL is missing")
	}

	// 尝试使用专门的rerank API
	rerankReq := &OllamaRerankRequest{
		Model:     modelName,
		Query:     query,
		Documents: candidates,
		TopN:      len(candidates),
	}

	rerankResp, err := p.sendRerankRequest(ctx, ollamaConfig.BaseURL, rerankReq)
	if err != nil {
		// 如果rerank API不可用，回退到embedding方式
		p.logger.Warn("Rerank API not available, falling back to embedding similarity", "error", err)
		return nil, nil, fmt.Errorf("ollama rerank API failed: %w", err)
	}

	// 处理rerank API响应
	sortedIdx := make([]int, len(rerankResp.Results))
	sortedScores := make([]float32, len(rerankResp.Results))

	for i, result := range rerankResp.Results {
		sortedIdx[i] = result.Index
		sortedScores[i] = result.Score
	}

	return sortedIdx, sortedScores, nil
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(0)
	}
	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return float32(0)
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func NewOllamaProvider(ctx context.Context, configurator config.IServiceConfigurator, logger logging.ILogger) (AIProvider, error) {
	baseCfg := configurator.GetBaseConfig()
	if baseCfg.AI == nil {
		return nil, fmt.Errorf("ollama provider base AI config is nil")
	}
	ollamaConfig := baseCfg.AI.Ollama
	if ollamaConfig == nil {
		return nil, fmt.Errorf("ollama provider config is nil")
	}
	if ollamaConfig.BaseURL == "" {
		return nil, fmt.Errorf("ollama base_url is required")
	}

	return &OllamaProvider{
		ctx:          ctx,
		configurator: configurator,
		name:         "ollama",
		logger:       logger.With("provider", "ollama"),
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // 5分钟超时
		},
	}, nil
}
