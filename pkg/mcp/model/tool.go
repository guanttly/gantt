package model

// 工具相关类型
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type CallToolRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// 辅助函数
func NewTextContent(text string) Content {
	return Content{
		Type: "text",
		Text: text,
	}
}

func NewDataContent(data string) Content {
	return Content{
		Type: "data",
		Data: data,
	}
}
