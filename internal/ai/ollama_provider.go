package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gantt-saas/internal/infra/config"

	"go.uber.org/zap"
)

// OllamaProvider 实现 Ollama 本地模型的 Provider。
type OllamaProvider struct {
	client  *http.Client
	baseURL string
	model   string
	logger  *zap.Logger
}

// NewOllamaProvider 创建 Ollama Provider。
func NewOllamaProvider(cfg *config.AIProviderConfig, logger *zap.Logger) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   cfg.Model,
		logger:  logger.Named("ollama"),
	}, nil
}

func (p *OllamaProvider) Name() string { return "ollama" }

type ollamaChatReq struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]ollamaMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, ollamaMessage{Role: m.Role, Content: m.Content})
	}

	ollamaReq := ollamaChatReq{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	bodyBytes, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Ollama 可能返回多个 JSON 行（即使 stream=false），累积所有内容
	var fullContent strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var ollamaResp ollamaChatResp
		if err := decoder.Decode(&ollamaResp); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("ollama: decode response failed: %w", err)
		}
		fullContent.WriteString(ollamaResp.Message.Content)
		if ollamaResp.Done {
			break
		}
	}

	return &ChatResponse{
		Content:      fullContent.String(),
		FinishReason: "stop",
		Usage:        TokenUsage{}, // Ollama 不总是返回 usage
	}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]ollamaMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, ollamaMessage{Role: m.Role, Content: m.Content})
	}

	ollamaReq := ollamaChatReq{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	bodyBytes, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal stream request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			p.logger.Error("ollama stream request failed", zap.Error(err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			p.logger.Error("ollama stream HTTP error", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var ollamaResp ollamaChatResp
			if err := decoder.Decode(&ollamaResp); err != nil {
				if err == io.EOF {
					break
				}
				p.logger.Error("ollama stream decode error", zap.Error(err))
				break
			}

			if ollamaResp.Message.Content != "" {
				select {
				case ch <- StreamChunk{Content: ollamaResp.Message.Content}:
				case <-ctx.Done():
					return
				}
			}

			if ollamaResp.Done {
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
