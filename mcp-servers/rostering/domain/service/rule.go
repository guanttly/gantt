package service

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

// IRuleService 规则服务接口
type IRuleService interface {
	Create(ctx context.Context, req *model.CreateRuleRequest) (*model.Rule, error)
	GetList(ctx context.Context, req *model.ListRulesRequest) (*model.ListRulesResponse, error)
	Get(ctx context.Context, id string) (*model.Rule, error)
	Update(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.Rule, error)
	Delete(ctx context.Context, id string) error
	GetForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.Rule, error)
	GetForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error)
	GetForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)
	// 批量查询
	GetForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error)
	GetForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error)
	GetForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error)

	// V4.1 新增方法
	CreateWithRelations(ctx context.Context, req *model.CreateRuleWithRelationsRequest) (*model.Rule, error)
	UpdateWithRelations(ctx context.Context, id string, req *model.UpdateRuleWithRelationsRequest) (*model.Rule, error)
	GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)
	GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)
}
