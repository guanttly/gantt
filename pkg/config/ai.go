package config

type AIConfig struct {
	ChatModel       AIModelProvider `mapstructure:"chat_model" yaml:"chat_model"`               // 聊天模型（默认模型）
	UpgradeModel    AIModelProvider `mapstructure:"upgrade_model" yaml:"upgrade_model"`         // 升级模型（重试时使用，未配置则回退到ChatModel）
	MaxModel        AIModelProvider `mapstructure:"max_model" yaml:"max_model"`                 // 最大模型（最终重试使用，未配置则回退到UpgradeModel或ChatModel）
	SplitQueryModel AIModelProvider `mapstructure:"split_query_model" yaml:"split_query_model"` // 分割查询模型
	EmbeddingModel  AIModelProvider `mapstructure:"embedding_model" yaml:"embedding_model"`     // 嵌入模型
	RerankerModel   AIModelProvider `mapstructure:"reranker_model" yaml:"reranker_model"`       // 重排模型
	HistoryNumber   int             `mapstructure:"history_number" yaml:"history_number"`       // 历史对话数量
	SysPrompt       string          `mapstructure:"sys_prompt" yaml:"sys_prompt"`               // 系统提示
	UserPrompt      string          `mapstructure:"user_prompt" yaml:"user_prompt"`             // 用户提示模板
	Ollama          *ProviderConfig `mapstructure:"ollama,omitempty" yaml:"ollama,omitempty"`
	OpenAI          *ProviderConfig `mapstructure:"openai,omitempty" yaml:"openai,omitempty"`
	Bailian         *ProviderConfig `mapstructure:"bailian,omitempty" yaml:"bailian,omitempty"`
	Local           *ProviderConfig `mapstructure:"local,omitempty" yaml:"local,omitempty"` // 本地模型配置
}

type AIModelProvider struct {
	Provider string `mapstructure:"provider" yaml:"provider"` // 模型提供者
	Name     string `mapstructure:"name" yaml:"name"`         // 模型名称
	Think    bool   `mapstructure:"think" yaml:"think"`       // 是否需要思考
}

type AIModel struct {
	Name        string  `mapstructure:"name" yaml:"name"` // 模型名称
	Type        string  `mapstructure:"type" yaml:"type"` // 模型类型（chat, embedding, reranker）
	Temperature float64 `mapstructure:"temperature" yaml:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens" yaml:"max_tokens"`
}

type ProviderConfig struct {
	BaseURL string    `mapstructure:"base_url" yaml:"base_url"` // 基础 URL
	APIKey  string    `mapstructure:"api_key" yaml:"api_key"`   // API 密钥
	Models  []AIModel `mapstructure:"models" yaml:"models"`     // 可用模型列表
}
