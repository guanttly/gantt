package ai

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const (
	AppScheduling          = "scheduling"
	WorkflowAIChat         = "ai.chat"
	WorkflowRuleParse      = "rule.parse"
	WorkflowScheduleCreate = "schedule.create"
	NodeIntentClassify     = "intent_classify"
	NodeChatReply          = "chat_reply"
	NodeRuleParse          = "rule_parse"
	NodeRuleBatchParse     = "rule_batch_parse"
	NodeAISelect           = "ai_select"
)

// NodeModelConfig describes the runtime model settings for one workflow node.
type NodeModelConfig struct {
	AppCode     string
	WorkflowKey string
	NodeKey     string
	Provider    string
	Model       string
	Timeout     time.Duration
	Temperature *float64
	MaxTokens   int
	Enabled     bool
}

// NodeModelResolver resolves model settings for app workflow nodes.
type NodeModelResolver interface {
	ResolveNodeModel(ctx context.Context, appCode, workflowKey, nodeKey string) (*NodeModelConfig, error)
}

// ProviderSelector selects a configured AI provider by name.
type ProviderSelector interface {
	Default() (Provider, error)
	Get(name string) (Provider, error)
}

func NormalizeTimeout(timeout time.Duration, fallback time.Duration) time.Duration {
	if timeout <= 0 {
		return fallback
	}
	if timeout > 0 && timeout < time.Millisecond {
		return timeout * time.Second
	}
	return timeout
}

func WithProviderTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	timeout = NormalizeTimeout(timeout, 0)
	if timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func ApplyNodeModelConfig(
	ctx context.Context,
	provider Provider,
	selector ProviderSelector,
	resolver NodeModelResolver,
	appCode string,
	workflowKey string,
	nodeKey string,
	req ChatRequest,
	fallbackTimeout time.Duration,
	logger *zap.Logger,
) (context.Context, context.CancelFunc, Provider, ChatRequest) {
	if resolver != nil {
		cfg, err := resolver.ResolveNodeModel(ctx, appCode, workflowKey, nodeKey)
		if err != nil {
			if logger != nil {
				logger.Warn("加载节点模型配置失败", zap.String("app", appCode), zap.String("workflow", workflowKey), zap.String("node", nodeKey), zap.Error(err))
			}
		} else if cfg != nil && cfg.Enabled {
			if cfg.Provider != "" && selector != nil {
				selected, err := selector.Get(cfg.Provider)
				if err != nil {
					if logger != nil {
						logger.Warn("节点指定的 AI provider 不可用", zap.String("provider", cfg.Provider), zap.Error(err))
					}
				} else {
					provider = selected
				}
			}
			if cfg.Model != "" {
				req.Model = cfg.Model
			}
			if cfg.Temperature != nil {
				req.Temperature = *cfg.Temperature
			}
			if cfg.MaxTokens > 0 {
				req.MaxTokens = cfg.MaxTokens
			}
			if cfg.Timeout > 0 {
				fallbackTimeout = cfg.Timeout
			}
		}
	}

	runtimeCtx, cancel := WithProviderTimeout(ctx, fallbackTimeout)
	return runtimeCtx, cancel, provider, req
}
