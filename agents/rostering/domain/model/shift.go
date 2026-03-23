package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的班次类型
type Shift = sdk_model.Shift
type CreateShiftRequest = sdk_model.CreateShiftRequest
type UpdateShiftRequest = sdk_model.UpdateShiftRequest
type ListShiftsRequest = sdk_model.ListShiftsRequest
type SetShiftGroupsRequest = sdk_model.SetShiftGroupsRequest
type AddShiftGroupRequest = sdk_model.AddShiftGroupRequest
type ShiftGroup = sdk_model.ShiftGroup

// 人数配置相关类型
type WeekdayStaffConfig = sdk_model.WeekdayStaffConfig
type ShiftWeeklyStaffConfig = sdk_model.ShiftWeeklyStaffConfig
type DailyStaffingResult = sdk_model.DailyStaffingResult
type StaffingCalculationPreview = sdk_model.StaffingCalculationPreview
