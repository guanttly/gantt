package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule_v3.create 工作流定义
// 渐进式排班创建工作流 V3
// ============================================================

const (
	// WorkflowScheduleCreateV3 排班创建工作流 V3
	WorkflowScheduleCreateV3 engine.Workflow = "schedule_v3.create"
)

// ============================================================
// schedule_v3.create 工作流 - 状态定义
// 命名规范: CreateV3State[StateName]
// ============================================================

const (
	// ========== 主流程状态 ==========

	// CreateV3StateInit 初始化状态
	CreateV3StateInit engine.State = "_schedule_v3_create_init_"

	// CreateV3StateInfoCollecting 信息收集阶段（调用 InfoCollect 子工作流）
	CreateV3StateInfoCollecting engine.State = "_schedule_v3_create_info_collecting_"

	// CreateV3StateConfirmPeriod 确认排班时间阶段
	CreateV3StateConfirmPeriod engine.State = "_schedule_v3_create_confirm_period_"

	// CreateV3StateConfirmShifts 确认班次选择阶段
	CreateV3StateConfirmShifts engine.State = "_schedule_v3_create_confirm_shifts_"

	// CreateV3StateConfirmStaffCount 确认人数配置阶段
	CreateV3StateConfirmStaffCount engine.State = "_schedule_v3_create_confirm_staff_count_"

	// CreateV3StatePersonalNeeds 个人需求收集与确认阶段
	CreateV3StatePersonalNeeds engine.State = "_schedule_v3_create_personal_needs_"

	// CreateV3StateRequirementAssessment 需求评估状态（V3核心，LLM评估所有需求并生成任务计划）
	CreateV3StateRequirementAssessment engine.State = "_schedule_v3_create_requirement_assessment_"

	// CreateV3StatePlanReview 计划预览与确认状态
	CreateV3StatePlanReview engine.State = "_schedule_v3_create_plan_review_"

	// CreateV3StateProgressiveTask 渐进式任务执行状态
	CreateV3StateProgressiveTask engine.State = "_schedule_v3_create_progressive_task_"

	// CreateV3StateTaskReview 任务审核状态（非连续排班模式下等待用户操作）
	CreateV3StateTaskReview engine.State = "_schedule_v3_create_task_review_"

	// CreateV3StateWaitingAdjustment 等待用户输入调整需求状态
	CreateV3StateWaitingAdjustment engine.State = "_schedule_v3_create_waiting_adjustment_"

	// CreateV3StateTaskFailed 任务执行失败状态（等待用户决定是否继续）
	CreateV3StateTaskFailed engine.State = "_schedule_v3_create_task_failed_"

	// CreateV3StateGlobalReview 全局评审状态（规则和个人需求逐条评审、对论迭代）
	CreateV3StateGlobalReview engine.State = "_schedule_v3_create_global_review_"

	// CreateV3StateGlobalReviewManual 全局评审人工处理状态（有冲突或未达共识的项目需人工介入）
	CreateV3StateGlobalReviewManual engine.State = "_schedule_v3_create_global_review_manual_"

	// CreateV3StateConfirmSaving 确认保存阶段
	CreateV3StateConfirmSaving engine.State = "_schedule_v3_create_confirm_saving_"

	// ========== 终态 ==========
	// 复用通用终态常量
	CreateV3StateCompleted = engine.StateCompleted // "_completed_"
	CreateV3StateFailed    = engine.StateFailed    // "_failed_"
	CreateV3StateCancelled = engine.StateCancelled // "_cancelled_"
)

// ============================================================
// schedule_v3.create 工作流 - 事件定义
// 命名规范: CreateV3Event[EventName]
// ============================================================

const (
	// ========== 启动事件 ==========

	// CreateV3EventStart 启动工作流事件
	CreateV3EventStart engine.Event = engine.EventStart // 复用通用启动事件

	// ========== 阶段完成事件 ==========

	// CreateV3EventInfoCollected 信息收集完成
	CreateV3EventInfoCollected engine.Event = "_schedule_v3_create_info_collected_"

	// CreateV3EventPeriodConfirmed 排班时间确认完成
	CreateV3EventPeriodConfirmed engine.Event = "_schedule_v3_create_period_confirmed_"

	// CreateV3EventShiftsConfirmed 班次选择确认完成
	CreateV3EventShiftsConfirmed engine.Event = "_schedule_v3_create_shifts_confirmed_"

	// CreateV3EventStaffCountConfirmed 人数配置确认完成
	CreateV3EventStaffCountConfirmed engine.Event = "_schedule_v3_create_staff_count_confirmed_"

	// CreateV3EventPersonalNeedsConfirmed 个人需求确认完成
	CreateV3EventPersonalNeedsConfirmed engine.Event = "_schedule_v3_create_personal_needs_confirmed_"

	// CreateV3EventRequirementAssessed 需求评估完成
	CreateV3EventRequirementAssessed engine.Event = "_schedule_v3_create_requirement_assessed_"

	// CreateV3EventTemporaryNeedsTextSubmitted 临时需求文本提交
	CreateV3EventTemporaryNeedsTextSubmitted engine.Event = "_schedule_v3_create_temporary_needs_text_submitted_"

	// CreateV3EventPlanConfirmed 任务计划确认并开始执行
	CreateV3EventPlanConfirmed engine.Event = "_schedule_v3_create_plan_confirmed_"

	// CreateV3EventPlanAdjust 任务计划调整请求
	CreateV3EventPlanAdjust engine.Event = "_schedule_v3_create_plan_adjust_"

	// CreateV3EventPlanAdjusted 任务计划调整完成
	CreateV3EventPlanAdjusted engine.Event = "_schedule_v3_create_plan_adjusted_"

	// CreateV3EventTaskCompleted 单个任务执行完成
	CreateV3EventTaskCompleted engine.Event = "_schedule_v3_create_task_completed_"

	// CreateV3EventAllTasksComplete 所有任务完成
	CreateV3EventAllTasksComplete engine.Event = "_schedule_v3_create_all_tasks_complete_"

	// CreateV3EventSaveCompleted 保存完成
	CreateV3EventSaveCompleted engine.Event = "_schedule_v3_create_save_completed_"

	// ========== 子工作流事件 ==========

	// CreateV3EventSubCancelled 子工作流被取消
	CreateV3EventSubCancelled engine.Event = "_schedule_v3_create_sub_cancelled_"

	// CreateV3EventSubFailed 子工作流失败
	CreateV3EventSubFailed engine.Event = "_schedule_v3_create_sub_failed_"

	// CreateV3EventEnterTaskReview 进入任务审核状态（非连续排班模式下，任务完成后进入审核）
	CreateV3EventEnterTaskReview engine.Event = "_schedule_v3_create_enter_task_review_"

	// CreateV3EventTaskReviewConfirmed 任务审核确认继续
	CreateV3EventTaskReviewConfirmed engine.Event = "_schedule_v3_create_task_review_confirmed_"

	// CreateV3EventTaskReviewAdjust 任务审核提出调整
	CreateV3EventTaskReviewAdjust engine.Event = "_schedule_v3_create_task_review_adjust_"

	// CreateV3EventUserAdjustmentMessage 用户发送调整需求消息
	CreateV3EventUserAdjustmentMessage engine.Event = "_schedule_v3_create_user_adjustment_message_"

	// CreateV3EventTaskAdjusted 任务调整完成事件（Adjust 子工作流返回）
	CreateV3EventTaskAdjusted engine.Event = "_schedule_v3_create_task_adjusted_"

	// CreateV3EventTaskFailed 任务执行失败事件（验证失败后触发）
	CreateV3EventTaskFailed engine.Event = "_schedule_v3_create_task_failed_"

	// CreateV3EventTaskFailedContinue 任务失败后用户选择继续
	CreateV3EventTaskFailedContinue engine.Event = "_schedule_v3_create_task_failed_continue_"

	// CreateV3EventTaskFailedCancel 任务失败后用户选择取消
	CreateV3EventTaskFailedCancel engine.Event = "_schedule_v3_create_task_failed_cancel_"

	// ========== 全局评审事件 ==========

	// CreateV3EventStartGlobalReview 开始全局评审
	CreateV3EventStartGlobalReview engine.Event = "_schedule_v3_create_start_global_review_"

	// CreateV3EventGlobalReviewCompleted 全局评审完成（无需人工介入）
	CreateV3EventGlobalReviewCompleted engine.Event = "_schedule_v3_create_global_review_completed_"

	// CreateV3EventGlobalReviewNeedsManual 全局评审需人工介入
	CreateV3EventGlobalReviewNeedsManual engine.Event = "_schedule_v3_create_global_review_needs_manual_"

	// CreateV3EventGlobalReviewManualConfirmed 人工处理确认完成
	CreateV3EventGlobalReviewManualConfirmed engine.Event = "_schedule_v3_create_global_review_manual_confirmed_"

	// CreateV3EventGlobalReviewSkip 跳过全局评审（用户选择直接保存）
	CreateV3EventGlobalReviewSkip engine.Event = "_schedule_v3_create_global_review_skip_"

	// ========== 用户操作事件 ==========

	// CreateV3EventUserCancel 用户取消操作
	CreateV3EventUserCancel engine.Event = engine.EventCancel // 复用通用取消事件

	// CreateV3EventUserConfirm 用户确认操作
	CreateV3EventUserConfirm engine.Event = engine.EventConfirm // 复用通用确认事件

	// CreateV3EventUserModify 用户修改操作
	CreateV3EventUserModify engine.Event = engine.EventModify // 复用通用修改事件

	// CreateV3EventModifyStaffCount 修改人数配置
	CreateV3EventModifyStaffCount engine.Event = "_schedule_v3_create_modify_staff_count_"

	// ========== 错误处理事件 ==========

	// CreateV3EventError 错误事件（通用）
	CreateV3EventError engine.Event = "_schedule_v3_create_error_"

	// CreateV3EventRetry 重试当前操作
	CreateV3EventRetry engine.Event = "_schedule_v3_create_retry_"

	// CreateV3EventSkipPhase 跳过当前阶段
	CreateV3EventSkipPhase engine.Event = "_schedule_v3_create_skip_phase_"
)

// ============================================================
// schedule_v3.create 工作流 - 状态描述（用于前端显示）
// ============================================================

var CreateV3StateDescriptions = map[engine.State]string{
	CreateV3StateInit:                  "初始化排班",
	CreateV3StateInfoCollecting:        "收集排班信息",
	CreateV3StateConfirmPeriod:         "确认排班时间",
	CreateV3StateConfirmShifts:         "确认班次选择",
	CreateV3StateConfirmStaffCount:     "确认人数配置",
	CreateV3StatePersonalNeeds:         "确认个人需求",
	CreateV3StateRequirementAssessment: "评估需求并生成任务计划",
	CreateV3StatePlanReview:            "预览并确认任务计划",
	CreateV3StateProgressiveTask:       "执行渐进式任务",
	CreateV3StateTaskReview:            "任务审核",
	CreateV3StateTaskFailed:            "任务执行失败",
	CreateV3StateGlobalReview:          "全局规则评审",
	CreateV3StateGlobalReviewManual:    "全局评审人工处理",
	CreateV3StateConfirmSaving:         "确认并保存",
	CreateV3StateCompleted:             "排班完成",
	CreateV3StateFailed:                "排班失败",
	CreateV3StateCancelled:             "已取消",
}

// GetCreateV3StateDescription 获取状态的中文描述
func GetCreateV3StateDescription(state engine.State) string {
	if desc, ok := CreateV3StateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}

// ============================================================
// 阶段常量定义（用于上下文中标识当前处理阶段）
// ============================================================

const (
	PhaseV3InfoCollect           = "info_collect"           // 信息收集
	PhaseV3PersonalNeed          = "personal_need"          // 个人需求
	PhaseV3RequirementAssessment = "requirement_assessment" // 需求评估
	PhaseV3PlanReview            = "plan_review"            // 计划预览
	PhaseV3ProgressiveTask       = "progressive_task"       // 渐进式任务执行
	PhaseV3GlobalReview          = "global_review"          // 全局评审
	PhaseV3ConfirmSave           = "confirm_save"           // 确认保存
)
