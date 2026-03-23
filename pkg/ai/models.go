package ai

type ChatRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func CreateUserMessage(content string) AIMessage {
	return AIMessage{
		Role:    "user",
		Content: content,
	}
}

func CreateSystemMessage(content string) AIMessage {
	return AIMessage{
		Role:    "system",
		Content: content,
	}
}

func CreateAssistantMessage(content string) AIMessage {
	return AIMessage{
		Role:    "assistant",
		Content: content,
	}
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	AIMessage ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	ReasoningContent string `json:"reasoning_content"`
	Content          string `json:"content"`
}
