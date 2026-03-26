package ai

import (
	"fmt"

	"gantt-saas/internal/infra/config"

	"go.uber.org/zap"
)

// Factory 管理所有 AI Provider 实例。
type Factory struct {
	providers       map[string]Provider
	defaultProvider string
	logger          *zap.Logger
}

// NewFactory 根据配置创建 Provider 工厂。
func NewFactory(cfg *config.AIConfig, logger *zap.Logger) *Factory {
	f := &Factory{
		providers:       make(map[string]Provider),
		defaultProvider: cfg.DefaultProvider,
		logger:          logger.Named("ai"),
	}

	if cfg.OpenAI != nil && cfg.OpenAI.Enabled {
		p, err := NewOpenAIProvider(cfg.OpenAI, logger)
		if err != nil {
			logger.Warn("初始化 OpenAI Provider 失败", zap.Error(err))
		} else {
			f.providers["openai"] = p
		}
	}

	if cfg.Bailian != nil && cfg.Bailian.Enabled {
		p, err := NewBailianProvider(cfg.Bailian, logger)
		if err != nil {
			logger.Warn("初始化百炼 Provider 失败", zap.Error(err))
		} else {
			f.providers["bailian"] = p
		}
	}

	if cfg.Ollama != nil && cfg.Ollama.Enabled {
		p, err := NewOllamaProvider(cfg.Ollama, logger)
		if err != nil {
			logger.Warn("初始化 Ollama Provider 失败", zap.Error(err))
		} else {
			f.providers["ollama"] = p
		}
	}

	logger.Info("AI Provider 工厂初始化完成",
		zap.Int("providers", len(f.providers)),
		zap.String("default", f.defaultProvider),
	)

	return f
}

// Get 获取指定名称的 Provider。
func (f *Factory) Get(name string) (Provider, error) {
	p, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown AI provider: %s", name)
	}
	return p, nil
}

// Default 返回默认 Provider。
func (f *Factory) Default() (Provider, error) {
	if f.defaultProvider != "" {
		if p, ok := f.providers[f.defaultProvider]; ok {
			return p, nil
		}
	}
	// 按优先级返回第一个可用的 Provider
	for _, name := range []string{"openai", "bailian", "ollama"} {
		if p, ok := f.providers[name]; ok {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no AI provider available")
}

// HasProvider 检查是否有可用的 Provider。
func (f *Factory) HasProvider() bool {
	return len(f.providers) > 0
}

// ListProviders 返回所有可用 Provider 名称。
func (f *Factory) ListProviders() []string {
	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}
