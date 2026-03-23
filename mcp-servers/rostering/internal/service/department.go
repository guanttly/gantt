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

type departmentService struct {
	logger         logging.ILogger
	cfg            config.IRosteringConfigurator
	managementRepo repository.IManagementRepository
}

func newDepartmentService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	managementRepo repository.IManagementRepository,
) service.IDepartmentService {
	return &departmentService{
		logger:         logger,
		cfg:            cfg,
		managementRepo: managementRepo,
	}
}

// Create 创建部门
func (s *departmentService) Create(ctx context.Context, req *model.CreateDepartmentRequest) (*model.Department, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "POST", "/departments", req)
	if err != nil {
		return nil, err
	}
	return parseDepartment(resp.Data)
}

// GetList 获取部门列表
func (s *departmentService) GetList(ctx context.Context, orgID string) (*model.ListDepartmentsResponse, error) {
	path := fmt.Sprintf("/departments?orgId=%s", orgID)

	err, resp := s.managementRepo.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	result := &model.ListDepartmentsResponse{
		Departments: []*model.Department{},
	}

	if data, ok := resp.Data.(map[string]interface{}); ok {
		if depts, ok := data["departments"].([]interface{}); ok {
			for _, item := range depts {
				if dept, err := parseDepartment(item); err == nil {
					result.Departments = append(result.Departments, dept)
				}
			}
		}
		if total, ok := data["totalCount"].(float64); ok {
			result.TotalCount = int(total)
		}
	}

	return result, nil
}

// Get 获取单个部门
func (s *departmentService) Get(ctx context.Context, id string) (*model.Department, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "GET", fmt.Sprintf("/departments/%s", id), nil)
	if err != nil {
		return nil, err
	}
	return parseDepartment(resp.Data)
}

// Update 更新部门
func (s *departmentService) Update(ctx context.Context, id string, req *model.UpdateDepartmentRequest) (*model.Department, error) {
	err, resp := s.managementRepo.DoRequest(ctx, "PUT", fmt.Sprintf("/departments/%s", id), req)
	if err != nil {
		return nil, err
	}
	return parseDepartment(resp.Data)
}

// Delete 删除部门
func (s *departmentService) Delete(ctx context.Context, id string) error {
	err, _ := s.managementRepo.DoRequest(ctx, "DELETE", fmt.Sprintf("/departments/%s", id), nil)
	return err
}

// GetTree 获取部门树
func (s *departmentService) GetTree(ctx context.Context, orgID string) ([]*model.DepartmentTreeNode, error) {
	query := url.Values{}
	query.Set("orgId", orgID)
	query.Set("tree", "true")

	path := "/departments?" + query.Encode()

	err, resp := s.managementRepo.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tree []*model.DepartmentTreeNode
	if data, ok := resp.Data.([]interface{}); ok {
		for _, item := range data {
			if node, err := parseDepartmentTreeNode(item); err == nil {
				tree = append(tree, node)
			}
		}
	}

	return tree, nil
}

// parseDepartment 解析部门响应
func parseDepartment(data interface{}) (*model.Department, error) {
	deptData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid department data format")
	}

	dept := &model.Department{}
	if id, ok := deptData["id"].(string); ok {
		dept.ID = id
	}
	if name, ok := deptData["name"].(string); ok {
		dept.Name = name
	}
	if parentID, ok := deptData["parentId"].(string); ok {
		dept.ParentID = &parentID
	}
	if orgID, ok := deptData["orgId"].(string); ok {
		dept.OrgID = orgID
	}

	return dept, nil
}

// parseDepartmentTreeNode 解析部门树节点
func parseDepartmentTreeNode(data interface{}) (*model.DepartmentTreeNode, error) {
	nodeData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tree node data format")
	}

	node := &model.DepartmentTreeNode{}
	if id, ok := nodeData["id"].(string); ok {
		node.ID = id
	}
	if name, ok := nodeData["name"].(string); ok {
		node.Name = name
	}
	if parentID, ok := nodeData["parentId"].(string); ok {
		node.ParentID = parentID
	}

	if children, ok := nodeData["children"].([]interface{}); ok {
		for _, child := range children {
			if childNode, err := parseDepartmentTreeNode(child); err == nil {
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node, nil
}
