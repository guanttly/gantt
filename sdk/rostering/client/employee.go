package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jusha/agent/sdk/rostering/model"
	"jusha/agent/sdk/rostering/tool"
)

// updateEmployeePayload 更新员工的完整载荷（包含ID）
type updateEmployeePayload struct {
	ID           string     `json:"id"`
	OrgID        string     `json:"orgId"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone,omitempty"`
	Email        string     `json:"email,omitempty"`
	DepartmentID string     `json:"department"`
	Position     string     `json:"position,omitempty"`
	Role         string     `json:"role,omitempty"`
	Status       string     `json:"status,omitempty"`
	HireDate     *time.Time `json:"hireDate,omitempty"`
}

// Employee 员工管理实现

func (c *rosteringClient) CreateEmployee(ctx context.Context, req *model.CreateEmployeeRequest) (string, error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolEmployeeCreate.String(), req)
	if err != nil {
		return "", fmt.Errorf("create employee: %w", err)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("unmarshal employee create response: %w", err)
	}

	return response.ID, nil
}

func (c *rosteringClient) UpdateEmployee(ctx context.Context, id string, req *model.UpdateEmployeeRequest) error {
	payload := updateEmployeePayload{
		ID:           id,
		OrgID:        req.OrgID,
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		DepartmentID: req.DepartmentID,
		Position:     req.Position,
		Role:         req.Role,
		Status:       req.Status,
		HireDate:     req.HireDate,
	}

	_, err := c.toolBus.Execute(ctx, tool.ToolEmployeeUpdate.String(), payload)
	if err != nil {
		return fmt.Errorf("update employee: %w", err)
	}
	return nil
}

func (c *rosteringClient) ListEmployees(ctx context.Context, req *model.ListEmployeesRequest) (*model.Page[*model.Employee], error) {
	result, err := c.toolBus.Execute(ctx, tool.ToolEmployeeList.String(), req)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	var response model.Page[*model.Employee]
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("unmarshal employee list response: %w", err)
	}

	return &response, nil
}
