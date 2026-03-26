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

	"go.uber.org/zap"
)

// BailianProvider 实现阿里百炼（DashScope）的 Provider。
type BailianProvider struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
	logger  *zap.Logger
}

// NewBailianProvider 创建百炼 Provider。
func NewBailianProvider(cfg *config.AIProviderConfig, logger *zap.Logger) (*BailianProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("bailian: api_key is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com"
	}

	return &BailianProvider{
		client:  &http.Client{},
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   cfg.Model,
		logger:  logger.Named("bailian"),
	}, nil
}

func (p *BailianProvider) Name() string { return "bailian" }

type dashscopeRequest struct {
	Model string `json:"model"`
	Input struct {
		Messages []Message `json:"messages"`
	} `json:"input"`
	Parameters struct {
		ResultFormat string `json:"result_format"`
	} `json:"parameters"`
}

type dashscopeResponse struct {
	Output struct {
		Choices []struct {
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

func (p *BailianProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	dsReq := dashscopeRequest{Model: model}
	dsReq.Input.Messages = req.Messages
	dsReq.Parameters.ResultFormat = "message"

	bodyBytes, err := json.Marshal(dsReq)
	if err != nil {
		return nil, fmt.Errorf("bailian: marshal request failed: %w", err)
	}

	url := p.baseURL + "/api/v1/services/aigc/text-generation/generation"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bailian: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("bailian: read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bailian: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var dsResp dashscopeResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return nil, fmt.Errorf("bailian: unmarshal failed: %w", err)
	}

	if len(dsResp.Output.Choices) == 0 {
		return nil, fmt.Errorf("bailian: empty response")
	}

	return &ChatResponse{
		Content:      dsResp.Output.Choices[0].Message.Content,
		FinishReason: dsResp.Output.Choices[0].FinishReason,
		Usage: TokenUsage{
			PromptTokens:     dsResp.Usage.InputTokens,
			CompletionTokens: dsResp.Usage.OutputTokens,
			TotalTokens:      dsResp.Usage.TotalTokens,
		},
	}, nil
}

type dashscopeStreamResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
				Role    string `json:"role"`
			} `json:"message"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	} `json:"output"`
}

func (p *BailianProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	type streamReq struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
		Input  struct {
			Messages []Message `json:"messages"`
		} `json:"input"`
		Parameters struct {
			ResultFormat      string `json:"result_format"`
			IncrementalOutput bool   `json:"incremental_output"`
		} `json:"parameters"`
	}

	sr := streamReq{Model: model, Stream: true}
	sr.Input.Messages = req.Messages
	sr.Parameters.ResultFormat = "message"
	sr.Parameters.IncrementalOutput = true

	bodyBytes, err := json.Marshal(sr)
	if err != nil {
		return nil, fmt.Errorf("bailian: marshal stream request failed: %w", err)
	}

	url := p.baseURL + "/api/v1/services/aigc/text-generation/generation"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-DashScope-SSE", "enable")

	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			p.logger.Error("bailian stream request failed", zap.Error(err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			p.logger.Error("bailian stream HTTP error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
			return
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

			var streamResp dashscopeStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Output.Choices) == 0 {
				continue
			}

			choice := streamResp.Output.Choices[0]
			if choice.Message.Content != "" {
				select {
				case ch <- StreamChunk{Content: choice.Message.Content}:
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
