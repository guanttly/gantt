enum WorkFlowEventType {
  // 通用行为
  Action_UserConfirm = '_event_action_user_confirm_', // 用户确认
  Action_UserSupplementary = '_event_action_user_supplementary', // 用户补充信息
  Action_AutoConfirm = '_event_action_auto_confirm_', // 系统自动确认
  Action_Cancel = '_event_action_cancel_', // 用户取消

  // 科室相关
  Dept_StaffUpdate = '_event_dept_staff_update_', // 科室人员更新

  // 排班创建工作流事件（与后端保持一致）
  Schedule_Create_Start = '_start_', // 启动排班创建流程
  Schedule_Create_PeriodConfirmed = '_schedule_create_period_confirmed_', // 确认排班周期
  Schedule_Create_PeriodModified = '_schedule_create_period_modified_', // 修改排班周期
  Schedule_Create_ShiftsConfirmed = '_schedule_create_shifts_confirmed_', // 确认班次列表
  Schedule_Create_ShiftsModified = '_schedule_create_shifts_modified_', // 修改班次列表
  Schedule_Create_StaffCountConfirmed = '_schedule_create_staff_count_confirmed_', // 确认人数需求
  Schedule_Create_StaffCountModified = '_schedule_create_staff_count_modified_', // 修改人数需求
  Schedule_Create_DraftConfirmed = '_schedule_create_draft_confirmed_', // 确认排班草案
  Schedule_Create_DraftRejected = '_schedule_create_draft_rejected_', // 拒绝排班草案
  Schedule_Create_UserCancelled = '_schedule_create_user_cancelled_', // 用户取消流程

  // 排班查询相关
  Schedule_Create_PreviewSchedule = '_event_schedule_create_preview_schedule_', // 预览排班初稿

  // V3 工作流事件（与后端保持一致）
  Schedule_V3_Create_Start = '_start_', // 启动V3排班创建流程
  Schedule_V3_Create_InfoCollected = '_schedule_v3_create_info_collected_', // 信息收集完成
  Schedule_V3_Create_PeriodConfirmed = '_schedule_v3_create_period_confirmed_', // 确认排班周期
  Schedule_V3_Create_ShiftsConfirmed = '_schedule_v3_create_shifts_confirmed_', // 确认班次列表
  Schedule_V3_Create_StaffCountConfirmed = '_schedule_v3_create_staff_count_confirmed_', // 确认人数需求
  Schedule_V3_Create_ModifyStaffCount = '_schedule_v3_create_modify_staff_count_', // 修改人数配置
  Schedule_V3_Create_PersonalNeedsConfirmed = '_schedule_v3_create_personal_needs_confirmed_', // 确认个人需求
  Schedule_V3_Create_TemporaryNeedsTextSubmitted = '_schedule_v3_create_temporary_needs_text_submitted_', // 临时需求文本提交
  Schedule_V3_Create_RequirementAssessed = '_schedule_v3_create_requirement_assessed_', // 需求评估完成
  Schedule_V3_Create_PlanConfirmed = '_schedule_v3_create_plan_confirmed_', // 确认任务计划
  Schedule_V3_Create_PlanAdjust = '_schedule_v3_create_plan_adjust_', // 调整任务计划
  Schedule_V3_Create_PlanAdjusted = '_schedule_v3_create_plan_adjusted_', // 任务计划调整完成
  Schedule_V3_Create_TaskCompleted = '_schedule_v3_create_task_completed_', // 单个任务执行完成
  Schedule_V3_Create_AllTasksComplete = '_schedule_v3_create_all_tasks_complete_', // 所有任务完成
  Schedule_V3_Create_EnterTaskReview = '_schedule_v3_create_enter_task_review_', // 进入任务审核
  Schedule_V3_Create_TaskReviewConfirmed = '_schedule_v3_create_task_review_confirmed_', // 任务审核确认
  Schedule_V3_Create_TaskReviewAdjust = '_schedule_v3_create_task_review_adjust_', // 任务审核提出调整
  Schedule_V3_Create_UserAdjustmentMessage = '_schedule_v3_create_user_adjustment_message_', // 用户发送调整需求消息
  Schedule_V3_Create_TaskAdjusted = '_schedule_v3_create_task_adjusted_', // 任务调整完成
  Schedule_V3_Create_TaskFailed = '_schedule_v3_create_task_failed_', // 任务执行失败
  Schedule_V3_Create_TaskFailedContinue = '_schedule_v3_create_task_failed_continue_', // 任务失败后继续
  Schedule_V3_Create_TaskFailedCancel = '_schedule_v3_create_task_failed_cancel_', // 任务失败后取消
  Schedule_V3_Create_Retry = '_schedule_v3_create_retry_', // 重试当前操作
  Schedule_V3_Create_SkipPhase = '_schedule_v3_create_skip_phase_', // 跳过当前阶段
  Schedule_V3_Create_SaveCompleted = '_schedule_v3_create_save_completed_', // 保存完成
  Schedule_V3_Create_SubCancelled = '_schedule_v3_create_sub_cancelled_', // 子工作流被取消
  Schedule_V3_Create_SubFailed = '_schedule_v3_create_sub_failed_', // 子工作流失败

  // V3 Core 子工作流 - 部分成功处理事件
  Schedule_V3_Core_RetryFailed = '_schedule_v3_core_retry_failed_', // 重试失败的班次
  Schedule_V3_Core_SkipFailed = '_schedule_v3_core_skip_failed_', // 跳过失败的班次，保存成功部分
  Schedule_V3_Core_CancelTask = '_schedule_v3_core_cancel_task_', // 取消任务
}

// ActionType 定义操作类型常量，与后端保持一致
export enum WorkFlowActionType {
  Workflow = '_type_action_workflow_', // 触发状态机转换的工作流操作
  Query = '_type_action_query_', // 查询操作，不改变状态
}

// ActionStyle 定义操作按钮样式，与后端保持一致
export enum WorkFlowActionStyle {
  ButtonDefault = '_style_action_button_default_',
  ButtonPrimary = '_style_action_button_primary_',
  ButtonDanger = '_style_action_button_danger_',
  Additional = '_style_action_additional_', // 不显示按钮，关联用户的下次输入
}

export default WorkFlowEventType
