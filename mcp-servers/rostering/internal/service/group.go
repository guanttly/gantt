package service

import (
	"context"
	"fmt"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type groupService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newGroupService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IGroupService {
	return &groupService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

func (s *groupService) Create(ctx context.Context, req *model.CreateGroupRequest) (*model.Group, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "POST", "/groups", req)
	if err != nil {
		return nil, err
	}
	return parseGroup(resp.Data)
}

func (s *groupService) GetList(ctx context.Context, req *model.ListGroupsRequest) (*model.ListGroupsResponse, error) {
	pageData, err := s.managementRepo.ListGroups(ctx, req)
	if err != nil {
		return nil, err
	}

	return &model.ListGroupsResponse{
		Groups:     pageData.Items,
		TotalCount: int(pageData.Total),
	}, nil
}

func (s *groupService) Get(ctx context.Context, id string) (*model.Group, error) {
	return s.managementRepo.GetGroup(ctx, id)
}

func (s *groupService) Update(ctx context.Context, id string, req *model.UpdateGroupRequest) (*model.Group, error) {
	return s.managementRepo.UpdateGroup(ctx, id, req)
}

func (s *groupService) Delete(ctx context.Context, id string) error {
	err, _ := s.managementRepo.DoRequest(ctx, "DELETE", fmt.Sprintf("/groups/%s", id), nil)
	return err
}

func (s *groupService) GetMembers(ctx context.Context, groupID string) (*model.GroupMembersResponse, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "GET", fmt.Sprintf("/groups/%s/members", groupID), nil)
	if err != nil {
		return nil, err
	}

	result := &model.GroupMembersResponse{
		Members: []*model.Employee{},
	}

	if data, ok := resp.Data.(map[string]interface{}); ok {
		if members, ok := data["members"].([]interface{}); ok {
			for _, item := range members {
				if member, err := parseEmployee(item); err == nil {
					result.Members = append(result.Members, member)
				}
			}
		}
	}

	return result, nil
}

func (s *groupService) AddMember(ctx context.Context, req *model.AddGroupMemberRequest) error {
	err, _ := s.managementRepo.DoRequest(ctx, "POST", fmt.Sprintf("/groups/%s/members", req.GroupID), req)
	return err
}

func (s *groupService) RemoveMember(ctx context.Context, req *model.RemoveGroupMemberRequest) error {
	err, _ := s.managementRepo.DoRequest(ctx, "DELETE", fmt.Sprintf("/groups/%s/members/%s", req.GroupID, req.EmployeeID), nil)
	return err
}

func parseGroup(data interface{}) (*model.Group, error) {
	groupData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid group data format")
	}

	group := &model.Group{}
	if id, ok := groupData["id"].(string); ok {
		group.ID = id
	}
	if name, ok := groupData["name"].(string); ok {
		group.Name = name
	}
	if groupType, ok := groupData["type"].(string); ok {
		group.Type = groupType
	}
	if orgID, ok := groupData["orgId"].(string); ok {
		group.OrgID = orgID
	}
	if memberCount, ok := groupData["memberCount"].(float64); ok {
		group.MemberCount = int(memberCount)
	}

	return group, nil
}
