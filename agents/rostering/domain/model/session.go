package model

import (
	"time"

	"jusha/mcp/pkg/workflow/session"
)

// 直接使用 pkg/workflow/session 的类型
type (
	Session        = session.Session
	SessionState   = session.SessionState
	Message        = session.Message
	MessageRole    = session.MessageRole
	WorkflowMeta   = session.WorkflowMeta
	WorkflowAction = session.WorkflowAction
)

// 常量别名
const (
	StateIdle       = session.StateIdle
	StateProcessing = session.StateProcessing
	StateWaiting    = session.StateWaiting
	StateCompleted  = session.StateCompleted
	StateFailed     = session.StateFailed

	RoleUser      = session.RoleUser
	RoleAssistant = session.RoleAssistant
	RoleSystem    = session.RoleSystem
)

// SessionMessage 保持向后兼容（别名）
type SessionMessage = Message

// StartSessionRequest 创建会话请求
type StartSessionRequest struct {
	OrgID          string         `json:"orgId"`          // 组织ID
	BizDateRange   string         `json:"bizDateRange"`   // 例如: 2025-10-01~2025-10-31
	Department     string         `json:"department"`     // 科室
	Modality       string         `json:"modality"`       // 设备类型（可选）
	InitialMessage string         `json:"initialMessage"` // 初始用户描述
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// SessionMessageRequest 用户追加消息
type SessionMessageRequest struct {
	Message string `json:"message"`
}

// Rostering 业务特定数据，存储在 Session.Data 中
// 使用这些 key 来访问业务数据：
const (
	DataKeyBizDateRange   = "bizDateRange"
	DataKeyDepartment     = "department"
	DataKeyModality       = "modality"
	DataKeyRules          = "rules"
	DataKeyScheduleResult = "scheduleResult"
	DataKeyContext        = "context"
	DataKeyDecisionLog    = "decisionLog"
	DataKeyExecutionPlan  = "executionPlan"
	DataKeyDraftVersion   = "draftVersion"
	DataKeyResultVersion  = "resultVersion"
)

// SchedulingContext 汇总外部依赖数据，减少重复调用
// 存储在 Session.Data[DataKeyContext]
type SchedulingContext struct {
	CurrentMonthSchedule  any            `json:"currentMonthSchedule,omitempty"`  // 本月排班（原始数据结构，由 data-server 返回）
	PreviousMonthSchedule any            `json:"previousMonthSchedule,omitempty"` // 上月排班
	StaffProfiles         any            `json:"staffProfiles,omitempty"`         // 人员及技能/负载信息
	GraphRules            any            `json:"graphRules,omitempty"`            // 图谱规则（技能分组间约束）
	CandidateRelations    any            `json:"candidateRelations,omitempty"`    // 候选人关系图（协作、上下级、冲突）
	Conflicts             any            `json:"conflicts,omitempty"`             // 冲突集合（请假、资质、排班冲突）
	Normalized            map[string]any `json:"normalized,omitempty"`            // 归一后的用于AI输入的结构
	Extra                 map[string]any `json:"extra,omitempty"`                 // 预留
}

// DecisionRecord 记录 AI 或系统的关键决策、推理片段，便于追踪
type DecisionRecord struct {
	Timestamp time.Time      `json:"timestamp"`
	Actor     string         `json:"actor"`  // ai/system/user
	Action    string         `json:"action"` // fetch_data/generate/resolve_conflict 等
	Detail    string         `json:"detail"`
	Data      map[string]any `json:"data,omitempty"`
	Stage     string         `json:"stage"` // collect|rules|graph|relation|generate|finalize
}

// ExecutionPlan 多意图执行计划
type ExecutionPlan struct {
	PlanID      string              `json:"planId"`                // 执行计划唯一ID
	Intents     []*IntentResult     `json:"intents"`               // 待执行的意图列表
	Current     int                 `json:"current"`               // 当前执行到第几个意图（索引）
	Status      ExecutionPlanStatus `json:"status"`                // 执行计划状态
	CreatedAt   time.Time           `json:"createdAt"`             // 计划创建时间
	UpdatedAt   time.Time           `json:"updatedAt"`             // 计划更新时间
	Results     []*IntentExecResult `json:"results,omitempty"`     // 每个意图的执行结果
	FailedIndex int                 `json:"failedIndex,omitempty"` // 失败的意图索引（-1表示没有失败）
}

// ExecutionPlanStatus 执行计划状态
type ExecutionPlanStatus string

const (
	ExecutionPlanStatusPending   ExecutionPlanStatus = "pending"   // 待确认
	ExecutionPlanStatusExecuting ExecutionPlanStatus = "executing" // 执行中
	ExecutionPlanStatusCompleted ExecutionPlanStatus = "completed" // 已完成
	ExecutionPlanStatusFailed    ExecutionPlanStatus = "failed"    // 执行失败
	ExecutionPlanStatusCancelled ExecutionPlanStatus = "cancelled" // 已取消
)

// IntentExecResult 意图执行结果
type IntentExecResult struct {
	IntentIndex int              `json:"intentIndex"` // 对应 ExecutionPlan.Intents 的索引
	IntentType  IntentType       `json:"intentType"`  // 意图类型
	Status      IntentExecStatus `json:"status"`      // 执行状态
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	Error       string           `json:"error,omitempty"`
	Result      map[string]any   `json:"result,omitempty"` // 执行结果数据
}

// IntentExecStatus 意图执行状态
type IntentExecStatus string

const (
	IntentExecStatusPending   IntentExecStatus = "pending"   // 待执行
	IntentExecStatusExecuting IntentExecStatus = "executing" // 执行中
	IntentExecStatusCompleted IntentExecStatus = "completed" // 已完成
	IntentExecStatusFailed    IntentExecStatus = "failed"    // 执行失败
	IntentExecStatusSkipped   IntentExecStatus = "skipped"   // 已跳过
)
