package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule_v4.create 工作流定义
// V4排班创建工作流 - 基于确定性规则引擎
// ============================================================

const (
	// WorkflowScheduleCreateV4 排班创建工作流 V4
	WorkflowScheduleCreateV4 engine.Workflow = "schedule_v4.create"
)

// ============================================================
// schedule_v4.create 工作流 - 状态定义
// 命名规范: CreateV4State[StateName]
// ============================================================

const (
	// ========== 主流程状态 ==========

	// CreateV4StateInit 初始化状态
	CreateV4StateInit engine.State = "_schedule_v4_create_init_"

	// CreateV4StateInfoCollecting 信息收集阶段
	CreateV4StateInfoCollecting engine.State = "_schedule_v4_create_info_collecting_"

	// CreateV4StateConfirmPeriod 确认排班时间阶段
	CreateV4StateConfirmPeriod engine.State = "_schedule_v4_create_confirm_period_"

	// CreateV4StateConfirmShifts 确认班次选择阶段
	CreateV4StateConfirmShifts engine.State = "_schedule_v4_create_confirm_shifts_"

	// CreateV4StateConfirmStaffCount 确认人数配置阶段
	CreateV4StateConfirmStaffCount engine.State = "_schedule_v4_create_confirm_staff_count_"

	// CreateV4StatePersonalNeeds 个人需求收集与确认阶段
	CreateV4StatePersonalNeeds engine.State = "_schedule_v4_create_personal_needs_"

	// CreateV4StateRuleOrganization 规则组织阶段（V4新增）
	CreateV4StateRuleOrganization engine.State = "_schedule_v4_create_rule_organization_"

	// CreateV4StateScheduling 排班执行阶段（使用确定性规则引擎）
	CreateV4StateScheduling engine.State = "_schedule_v4_create_scheduling_"


	// CreateV4StateValidation 确定性校验阶段
	CreateV4StateValidation engine.State = "_schedule_v4_create_validation_"

	// CreateV4StateReview 排班审核阶段
	CreateV4StateReview engine.State = "_schedule_v4_create_review_"

	// CreateV4StateCompleted 完成状态
	CreateV4StateCompleted engine.State = "_schedule_v4_create_completed_"

	// CreateV4StateFailed 失败状态
	CreateV4StateFailed engine.State = "_schedule_v4_create_failed_"

	// CreateV4StateCancelled 取消状态
	CreateV4StateCancelled engine.State = "_schedule_v4_create_cancelled_"
)

// ============================================================
// schedule_v4.create 工作流 - 事件定义
// 命名规范: CreateV4Event[EventName]
// ============================================================

const (
	// CreateV4EventStart 开始工作流
	CreateV4EventStart engine.Event = engine.EventStart // 复用通用启动事件

	// CreateV4EventInfoCollected 信息收集完成
	CreateV4EventInfoCollected engine.Event = "_schedule_v4_create_info_collected_"

	// CreateV4EventPeriodConfirmed 排班时间确认
	CreateV4EventPeriodConfirmed engine.Event = "_schedule_v4_create_period_confirmed_"

	// CreateV4EventShiftsConfirmed 班次确认
	CreateV4EventShiftsConfirmed engine.Event = "_schedule_v4_create_shifts_confirmed_"

	// CreateV4EventStaffCountConfirmed 人数配置确认
	CreateV4EventStaffCountConfirmed engine.Event = "_schedule_v4_create_staff_count_confirmed_"

	// CreateV4EventPersonalNeedsConfirmed 个人需求确认
	CreateV4EventPersonalNeedsConfirmed engine.Event = "_schedule_v4_create_personal_needs_confirmed_"

	// CreateV4EventTemporaryNeedsTextSubmitted 临时需求文本提交
	CreateV4EventTemporaryNeedsTextSubmitted engine.Event = "_schedule_v4_create_temporary_needs_text_submitted_"

	// CreateV4EventRulesOrganized 规则组织完成
	CreateV4EventRulesOrganized engine.Event = "_schedule_v4_create_rules_organized_"

	// CreateV4EventSchedulingComplete 排班执行完成
	CreateV4EventSchedulingComplete engine.Event = "_schedule_v4_create_scheduling_complete_"


	// CreateV4EventValidationComplete 校验完成
	CreateV4EventValidationComplete engine.Event = "_schedule_v4_create_validation_complete_"

	// CreateV4EventReviewConfirmed 审核确认
	CreateV4EventReviewConfirmed engine.Event = engine.EventConfirm // 复用通用确认事件

	// CreateV4EventUserCancel 用户取消
	CreateV4EventUserCancel engine.Event = engine.EventCancel // 复用通用取消事件

	// CreateV4EventUserModify 用户修改
	CreateV4EventUserModify engine.Event = engine.EventModify // 复用通用修改事件
)
