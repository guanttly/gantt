package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/utils"

	"github.com/google/uuid"
)

// ChatSession 表示一次对话会话
type ChatSession struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	logger     logging.ILogger

	ID            string
	Provider      AIProvider
	ModelName     string
	History       []AIMessage
	HistoryNumber int
	HistoryMux    sync.Mutex
	CreatedAt     time.Time
	UpdatedAt     time.Time

	sysPrompt  string // 系统提示词
	userPrompt string // 用户提示词
}

// NewChat 创建一个新的 ChatSession
func NewChat(
	ctx context.Context,
	logger logging.ILogger,
	provider AIProvider,
	modelName string,
	sysPrompt string,
	historyNumber int) *ChatSession {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	return &ChatSession{
		ctx:        ctxWithCancel,
		cancelFunc: cancel,
		logger:     logger,

		ID:            uuid.NewString(),
		Provider:      provider,
		ModelName:     modelName,
		History:       make([]AIMessage, 0),
		HistoryNumber: historyNumber,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),

		sysPrompt:  sysPrompt,
		userPrompt: "",
	}
}

// AddUserMessage 添加用户消息到历史
func (c *ChatSession) AddUserMessage(content string) {
	c.HistoryMux.Lock()
	defer c.HistoryMux.Unlock()
	c.History = append(c.History, CreateUserMessage(content))
	if c.HistoryNumber > 0 && len(c.History) > c.HistoryNumber {
		c.History = c.History[len(c.History)-c.HistoryNumber:]
	}
}

// AddAssistantMessage 添加助手消息到历史
func (c *ChatSession) AddAssistantMessage(content string) {
	// content 可能包含Think，需要移除
	if content == "" {
		c.logger.Warn("尝试添加空助手消息到历史", slog.String("sessionID", c.ID))
		return
	}
	_, content = utils.ParseAIContent(content)
	if len(content) > 0 && content[len(content)-1] == '\n' {
		content = content[:len(content)-1] // 移除末尾的换行符
	}

	c.HistoryMux.Lock()
	defer c.HistoryMux.Unlock()
	c.History = append(c.History, CreateAssistantMessage(content))
	if c.HistoryNumber > 0 && len(c.History) > c.HistoryNumber {
		c.History = c.History[len(c.History)-c.HistoryNumber:]
	}
}

// GetHistory 获取当前历史
func (c *ChatSession) GetHistory() []AIMessage {
	c.HistoryMux.Lock()
	defer c.HistoryMux.Unlock()
	h := make([]AIMessage, len(c.History))
	copy(h, c.History)
	c.logger.Debug("获取历史记录",
		slog.Int("history_length", len(h)),
		slog.String("sessionID", c.ID),
	)
	return h
}

// ChatOnce 用户发起一次对话，自动记录历史
func (c *ChatSession) ChatOnce(ctx context.Context, think bool, prompt string) (AIResponse, error) {
	defer func() {
		c.UpdatedAt = time.Now()
	}()
	if c.userPrompt != "" {
		prompt = fmt.Sprintf(c.userPrompt, prompt)
	}
	resp, err := c.Provider.CallModel(ctx, c.ModelName, think, c.sysPrompt, prompt, []AIMessage{})
	if err != nil {
		return AIResponse{}, err
	}
	return resp, nil
}

// ChatStream 用户发起一次流式对话，自动记录历史
func (c *ChatSession) ChatStream(ctx context.Context, think bool, message string, prompt string) (chan AIResponse, error) {
	defer func() {
		c.UpdatedAt = time.Now()
	}()
	if c.userPrompt != "" {
		prompt = fmt.Sprintf(c.userPrompt, prompt)
	}
	ch, err := c.Provider.CallModelStream(ctx, c.ModelName, think, c.sysPrompt, prompt, c.GetHistory())
	if err != nil {
		return nil, err
	}

	c.AddUserMessage(message)
	// 自动记录最后一条助手消息
	outCh := make(chan AIResponse, 1)
	go func() {
		defer close(outCh)
		var last AIResponse
		for resp := range ch {
			last = resp
			outCh <- resp
		}
		if last.Content != "" {
			c.AddAssistantMessage(last.Content)
		}
	}()
	return outCh, nil
}

// ResetHistory 清空历史
func (c *ChatSession) ResetHistory() {
	c.HistoryMux.Lock()
	defer c.HistoryMux.Unlock()
	c.History = make([]AIMessage, 0)
}

// SwitchModel 切换当前会话的模型和Provider，并校验Provider是否支持该模型
func (c *ChatSession) SwitchModel(provider AIProvider, modelName string) error {
	// 假设AIProvider有一个SupportsModel方法用于校验
	if !provider.SupportsModel(modelName) {
		return fmt.Errorf("provider does not support model: %s", modelName)
	}
	c.Provider = provider
	c.ModelName = modelName
	return nil
}

func (c *ChatSession) GetModelName() string {
	return c.ModelName
}

func (c *ChatSession) GetProviderName() string {
	if c.Provider == nil {
		return ""
	}
	return c.Provider.GetName()
}

func (c *ChatSession) CleanUp() {
	c.logger.Info("ChatSession cleaned up", slog.String("sessionID", c.ID))
	// 取消上下文
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	// 释放其他资源
	c.ResetHistory()
	c.Provider = nil
	c.userPrompt = ""

	c.logger.Info("ChatSession cleanup completed", slog.String("sessionID", c.ID))
}
