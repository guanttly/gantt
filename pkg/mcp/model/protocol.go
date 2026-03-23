package model

// MCP (Model Context Protocol) 协议常量和规范定义

const (
	// 标准方法名
	MethodInitialize = "initialize"
	MethodToolsList  = "tools/list"
	MethodToolsCall  = "tools/call"

	// 通知方法名
	NotificationInitialized = "notifications/initialized"
	NotificationProgress    = "notifications/progress"
	NotificationMessage     = "notifications/message"

	// 资源相关方法
	MethodResourcesList        = "resources/list"
	MethodResourcesRead        = "resources/read"
	MethodResourcesSubscribe   = "resources/subscribe"
	MethodResourcesUnsubscribe = "resources/unsubscribe"

	// 提示相关方法
	MethodPromptsList = "prompts/list"
	MethodPromptsGet  = "prompts/get"

	// 采样相关方法
	MethodSamplingCreateMessage = "sampling/createMessage"
)

// 内容类型常量
const (
	ContentTypeText  = "text"
	ContentTypeImage = "image"
	ContentTypeData  = "data"
)

// 工具输入模式类型
const (
	SchemaTypeObject  = "object"
	SchemaTypeString  = "string"
	SchemaTypeNumber  = "number"
	SchemaTypeInteger = "integer"
	SchemaTypeBoolean = "boolean"
	SchemaTypeArray   = "array"
)

// 日志级别
const (
	LogLevelDebug     = "debug"
	LogLevelInfo      = "info"
	LogLevelNotice    = "notice"
	LogLevelWarning   = "warning"
	LogLevelError     = "error"
	LogLevelCritical  = "critical"
	LogLevelAlert     = "alert"
	LogLevelEmergency = "emergency"
)

// 标准工具输入模式构建器
type SchemaBuilder struct {
	schema map[string]any
}

func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: map[string]any{
			"type":       "object",
			"properties": make(map[string]any),
		},
	}
}

func (b *SchemaBuilder) AddStringProperty(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "string",
		"description": description,
	}

	if required {
		b.addRequired(name)
	}

	return b
}

func (b *SchemaBuilder) AddNumberProperty(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "number",
		"description": description,
	}

	if required {
		b.addRequired(name)
	}

	return b
}

func (b *SchemaBuilder) AddBooleanProperty(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "boolean",
		"description": description,
	}

	if required {
		b.addRequired(name)
	}

	return b
}

func (b *SchemaBuilder) AddArrayProperty(name, description string, itemType string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": itemType,
		},
	}

	if required {
		b.addRequired(name)
	}

	return b
}

func (b *SchemaBuilder) addRequired(name string) {
	if b.schema["required"] == nil {
		b.schema["required"] = []string{}
	}
	required := b.schema["required"].([]string)
	b.schema["required"] = append(required, name)
}

func (b *SchemaBuilder) Build() map[string]any {
	return b.schema
}

// 初始化相关类型
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ClientCapabilities struct {
	Experimental map[string]any      `json:"experimental,omitempty"`
	Sampling     *SamplingCapability `json:"sampling,omitempty"`
}

type ServerCapabilities struct {
	Experimental map[string]any       `json:"experimental,omitempty"`
	Logging      *LoggingCapability   `json:"logging,omitempty"`
	Prompts      *PromptsCapability   `json:"prompts,omitempty"`
	Resources    *ResourcesCapability `json:"resources,omitempty"`
	Tools        *ToolsCapability     `json:"tools,omitempty"`
}

type SamplingCapability struct{}
type LoggingCapability struct{}
type PromptsCapability struct{}
type ResourcesCapability struct{}
type ToolsCapability struct{}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// 工具列表相关类型
type ListToolsRequest struct {
	Cursor *string `json:"cursor,omitempty"`
}

type ListToolsResult struct {
	Tools      []Tool  `json:"tools"`
	NextCursor *string `json:"nextCursor,omitempty"`
}
