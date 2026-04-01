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

	"gantt-saas/internal/infra/config"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
)

// BailianProvider 实现阿里百炼（DashScope）的 Provider，使用 OpenAI 兼容模式。
type BailianProvider struct {
	client     *openai.Client
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
	logger     *zap.Logger
}

const defaultBailianBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

// NewBailianProvider 创建百炼 Provider。
func NewBailianProvider(cfg *config.AIProviderConfig, logger *zap.Logger) (*BailianProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("bailian: api_key is required")
	}

	baseURL := normalizeBailianURL(cfg.BaseURL)

	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(baseURL),
	)

	return &BailianProvider{
		client:     &client,
		httpClient: &http.Client{},
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		model:      cfg.Model,
		logger:     logger.Named("bailian"),
	}, nil
}

// normalizeBailianURL 将各种形式的 DashScope URL 统一为 OpenAI 兼容端点。
func normalizeBailianURL(raw string) string {
	if raw == "" {
		return defaultBailianBaseURL
	}
	u := strings.TrimRight(raw, "/")
	// 去掉旧版 native API 路径
	u = strings.TrimSuffix(u, "/api/v1/services/aigc/text-generation/generation")
	u = strings.TrimSuffix(u, "/api/v1")
	u = strings.TrimSuffix(u, "/v1")
	u = strings.TrimRight(u, "/")
	// 如果已经是 compatible-mode 路径则直接返回
	if strings.HasSuffix(u, "/compatible-mode/v1") {
		return u
	}
	u = strings.TrimSuffix(u, "/compatible-mode")
	return u + "/compatible-mode/v1"
}

func (p *BailianProvider) Name() string { return "bailian" }

func (p *BailianProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(m.Content))
		case "user":
			messages = append(messages, openai.UserMessage(m.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    model,
	}

	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("bailian chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("bailian: empty response")
	}

	return &ChatResponse{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage: TokenUsage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}, nil
}

func (p *BailianProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type streamReqBody struct {
		Model         string    `json:"model"`
		Messages      []chatMsg `json:"messages"`
		Stream        bool      `json:"stream"`
		Temperature   float64   `json:"temperature,omitempty"`
		StreamOptions *struct {
			IncludeUsage bool `json:"include_usage"`
		} `json:"stream_options,omitempty"`
	}

	msgs := make([]chatMsg, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, chatMsg{Role: m.Role, Content: m.Content})
	}

	body := streamReqBody{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	}
	if req.Temperature > 0 {
		body.Temperature = req.Temperature
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("bailian: marshal stream request failed: %w", err)
	}

	url := strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			p.logger.Error("bailian stream request failed", zap.Error(err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			p.logger.Error("bailian stream HTTP error",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(respBody)))
			return
		}

		// DashScope SSE 格式: data: {...}\n\n
		type deltaObj struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		}
		type choiceObj struct {
			Delta        deltaObj `json:"delta"`
			FinishReason *string  `json:"finish_reason"`
		}
		type ssePayload struct {
			Choices []choiceObj `json:"choices"`
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				break
			}

			var payload ssePayload
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				continue
			}

			if len(payload.Choices) == 0 {
				continue
			}

			choice := payload.Choices[0]

			// 发送 reasoning_content（Qwen 思考过程）
			if choice.Delta.ReasoningContent != "" {
				select {
				case ch <- StreamChunk{Reasoning: choice.Delta.ReasoningContent}:
				case <-ctx.Done():
					return
				}
			}

			// 发送 content（最终输出）
			if choice.Delta.Content != "" {
				select {
				case ch <- StreamChunk{Content: choice.Delta.Content}:
				case <-ctx.Done():
					return
				}
			}

			if choice.FinishReason != nil && *choice.FinishReason == "stop" {
				break
			}
		}

		select {
		case ch <- StreamChunk{Done: true}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}
