package tool

type ToolName string

func (t ToolName) String() string {
	return string(t)
}

// Conversation 会话管理工具
const (
	ToolConversationNew                ToolName = "conversation.new"
	ToolConversationAppend             ToolName = "conversation.append"
	ToolConversationHistory            ToolName = "conversation.history"
	ToolConversationList               ToolName = "conversation.list"
	ToolConversationUpdateWorkflowContext ToolName = "conversation.update_workflow_context"
	ToolConversationGetWorkflowContext ToolName = "conversation.get_workflow_context"
	ToolConversationUpdateMeta         ToolName = "conversation.update_meta"
)
