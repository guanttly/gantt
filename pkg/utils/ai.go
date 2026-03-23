package utils

import "strings"

// ParseAIContent 解析模型输出，区分思考和答案
// 支持格式: "<think>思考内容</think>答案内容" 或 "答案内容"
func ParseAIContent(fullResponse string) (think string, answer string) {
	thinkStartTag := "<think>"
	thinkEndTag := "</think>"

	// 查找 <think> 标签
	startIdx := strings.Index(fullResponse, thinkStartTag)
	if startIdx == -1 {
		// 没有找到 <think> 标签，整个内容作为答案
		return "", strings.TrimSpace(fullResponse)
	}

	// 查找 </think> 标签
	endIdx := strings.Index(fullResponse, thinkEndTag)
	if endIdx == -1 {
		// 找到开始标签但没有结束标签，从开始标签后的内容作为思考
		thinkContent := fullResponse[startIdx+len(thinkStartTag):]
		return strings.TrimSpace(thinkContent), ""
	}

	// 提取思考内容
	thinkContent := fullResponse[startIdx+len(thinkStartTag) : endIdx]
	think = strings.TrimSpace(thinkContent)

	// 提取答案内容（标签之前的内容 + 标签之后的内容）
	beforeThink := fullResponse[:startIdx]
	afterThink := fullResponse[endIdx+len(thinkEndTag):]
	answerContent := beforeThink + afterThink
	answer = strings.TrimSpace(answerContent)
	// 移除起始的换行符
	answer = strings.TrimPrefix(answer, "\n")

	return think, answer
}
