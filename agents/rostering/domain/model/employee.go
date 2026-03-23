package model

import (
	sdk_model "jusha/agent/sdk/rostering/model"
)

// 直接使用 SDK model 的员工类型
type Employee = sdk_model.Employee
type CreateEmployeeRequest = sdk_model.CreateEmployeeRequest
type UpdateEmployeeRequest = sdk_model.UpdateEmployeeRequest
type ListEmployeesRequest = sdk_model.ListEmployeesRequest

// 向后兼容的别名（Staff -> Employee）
type Staff = sdk_model.Employee
type StaffCreateRequest = sdk_model.CreateEmployeeRequest
type StaffListFilter = sdk_model.ListEmployeesRequest
type StaffSearchFilter = sdk_model.ListEmployeesRequest
type StaffListResult = sdk_model.Page[*sdk_model.Employee]
