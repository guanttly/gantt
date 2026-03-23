package service

import (
	"context"
	"encoding/json"
	"fmt"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
	"log/slog"

	"jusha/agent/rostering/domain/service"
	scheduling_domain "jusha/agent/sdk/rostering/domain"
	client_model "jusha/agent/sdk/rostering/model"

	d_model "jusha/agent/rostering/domain/model"
)

// rosteringServiceImpl 数据服务实现
type rosteringServiceImpl struct {
	logger          logging.ILogger
	rosteringClient scheduling_domain.IClient
	toolBus         mcp.IToolBus // 用于调用 MCP 工具（固定人员配置等）
}

// NewRosteringService 创建数据服务实现
func NewRosteringService(
	logger logging.ILogger,
	rosteringClient scheduling_domain.IClient,
	toolBus mcp.IToolBus,
) service.IRosteringService {
	if logger == nil {
		logger = slog.Default()
	}

	return &rosteringServiceImpl{
		logger:          logger.With("component", "RosteringService"),
		rosteringClient: rosteringClient,
		toolBus:         toolBus,
	}
}

// 排班数据服务实现

// QuerySchedules 查询排班
func (s *rosteringServiceImpl) QuerySchedules(ctx context.Context, filter d_model.ScheduleQueryFilter) (*d_model.ScheduleQueryResult, error) {
	s.logger.Debug("Querying schedules", "filter", filter)

	req := client_model.GetScheduleByDateRangeRequest{
		OrgID:     filter.OrgID,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}

	resp, err := s.rosteringClient.GetScheduleByDateRange(ctx, req)
	if err != nil {
		s.logger.Error("Failed to query schedules", "error", err, "filter", filter)
		return nil, fmt.Errorf("query schedules: %w", err)
	}

	var schedules []*client_model.ScheduleAssignment

	// 处理 interface{} 类型的 Schedules
	if resp.Schedules != nil {
		// 尝试直接类型断言
		if items, ok := resp.Schedules.([]*client_model.ScheduleAssignment); ok {
			schedules = items
		} else if itemsAny, ok := resp.Schedules.([]interface{}); ok {
			// JSON 反序列化后通常是 []interface{}
			schedules = make([]*client_model.ScheduleAssignment, 0, len(itemsAny))
			for _, item := range itemsAny {
				if itemMap, ok := item.(map[string]interface{}); ok {
					schedule := &client_model.ScheduleAssignment{}
					if id, ok := itemMap["id"].(string); ok {
						schedule.ID = id
					}
					if orgID, ok := itemMap["orgId"].(string); ok {
						schedule.OrgID = orgID
					}
					if date, ok := itemMap["date"].(string); ok {
						schedule.Date = date
					}
					if employeeID, ok := itemMap["employeeId"].(string); ok {
						schedule.EmployeeID = employeeID
					}
					if shiftID, ok := itemMap["shiftId"].(string); ok {
						schedule.ShiftID = shiftID
					}
					if notes, ok := itemMap["notes"].(string); ok {
						schedule.Notes = notes
					}
					schedules = append(schedules, schedule)
				}
			}
		}
	}

	result := &d_model.ScheduleQueryResult{
		Schedules: schedules,
		Total:     len(schedules),
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		HasMore:   false,
	}

	s.logger.Debug("Successfully queried schedules", "count", len(result.Schedules))
	return result, nil
}

// UpsertSchedule 创建或更新排班
func (s *rosteringServiceImpl) UpsertSchedule(ctx context.Context, req d_model.ScheduleUpsertRequest) (*d_model.ScheduleEntry, error) {
	s.logger.Debug("Upserting schedule", "request", req)

	assignment := &client_model.ScheduleAssignment{
		OrgID:      req.OrgID,
		Date:       req.WorkDate,
		EmployeeID: req.UserID,
		ShiftID:    req.ShiftCode,
		Notes:      req.Notes,
	}

	batchReq := client_model.BatchAssignRequest{
		OrgID:       req.OrgID,
		Assignments: []*client_model.ScheduleAssignment{assignment},
	}

	err := s.rosteringClient.BatchAssignSchedule(ctx, batchReq)
	if err != nil {
		s.logger.Error("Failed to upsert schedule", "error", err, "request", req)
		return nil, fmt.Errorf("upsert schedule: %w", err)
	}

	entry := &d_model.ScheduleEntry{
		UserID:    req.UserID,
		WorkDate:  req.WorkDate,
		ShiftCode: req.ShiftCode,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Notes:     req.Notes,
	}

	s.logger.Debug("Successfully upserted schedule")
	return entry, nil
}

// BatchUpsertSchedules 批量创建或更新排班
func (s *rosteringServiceImpl) BatchUpsertSchedules(ctx context.Context, batch d_model.ScheduleBatch) (*d_model.BatchUpsertResult, error) {
	s.logger.Debug("Batch upserting schedules", "itemCount", len(batch.Items))

	if len(batch.Items) == 0 {
		return &d_model.BatchUpsertResult{Total: 0, Upserted: 0, Failed: 0}, nil
	}

	assignments := make([]*client_model.ScheduleAssignment, len(batch.Items))
	for i, item := range batch.Items {
		assignments[i] = &client_model.ScheduleAssignment{
			OrgID:      item.OrgID,
			Date:       item.WorkDate,
			EmployeeID: item.UserID,
			ShiftID:    item.ShiftCode,
			Notes:      item.Notes,
		}
	}

	batchReq := client_model.BatchAssignRequest{
		OrgID:       batch.Items[0].OrgID,
		Assignments: assignments,
	}

	err := s.rosteringClient.BatchAssignSchedule(ctx, batchReq)
	if err != nil {
		s.logger.Error("Failed to batch upsert schedules", "error", err, "itemCount", len(batch.Items))
		return nil, fmt.Errorf("batch upsert schedules: %w", err)
	}

	result := &d_model.BatchUpsertResult{
		Total:    len(batch.Items),
		Upserted: len(batch.Items),
		Failed:   0,
	}

	s.logger.Debug("Successfully batch upserted schedules", "upserted", result.Upserted)
	return result, nil
}

// DeleteSchedule 删除排班
func (s *rosteringServiceImpl) DeleteSchedule(ctx context.Context, userID, date string) error {
	s.logger.Debug("Deleting schedule", "userID", userID, "date", date)

	err := s.rosteringClient.DeleteSchedule(ctx, "", userID, date)
	if err != nil {
		s.logger.Error("Failed to delete schedule", "error", err, "userID", userID, "date", date)
		return fmt.Errorf("delete schedule: %w", err)
	}

	s.logger.Debug("Successfully deleted schedule", "userID", userID, "date", date)
	return nil
}

// 人员数据服务实现

// GetStaffProfiles 获取人员档案
func (s *rosteringServiceImpl) GetStaffProfiles(ctx context.Context, orgID, department, modality string) ([]*d_model.Staff, error) {
	s.logger.Debug("Getting staff profiles", "orgID", orgID, "department", department, "modality", modality)

	req := &client_model.ListEmployeesRequest{
		OrgID:        orgID,
		DepartmentID: department,
		Page:         1,
		PageSize:     100,
	}

	resp, err := s.rosteringClient.ListEmployees(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get staff profiles", "error", err, "orgID", orgID)
		return nil, fmt.Errorf("get staff profiles: %w", err)
	}

	s.logger.Debug("Successfully got staff profiles", "count", len(resp.Items))
	return resp.Items, nil
} // ListStaff 列出人员
func (s *rosteringServiceImpl) ListStaff(ctx context.Context, filter d_model.StaffListFilter) (*d_model.StaffListResult, error) {
	s.logger.Debug("Listing staff", "filter", filter)

	req := client_model.ListEmployeesRequest(filter)
	// 如果 Page 或 PageSize 为 0，设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 1000
	}
	resp, err := s.rosteringClient.ListEmployees(ctx, &req)
	if err != nil {
		s.logger.Error("Failed to list staff", "error", err, "filter", filter)
		return nil, fmt.Errorf("list staff: %w", err)
	}

	s.logger.Debug("Successfully listed staff", "count", len(resp.Items), "total", resp.Total)
	return resp, nil
}

// SearchStaff 搜索人员
func (s *rosteringServiceImpl) SearchStaff(ctx context.Context, filter d_model.StaffSearchFilter) (*d_model.StaffListResult, error) {
	s.logger.Debug("Searching staff", "filter", filter)

	req := client_model.ListEmployeesRequest(filter)
	// 如果 Page 或 PageSize 为 0，设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 1000
	}
	resp, err := s.rosteringClient.ListEmployees(ctx, &req)
	if err != nil {
		s.logger.Error("Failed to search staff", "error", err, "filter", filter)
		return nil, fmt.Errorf("search staff: %w", err)
	}

	s.logger.Debug("Successfully searched staff", "count", len(resp.Items))
	return resp, nil
}

// CheckStaffExists 检查员工是否存在
func (s *rosteringServiceImpl) CheckStaffExists(ctx context.Context, orgID, name string) (bool, error) {
	s.logger.Debug("Checking if staff exists", "orgID", orgID, "name", name)

	req := &client_model.ListEmployeesRequest{
		OrgID:    orgID,
		Keyword:  name,
		Page:     1,
		PageSize: 1,
	}

	resp, err := s.rosteringClient.ListEmployees(ctx, req)
	if err != nil {
		s.logger.Error("Failed to check staff exists", "error", err, "orgID", orgID, "name", name)
		return false, fmt.Errorf("check staff exists: %w", err)
	}

	exists := resp.Total > 0
	s.logger.Debug("Successfully checked staff existence", "exists", exists, "name", name)
	return exists, nil
}

// CreateStaff 创建人员
func (s *rosteringServiceImpl) CreateStaff(ctx context.Context, req d_model.StaffCreateRequest) (string, error) {
	s.logger.Debug("Creating staff", "request", req)

	clientReq := client_model.CreateEmployeeRequest(req)
	id, err := s.rosteringClient.CreateEmployee(ctx, &clientReq)
	if err != nil {
		s.logger.Error("Failed to create staff", "error", err, "request", req)
		return "", fmt.Errorf("create staff: %w", err)
	}

	s.logger.Debug("Successfully created staff", "id", id)
	return id, nil
}

// GetStaff 根据 userID 获取单个人员信息
func (s *rosteringServiceImpl) GetStaff(ctx context.Context, userID string) (*d_model.Staff, error) {
	s.logger.Debug("Getting staff", "userID", userID)

	req := &client_model.ListEmployeesRequest{
		Keyword:  userID,
		Page:     1,
		PageSize: 10,
	}

	resp, err := s.rosteringClient.ListEmployees(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get staff", "error", err, "userID", userID)
		return nil, fmt.Errorf("get staff: %w", err)
	}

	for _, emp := range resp.Items {
		if emp.UserID == userID {
			s.logger.Debug("Successfully got staff", "userID", userID, "name", emp.Name)
			return emp, nil
		}
	}

	return nil, fmt.Errorf("staff not found: %s", userID)
}

// 配置数据服务实现

// ListGroups 列出团队
func (s *rosteringServiceImpl) ListGroups(ctx context.Context, orgID string) ([]*d_model.Group, error) {
	s.logger.Debug("Listing groups", "orgID", orgID)

	req := &client_model.ListGroupsRequest{
		OrgID:    orgID,
		Page:     1,
		PageSize: 100,
	}

	resp, err := s.rosteringClient.ListGroups(ctx, req)
	if err != nil {
		s.logger.Error("Failed to list groups", "error", err, "orgID", orgID)
		return nil, fmt.Errorf("list groups: %w", err)
	}

	s.logger.Debug("Successfully listed groups", "count", len(resp.Groups))
	return resp.Groups, nil
} // CreateGroup 创建团队
func (s *rosteringServiceImpl) CreateGroup(ctx context.Context, req d_model.CreateGroupRequest) (string, error) {
	s.logger.Debug("Creating group", "request", req)

	clientReq := client_model.CreateGroupRequest(req)
	id, err := s.rosteringClient.CreateGroup(ctx, &clientReq)
	if err != nil {
		s.logger.Error("Failed to create group", "error", err, "request", req)
		return "", fmt.Errorf("create group: %w", err)
	}

	s.logger.Debug("Successfully created group", "id", id)
	return id, nil
}

// UpdateGroup 更新团队
func (s *rosteringServiceImpl) UpdateGroup(ctx context.Context, req d_model.UpdateGroupRequest) error {
	s.logger.Debug("Updating group", "request", req)

	clientReq := client_model.UpdateGroupRequest(req)
	err := s.rosteringClient.UpdateGroup(ctx, "", &clientReq)
	if err != nil {
		s.logger.Error("Failed to update group", "error", err, "request", req)
		return fmt.Errorf("update group: %w", err)
	}

	s.logger.Debug("Successfully updated group")
	return nil
}

// AssignGroupMembers 分配团队成员
func (s *rosteringServiceImpl) AssignGroupMembers(ctx context.Context, req d_model.AddGroupMemberRequest) error {
	s.logger.Debug("Assigning group members", "request", req)

	clientReq := client_model.AddGroupMemberRequest(req)
	err := s.rosteringClient.AddGroupMember(ctx, &clientReq)
	if err != nil {
		s.logger.Error("Failed to assign group members", "error", err, "request", req)
		return fmt.Errorf("assign group members: %w", err)
	}

	s.logger.Debug("Successfully assigned group members")
	return nil
}

// GetGroupMembers 获取分组成员
func (s *rosteringServiceImpl) GetGroupMembers(ctx context.Context, groupID string) ([]*d_model.Employee, error) {
	s.logger.Debug("Getting group members", "groupID", groupID)

	resp, err := s.rosteringClient.GetGroupMembers(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get group members", "error", err, "groupID", groupID)
		return nil, fmt.Errorf("get group members: %w", err)
	}

	s.logger.Debug("Successfully got group members", "count", len(resp.Members))
	return resp.Members, nil
}

// ListShifts 查询班次列表
func (s *rosteringServiceImpl) ListShifts(ctx context.Context, orgID string, groupID string) ([]*d_model.Shift, error) {
	s.logger.Debug("Listing shifts", "orgID", orgID, "groupID", groupID)

	req := &client_model.ListShiftsRequest{
		OrgID:    orgID,
		Page:     1,
		PageSize: 100,
	}

	resp, err := s.rosteringClient.ListShifts(ctx, req)
	if err != nil {
		s.logger.Error("Failed to list shifts", "error", err, "orgID", orgID)
		return nil, fmt.Errorf("list shifts: %w", err)
	}

	s.logger.Debug("Successfully listed shifts", "count", len(resp.Items))
	return resp.Items, nil
} // 请假数据服务实现

// CreateLeave 创建请假记录
func (s *rosteringServiceImpl) CreateLeave(ctx context.Context, req d_model.CreateLeaveRequest) (string, error) {
	s.logger.Debug("Creating leave", "request", req)

	clientReq := client_model.CreateLeaveRequest(req)
	id, err := s.rosteringClient.CreateLeave(ctx, clientReq)
	if err != nil {
		s.logger.Error("Failed to create leave", "error", err, "request", req)
		return "", fmt.Errorf("create leave: %w", err)
	}

	s.logger.Debug("Successfully created leave", "id", id)
	return id, nil
}

// GetLeaveRecords 获取请假记录（单个员工）
func (s *rosteringServiceImpl) GetLeaveRecords(ctx context.Context, orgID, staffID string, startDate, endDate string) ([]*d_model.LeaveRecord, error) {
	req := client_model.ListLeavesRequest{
		OrgID:      orgID,
		EmployeeID: staffID,
		StartDate:  startDate,
		EndDate:    endDate,
		Page:       1,
		PageSize:   100,
	}

	resp, err := s.rosteringClient.ListLeaves(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get leave records", "error", err, "staffID", staffID)
		return nil, fmt.Errorf("get leave records: %w", err)
	}

	s.logger.Debug("Successfully got leave records", "count", len(resp.Leaves))
	return resp.Leaves, nil
}

// BatchGetLeaveRecords 批量获取请假记录（所有员工，按员工ID分组）
// 优化：一次查询获取所有员工的请假记录，避免多次 RPC 调用
func (s *rosteringServiceImpl) BatchGetLeaveRecords(ctx context.Context, orgID string, startDate, endDate string) (map[string][]*d_model.LeaveRecord, error) {
	// 不传 EmployeeID，一次性获取所有员工的请假记录
	req := client_model.ListLeavesRequest{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      1,
		PageSize:  1000, // 增大页面大小以获取所有记录
	}

	resp, err := s.rosteringClient.ListLeaves(ctx, req)
	if err != nil {
		s.logger.Error("Failed to batch get leave records", "error", err, "orgID", orgID)
		return nil, fmt.Errorf("batch get leave records: %w", err)
	}

	// 按员工ID分组
	staffLeaveMap := make(map[string][]*d_model.LeaveRecord)
	for _, leave := range resp.Leaves {
		if leave.EmployeeID != "" {
			staffLeaveMap[leave.EmployeeID] = append(staffLeaveMap[leave.EmployeeID], leave)
		}
	}

	s.logger.Debug("Successfully batch got leave records", "totalCount", len(resp.Leaves), "staffCount", len(staffLeaveMap))
	return staffLeaveMap, nil
} // 排班范围数据服务实现

// GetGroupsByShiftID 根据班次ID获取可用的团队列表
func (s *rosteringServiceImpl) GetGroupsByShiftID(ctx context.Context, orgID, shiftID string, withFallback bool) ([]*d_model.Group, error) {
	s.logger.Debug("Getting groups by shift ID", "orgID", orgID, "shiftID", shiftID, "withFallback", withFallback)

	// 1. 尝试获取班次关联的分组
	shiftGroups, err := s.GetShiftGroups(ctx, shiftID)
	if err == nil && len(shiftGroups) > 0 {
		var groups []*d_model.Group
		for _, sg := range shiftGroups {
			groups = append(groups, &d_model.Group{
				ID:    sg.GroupID,
				Name:  sg.GroupName,
				Code:  sg.GroupCode,
				OrgID: orgID,
			})
		}
		s.logger.Debug("Found associated groups for shift", "count", len(groups))
		return groups, nil
	}

	if !withFallback {
		return []*d_model.Group{}, nil
	}

	// 2. 如果没有关联分组，且允许回退，则返回所有团队（分组）
	groups, err := s.ListGroups(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to list groups for shift", "error", err, "shiftID", shiftID)
		return nil, err
	}

	s.logger.Debug("Returning all groups for shift as fallback", "shiftID", shiftID, "count", len(groups))
	return groups, nil
}

// 规则数据服务实现

// ListRules 查询规则列表
func (s *rosteringServiceImpl) ListRules(ctx context.Context, req d_model.ListRulesRequest) ([]*d_model.Rule, error) {
	s.logger.Debug("Listing rules", "request", req)

	result, err := s.rosteringClient.ListRules(ctx, req)
	if err != nil {
		s.logger.Error("Failed to list rules", "error", err, "request", req)
		return nil, fmt.Errorf("list rules: %w", err)
	}

	s.logger.Debug("Successfully listed rules", "count", len(result.Items))
	return result.Items, nil
}

// GetRulesForEmployee 获取员工的所有生效规则
func (s *rosteringServiceImpl) GetRulesForEmployee(ctx context.Context, orgID, employeeID, date string) ([]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for employee", "orgID", orgID, "employeeID", employeeID, "date", date)

	rules, err := s.rosteringClient.GetRulesForEmployee(ctx, orgID, employeeID, date)
	if err != nil {
		s.logger.Error("Failed to get rules for employee", "error", err, "orgID", orgID, "employeeID", employeeID)
		return nil, fmt.Errorf("get rules for employee: %w", err)
	}

	s.logger.Debug("Successfully got rules for employee", "count", len(rules))
	return rules, nil
}

// GetRulesForGroup 获取分组的所有生效规则
func (s *rosteringServiceImpl) GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for group", "orgID", orgID, "groupID", groupID)

	rules, err := s.rosteringClient.GetRulesForGroup(ctx, orgID, groupID)
	if err != nil {
		s.logger.Error("Failed to get rules for group", "error", err, "orgID", orgID, "groupID", groupID)
		return nil, fmt.Errorf("get rules for group: %w", err)
	}

	s.logger.Debug("Successfully got rules for group", "count", len(rules))
	return rules, nil
}

// GetRulesForShift 获取班次的所有生效规则
func (s *rosteringServiceImpl) GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for shift", "orgID", orgID, "shiftID", shiftID)

	rules, err := s.rosteringClient.GetRulesForShift(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to get rules for shift", "error", err, "orgID", orgID, "shiftID", shiftID)
		return nil, fmt.Errorf("get rules for shift: %w", err)
	}

	s.logger.Debug("Successfully got rules for shift", "count", len(rules))
	return rules, nil
}

// GetRulesForEmployees 批量获取多个员工的所有生效规则
func (s *rosteringServiceImpl) GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for employees", "orgID", orgID, "employeeCount", len(employeeIDs))

	rules, err := s.rosteringClient.GetRulesForEmployees(ctx, orgID, employeeIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for employees", "error", err, "orgID", orgID, "employeeCount", len(employeeIDs))
		return nil, fmt.Errorf("get rules for employees: %w", err)
	}

	s.logger.Debug("Successfully got rules for employees", "employeeCount", len(rules))
	return rules, nil
}

// GetRulesForShifts 批量获取多个班次的所有生效规则
func (s *rosteringServiceImpl) GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for shifts", "orgID", orgID, "shiftCount", len(shiftIDs))

	rules, err := s.rosteringClient.GetRulesForShifts(ctx, orgID, shiftIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for shifts", "error", err, "orgID", orgID, "shiftCount", len(shiftIDs))
		return nil, fmt.Errorf("get rules for shifts: %w", err)
	}

	s.logger.Debug("Successfully got rules for shifts", "shiftCount", len(rules))
	return rules, nil
}

// GetRulesForGroups 批量获取多个分组的所有生效规则
func (s *rosteringServiceImpl) GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*d_model.Rule, error) {
	s.logger.Debug("Getting rules for groups", "orgID", orgID, "groupCount", len(groupIDs))

	rules, err := s.rosteringClient.GetRulesForGroups(ctx, orgID, groupIDs)
	if err != nil {
		s.logger.Error("Failed to get rules for groups", "error", err, "orgID", orgID, "groupCount", len(groupIDs))
		return nil, fmt.Errorf("get rules for groups: %w", err)
	}

	s.logger.Debug("Successfully got rules for groups", "groupCount", len(rules))
	return rules, nil
}

// 数据库管理服务实现

// EnsureScheduleDBConfigured 确保排班数据库配置
func (s *rosteringServiceImpl) EnsureScheduleDBConfigured() error {
	s.logger.Debug("Ensuring schedule DB configuration")

	// TODO: SDK 暂未提供此接口
	s.logger.Warn("EnsureScheduleDBConfigured not implemented yet")
	return nil
}

// AutoMigrateStaffing 自动迁移人员配置数据
func (s *rosteringServiceImpl) AutoMigrateStaffing(ctx context.Context) error {
	s.logger.Debug("Auto migrating staffing database")

	// TODO: SDK 暂未提供此接口
	s.logger.Warn("AutoMigrateStaffing not implemented yet")
	return nil
}

//班次分组服务实现

// AddShiftGroup 添加班次分组
func (s *rosteringServiceImpl) AddShiftGroup(ctx context.Context, shiftID string, groupID string, priority int) error {
	s.logger.Debug("Adding shift group", "shiftID", shiftID, "groupID", groupID, "priority", priority)

	req := &client_model.AddShiftGroupRequest{
		GroupID:  groupID,
		Priority: priority,
	}

	err := s.rosteringClient.AddShiftGroup(ctx, shiftID, req)
	if err != nil {
		s.logger.Error("Failed to add shift group", "error", err, "shiftID", shiftID, "groupID", groupID)
		return fmt.Errorf("add shift group: %w", err)
	}

	s.logger.Debug("Successfully added shift group")
	return nil
}

// RemoveShiftGroup 移除班次分组
func (s *rosteringServiceImpl) RemoveShiftGroup(ctx context.Context, shiftID string, groupID string) error {
	s.logger.Debug("Removing shift group", "shiftID", shiftID, "groupID", groupID)

	err := s.rosteringClient.RemoveShiftGroup(ctx, shiftID, groupID)
	if err != nil {
		s.logger.Error("Failed to remove shift group", "error", err, "shiftID", shiftID, "groupID", groupID)
		return fmt.Errorf("remove shift group: %w", err)
	}

	s.logger.Debug("Successfully removed shift group")
	return nil
}

// GetShiftGroups 获取班次分组
func (s *rosteringServiceImpl) GetShiftGroups(ctx context.Context, shiftID string) ([]*d_model.ShiftGroup, error) {
	s.logger.Debug("Getting shift groups", "shiftID", shiftID)

	groups, err := s.rosteringClient.GetShiftGroups(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift groups", "error", err, "shiftID", shiftID)
		return nil, fmt.Errorf("get shift groups: %w", err)
	}

	s.logger.Debug("Successfully got shift groups", "count", len(groups))
	return groups, nil
}

// GetShiftGroupMembers 获取班次分组成员
func (s *rosteringServiceImpl) GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*d_model.Employee, error) {
	s.logger.Debug("Getting shift group members", "shiftID", shiftID)

	members, err := s.rosteringClient.GetShiftGroupMembers(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift group members", "error", err, "shiftID", shiftID)
		return nil, fmt.Errorf("get shift group members: %w", err)
	}

	// 转换模型
	result := make([]*d_model.Employee, len(members))
	for i, m := range members {
		result[i] = &d_model.Employee{
			ID:           m.ID,
			OrgID:        m.OrgID,
			EmployeeID:   m.EmployeeID,
			Name:         m.Name,
			DepartmentID: m.DepartmentID,
			Position:     m.Position,
			Status:       m.Status,
		}
	}

	return result, nil
}

// GetWeeklyStaffConfig 获取班次周人数配置
func (s *rosteringServiceImpl) GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*d_model.ShiftWeeklyStaffConfig, error) {
	s.logger.Debug("Getting weekly staff config", "orgID", orgID, "shiftID", shiftID)

	config, err := s.rosteringClient.GetWeeklyStaffConfig(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to get weekly staff config", "error", err, "shiftID", shiftID)
		return nil, fmt.Errorf("get weekly staff config: %w", err)
	}

	return config, nil
}

// SetWeeklyStaffConfig 设置班次周人数配置
func (s *rosteringServiceImpl) SetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string, config []d_model.WeekdayStaffConfig) error {
	s.logger.Debug("Setting weekly staff config", "orgID", orgID, "shiftID", shiftID)

	if err := s.rosteringClient.SetWeeklyStaffConfig(ctx, orgID, shiftID, config); err != nil {
		s.logger.Error("Failed to set weekly staff config", "error", err, "shiftID", shiftID)
		return fmt.Errorf("set weekly staff config: %w", err)
	}

	return nil
}

// CalculateStaffing 计算班次推荐人数
func (s *rosteringServiceImpl) CalculateStaffing(ctx context.Context, orgID, shiftID string) (*d_model.StaffingCalculationPreview, error) {
	s.logger.Debug("Calculating staffing", "orgID", orgID, "shiftID", shiftID)

	preview, err := s.rosteringClient.CalculateStaffing(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to calculate staffing", "error", err, "shiftID", shiftID)
		return nil, fmt.Errorf("calculate staffing: %w", err)
	}

	return preview, nil
}

// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
// 通过 MCP 工具调用 management-service 的批量计算接口
func (s *rosteringServiceImpl) CalculateMultipleFixedSchedules(
	ctx context.Context,
	shiftIDs []string,
	startDate, endDate string,
) (map[string]map[string][]string, error) {
	s.logger.Debug("Calculating multiple fixed schedules",
		"shiftCount", len(shiftIDs),
		"startDate", startDate,
		"endDate", endDate)

	if s.toolBus == nil {
		s.logger.Warn("Tool bus not available, returning empty result")
		result := make(map[string]map[string][]string)
		for _, shiftID := range shiftIDs {
			result[shiftID] = make(map[string][]string)
		}
		return result, nil
	}

	// 调用 MCP 工具：rostering.fixed_assignment.calculate_multiple
	toolName := "rostering.fixed_assignment.calculate_multiple"

	// 构建工具参数
	args := map[string]any{
		"shiftIds":  shiftIDs,
		"startDate": startDate,
		"endDate":   endDate,
	}

	// 调用工具
	resultBytes, err := s.toolBus.Execute(ctx, toolName, args)
	if err != nil {
		s.logger.Error("Failed to call fixed assignment calculate tool", "error", err)
		return nil, fmt.Errorf("调用固定人员配置计算工具失败: %w", err)
	}

	// 解析结果
	// 尝试直接解析为目标格式
	var result map[string]map[string][]string
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		// 如果直接解析失败，尝试作为 CallToolResult 格式解析
		var toolResult struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			} `json:"content"`
		}
		if err2 := json.Unmarshal(resultBytes, &toolResult); err2 == nil {
			// 提取文本内容
			var textContent string
			for _, content := range toolResult.Content {
				if content.Type == "text" && content.Text != "" {
					textContent = content.Text
					break
				}
			}

			if textContent != "" {
				// 解析文本内容
				if err3 := json.Unmarshal([]byte(textContent), &result); err3 != nil {
					s.logger.Error("Failed to parse tool result text content", "error", err3, "content", textContent)
					return nil, fmt.Errorf("解析工具结果内容失败: %w", err3)
				}
			} else {
				s.logger.Error("Failed to parse tool result in any format", "error", err, "result", string(resultBytes))
				return nil, fmt.Errorf("解析工具结果失败: %w", err)
			}
		} else {
			s.logger.Error("Failed to parse tool result in any format", "error", err, "result", string(resultBytes))
			return nil, fmt.Errorf("解析工具结果失败: %w", err)
		}
	}

	s.logger.Debug("Successfully calculated multiple fixed schedules", "shiftCount", len(result))
	return result, nil
}

// GetSystemSetting 获取系统设置值
func (s *rosteringServiceImpl) GetSystemSetting(ctx context.Context, orgID, key string) (string, error) {
	s.logger.Debug("Getting system setting", "orgID", orgID, "key", key)

	if s.rosteringClient == nil {
		return "", fmt.Errorf("rostering client not available")
	}

	value, err := s.rosteringClient.GetSetting(ctx, orgID, key)
	if err != nil {
		s.logger.Error("Failed to get system setting", "error", err, "orgID", orgID, "key", key)
		return "", fmt.Errorf("get system setting: %w", err)
	}

	return value, nil
}

// SetSystemSetting 设置系统设置值
func (s *rosteringServiceImpl) SetSystemSetting(ctx context.Context, orgID, key, value string) error {
	s.logger.Debug("Setting system setting", "orgID", orgID, "key", key)

	if s.rosteringClient == nil {
		return fmt.Errorf("rostering client not available")
	}

	err := s.rosteringClient.SetSetting(ctx, orgID, key, value)
	if err != nil {
		s.logger.Error("Failed to set system setting", "error", err, "orgID", orgID, "key", key)
		return fmt.Errorf("set system setting: %w", err)
	}

	return nil
}

// GetUserWorkflowVersion 获取用户工作流版本偏好
func (s *rosteringServiceImpl) GetUserWorkflowVersion(ctx context.Context, orgID, userID string) (string, error) {
	key := fmt.Sprintf("user_%s_workflow_version", userID)
	version, err := s.GetSystemSetting(ctx, orgID, key)
	if err != nil {
		// 如果获取失败或不存在，返回默认值 "v2"
		s.logger.Debug("User workflow version not found, using default v2", "orgID", orgID, "userID", userID, "error", err)
		return "v2", nil
	}
	// 验证版本值
	if version != "v2" && version != "v3" && version != "v4" {
		s.logger.Warn("Invalid workflow version, using default v2", "orgID", orgID, "userID", userID, "version", version)
		return "v2", nil
	}
	return version, nil
}

// SetUserWorkflowVersion 设置用户工作流版本偏好
func (s *rosteringServiceImpl) SetUserWorkflowVersion(ctx context.Context, orgID, userID, version string) error {
	// 验证版本值
	if version != "v2" && version != "v3" && version != "v4" {
		return fmt.Errorf("invalid workflow version: %s, must be 'v2', 'v3' or 'v4'", version)
	}

	key := fmt.Sprintf("user_%s_workflow_version", userID)
	return s.SetSystemSetting(ctx, orgID, key, version)
}
