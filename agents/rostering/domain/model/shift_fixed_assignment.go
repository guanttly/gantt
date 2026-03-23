package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的固定人员配置相关类型
type PatternType = sdk_model.PatternType

const (
	PatternTypeWeekly   = sdk_model.PatternTypeWeekly
	PatternTypeMonthly  = sdk_model.PatternTypeMonthly
	PatternTypeSpecific = sdk_model.PatternTypeSpecific
)

type WeekPattern = sdk_model.WeekPattern

const (
	WeekPatternEvery = sdk_model.WeekPatternEvery
	WeekPatternOdd   = sdk_model.WeekPatternOdd
	WeekPatternEven  = sdk_model.WeekPatternEven
)

type ShiftFixedAssignment = sdk_model.ShiftFixedAssignment
type CreateShiftFixedAssignmentRequest = sdk_model.CreateShiftFixedAssignmentRequest
type UpdateShiftFixedAssignmentRequest = sdk_model.UpdateShiftFixedAssignmentRequest
type BatchCreateShiftFixedAssignmentsRequest = sdk_model.BatchCreateShiftFixedAssignmentsRequest
type ListShiftFixedAssignmentsRequest = sdk_model.ListShiftFixedAssignmentsRequest
type ShiftFixedAssignmentWithStaff = sdk_model.ShiftFixedAssignmentWithStaff
type CalculatedScheduleDate = sdk_model.CalculatedScheduleDate
type CalculatedScheduleResult = sdk_model.CalculatedScheduleResult

