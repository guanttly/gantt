package service

import (
	"context"
	"fmt"
	"net/url"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type leaveService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newLeaveService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.ILeaveService {
	return &leaveService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *leaveService) Create(ctx context.Context, req *model.CreateLeaveRequest) (*model.Leave, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "POST", "/leaves", req)
	if err != nil {
		return nil, err
	}
	return parseLeave(resp.Data)
}

func (s *leaveService) GetList(ctx context.Context, req *model.ListLeavesRequest) (*model.ListLeavesResponse, error) {
	query := url.Values{}
	if req.OrgID != "" {
		query.Set("orgId", req.OrgID)
	}
	if req.EmployeeID != "" {
		query.Set("employeeId", req.EmployeeID)
	}
	if req.StartDate != "" {
		query.Set("startDate", req.StartDate)
	}
	if req.EndDate != "" {
		query.Set("endDate", req.EndDate)
	}
	if req.Status != "" {
		query.Set("status", req.Status)
	}
	if req.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", req.Page))
	}
	if req.PageSize > 0 {
		query.Set("size", fmt.Sprintf("%d", req.PageSize))
	}

	path := "/leaves"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	err, resp := s.managementRepo.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	result := &model.ListLeavesResponse{
		Leaves: []*model.Leave{},
	}

	if data, ok := resp.Data.(map[string]interface{}); ok {
		if leaves, ok := data["leaves"].([]interface{}); ok {
			for _, item := range leaves {
				if leave, err := parseLeave(item); err == nil {
					result.Leaves = append(result.Leaves, leave)
				}
			}
		}
		if total, ok := data["totalCount"].(float64); ok {
			result.TotalCount = int(total)
		}
	}

	return result, nil
}

func (s *leaveService) Get(ctx context.Context, id string) (*model.Leave, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "GET", fmt.Sprintf("/leaves/%s", id), nil)
	if err != nil {
		return nil, err
	}
	return parseLeave(resp.Data)
}

func (s *leaveService) Update(ctx context.Context, id string, req *model.UpdateLeaveRequest) (*model.Leave, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "PUT", fmt.Sprintf("/leaves/%s", id), req)
	if err != nil {
		return nil, err
	}
	return parseLeave(resp.Data)
}

func (s *leaveService) Delete(ctx context.Context, id string) error {
	err, _ := s.managementRepo.DoRequest(ctx, "DELETE", fmt.Sprintf("/leaves/%s", id), nil)
	return err
}

func (s *leaveService) GetBalance(ctx context.Context, employeeID string) ([]*model.LeaveBalance, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "GET", fmt.Sprintf("/leaves/balance?employeeId=%s", employeeID), nil)
	if err != nil {
		return nil, err
	}

	var balances []*model.LeaveBalance
	if data, ok := resp.Data.([]interface{}); ok {
		for _, item := range data {
			if balance, err := parseLeaveBalance(item); err == nil {
				balances = append(balances, balance)
			}
		}
	}

	return balances, nil
}

func parseLeave(data interface{}) (*model.Leave, error) {
	leaveData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid leave data format")
	}

	leave := &model.Leave{}
	if id, ok := leaveData["id"].(string); ok {
		leave.ID = id
	}
	if employeeID, ok := leaveData["employeeId"].(string); ok {
		leave.EmployeeID = employeeID
	}
	if startDate, ok := leaveData["startDate"].(string); ok {
		leave.StartDate = startDate
	}
	if endDate, ok := leaveData["endDate"].(string); ok {
		leave.EndDate = endDate
	}
	if leaveType, ok := leaveData["leaveType"].(string); ok {
		leave.LeaveType = leaveType
	}
	if status, ok := leaveData["status"].(string); ok {
		leave.Status = status
	}
	if reason, ok := leaveData["reason"].(string); ok {
		leave.Reason = reason
	}

	return leave, nil
}

func parseLeaveBalance(data interface{}) (*model.LeaveBalance, error) {
	balanceData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid leave balance data format")
	}

	balance := &model.LeaveBalance{}
	if leaveType, ok := balanceData["leaveType"].(string); ok {
		balance.LeaveType = leaveType
	}
	if total, ok := balanceData["total"].(float64); ok {
		balance.Total = total
	}
	if used, ok := balanceData["used"].(float64); ok {
		balance.Used = used
	}
	if remaining, ok := balanceData["remaining"].(float64); ok {
		balance.Remaining = remaining
	}

	return balance, nil
}
