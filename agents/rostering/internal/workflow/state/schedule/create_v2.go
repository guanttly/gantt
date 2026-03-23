package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// schedule_v2.create 工作流定义
// 新版排班创建工作流，按优先级顺序处理不同类型班次
// ============================================================

const (
	// WorkflowScheduleCreateV2 排班创建工作流 V2
	WorkflowScheduleCreateV2 engine.Workflow = "schedule_v2.create"

	// WorkflowScheduleAdjustV2 排班调整子工作流 V2
	WorkflowScheduleAdjustV2 engine.Workflow = "schedule_v2.adjust"
)

// ============================================================
// 班次类型常量
// 用于 Shift.Type 字段的标准化值
// ============================================================

const (
	// 后端工作流类型（优先级排序）
	ShiftTypeFixed    = "fixed"    // 固定班次（不需要AI排班）
	ShiftTypeSpecial  = "special"  // 特殊班次（有技能要求，优先排班）
	ShiftTypeNormal   = "normal"   // 普通班次（常规工作班次）
	ShiftTypeResearch = "research" // 科研班次（优先级较低）
	ShiftTypeFill     = "fill"     // 填充班次（年假、行政班等）
	ShiftTypeLeave    = "leave"    // 请假班次（属于填充类）

	// 前端使用的类型（兼容前端管理界面）
	ShiftTypeRegular  = "regular"  // 常规班次（前端）-> 映射到 normal
	ShiftTypeOvertime = "overtime" // 加班班次（前端）-> 映射到 special
	ShiftTypeStandby  = "standby"  // 备班班次（前端）-> 映射到 special
)

// ShiftTypeFrontendMapping 前端类型到工作流类型的映射
var ShiftTypeFrontendMapping = map[string]string{
	ShiftTypeRegular:  ShiftTypeNormal,  // 常规班次 -> 普通班次
	ShiftTypeOvertime: ShiftTypeSpecial, // 加班班次 -> 特殊班次（优先排班）
	ShiftTypeStandby:  ShiftTypeSpecial, // 备班班次 -> 特殊班次（优先排班）
}

// ============================================================
// schedule_v2.create 工作流 - 状态定义
// 命名规范: CreateV2State[StateName]
// ============================================================

const (
	// ========== 主流程状态 ==========

	// CreateV2StateInit 初始化状态
	CreateV2StateInit engine.State = "_schedule_v2_create_init_"

	// CreateV2StateInfoCollecting 信息收集阶段（调用 InfoCollect 子工作流）
	CreateV2StateInfoCollecting engine.State = "_schedule_v2_create_info_collecting_"

	// CreateV2StateConfirmPeriod 确认排班时间阶段
	CreateV2StateConfirmPeriod engine.State = "_schedule_v2_create_confirm_period_"

	// CreateV2StateConfirmShifts 确认班次选择阶段
	CreateV2StateConfirmShifts engine.State = "_schedule_v2_create_confirm_shifts_"

	// CreateV2StateConfirmStaffCount 确认人数配置阶段
	CreateV2StateConfirmStaffCount engine.State = "_schedule_v2_create_confirm_staff_count_"

	// CreateV2StatePersonalNeeds 个人需求收集与确认阶段
	CreateV2StatePersonalNeeds engine.State = "_schedule_v2_create_personal_needs_"

	// CreateV2StateFixedShift 固定班次处理阶段
	CreateV2StateFixedShift engine.State = "_schedule_v2_create_fixed_shift_"

	// CreateV2StateSpecialShift 特殊班次排班阶段（循环调用 Core 子工作流）
	CreateV2StateSpecialShift engine.State = "_schedule_v2_create_special_shift_"

	// CreateV2StateNormalShift 普通班次排班阶段（循环调用 Core 子工作流）
	CreateV2StateNormalShift engine.State = "_schedule_v2_create_normal_shift_"

	// CreateV2StateResearchShift 科研班次排班阶段（循环调用 Core 子工作流）
	CreateV2StateResearchShift engine.State = "_schedule_v2_create_research_shift_"

	// CreateV2StateFillShift 填充班次处理阶段
	CreateV2StateFillShift engine.State = "_schedule_v2_create_fill_shift_"

	// CreateV2StateShiftReview 班次审核状态（非连续排班模式下等待用户操作）
	CreateV2StateShiftReview engine.State = "_schedule_v2_create_shift_review_"

	// CreateV2StateWaitingAdjustment 等待用户输入调整需求状态
	CreateV2StateWaitingAdjustment engine.State = "_schedule_v2_create_waiting_adjustment_"

	// CreateV2StateShiftFailed 班次排班失败状态（等待用户决定是否继续）
	CreateV2StateShiftFailed engine.State = "_schedule_v2_create_shift_failed_"

	// CreateV2StateConfirmSaving 确认保存阶段
	CreateV2StateConfirmSaving engine.State = "_schedule_v2_create_confirm_saving_"

	// ========== 终态 ==========
	// 复用通用终态常量
	CreateV2StateCompleted = engine.StateCompleted // "_completed_"
	CreateV2StateFailed    = engine.StateFailed    // "_failed_"
	CreateV2StateCancelled = engine.StateCancelled // "_cancelled_"
)

// ============================================================
// schedule_v2.adjust 子工作流 - 状态定义
// 命名规范: AdjustV2State[StateName]
// ============================================================

const (
	// AdjustV2StateInit 初始化状态
	AdjustV2StateInit engine.State = "_schedule_v2_adjust_init_"

	// AdjustV2StateAnalyzingIntent 分析调整意图状态
	AdjustV2StateAnalyzingIntent engine.State = "_schedule_v2_adjust_analyzing_intent_"

	// AdjustV2StateRegenerating 重排班次状态（调用 core 子流程）
	AdjustV2StateRegenerating engine.State = "_schedule_v2_adjust_regenerating_"

	// AdjustV2StateModifying 修改排班状态（应用调整计划）
	AdjustV2StateModifying engine.State = "_schedule_v2_adjust_modifying_"

	// AdjustV2StateCompleted 完成状态
	AdjustV2StateCompleted engine.State = "_schedule_v2_adjust_completed_"

	// AdjustV2StateFailed 失败状态
	AdjustV2StateFailed engine.State = "_schedule_v2_adjust_failed_"
)

// ============================================================
// schedule_v2.adjust 子工作流 - 事件定义
// 命名规范: AdjustV2Event[EventName]
// ============================================================

const (
	// AdjustV2EventStart 启动事件
	AdjustV2EventStart engine.Event = engine.EventSubWorkflowStart // 复用子工作流标准启动事件

	// AdjustV2EventIntentAnalyzed 意图分析完成事件
	AdjustV2EventIntentAnalyzed engine.Event = "_schedule_v2_adjust_intent_analyzed_"

	// AdjustV2EventRegenerateComplete 重排完成事件
	AdjustV2EventRegenerateComplete engine.Event = "_schedule_v2_adjust_regenerate_complete_"

	// AdjustV2EventModifyComplete 修改完成事件
	AdjustV2EventModifyComplete engine.Event = "_schedule_v2_adjust_modify_complete_"

	// AdjustV2EventError 错误事件
	AdjustV2EventError engine.Event = "_schedule_v2_adjust_error_"
)

// ============================================================
// schedule_v2.create 工作流 - 事件定义
// 命名规范: CreateV2Event[EventName]
// ============================================================

const (
	// ========== 启动事件 ==========

	// CreateV2EventStart 启动工作流事件
	CreateV2EventStart engine.Event = engine.EventStart // 复用通用启动事件

	// ========== 阶段完成事件 ==========

	// CreateV2EventInfoCollected 信息收集完成
	CreateV2EventInfoCollected engine.Event = "_schedule_v2_create_info_collected_"

	// CreateV2EventPeriodConfirmed 排班时间确认完成
	CreateV2EventPeriodConfirmed engine.Event = "_schedule_v2_create_period_confirmed_"

	// CreateV2EventShiftsConfirmed 班次选择确认完成
	CreateV2EventShiftsConfirmed engine.Event = "_schedule_v2_create_shifts_confirmed_"

	// CreateV2EventStaffCountConfirmed 人数配置确认完成
	CreateV2EventStaffCountConfirmed engine.Event = "_schedule_v2_create_staff_count_confirmed_"

	// CreateV2EventPersonalNeedsConfirmed 个人需求确认完成
	CreateV2EventPersonalNeedsConfirmed engine.Event = "_schedule_v2_create_personal_needs_confirmed_"

	// CreateV2EventFixedShiftConfirmed 固定班次确认完成
	CreateV2EventFixedShiftConfirmed engine.Event = "_schedule_v2_create_fixed_shift_confirmed_"

	// CreateV2EventShiftPhaseComplete 某一类班次阶段完成（通用事件）
	CreateV2EventShiftPhaseComplete engine.Event = "_schedule_v2_create_shift_phase_complete_"

	// CreateV2EventSaveCompleted 保存完成
	CreateV2EventSaveCompleted engine.Event = "_schedule_v2_create_save_completed_"

	// ========== 子工作流事件 ==========

	// CreateV2EventSubCancelled 子工作流被取消
	CreateV2EventSubCancelled engine.Event = "_schedule_v2_create_sub_cancelled_"

	// CreateV2EventSubFailed 子工作流失败
	CreateV2EventSubFailed engine.Event = "_schedule_v2_create_sub_failed_"

	// CreateV2EventShiftCompleted 单个班次排班完成（Core 子工作流返回）
	CreateV2EventShiftCompleted engine.Event = "_schedule_v2_create_shift_completed_"

	// CreateV2EventEnterShiftReview 进入班次审核状态（非连续排班模式下，班次完成后进入审核）
	CreateV2EventEnterShiftReview engine.Event = "_schedule_v2_create_enter_shift_review_"

	// CreateV2EventShiftReviewConfirmed 班次审核确认继续
	CreateV2EventShiftReviewConfirmed engine.Event = "_schedule_v2_create_shift_review_confirmed_"

	// CreateV2EventShiftReviewAdjust 班次审核提出调整
	CreateV2EventShiftReviewAdjust engine.Event = "_schedule_v2_create_shift_review_adjust_"

	// CreateV2EventUserAdjustmentMessage 用户发送调整需求消息
	CreateV2EventUserAdjustmentMessage engine.Event = "_schedule_v2_create_user_adjustment_message_"

	// CreateV2EventShiftAdjusted 班次调整完成事件（Adjust 子工作流返回）
	CreateV2EventShiftAdjusted engine.Event = "_schedule_v2_create_shift_adjusted_"

	// CreateV2EventShiftFailed 班次排班失败事件（验证失败后触发）
	CreateV2EventShiftFailed engine.Event = "_schedule_v2_create_shift_failed_"

	// CreateV2EventShiftFailedContinue 班次失败后用户选择继续
	CreateV2EventShiftFailedContinue engine.Event = "_schedule_v2_create_shift_failed_continue_"

	// CreateV2EventShiftFailedCancel 班次失败后用户选择取消
	CreateV2EventShiftFailedCancel engine.Event = "_schedule_v2_create_shift_failed_cancel_"

	// ========== 用户操作事件 ==========

	// CreateV2EventUserCancel 用户取消操作
	CreateV2EventUserCancel engine.Event = engine.EventCancel // 复用通用取消事件

	// CreateV2EventUserConfirm 用户确认操作
	CreateV2EventUserConfirm engine.Event = engine.EventConfirm // 复用通用确认事件

	// CreateV2EventUserModify 用户修改操作
	CreateV2EventUserModify engine.Event = engine.EventModify // 复用通用修改事件

	// CreateV2EventModifyStaffCount 修改人数配置
	CreateV2EventModifyStaffCount engine.Event = "_schedule_v2_create_modify_staff_count_"

	// ========== 错误处理事件 ==========

	// CreateV2EventError 错误事件（通用）
	CreateV2EventError engine.Event = "_schedule_v2_create_error_"

	// CreateV2EventRetry 重试当前操作
	CreateV2EventRetry engine.Event = "_schedule_v2_create_retry_"

	// CreateV2EventSkipPhase 跳过当前阶段
	CreateV2EventSkipPhase engine.Event = "_schedule_v2_create_skip_phase_"
)

// ============================================================
// schedule_v2.create 工作流 - 状态描述（用于前端显示）
// ============================================================

var CreateV2StateDescriptions = map[engine.State]string{
	CreateV2StateInit:           "初始化排班",
	CreateV2StateInfoCollecting: "收集排班信息",
	CreateV2StateConfirmPeriod:  "确认排班时间",
	CreateV2StateConfirmShifts:      "确认班次选择",
	CreateV2StateConfirmStaffCount:  "确认人数配置",
	CreateV2StatePersonalNeeds:      "确认个人需求",
	CreateV2StateFixedShift:     "处理固定班次",
	CreateV2StateSpecialShift:   "排特殊班次",
	CreateV2StateNormalShift:    "排普通班次",
	CreateV2StateResearchShift:  "排科研班次",
	CreateV2StateFillShift:      "填充补充班次",
	CreateV2StateShiftReview:    "班次审核",
	CreateV2StateShiftFailed:    "班次排班失败",
	CreateV2StateConfirmSaving:  "确认并保存",
	CreateV2StateCompleted:      "排班完成",
	CreateV2StateFailed:         "排班失败",
	CreateV2StateCancelled:      "已取消",
}

// GetCreateV2StateDescription 获取状态的中文描述
func GetCreateV2StateDescription(state engine.State) string {
	if desc, ok := CreateV2StateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}

// ============================================================
// 阶段常量定义（用于上下文中标识当前处理阶段）
// ============================================================

const (
	PhaseInfoCollect   = "info_collect"   // 信息收集
	PhasePersonalNeed  = "personal_need"  // 个人需求
	PhaseFixedShift    = "fixed_shift"    // 固定班次
	PhaseSpecialShift  = "special_shift"  // 特殊班次
	PhaseNormalShift   = "normal_shift"   // 普通班次
	PhaseResearchShift = "research_shift" // 科研班次
	PhaseFillShift     = "fill_shift"     // 填充班次
	PhaseConfirmSave   = "confirm_save"   // 确认保存
)
