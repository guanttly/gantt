package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/model"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/mcp/pkg/logging"
)

type managementClient struct {
	logger       logging.ILogger
	cfg          config.IRosteringConfigurator
	namingClient naming_client.INamingClient
	serviceName  string
	httpClient   *http.Client
	baseURL      string // 如果不使用服务发现，可以直接配置
}

// newManagementClient 创建基础客户端
func newManagementClient(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
	namingClient naming_client.INamingClient,
	serviceName string,
	baseURL string,
) repository.IManagementRepository {
	return &managementClient{
		logger:       logger,
		cfg:          cfg,
		namingClient: namingClient,
		serviceName:  serviceName,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getServiceURL 从 Nacos 获取服务实例或使用静态配置
func (c *managementClient) getServiceURL(_ context.Context) (string, error) {
	// 否则从 Nacos 服务发现获取
	if c.namingClient == nil {
		return "", fmt.Errorf("no naming client available for service discovery")
	}

	instance, err := c.namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: c.serviceName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to select healthy instance: %w", err)
	}

	return fmt.Sprintf("http://%s:%d/%s", instance.Ip, instance.Port, c.baseURL), nil
}

// DoRequest 统一的 HTTP 请求处理
func (c *managementClient) DoRequest(ctx context.Context, method, path string, body any) (err error, response *model.APIResponse) {
	baseURL, err := c.getServiceURL(ctx)
	if err != nil {
		return err, nil
	}

	fullURL := baseURL + path
	c.logger.Debug("Making request", "method", method, "url", fullURL)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err), nil
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err), nil
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err), nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err), nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody)), nil
	}

	response = &model.APIResponse{}
	if err := json.Unmarshal(respBody, response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err), nil
	}

	return nil, response
}

// DoRequestWithData 执行HTTP请求并反序列化到指定类型
func (c *managementClient) DoRequestWithData(ctx context.Context, method, path string, body any, data any) error {
	err, resp := c.DoRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	// 将 resp.Data 序列化后再反序列化到目标类型
	jsonData, err := json.Marshal(resp.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	if err := json.Unmarshal(jsonData, data); err != nil {
		return fmt.Errorf("failed to unmarshal to target type: %w", err)
	}

	return nil
}

// doPageRequest 通用的分页请求处理
func doPageRequest[T any](c *managementClient, ctx context.Context, path string) (*model.PageData[T], error) {
	err, resp := c.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// resp.Data 应该是分页数据
	jsonData, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	// 先解析为 map 以处理时间格式问题（MySQL datetime -> RFC3339）
	var rawData map[string]any
	if err := json.Unmarshal(jsonData, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	// 处理 items 数组中的时间字段（MySQL datetime 格式转换为 RFC3339）
	if items, ok := rawData["items"].([]any); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]any); ok {
				normalizeTimeFields(itemMap)
			}
		}
	}

	// 重新序列化处理后的数据
	normalizedData, err := json.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized data: %w", err)
	}

	var pageData model.PageData[T]
	if err := json.Unmarshal(normalizedData, &pageData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal page data: %w", err)
	}

	return &pageData, nil
}

// normalizeTimeFields 将 MySQL datetime 格式的时间字符串转换为 RFC3339 格式
// 同时处理 department 字段（对象 -> 字符串ID）
// MySQL 格式: "2025-11-06 13:43:09"
// RFC3339 格式: "2025-11-06T13:43:09Z"
func normalizeTimeFields(m map[string]any) {
	// 处理时间字段
	timeFields := []string{"createdAt", "updatedAt", "hireDate", "deletedAt"}
	for _, field := range timeFields {
		if val, ok := m[field]; ok && val != nil {
			if strVal, ok := val.(string); ok && strVal != "" {
				// 尝试解析 MySQL datetime 格式
				if t, err := parseMySQLDateTime(strVal); err == nil {
					// 转换为 RFC3339 格式
					m[field] = t.Format(time.RFC3339)
				}
			}
		}
	}

	// 处理 department 字段：如果是对象，提取 id 或 departmentId
	if deptVal, ok := m["department"]; ok && deptVal != nil {
		if deptMap, ok := deptVal.(map[string]any); ok {
			// 尝试从对象中提取 id 或 departmentId
			if id, ok := deptMap["id"].(string); ok && id != "" {
				m["department"] = id
			} else if id, ok := deptMap["departmentId"].(string); ok && id != "" {
				m["department"] = id
			}
		}
		// 如果已经是字符串，保持不变
	}
}

// parseMySQLDateTime 解析 MySQL datetime 格式的字符串
// 支持格式: "2025-11-06 13:43:09" 或 "2025-11-06 13:43:09.000000"
func parseMySQLDateTime(s string) (time.Time, error) {
	// 移除可能的微秒部分
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}
	// 尝试解析标准 MySQL datetime 格式
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", s)
}

// ==================== 员工相关 ====================

func (c *managementClient) GetEmployee(ctx context.Context, id string) (*model.Employee, error) {
	var employee model.Employee
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/employees/%s", id), nil, &employee); err != nil {
		return nil, err
	}
	return &employee, nil
}

func (c *managementClient) ListEmployees(ctx context.Context, req *model.ListEmployeesRequest) (*model.PageData[*model.Employee], error) {
	// 使用 url.Values 构建查询参数，确保正确的 URL 编码
	params := url.Values{}
	params.Set("orgId", req.OrgID)
	if req.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", req.Page))
	} else {
		params.Set("page", "1") // 默认值
	}
	if req.PageSize > 0 {
		params.Set("size", fmt.Sprintf("%d", req.PageSize)) // 管理服务使用 "size" 而不是 "pageSize"
	} else {
		params.Set("size", "100") // 默认值
	}
	if req.DepartmentID != "" {
		params.Set("department", req.DepartmentID) // 管理服务使用 "department" 而不是 "departmentId"
	}
	if req.Keyword != "" {
		params.Set("keyword", req.Keyword)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	path := "/employees?" + params.Encode()
	return doPageRequest[*model.Employee](c, ctx, path)
}

func (c *managementClient) CreateEmployee(ctx context.Context, req *model.CreateEmployeeRequest) (*model.Employee, error) {
	var employee model.Employee
	if err := c.DoRequestWithData(ctx, "POST", "/employees", req, &employee); err != nil {
		return nil, err
	}
	return &employee, nil
}

func (c *managementClient) UpdateEmployee(ctx context.Context, id string, req *model.UpdateEmployeeRequest) (*model.Employee, error) {
	var employee model.Employee
	if err := c.DoRequestWithData(ctx, "PUT", fmt.Sprintf("/employees/%s", id), req, &employee); err != nil {
		return nil, err
	}
	return &employee, nil
}

func (c *managementClient) DeleteEmployee(ctx context.Context, id string) error {
	err, _ := c.DoRequest(ctx, "DELETE", fmt.Sprintf("/employees/%s", id), nil)
	return err
}

// ==================== 班次相关 ====================

func (c *managementClient) GetShift(ctx context.Context, id string) (*model.Shift, error) {
	var shift model.Shift
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/shifts/%s", id), nil, &shift); err != nil {
		return nil, err
	}
	return &shift, nil
}

func (c *managementClient) ListShifts(ctx context.Context, req *model.ListShiftsRequest) (*model.PageData[*model.Shift], error) {
	// 注意：管理服务的分页参数名为 "size"，不是 "pageSize"
	path := fmt.Sprintf("/shifts?orgId=%s&page=%d&size=%d", req.OrgID, req.Page, req.PageSize)
	if req.Keyword != "" {
		path += "&keyword=" + req.Keyword
	}
	// 若调用方未指定 isActive 过滤，则默认只返回启用的班次（排班场景只需要激活的班次）
	if req.Status != "" {
		path += "&isActive=" + req.Status
	} else {
		path += "&isActive=true"
	}
	return doPageRequest[*model.Shift](c, ctx, path)
}

func (c *managementClient) CreateShift(ctx context.Context, req *model.CreateShiftRequest) (*model.Shift, error) {
	var shift model.Shift
	if err := c.DoRequestWithData(ctx, "POST", "/shifts", req, &shift); err != nil {
		return nil, err
	}
	return &shift, nil
}

func (c *managementClient) UpdateShift(ctx context.Context, id string, req *model.UpdateShiftRequest) (*model.Shift, error) {
	var shift model.Shift
	if err := c.DoRequestWithData(ctx, "PUT", fmt.Sprintf("/shifts/%s", id), req, &shift); err != nil {
		return nil, err
	}
	return &shift, nil
}

func (c *managementClient) DeleteShift(ctx context.Context, id string) error {
	err, _ := c.DoRequest(ctx, "DELETE", fmt.Sprintf("/shifts/%s", id), nil)
	return err
}

func (c *managementClient) SetShiftGroups(ctx context.Context, shiftID string, req *model.SetShiftGroupsRequest) error {
	err, _ := c.DoRequest(ctx, "PUT", fmt.Sprintf("/shifts/%s/groups", shiftID), req)
	return err
}

func (c *managementClient) AddShiftGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error {
	err, _ := c.DoRequest(ctx, "POST", fmt.Sprintf("/shifts/%s/groups/%s", shiftID, req.GroupID), req)
	return err
}

func (c *managementClient) RemoveShiftGroup(ctx context.Context, shiftID string, groupID string) error {
	err, _ := c.DoRequest(ctx, "DELETE", fmt.Sprintf("/shifts/%s/groups/%s", shiftID, groupID), nil)
	return err
}

func (c *managementClient) GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error) {
	var groups []*model.ShiftGroup
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/shifts/%s/groups", shiftID), nil, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *managementClient) ToggleShiftStatus(ctx context.Context, id string, status string) error {
	req := map[string]string{"status": status}
	err, _ := c.DoRequest(ctx, "PATCH", fmt.Sprintf("/shifts/%s/status", id), req)
	return err
}

func (c *managementClient) GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error) {
	var members []*model.Employee
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/shifts/%s/members", shiftID), nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// ==================== 规则相关 ====================

func (c *managementClient) GetRule(ctx context.Context, id string) (*model.Rule, error) {
	var rule model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/scheduling-rules/%s", id), nil, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *managementClient) ListRules(ctx context.Context, req *model.ListRulesRequest) (*model.PageData[*model.Rule], error) {
	path := fmt.Sprintf("/scheduling-rules?orgId=%s&page=%d&pageSize=%d", req.OrgID, req.Page, req.PageSize)
	if req.RuleType != "" {
		path += "&ruleType=" + req.RuleType
	}
	if req.ApplyScope != "" {
		path += "&applyScope=" + req.ApplyScope
	}
	if req.TimeScope != "" {
		path += "&timeScope=" + req.TimeScope
	}
	if req.IsActive != nil {
		path += fmt.Sprintf("&isActive=%t", *req.IsActive)
	}
	if req.Keyword != "" {
		path += "&keyword=" + req.Keyword
	}
	return doPageRequest[*model.Rule](c, ctx, path)
}

func (c *managementClient) CreateRule(ctx context.Context, req *model.CreateRuleRequest) (*model.Rule, error) {
	var rule model.Rule
	if err := c.DoRequestWithData(ctx, "POST", "/scheduling-rules", req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *managementClient) UpdateRule(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.Rule, error) {
	var rule model.Rule
	if err := c.DoRequestWithData(ctx, "PUT", fmt.Sprintf("/scheduling-rules/%s", id), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *managementClient) DeleteRule(ctx context.Context, id string) error {
	err, _ := c.DoRequest(ctx, "DELETE", fmt.Sprintf("/scheduling-rules/%s", id), nil)
	return err
}

func (c *managementClient) GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.Rule, error) {
	var rules []*model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/employees/%s/scheduling-rules?orgId=%s", employeeID, orgID), nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *managementClient) GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error) {
	var rules []*model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/groups/%s/scheduling-rules?orgId=%s", groupID, orgID), nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *managementClient) GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	var rules []*model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/shifts/%s/scheduling-rules?orgId=%s", shiftID, orgID), nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *managementClient) GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error) {
	var result map[string][]*model.Rule
	req := map[string]any{
		"employeeIds": employeeIDs,
	}
	if err := c.DoRequestWithData(ctx, "POST", fmt.Sprintf("/scheduling-rules/batch/employees?orgId=%s", orgID), req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *managementClient) GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error) {
	var result map[string][]*model.Rule
	req := map[string]any{
		"shiftIds": shiftIDs,
	}
	if err := c.DoRequestWithData(ctx, "POST", fmt.Sprintf("/scheduling-rules/batch/shifts?orgId=%s", orgID), req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *managementClient) GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error) {
	var result map[string][]*model.Rule
	req := map[string]any{
		"groupIds": groupIDs,
	}
	if err := c.DoRequestWithData(ctx, "POST", fmt.Sprintf("/scheduling-rules/batch/groups?orgId=%s", orgID), req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ==================== V4.1 规则班次关系相关 ====================

func (c *managementClient) CreateRuleWithRelations(ctx context.Context, req *model.CreateRuleWithRelationsRequest) (*model.Rule, error) {
	var rule model.Rule
	if err := c.DoRequestWithData(ctx, "POST", "/scheduling-rules", req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *managementClient) UpdateRuleWithRelations(ctx context.Context, id string, req *model.UpdateRuleWithRelationsRequest) (*model.Rule, error) {
	var rule model.Rule
	if err := c.DoRequestWithData(ctx, "PUT", fmt.Sprintf("/scheduling-rules/%s", id), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *managementClient) GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	var rules []*model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/scheduling-rules/by-subject-shift/%s?orgId=%s", shiftID, orgID), nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *managementClient) GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error) {
	var rules []*model.Rule
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/scheduling-rules/by-object-shift/%s?orgId=%s", shiftID, orgID), nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// ==================== 分组相关 ====================

func (c *managementClient) GetGroup(ctx context.Context, id string) (*model.Group, error) {
	var group model.Group
	if err := c.DoRequestWithData(ctx, "GET", fmt.Sprintf("/groups/%s", id), nil, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (c *managementClient) ListGroups(ctx context.Context, req *model.ListGroupsRequest) (*model.PageData[*model.Group], error) {
	path := fmt.Sprintf("/groups?orgId=%s&page=%d&pageSize=%d", req.OrgID, req.Page, req.PageSize)
	if req.Type != "" {
		path += "&type=" + req.Type
	}
	return doPageRequest[*model.Group](c, ctx, path)
}

func (c *managementClient) CreateGroup(ctx context.Context, req *model.CreateGroupRequest) (*model.Group, error) {
	var group model.Group
	if err := c.DoRequestWithData(ctx, "POST", "/groups", req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (c *managementClient) UpdateGroup(ctx context.Context, id string, req *model.UpdateGroupRequest) (*model.Group, error) {
	var group model.Group
	if err := c.DoRequestWithData(ctx, "PUT", fmt.Sprintf("/groups/%s", id), req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (c *managementClient) DeleteGroup(ctx context.Context, id string) error {
	err, _ := c.DoRequest(ctx, "DELETE", fmt.Sprintf("/groups/%s", id), nil)
	return err
}

// ==================== 班次周人数相关 ====================

func (c *managementClient) GetShiftWeeklyStaff(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error) {
	var config model.ShiftWeeklyStaffConfig
	path := fmt.Sprintf("/shifts/%s/weekly-staff?orgId=%s", shiftID, orgID)
	if err := c.DoRequestWithData(ctx, "GET", path, nil, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *managementClient) SetShiftWeeklyStaff(ctx context.Context, orgID, shiftID string, req *model.SetShiftWeeklyStaffRequest) error {
	path := fmt.Sprintf("/shifts/%s/weekly-staff?orgId=%s", shiftID, orgID)
	err, _ := c.DoRequest(ctx, "PUT", path, req)
	return err
}

func (c *managementClient) CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error) {
	var preview model.StaffingCalculationPreview
	path := fmt.Sprintf("/staffing/calculate?orgId=%s", orgID)
	body := map[string]string{"shiftId": shiftID}
	if err := c.DoRequestWithData(ctx, "POST", path, body, &preview); err != nil {
		return nil, err
	}
	return &preview, nil
}

// ListFixedAssignmentsByShift 获取班次的所有固定人员配置
func (c *managementClient) ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error) {
	var assignments []*model.ShiftFixedAssignment
	path := fmt.Sprintf("/shifts/%s/fixed-assignments", shiftID)
	if err := c.DoRequestWithData(ctx, "GET", path, nil, &assignments); err != nil {
		return nil, err
	}
	return assignments, nil
}

// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
func (c *managementClient) CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error) {
	var result map[string][]string
	path := fmt.Sprintf("/shifts/%s/fixed-assignments/calculate", shiftID)
	body := map[string]string{
		"startDate": startDate,
		"endDate":   endDate,
	}
	if err := c.DoRequestWithData(ctx, "POST", path, body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
func (c *managementClient) CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error) {
	var result map[string]map[string][]string
	path := "/fixed-assignments/calculate-multiple"
	body := map[string]any{
		"shiftIds":  shiftIDs,
		"startDate": startDate,
		"endDate":   endDate,
	}
	if err := c.DoRequestWithData(ctx, "POST", path, body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSystemSetting 获取系统设置值
func (c *managementClient) GetSystemSetting(ctx context.Context, orgID, key string) (string, error) {
	// DoRequestWithData 会将 resp.Data 直接反序列化到目标类型
	// resp.Data 的格式是 map[string]string{"key": "...", "value": "..."}
	var response struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	// 对 key 进行 URL 编码，确保特殊字符能正确传递
	encodedKey := url.PathEscape(key)
	path := fmt.Sprintf("/system-settings/%s?orgId=%s", encodedKey, url.QueryEscape(orgID))
	if err := c.DoRequestWithData(ctx, "GET", path, nil, &response); err != nil {
		c.logger.Error("Failed to get system setting", "error", err, "orgID", orgID, "key", key, "path", path)
		return "", err
	}
	c.logger.Debug("System setting retrieved", "key", response.Key, "value", response.Value)
	return response.Value, nil
}

// SetSystemSetting 设置系统设置值
func (c *managementClient) SetSystemSetting(ctx context.Context, orgID, key, value string) error {
	path := fmt.Sprintf("/system-settings/%s?orgId=%s", key, orgID)
	body := map[string]string{
		"value": value,
	}
	err, _ := c.DoRequest(ctx, "PUT", path, body)
	return err
}
