package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的班次类型相关类型
type ShiftType = sdk_model.ShiftType
type CreateShiftTypeRequest = sdk_model.CreateShiftTypeRequest
type UpdateShiftTypeRequest = sdk_model.UpdateShiftTypeRequest
type ListShiftTypesRequest = sdk_model.ListShiftTypesRequest
type ShiftTypeStats = sdk_model.ShiftTypeStats
type WorkflowPhaseInfo = sdk_model.WorkflowPhaseInfo

// GetWorkflowPhases 获取所有工作流阶段定义
func GetWorkflowPhases() []WorkflowPhaseInfo {
	return sdk_model.GetWorkflowPhases()
}

