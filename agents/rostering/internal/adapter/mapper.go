package adapter

import (
	"context"
	"jusha/agent/rostering/domain/service"
	"jusha/agent/rostering/internal/workflow/state/schedule"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/workflow/wsbridge"
)

// WorkflowInitializer 工作流初始化器
// 实现 session.IWorkflowInitializer 接口
type WorkflowInitializer struct {
	rosteringService service.IRosteringService
	logger           logging.ILogger
}

// NewWorkflowInitializer 创建工作流初始化器
func NewWorkflowInitializer(rosteringService service.IRosteringService) session.IWorkflowInitializer {
	return &WorkflowInitializer{
		rosteringService: rosteringService,
		logger:           nil, // 不使用日志，如果需要可以注入
	}
}

// InitializeWorkflow 根据 agentType 初始化工作流（旧接口，保持向后兼容）
// orgID 和 userID 用于获取用户特定的工作流偏好（如版本选择）
func (w *WorkflowInitializer) InitializeWorkflow(agentType, orgID, userID string) string {
	result, _ := w.InitializeWorkflowWithVersion(agentType, orgID, userID)
	return result.WorkflowName
}

// InitializeWorkflowWithVersion 根据 agentType 初始化工作流并返回版本信息
func (w *WorkflowInitializer) InitializeWorkflowWithVersion(agentType, orgID, userID string) (session.WorkflowInitResult, bool) {
	switch agentType {
	case "rostering":
		// 获取用户工作流版本偏好
		if w.rosteringService != nil && orgID != "" && userID != "" {
			version, err := w.rosteringService.GetUserWorkflowVersion(context.Background(), orgID, userID)
			if err != nil {
				// 如果获取失败，使用默认版本 v2
				if w.logger != nil {
					w.logger.Warn("Failed to get user workflow version, using default v2", "orgID", orgID, "userID", userID, "error", err)
				}
				return session.WorkflowInitResult{
					WorkflowName: string(schedule.WorkflowScheduleCreateV2),
					Version:      "v2",
				}, true
			}
			// 根据版本返回对应工作流
			switch version {
			case "v4":
				return session.WorkflowInitResult{
					WorkflowName: string(schedule.WorkflowScheduleCreateV4),
					Version:      "v4",
				}, true
			case "v3":
				return session.WorkflowInitResult{
					WorkflowName: string(schedule.WorkflowScheduleCreateV3),
					Version:      "v3",
				}, true
			case "v2":
				return session.WorkflowInitResult{
					WorkflowName: string(schedule.WorkflowScheduleCreateV2),
					Version:      "v2",
				}, true
			default:
				// 默认使用 v2
				return session.WorkflowInitResult{
					WorkflowName: string(schedule.WorkflowScheduleCreateV2),
					Version:      "v2",
				}, true
			}
		}
		// 如果没有服务或用户信息，使用默认版本 v2
		return session.WorkflowInitResult{
			WorkflowName: string(schedule.WorkflowScheduleCreateV2),
			Version:      "v2",
		}, true
	default:
		return session.WorkflowInitResult{}, false
	}
}

// CommandMapper 命令映射器
// 实现 wsbridge.ICommandMapper 接口
type CommandMapper struct{}

// NewCommandMapper 创建命令映射器
func NewCommandMapper() wsbridge.ICommandMapper {
	return &CommandMapper{}
}

// MapCommandToEvent 将前端命令映射为工作流事件
func (m *CommandMapper) MapCommandToEvent(command string) engine.Event {
	// 前端命令到后端事件的映射
	// 注意：前端使用完整的事件名称（如 "_schedule_create_period_confirmed_"）
	commandEventMap := map[string]engine.Event{
		// 工作流启动事件
		"_start_": engine.EventStart,

		// 通用事件（适用于所有工作流）
		// 注意：这些事件可能被特定工作流的事件别名覆盖（如 CreateV2EventUserCancel = engine.EventCancel）
		string(engine.EventConfirm): engine.EventConfirm, // "_confirm_"
		string(engine.EventModify):  engine.EventModify,  // "_modify_"
		string(engine.EventCancel):  engine.EventCancel,  // "_cancel_"
		string(engine.EventReject):  engine.EventReject,  // "_reject_"

		// schedule.create 工作流事件（与前端 TypeScript WorkFlowEventType 保持一致）
		string(schedule.EventPeriodConfirmed):     schedule.EventPeriodConfirmed,
		string(schedule.EventPeriodModified):      schedule.EventPeriodModified,
		string(schedule.EventShiftsConfirmed):     schedule.EventShiftsConfirmed,
		string(schedule.EventShiftsModified):      schedule.EventShiftsModified,
		string(schedule.EventStaffCountConfirmed): schedule.EventStaffCountConfirmed,
		string(schedule.EventStaffCountModified):  schedule.EventStaffCountModified,
		string(schedule.EventDraftConfirmed):      schedule.EventDraftConfirmed,
		string(schedule.EventDraftRejected):       schedule.EventDraftRejected,
		string(schedule.EventUserCancelled):       schedule.EventUserCancelled,

		// schedule.info-collect 子工作流事件（用户触发）
		string(schedule.InfoCollectEventPeriodConfirmed):     schedule.InfoCollectEventPeriodConfirmed,
		string(schedule.InfoCollectEventPeriodModified):      schedule.InfoCollectEventPeriodModified,
		string(schedule.InfoCollectEventShiftsConfirmed):     schedule.InfoCollectEventShiftsConfirmed,
		string(schedule.InfoCollectEventShiftsModified):      schedule.InfoCollectEventShiftsModified,
		string(schedule.InfoCollectEventStaffCountConfirmed): schedule.InfoCollectEventStaffCountConfirmed,
		string(schedule.InfoCollectEventStaffCountModified):  schedule.InfoCollectEventStaffCountModified,
		string(schedule.InfoCollectEventCancel):              schedule.InfoCollectEventCancel,

		// schedule.collect-staff-count 子工作流事件（用户触发）
		string(schedule.CollectStaffCountEventConfirmed): schedule.CollectStaffCountEventConfirmed,
		string(schedule.CollectStaffCountEventModified):  schedule.CollectStaffCountEventModified,
		string(schedule.CollectStaffCountEventCancel):    schedule.CollectStaffCountEventCancel,

		// schedule.confirm-save 子工作流事件（用户触发）
		string(schedule.ConfirmSaveEventConfirm): schedule.ConfirmSaveEventConfirm,
		string(schedule.ConfirmSaveEventReject):  schedule.ConfirmSaveEventReject,
		string(schedule.ConfirmSaveEventCancel):  schedule.ConfirmSaveEventCancel,

		// schedule.adjust 工作流事件（用户触发）
		string(schedule.EventAdjustDateRangeSelected): schedule.EventAdjustDateRangeSelected,
		string(schedule.EventAdjustShiftSelected):     schedule.EventAdjustShiftSelected,
		string(schedule.EventAdjustShiftChanged):      schedule.EventAdjustShiftChanged,
		string(schedule.EventAdjustIntentSubmitted):   schedule.EventAdjustIntentSubmitted,
		string(schedule.EventAdjustQuickSwap):         schedule.EventAdjustQuickSwap,
		string(schedule.EventAdjustQuickReplace):      schedule.EventAdjustQuickReplace,
		string(schedule.EventAdjustQuickAdd):          schedule.EventAdjustQuickAdd,
		string(schedule.EventAdjustQuickRemove):       schedule.EventAdjustQuickRemove,
		string(schedule.EventAdjustBackToIntent):      schedule.EventAdjustBackToIntent,
		string(schedule.EventAdjustFinish):            schedule.EventAdjustFinish,
		string(schedule.EventAdjustPlanConfirmed):     schedule.EventAdjustPlanConfirmed,
		string(schedule.EventAdjustPlanRejected):      schedule.EventAdjustPlanRejected,
		string(schedule.EventAdjustPlanModified):      schedule.EventAdjustPlanModified,
		string(schedule.EventAdjustUndo):              schedule.EventAdjustUndo,
		string(schedule.EventAdjustRedo):              schedule.EventAdjustRedo,
		string(schedule.EventAdjustContinueAdjust):    schedule.EventAdjustContinueAdjust,
		string(schedule.EventAdjustUserCancelled):     schedule.EventAdjustUserCancelled,

		// schedule.adjust 重排班次相关事件
		string(schedule.EventAdjustRegenerateStart):     schedule.EventAdjustRegenerateStart,
		string(schedule.EventAdjustStaffCountConfirmed): schedule.EventAdjustStaffCountConfirmed,
		string(schedule.EventAdjustRegenerateAborted):   schedule.EventAdjustRegenerateAborted,

		// schedule_v2.create 工作流事件（用户触发）
		// 注意：CreateV2EventUserCancel 和 CreateV2EventUserModify 是通用事件的别名，
		// 它们的字符串值就是 "_cancel_" 和 "_modify_"，已在上面通用事件中映射
		string(schedule.CreateV2EventPeriodConfirmed):        schedule.CreateV2EventPeriodConfirmed,
		string(schedule.CreateV2EventShiftsConfirmed):        schedule.CreateV2EventShiftsConfirmed,
		string(schedule.CreateV2EventStaffCountConfirmed):    schedule.CreateV2EventStaffCountConfirmed,
		string(schedule.CreateV2EventModifyStaffCount):       schedule.CreateV2EventModifyStaffCount,
		string(schedule.CreateV2EventPersonalNeedsConfirmed): schedule.CreateV2EventPersonalNeedsConfirmed,
		string(schedule.CreateV2EventFixedShiftConfirmed):    schedule.CreateV2EventFixedShiftConfirmed,
		string(schedule.CreateV2EventSaveCompleted):          schedule.CreateV2EventSaveCompleted,
		string(schedule.CreateV2EventSkipPhase):              schedule.CreateV2EventSkipPhase,
		string(schedule.CreateV2EventShiftReviewConfirmed):   schedule.CreateV2EventShiftReviewConfirmed,
		string(schedule.CreateV2EventShiftReviewAdjust):      schedule.CreateV2EventShiftReviewAdjust,
		string(schedule.CreateV2EventUserAdjustmentMessage):  schedule.CreateV2EventUserAdjustmentMessage,
		string(schedule.CreateV2EventShiftFailedContinue):    schedule.CreateV2EventShiftFailedContinue,
		string(schedule.CreateV2EventShiftFailedCancel):      schedule.CreateV2EventShiftFailedCancel,

		// schedule_v3.create 工作流事件（用户触发）
		// 注意：CreateV3EventUserCancel、CreateV3EventUserConfirm 和 CreateV3EventUserModify 是通用事件的别名，
		// 它们的字符串值就是 "_cancel_"、"_confirm_" 和 "_modify_"，已在上面通用事件中映射
		string(schedule.CreateV3EventPeriodConfirmed):             schedule.CreateV3EventPeriodConfirmed,
		string(schedule.CreateV3EventShiftsConfirmed):             schedule.CreateV3EventShiftsConfirmed,
		string(schedule.CreateV3EventStaffCountConfirmed):         schedule.CreateV3EventStaffCountConfirmed,
		string(schedule.CreateV3EventModifyStaffCount):            schedule.CreateV3EventModifyStaffCount,
		string(schedule.CreateV3EventPersonalNeedsConfirmed):      schedule.CreateV3EventPersonalNeedsConfirmed,
		string(schedule.CreateV3EventTemporaryNeedsTextSubmitted): schedule.CreateV3EventTemporaryNeedsTextSubmitted,
		string(schedule.CreateV3EventRequirementAssessed):         schedule.CreateV3EventRequirementAssessed,
		string(schedule.CreateV3EventPlanConfirmed):               schedule.CreateV3EventPlanConfirmed,
		string(schedule.CreateV3EventPlanAdjust):                  schedule.CreateV3EventPlanAdjust,
		string(schedule.CreateV3EventPlanAdjusted):                schedule.CreateV3EventPlanAdjusted,
		string(schedule.CreateV3EventSaveCompleted):               schedule.CreateV3EventSaveCompleted,
		string(schedule.CreateV3EventSkipPhase):                   schedule.CreateV3EventSkipPhase,
		string(schedule.CreateV3EventTaskReviewConfirmed):         schedule.CreateV3EventTaskReviewConfirmed,
		string(schedule.CreateV3EventTaskReviewAdjust):            schedule.CreateV3EventTaskReviewAdjust,
		string(schedule.CreateV3EventUserAdjustmentMessage):       schedule.CreateV3EventUserAdjustmentMessage,
		string(schedule.CreateV3EventTaskFailedContinue):          schedule.CreateV3EventTaskFailedContinue,
		string(schedule.CreateV3EventTaskFailedCancel):            schedule.CreateV3EventTaskFailedCancel,
		string(schedule.CreateV3EventRetry):                       schedule.CreateV3EventRetry,

		// schedule_v3.create 全局评审相关事件（用户触发）
		string(schedule.CreateV3EventGlobalReviewManualConfirmed): schedule.CreateV3EventGlobalReviewManualConfirmed,
		string(schedule.CreateV3EventGlobalReviewSkip):            schedule.CreateV3EventGlobalReviewSkip,

		// schedule_v4.create 工作流事件（用户触发）
		// 注意：CreateV4EventReviewConfirmed、CreateV4EventUserModify、CreateV4EventUserCancel
		// 复用了通用事件（_confirm_、_modify_、_cancel_），已在上方通用事件中映射
		string(schedule.CreateV4EventPeriodConfirmed):             schedule.CreateV4EventPeriodConfirmed,
		string(schedule.CreateV4EventShiftsConfirmed):             schedule.CreateV4EventShiftsConfirmed,
		string(schedule.CreateV4EventStaffCountConfirmed):         schedule.CreateV4EventStaffCountConfirmed,
		string(schedule.CreateV4EventPersonalNeedsConfirmed):      schedule.CreateV4EventPersonalNeedsConfirmed,
		string(schedule.CreateV4EventTemporaryNeedsTextSubmitted): schedule.CreateV4EventTemporaryNeedsTextSubmitted,
	}

	if event, ok := commandEventMap[command]; ok {
		return event
	}

	return ""
}
