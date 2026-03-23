package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的请假类型
type Leave = sdk_model.Leave
type CreateLeaveRequest = sdk_model.CreateLeaveRequest
type UpdateLeaveRequest = sdk_model.UpdateLeaveRequest
type ListLeavesRequest = sdk_model.ListLeavesRequest
type ListLeavesResponse = sdk_model.ListLeavesResponse

// 向后兼容的别名
type LeaveRecord = sdk_model.Leave
