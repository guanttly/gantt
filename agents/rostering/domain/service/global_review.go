package service

import (
	"context"

	"jusha/agent/rostering/domain/model"
)

// IGlobalReviewService 全局评审服务接口
// 提供草案完成后的规则和个人需求逐条评审、对论迭代、批量修改功能
type IGlobalReviewService interface {
	// ============================================================
	// 评审项合并
	// ============================================================

	// MergeToReviewItems 将规则和个人需求合并为统一的评审项列表
	// 按优先级排序，相同优先级时规则优先于个人需求
	// rules: 规则列表
	// personalNeeds: 个人需求列表
	// 返回: 排序后的评审项列表
	MergeToReviewItems(rules []*model.Rule, personalNeeds []*model.PersonalNeed) []*model.ReviewItem

	// ============================================================
	// 逐条评审
	// ============================================================

	// ReviewItemAgainstDraft 逐条评审项对照排班表评审
	// 让LLM检查该评审项是否在排班表中被满足
	// ctx: 上下文
	// item: 评审项
	// draft: 完整排班草案
	// allStaffList: 所有人员列表（用于姓名映射）
	// allShifts: 所有班次列表（用于班次信息展示）
	// 返回: 修改意见（如果违规则包含具体建议）
	ReviewItemAgainstDraft(
		ctx context.Context,
		item *model.ReviewItem,
		draft *model.ScheduleDraft,
		allStaffList []*model.Employee,
		allShifts []*model.Shift,
	) (*model.ModificationOpinion, error)

	// ReviewAllItems 批量评审所有评审项
	// 逐条调用 ReviewItemAgainstDraft 并收集结果
	// progressCallback: 进度回调（可选）
	// 返回: 所有修改意见列表
	ReviewAllItems(
		ctx context.Context,
		items []*model.ReviewItem,
		draft *model.ScheduleDraft,
		allStaffList []*model.Employee,
		allShifts []*model.Shift,
		progressCallback model.GlobalReviewProgressCallback,
	) ([]*model.ModificationOpinion, error)

	// ============================================================
	// 冲突检测
	// ============================================================

	// DetectConflicts 检测修改意见之间的冲突
	// 识别相互矛盾的意见（如一个要求加人、另一个要求减人）
	// 冲突的意见需要人工介入
	// opinions: 修改意见列表
	// 返回: 冲突组列表
	DetectConflicts(opinions []*model.ModificationOpinion) []*model.ConflictGroup

	// ============================================================
	// 对论迭代
	// ============================================================

	// DebateOpinions 对修改意见进行对论迭代
	// 评审者质疑 → 辩护者回应，最多迭代maxRounds轮
	// 达成共识或超过最大轮次后返回结果
	// ctx: 上下文
	// opinions: 待对论的修改意见（不含冲突项）
	// draft: 当前排班草案
	// maxRounds: 最大对论轮次（默认3）
	// progressCallback: 进度回调（可选）
	// 返回: 对论结果
	DebateOpinions(
		ctx context.Context,
		opinions []*model.ModificationOpinion,
		draft *model.ScheduleDraft,
		maxRounds int,
		progressCallback model.GlobalReviewProgressCallback,
	) (*model.DebateResult, error)

	// ============================================================
	// 批量修改
	// ============================================================

	// ApplyApprovedOpinions 应用通过的修改意见到草案
	// 将对论通过的意见一次性传给LLM执行排班调整
	// ctx: 上下文
	// draft: 当前排班草案
	// approvedOpinions: 已通过的修改意见列表
	// allStaffList: 所有人员列表
	// allShifts: 所有班次列表
	// 返回: 修改后的草案
	ApplyApprovedOpinions(
		ctx context.Context,
		draft *model.ScheduleDraft,
		approvedOpinions []*model.ModificationOpinion,
		allStaffList []*model.Employee,
		allShifts []*model.Shift,
	) (*model.ScheduleDraft, error)

	// ============================================================
	// 完整流程
	// ============================================================

	// ExecuteGlobalReview 执行完整的全局评审流程
	// 1. 合并规则和个人需求为评审项
	// 2. 逐条评审收集修改意见
	// 3. 检测冲突，冲突项标记人工介入
	// 4. 非冲突项进入对论迭代
	// 5. 应用通过的修改意见
	// 6. 返回结果（含需人工处理的项目）
	ExecuteGlobalReview(
		ctx context.Context,
		rules []*model.Rule,
		personalNeeds []*model.PersonalNeed,
		draft *model.ScheduleDraft,
		allStaffList []*model.Employee,
		allShifts []*model.Shift,
		maxDebateRounds int,
		progressCallback model.GlobalReviewProgressCallback,
	) (*model.GlobalReviewResult, error)
}
