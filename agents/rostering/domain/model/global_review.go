package model

// ============================================================
// 全局评审数据模型
// 用于草案完成后的规则和个人需求逐条评审、对论迭代、批量修改
// ============================================================

// ReviewItemType 评审项类型
type ReviewItemType string

const (
	// ReviewItemTypeRule 规则类型
	ReviewItemTypeRule ReviewItemType = "rule"
	// ReviewItemTypePersonalNeed 个人需求类型
	ReviewItemTypePersonalNeed ReviewItemType = "personal_need"
)

// ReviewItem 统一评审项（规则或个人需求的抽象）
type ReviewItem struct {
	// ID 评审项唯一标识
	ID string `json:"id"`

	// Type 类型: rule | personal_need
	Type ReviewItemType `json:"type"`

	// Name 名称（规则名称或需求描述）
	Name string `json:"name"`

	// Description 详细描述
	Description string `json:"description"`

	// Priority 优先级 (1-10, 数字越小优先级越高)
	Priority int `json:"priority"`

	// SourceRule 来源规则（当 Type=rule 时）
	SourceRule *Rule `json:"sourceRule,omitempty"`

	// SourceNeed 来源个人需求（当 Type=personal_need 时）
	SourceNeed *PersonalNeed `json:"sourceNeed,omitempty"`

	// AffectedStaffIDs 受影响的人员ID列表
	AffectedStaffIDs []string `json:"affectedStaffIds,omitempty"`

	// AffectedDates 受影响的日期列表
	AffectedDates []string `json:"affectedDates,omitempty"`

	// AffectedShiftIDs 受影响的班次ID列表
	AffectedShiftIDs []string `json:"affectedShiftIds,omitempty"`
}

// ModificationOpinionStatus 修改意见状态
type ModificationOpinionStatus string

const (
	// OpinionStatusPending 待评审
	OpinionStatusPending ModificationOpinionStatus = "pending"
	// OpinionStatusApproved 已通过
	OpinionStatusApproved ModificationOpinionStatus = "approved"
	// OpinionStatusRejected 已拒绝
	OpinionStatusRejected ModificationOpinionStatus = "rejected"
	// OpinionStatusConflict 存在冲突，需人工介入
	OpinionStatusConflict ModificationOpinionStatus = "conflict"
	// OpinionStatusManualReview 需人工评审（对论未达共识）
	OpinionStatusManualReview ModificationOpinionStatus = "manual_review"
)

// ModificationOpinion 修改意见
type ModificationOpinion struct {
	// ID 意见唯一标识
	ID string `json:"id"`

	// ReviewItemID 关联的评审项ID
	ReviewItemID string `json:"reviewItemId"`

	// ReviewItemType 评审项类型
	ReviewItemType ReviewItemType `json:"reviewItemType"`

	// ReviewItemName 评审项名称
	ReviewItemName string `json:"reviewItemName"`

	// IsViolated 是否违规
	IsViolated bool `json:"isViolated"`

	// ViolationDescription 违规描述
	ViolationDescription string `json:"violationDescription,omitempty"`

	// Suggestion 修改建议
	Suggestion string `json:"suggestion,omitempty"`

	// ProposedChanges 建议的具体修改
	// 格式: date -> { "add": []staffID, "remove": []staffID }
	ProposedChanges map[string]*ReviewScheduleChange `json:"proposedChanges,omitempty"`

	// AffectedStaffIDs 受影响的人员ID列表
	AffectedStaffIDs []string `json:"affectedStaffIds,omitempty"`

	// AffectedDates 受影响的日期列表
	AffectedDates []string `json:"affectedDates,omitempty"`

	// AffectedShiftIDs 受影响的班次ID列表
	AffectedShiftIDs []string `json:"affectedShiftIds,omitempty"`

	// Severity 严重程度: critical | warning | low
	Severity string `json:"severity"`

	// Status 状态
	Status ModificationOpinionStatus `json:"status"`

	// ReviewComments 评审意见（对论过程中的反馈）
	ReviewComments string `json:"reviewComments,omitempty"`

	// ConflictingOpinionIDs 冲突的意见ID列表
	ConflictingOpinionIDs []string `json:"conflictingOpinionIds,omitempty"`
}

// ReviewScheduleChange 评审修改建议的排班变更
type ReviewScheduleChange struct {
	// ShiftID 班次ID
	ShiftID string `json:"shiftId"`

	// AddStaffIDs 需要添加的人员ID列表
	AddStaffIDs []string `json:"addStaffIds,omitempty"`

	// RemoveStaffIDs 需要移除的人员ID列表
	RemoveStaffIDs []string `json:"removeStaffIds,omitempty"`

	// Reason 变更原因
	Reason string `json:"reason,omitempty"`
}

// ============================================================
// 对论迭代上下文
// ============================================================

// DebateContext 对论迭代上下文
type DebateContext struct {
	// CurrentRound 当前轮次（从1开始）
	CurrentRound int `json:"currentRound"`

	// MaxRounds 最大轮次（默认3）
	MaxRounds int `json:"maxRounds"`

	// Opinions 待对论的修改意见列表
	Opinions []*ModificationOpinion `json:"opinions"`

	// DebateHistory 对论历史记录
	DebateHistory []*DebateRound `json:"debateHistory"`

	// Converged 是否已达成共识
	Converged bool `json:"converged"`
}

// DebateRound 对论轮次记录
type DebateRound struct {
	// Round 轮次编号
	Round int `json:"round"`

	// ReviewerChallenge 评审者质疑内容
	ReviewerChallenge string `json:"reviewerChallenge"`

	// DefenderResponse 辩护者回应内容
	DefenderResponse string `json:"defenderResponse"`

	// OpinionVerdicts 单个意见的裁定结果
	OpinionVerdicts []OpinionVerdict `json:"opinionVerdicts,omitempty"`

	// Verdict 本轮裁定: approved | rejected | continue
	Verdict string `json:"verdict"`

	// Reason 裁定理由
	Reason string `json:"reason"`
}

// OpinionVerdict 单个意见的裁定
type OpinionVerdict struct {
	// OpinionID 意见ID
	OpinionID string `json:"opinionId"`

	// Verdict 裁定: approved | rejected | continue
	Verdict string `json:"verdict"`

	// Reason 裁定理由
	Reason string `json:"reason"`
}

// NewDebateContext 创建新的对论上下文
func NewDebateContext(opinions []*ModificationOpinion, maxRounds int) *DebateContext {
	if maxRounds <= 0 {
		maxRounds = 3 // 默认最多3轮
	}
	return &DebateContext{
		CurrentRound:  0,
		MaxRounds:     maxRounds,
		Opinions:      opinions,
		DebateHistory: make([]*DebateRound, 0),
		Converged:     false,
	}
}

// ============================================================
// 对论结果
// ============================================================

// DebateResult 对论结果
type DebateResult struct {
	// Converged 是否收敛（达成共识）
	Converged bool `json:"converged"`

	// TotalRounds 实际迭代轮次
	TotalRounds int `json:"totalRounds"`

	// ApprovedOpinions 通过的修改意见
	ApprovedOpinions []*ModificationOpinion `json:"approvedOpinions"`

	// RejectedOpinions 被拒绝的修改意见
	RejectedOpinions []*ModificationOpinion `json:"rejectedOpinions"`

	// ConflictOpinions 存在冲突的意见（需人工介入）
	ConflictOpinions []*ModificationOpinion `json:"conflictOpinions"`

	// ManualReviewOpinions 需人工评审的意见（对论未达共识）
	ManualReviewOpinions []*ModificationOpinion `json:"manualReviewOpinions"`

	// Summary 对论总结
	Summary string `json:"summary"`
}

// HasManualReviewItems 是否有需人工处理的项目
func (r *DebateResult) HasManualReviewItems() bool {
	return len(r.ConflictOpinions) > 0 || len(r.ManualReviewOpinions) > 0
}

// GetApprovedChanges 获取所有通过的变更
func (r *DebateResult) GetApprovedChanges() map[string]map[string]*ReviewScheduleChange {
	// 返回: shiftID -> date -> ReviewScheduleChange
	result := make(map[string]map[string]*ReviewScheduleChange)
	for _, opinion := range r.ApprovedOpinions {
		for date, change := range opinion.ProposedChanges {
			if change == nil {
				continue
			}
			shiftID := change.ShiftID
			if shiftID == "" {
				continue
			}
			if result[shiftID] == nil {
				result[shiftID] = make(map[string]*ReviewScheduleChange)
			}
			// 合并同一日期的变更
			if existing, ok := result[shiftID][date]; ok {
				existing.AddStaffIDs = append(existing.AddStaffIDs, change.AddStaffIDs...)
				existing.RemoveStaffIDs = append(existing.RemoveStaffIDs, change.RemoveStaffIDs...)
			} else {
				result[shiftID][date] = change
			}
		}
	}
	return result
}

// ============================================================
// 全局评审执行结果
// ============================================================

// GlobalReviewResult 全局评审执行结果
type GlobalReviewResult struct {
	// TotalItems 评审项总数
	TotalItems int `json:"totalItems"`

	// ReviewedItems 已评审项数
	ReviewedItems int `json:"reviewedItems"`

	// ViolatedItems 违规项数
	ViolatedItems int `json:"violatedItems"`

	// AllOpinions 所有修改意见
	AllOpinions []*ModificationOpinion `json:"allOpinions"`

	// DebateResult 对论结果
	DebateResult *DebateResult `json:"debateResult,omitempty"`

	// ModifiedDraft 修改后的草案
	ModifiedDraft *ScheduleDraft `json:"modifiedDraft,omitempty"`

	// NeedsManualReview 是否需要人工介入
	NeedsManualReview bool `json:"needsManualReview"`

	// ManualReviewItems 需人工处理的项目
	ManualReviewItems []*ModificationOpinion `json:"manualReviewItems,omitempty"`

	// Summary 评审总结
	Summary string `json:"summary"`

	// ExecutionTime 执行时间（秒）
	ExecutionTime float64 `json:"executionTime"`
}

// ============================================================
// 冲突检测
// ============================================================

// ConflictGroup 冲突组（一组相互冲突的意见）
type ConflictGroup struct {
	// ID 冲突组ID
	ID string `json:"id"`

	// OpinionIDs 冲突意见ID列表
	OpinionIDs []string `json:"opinionIds"`

	// Opinions 冲突意见列表
	Opinions []*ModificationOpinion `json:"opinions"`

	// ConflictType 冲突类型: staff_overload | contradicting_rules | resource_contention
	ConflictType string `json:"conflictType"`

	// Description 冲突描述
	Description string `json:"description"`

	// AffectedDates 受影响日期
	AffectedDates []string `json:"affectedDates"`

	// AffectedStaffIDs 受影响人员
	AffectedStaffIDs []string `json:"affectedStaffIds"`
}

// ============================================================
// 评审进度回调
// ============================================================

// GlobalReviewProgressType 全局评审进度类型
type GlobalReviewProgressType string

const (
	// ReviewProgressItemReviewing 正在评审某项
	ReviewProgressItemReviewing GlobalReviewProgressType = "item_reviewing"
	// ReviewProgressItemCompleted 某项评审完成
	ReviewProgressItemCompleted GlobalReviewProgressType = "item_completed"
	// ReviewProgressDebating 对论中
	ReviewProgressDebating GlobalReviewProgressType = "debating"
	// ReviewProgressModifying 修改草案中
	ReviewProgressModifying GlobalReviewProgressType = "modifying"
	// ReviewProgressNeedsManual 需人工介入
	ReviewProgressNeedsManual GlobalReviewProgressType = "needs_manual"
	// ReviewProgressCompleted 评审完成
	ReviewProgressCompleted GlobalReviewProgressType = "completed"
)

// GlobalReviewProgress 全局评审进度
type GlobalReviewProgress struct {
	// Type 进度类型
	Type GlobalReviewProgressType `json:"type"`

	// CurrentItem 当前评审项序号（从1开始）
	CurrentItem int `json:"currentItem"`

	// TotalItems 评审项总数
	TotalItems int `json:"totalItems"`

	// CurrentItemName 当前评审项名称
	CurrentItemName string `json:"currentItemName"`

	// CurrentItemType 当前评审项类型
	CurrentItemType ReviewItemType `json:"currentItemType"`

	// DebateRound 当前对论轮次（仅对论阶段有效）
	DebateRound int `json:"debateRound,omitempty"`

	// ViolatedCount 已发现违规数
	ViolatedCount int `json:"violatedCount"`

	// Message 进度消息
	Message string `json:"message"`
}

// GlobalReviewProgressCallback 全局评审进度回调函数
type GlobalReviewProgressCallback func(progress *GlobalReviewProgress)

// ============================================================
// 人工处理相关模型
// ============================================================

// ManualReviewContext 人工评审上下文
type ManualReviewContext struct {
	// ManualReviewItems 需人工处理的项目列表
	ManualReviewItems []*ManualReviewItem `json:"manualReviewItems"`
}

// ManualReviewItem 需人工处理的项目
type ManualReviewItem struct {
	// OpinionID 意见ID
	OpinionID string `json:"opinionId"`

	// ReviewItemName 评审项名称
	ReviewItemName string `json:"reviewItemName"`

	// ReviewItemType 评审项类型
	ReviewItemType ReviewItemType `json:"reviewItemType"`

	// ViolationDescription 违规描述
	ViolationDescription string `json:"violationDescription"`

	// Suggestion 修改建议
	Suggestion string `json:"suggestion"`

	// Status 状态
	Status ModificationOpinionStatus `json:"status"`

	// ConflictReason 冲突原因（如果是冲突项）
	ConflictReason string `json:"conflictReason,omitempty"`
}

// ManualReviewModifyResult 人工修改结果
type ManualReviewModifyResult struct {
	// Success 是否成功
	Success bool `json:"success"`

	// Summary 处理摘要
	Summary string `json:"summary"`

	// AppliedChanges 应用的变更描述
	AppliedChanges []string `json:"appliedChanges,omitempty"`

	// ModifiedDraft 修改后的草案
	ModifiedDraft *ScheduleDraft `json:"modifiedDraft,omitempty"`
}
