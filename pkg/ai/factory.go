package ai

import (
	"context"
	"fmt"

	// 从 jusha/mcp/pkg/logging 更改
	"jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"
)

type AIProviderFactory struct {
	ctx          context.Context
	configurator config.IServiceConfigurator // 已更改：IServiceConfigurator 而不是 *IServiceConfigurator
	logger       logging.ILogger             // 已添加
	providers    map[string]AIProvider
}

// NewAIModelFactory 现在接受 IServiceConfigurator 和logging.ILogger。
// AIConfig 通过 configurator 动态获取。
func NewAIModelFactory(ctx context.Context, configurator config.IServiceConfigurator, logger logging.ILogger) *AIProviderFactory { // 参数已更改
	return &AIProviderFactory{
		ctx:          ctx,
		configurator: configurator,
		logger:       logger, // 存储的 logger
		providers:    make(map[string]AIProvider),
	}
}

func (f *AIProviderFactory) GetDefaultProvider() (AIProvider, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil, fmt.Errorf("AI configuration is not available")
	}
	providerName := aiCfg.ChatModel.Provider
	if providerName == "" {
		return nil, fmt.Errorf("default AI provider is not configured")
	}
	return f.GetProvider(providerName)
}

func (f *AIProviderFactory) GetDefaultProviderNameSafe() string {
	if provider, err := f.GetDefaultProvider(); err == nil {
		return provider.GetName()
	}
	return "Unknown"
}

func (f *AIProviderFactory) GetDefaultModel() (config.AIModel, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return config.AIModel{}, fmt.Errorf("AI configuration is not available")
	}
	modelName := aiCfg.ChatModel.Name
	if modelName == "" {
		return config.AIModel{}, fmt.Errorf("default AI model is not configured")
	}
	provider, err := f.GetDefaultProvider()
	if err != nil {
		return config.AIModel{}, err
	}
	return provider.GetModel(modelName)
}

func (f *AIProviderFactory) CallDefault(ctx context.Context, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return AIResponse{}, fmt.Errorf("AI configuration is not available")
	}
	providerName := aiCfg.ChatModel.Provider
	if providerName == "" {
		return AIResponse{}, fmt.Errorf("default AI provider is not configured")
	}
	provider, err := f.GetProvider(providerName)
	if err != nil {
		return AIResponse{}, err
	}
	modelName := aiCfg.ChatModel.Name
	if modelName == "" {
		return AIResponse{}, fmt.Errorf("default AI chat model is not configured")
	}
	if !provider.SupportsModel(modelName) {
		return AIResponse{}, fmt.Errorf("provider '%s' does not support model '%s'", providerName, modelName)
	}

	return provider.CallModel(ctx, modelName, aiCfg.ChatModel.Think, sysPrompt, prompt, history)
}

// CallUpgrade 使用升级模型调用（第一次重试时使用）
// 如果升级模型未配置，则回退到默认模型
func (f *AIProviderFactory) CallUpgrade(ctx context.Context, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return AIResponse{}, fmt.Errorf("AI configuration is not available")
	}

	// 检查升级模型是否配置，未配置则回退到默认模型
	if aiCfg.UpgradeModel.Provider == "" || aiCfg.UpgradeModel.Name == "" {
		return f.CallDefault(ctx, sysPrompt, prompt, history)
	}

	provider, err := f.GetProvider(aiCfg.UpgradeModel.Provider)
	if err != nil {
		// 获取provider失败，回退到默认模型
		return f.CallDefault(ctx, sysPrompt, prompt, history)
	}

	if !provider.SupportsModel(aiCfg.UpgradeModel.Name) {
		// 模型不支持，回退到默认模型
		return f.CallDefault(ctx, sysPrompt, prompt, history)
	}

	return provider.CallModel(ctx, aiCfg.UpgradeModel.Name, aiCfg.UpgradeModel.Think, sysPrompt, prompt, history)
}

// CallMax 使用最大模型调用（最终重试时使用）
// 如果最大模型未配置，则回退到升级模型；如果升级模型也未配置，则回退到默认模型
func (f *AIProviderFactory) CallMax(ctx context.Context, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return AIResponse{}, fmt.Errorf("AI configuration is not available")
	}

	// 检查最大模型是否配置，未配置则回退到升级模型
	if aiCfg.MaxModel.Provider == "" || aiCfg.MaxModel.Name == "" {
		return f.CallUpgrade(ctx, sysPrompt, prompt, history)
	}

	provider, err := f.GetProvider(aiCfg.MaxModel.Provider)
	if err != nil {
		// 获取provider失败，回退到升级模型
		return f.CallUpgrade(ctx, sysPrompt, prompt, history)
	}

	if !provider.SupportsModel(aiCfg.MaxModel.Name) {
		// 模型不支持，回退到升级模型
		return f.CallUpgrade(ctx, sysPrompt, prompt, history)
	}

	return provider.CallModel(ctx, aiCfg.MaxModel.Name, aiCfg.MaxModel.Think, sysPrompt, prompt, history)
}

// CallWithRetryLevel 根据重试层级选择模型调用
// level: 0=默认模型, 1=升级模型, 2=最大模型
// 每个层级如果对应模型未配置，会自动回退到下一个可用的模型
func (f *AIProviderFactory) CallWithRetryLevel(ctx context.Context, level int, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	switch level {
	case 0:
		return f.CallDefault(ctx, sysPrompt, prompt, history)
	case 1:
		return f.CallUpgrade(ctx, sysPrompt, prompt, history)
	case 2:
		return f.CallMax(ctx, sysPrompt, prompt, history)
	default:
		// 超过2级时使用最大模型
		return f.CallMax(ctx, sysPrompt, prompt, history)
	}
}

// GetUpgradeModelConfig 获取升级模型配置（如果未配置则返回默认模型配置）
func (f *AIProviderFactory) GetUpgradeModelConfig() *config.AIModelProvider {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil
	}
	if aiCfg.UpgradeModel.Provider != "" && aiCfg.UpgradeModel.Name != "" {
		return &aiCfg.UpgradeModel
	}
	return &aiCfg.ChatModel
}

// GetMaxModelConfig 获取最大模型配置（如果未配置则返回升级模型配置，再未配置则返回默认模型配置）
func (f *AIProviderFactory) GetMaxModelConfig() *config.AIModelProvider {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil
	}
	if aiCfg.MaxModel.Provider != "" && aiCfg.MaxModel.Name != "" {
		return &aiCfg.MaxModel
	}
	return f.GetUpgradeModelConfig()
}

// CallWithModel 使用指定的模型配置调用AI
// 如果 modelConfig 为 nil 或配置不完整，则回退到 CallDefault
func (f *AIProviderFactory) CallWithModel(ctx context.Context, modelConfig *config.AIModelProvider, sysPrompt, prompt string, history []AIMessage) (AIResponse, error) {
	// 如果没有指定模型配置，使用默认配置
	if modelConfig == nil || modelConfig.Provider == "" || modelConfig.Name == "" {
		return f.CallDefault(ctx, sysPrompt, prompt, history)
	}

	// 使用指定的模型配置
	provider, err := f.GetProvider(modelConfig.Provider)
	if err != nil {
		return AIResponse{}, fmt.Errorf("get provider '%s' failed: %w", modelConfig.Provider, err)
	}

	if !provider.SupportsModel(modelConfig.Name) {
		return AIResponse{}, fmt.Errorf("provider '%s' does not support model '%s'", modelConfig.Provider, modelConfig.Name)
	}

	return provider.CallModel(ctx, modelConfig.Name, modelConfig.Think, sysPrompt, prompt, history)
}

// CallDefaultStream 使用默认模型配置进行流式调用
func (f *AIProviderFactory) CallDefaultStream(ctx context.Context, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil, fmt.Errorf("AI configuration is not available")
	}
	providerName := aiCfg.ChatModel.Provider
	if providerName == "" {
		return nil, fmt.Errorf("default AI provider is not configured")
	}
	provider, err := f.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	modelName := aiCfg.ChatModel.Name
	if modelName == "" {
		return nil, fmt.Errorf("default AI chat model is not configured")
	}
	if !provider.SupportsModel(modelName) {
		return nil, fmt.Errorf("provider '%s' does not support model '%s'", providerName, modelName)
	}

	return provider.CallModelStream(ctx, modelName, aiCfg.ChatModel.Think, sysPrompt, prompt, history)
}

// CallWithModelStream 使用指定的模型配置进行流式调用
// 如果 modelConfig 为 nil 或配置不完整，则回退到 CallDefaultStream
func (f *AIProviderFactory) CallWithModelStream(ctx context.Context, modelConfig *config.AIModelProvider, sysPrompt, prompt string, history []AIMessage) (chan AIResponse, error) {
	// 如果没有指定模型配置，使用默认配置
	if modelConfig == nil || modelConfig.Provider == "" || modelConfig.Name == "" {
		return f.CallDefaultStream(ctx, sysPrompt, prompt, history)
	}

	// 使用指定的模型配置
	provider, err := f.GetProvider(modelConfig.Provider)
	if err != nil {
		return nil, fmt.Errorf("get provider '%s' failed: %w", modelConfig.Provider, err)
	}

	if !provider.SupportsModel(modelConfig.Name) {
		return nil, fmt.Errorf("provider '%s' does not support model '%s'", modelConfig.Provider, modelConfig.Name)
	}

	return provider.CallModelStream(ctx, modelConfig.Name, modelConfig.Think, sysPrompt, prompt, history)
}

func (f *AIProviderFactory) EmbeddingDefault(ctx context.Context, prompt string) ([][]float32, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil, fmt.Errorf("AI configuration is not available")
	}
	providerName := aiCfg.EmbeddingModel.Provider
	if providerName == "" {
		return nil, fmt.Errorf("default AI embedding provider is not configured")
	}
	provider, err := f.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	modelName := aiCfg.EmbeddingModel.Name
	if modelName == "" {
		return nil, fmt.Errorf("default AI embedding model is not configured")
	}
	if !provider.SupportsModel(modelName) {
		return nil, fmt.Errorf("provider '%s' does not support model '%s'", providerName, modelName)
	}

	return provider.Embedding(ctx, modelName, prompt)
}

func (f *AIProviderFactory) RerankDefault(ctx context.Context, query string, candidates []string) ([]int, []float32, error) {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil, nil, fmt.Errorf("AI configuration is not available")
	}
	providerName := aiCfg.RerankerModel.Provider
	if providerName == "" {
		return nil, nil, fmt.Errorf("default AI rerank provider is not configured")
	}
	provider, err := f.GetProvider(providerName)
	if err != nil {
		return nil, nil, err
	}
	modelName := aiCfg.RerankerModel.Name
	if modelName == "" {
		return nil, nil, fmt.Errorf("default AI rerank model is not configured")
	}
	if !provider.SupportsModel(modelName) {
		return nil, nil, fmt.Errorf("provider '%s' does not support model '%s'", providerName, modelName)
	}

	return provider.Rerank(ctx, modelName, query, candidates)
}

func (f *AIProviderFactory) GetProvider(providerName string) (AIProvider, error) {
	if aiProvider, ok := f.providers[providerName]; ok {
		return aiProvider, nil
	}

	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return nil, fmt.Errorf("AI configuration is not available")
	}

	var newProvider AIProvider
	var err error

	// 使用注入的 logger f.logger 而不是 logging.GetLogger()
	// 提供者构造函数现在期望 configurator 和 logger

	switch providerName {
	case "ollama":
		if aiCfg.Ollama == nil {
			return nil, fmt.Errorf("提供者 '%s' 缺少 ollama 配置部分", providerName)
		}
		// 传递主 configurator 和工厂的 logger
		newProvider, err = NewOllamaProvider(f.ctx, f.configurator, f.logger)
	case "openai":
		if aiCfg.OpenAI == nil {
			return nil, fmt.Errorf("提供者 '%s' 缺少 openai 配置部分", providerName)
		}
		// 传递主 configurator 和工厂的 logger
		newProvider, err = NewOpenAIProvider(f.ctx, f.configurator, f.logger)
	case "bailian":
		if aiCfg.Bailian == nil {
			return nil, fmt.Errorf("提供者 '%s' 缺少 bailian 配置部分", providerName)
		}
		// 传递主 configurator 和工厂的 logger
		newProvider, err = NewBailianProvider(f.ctx, f.configurator, f.logger)
	case "local":
		if aiCfg.Local == nil {
			return nil, fmt.Errorf("提供者 '%s' 缺少 local 配置部分", providerName)
		}
		// 传递主 configurator 和工厂的 logger
		newProvider, err = NewLocalProvider(f.ctx, f.configurator, f.logger)
	default:
		return nil, fmt.Errorf("不支持的 AI 服务提供者: %s", providerName)
	}

	if err != nil {
		return nil, fmt.Errorf("创建 AI 提供者 '%s' 失败: %w", providerName, err)
	}

	f.providers[providerName] = newProvider
	return newProvider, nil
}

func (f *AIProviderFactory) GetAllProviders() []string {
	aiCfg := f.configurator.GetBaseConfig().AI
	if aiCfg == nil {
		return []string{} // 没有 AI 配置，因此没有提供者
	}

	providerNames := make([]string, 0, 3) // 最多 3 个已知提供者
	if aiCfg.Ollama != nil {
		providerNames = append(providerNames, "ollama")
	}
	if aiCfg.OpenAI != nil {
		providerNames = append(providerNames, "openai")
	}
	if aiCfg.Bailian != nil {
		providerNames = append(providerNames, "bailian")
	}
	return providerNames
}

// GetAIModelByType 可能已弃用或需要重新评估。
// "Type" 可能意味着提供者名称。如果是这样，它类似于 GetProvider。
// 如果 "Type" 意味着其他东西（例如 "chat", "embedding"），则需要更复杂的逻辑。
// 目前，假设 "modelType" 等同于 "providerName"。
func (f *AIProviderFactory) GetAIModelByType(modelTypeAsProviderName string) (AIProvider, error) {
	return f.GetProvider(modelTypeAsProviderName)
}
