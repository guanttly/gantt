package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的部门类型
type Department = sdk_model.Department
type CreateDepartmentRequest = sdk_model.CreateDepartmentRequest
type UpdateDepartmentRequest = sdk_model.UpdateDepartmentRequest
type ListDepartmentsResponse = sdk_model.ListDepartmentsResponse
