package service

import (
	"context"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type ruleService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newRuleService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IRuleService {
	return &ruleService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *ruleService) Create(ctx context.Context, req *model.CreateRuleRequest) (*model.Rule, error) {
	return s.managementRepo.CreateRule(ctx, req)
}

func (s *ruleService) GetList(ctx context.Context, req *model.ListRulesRequest) (*model.ListRulesResponse, error) {
	pageData, err := s.managementRepo.ListRules(ctx, req)
	if err != nil {
		return nil, err
	}

	return &model.ListRulesResponse{
		Rules:      pageData.Items,
		TotalCount: int(pageData.Total),
	}, nil
}

func (s *ruleService) Get(ctx context.Context, id string) (*model.Rule, error) {
	return s.managementRepo.GetRule(ctx, id)
}

func (s *ruleService) Update(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.Rule, error) {
	return s.managementRepo.UpdateRule(ctx, id, req)
}

func (s *ruleService) Delete(ctx context.Context, id string) error {
	return s.managementRepo.DeleteRule(ctx, id)
}

func (s *ruleService) GetForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.Rule, error) {
	return s.managementRepo.GetRulesForEmployee(ctx, orgID, employeeID)
}

func (s *ruleService) GetForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error) {
	return s.managementRepo.GetRulesForGroup(ctx, orgID, groupID)
}

func (s *ruleService) GetForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	return s.managementRepo.GetRulesForShift(ctx, orgID, shiftID)
}

func (s *ruleService) GetForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error) {
	return s.managementRepo.GetRulesForEmployees(ctx, orgID, employeeIDs)
}

func (s *ruleService) GetForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error) {
	return s.managementRepo.GetRulesForShifts(ctx, orgID, shiftIDs)
}

func (s *ruleService) GetForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error) {
	return s.managementRepo.GetRulesForGroups(ctx, orgID, groupIDs)
}

// V4.1 新增方法

func (s *ruleService) CreateWithRelations(ctx context.Context, req *model.CreateRuleWithRelationsRequest) (*model.Rule, error) {
	return s.managementRepo.CreateRuleWithRelations(ctx, req)
}

func (s *ruleService) UpdateWithRelations(ctx context.Context, id string, req *model.UpdateRuleWithRelationsRequest) (*model.Rule, error) {
	return s.managementRepo.UpdateRuleWithRelations(ctx, id, req)
}

func (s *ruleService) GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	return s.managementRepo.GetRulesBySubjectShift(ctx, orgID, shiftID)
}

func (s *ruleService) GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	return s.managementRepo.GetRulesByObjectShift(ctx, orgID, shiftID)
}
