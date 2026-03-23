// Package schedule 定义确认保存子工作流的状态和事件
package schedule

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// 确认保存子工作流定义
// 用于草案预览、确认和保存
// ============================================================

const (
	// WorkflowConfirmSave 确认保存子工作流
	WorkflowConfirmSave engine.Workflow = "schedule.confirm-save"
)

// ============================================================
// schedule.confirm-save 子工作流 - 状态定义
// ============================================================

const (
	// 阶段1: 预览草案
	ConfirmSaveStatePreview engine.State = "_confirm_save_preview_"

	// 阶段2: 确认中
	ConfirmSaveStateConfirming engine.State = "_confirm_save_confirming_"

	// 阶段3: 保存中
	ConfirmSaveStateSaving engine.State = "_confirm_save_saving_"

	// 终态
	ConfirmSaveStateCompleted engine.State = "_confirm_save_completed_"
	ConfirmSaveStateCancelled engine.State = "_confirm_save_cancelled_"
	ConfirmSaveStateFailed    engine.State = "_confirm_save_failed_"
)

// ============================================================
// schedule.confirm-save 子工作流 - 事件定义
// ============================================================

const (
	// 启动事件
	ConfirmSaveEventStart engine.Event = "_confirm_save_start_"

	// 预览事件
	ConfirmSaveEventPreviewReady engine.Event = "_confirm_save_preview_ready_"

	// 确认事件
	ConfirmSaveEventConfirm engine.Event = "_confirm_save_confirm_"
	ConfirmSaveEventReject  engine.Event = "_confirm_save_reject_"
	ConfirmSaveEventModify  engine.Event = "_confirm_save_modify_"

	// 保存事件
	ConfirmSaveEventSaveSuccess engine.Event = "_confirm_save_save_success_"
	ConfirmSaveEventSaveFailed  engine.Event = "_confirm_save_save_failed_"
	ConfirmSaveEventRetry       engine.Event = "_confirm_save_retry_"

	// 完成和取消事件
	ConfirmSaveEventComplete engine.Event = "_confirm_save_complete_"
	ConfirmSaveEventCancel   engine.Event = "_confirm_save_cancel_"
	ConfirmSaveEventError    engine.Event = "_confirm_save_error_"
	ConfirmSaveEventReturn   engine.Event = "_confirm_save_return_"
)

// ============================================================
// schedule.confirm-save 子工作流 - 输入输出结构
// ============================================================

// ConfirmSaveInput 确认保存子工作流输入
type ConfirmSaveInput struct {
	// SourceType 来源类型: "create" | "adjust"
	SourceType string `json:"source_type"`

	// 排班周期
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`

	// 班次排班结果（shift_id -> 排班草案）
	ShiftResults map[string]any `json:"shift_results"`

	// 统计信息
	TotalShifts  int `json:"total_shifts"`
	SkippedCount int `json:"skipped_count"`

	// 草案数据键名（可选，从 session.Data 中读取）
	DraftDataKey string `json:"draft_data_key,omitempty"`
}

// ConfirmSaveOutput 确认保存子工作流输出
type ConfirmSaveOutput struct {
	// 是否保存成功
	Success bool `json:"success"`

	// 保存后的排班 ID
	ScheduleID string `json:"schedule_id,omitempty"`

	// 保存的记录数
	SavedCount int `json:"saved_count"`

	// 失败的记录数
	FailedCount int `json:"failed_count"`

	// 错误信息（如果有）
	ErrorMessage string `json:"error_message,omitempty"`
}

// ============================================================
// schedule.confirm-save 子工作流 - 状态描述
// ============================================================

var ConfirmSaveStateDescriptions = map[engine.State]string{
	ConfirmSaveStatePreview:    "预览排班草案",
	ConfirmSaveStateConfirming: "确认排班草案",
	ConfirmSaveStateSaving:     "保存排班",
	ConfirmSaveStateCompleted:  "保存完成",
	ConfirmSaveStateCancelled:  "已取消",
	ConfirmSaveStateFailed:     "保存失败",
}

// GetConfirmSaveStateDescription 获取确认保存子工作流状态描述
func GetConfirmSaveStateDescription(state engine.State) string {
	if desc, ok := ConfirmSaveStateDescriptions[state]; ok {
		return desc
	}
	return string(state)
}
