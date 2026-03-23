// Package session 提供通用的会话管理模型和接口
// 适用于所有基于对话的 Agent
package session

import (
	"time"
)

// WorkflowState 工作流状态（FSM 状态）
// 从 engine.State 导出，避免循环依赖
type WorkflowState string

// WorkflowEvent 工作流事件
// 从 engine.Event 导出，避免循环依赖
type WorkflowEvent string

// Session 通用会话模型
// 所有 Agent 共享相同的会话结构，业务特定数据通过 Data 字段扩展
type Session struct {
	// 基础标识
	ID        string `json:"id"`
	OrgID     string `json:"orgId"`
	UserID    string `json:"userId"`
	AgentType string `json:"agentType"` // "rostering", "department", "rule", etc.

	// 会话状态
	State     SessionState `json:"state"`     // 当前状态
	StateDesc string       `json:"stateDesc"` // 状态描述（用户可读）

	// 错误信息
	Error string `json:"error,omitempty"`

	// 消息历史
	Messages []Message `json:"messages"`

	// 工作流元数据
	WorkflowMeta *WorkflowMeta `json:"workflowMeta,omitempty"`

	// 业务数据（可扩展字段）
	// 不同 Agent 可以在这里存储业务特定的数据
	// 例如 rostering: {"intent": {...}, "scheduleResult": {...}}
	// 例如 department: {"deptId": "xxx", "action": "create"}
	Data map[string]any `json:"data"`

	// 元数据（系统级扩展）
	Metadata map[string]any `json:"metadata,omitempty"`

	// 版本控制（用于 CAS 操作）
	Version int64 `json:"version"`

	// 时间戳
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpireAt  time.Time `json:"expireAt,omitempty"`
}

// SessionState 会话状态枚举
type SessionState string

const (
	StateIdle       SessionState = "idle"       // 空闲
	StateProcessing SessionState = "processing" // 处理中
	StateWaiting    SessionState = "waiting"    // 等待用户输入
	StateCompleted  SessionState = "completed"  // 已完成
	StateFailed     SessionState = "failed"     // 失败
)

// Message 会话消息
type Message struct {
	ID        string           `json:"id"`
	Role      MessageRole      `json:"role"`
	Content   string           `json:"content"`
	Timestamp time.Time        `json:"timestamp"`
	Actions   []WorkflowAction `json:"actions,omitempty"`  // 随消息持久化的操作按钮
	Metadata  map[string]any   `json:"metadata,omitempty"` // 扩展字段（如 token 消耗等）
}

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"      // 用户消息
	RoleAssistant MessageRole = "assistant" // 助手消息
	RoleSystem    MessageRole = "system"    // 系统消息
)

// WorkflowMeta 工作流元数据
type WorkflowMeta struct {
	Workflow            string           `json:"workflow"`                      // 工作流名称
	Version             string           `json:"version,omitempty"`             // 工作流版本（如 "v2", "v3"）
	InstanceID          string           `json:"instanceId"`                    // 实例 ID
	Description         string           `json:"description"`                   // 状态描述（用户可读）
	Phase               WorkflowState    `json:"phase"`                         // 当前阶段（FSM 状态）
	Actions             []WorkflowAction `json:"actions,omitempty"`             // 可用操作
	ActionsTransitionID string           `json:"actionsTransitionId,omitempty"` // Actions 所属的状态转换 ID
	Extra               map[string]any   `json:"extra,omitempty"`               // 额外元数据
}

// WorkflowActionType 工作流操作类型
type WorkflowActionType string

const (
	ActionTypeWorkflow WorkflowActionType = "workflow" // 工作流事件触发
	ActionTypeQuery    WorkflowActionType = "query"    // 查询操作
	ActionTypeCommand  WorkflowActionType = "command"  // 命令操作
	ActionTypeNavigate WorkflowActionType = "navigate" // 导航操作
)

// WorkflowActionStyle 工作流操作样式
type WorkflowActionStyle string

const (
	ActionStylePrimary   WorkflowActionStyle = "primary"   // 主要操作（蓝色）
	ActionStyleSecondary WorkflowActionStyle = "secondary" // 次要操作（灰色）
	ActionStyleSuccess   WorkflowActionStyle = "success"   // 成功操作（绿色）
	ActionStyleDanger    WorkflowActionStyle = "danger"    // 危险操作（红色）
	ActionStyleWarning   WorkflowActionStyle = "warning"   // 警告操作（黄色）
	ActionStyleInfo      WorkflowActionStyle = "info"      // 信息操作（浅蓝）
	ActionStyleLink      WorkflowActionStyle = "link"      // 链接样式
)

// WorkflowActionFieldType 字段类型
type WorkflowActionFieldType string

const (
	FieldTypeText         WorkflowActionFieldType = "text"          // 文本输入
	FieldTypeNumber       WorkflowActionFieldType = "number"        // 数字输入
	FieldTypeDate         WorkflowActionFieldType = "date"          // 日期选择
	FieldTypeDatetime     WorkflowActionFieldType = "datetime"      // 日期时间选择
	FieldTypeSelect       WorkflowActionFieldType = "select"        // 下拉选择（单选）
	FieldTypeCheckbox     WorkflowActionFieldType = "checkbox"      // 复选框
	FieldTypeTextarea     WorkflowActionFieldType = "textarea"      // 多行文本
	FieldTypeMultiSelect  WorkflowActionFieldType = "multi-select"  // 多选列表（带搜索、全选、反选）
	FieldTypeCheckedTable WorkflowActionFieldType = "checked-table" // 可选表格（带复选框的表格视图）
	FieldTypeDailyGrid    WorkflowActionFieldType = "daily-grid"    // 每日网格配置（按周分页、支持折叠、摘要）
	FieldTypeRepeat       WorkflowActionFieldType = "repeat"        // 可重复字段组（用于动态添加多个表单项）
	FieldTypeTableForm    WorkflowActionFieldType = "table-form"    // 表格+表单组合（上半部分表格，下半部分表单）
)

// WorkflowActionField 操作字段定义（用于动态表单生成）
type WorkflowActionField struct {
	Name         string                  `json:"name"`                   // 字段名称（用于 payload key）
	Label        string                  `json:"label"`                  // 显示标签
	Type         WorkflowActionFieldType `json:"type"`                   // 字段类型
	Required     bool                    `json:"required,omitempty"`     // 是否必填
	Placeholder  string                  `json:"placeholder,omitempty"`  // 占位符
	DefaultValue any                     `json:"defaultValue,omitempty"` // 默认值
	Options      []FieldOption           `json:"options,omitempty"`      // 选项（用于 select/checkbox）
	Validation   *FieldValidation        `json:"validation,omitempty"`   // 验证规则
	Extra        map[string]any          `json:"extra,omitempty"`        // 额外数据（用于扩展配置，如多选日期）
}

// FieldOption 字段选项
type FieldOption struct {
	Label       string         `json:"label"`                 // 显示文本
	Value       any            `json:"value"`                 // 实际值
	Description string         `json:"description,omitempty"` // 描述信息（可选）
	Disabled    bool           `json:"disabled,omitempty"`    // 是否禁用
	Icon        string         `json:"icon,omitempty"`        // 图标（可选）
	Extra       map[string]any `json:"extra,omitempty"`       // 额外数据（用于富展示）
}

// FieldValidation 字段验证规则
type FieldValidation struct {
	Min     *float64 `json:"min,omitempty"`     // 最小值（数字）/ 最小长度（文本）
	Max     *float64 `json:"max,omitempty"`     // 最大值（数字）/ 最大长度（文本）
	Pattern string   `json:"pattern,omitempty"` // 正则表达式
	Message string   `json:"message,omitempty"` // 验证失败提示
}

// WorkflowAction 工作流操作
type WorkflowAction struct {
	ID      string                `json:"id"`
	Type    WorkflowActionType    `json:"type"`              // 操作类型（类型安全）
	Label   string                `json:"label"`             // 显示文本
	Event   WorkflowEvent         `json:"event"`             // 触发事件（类型安全）
	Style   WorkflowActionStyle   `json:"style,omitempty"`   // 按钮样式（类型安全）
	Payload any                   `json:"payload,omitempty"` // 携带数据（V3改进：使用any支持强类型结构体）
	Fields  []WorkflowActionField `json:"fields,omitempty"`  // 字段定义（用于动态表单）
}

// NewSession 创建新会话
func NewSession(orgID, userID, agentType string) *Session {
	now := time.Now()
	return &Session{
		ID:        generateID(),
		OrgID:     orgID,
		UserID:    userID,
		AgentType: agentType,
		State:     StateIdle,
		Messages:  make([]Message, 0),
		Data:      make(map[string]any),
		Metadata:  make(map[string]any),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage 添加消息
func (s *Session) AddMessage(role MessageRole, content string) {
	s.Messages = append(s.Messages, Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// GetLastUserMessage 获取最后一条用户消息
func (s *Session) GetLastUserMessage() *Message {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if s.Messages[i].Role == RoleUser {
			return &s.Messages[i]
		}
	}
	return nil
}

// SetState 设置状态
func (s *Session) SetState(state SessionState, desc string) {
	s.State = state
	s.StateDesc = desc
	s.UpdatedAt = time.Now()
}

// SetError 设置错误
func (s *Session) SetError(err string) {
	s.Error = err
	s.State = StateFailed
	s.UpdatedAt = time.Now()
}

// IsActive 是否活跃（未完成且未失败）
func (s *Session) IsActive() bool {
	return s.State != StateCompleted && s.State != StateFailed
}

// IsCompleted 是否已完成
func (s *Session) IsCompleted() bool {
	return s.State == StateCompleted
}

// IsFailed 是否失败
func (s *Session) IsFailed() bool {
	return s.State == StateFailed
}

// generateID 生成唯一 ID（简单实现）
func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
