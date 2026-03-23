package service

import (
	"fmt"
	"jusha/gantt/mcp/rostering/domain/model"
)

// parseEmployee 解析员工响应
func parseEmployee(data interface{}) (*model.Employee, error) {
	empData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid employee data format")
	}

	emp := &model.Employee{}
	if id, ok := empData["id"].(string); ok {
		emp.ID = id
	}
	if empID, ok := empData["employeeId"].(string); ok {
		emp.EmployeeID = empID
	}
	if name, ok := empData["name"].(string); ok {
		emp.Name = name
	}
	if deptID, ok := empData["departmentId"].(string); ok {
		emp.DepartmentID = deptID
	}
	if orgID, ok := empData["orgId"].(string); ok {
		emp.OrgID = orgID
	}

	return emp, nil
}
