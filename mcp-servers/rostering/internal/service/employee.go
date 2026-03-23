package service

import (
	"context"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type employeeService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newEmployeeService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IEmployeeService {
	return &employeeService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

// Create 创建员工
func (s *employeeService) Create(ctx context.Context, req *model.CreateEmployeeRequest) (*model.Employee, error) {
	return s.managementRepo.CreateEmployee(ctx, req)
}

// GetList 获取员工列表
func (s *employeeService) GetList(ctx context.Context, req *model.ListEmployeesRequest) (*model.ListEmployeesResponse, error) {
	pageData, err := s.managementRepo.ListEmployees(ctx, req)
	if err != nil {
		return nil, err
	}

	return &model.ListEmployeesResponse{
		Employees:  pageData.Items,
		TotalCount: int(pageData.Total),
	}, nil
}

// Get 获取单个员工
func (s *employeeService) Get(ctx context.Context, id string) (*model.Employee, error) {
	return s.managementRepo.GetEmployee(ctx, id)
}

// Update 更新员工
func (s *employeeService) Update(ctx context.Context, id string, req *model.UpdateEmployeeRequest) (*model.Employee, error) {
	return s.managementRepo.UpdateEmployee(ctx, id, req)
}

// Delete 删除员工
func (s *employeeService) Delete(ctx context.Context, id string) error {
	return s.managementRepo.DeleteEmployee(ctx, id)
}
