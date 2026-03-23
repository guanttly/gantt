package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"

	d_service "jusha/agent/rostering/domain/service"
	common_config "jusha/mcp/pkg/config"
)

// ============================================================
// 类型定义
// ============================================================

// schedulingAIService AI 排班服务实现
type schedulingAIService struct {
	logger           logging.ILogger
	configurator     config.IRosteringConfigurator
	aiFactory        *ai.AIProviderFactory
	baseConfigurator common_config.IServiceConfigurator
}

// StaffNameMapping 员工姓名映射（用于AI交互时的姓名-ID转换）
type StaffNameMapping struct {
	NameToID     map[string]string // displayName -> internal ID
	IDToName     map[string]string // internal ID -> displayName
	DisplayNames []string          // 所有显示名列表（用于提示词）
}

// ============================================================
// 常量定义
// ============================================================

// TaskModel 键名常量 - 用于配置不同AI任务的模型
const (
	TaskModelStaffPreSelection = "staffPreSelection" // 人员预选
	TaskModelScheduleDraft     = "scheduleDraft"     // 排班草案生成
	TaskModelRuleSorting       = "ruleSorting"       // 规则排序
	TaskModelTodoPlan          = "todoPlan"          // Todo计划生成
	TaskModelTodoExecution     = "todoExecution"     // Todo执行
	TaskModelValidation        = "validation"        // 排班校验
	TaskModelProgressiveTask   = "progressiveTask"   // 渐进式任务执行（V3）
)

// ============================================================
// 构造函数
// ============================================================

// NewSchedulingAIService 创建 AI 排班服务
func NewSchedulingAIService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
) (d_service.ISchedulingAIService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configurator is required for scheduling AI service")
	}
	base := cfg.(common_config.IServiceConfigurator)
	if base.GetBaseConfig().AI == nil {
		return nil, fmt.Errorf("AI configuration missing for scheduling AI service")
	}
	factory := ai.NewAIModelFactory(context.Background(), base, logger)
	return &schedulingAIService{
		logger:           logger.With("component", "SchedulingAIService"),
		configurator:     cfg,
		aiFactory:        factory,
		baseConfigurator: base,
	}, nil
}

// ============================================================
// JSON 解析方法
// ============================================================

func (s *schedulingAIService) extractJSON(raw string) string {
	jsonStr := raw
	if strings.Contains(raw, "```") {
		lines := strings.Split(raw, "\n")
		var jsonLines []string
		inCodeBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}
			if inCodeBlock || (!strings.HasPrefix(trimmed, "```") && len(trimmed) > 0) {
				jsonLines = append(jsonLines, line)
			}
		}
		jsonStr = strings.Join(jsonLines, "\n")
	}
	return strings.TrimSpace(jsonStr)
}

// parseTodoPlanResult 解析AI返回的Todo计划结果为强类型
func (s *schedulingAIService) parseTodoPlanResult(raw string) (*d_model.TodoPlanResult, error) {
	jsonStr := s.extractJSON(raw)
	var rawResult struct {
		Todos []struct {
			ID               int      `json:"id"`
			Title            string   `json:"title"`
			Description      string   `json:"description"`
			Priority         int      `json:"priority"`
			TargetDates      []string `json:"targetDates"`
			TargetStaffCount int      `json:"targetStaffCount"`
		} `json:"todos"`
		Summary   string `json:"summary"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal todo plan result failed: %w", err)
	}

	// 转换为强类型
	result := &d_model.TodoPlanResult{
		Todos:     make([]*d_model.SchedulingTodo, 0, len(rawResult.Todos)),
		Summary:   rawResult.Summary,
		Reasoning: rawResult.Reasoning,
	}

	for i, todo := range rawResult.Todos {
		// SchedulingTodo.ID 是 string，Priority 是 string，使用 TargetCount 而不是 TargetStaffCount
		result.Todos = append(result.Todos, &d_model.SchedulingTodo{
			ID:          fmt.Sprintf("%d", todo.ID),
			Order:       i + 1,
			Title:       todo.Title,
			Description: todo.Description,
			Priority:    fmt.Sprintf("%d", todo.Priority),
			TargetDates: todo.TargetDates,
			TargetCount: todo.TargetStaffCount,
			Status:      "pending",
		})
	}

	return result, nil
}

// parseTodoExecutionResult 解析AI返回的Todo执行结果为强类型
func (s *schedulingAIService) parseTodoExecutionResult(raw string, nameMapping *StaffNameMapping) (*d_model.TodoExecutionResult, error) {
	jsonStr := s.extractJSON(raw)
	var rawResult struct {
		Schedule        map[string][]string               `json:"schedule"`
		ScheduleActions map[string]d_model.ScheduleAction `json:"scheduleActions"` // 新增：解析操作类型
		UpdatedStaff    []string                          `json:"updatedStaff"`
		Explanation     string                            `json:"explanation"`
		RemainingIssues []string                          `json:"remainingIssues"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal todo execution result failed: %w", err)
	}

	// 记录解析后的结果
	s.logger.Info("Parsed todo execution result",
		"hasSchedule", rawResult.Schedule != nil,
		"scheduleSize", len(rawResult.Schedule),
		"hasScheduleActions", rawResult.ScheduleActions != nil,
		"scheduleActionsSize", len(rawResult.ScheduleActions))

	// 将AI返回的姓名转换为内部ID
	convertedSchedule := make(map[string][]string)
	for date, names := range rawResult.Schedule {
		staffIDs := make([]string, 0, len(names))
		for _, name := range names {
			if id, exists := nameMapping.NameToID[name]; exists {
				staffIDs = append(staffIDs, id)
			} else if id := s.fuzzyMatchName(name, nameMapping.NameToID); id != "" {
				staffIDs = append(staffIDs, id)
			} else {
				s.logger.Warn("Staff name not found in mapping", "name", name)
			}
		}
		convertedSchedule[date] = staffIDs
	}

	// 注意：TodoExecutionResult 使用 Issues 而不是 RemainingIssues
	result := &d_model.TodoExecutionResult{
		Schedule:        convertedSchedule,
		ScheduleActions: rawResult.ScheduleActions, // 新增：传递操作类型
		Explanation:     rawResult.Explanation,
		Issues:          rawResult.RemainingIssues, // AI 返回的 remainingIssues 映射到 Issues
		Success:         true,
	}

	s.logger.Info("Final execution result",
		"scheduleSize", len(result.Schedule),
		"scheduleActionsSize", len(result.ScheduleActions))

	return result, nil
}

// parseValidationResult 解析AI返回的校验结果为强类型
func (s *schedulingAIService) parseValidationResult(raw string, nameMapping *StaffNameMapping) (*d_model.ValidationResult, error) {
	jsonStr := s.extractJSON(raw)
	var rawResult struct {
		Passed bool `json:"passed"`
		Issues []struct {
			Type          string   `json:"type"`
			Severity      string   `json:"severity"`
			Description   string   `json:"description"`
			AffectedDates []string `json:"affectedDates"`
			AffectedStaff []string `json:"affectedStaff"`
		} `json:"issues"`
		AdjustedSchedule map[string][]string `json:"adjustedSchedule"`
		Summary          string              `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal validation result failed: %w", err)
	}

	// 转换issues中的affectedStaff
	issues := make([]*d_model.ValidationIssue, 0, len(rawResult.Issues))
	for _, issue := range rawResult.Issues {
		convertedStaff := make([]string, 0, len(issue.AffectedStaff))
		for _, name := range issue.AffectedStaff {
			if id, exists := nameMapping.NameToID[name]; exists {
				convertedStaff = append(convertedStaff, id)
			} else if id := s.fuzzyMatchName(name, nameMapping.NameToID); id != "" {
				convertedStaff = append(convertedStaff, id)
			}
		}
		issues = append(issues, &d_model.ValidationIssue{
			Type:          issue.Type,
			Severity:      issue.Severity,
			Description:   issue.Description,
			AffectedDates: issue.AffectedDates,
			AffectedStaff: convertedStaff,
		})
	}

	// 转换adjustedSchedule
	var adjustedSchedule map[string][]string
	if rawResult.AdjustedSchedule != nil {
		adjustedSchedule = make(map[string][]string)
		for date, names := range rawResult.AdjustedSchedule {
			staffIDs := make([]string, 0, len(names))
			for _, name := range names {
				if id, exists := nameMapping.NameToID[name]; exists {
					staffIDs = append(staffIDs, id)
				} else if id := s.fuzzyMatchName(name, nameMapping.NameToID); id != "" {
					staffIDs = append(staffIDs, id)
				}
			}
			adjustedSchedule[date] = staffIDs
		}
	}

	result := &d_model.ValidationResult{
		Passed:           rawResult.Passed,
		Issues:           issues,
		AdjustedSchedule: adjustedSchedule,
		Summary:          rawResult.Summary,
	}

	return result, nil
}

// ============================================================
// 姓名映射工具方法
// ============================================================

// buildStaffNameMapping 构建员工姓名映射
// 检测同名员工并生成唯一显示名，格式：姓名 或 姓名-(ID后4位)
func (s *schedulingAIService) buildStaffNameMapping(staffList []*d_model.StaffInfoForAI) *StaffNameMapping {
	mapping := &StaffNameMapping{
		NameToID:     make(map[string]string),
		IDToName:     make(map[string]string),
		DisplayNames: make([]string, 0, len(staffList)),
	}

	// 第一遍：统计同名情况
	nameCount := make(map[string]int)
	for _, staff := range staffList {
		if staff != nil && staff.Name != "" {
			nameCount[staff.Name]++
		}
	}

	// 第二遍：生成显示名并建立映射
	for _, staff := range staffList {
		if staff == nil {
			continue
		}

		// 生成显示名（同名时使用 ID 后 4 位区分）
		displayName := s.getDisplayName(staff.Name, staff.ID, nameCount[staff.Name] > 1)

		// 建立双向映射
		mapping.NameToID[displayName] = staff.ID
		mapping.IDToName[staff.ID] = displayName
		mapping.DisplayNames = append(mapping.DisplayNames, displayName)
	}

	return mapping
}

// buildStaffNameMappingWithAllStaff 构建员工姓名映射（使用 AllStaffList 确保姓名正确）
// 为所有在 allStaffList 中的员工建立映射，确保固定排班人员和已占位信息中的员工也能正确显示姓名
// buildStaffNameMappingWithAllStaff 构建员工姓名映射（使用 AllStaffList 确保姓名正确）
// allStaffList 已经包含所有员工，直接为所有员工建立 ID -> Name 映射
func (s *schedulingAIService) buildStaffNameMappingWithAllStaff(allStaffList []*d_model.Employee) *StaffNameMapping {
	// 第一遍：统计所有员工的同名情况
	nameCount := make(map[string]int)
	for _, emp := range allStaffList {
		if emp != nil && emp.Name != "" {
			nameCount[emp.Name]++
		}
	}

	// 第二遍：为所有员工建立映射
	mapping := &StaffNameMapping{
		NameToID:     make(map[string]string),
		IDToName:     make(map[string]string),
		DisplayNames: nil, // 不需要构建，使用时单独构建（只包含候选人员）
	}

	// 直接为 allStaffList 中的所有员工建立映射
	for _, emp := range allStaffList {
		if emp == nil || emp.ID == "" {
			continue
		}

		// 获取员工姓名
		staffName := emp.Name
		if staffName == "" {
			staffName = emp.ID // 如果没有姓名，使用 ID（不应该发生）
		}

		// 生成显示名（同名时使用 ID 后 4 位区分）
		displayName := s.getDisplayName(staffName, emp.ID, nameCount[staffName] > 1)

		// 建立双向映射
		mapping.NameToID[displayName] = emp.ID
		mapping.IDToName[emp.ID] = displayName
	}

	return mapping
}

// getDisplayName 生成员工显示名
// 同名时格式为：姓名-(ID后4位或全部)
func (s *schedulingAIService) getDisplayName(name, staffID string, hasDuplicate bool) string {
	if !hasDuplicate {
		return name
	}
	// 取 ID 后4位或全部（不足4位时）
	suffix := staffID
	if len(staffID) > 4 {
		suffix = staffID[len(staffID)-4:]
	}
	return fmt.Sprintf("%s-(%s)", name, suffix)
}

// convertScheduleNamesToIDs 将AI返回的schedule中的姓名转换为内部ID
func (s *schedulingAIService) convertScheduleNamesToIDs(schedule map[string]any, nameToID map[string]string) map[string]any {
	converted := make(map[string]any)
	for date, staffsAny := range schedule {
		var staffIDs []string
		switch staffs := staffsAny.(type) {
		case []any:
			staffIDs = make([]string, 0, len(staffs))
			for _, staffAny := range staffs {
				if staffName, ok := staffAny.(string); ok {
					if id, exists := nameToID[staffName]; exists {
						staffIDs = append(staffIDs, id)
					} else {
						// 尝试模糊匹配（处理AI可能省略后缀的情况）
						if id := s.fuzzyMatchName(staffName, nameToID); id != "" {
							staffIDs = append(staffIDs, id)
						} else {
							s.logger.Warn("Staff name not found in mapping", "name", staffName)
						}
					}
				}
			}
		case []string:
			staffIDs = make([]string, 0, len(staffs))
			for _, staffName := range staffs {
				if id, exists := nameToID[staffName]; exists {
					staffIDs = append(staffIDs, id)
				} else {
					if id := s.fuzzyMatchName(staffName, nameToID); id != "" {
						staffIDs = append(staffIDs, id)
					} else {
						s.logger.Warn("Staff name not found in mapping", "name", staffName)
					}
				}
			}
		}
		converted[date] = staffIDs
	}
	return converted
}

// convertStaffListNamesToIDs 将AI返回的员工姓名列表转换为内部ID列表
func (s *schedulingAIService) convertStaffListNamesToIDs(staffNames []any, nameToID map[string]string) []string {
	staffIDs := make([]string, 0, len(staffNames))
	for _, nameAny := range staffNames {
		if name, ok := nameAny.(string); ok {
			if id, exists := nameToID[name]; exists {
				staffIDs = append(staffIDs, id)
			} else if id := s.fuzzyMatchName(name, nameToID); id != "" {
				staffIDs = append(staffIDs, id)
			}
		}
	}
	return staffIDs
}

// fuzzyMatchName 模糊匹配姓名（处理AI可能省略工号后缀的情况）
func (s *schedulingAIService) fuzzyMatchName(inputName string, nameToID map[string]string) string {
	// 精确匹配
	if id, exists := nameToID[inputName]; exists {
		return id
	}
	// 尝试前缀匹配（输入"张三"匹配"张三-(0012)"）
	for displayName, id := range nameToID {
		if strings.HasPrefix(displayName, inputName+"-") {
			return id
		}
		// 也匹配"张三"在"张三-(0012)"中的情况
		if strings.Contains(displayName, inputName) && strings.Contains(displayName, "-(") {
			baseName := strings.Split(displayName, "-(")[0]
			if baseName == inputName {
				return id
			}
		}
	}
	return ""
}

// convertIDsToNames 将内部ID列表转换为显示名列表（用于提示词构建）
func (s *schedulingAIService) convertIDsToNames(staffIDs []string, idToName map[string]string) []string {
	names := make([]string, 0, len(staffIDs))
	for _, id := range staffIDs {
		if name, exists := idToName[id]; exists {
			names = append(names, name)
		}
	}
	return names
}

// ============================================================
// 三阶段排班AI方法实现
// ============================================================

// GenerateShiftTodoPlan AI生成单个班次的Todo计划
func (s *schedulingAIService) GenerateShiftTodoPlan(
	ctx context.Context,
	shiftInfo *d_model.ShiftInfo,
	staffList []*d_model.StaffInfoForAI,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	previousDraft *d_model.ShiftScheduleDraft,
	fixedShiftAssignments map[string][]string,
	temporaryNeeds []*d_model.PersonalNeed,
) (*d_model.TodoPlanResult, error) {
	s.logger.Info("Generating shift todo plan with AI")

	systemPrompt := s.buildTodoPlanSystemPrompt()
	userPrompt := s.buildTodoPlanUserPrompt(shiftInfo, staffList, rules, staffRequirements, previousDraft, fixedShiftAssignments, temporaryNeeds)

	// 获取配置的模型
	cfg := s.configurator.GetConfig()
	var todoPlanModel *common_config.AIModelProvider
	if cfg.SchedulingAI.TaskModels != nil {
		if model, ok := cfg.SchedulingAI.TaskModels[TaskModelTodoPlan]; ok && model.Provider != "" && model.Name != "" {
			todoPlanModel = &model
		}
	}

	resp, err := s.aiFactory.CallWithModel(ctx, todoPlanModel, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI todo plan generation failed: %w", err)
	}

	// 检查响应内容是否为空
	raw := strings.TrimSpace(resp.Content)
	if raw == "" {
		s.logger.Error("AI returned empty response for todo plan generation")
		return nil, fmt.Errorf("AI returned empty response - please check AI service configuration and model availability")
	}

	// 记录原始响应用于调试
	s.logger.Debug("AI todo plan raw response", "length", len(raw), "preview", raw[:min(200, len(raw))])

	// 解析结果
	result, parseErr := s.parseTodoPlanResult(raw)
	if parseErr != nil {
		s.logger.Error("Failed to parse todo plan JSON",
			"error", parseErr,
			"rawResponse", raw,
		)
		return nil, fmt.Errorf("parse todo plan json failed: %w (raw response: %s)", parseErr, raw)
	}

	// 验证解析结果的有效性
	if result == nil || len(result.Todos) == 0 {
		s.logger.Error("AI returned empty todo list")
		return nil, fmt.Errorf("AI generated empty todo list - please check prompts and model configuration")
	}

	return result, nil
}

// ExecuteTodoTask AI执行单个Todo任务
func (s *schedulingAIService) ExecuteTodoTask(
	ctx context.Context,
	todoTask *d_model.SchedulingTodo,
	shiftInfo *d_model.ShiftInfo,
	availableStaff []*d_model.StaffInfoForAI,
	currentDraft *d_model.ShiftScheduleDraft,
	staffRequirements map[string]int,
	fixedShiftAssignments map[string][]string,
	temporaryNeeds []*d_model.PersonalNeed,
	allStaffList []*d_model.Employee,
	// 新增参数：V3增强上下文
	allShifts []*d_model.Shift,
	workingDraft *d_model.ScheduleDraft,
) (*d_model.TodoExecutionResult, error) {
	s.logger.Info("Executing todo task with AI")

	// 构建姓名映射（用于AI交互）
	// 使用 AllStaffList 确保姓名映射完整，避免UUID显示
	var nameMapping *StaffNameMapping
	if len(allStaffList) > 0 {
		nameMapping = s.buildStaffNameMappingWithAllStaff(allStaffList)
	} else {
		// 如果没有 AllStaffList，回退到使用 availableStaff
		nameMapping = s.buildStaffNameMapping(availableStaff)
	}

	// 【V3增强】构建排班上下文
	var schedulingContext *d_model.V3SchedulingContext
	if len(allShifts) > 0 && workingDraft != nil && len(todoTask.TargetDates) > 0 {
		// 取第一个目标日期构建上下文
		targetDate := todoTask.TargetDates[0]

		// 找到目标班次
		var targetShift *d_model.Shift
		for _, shift := range allShifts {
			if shift != nil && shift.ID == shiftInfo.ShiftID {
				targetShift = shift
				break
			}
		}

		if targetShift != nil {
			// 获取该日期的需求人数
			requiredCount := staffRequirements[targetDate]

			// 转换availableStaff为Employee类型
			staffListForContext := make([]*d_model.Employee, 0)
			for _, staff := range availableStaff {
				if staff != nil {
					emp := &d_model.Employee{
						ID:   staff.ID,
						Name: staff.Name,
					}
					staffListForContext = append(staffListForContext, emp)
				}
			}

			// 构建V3排班上下文
			schedulingContext = buildSchedulingContextForTodo(
				targetDate,
				targetShift,
				requiredCount,
				workingDraft,
				staffListForContext,
				allStaffList,
				allShifts,
			)

			s.logger.Info("V3 scheduling context built",
				"date", targetDate,
				"shift", targetShift.Name,
				"staffSchedulesCount", len(schedulingContext.StaffSchedules))
		}
	}

	systemPrompt := s.buildTodoExecutionSystemPrompt()
	userPrompt := s.buildTodoExecutionUserPrompt(
		todoTask, shiftInfo, availableStaff, currentDraft, nameMapping,
		staffRequirements, fixedShiftAssignments, temporaryNeeds,
		allShifts, schedulingContext)

	s.logger.Info("Todo execution context",
		"todoTitle", todoTask.Title,
		"todoTargetDates", todoTask.TargetDates,
		"todoTargetCount", todoTask.TargetCount,
		"availableStaffCount", len(availableStaff),
		"currentDraftScheduleSize", len(currentDraft.Schedule),
		"staffRequirementsSize", len(staffRequirements),
		"hasV3Context", schedulingContext != nil)
	s.logger.Info("User Prompt for Todo Execution", "prompt", userPrompt)

	// 获取配置的模型
	cfg := s.configurator.GetConfig()
	var todoExecModel *common_config.AIModelProvider
	if cfg.SchedulingAI.TaskModels != nil {
		if model, ok := cfg.SchedulingAI.TaskModels[TaskModelTodoExecution]; ok && model.Provider != "" && model.Name != "" {
			todoExecModel = &model
		}
	}

	resp, err := s.aiFactory.CallWithModel(ctx, todoExecModel, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI todo execution failed: %w", err)
	}

	// 记录AI原始返回
	raw := strings.TrimSpace(resp.Content)
	s.logger.Info("AI raw response for todo execution", "rawContent", raw)

	// 解析结果
	result, parseErr := s.parseTodoExecutionResult(raw, nameMapping)
	if parseErr != nil {
		s.logger.Error("Failed to parse todo execution result JSON", "error", parseErr, "rawContent", raw)
		return nil, fmt.Errorf("parse todo execution result json failed: %w", parseErr)
	}

	return result, nil
}

// ValidateAndAdjustShiftSchedule AI校验并调整班次排班
func (s *schedulingAIService) ValidateAndAdjustShiftSchedule(
	ctx context.Context,
	shiftDraft *d_model.ShiftScheduleDraft,
	shiftInfo *d_model.ShiftInfo,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	staffList []*d_model.StaffInfoForAI,
	taskInfo *d_model.ProgressiveTask, // 新增：任务信息
) (*d_model.ValidationResult, error) {
	s.logger.Info("Validating and adjusting shift schedule with AI")

	// 构建姓名映射（用于AI交互）
	nameMapping := s.buildStaffNameMapping(staffList)

	systemPrompt := s.buildValidationSystemPrompt()
	userPrompt := s.buildValidationUserPrompt(shiftDraft, shiftInfo, rules, staffRequirements, nameMapping, taskInfo)

	// 获取配置的模型
	cfg := s.configurator.GetConfig()
	var validationModel *common_config.AIModelProvider
	if cfg.SchedulingAI.TaskModels != nil {
		if model, ok := cfg.SchedulingAI.TaskModels[TaskModelValidation]; ok && model.Provider != "" && model.Name != "" {
			validationModel = &model
		}
	}

	resp, err := s.aiFactory.CallWithModel(ctx, validationModel, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI validation failed: %w", err)
	}

	// 解析结果
	raw := strings.TrimSpace(resp.Content)
	result, parseErr := s.parseValidationResult(raw, nameMapping)
	if parseErr != nil {
		s.logger.Error("Failed to parse validation result JSON", "error", parseErr)
		return nil, fmt.Errorf("parse validation result json failed: %w", parseErr)
	}

	return result, nil
}

// ============================================================
// 三阶段排班提示词构建
// ============================================================

// buildTodoPlanSystemPrompt 构建Todo计划生成的系统提示词
func (s *schedulingAIService) buildTodoPlanSystemPrompt() string {
	return `你是一个专业的排班规划助手。你的任务是为单个班次制定**简洁高效**的排班Todo计划。

**核心原则**：
- 保持任务数量精简（通常2-4个任务，最多不超过5个）
- 合并相似的任务，避免按日期逐一拆分
- 优先处理强约束规则（如固定班、特殊分组）
- **description分两部分**：先概括任务，再附加完整规则原文

**⚠️ 关键要求 - 是否排班由班次要求人数决定**：
- **最重要**：是否排班完全由"每日人数需求"决定，而不是规则
- 如果某日期的班次要求人数 > 0，则该日期需要排班（无论是工作日还是周末）
- 如果某日期的班次要求人数 = 0，则该日期不需要排班（即使规则没有排除该日期）
- targetDates必须只包含"每日人数需求"中人数 > 0 的日期
- 规则只用于约束如何排班（如哪些人员可以排、排班方式等），不用于决定是否需要排班

请返回 JSON 格式，包含以下字段：
- todos: 有序的任务列表（1-5个），每个任务包含：
  - id: 任务ID（从1开始递增）
  - title: 任务标题（简明扼要，如"安排固定班人员"）
  - description: **任务概述 + 完整规则原文**（第一部分概括任务目标和执行策略，第二部分逐条列出所有相关规则的完整内容）
  - priority: 优先级（1-高，2-中，3-低）
  - targetDates: 涉及的日期列表（**必须只包含"每日人数需求"中人数 > 0 的日期，规则只用于约束如何排班**）
  - targetStaffCount: 目标安排人数（多个日期可累加总数）
- summary: 整体规划说明（简要阐述排班策略）
- reasoning: AI的思考过程（为什么这样拆分任务）

**description编写规范**（非常重要）：
第一部分 - 任务概述（2-3句话）：
- 说明这个任务要做什么
- 涉及哪些人员或分组
- **明确说明规则限定的工作日（如"仅周一至周五"）**

第二部分 - 完整规则原文（必须）：
- 用分隔线"\n\n---相关规则详情---\n"分隔
- 逐条列出所有相关规则，格式：规则N: [名称] - 规则内容: [ruleData完整内容]
- 如果有10个规则，就完整列出10条，不能省略
- 如果规则中有人员名单、分组、排除条件等，必须原封不动复制

**日期计算示例**：
排班周期：2025-12-01（周一）至 2025-12-07（周日）
每日人数需求：
- 2025-12-01: 10人
- 2025-12-02: 10人
- 2025-12-03: 10人
- 2025-12-04: 10人
- 2025-12-05: 10人
- 2025-12-06: 0人  ← 周六，人数为0，不需要排班
- 2025-12-07: 0人  ← 周日，人数为0，不需要排班

正确的targetDates: ["2025-12-01", "2025-12-02", "2025-12-03", "2025-12-04", "2025-12-05"]  ✅ 只包含人数 > 0 的日期
错误的targetDates: ["2025-12-01", "2025-12-02", "2025-12-03", "2025-12-04", "2025-12-05", "2025-12-06", "2025-12-07"] ❌ 不能包含人数为0的日期

**重要**：即使规则没有排除周六周日，如果"每日人数需求"中周六周日的人数为0，则不需要排班。

示例（正确的description写法）：
{
  "todos": [
    {
      "id": 1,
      "title": "安排固定班人员",
      "description": "需要将10名固定班人员安排到排班周期内的工作日（**仅周一至周五**，不含周六周日）。根据"每日人数需求"，只有周一至周五的人数 > 0，周六周日的人数为0，因此只安排周一到周五。根据规则，这些人员只在周一到周五上班，必须严格遵守。\n\n---相关规则详情---\n规则1: 固定班-张三 - 规则内容: 张三固定周一、周二、周三、周四、周五上（XX班）的班\n规则2: 固定班-李四 - 规则内容: 李四固定周一、周二、周三、周四、周五上（XX班）的班",
      "priority": 1,
      "targetDates": ["2025-12-01", "2025-12-02", "2025-12-03", "2025-12-04", "2025-12-05"],
      "targetStaffCount": 50
    }
  ],
  "summary": "根据"每日人数需求"分析，只有周一至周五的人数 > 0（共5天，每天10人），周六周日的人数为0，因此只安排周一到周五。规则用于约束如何排班（固定班人员只在周一到周五上班）。",
  "reasoning": "每日人数需求中，周一至周五的人数 > 0，周六周日的人数为0，所以targetDates只能包含周一到周五的日期。规则只用于约束排班方式，不用于决定是否需要排班。"
}

**错误示例**：
{
  "description": "安排到2025-12-01至2025-12-07的所有日期（周一至周日）..." ❌ 
  // 错误：即使规则没有排除周六周日，如果"每日人数需求"中周六周日的人数为0，则不需要排班！
  // 正确做法：只包含"每日人数需求"中人数 > 0 的日期
}

请只返回 JSON 对象，不要包含其他内容。`
}

// buildTodoExecutionSystemPrompt 构建Todo执行的系统提示词
func (s *schedulingAIService) buildTodoExecutionSystemPrompt() string {
	return `你是一个专业的排班执行助手。你的任务是根据Todo任务说明，为指定日期安排合适的人员。

**排班流程说明**：
这是排班流程的执行阶段。你正在执行一个具体的排班任务。
前置流程：
1. 需求评估：系统已分析所有需求（规则、个人需求、固定排班等），并生成了任务计划
2. 任务规划：当前任务是从任务计划中提取的，包含明确的目标和范围
3. 不可用人员过滤：系统已从负向需求中构建了不可用人员清单，这些人员已在可用人员列表中被自动排除
当前阶段：
4. 任务执行（当前）：你需要根据任务说明，从可用人员列表中选择合适的人员进行排班
后续流程：
5. 规则校验：系统会自动校验你的排班结果是否符合规则

**⚠️ 任务拆分说明**：
- 任务按业务逻辑组织（如"正向需求填充"、"特殊班次填充"、"剩余人员填充"等），系统会在执行阶段自动将多班次任务拆分成单班次子任务
- **当前任务**是业务任务的一个子任务，只涉及一个班次，但需理解整体业务目标（从任务描述中获取）

**核心职责**：
- 必须从可用人员列表中选择人员（使用姓名）
- 必须满足targetStaffCount的人数要求
- **必须严格遵循规则中指定的日期限制**
- **⚠️ 必须严格遵守每日人数上限，这是硬性限制**

**⚠️ 关键要求 - 每日人数上限（硬性限制，最高优先级）**：
- 每日安排人数**不得超过**【每日人数需求】中指定的上限
- 例如：某天需求为3人，则最多安排3人，可以少于3人但绝不能超过3人
- 如果规则要求安排某人，但会导致当天人数超限，则**不安排该人员**
- 人数上限优先于其他规则，这是硬性约束
- 如果当天已有排班人数接近上限，只能安排剩余名额内的人员

**⚠️ 关键要求 - 规则日期限制必须严格遵守**：
- 如果规则说"周二、周四、周五上班"，则该人员**只能**安排在周二、周四、周五
- 即使targetDates包含周一、周三、周六、周日，也**不能**在这些天安排该人员
- 每个人员的排班日期必须与其对应规则中指定的星期几完全匹配
- 不同人员可能有不同的日期限制，需要分别处理

**⚠️ 关键要求 - 任务范围必须严格遵守**：
- 必须仔细阅读任务说明（Description），明确本次任务的具体范围
- **重要**：固定排班已在任务执行前100%完成，所有固定排班人员已写入排班表并标记为占用
- **如果任务说明是"只安排固定班人员"**：这是历史遗留的任务类型，固定排班已完成，此类任务应返回空的schedule {}，不需要处理
- **如果任务说明是"补充剩余人员"**：只排非固定人员，不要重复排固定人员（固定人员已在排班表中）
- 严格按照任务说明的范围执行，不要超出任务范围

**⚠️ 关键要求 - 执行前必须检查需求**：
- 在执行排班前，必须先检查每个目标日期的当前已排班人数是否已满足需求
- 如果已满足需求，则不需要再安排人员，在schedule中不要包含该日期（或返回空数组）
- 如果所有目标日期都已满足需求，则返回空的schedule {}，不要复现当前已排班
- 如果未满足需求，则只安排需要补充的人数，不要超过需求人数

**⚠️ 关键要求 - 明确排班操作类型（新增重要功能）**：
- 对于每个日期的排班，你必须在 scheduleActions 中明确指定操作类型
- 操作类型有两种：
  1. **"append"（追加）**：将人员追加到该日期的已有排班中（用于新增排班）
  2. **"replace"（替换）**：完全替换该日期的已有排班（用于修正错误）
- 如何选择操作类型：
  - 如果【当前已排班】中该日期**没有排班**或排班**正确**，使用 "append"
  - 如果【当前已排班】中该日期**存在违规人员**需要修正，使用 "replace"
- scheduleActions 格式：{"日期": "操作类型"}

**日期匹配示例**：
规则: "员工A固定周二、周四、周五上（某班次）的班"
targetDates: ["2025-12-01(周一)", "2025-12-02(周二)", "2025-12-03(周三)", "2025-12-04(周四)", "2025-12-05(周五)"]
正确做法: 员工A只安排在 2025-12-02(周二)、2025-12-04(周四)、2025-12-05(周五)
错误做法: 员工A安排在所有5天 ❌（违反规则，周一周三不应安排）

**操作类型选择示例**：
场景1 - 新增排班（使用 append）：
【当前已排班】: 2025-12-02 已排["员工乙"]
本次任务: 为 2025-12-02 再安排员工丙、员工丁
返回: 
  schedule: {"2025-12-02": ["员工丙", "员工丁"]}
  scheduleActions: {"2025-12-02": "append"}
结果: 2025-12-02 最终为 ["员工乙", "员工丙", "员工丁"]

场景2 - 修正错误（使用 replace）：
【当前已排班】: 2025-12-01(周一) 已排["员工A", "员工乙"]
规则显示: 员工A固定周二、周四、周五上班（周一不应排班）
本次任务: 修正 2025-12-01 的错误
返回:
  schedule: {"2025-12-01": ["员工乙", "员工丙"]}
  scheduleActions: {"2025-12-01": "replace"}
结果: 2025-12-01 最终为 ["员工乙", "员工丙"]（移除了违规的员工A）

场景3 - 混合操作：
【当前已排班】: 
  2025-12-01 已排["员工A", "员工乙"]（员工A违规）
  2025-12-02 已排["员工丙"]（正确）
本次任务: 修正 12-01 的错误，并为 12-02 增加人员
返回:
  schedule: {
    "2025-12-01": ["员工乙", "员工丁"],
    "2025-12-02": ["员工戊", "员工己"]
  }
  scheduleActions: {
    "2025-12-01": "replace",
    "2025-12-02": "append"
  }

**人数上限示例**：
每日人数上限: 2025-12-02 最多3人
当前已排班: 2025-12-02 已有2人
本次可再安排: 最多1人
如果规则要求安排2人，但只剩1个名额，则只安排1人，并在remainingIssues中说明

**严禁行为**：
- 不得以"缺少信息"为借口拒绝排班
- 不得将targetStaffCount设为0作为借口
- **不得忽略规则中的日期限制（如"周二、周四、周五"）**
- **不得在规则不允许的日期安排人员**
- **不得超过每日人数上限（硬性限制）**
- **不得忽视【当前已排班】中的违规情况**
- **不得遗漏 scheduleActions 字段（必须为每个日期指定操作类型）**
- **注意**：如果所有目标日期都已满足需求，可以返回空的schedule对象（{}），这是允许的

请返回 JSON 格式，包含以下字段：
- schedule: 本次任务的排班安排，格式为 {日期: [人员姓名列表]}
- scheduleActions: **必须提供**，格式为 {日期: 操作类型}
  - 操作类型只能是 "append" 或 "replace"
  - 必须为 schedule 中的每个日期都指定操作类型
- updatedStaff: 本次任务中被安排的人员姓名列表
- explanation: 执行说明（**必须说明每个人员被安排在哪些天，以及为什么；如果使用了 replace 操作也要说明原因**）
- remainingIssues: 遗留问题（如果有无法完全满足的情况，如人数超限导致无法安排）

示例1（新增排班 - 使用 append）：
{
  "schedule": {
    "2025-12-02": ["员工甲", "员工乙"],
    "2025-12-04": ["员工甲", "员工丙"],
    "2025-12-05": ["员工甲", "员工丁"]
  },
  "scheduleActions": {
    "2025-12-02": "append",
    "2025-12-04": "append",
    "2025-12-05": "append"
  },
  "updatedStaff": ["员工甲", "员工乙", "员工丙", "员工丁"],
  "explanation": "根据规则，员工甲只在周二(12-02)、周四(12-04)、周五(12-05)排班，严格遵循'周二、周四、周五'的限制。所有日期使用追加模式（append），不影响已有排班。",
  "remainingIssues": []
}

示例2（修正错误 - 使用 replace）：
{
  "schedule": {
    "2025-12-01": ["员工乙", "员工丙"],
    "2025-12-03": ["员工丁", "员工戊"]
  },
  "scheduleActions": {
    "2025-12-01": "replace",
    "2025-12-03": "replace"
  },
  "updatedStaff": ["员工乙", "员工丙", "员工丁", "员工戊"],
  "explanation": "发现【当前已排班】中员工甲在12-01和12-03违反规则（应只在周二、周四、周五上班），使用替换模式（replace）将其从这两天移除，并用符合规则的员工替换。",
  "remainingIssues": []
}

示例3（混合操作）：
{
  "schedule": {
    "2025-12-01": ["员工乙", "员工丙"],
    "2025-12-02": ["员工丁"]
  },
  "scheduleActions": {
    "2025-12-01": "replace",
    "2025-12-02": "append"
  },
  "updatedStaff": ["员工乙", "员工丙", "员工丁"],
  "explanation": "12-01存在违规人员，使用replace修正；12-02排班正确，使用append追加新人员。",
  "remainingIssues": []
}

注意：
1. 日期格式必须是 YYYY-MM-DD
2. 人员姓名必须**完全匹配**可用人员列表中的姓名（包括工号后缀，如"员工A-(0012)"）
3. **scheduleActions 是必需字段**，必须为每个日期指定 "append" 或 "replace"
4. **每个人员只能安排在其规则允许的日期**
5. 如果规则说"周二、周四、周五"，则周一、周三、周六、周日不能安排该人员
6. **每日安排人数不得超过人数需求上限（硬性限制，最高优先级）**
7. **如果发现【当前已排班】违规，使用 replace 修正**

请只返回 JSON 对象，不要包含其他内容。`
}

// buildValidationSystemPrompt 构建校验的系统提示词
func (s *schedulingAIService) buildValidationSystemPrompt() string {
	return `你是一个专业的排班质量检查助手（LLMQC）。你的任务是根据本次排班任务的目标和规范，综合判定排班结果的质量。

**核心原则**：
1. **理解任务目标**：仔细阅读任务描述，明确本次任务要完成什么（如"固定排班填充"、"个人需求填充"、"剩余人员填充"等）
2. **针对性校验**：根据任务类型进行有针对性的校验
   - 固定排班配置任务：主要检查固定人员是否按配置正确填充，不需要校验是否填满所有需求
   - 个人需求填充任务：主要检查个人需求是否得到满足
   - 全员填充任务：需要检查人数是否满足要求、规则是否遵守
3. **综合判定**：综合考虑任务目标、规则要求、人数需求等因素，给出合理的评价
4. **避免过度严格**：如果任务本身就是部分填充（如只填充固定人员），不应将"未填满"视为问题

**任务类型识别**（通过任务描述判断）：
- 如果任务描述包含"固定排班"、"固定班次"、"配置填充"等关键词 → 固定排班配置任务
- 如果任务描述包含"个人需求"、"请假"、"调休"等关键词 → 个人需求填充任务
- 如果任务描述包含"剩余人员"、"补充"、"全员"等关键词 → 全员填充任务

**针对性校验标准**：
固定排班配置任务：
  ✓ 检查：固定人员是否按配置正确填充
  ✓ 检查：固定人员的排班日期是否符合配置规则
  ✗ 不检查：是否填满所有需求人数（这不是本任务的目标）
  ✗ 不检查：是否有人员缺岗（后续任务会补充）

个人需求填充任务：
  ✓ 检查：个人需求（请假、调休等）是否得到满足
  ✓ 检查：是否存在需求冲突
  ✗ 不检查：是否填满所有需求人数（除非任务明确要求）

全员填充任务：
  ✓ 检查：人数是否满足要求
  ✓ 检查：是否存在规则违反
  ✓ 检查：是否存在负荷不均

**重要：仅输出错误项，不要输出警告**
- severity 必须为 "critical"，表示必须修正的错误
- 不要输出 medium/low 级别的问题
- 不要输出优化建议类的问题
- 只输出影响排班正确性的严重错误

请返回 JSON 格式，包含以下字段：
- passed: 是否通过校验（true/false）
- issues: 问题列表，每个问题包含：
  - type: 问题类型（rule_violation/requirement_mismatch）
  - severity: 必须为 "critical"（只输出严重错误）
  - description: 问题描述
  - affectedDates: 受影响的日期
  - affectedStaff: 受影响的人员姓名列表
- adjustedSchedule: 调整后的排班（如果需要调整），格式为 {日期: [人员姓名列表]}
- summary: 校验总结（整体评价和建议）

示例：
{
  "passed": false,
  "issues": [
    {
      "type": "rule_violation",
      "severity": "critical",
      "description": "员工张三连续工作4天，违反最大连续3天规则",
      "affectedDates": ["2025-11-25", "2025-11-26", "2025-11-27", "2025-11-28"],
      "affectedStaff": ["张三"]
    }
  ],
  "adjustedSchedule": {
    "2025-11-28": ["李四", "王五"]
  },
  "summary": "发现1处规则违反，已进行调整。"
}

请只返回 JSON 对象，不要包含其他内容。`
}

// buildTodoPlanUserPrompt 构建Todo计划的用户提示词
func (s *schedulingAIService) buildTodoPlanUserPrompt(
	shiftInfo *d_model.ShiftInfo,
	staffList []*d_model.StaffInfoForAI,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	previousDraft *d_model.ShiftScheduleDraft,
	fixedShiftAssignments map[string][]string,
	temporaryNeeds []*d_model.PersonalNeed,
) string {
	var sb strings.Builder

	// 1. 班次基本信息
	sb.WriteString("【班次信息】\n")
	if shiftInfo != nil {
		if shiftInfo.ShiftName != "" {
			sb.WriteString(fmt.Sprintf("班次名称: %s\n", shiftInfo.ShiftName))
		}
		if shiftInfo.StartTime != "" {
			sb.WriteString(fmt.Sprintf("班次时间: %s", shiftInfo.StartTime))
			if shiftInfo.EndTime != "" {
				sb.WriteString(fmt.Sprintf(" - %s", shiftInfo.EndTime))
			}
			sb.WriteString("\n")
		}
		if shiftInfo.StartDate != "" {
			sb.WriteString(fmt.Sprintf("排班周期: %s", shiftInfo.StartDate))
			if shiftInfo.EndDate != "" {
				sb.WriteString(fmt.Sprintf(" 至 %s", shiftInfo.EndDate))
			}
			sb.WriteString("\n")
		}
	}

	// 2. 人数需求
	if len(staffRequirements) > 0 {
		sb.WriteString("\n【每日人数需求】\n")
		sb.WriteString("**⚠️ 重要：是否排班完全由人数需求决定**\n")
		sb.WriteString("- 如果某日期的人数 > 0，则该日期需要排班（无论是工作日还是周末）\n")
		sb.WriteString("- 如果某日期的人数 = 0，则该日期不需要排班（即使规则没有排除该日期）\n")
		sb.WriteString("- targetDates必须只包含人数 > 0 的日期\n\n")
		dates := make([]string, 0, len(staffRequirements))
		for date := range staffRequirements {
			dates = append(dates, date)
		}
		// 简单排序
		for i := 0; i < len(dates); i++ {
			for j := i + 1; j < len(dates); j++ {
				if dates[i] > dates[j] {
					dates[i], dates[j] = dates[j], dates[i]
				}
			}
		}
		for _, date := range dates {
			count := staffRequirements[date]
			if count > 0 {
				sb.WriteString(fmt.Sprintf("- %s: %d人 ✅ 需要排班\n", date, count))
			} else {
				sb.WriteString(fmt.Sprintf("- %s: %d人 ❌ 不需要排班（人数为0）\n", date, count))
			}
		}
	} else {
		sb.WriteString("\n【每日人数需求】\n")
		sb.WriteString("⚠️ 警告：没有提供人数需求，无法确定需要排班的日期\n")
	}

	// 3. 适用规则
	sb.WriteString("\n【适用规则】\n")
	if len(rules) > 0 {
		for i, rule := range rules {
			if rule == nil {
				continue
			}
			// 优先使用 Condition（详细规则内容），其次使用 Description
			ruleContent := ""
			if rule.Condition != "" {
				ruleContent = rule.Condition
			} else if rule.Description != "" {
				ruleContent = rule.Description
			}
			sb.WriteString(fmt.Sprintf("%d. %s", i+1, rule.Name))
			if ruleContent != "" {
				sb.WriteString(fmt.Sprintf(" - 规则内容: %s", ruleContent))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("无特殊规则\n")
	}

	// 4. 可用人员统计
	sb.WriteString(fmt.Sprintf("\n【可用人员】\n共 %d 人可用于本班次排班\n", len(staffList)))

	// 5. 之前班次排班情况（简要）
	if previousDraft != nil {
		sb.WriteString("\n【已有排班】\n之前班次已安排部分人员，请注意避免冲突和过度劳累\n")
	}

	// 6. 固定排班人员（如果存在）
	if len(fixedShiftAssignments) > 0 {
		sb.WriteString("\n【固定排班人员（必须包含）】\n")
		sb.WriteString("**⚠️ 重要说明**：以下日期中列出的**所有人员**都已经在固定班次中安排，**必须包含在最终排班中**。\n")
		sb.WriteString("**总需求人数已包含固定人员**，你需要确保这些固定人员都在排班结果中，然后补充其他人员以满足总需求人数。\n\n")
		dates := make([]string, 0, len(fixedShiftAssignments))
		for date := range fixedShiftAssignments {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := fixedShiftAssignments[date]
			reqCount := staffRequirements[date]
			sb.WriteString(fmt.Sprintf("- **%s**：固定人员 %d 人，总需求 %d 人（固定人员已包含在总需求中）\n", date, len(staffIDs), reqCount))
		}
		sb.WriteString("\n**操作要求**：\n")
		sb.WriteString("1. 固定人员必须包含在最终排班中，不能遗漏\n")
		sb.WriteString("2. 总人数必须等于需求人数（需求人数已包含固定人员）\n")
		sb.WriteString("3. 如果需求是16人，固定人员是3人，你需要安排这3个固定人员+另外13人=总共16人\n\n")
	}

	// 7. 临时需求（如果存在）
	if len(temporaryNeeds) > 0 {
		sb.WriteString("\n【临时需求（必须遵守）】\n")
		sb.WriteString("以下临时需求必须遵守，确保不会安排已出差、请假或有事的员工：\n")
		for i, need := range temporaryNeeds {
			if need == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d. %s", i+1, need.StaffName))
			if need.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", need.Description))
			}
			if len(need.TargetDates) > 0 {
				sb.WriteString(fmt.Sprintf(" (日期: %s)", strings.Join(need.TargetDates, ", ")))
			}
			if need.RequestType == "avoid" {
				sb.WriteString(" [不能排班]")
			} else if need.RequestType == "must" {
				sb.WriteString(" [必须排班]")
			}
			sb.WriteString("\n")
		}
	}

	// 8. 任务要求
	sb.WriteString("\n【任务要求】\n")
	sb.WriteString("请为本班次制定**简洁高效**的排班Todo计划，要求：\n")
	sb.WriteString("1. 分析规则和人数需求，合并相似的任务，避免过度拆分\n")
	sb.WriteString("2. 优先级高的规则（如固定班人员、特殊分组）单独成任务\n")
	sb.WriteString("3. 常规日期的人员安排可以合并为一个任务（如工作日、周末）\n")
	sb.WriteString("4. 通常1-4个任务即可覆盖整个排班，不要超过5个任务\n")
	sb.WriteString("5. 每个任务目标明确，可独立执行\n")
	sb.WriteString("\n【⚠️ 关键要求 - 日期必须根据人数需求决定】\n")
	sb.WriteString("- **最重要**：是否排班完全由\"每日人数需求\"决定，而不是规则\n")
	sb.WriteString("- 如果某日期的人数 > 0，则该日期需要排班（无论是工作日还是周末）\n")
	sb.WriteString("- 如果某日期的人数 = 0，则该日期不需要排班（即使规则没有排除该日期）\n")
	sb.WriteString("- targetDates必须只包含\"每日人数需求\"中人数 > 0 的日期\n")
	sb.WriteString("- 规则只用于约束如何排班（如哪些人员可以排、排班方式等），不用于决定是否需要排班\n")
	sb.WriteString("- **不要假设\"排班周期内的所有日期\"都需要排班**，要根据人数需求判断\n")
	sb.WriteString("\n【关键要求 - description格式：概述+规则原文】\n")
	sb.WriteString("- 每个TODO的description字段分为**两部分**：\n")
	sb.WriteString("\n  第一部分 - 任务概述（2-3句话）：\n")
	sb.WriteString("  • 说明任务目标和执行策略\n")
	sb.WriteString("  • 列出涉及的关键人员或分组\n")
	sb.WriteString("  • **明确说明规则限定的工作日（如\"仅周一至周五\"）**\n")
	sb.WriteString("\n  第二部分 - 完整规则原文（必须包含）：\n")
	sb.WriteString("  • 用分隔线\"\\n\\n---相关规则详情---\\n\"分隔两部分\n")
	sb.WriteString("  • 规则格式：\"规则N: [规则名称] - 规则内容: [ruleData完整内容]\"\n")
	sb.WriteString("  • 逐条列出所有相关规则，不能省略或用范围概括\n")
	sb.WriteString("  • 规则内容必须原封不动复制，包括所有人员名单、分组、排除条件等\n")
	sb.WriteString("\n- 示例description（正确 - 概述准确描述规则限定的日期）：\n")
	sb.WriteString("  \"需要将10名固定班人员安排到排班周期内的**工作日（仅周一至周五）**。根据规则，这些人员只在周一到周五上班，周六周日不排班。\n")
	sb.WriteString("   \n")
	sb.WriteString("   ---相关规则详情---\n")
	sb.WriteString("   规则1: 固定班-张三 - 规则内容: 张三固定周一、周二、周三、周四、周五上（XX班）的班\n")
	sb.WriteString("   规则2: 固定班-李四 - 规则内容: 李四固定周一、周二、周三、周四、周五上（XX班）的班\n")
	sb.WriteString("\n- 示例description（错误 - 日期范围超出规则限定）：\n")
	sb.WriteString("  \"安排到2025-12-01至2025-12-07的所有日期（周一至周日）...\" ❌ 规则说的是周一到周五，不能擅自扩展到周六周日！\n")
	sb.WriteString("\n【重要提醒】\n")
	sb.WriteString("- 每个todo任务**必须**包含targetDates（具体日期列表）\n")
	sb.WriteString("- targetDates**只能包含规则允许的日期**，不是排班周期内的所有日期\n")
	sb.WriteString("- targetDates格式：[\"YYYY-MM-DD\", \"YYYY-MM-DD\", ...]，根据规则从排班周期中筛选\n")
	sb.WriteString("- targetStaffCount是这些日期的总人数需求（多个日期可累加）\n")
	sb.WriteString("\n【日期筛选示例】\n")
	sb.WriteString("排班周期: 2025-12-01(周一) 至 2025-12-07(周日)\n")
	sb.WriteString("规则: \"张三固定周一、周二、周三、周四、周五上班\"\n")
	sb.WriteString("正确targetDates: [\"2025-12-01\", \"2025-12-02\", \"2025-12-03\", \"2025-12-04\", \"2025-12-05\"]\n")
	sb.WriteString("错误targetDates: [\"2025-12-01\"...\"2025-12-07\"] ❌ 不应包含周六周日\n")

	return sb.String()
}

// buildTodoExecutionUserPrompt 构建Todo执行的用户提示词（V3增强版）
// 注意：此方法需要配合 buildStaffNameMapping 生成的映射使用
func (s *schedulingAIService) buildTodoExecutionUserPrompt(
	todoTask *d_model.SchedulingTodo,
	shiftInfo *d_model.ShiftInfo,
	availableStaff []*d_model.StaffInfoForAI,
	currentDraft *d_model.ShiftScheduleDraft,
	nameMapping *StaffNameMapping,
	staffRequirements map[string]int,
	fixedShiftAssignments map[string][]string,
	temporaryNeeds []*d_model.PersonalNeed,
	allShifts []*d_model.Shift,
	schedulingContext *d_model.V3SchedulingContext,
) string {
	var sb strings.Builder

	// 1. 当前任务信息
	sb.WriteString("【当前任务】\n")
	if todoTask != nil {
		if todoTask.Title != "" {
			sb.WriteString(fmt.Sprintf("任务: %s\n", todoTask.Title))
		}
		if todoTask.Description != "" {
			sb.WriteString(fmt.Sprintf("说明: %s\n", todoTask.Description))
		}
		if len(todoTask.TargetDates) > 0 {
			sb.WriteString("目标日期: ")
			sb.WriteString(strings.Join(todoTask.TargetDates, ", "))
			sb.WriteString("\n")
		}
		if todoTask.TargetCount > 0 {
			sb.WriteString(fmt.Sprintf("目标人数: %d人\n", todoTask.TargetCount))
		}
	}

	// ============================================================
	// 【V3增强】添加强制约束（置顶显示）
	// ============================================================
	if schedulingContext != nil {
		sb.WriteString("\n【⚠️ 强制约束（违反则拒绝）】\n")
		sb.WriteString(fmt.Sprintf("- ⏰ 当日每人最多工作 %.1f 小时（硬限制，超过则违规）\n", schedulingContext.MaxDailyHours))
		sb.WriteString("- ❌ 严格禁止时间重叠：不得将同一人安排到时间冲突的班次\n")
		sb.WriteString("- 🔒 标记为❌的人员（已超时或已冲突）绝对不可安排\n\n")
	}

	// ============================================================
	// 【V3增强】添加班次时间表（智能截断优化）
	// ============================================================
	if len(allShifts) > 0 {
		sb.WriteString("\n【📋 班次时间表】\n")
		sb.WriteString("所有班次的时间安排如下，请注意避免时间冲突：\n\n")

		// 智能截断：优先显示当前班次和有时间重叠风险的班次
		maxShiftsToShow := 10 // 降低显示数量
		if len(allShifts) > 50 {
			maxShiftsToShow = 8 // 班次过多时进一步压缩
		}

		shownCount := 0
		for i, shift := range allShifts {
			if shownCount >= maxShiftsToShow {
				sb.WriteString(fmt.Sprintf("... 还有 %d 个班次（时间冲突检测已在后端完成）\n", len(allShifts)-shownCount))
				break
			}
			// 优先显示当前处理的班次
			if schedulingContext != nil && shift.ID == schedulingContext.TargetShiftID {
				// 当前班次，必须显示
			} else if shownCount >= maxShiftsToShow-1 && i < len(allShifts)-1 {
				// 已达到显示上限，跳过剩余班次
				continue
			}

			duration := float64(shift.Duration) / 60.0
			if shift.Duration == 0 && shift.StartTime != "" && shift.EndTime != "" {
				// 降级：根据时间计算
				duration = calculateShiftDurationHours(shift)
			}

			timeStr := fmt.Sprintf("%s-%s (%.1f小时)", shift.StartTime, shift.EndTime, duration)
			if shift.IsOvernight {
				timeStr += " [跨夜]"
			}

			sb.WriteString(fmt.Sprintf("- %s: %s\n", shift.Name, timeStr))
		}
		sb.WriteString("\n")
	}

	// ============================================================
	// 【V3增强】添加人员当日排班状态
	// ============================================================
	if schedulingContext != nil && len(schedulingContext.StaffSchedules) > 0 {
		sb.WriteString(fmt.Sprintf("\n【👥 人员当日排班状态】（%s）\n", schedulingContext.TargetDate))
		sb.WriteString("以下人员在目标日期的排班情况，请根据此信息避免冲突：\n\n")

		// 分为已排班和未排班两组
		scheduledStaff := make([]*d_model.StaffCurrentSchedule, 0)
		unscheduledStaff := make([]*d_model.StaffCurrentSchedule, 0)

		for _, schedule := range schedulingContext.StaffSchedules {
			if len(schedule.Shifts) > 0 || schedule.TotalHours > 0 {
				scheduledStaff = append(scheduledStaff, schedule)
			} else {
				unscheduledStaff = append(unscheduledStaff, schedule)
			}
		}

		// 显示已排班人员（详细信息）- 优化截断策略
		if len(scheduledStaff) > 0 {
			sb.WriteString("【已有排班人员】\n")
			// 动态调整显示数量：人员越多，显示越少
			maxDisplay := 20
			if len(scheduledStaff) > 100 {
				maxDisplay = 10 // 人员过多时大幅压缩
			} else if len(scheduledStaff) > 50 {
				maxDisplay = 15
			}

			// 优先显示有错误的人员
			errorStaff := make([]*d_model.StaffCurrentSchedule, 0)
			normalStaff := make([]*d_model.StaffCurrentSchedule, 0)
			for _, schedule := range scheduledStaff {
				if len(schedule.Errors) > 0 {
					errorStaff = append(errorStaff, schedule)
				} else {
					normalStaff = append(normalStaff, schedule)
				}
			}

			displayedCount := 0
			// 先显示有错误的人员（这些是LLM必须知道的）
			for _, schedule := range errorStaff {
				if displayedCount >= maxDisplay {
					break
				}
				s.formatStaffScheduleInfo(&sb, schedule)
				displayedCount++
			}

			// 再显示部分正常人员
			for i, schedule := range normalStaff {
				if displayedCount >= maxDisplay {
					sb.WriteString(fmt.Sprintf("... 还有 %d 人已排班（无错误）\n", len(normalStaff)-i))
					break
				}
				s.formatStaffScheduleInfo(&sb, schedule)
				displayedCount++
			}

			if displayedCount == 0 && len(scheduledStaff) > 0 {
				// 确保至少显示了一些信息
				for i, schedule := range scheduledStaff {
					if i >= maxDisplay {
						sb.WriteString(fmt.Sprintf("... 还有 %d 人已排班\n", len(scheduledStaff)-maxDisplay))
						break
					}
					s.formatStaffScheduleInfo(&sb, schedule)
				}
			}
			sb.WriteString("\n")
		}

		// 显示未排班人员（简要列表）- 进一步压缩
		if len(unscheduledStaff) > 0 {
			sb.WriteString("【未排班人员】✅ 优先考虑\n")
			// 动态调整显示数量
			maxUnscheduled := 30
			if len(unscheduledStaff) > 100 {
				maxUnscheduled = 15 // 人员过多时大幅压缩
			} else if len(unscheduledStaff) > 50 {
				maxUnscheduled = 20
			}

			unscheduledNames := make([]string, 0)
			for i, schedule := range unscheduledStaff {
				if i >= maxUnscheduled {
					unscheduledNames = append(unscheduledNames, fmt.Sprintf("... 还有 %d 人", len(unscheduledStaff)-maxUnscheduled))
					break
				}
				unscheduledNames = append(unscheduledNames, schedule.StaffName)
			}
			sb.WriteString(fmt.Sprintf("%s\n\n", strings.Join(unscheduledNames, ", ")))
		}
	}

	// 1.5. ⚠️ 任务范围说明（必须严格遵守）
	sb.WriteString("\n【⚠️ 任务范围说明（必须严格遵守）】\n")
	sb.WriteString("**仔细阅读任务说明（Description），明确本次任务的具体范围**：\n")
	sb.WriteString("- 如果任务说明是\"补充剩余人员\"或\"补充其他人员\"，则**只补充人员**，不要重复排已有人员\n")
	sb.WriteString("- 如果任务说明是\"安排所有人员\"，则安排总人数等于需求人数\n")
	sb.WriteString("- **关键**：严格按照任务说明的范围执行，不要超出任务范围\n")
	// 2. ⚠️ 每日人数需求（硬性上限）
	if len(staffRequirements) > 0 {
		sb.WriteString("\n【⚠️ 每日人数需求（硬性上限）】\n")
		sb.WriteString("以下是每日安排人数的**上限**，绝对不能超过：\n")
		dates := make([]string, 0, len(staffRequirements))
		for date := range staffRequirements {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			sb.WriteString(fmt.Sprintf("- %s: 最多 %d 人\n", date, staffRequirements[date]))
		}
	}

	// 3. 班次信息
	sb.WriteString("\n【班次信息】\n")
	if shiftInfo != nil {
		if shiftInfo.ShiftCode != "" {
			sb.WriteString(fmt.Sprintf("班次代码: %s\n", shiftInfo.ShiftCode))
		}
		if shiftInfo.ShiftName != "" {
			sb.WriteString(fmt.Sprintf("班次名称: %s\n", shiftInfo.ShiftName))
		}
		if shiftInfo.StartTime != "" {
			sb.WriteString(fmt.Sprintf("时间: %s - %s\n", shiftInfo.StartTime, shiftInfo.EndTime))
		}
		if shiftInfo.StartDate != "" {
			sb.WriteString(fmt.Sprintf("排班周期: %s 至 %s\n", shiftInfo.StartDate, shiftInfo.EndDate))
		}
	}

	// 4. 可用人员（显示姓名和已排班标记）
	sb.WriteString(fmt.Sprintf("\n【可用人员】共 %d 人\n", len(availableStaff)))
	for i, staff := range availableStaff {
		if staff == nil {
			continue
		}
		// 获取显示名
		var displayName string
		if nameMapping != nil {
			if name, exists := nameMapping.IDToName[staff.ID]; exists {
				displayName = name
			}
		}
		// 如果映射中没有，直接使用name字段
		if displayName == "" {
			displayName = staff.Name
		}

		sb.WriteString(fmt.Sprintf("%d. %s", i+1, displayName))

		// 显示分组信息（重要：用于规则匹配）
		if len(staff.Groups) > 0 {
			sb.WriteString(fmt.Sprintf(" [分组: %s]", strings.Join(staff.Groups, ", ")))
		}

		// 显示已排班标记信息（如果有）
		if len(staff.ScheduledShifts) > 0 {
			sb.WriteString(" [已排班: ")
			markStrs := make([]string, 0)
			// 按日期排序
			dates := make([]string, 0, len(staff.ScheduledShifts))
			for date := range staff.ScheduledShifts {
				dates = append(dates, date)
			}
			sort.Strings(dates)
			for _, date := range dates {
				marks := staff.ScheduledShifts[date]
				for _, mark := range marks {
					// 格式: 12-01 上午班(08:00-12:00)
					dateShort := date[5:] // 去掉年份，只保留 MM-DD
					markStrs = append(markStrs, fmt.Sprintf("%s %s(%s-%s)", dateShort, mark.ShiftName, mark.StartTime, mark.EndTime))
				}
			}
			sb.WriteString(strings.Join(markStrs, ", "))
			sb.WriteString("]")
		}
		sb.WriteString("\n")
	}

	// 4. 排班约束（重要：告知AI时段冲突规则）
	sb.WriteString("\n【排班约束】\n")
	sb.WriteString("**重要说明**：人员的[已排班]信息表示该人员在其他班次（或其他日期）的排班情况，这**不影响**该人员在**当前班次未排班的日期**的安排。\n\n")
	sb.WriteString("**时段冲突规则（仅限同一天）**：\n")
	sb.WriteString("1. **同一天**时间段重叠的班次不可重复安排同一人（如已排08:00-12:00，不可再排08:00-12:00）\n")
	sb.WriteString("2. **同一天**时间段不重叠的班次可以安排同一人（如上午班+下午班）\n")
	sb.WriteString("3. 建议：晚班最好与下午班搭配安排\n")
	sb.WriteString("4. 建议：避免一人一天安排超过2个班次\n\n")
	sb.WriteString("**关键理解**：\n")
	sb.WriteString("- 如果人员在2025-12-01已有其他班次排班，这**不影响**他在2025-12-02或其他日期安排当前班次\n")
	sb.WriteString("- [已排班]信息用于判断**同一天**是否有时段冲突，**不是**用来排除该人员\n")
	sb.WriteString("- 只要目标日期没有时段冲突，就可以安排该人员\n")

	// 5. 当前排班草案状态（显示姓名而非ID）
	// 5. 当前已排班情况
	if currentDraft != nil && len(currentDraft.Schedule) > 0 {
		sb.WriteString("\n【当前已排班】\n")
		sb.WriteString("**重要说明**：当前已排班只包含动态排班数据（之前任务的执行结果），不包含固定排班。固定排班已100%完成，你不需要处理。\n")
		// 按日期排序
		dates := make([]string, 0, len(currentDraft.Schedule))
		for date := range currentDraft.Schedule {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := currentDraft.Schedule[date]
			sb.WriteString(fmt.Sprintf("- %s: ", date))
			// 将ID转换为姓名显示
			if nameMapping != nil {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				if len(names) > 0 {
					sb.WriteString(strings.Join(names, ", "))
				} else {
					sb.WriteString(fmt.Sprintf("%d人", len(staffIDs)))
				}
			} else {
				sb.WriteString(fmt.Sprintf("%d人", len(staffIDs)))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("\n【当前已排班】\n")
		sb.WriteString("当前没有已排班，可以自由安排人员。\n")
	}

	// 5.5. ⚠️ 需求检查（执行前必须检查）
	sb.WriteString("\n【⚠️ 需求检查（执行前必须检查）】\n")
	sb.WriteString("在执行排班前，**必须先检查**每个目标日期的当前已排班人数是否已满足需求：\n")
	sb.WriteString("- 如果某日期的当前已排班人数**已经等于或超过需求人数**，则：\n")
	sb.WriteString("  * 该日期**不需要再安排人员**\n")
	sb.WriteString("  * 在schedule中**不要包含该日期**（或返回空数组）\n")
	sb.WriteString("  * 在explanation中说明：\"XX日期已满足需求（当前X人，需求X人），无需补充\"\n")
	sb.WriteString("- 如果某日期的当前已排班人数**小于需求人数**，则：\n")
	sb.WriteString("  * 计算需要补充的人数 = 需求人数 - 当前已排班人数\n")
	sb.WriteString("  * 只安排需要补充的人数，不要超过需求人数\n")
	sb.WriteString("- **关键**：不要盲目安排，先检查再执行\n")
	sb.WriteString("- **特别重要**：如果所有目标日期都已满足需求，则返回空的schedule，不要复现当前已排班\n")

	// 6. 固定排班信息
	if len(fixedShiftAssignments) > 0 {
		sb.WriteString("\n【固定排班信息】\n")
		sb.WriteString("以下人员已被固定排班占用，你**不需要**在结果中包含这些人员：\n\n")
		dates := make([]string, 0, len(fixedShiftAssignments))
		for date := range fixedShiftAssignments {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := fixedShiftAssignments[date]
			reqCount := staffRequirements[date]
			dynamicNeeded := reqCount - len(staffIDs)
			// 将ID转换为姓名显示
			if nameMapping != nil {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				if len(names) > 0 {
					sb.WriteString(fmt.Sprintf("- %s: 固定%d人（%s），你需安排%d人\n", date, len(names), strings.Join(names, ", "), dynamicNeeded))
				} else {
					sb.WriteString(fmt.Sprintf("- %s: 固定%d人，你需安排%d人\n", date, len(staffIDs), dynamicNeeded))
				}
			} else {
				sb.WriteString(fmt.Sprintf("- %s: 固定%d人，你需安排%d人\n", date, len(staffIDs), dynamicNeeded))
			}
		}
		sb.WriteString("\n")
		sb.WriteString("**说明**：\n")
		sb.WriteString("- 固定人员已完成排班，你只需安排剩余的动态人数\n")
		sb.WriteString("- 不要在返回结果中包含固定排班人员\n\n")
	}

	// 7. 临时需求（如果存在）
	if len(temporaryNeeds) > 0 {
		sb.WriteString("\n【临时需求（必须遵守）】\n")
		sb.WriteString("以下临时需求必须遵守，确保不会安排已出差、请假或有事的员工：\n")
		for i, need := range temporaryNeeds {
			if need == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d. %s", i+1, need.StaffName))
			if need.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", need.Description))
			}
			if len(need.TargetDates) > 0 {
				sb.WriteString(fmt.Sprintf(" (日期: %s)", strings.Join(need.TargetDates, ", ")))
			}
			if need.RequestType == "avoid" {
				sb.WriteString(" [不能排班]")
			} else if need.RequestType == "must" {
				sb.WriteString(" [必须排班]")
			}
			sb.WriteString("\n")
		}
	}

	// 8. 执行要求
	sb.WriteString("\n【执行要求】\n")
	sb.WriteString("请根据任务说明，为指定日期安排合适的人员：\n")
	sb.WriteString("1. **首先检查**：每个目标日期的当前已排班人数是否已满足需求\n")
	sb.WriteString("2. **严格遵守任务范围**：只安排任务说明中指定的人员类型（固定人员/其他人员/所有人员）\n")
	sb.WriteString("3. **不要重复安排**：如果固定人员已在当前已排班中，不要重复安排\n")
	sb.WriteString("4. **不要超量安排**：如果已满足需求，不要继续补充，返回空的schedule\n")
	sb.WriteString("5. **如果所有日期都已满足需求**：返回空的schedule {}，不要复现当前已排班\n")
	sb.WriteString("6. 必须使用可用人员列表中的**完整姓名**（包含工号后缀，如有）\n")
	sb.WriteString("7. 如有无法满足的情况，请在remainingIssues中说明\n")
	sb.WriteString("\n【重要提醒】\n")
	sb.WriteString("- schedule格式：{\"YYYY-MM-DD\": [\"姓名1\", \"姓名2\", ...]}\n")
	sb.WriteString("- 姓名必须**完全匹配**上面的可用人员列表（区分大小写和后缀）\n")
	sb.WriteString("- 必须为targetDates中的**每个日期**安排人员\n")
	sb.WriteString("- 如果任务说明中有明确的人员名单，必须从可用人员中匹配并安排\n")
	sb.WriteString("- updatedStaff必须包含本次安排的所有人员姓名（去重）\n")

	return sb.String()
}

// buildValidationUserPrompt 构建校验的用户提示词
func (s *schedulingAIService) buildValidationUserPrompt(
	shiftDraft *d_model.ShiftScheduleDraft,
	shiftInfo *d_model.ShiftInfo,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	nameMapping *StaffNameMapping,
	taskInfo *d_model.ProgressiveTask, // 新增：任务信息
) string {
	var sb strings.Builder

	// 0. 任务信息（最重要，放在最前面）
	if taskInfo != nil {
		sb.WriteString("【本次任务目标】\n")
		sb.WriteString("**⚠️ 重要：请根据任务目标进行针对性校验，不要过度严格**\n\n")

		if taskInfo.Title != "" {
			sb.WriteString(fmt.Sprintf("任务标题: %s\n", taskInfo.Title))
		}

		if taskInfo.Description != "" {
			sb.WriteString(fmt.Sprintf("任务描述: %s\n", taskInfo.Description))
		}

		// 根据任务标题和描述判断任务类型
		taskType := "未知"
		taskTypeLower := strings.ToLower(taskInfo.Title + " " + taskInfo.Description)
		if strings.Contains(taskTypeLower, "固定排班") || strings.Contains(taskTypeLower, "固定班次") || strings.Contains(taskTypeLower, "配置填充") || strings.Contains(taskTypeLower, "fixed") {
			taskType = "固定排班配置任务"
			sb.WriteString("\n**任务类型**: 固定排班配置任务\n")
			sb.WriteString("**校验重点**：\n")
			sb.WriteString("✓ 检查：固定人员是否按配置正确填充\n")
			sb.WriteString("✓ 检查：固定人员的排班日期是否符合配置规则\n")
			sb.WriteString("✗ **不检查**：是否填满所有需求人数（这不是本任务的目标）\n")
			sb.WriteString("✗ **不检查**：是否有人员缺岗（后续任务会补充）\n")
		} else if strings.Contains(taskTypeLower, "个人需求") || strings.Contains(taskTypeLower, "请假") || strings.Contains(taskTypeLower, "调休") {
			taskType = "个人需求填充任务"
			sb.WriteString("\n**任务类型**: 个人需求填充任务\n")
			sb.WriteString("**校验重点**：\n")
			sb.WriteString("✓ 检查：个人需求（请假、调休等）是否得到满足\n")
			sb.WriteString("✓ 检查：是否存在需求冲突\n")
			sb.WriteString("✗ **不检查**：是否填满所有需求人数（除非任务明确要求）\n")
		} else if strings.Contains(taskTypeLower, "剩余人员") || strings.Contains(taskTypeLower, "补充") || strings.Contains(taskTypeLower, "全员") {
			taskType = "全员填充任务"
			sb.WriteString("\n**任务类型**: 全员填充任务\n")
			sb.WriteString("**校验重点**：\n")
			sb.WriteString("✓ 检查：人数是否满足要求\n")
			sb.WriteString("✓ 检查：是否存在规则违反\n")
			sb.WriteString("✓ 检查：是否存在负载不均\n")
		} else {
			sb.WriteString(fmt.Sprintf("\n**任务类型**: %s\n", taskType))
			sb.WriteString("**校验重点**：根据任务描述综合判定\n")
		}
		sb.WriteString("\n")
	}

	// 1. 班次信息
	sb.WriteString("【班次信息】\n")
	if shiftInfo != nil {
		if shiftInfo.ShiftCode != "" {
			sb.WriteString(fmt.Sprintf("班次代码: %s\n", shiftInfo.ShiftCode))
		}
		if shiftInfo.ShiftName != "" {
			sb.WriteString(fmt.Sprintf("班次名称: %s\n", shiftInfo.ShiftName))
		}
	}

	// 2. 人数需求
	if len(staffRequirements) > 0 {
		sb.WriteString("\n【人数需求】\n")
		dates := make([]string, 0, len(staffRequirements))
		for date := range staffRequirements {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			sb.WriteString(fmt.Sprintf("- %s: 需要%d人\n", date, staffRequirements[date]))
		}
	}

	// 3. 适用规则
	sb.WriteString("\n【适用规则】\n")
	if len(rules) > 0 {
		for i, rule := range rules {
			if rule == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("%d. %s", i+1, rule.Name))
			if rule.Description != "" {
				sb.WriteString(fmt.Sprintf(" - %s", rule.Description))
			}
			sb.WriteString("\n")
		}
	}

	// 4. 实际排班结果（将ID转换为姓名展示）
	sb.WriteString("\n【实际排班结果】\n")
	if shiftDraft != nil && len(shiftDraft.Schedule) > 0 {
		dates := make([]string, 0, len(shiftDraft.Schedule))
		for date := range shiftDraft.Schedule {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := shiftDraft.Schedule[date]
			sb.WriteString(fmt.Sprintf("- %s: ", date))
			if nameMapping != nil {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				sb.WriteString(strings.Join(names, ", "))
			} else {
				sb.WriteString(fmt.Sprintf("%d人", len(staffIDs)))
			}
			sb.WriteString("\n")
		}
	}

	// 5. 校验要求
	sb.WriteString("\n【校验要求】\n")
	sb.WriteString("请全面检查排班结果，包括：\n")
	sb.WriteString("1. 人数需求是否满足（每日实际人数与需求人数对比）\n")
	sb.WriteString("2. 是否违反任何规则约束\n")
	sb.WriteString("3. 人员工作负荷是否均衡\n")
	sb.WriteString("4. 是否有优化空间\n")
	sb.WriteString("\n如发现问题，请在adjustedSchedule中使用**完整姓名**（与上面相同）提供调整方案。\n")

	return sb.String()
}

// ============================================================
// 排班调整工作流 AI 方法
// ============================================================

// AnalyzeAdjustIntent 分析用户的排班调整意图
func (s *schedulingAIService) AnalyzeAdjustIntent(ctx context.Context, userInput string, messages []session.Message) (*d_model.AdjustIntent, error) {
	s.logger.Info("Analyzing adjust intent", "userInput", userInput)

	// 获取 scheduleAdjust 策略配置
	cfg := s.configurator.GetConfig()
	strategy, ok := cfg.Intent.Strategies["scheduleAdjust"]
	if !ok {
		s.logger.Warn("scheduleAdjust strategy not found, using default prompt")
		return s.analyzeAdjustIntentWithDefaultPrompt(ctx, userInput, messages)
	}

	// 构建系统提示词，替换日期占位符
	systemPrompt := strings.TrimSpace(strategy.SystemPrompt)
	if systemPrompt == "" {
		return s.analyzeAdjustIntentWithDefaultPrompt(ctx, userInput, messages)
	}

	// 替换 {currentDate} 占位符
	currentDate := s.getCurrentDateString()
	systemPrompt = strings.ReplaceAll(systemPrompt, "{currentDate}", currentDate)

	// 构建用户提示词
	userPrompt := s.buildAdjustIntentUserPrompt(userInput, messages)

	// 选择 AI 模型
	var model *common_config.AIModelProvider
	if strategy.Model != nil && strategy.Model.Provider != "" && strategy.Model.Name != "" {
		model = strategy.Model
	}

	// 调用 AI
	resp, err := s.aiFactory.CallWithModel(ctx, model, systemPrompt, userPrompt, nil)
	if err != nil {
		s.logger.Error("AI call failed for adjust intent", "error", err)
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	s.logger.Debug("AI response received for adjust intent", "contentLength", len(resp.Content))

	// 解析 AI 返回结果
	return s.parseAdjustIntentResponse(resp.Content, userInput)
}

// getCurrentDateString 获取当前日期字符串
func (s *schedulingAIService) getCurrentDateString() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
}

// buildAdjustIntentUserPrompt 构建调整意图分析的用户提示词
func (s *schedulingAIService) buildAdjustIntentUserPrompt(userInput string, messages []session.Message) string {
	var sb strings.Builder

	// 添加对话历史（如果有）
	if len(messages) > 0 {
		sb.WriteString("## 对话历史：\n")
		for i, msg := range messages {
			role := "用户"
			switch msg.Role {
			case session.RoleAssistant:
				role = "助手"
			case session.RoleSystem:
				role = "系统"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, role, msg.Content))
		}
		sb.WriteString("\n")
	}

	// 添加当前用户输入
	sb.WriteString("## 当前用户输入：\n")
	sb.WriteString(userInput)
	sb.WriteString("\n\n请分析用户的调整意图。")

	return sb.String()
}

// parseAdjustIntentResponse 解析 AI 返回的调整意图
func (s *schedulingAIService) parseAdjustIntentResponse(rawContent string, userInput string) (*d_model.AdjustIntent, error) {
	// 提取 JSON
	jsonStr := s.extractJSON(rawContent)

	// 尝试解析为数组
	var results []struct {
		Type       string         `json:"type"`
		Confidence float64        `json:"confidence"`
		Summary    string         `json:"summary"`
		Arguments  map[string]any `json:"arguments"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		s.logger.Error("Failed to parse adjust intent JSON", "error", err, "raw", rawContent)
		// 返回一个默认的 other 类型意图
		return &d_model.AdjustIntent{
			Type:    d_model.AdjustIntentOther,
			RawText: userInput,
		}, nil
	}

	if len(results) == 0 {
		return &d_model.AdjustIntent{
			Type:    d_model.AdjustIntentOther,
			RawText: userInput,
		}, nil
	}

	// 取置信度最高的结果
	var best = results[0]
	for _, r := range results[1:] {
		if r.Confidence > best.Confidence {
			best = r
		}
	}

	// 转换为 AdjustIntent
	intent := &d_model.AdjustIntent{
		Type:           s.parseAdjustIntentType(best.Type),
		Confidence:     best.Confidence,
		RawDescription: best.Summary,
		RawText:        userInput,
	}

	// 提取参数
	if best.Arguments != nil {
		if date, ok := best.Arguments["date"].(string); ok {
			intent.Date = date
			intent.TargetDates = []string{date}
		}
		if staffA, ok := best.Arguments["staffA"].(string); ok {
			intent.StaffA = staffA
			intent.TargetStaff = []string{staffA}
		}
		if oldStaff, ok := best.Arguments["oldStaff"].(string); ok {
			intent.StaffA = oldStaff
			intent.TargetStaff = []string{oldStaff}
		}
		if staffB, ok := best.Arguments["staffB"].(string); ok {
			intent.StaffB = staffB
			intent.ReplacementStaff = []string{staffB}
		}
		if newStaff, ok := best.Arguments["newStaff"].(string); ok {
			intent.StaffB = newStaff
			intent.ReplacementStaff = []string{newStaff}
		}
		if staff, ok := best.Arguments["staff"].(string); ok {
			if intent.StaffA == "" {
				intent.StaffA = staff
			}
			intent.TargetStaff = []string{staff}
		}
		if reason, ok := best.Arguments["reason"].(string); ok {
			intent.Reason = reason
		}
		if desc, ok := best.Arguments["description"].(string); ok {
			if intent.RawDescription == "" {
				intent.RawDescription = desc
			}
		}
	}

	s.logger.Info("Adjust intent parsed",
		"type", intent.Type,
		"confidence", intent.Confidence,
		"date", intent.Date,
		"staffA", intent.StaffA,
		"staffB", intent.StaffB)

	return intent, nil
}

// AdjustShiftSchedule 直接根据用户需求调整排班
func (s *schedulingAIService) AdjustShiftSchedule(
	ctx context.Context,
	userRequirement string,
	originalDraft *d_model.ShiftScheduleDraft,
	shiftInfo *d_model.ShiftInfo,
	staffList []*d_model.StaffInfoForAI,
	allStaffList []*d_model.Employee,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	existingScheduleMarks map[string]map[string]bool,
	fixedShiftAssignments map[string][]string,
) (*d_model.AdjustScheduleResult, error) {
	s.logger.Info("Adjusting shift schedule with AI", "shiftID", shiftInfo.ShiftID, "userRequirement", userRequirement)

	// 构建姓名映射（使用 allStaffList 确保姓名正确）
	nameMapping := s.buildStaffNameMappingWithAllStaff(allStaffList)

	// 构建提示词
	systemPrompt := s.buildAdjustScheduleSystemPrompt()
	userPrompt := s.buildAdjustScheduleUserPrompt(
		userRequirement,
		originalDraft,
		shiftInfo,
		staffList,
		rules,
		staffRequirements,
		existingScheduleMarks,
		fixedShiftAssignments,
		nameMapping,
	)

	// 获取配置的模型（使用 scheduleDraft 模型）
	cfg := s.configurator.GetConfig()
	var adjustModel *common_config.AIModelProvider
	if cfg.SchedulingAI.TaskModels != nil {
		if model, ok := cfg.SchedulingAI.TaskModels[TaskModelScheduleDraft]; ok && model.Provider != "" && model.Name != "" {
			adjustModel = &model
		}
	}
	s.logger.Info("Adjust schedule context",
		"systemPrompt", systemPrompt,
		"userPrompt", userPrompt,
		"adjustModel", adjustModel)
	// 调用 AI
	resp, err := s.aiFactory.CallWithModel(ctx, adjustModel, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI adjustment failed: %w", err)
	}

	// 解析结果
	raw := strings.TrimSpace(resp.Content)
	resultMap, err := s.parseAdjustScheduleResultJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parse adjust schedule result failed: %w", err)
	}

	// 提取 summary
	summary := ""
	if summaryRaw, ok := resultMap["summary"]; ok {
		if summaryStr, ok := summaryRaw.(string); ok {
			summary = summaryStr
		}
	}

	// 转换为 ShiftScheduleDraft（AI返回的排班，可能只包含修改的日期）
	aiDraft, err := s.convertScheduleMapToDraft(resultMap, nameMapping)
	if err != nil {
		return nil, fmt.Errorf("convert schedule to draft failed: %w", err)
	}

	// 记录AI返回的排班信息（用于调试）
	s.logger.Info("AdjustShiftSchedule: AI draft parsed",
		"aiDraftDates", len(aiDraft.Schedule),
		"originalDraftDates", func() int {
			if originalDraft != nil {
				return len(originalDraft.Schedule)
			}
			return 0
		}())

	// 合并原始排班和AI返回的排班，确保包含所有日期
	finalDraft := d_model.NewShiftScheduleDraft()
	// 1. 先复制原始排班的所有日期
	if originalDraft != nil {
		for date, staffIDs := range originalDraft.Schedule {
			finalDraft.Schedule[date] = append([]string{}, staffIDs...)
		}
		s.logger.Info("AdjustShiftSchedule: Original draft copied",
			"originalDates", len(originalDraft.Schedule))
	}
	// 2. 用AI返回的排班覆盖（只覆盖AI返回的日期）
	aiDatesCount := 0
	for date, staffIDs := range aiDraft.Schedule {
		originalCount := 0
		if originalDraft != nil {
			if originalStaffIDs, exists := originalDraft.Schedule[date]; exists {
				originalCount = len(originalStaffIDs)
			}
		}
		finalDraft.Schedule[date] = append([]string{}, staffIDs...)
		aiDatesCount++
		s.logger.Info("AdjustShiftSchedule: AI draft date merged",
			"date", date,
			"originalCount", originalCount,
			"aiCount", len(staffIDs),
			"finalCount", len(finalDraft.Schedule[date]))
	}
	s.logger.Info("AdjustShiftSchedule: Merge completed",
		"aiDatesCount", aiDatesCount,
		"finalDatesCount", len(finalDraft.Schedule))

	// 记录最终排班信息
	s.logger.Info("AdjustShiftSchedule: Final draft prepared",
		"finalDraftDates", len(finalDraft.Schedule),
		"summaryLength", len(summary))

	// 第二步：调用AI根据调整说明生成变化列表（确保一致性）
	changes, err := s.generateChangesFromSummary(ctx, summary, originalDraft, finalDraft, nameMapping, adjustModel)
	if err != nil {
		s.logger.Warn("AdjustShiftSchedule: Failed to generate changes from AI, using calculated changes",
			"error", err)
		// 如果AI生成失败，回退到计算方式
		changes = s.calculateScheduleChanges(originalDraft, finalDraft)
	}

	s.logger.Info("AdjustShiftSchedule: Changes generated",
		"changesCount", len(changes))

	return &d_model.AdjustScheduleResult{
		Draft:   finalDraft,
		Summary: summary,
		Changes: changes,
	}, nil
}

// buildAdjustScheduleSystemPrompt 构建调整排班的系统提示词
func (s *schedulingAIService) buildAdjustScheduleSystemPrompt() string {
	return `你是排班调整助手。根据用户需求调整排班，保留其他正确排班。

**核心原则**：
1. **用户需求是最高优先级**：准确理解并响应用户需求
2. **最小化修改**：只调整用户要求的部分，保留其他排班
3. **遵守约束**：人数要求、规则约束、固定人员、冲突避免

**用户需求理解**：
- "安排到X以后" = 从原日期移除，新增到X及之后（包括X）
- "无法排班"/"不能排班" = 从所有日期移除
- "X以后"/"X及以后"/"X之后"都包括X本身

**输出格式**：
{
  "schedule": {"YYYY-MM-DD": ["人员姓名", ...], ...},
  "summary": "调整说明：..."
}

**summary 要求（重要）**：
1. **必须首先说明如何响应用户需求**（如"根据用户需求'张三周二有行政班，需要出外勤，把他的班次调整到周四以后'，将张三从周二移除，并在周四及之后新增"）
2. **然后列出具体变化**（按日期顺序，如"2025-12-30移除张三，2026-01-01、2026-01-02新增张三"）
3. **必须与schedule完全一致**，不能出现描述与实际不符
4. **调整说明必须解释如何调整用户需求**，不能只说调整了哪些人员，没有解释如何调整的

**重要约束**：
- 固定排班人员：不能移除，不能用于新增（只能从候选人员列表新增）
- 每日人数：必须等于需求人数（除非用户明确要求改变）
- 规则约束：必须遵守
- 冲突避免：不能与已占位人员冲突

**输出要求（重要）**：
- **schedule必须包含原始排班的所有日期**，即使某个日期不需要修改，也要包含该日期及其完整人员列表
- **每日人数必须严格等于需求人数**（除非用户明确要求改变）
- 使用完整人员姓名（与列表完全匹配）
- 日期格式：YYYY-MM-DD
- 只返回JSON，不要其他内容

**关键提醒**：
- 如果某个日期需要修改，返回修改后的完整人员列表（人数必须等于需求人数）
- 如果某个日期不需要修改，也要返回该日期的完整人员列表（保持原样）
- **不要只返回修改的日期，必须返回所有日期**`
}

// buildAdjustScheduleUserPrompt 构建调整排班的用户提示词
func (s *schedulingAIService) buildAdjustScheduleUserPrompt(
	userRequirement string,
	originalDraft *d_model.ShiftScheduleDraft,
	shiftInfo *d_model.ShiftInfo,
	staffList []*d_model.StaffInfoForAI,
	rules []*d_model.RuleInfo,
	staffRequirements map[string]int,
	existingScheduleMarks map[string]map[string]bool,
	fixedShiftAssignments map[string][]string,
	nameMapping *StaffNameMapping,
) string {
	var sb strings.Builder

	// 1. 用户需求（最优先，放在最前面）
	sb.WriteString("【用户调整需求】\n")
	sb.WriteString(userRequirement)
	sb.WriteString("\n\n")

	// 2. 班次信息
	sb.WriteString("【班次信息】\n")
	if shiftInfo != nil {
		if shiftInfo.ShiftName != "" {
			sb.WriteString(fmt.Sprintf("班次名称: %s\n", shiftInfo.ShiftName))
		}
		if shiftInfo.StartTime != "" {
			sb.WriteString(fmt.Sprintf("班次时间: %s", shiftInfo.StartTime))
			if shiftInfo.EndTime != "" {
				sb.WriteString(fmt.Sprintf(" - %s", shiftInfo.EndTime))
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	// 3. 原始排班
	sb.WriteString("【原始排班（需要在此基础上调整）】\n")
	if originalDraft != nil && len(originalDraft.Schedule) > 0 {
		dates := make([]string, 0, len(originalDraft.Schedule))
		for date := range originalDraft.Schedule {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := originalDraft.Schedule[date]
			sb.WriteString(fmt.Sprintf("%s: ", date))
			if len(staffIDs) > 0 {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				sb.WriteString(strings.Join(names, ", "))
			} else {
				sb.WriteString("无")
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("暂无排班\n")
	}
	sb.WriteString("\n")

	// 3. 每日人数需求
	if len(staffRequirements) > 0 {
		sb.WriteString("【每日人数需求】\n")
		sb.WriteString("每日安排人数必须等于需求人数（除非用户明确要求改变）。\n")
		dates := make([]string, 0, len(staffRequirements))
		for date := range staffRequirements {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			count := staffRequirements[date]
			if count > 0 {
				sb.WriteString(fmt.Sprintf("%s: %d人\n", date, count))
			} else {
				sb.WriteString(fmt.Sprintf("%s: 0人\n", date))
			}
		}
		sb.WriteString("\n")
	}

	// 4. 候选人员（用于新增）
	if len(staffList) > 0 {
		sb.WriteString("【候选人员（用于新增）】\n")
		sb.WriteString("只能从此列表选择新增人员：\n")
		for _, staff := range staffList {
			if staff != nil {
				displayName := staff.ID
				if nameMapping != nil {
					if mappedName, ok := nameMapping.IDToName[staff.ID]; ok {
						displayName = mappedName
					}
				}
				if displayName == staff.ID && staff.Name != "" {
					displayName = staff.Name
				}
				sb.WriteString(fmt.Sprintf("- %s\n", displayName))
			}
		}
		sb.WriteString("\n")
	}

	// 5. 固定排班人员（不能移除，不能用于新增）
	if len(fixedShiftAssignments) > 0 {
		sb.WriteString("【固定排班人员（必须保留，不能用于新增）】\n")
		dates := make([]string, 0, len(fixedShiftAssignments))
		for date := range fixedShiftAssignments {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := fixedShiftAssignments[date]
			if len(staffIDs) > 0 {
				sb.WriteString(fmt.Sprintf("%s: ", date))
				names := make([]string, 0, len(staffIDs))
				for _, staffID := range staffIDs {
					if nameMapping != nil {
						if name, ok := nameMapping.IDToName[staffID]; ok {
							names = append(names, name)
						} else {
							names = append(names, staffID)
						}
					} else {
						names = append(names, staffID)
					}
				}
				sb.WriteString(strings.Join(names, ", "))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// 6. 排班规则
	if len(rules) > 0 {
		sb.WriteString("【排班规则】\n")
		for i, rule := range rules {
			if rule != nil {
				sb.WriteString(fmt.Sprintf("规则%d: %s", i+1, rule.Name))
				if rule.Condition != "" {
					sb.WriteString(fmt.Sprintf(" - %s", rule.Condition))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// 7. 已占位人员（避免冲突）
	if len(existingScheduleMarks) > 0 {
		sb.WriteString("【已占位人员（不能安排）】\n")
		dates := make([]string, 0)
		for date := range existingScheduleMarks {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := existingScheduleMarks[date]
			if len(staffIDs) > 0 {
				sb.WriteString(fmt.Sprintf("%s: ", date))
				names := make([]string, 0)
				for staffID := range staffIDs {
					if staffIDs[staffID] {
						if nameMapping != nil {
							if name, ok := nameMapping.IDToName[staffID]; ok {
								names = append(names, name)
							} else {
								names = append(names, staffID)
							}
						} else {
							names = append(names, staffID)
						}
					}
				}
				sb.WriteString(strings.Join(names, ", "))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// 8. 输出要求（强化）
	sb.WriteString("【输出要求（必须严格遵守）】\n")
	sb.WriteString("1. **schedule必须包含原始排班的所有日期**，即使某个日期不需要修改，也要包含该日期及其完整人员列表\n")
	sb.WriteString("2. **每日人数必须严格等于需求人数**：每个日期的安排人数必须等于【每日人数需求】中该日期的需求人数\n")
	sb.WriteString("3. **必须包含固定人员**：如果某个日期有【固定排班人员】，这些人员必须包含在schedule中\n")
	sb.WriteString("4. **不要遗漏日期**：不要只返回修改的日期，必须返回所有日期\n")
	sb.WriteString("5. **不要重复人员**：每个日期的人员列表不能有重复\n")
	sb.WriteString("6. summary必须首先说明如何响应用户需求，然后列出具体变化\n")
	sb.WriteString("7. 遵守规则约束、冲突避免\n")
	sb.WriteString("\n**关键示例**：如果原始排班有5个日期（2025-12-29到2026-01-02），即使只修改了2个日期，schedule也必须包含所有5个日期，每个日期的人数必须等于需求人数。\n")

	return sb.String()
}

// convertScheduleMapToDraft 将排班 map 转换为 ShiftScheduleDraft
func (s *schedulingAIService) convertScheduleMapToDraft(scheduleMap map[string]any, nameMapping *StaffNameMapping) (*d_model.ShiftScheduleDraft, error) {
	if scheduleMap == nil {
		s.logger.Warn("convertScheduleMapToDraft: scheduleMap is nil")
		return &d_model.ShiftScheduleDraft{Schedule: make(map[string][]string)}, nil
	}

	scheduleData, ok := scheduleMap["schedule"]
	if !ok {
		s.logger.Warn("convertScheduleMapToDraft: schedule field not found in resultMap")
		return &d_model.ShiftScheduleDraft{Schedule: make(map[string][]string)}, nil
	}

	scheduleObj, ok := scheduleData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schedule is not a map, got type: %T", scheduleData)
	}

	result := &d_model.ShiftScheduleDraft{
		Schedule: make(map[string][]string),
	}

	// 记录姓名映射的统计信息
	totalNames := 0
	mappedCount := 0
	unmappedNames := make([]string, 0)

	for date, staffList := range scheduleObj {
		if staffList == nil {
			result.Schedule[date] = []string{}
			continue
		}

		staffListArr, ok := staffList.([]any)
		if !ok {
			s.logger.Warn("convertScheduleMapToDraft: staff list is not an array",
				"date", date,
				"type", fmt.Sprintf("%T", staffList))
			result.Schedule[date] = []string{}
			continue
		}

		staffIDs := make([]string, 0, len(staffListArr))
		for _, staffItem := range staffListArr {
			staffName, ok := staffItem.(string)
			if !ok {
				s.logger.Warn("convertScheduleMapToDraft: staff item is not a string",
					"date", date,
					"type", fmt.Sprintf("%T", staffItem))
				continue
			}

			totalNames++

			// 将姓名转换为 ID
			if nameMapping != nil {
				if staffID, ok := nameMapping.NameToID[staffName]; ok {
					staffIDs = append(staffIDs, staffID)
					mappedCount++
				} else {
					unmappedNames = append(unmappedNames, staffName)
					s.logger.Warn("convertScheduleMapToDraft: Staff name not found in mapping",
						"name", staffName,
						"date", date,
						"nameMappingSize", len(nameMapping.NameToID))
				}
			} else {
				s.logger.Warn("convertScheduleMapToDraft: nameMapping is nil",
					"date", date,
					"staffName", staffName)
			}
		}

		result.Schedule[date] = staffIDs
		s.logger.Info("convertScheduleMapToDraft: Date converted",
			"date", date,
			"originalCount", len(staffListArr),
			"mappedCount", len(staffIDs))
	}

	// 记录转换统计
	s.logger.Info("convertScheduleMapToDraft: Conversion summary",
		"totalDates", len(scheduleObj),
		"totalNames", totalNames,
		"mappedCount", mappedCount,
		"unmappedCount", len(unmappedNames),
		"nameMappingSize", func() int {
			if nameMapping != nil {
				return len(nameMapping.NameToID)
			}
			return 0
		}())

	if len(unmappedNames) > 0 {
		s.logger.Warn("convertScheduleMapToDraft: Unmapped staff names",
			"unmappedNames", unmappedNames[:min(10, len(unmappedNames))]) // 只显示前10个
	}

	return result, nil
}

// parseAdjustScheduleResultJSON 解析AI返回的调整排班结果JSON（包含schedule和summary）
func (s *schedulingAIService) parseAdjustScheduleResultJSON(raw string) (map[string]any, error) {
	// 复用 parseScheduleDraftJSON 的逻辑，因为它也解析 map[string]any
	// 但我们需要支持包含 summary 字段的格式
	jsonStr := s.extractJSON(raw)
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshal adjust schedule result failed: %w", err)
	}
	return result, nil
}

// generateChangesFromSummary 根据调整说明生成变化列表（第二步AI调用）
func (s *schedulingAIService) generateChangesFromSummary(
	ctx context.Context,
	summary string,
	originalDraft, finalDraft *d_model.ShiftScheduleDraft,
	nameMapping *StaffNameMapping,
	model *common_config.AIModelProvider,
) ([]d_model.AdjustScheduleChange, error) {
	s.logger.Info("generateChangesFromSummary: Generating changes from summary")

	// 构建提示词
	systemPrompt := s.buildChangesGenerationSystemPrompt()
	userPrompt := s.buildChangesGenerationUserPrompt(summary, originalDraft, finalDraft, nameMapping)

	// 调用 AI
	resp, err := s.aiFactory.CallWithModel(ctx, model, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI changes generation failed: %w", err)
	}

	// 解析结果
	raw := strings.TrimSpace(resp.Content)
	jsonStr := s.extractJSON(raw)
	var result struct {
		Changes []struct {
			Date    string   `json:"date"`
			Added   []string `json:"added"`
			Removed []string `json:"removed"`
		} `json:"changes"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse changes JSON failed: %w", err)
	}

	// 转换为 AdjustScheduleChange 列表
	changes := make([]d_model.AdjustScheduleChange, 0, len(result.Changes))
	for _, change := range result.Changes {
		// 将姓名转换为ID
		addedIDs := make([]string, 0, len(change.Added))
		for _, name := range change.Added {
			if nameMapping != nil {
				if id, ok := nameMapping.NameToID[name]; ok {
					addedIDs = append(addedIDs, id)
				} else {
					s.logger.Warn("generateChangesFromSummary: Added staff name not found in mapping",
						"name", name)
				}
			}
		}
		removedIDs := make([]string, 0, len(change.Removed))
		for _, name := range change.Removed {
			if nameMapping != nil {
				if id, ok := nameMapping.NameToID[name]; ok {
					removedIDs = append(removedIDs, id)
				} else {
					s.logger.Warn("generateChangesFromSummary: Removed staff name not found in mapping",
						"name", name)
				}
			}
		}

		changes = append(changes, d_model.AdjustScheduleChange{
			Date:    change.Date,
			Added:   addedIDs,
			Removed: removedIDs,
		})
	}

	s.logger.Info("generateChangesFromSummary: Changes generated from AI",
		"changesCount", len(changes))

	return changes, nil
}

// buildChangesGenerationSystemPrompt 构建变化列表生成的系统提示词
func (s *schedulingAIService) buildChangesGenerationSystemPrompt() string {
	return `你是一个专业的排班变化分析助手。你的任务是根据调整说明和排班方案，生成详细的变化列表。

**核心职责**：
- 仔细阅读调整说明，理解所有变化
- 对比原始排班和调整后排班，识别所有变化
- 生成准确的变化列表，确保与调整说明完全一致

**关键要求**：
1. **必须与调整说明完全一致**：变化列表必须准确反映调整说明中描述的所有变化
2. **必须包含所有变化**：不能遗漏任何在调整说明中提到的变化
3. **必须使用正确的日期格式**：日期格式必须是 YYYY-MM-DD
4. **必须使用正确的人员姓名**：人员姓名必须与提供的排班方案中的姓名完全匹配

**输出格式**：
请返回 JSON 格式，包含以下字段：
{
  "changes": [
    {
      "date": "YYYY-MM-DD",
      "added": ["人员姓名1", "人员姓名2", ...],
      "removed": ["人员姓名1", "人员姓名2", ...]
    },
    ...
  ]
}

**重要**：
- 如果某个日期既有新增又有移除，必须在同一个 change 对象中列出
- 如果某个日期只有新增或只有移除，也要列出
- 如果调整说明中提到某个日期有变化，但排班方案中没有该日期，也要根据说明列出变化

请只返回 JSON 对象，不要包含其他内容。`
}

// buildChangesGenerationUserPrompt 构建变化列表生成的用户提示词
func (s *schedulingAIService) buildChangesGenerationUserPrompt(
	summary string,
	originalDraft, finalDraft *d_model.ShiftScheduleDraft,
	nameMapping *StaffNameMapping,
) string {
	var sb strings.Builder

	sb.WriteString("【调整说明】\n")
	sb.WriteString(summary)
	sb.WriteString("\n\n")

	// 原始排班
	sb.WriteString("【原始排班】\n")
	if originalDraft != nil && len(originalDraft.Schedule) > 0 {
		dates := make([]string, 0, len(originalDraft.Schedule))
		for date := range originalDraft.Schedule {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := originalDraft.Schedule[date]
			sb.WriteString(fmt.Sprintf("%s: ", date))
			if len(staffIDs) > 0 {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				sb.WriteString(strings.Join(names, ", "))
			} else {
				sb.WriteString("无")
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("暂无排班\n")
	}
	sb.WriteString("\n")

	// 调整后排班
	sb.WriteString("【调整后排班】\n")
	if finalDraft != nil && len(finalDraft.Schedule) > 0 {
		dates := make([]string, 0, len(finalDraft.Schedule))
		for date := range finalDraft.Schedule {
			dates = append(dates, date)
		}
		sort.Strings(dates)
		for _, date := range dates {
			staffIDs := finalDraft.Schedule[date]
			sb.WriteString(fmt.Sprintf("%s: ", date))
			if len(staffIDs) > 0 {
				names := s.convertIDsToNames(staffIDs, nameMapping.IDToName)
				sb.WriteString(strings.Join(names, ", "))
			} else {
				sb.WriteString("无")
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("暂无排班\n")
	}
	sb.WriteString("\n")

	sb.WriteString("【任务要求】\n")
	sb.WriteString("请根据上述调整说明，对比原始排班和调整后排班，生成详细的变化列表。\n")
	sb.WriteString("确保变化列表与调整说明完全一致，包含所有在调整说明中提到的变化。\n")

	return sb.String()
}

// calculateScheduleChanges 计算排班变化列表（备用方法）
func (s *schedulingAIService) calculateScheduleChanges(originalDraft, finalDraft *d_model.ShiftScheduleDraft) []d_model.AdjustScheduleChange {
	if originalDraft == nil || finalDraft == nil {
		return nil
	}

	changes := make([]d_model.AdjustScheduleChange, 0)

	// 收集所有日期
	allDates := make(map[string]bool)
	for date := range originalDraft.Schedule {
		allDates[date] = true
	}
	for date := range finalDraft.Schedule {
		allDates[date] = true
	}

	// 对每个日期计算变化
	for date := range allDates {
		originalStaff := make(map[string]bool)
		if originalDraft.Schedule != nil {
			if staffIDs, ok := originalDraft.Schedule[date]; ok {
				for _, id := range staffIDs {
					originalStaff[id] = true
				}
			}
		}

		finalStaff := make(map[string]bool)
		if finalDraft.Schedule != nil {
			if staffIDs, ok := finalDraft.Schedule[date]; ok {
				for _, id := range staffIDs {
					finalStaff[id] = true
				}
			}
		}

		// 计算新增和移除的人员
		added := make([]string, 0)
		removed := make([]string, 0)

		for id := range finalStaff {
			if !originalStaff[id] {
				added = append(added, id)
			}
		}

		for id := range originalStaff {
			if !finalStaff[id] {
				removed = append(removed, id)
			}
		}

		// 如果有变化，添加到列表
		if len(added) > 0 || len(removed) > 0 {
			changes = append(changes, d_model.AdjustScheduleChange{
				Date:    date,
				Added:   added,
				Removed: removed,
			})
		}
	}

	return changes
}

// parseAdjustIntentType 解析调整意图类型字符串
func (s *schedulingAIService) parseAdjustIntentType(typeStr string) d_model.AdjustIntentType {
	switch strings.ToLower(typeStr) {
	case "swap":
		return d_model.AdjustIntentSwap
	case "replace":
		return d_model.AdjustIntentReplace
	case "add":
		return d_model.AdjustIntentAdd
	case "remove":
		return d_model.AdjustIntentRemove
	case "modify":
		return d_model.AdjustIntentModify
	case "batch":
		return d_model.AdjustIntentBatch
	case "regenerate":
		return d_model.AdjustIntentRegenerate
	case "custom":
		return d_model.AdjustIntentCustom
	case "other", "unknown":
		return d_model.AdjustIntentOther
	default:
		return d_model.AdjustIntentOther
	}
}

// analyzeAdjustIntentWithDefaultPrompt 使用默认提示词分析调整意图
func (s *schedulingAIService) analyzeAdjustIntentWithDefaultPrompt(ctx context.Context, userInput string, messages []session.Message) (*d_model.AdjustIntent, error) {
	systemPrompt := `你是排班调整助手的意图识别器。分析用户输入，识别调整意图类型。

## 调整意图类型：
1. swap(换班): 两人互换班次
2. replace(替班): 用新人替换原有人员
3. add(添加人员): 在某天增加一个人
4. remove(移除人员): 从某天移除一个人
5. batch(批量调整): 一次调整多天或多人
6. regenerate(重新生成): 对某段时间重新排班
7. custom(自定义): 复杂的自定义调整需求
8. other(无法识别): 意图不明确

## 输出格式：
返回 JSON 数组：
[{"type": "swap", "confidence": 0.9, "summary": "描述", "arguments": {"date": "...", "staffA": "...", "staffB": "..."}}]

只输出纯JSON数组。`

	userPrompt := s.buildAdjustIntentUserPrompt(userInput, messages)

	resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	return s.parseAdjustIntentResponse(resp.Content, userInput)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================
// ExtractTemporaryNeeds - 从用户消息中提取临时需求
// ============================================================

// ExtractTemporaryNeeds 从用户消息中提取临时需求
func (s *schedulingAIService) ExtractTemporaryNeeds(ctx context.Context, userMessage string, allStaffList []*d_model.Employee, startDate, endDate string, messages []session.Message) ([]*d_model.PersonalNeed, error) {
	s.logger.Info("ExtractTemporaryNeeds: Starting extraction",
		"userMessageLength", len(userMessage),
		"staffCount", len(allStaffList),
		"startDate", startDate,
		"endDate", endDate)

	// 构建系统提示词
	systemPrompt := s.buildExtractTemporaryNeedsSystemPrompt()

	// 构建用户提示词
	userPrompt := s.buildExtractTemporaryNeedsUserPrompt(userMessage, allStaffList, startDate, endDate, messages)

	// 调用 AI
	resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	// 解析结果
	needs, err := s.parseTemporaryNeedsResponse(resp.Content, allStaffList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse temporary needs: %w", err)
	}

	s.logger.Info("ExtractTemporaryNeeds: Extraction completed",
		"needsCount", len(needs))

	return needs, nil
}

// buildExtractTemporaryNeedsSystemPrompt 构建提取临时需求的系统提示词
func (s *schedulingAIService) buildExtractTemporaryNeedsSystemPrompt() string {
	return `你是一个专业的排班需求分析助手。你的任务是从用户的调整需求消息中识别出临时需求（如某人出差、某天有事等），这些需求应该被添加到临时需求列表中，以便后续排班时能够响应。

**核心职责**：
- 识别用户消息中提到的临时需求（如出差、请假、有事等）
- **明确用户提出需求的范围**：识别用户需求涉及的人员、日期、时间范围
- 提取相关的人员姓名和日期信息
- 将临时需求转换为标准化的 PersonalNeed 结构

**用户提出需求的范围识别**：
用户的需求可能涉及以下范围，需要明确识别：
1. **人员范围**：
   - 明确提到的人员姓名（如"张三"、"张三"）
   - 如果提到"某人"、"他"、"她"等代词，需要结合上下文推断具体人员
   - 如果提到"所有人"、"大家"等，需要标记为影响所有人员

2. **日期范围**：
   - 具体日期（如"12月25日"、"周二"）
   - 日期范围（如"12月25日到12月30日"、"周二到周五"）
   - 相对日期（如"明天"、"下周"），需要根据排班周期推断具体日期
   - 如果未明确日期，但提到"这段时间"、"这个周期"等，应理解为整个排班周期

3. **时间范围**：
   - 全天（如"出差"、"请假"）
   - 部分时段（如"上午有事"、"下午不在"）

**临时需求的识别标准**：
1. **人员相关临时需求**：
   - 某人出差、某人请假、某人有事、某人不在、某人无法排班等
   - 例如："张三周二要公出"、"张三周三请假"、"王五周五有事"
   - 识别要点：提取人员姓名 + 日期 + 原因

2. **日期相关临时需求**：
   - 某天有事、某天不能排班、某天需要调整等
   - 例如："周二有事"、"周三不能排班"、"周五需要调整"
   - 识别要点：提取日期 + 原因（如果未明确人员，可能影响所有人员）

3. **时间范围相关临时需求**：
   - 某段时间出差、某段时间请假、某段时间不在等
   - 例如："12月25日到12月30日出差"、"下周一到下周三请假"
   - 识别要点：提取开始日期 + 结束日期 + 原因 + 人员（如果有）

**输出格式**：
请返回 JSON 格式的数组，每个元素是一个临时需求对象，包含以下字段：
- staffName: 人员姓名（如果消息中提到了具体人员，必须填写；如果未明确人员，留空）
- staffId: 人员ID（如果能够匹配到人员列表中的姓名，则填写对应的ID；如果无法匹配，则留空）
- targetDates: 目标日期列表 (YYYY-MM-DD格式)，必须填写：
  - 如果消息中提到了具体日期，填写这些日期
  - 如果消息中提到了日期范围，填写范围内的所有日期
  - 如果消息中提到了相对日期（如"明天"、"下周"），根据排班周期推断具体日期
  - 如果未明确日期但提到"这段时间"，填写整个排班周期的所有日期
- description: 需求描述（从用户消息中提取的原始描述，必须清晰说明人员、日期、原因）
- requestType: 请求类型，通常为 "avoid"（回避，表示该人员在这些日期不能排班）
- priority: 优先级 (1-10, 数字越小优先级越高)，临时需求通常为 5

**重要规则**：
1. **必须明确用户提出需求的范围**：
   - 如果用户说"张三周二要公出"，范围是：人员=张三，日期=周二
   - 如果用户说"周二有事"，范围是：日期=周二，人员=未明确（可能影响所有人员）
   - 如果用户说"12月25日到12月30日出差"，范围是：日期=12月25日到12月30日，人员=未明确

2. **日期推断规则**：
   - "明天" = 排班周期开始日期的下一天
   - "下周" = 排班周期开始日期的下一周
   - "周二"、"周三"等 = 排班周期内对应的所有周二、周三
   - "12月25日" = 如果排班周期包含该日期，则填写；否则留空

3. **人员匹配规则**：
   - 优先精确匹配人员列表中的姓名
   - 如果姓名不完全匹配，尝试模糊匹配（如"张三"匹配"张三-(0012)"）
   - 如果无法匹配，staffId 留空，但 staffName 必须填写（从消息中提取的原始姓名）

4. **只提取临时需求**：
   - 临时需求：有明确时间限制的需求（如出差、请假、有事等）
   - 不要提取常态化需求（如"希望每周休息两天"）

**注意事项**：
- 如果消息中没有提到临时需求，返回空数组 []
- 如果无法确定具体的人员或日期，尽量从消息中推断，但不要猜测
- 日期格式必须为 YYYY-MM-DD
- description 必须清晰，便于后续排班时理解需求

请只返回 JSON 数组，不要包含其他内容。`
}

// buildExtractTemporaryNeedsUserPrompt 构建提取临时需求的用户提示词
func (s *schedulingAIService) buildExtractTemporaryNeedsUserPrompt(userMessage string, allStaffList []*d_model.Employee, startDate, endDate string, messages []session.Message) string {
	var sb strings.Builder

	sb.WriteString("请从以下用户消息中提取临时需求，并明确用户提出需求的范围（人员范围、日期范围、时间范围）：\n\n")
	sb.WriteString("【用户消息】\n")
	sb.WriteString(userMessage)
	sb.WriteString("\n\n")

	sb.WriteString("【排班周期】\n")
	sb.WriteString(fmt.Sprintf("开始日期: %s\n", startDate))
	sb.WriteString(fmt.Sprintf("结束日期: %s\n\n", endDate))

	// 添加人员列表（用于姓名匹配）
	if len(allStaffList) > 0 {
		sb.WriteString("【可用人员列表】（用于姓名匹配）\n")
		// 只显示前50个人员，避免提示词过长
		displayCount := len(allStaffList)
		if displayCount > 50 {
			displayCount = 50
		}
		for i := 0; i < displayCount; i++ {
			staff := allStaffList[i]
			if staff != nil {
				sb.WriteString(fmt.Sprintf("%d. %s (ID: %s)\n", i+1, staff.Name, staff.ID))
			}
		}
		if len(allStaffList) > 50 {
			sb.WriteString(fmt.Sprintf("... 还有 %d 个人员（共 %d 个）\n", len(allStaffList)-50, len(allStaffList)))
		}
		sb.WriteString("\n")
	}

	// 添加最近的会话消息（用于上下文理解，特别是代词指代）
	if len(messages) > 0 {
		sb.WriteString("【最近会话消息】（用于上下文理解，特别是识别代词指代的人员）\n")
		// 只取最近5条消息
		startIdx := len(messages) - 5
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(messages); i++ {
			msg := messages[i]
			role := "用户"
			if msg.Role == "assistant" {
				role = "助手"
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("【任务要求】\n")
	sb.WriteString("1. 明确识别用户提出需求的范围：\n")
	sb.WriteString("   - 人员范围：哪些人员受到影响（如果未明确，说明未明确）\n")
	sb.WriteString("   - 日期范围：哪些日期受到影响（必须转换为 YYYY-MM-DD 格式）\n")
	sb.WriteString("   - 时间范围：全天还是部分时段（如果未明确，默认为全天）\n")
	sb.WriteString("2. 提取临时需求并返回 JSON 数组\n")
	sb.WriteString("3. 如果消息中没有临时需求，返回空数组 []\n\n")

	sb.WriteString("请提取临时需求并返回 JSON 数组。")

	return sb.String()
}

// parseTemporaryNeedsResponse 解析 AI 返回的临时需求
func (s *schedulingAIService) parseTemporaryNeedsResponse(content string, allStaffList []*d_model.Employee) ([]*d_model.PersonalNeed, error) {
	// 清理内容，提取 JSON 部分
	content = strings.TrimSpace(content)

	// 尝试提取 JSON 数组
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return []*d_model.PersonalNeed{}, nil // 没有找到 JSON 数组，返回空数组
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	// 解析 JSON
	var rawNeeds []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &rawNeeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 创建姓名到ID的映射
	nameToID := make(map[string]string)
	for _, staff := range allStaffList {
		if staff != nil {
			nameToID[staff.Name] = staff.ID
		}
	}

	// 转换为 PersonalNeed 结构
	needs := make([]*d_model.PersonalNeed, 0, len(rawNeeds))
	for _, raw := range rawNeeds {
		need := &d_model.PersonalNeed{
			NeedType:    "temporary",
			RequestType: "avoid", // 临时需求通常是回避类型
			Priority:    5,       // 默认优先级
			Source:      "user",
			Confirmed:   false,
		}

		// 提取 staffName
		if name, ok := raw["staffName"].(string); ok && name != "" {
			need.StaffName = name
			// 尝试匹配 staffId
			if id, exists := nameToID[name]; exists {
				need.StaffID = id
			}
		}

		// 提取 targetDates
		if dates, ok := raw["targetDates"].([]any); ok {
			need.TargetDates = make([]string, 0, len(dates))
			for _, d := range dates {
				if date, ok := d.(string); ok {
					need.TargetDates = append(need.TargetDates, date)
				}
			}
		}

		// 提取 description
		if desc, ok := raw["description"].(string); ok {
			need.Description = desc
		} else {
			// 如果没有 description，使用 staffName 和 targetDates 构建
			if need.StaffName != "" {
				need.Description = need.StaffName
				if len(need.TargetDates) > 0 {
					need.Description += " " + strings.Join(need.TargetDates, ", ")
				}
			}
		}

		// 提取 requestType
		if rt, ok := raw["requestType"].(string); ok {
			need.RequestType = rt
		}

		// 提取 priority
		if p, ok := raw["priority"].(float64); ok {
			need.Priority = int(p)
		}

		// 只添加有效的需求（至少要有 staffName 或 targetDates）
		if need.StaffName != "" || len(need.TargetDates) > 0 {
			needs = append(needs, need)
		}
	}

	return needs, nil
}

// ============================================================
// V3增强辅助函数
// ============================================================

// formatStaffScheduleInfo 格式化人员排班信息（辅助函数）
func (s *schedulingAIService) formatStaffScheduleInfo(sb *strings.Builder, schedule *d_model.StaffCurrentSchedule) {
	sb.WriteString(fmt.Sprintf("- %s (%.1fh)", schedule.StaffName, schedule.TotalHours))

	if len(schedule.Shifts) > 0 {
		sb.WriteString(": ")
		shiftDetails := make([]string, 0)
		for _, shift := range schedule.Shifts {
			detail := fmt.Sprintf("%s(%s-%s,%.1fh)",
				shift.ShiftName, shift.StartTime, shift.EndTime, shift.Duration)
			if shift.IsFixed {
				detail += "[固定]"
			}
			if shift.IsOvernight {
				detail += "[跨夜]"
			}
			shiftDetails = append(shiftDetails, detail)
		}
		sb.WriteString(strings.Join(shiftDetails, " | "))
	}

	// 显示错误（阻断性）
	if len(schedule.Errors) > 0 {
		sb.WriteString(fmt.Sprintf(" ❌ %s", strings.Join(schedule.Errors, "; ")))
	}

	// 显示警告（提示性）
	if len(schedule.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf(" ⚠️ %s", strings.Join(schedule.Warnings, "; ")))
	}

	sb.WriteString("\n")
}

// buildSchedulingContextForTodo 为Todo执行构建排班上下文（简化版）
// 注意：这是对schedule_v3/utils包中BuildSchedulingContext的包装
func buildSchedulingContextForTodo(
	date string,
	targetShift *d_model.Shift,
	requiredCount int,
	workingDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) *d_model.V3SchedulingContext {
	// 注意：由于循环依赖问题，这里需要重新实现或者调用utils包的方法
	// 为了避免循环依赖，我们在这里实现简化版本

	// 构建班次映射
	shiftMap := make(map[string]*d_model.Shift)
	for _, shift := range allShifts {
		if shift != nil {
			shiftMap[shift.ID] = shift
		}
	}

	// 构建人员名称映射
	staffNamesMap := make(map[string]string)
	for _, staff := range append(staffList, allStaffList...) {
		if staff != nil {
			staffNamesMap[staff.ID] = staff.Name
		}
	}

	// 构建人员当前排班状态
	staffScheduleMap := make(map[string]*d_model.StaffCurrentSchedule)

	if workingDraft != nil && workingDraft.Shifts != nil {
		for shiftID, shiftDraft := range workingDraft.Shifts {
			if shiftDraft == nil || shiftDraft.Days == nil {
				continue
			}

			shift := shiftMap[shiftID]
			if shift == nil {
				continue
			}

			dayShift := shiftDraft.Days[date]
			if dayShift != nil {
				for _, staffID := range dayShift.StaffIDs {
					if staffScheduleMap[staffID] == nil {
						staffScheduleMap[staffID] = &d_model.StaffCurrentSchedule{
							StaffID:   staffID,
							StaffName: staffNamesMap[staffID],
							Date:      date,
							Shifts:    make([]*d_model.AssignedShiftInfo, 0),
							Errors:    make([]string, 0),
							Warnings:  make([]string, 0),
						}
					}

					assignedShift := &d_model.AssignedShiftInfo{
						ShiftID:     shift.ID,
						ShiftName:   shift.Name,
						StartTime:   shift.StartTime,
						EndTime:     shift.EndTime,
						Duration:    float64(shift.Duration) / 60.0,
						IsOvernight: shift.IsOvernight,
						IsFixed:     dayShift.IsFixed,
					}

					staffScheduleMap[staffID].Shifts = append(
						staffScheduleMap[staffID].Shifts,
						assignedShift,
					)
					staffScheduleMap[staffID].TotalHours += assignedShift.Duration
				}
			}
		}
	}

	// 检测错误和警告
	for _, schedule := range staffScheduleMap {
		// 检测时间冲突
		if len(schedule.Shifts) > 1 && hasTimeOverlapSimple(schedule.Shifts) {
			schedule.Errors = append(schedule.Errors, "存在时间冲突")
		}

		// 检测超时
		if schedule.TotalHours > 12.0 {
			schedule.Errors = append(schedule.Errors,
				fmt.Sprintf("超时：已安排%.1f小时（超过12.0小时限制）", schedule.TotalHours))
		} else if schedule.TotalHours > 10.8 {
			schedule.Warnings = append(schedule.Warnings,
				fmt.Sprintf("接近超时：已安排%.1f小时", schedule.TotalHours))
		}
	}

	// 转换为数组
	staffSchedules := make([]*d_model.StaffCurrentSchedule, 0, len(staffScheduleMap))
	for _, schedule := range staffScheduleMap {
		staffSchedules = append(staffSchedules, schedule)
	}

	// 构建上下文
	return &d_model.V3SchedulingContext{
		TargetDate:      date,
		TargetShiftID:   targetShift.ID,
		TargetShiftName: targetShift.Name,
		TargetShiftTime: fmt.Sprintf("%s-%s", targetShift.StartTime, targetShift.EndTime),
		RequiredCount:   requiredCount,
		AllShifts:       allShifts,
		StaffSchedules:  staffSchedules,
		MaxDailyHours:   12.0,
		MinRestHours:    12.0,
	}
}

// hasTimeOverlapSimple 简化版时间重叠检测
func hasTimeOverlapSimple(shifts []*d_model.AssignedShiftInfo) bool {
	if len(shifts) <= 1 {
		return false
	}

	for i := 0; i < len(shifts)-1; i++ {
		for j := i + 1; j < len(shifts); j++ {
			if checkTimeOverlapSimple(shifts[i], shifts[j]) {
				return true
			}
		}
	}

	return false
}

// checkTimeOverlapSimple 简化版检查两个班次是否时间重叠
func checkTimeOverlapSimple(shift1, shift2 *d_model.AssignedShiftInfo) bool {
	if shift1 == nil || shift2 == nil {
		return false
	}

	s1 := timeToMinutesSimple(shift1.StartTime)
	e1 := timeToMinutesSimple(shift1.EndTime)
	s2 := timeToMinutesSimple(shift2.StartTime)
	e2 := timeToMinutesSimple(shift2.EndTime)

	if shift1.IsOvernight {
		e1 += 24 * 60
	}
	if shift2.IsOvernight {
		e2 += 24 * 60
	}

	return !(e1 <= s2 || e2 <= s1)
}

// timeToMinutesSimple 将HH:MM转换为分钟数
func timeToMinutesSimple(timeStr string) int {
	if timeStr == "" {
		return 0
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])

	return hours*60 + minutes
}

// calculateShiftDurationHours 计算班次时长（小时）
func calculateShiftDurationHours(shift *d_model.Shift) float64 {
	if shift == nil {
		return 0
	}

	start := timeToMinutesSimple(shift.StartTime)
	end := timeToMinutesSimple(shift.EndTime)

	if shift.IsOvernight {
		end += 24 * 60
	}

	duration := end - start
	if duration < 0 {
		duration = 0
	}

	return float64(duration) / 60.0
}

// ============================================================
// 全局评审人工处理方法
// ============================================================

// ProcessManualReviewModification 处理人工评审修改
func (s *schedulingAIService) ProcessManualReviewModification(
	ctx context.Context,
	userMessage string,
	manualContext *d_model.ManualReviewContext,
	currentDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	shifts []*d_model.Shift,
) (*d_model.ManualReviewModifyResult, error) {
	s.logger.Debug("ProcessManualReviewModification called",
		"userMessage", userMessage,
		"draftExists", currentDraft != nil,
	)

	// 构建系统提示词
	systemPrompt := s.buildManualReviewSystemPrompt()

	// 构建用户提示词
	userPrompt := s.buildManualReviewUserPrompt(userMessage, manualContext, currentDraft, staffList, shifts)

	// 调用 LLM
	resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("调用LLM处理人工修改失败: %w", err)
	}

	// 解析响应
	result, err := s.parseManualReviewResponse(resp.Content, currentDraft)
	if err != nil {
		s.logger.Warn("解析人工修改响应失败",
			"error", err,
			"rawContent", resp.Content,
		)
		return &d_model.ManualReviewModifyResult{
			Success:       false,
			Summary:       fmt.Sprintf("无法理解您的修改意图: %v", err),
			ModifiedDraft: currentDraft,
		}, nil
	}

	return result, nil
}

// buildManualReviewSystemPrompt 构建人工处理系统提示词
func (s *schedulingAIService) buildManualReviewSystemPrompt() string {
	return `你是一个专业的排班调整助手。用户正在处理全局规则评审中需要人工介入的项目。

你的职责是：
1. 理解用户的修改意图
2. 根据用户指示调整排班草案
3. 确保修改后的排班仍然合理

请以JSON格式输出处理结果：
{
  "understood": true/false,
  "summary": "处理摘要",
  "appliedChanges": ["变更1描述", "变更2描述"],
  "shifts": {
    "班次ID": {
      "日期YYYY-MM-DD": ["人员ID1", "人员ID2", ...]
    }
  }
}

注意：
- 如果无法理解用户意图，设置 understood 为 false 并在 summary 中说明
- shifts 字段只包含需要修改的班次和日期，未修改的部分不要输出
- 每个日期的 staffIds 是修改后该日期的完整人员列表`
}

// buildManualReviewUserPrompt 构建人工处理用户提示词
func (s *schedulingAIService) buildManualReviewUserPrompt(
	userMessage string,
	manualContext *d_model.ManualReviewContext,
	currentDraft *d_model.ScheduleDraft,
	staffList []*d_model.Employee,
	shifts []*d_model.Shift,
) string {
	var sb strings.Builder

	// 用户需求
	sb.WriteString("## 用户修改需求\n\n")
	sb.WriteString(userMessage)
	sb.WriteString("\n\n")

	// 需处理的评审项
	if manualContext != nil && len(manualContext.ManualReviewItems) > 0 {
		sb.WriteString("## 需人工处理的评审项\n\n")
		for i, item := range manualContext.ManualReviewItems {
			sb.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, item.ReviewItemName, item.ReviewItemType))
			sb.WriteString(fmt.Sprintf("   - 问题: %s\n", item.ViolationDescription))
			sb.WriteString(fmt.Sprintf("   - 建议: %s\n", item.Suggestion))
			if item.ConflictReason != "" {
				sb.WriteString(fmt.Sprintf("   - 冲突: %s\n", item.ConflictReason))
			}
			sb.WriteString("\n")
		}
	}

	// 只输出相关人员信息（最多20人）
	sb.WriteString("## 人员信息\n\n")
	staffCount := 0
	for _, staff := range staffList {
		if staff != nil {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", staff.ID, staff.Name))
			staffCount++
			if staffCount >= 20 {
				sb.WriteString(fmt.Sprintf("- ... (共%d人，已省略部分)\n", len(staffList)))
				break
			}
		}
	}

	// 只输出相关班次信息
	sb.WriteString("\n## 班次信息\n\n")
	for _, shift := range shifts {
		if shift != nil {
			sb.WriteString(fmt.Sprintf("- %s (%s): %s-%s\n", shift.Name, shift.ID, shift.StartTime, shift.EndTime))
		}
	}

	// 当前排班（只输出相关日期的）
	sb.WriteString("\n## 当前排班草案\n\n")
	if currentDraft != nil && currentDraft.Shifts != nil {
		for shiftID, shiftDraft := range currentDraft.Shifts {
			if shiftDraft == nil || shiftDraft.Days == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n", shiftID))
			// 按日期排序
			dates := make([]string, 0, len(shiftDraft.Days))
			for date := range shiftDraft.Days {
				dates = append(dates, date)
			}
			sort.Strings(dates)
			for _, date := range dates {
				dayShift := shiftDraft.Days[date]
				if dayShift != nil {
					sb.WriteString(fmt.Sprintf("- %s: %v\n", date, dayShift.StaffIDs))
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n请根据用户的修改需求处理上述评审项，输出处理结果。")

	return sb.String()
}

// parseManualReviewResponse 解析人工处理响应
func (s *schedulingAIService) parseManualReviewResponse(content string, originalDraft *d_model.ScheduleDraft) (*d_model.ManualReviewModifyResult, error) {
	jsonStr := s.extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("无法从响应中提取JSON")
	}

	var resp struct {
		Understood     bool                           `json:"understood"`
		Summary        string                         `json:"summary"`
		AppliedChanges []string                       `json:"appliedChanges"`
		Shifts         map[string]map[string][]string `json:"shifts"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	result := &d_model.ManualReviewModifyResult{
		Success:        resp.Understood,
		Summary:        resp.Summary,
		AppliedChanges: resp.AppliedChanges,
	}

	// 如果有修改，基于原草案应用变更
	if resp.Understood && len(resp.Shifts) > 0 {
		// 深拷贝原始草案
		newDraft := &d_model.ScheduleDraft{
			StartDate:  originalDraft.StartDate,
			EndDate:    originalDraft.EndDate,
			Shifts:     make(map[string]*d_model.ShiftDraft),
			Summary:    originalDraft.Summary,
			StaffStats: originalDraft.StaffStats,
			Conflicts:  originalDraft.Conflicts,
		}

		// 首先复制原始草案的所有班次
		for shiftID, shiftDraft := range originalDraft.Shifts {
			if shiftDraft == nil {
				continue
			}
			newShiftDraft := &d_model.ShiftDraft{
				ShiftID:  shiftDraft.ShiftID,
				Priority: shiftDraft.Priority,
				Days:     make(map[string]*d_model.DayShift),
			}
			for date, dayShift := range shiftDraft.Days {
				if dayShift == nil {
					continue
				}
				newDayShift := &d_model.DayShift{
					StaffIDs:      make([]string, len(dayShift.StaffIDs)),
					RequiredCount: dayShift.RequiredCount,
					ActualCount:   dayShift.ActualCount,
					IsFixed:       dayShift.IsFixed,
				}
				copy(newDayShift.StaffIDs, dayShift.StaffIDs)
				newShiftDraft.Days[date] = newDayShift
			}
			newDraft.Shifts[shiftID] = newShiftDraft
		}

		// 然后应用 LLM 返回的变更
		for shiftID, dates := range resp.Shifts {
			shiftDraft, ok := newDraft.Shifts[shiftID]
			if !ok {
				// 新班次
				shiftDraft = &d_model.ShiftDraft{
					ShiftID: shiftID,
					Days:    make(map[string]*d_model.DayShift),
				}
				newDraft.Shifts[shiftID] = shiftDraft
			}

			for date, staffIDs := range dates {
				dayShift, ok := shiftDraft.Days[date]
				if !ok {
					dayShift = &d_model.DayShift{}
					shiftDraft.Days[date] = dayShift
				}
				dayShift.StaffIDs = staffIDs
				dayShift.ActualCount = len(staffIDs)
			}
		}

		result.ModifiedDraft = newDraft
	} else {
		result.ModifiedDraft = originalDraft
	}

	return result, nil
}
