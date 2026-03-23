package ai

import (
	"context"

	"jusha/mcp/pkg/config"
)

// 定义AI模型调用接口

type AIResponse struct {
	Think   string `json:"think"`
	Content string `json:"content"`
}

type AIProvider interface {
	// 对话/生成
	CallModel(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (AIResponse, error)
	CallModelStream(ctx context.Context, modelName string, think bool, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error)

	// 文本向量化
	Embedding(ctx context.Context, modelName string, prompt string) ([][]float32, error)

	// 召回结果重排
	Rerank(ctx context.Context, modelName string, query string, candidates []string) ([]int, []float32, error)

	GetName() string

	GetModel(modelName string) (config.AIModel, error)
	GetAllModels() []string

	SupportsModel(modelName string) bool
}
