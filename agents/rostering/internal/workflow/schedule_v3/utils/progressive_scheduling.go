package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"

	d_model "jusha/agent/rostering/domain/model"
	common_config "jusha/mcp/pkg/config"
)

// isUUID 检查字符串是否是UUID格式
func isUUID(s string) bool {
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidPattern.MatchString(s)
}

// ============================================================
// 渐进式排班服务
// ============================================================

// IProgressiveSchedulingService 渐进式排班服务接口
type IProgressiveSchedulingService interface {
	// AssessRequirementsAndPlanTasks 评估所有需求并生成渐进式任务计划
	AssessRequirementsAndPlanTasks(
		ctx context.Context,
		shifts []*d_model.Shift,
		rules []*d_model.Rule,
		personalNeeds map[string][]*d_model.PersonalNeed, // 使用强类型，禁止使用interface{}
		fixedShiftAssignments map[string][]string,
		staffRequirements map[string]map[string]int,
		allStaffList []*d_model.Employee, // 所有员工列表（用于ID到姓名映射）
		startDate, endDate string,
	) (*d_model.ProgressiveTaskPlan, error)
}

// ProgressiveSchedulingService 渐进式排班服务实现
type ProgressiveSchedulingService struct {
	logger           logging.ILogger
	aiFactory        *ai.AIProviderFactory
	configurator     config.IRosteringConfigurator
	baseConfigurator common_config.IServiceConfigurator
}

// NewProgressiveSchedulingService 创建渐进式排班服务
func NewProgressiveSchedulingService(
	logger logging.ILogger,
	aiFactory *ai.AIProviderFactory,
	configurator config.IRosteringConfigurator,
) IProgressiveSchedulingService {
	baseConfigurator := configurator.(common_config.IServiceConfigurator)
	return &ProgressiveSchedulingService{
		logger:           logger.With("component", "ProgressiveSchedulingService"),
		aiFactory:        aiFactory,
		configurator:     configurator,
		baseConfigurator: baseConfigurator,
	}
}

// AssessRequirementsAndPlanTasks 评估所有需求并生成渐进式任务计划
func (s *ProgressiveSchedulingService) AssessRequirementsAndPlanTasks(
	ctx context.Context,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed, // 使用强类型，禁止使用interface{}
	fixedShiftAssignments map[string][]string,
	staffRequirements map[string]map[string]int,
	allStaffList []*d_model.Employee, // 所有员工列表（用于ID到姓名映射）
	startDate, endDate string,
) (*d_model.ProgressiveTaskPlan, error) {
	s.logger.Info("Assessing requirements and planning progressive tasks",
		"shiftsCount", len(shifts),
		"rulesCount", len(rules),
		"startDate", startDate,
		"endDate", endDate)

	// 构建ID映射表
	shiftForwardMappings, shiftReverseMappings := BuildShiftIDMappings(shifts)
	ruleForwardMappings, ruleReverseMappings := BuildRuleIDMappings(rules)

	// 合并所有映射表用于提示词替换
	allForwardMappings := make(map[string]string)
	for k, v := range shiftForwardMappings {
		allForwardMappings[k] = v
	}
	for k, v := range ruleForwardMappings {
		allForwardMappings[k] = v
	}

	systemPrompt := s.buildRequirementAssessmentSystemPrompt()
	userPrompt := s.buildRequirementAssessmentUserPrompt(
		shifts, rules, personalNeeds, fixedShiftAssignments, staffRequirements,
		allStaffList, shiftForwardMappings, ruleForwardMappings,
		startDate, endDate)

	// 在调用LLM前，替换提示词中的ID
	userPrompt = ReplaceIDsWithShortIDs(userPrompt, allForwardMappings)

	// 获取配置的模型
	cfg := s.configurator.GetConfig()
	var assessmentModel *common_config.AIModelProvider
	if cfg.SchedulingAI.TaskModels != nil {
		if model, ok := cfg.SchedulingAI.TaskModels["requirement_assessment"]; ok && model.Provider != "" && model.Name != "" {
			assessmentModel = &model
		}
	}

	// 调用AI（使用与scheduling_ai.go相同的方式）
	llmStart := time.Now()
	resp, err := s.aiFactory.CallWithModel(ctx, assessmentModel, systemPrompt, userPrompt, nil)
	llmDuration := time.Since(llmStart)

	// 记录 LLM 调试日志
	if debugLogger := logging.GetLLMDebugLogger(); debugLogger != nil && debugLogger.IsEnabled() {
		modelName := ""
		if assessmentModel != nil {
			modelName = assessmentModel.Name
		}
		debugLogger.LogLLMCall(
			"requirement_assessment",
			logging.LLMCallRequirementAssessment,
			"",
			fmt.Sprintf("%s_%s", startDate, endDate),
			modelName,
			systemPrompt,
			userPrompt,
			resp.Content,
			llmDuration,
			err,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("AI requirement assessment failed: %w", err)
	}

	// 解析结果
	raw := strings.TrimSpace(resp.Content)
	result, parseErr := s.parseRequirementAssessmentResult(raw)
	if parseErr != nil {
		s.logger.Error("Failed to parse requirement assessment result JSON", "error", parseErr, "rawContent", raw)
		return nil, fmt.Errorf("parse requirement assessment result json failed: %w", parseErr)
	}

	// 在解析后，将任务计划中的简短ID替换回真实ID
	ReplaceIDsInTaskPlan(result, shiftReverseMappings, ruleReverseMappings)

	// 按班次排班优先级对任务进行代码排序（不依赖LLM排序）
	sortTasksByShiftPriority(result, shifts)

	s.logger.Info("Requirement assessment completed",
		"tasksCount", len(result.Tasks),
		"summary", result.Summary)

	return result, nil
}

// buildRequirementAssessmentSystemPrompt 构建需求评估的系统提示词
func (s *ProgressiveSchedulingService) buildRequirementAssessmentSystemPrompt() string {
	return `你是排班需求评估助手。分析所有排班需求，生成渐进式任务计划（JSON）。

## 前置条件
固定排班已由系统预先完成（写入排班表并标记占用），禁止生成任何固定排班相关任务。

## 个人需求分类
- 正向需求：RequestType为prefer/must且指定了TargetShiftID → 将人员安排到指定日期班次
- 负向需求：RequestType为avoid或未指定TargetShiftID → 后续排班时避开这些人员日期

## 任务阶段（严格按序）
1. 正向需求填充：处理正向个人需求
2. 特殊班次填充：根据【班次信息】中的排班优先级（数值越小越优先）和【排班规则】中的约束数量，识别出规则复杂、人数需求少、对人员资质有严格要求的班次，优先占位以确保合规
3. 剩余人员填充：补齐常规班次的人员缺口（同样避开负向需求）

## 严禁幻觉
- 所有班次名称和ID必须严格来源于输入的【班次信息】，禁止编造、推测或组合出不存在的班次名称
- targetShifts中的每个ID都必须能在【班次信息】中找到对应记录
- 任务描述中提到的班次名称必须与【班次信息】中完全一致

## 任务组织
- 按业务逻辑组织，一个任务可涉及多个班次（系统执行时自动拆分为单班次子任务）
- 通过targetShifts字段指定涉及的班次，系统根据该字段自动匹配
- 存在业务关联的班次（如互斥约束、轮转要求、人员共享关系的班次）必须放在同一个任务中一起安排，避免后续任务覆盖前序结果
- 粒度按复杂度决定：规则相似可合并，简单场景2-3个任务即可
- 每个任务必须有明确目标，不生成无意义任务

## 输出格式
{
  "tasks": [{
    "id": "task_1",
    "order": 1,
    "title": "简明标题（如：正向需求填充：安排指定人员到要求的班次）",
    "description": "详细说明涉及的规则、人员、日期、班次类型，明确正向/负向需求处理方式",
    "targetShifts": ["班次ID（可选）"],
    "targetDates": ["YYYY-MM-DD"],
    "targetStaff": ["人员ID（可选）"],
    "ruleIds": ["规则ID"],
    "priority": 1
  }],
  "summary": "整体规划说明",
  "reasoning": "分解思路"
}

只返回JSON，不要包含其他内容。`
}

// buildRequirementAssessmentUserPrompt 构建需求评估的用户提示词
func (s *ProgressiveSchedulingService) buildRequirementAssessmentUserPrompt(
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	personalNeeds map[string][]*d_model.PersonalNeed, // 使用强类型，禁止使用interface{}
	fixedShiftAssignments map[string][]string,
	staffRequirements map[string]map[string]int,
	allStaffList []*d_model.Employee,
	shiftMappings, ruleMappings map[string]string,
	startDate, endDate string,
) string {
	// 构建人员ID到姓名的映射
	staffIDToName := BuildStaffIDToNameMapping(allStaffList)
	// 构建人员ID到shortID的映射（用于替换UUID，避免泄露给LLM）
	staffMappings, _ := BuildStaffIDMappings(allStaffList)
	var sb strings.Builder

	// 1. 排班周期
	sb.WriteString("【排班周期】\n")
	sb.WriteString(fmt.Sprintf("开始日期: %s\n", startDate))
	sb.WriteString(fmt.Sprintf("结束日期: %s\n", endDate))
	sb.WriteString("\n")

	// 2. 班次信息（按排班优先级排序）
	sb.WriteString("【班次信息】（按排班优先级排序，数值越小越优先）\n")
	if len(shifts) > 0 {
		// 按 SchedulingPriority 排序
		sortedShifts := make([]*d_model.Shift, len(shifts))
		copy(sortedShifts, shifts)
		sort.Slice(sortedShifts, func(i, j int) bool {
			return sortedShifts[i].SchedulingPriority < sortedShifts[j].SchedulingPriority
		})
		for i, shift := range sortedShifts {
			shortID := shiftMappings[shift.ID]
			if shortID == "" {
				shortID = shift.ID
			}
			sb.WriteString(fmt.Sprintf("%d. %s (ID: %s, 类型: %s, 排班优先级: %d)\n", i+1, shift.Name, shortID, shift.Type, shift.SchedulingPriority))
		}
	} else {
		sb.WriteString("无班次信息\n")
	}
	sb.WriteString("\n")

	// 3. 人数需求
	sb.WriteString("【每日人数需求】\n")
	if len(staffRequirements) > 0 {
		// 按班次分组显示
		for shiftID, dates := range staffRequirements {
			shiftName := shiftID
			for _, shift := range shifts {
				if shift.ID == shiftID {
					shiftName = shift.Name
					break
				}
			}
			shortID := shiftMappings[shiftID]
			if shortID == "" {
				shortID = shiftID // 如果不在映射表中，使用原ID
			}
			sb.WriteString(fmt.Sprintf("\n班次: %s (ID: %s)\n", shiftName, shortID))
			dateList := make([]string, 0, len(dates))
			for date := range dates {
				dateList = append(dateList, date)
			}
			// 简单排序
			for i := 0; i < len(dateList); i++ {
				for j := i + 1; j < len(dateList); j++ {
					if dateList[i] > dateList[j] {
						dateList[i], dateList[j] = dateList[j], dateList[i]
					}
				}
			}
			for _, date := range dateList {
				count := dates[date]
				sb.WriteString(fmt.Sprintf("  - %s: %d人\n", date, count))
			}
		}
	} else {
		sb.WriteString("无人数需求信息\n")
	}
	sb.WriteString("\n")

	// 4. 排班规则
	sb.WriteString("【排班规则】\n")
	if len(rules) > 0 {
		ruleIndex := 1
		for _, rule := range rules {
			if rule == nil || !rule.IsActive {
				continue
			}
			shortID := ruleMappings[rule.ID]
			if shortID == "" {
				shortID = rule.ID // 如果不在映射表中，使用原ID
			}
			sb.WriteString(fmt.Sprintf("%d. %s (ID: %s, 类型: %s)\n", ruleIndex, rule.Name, shortID, rule.RuleType))
			ruleIndex++
			if rule.Description != "" {
				// 替换描述中的UUID为shortID或名称，避免泄露给LLM
				desc := rule.Description
				desc = ReplaceIDsWithShortIDs(desc, shiftMappings)
				desc = ReplaceIDsWithShortIDs(desc, ruleMappings)
				desc = ReplaceStaffIDsWithNames(desc, staffIDToName)
				sb.WriteString(fmt.Sprintf("   描述: %s\n", desc))
			}
			if rule.RuleData != "" {
				// 替换规则内容中的UUID为shortID或名称，避免泄露给LLM
				ruleData := rule.RuleData
				ruleData = ReplaceIDsWithShortIDs(ruleData, shiftMappings)
				ruleData = ReplaceIDsWithShortIDs(ruleData, ruleMappings)
				ruleData = ReplaceStaffIDsWithNames(ruleData, staffIDToName)
				sb.WriteString(fmt.Sprintf("   规则内容: %s\n", ruleData))
			}
		}
	} else {
		sb.WriteString("无排班规则\n")
	}
	sb.WriteString("\n")

	// 5. 个人需求
	sb.WriteString("【个人需求】\n")
	if len(personalNeeds) > 0 {
		positiveNeeds := 0 // 正向需求（要求在指定日期上指定班次）
		negativeNeeds := 0 // 负向需求（要求不在指定日期上班）

		for staffID, needs := range personalNeeds {
			// 使用 staffIDToName 映射获取姓名
			staffName := staffIDToName[staffID]
			if staffName == "" {
				// 如果不在映射表中，使用shortID而不是真实UUID，避免泄露给LLM
				if shortID, ok := staffMappings[staffID]; ok {
					staffName = fmt.Sprintf("人员[%s]", shortID)
				} else {
					staffName = "未知人员" // 如果找不到，使用占位符而不是UUID
				}
			}
			sb.WriteString(fmt.Sprintf("\n人员: %s\n", staffName))
			for j, need := range needs {
				if need == nil {
					continue
				}

				// 区分正向和负向需求
				// 正向需求：明确要求在指定日期上指定班次（RequestType为prefer/must且指定了TargetShiftID）
				// 负向需求：回避某日期/某班次，或要求休息（RequestType为avoid，或未指定TargetShiftID）
				// 注意：RequestType为prefer/must但未指定TargetShiftID视为负向需求（表示休息或回避）
				isPositive := (need.RequestType == "prefer" || need.RequestType == "must") && need.TargetShiftID != ""
				if isPositive {
					positiveNeeds++
				} else {
					negativeNeeds++
				}

				// 直接使用结构体字段，格式化为语义化文本
				// 需求类型映射：permanent -> 常态化, temporary -> 临时
				needTypeText := "临时"
				if need.NeedType == "permanent" {
					needTypeText = "常态化"
				}
				// 请求类型映射：prefer -> 偏好, avoid -> 回避, must -> 必须
				requestTypeText := need.RequestType
				switch need.RequestType {
				case "prefer":
					requestTypeText = "偏好"
				case "avoid":
					requestTypeText = "回避"
				case "must":
					requestTypeText = "必须"
				}

				// 标注正向/负向需求
				needCategoryText := "负向需求"
				if isPositive {
					needCategoryText = "正向需求"
				}

				// 替换需求描述中的UUID
				needDesc := need.Description
				if needDesc != "" {
					needDesc = ReplaceIDsWithShortIDs(needDesc, shiftMappings)
					needDesc = ReplaceIDsWithShortIDs(needDesc, ruleMappings)
					needDesc = ReplaceStaffIDsWithNames(needDesc, staffIDToName)
				}
				sb.WriteString(fmt.Sprintf("  %d. [%s/%s/%s] %s\n", j+1, needTypeText, requestTypeText, needCategoryText, needDesc))
				if len(need.TargetDates) > 0 {
					sb.WriteString(fmt.Sprintf("     目标日期: %s\n", strings.Join(need.TargetDates, ", ")))
				}
				if need.TargetShiftName != "" {
					sb.WriteString(fmt.Sprintf("     目标班次: %s\n", need.TargetShiftName))
				} else if need.TargetShiftID != "" {
					// 如果只有TargetShiftID没有TargetShiftName，使用shortID而不是真实UUID
					shortShiftID := shiftMappings[need.TargetShiftID]
					if shortShiftID == "" {
						shortShiftID = need.TargetShiftID // 如果不在映射表中，使用原ID（理论上不应该发生）
					}
					sb.WriteString(fmt.Sprintf("     目标班次ID: %s\n", shortShiftID))
				}
				if need.Priority > 0 {
					sb.WriteString(fmt.Sprintf("     优先级: %d\n", need.Priority))
				}
			}
		}
		totalNeeds := positiveNeeds + negativeNeeds
		sb.WriteString(fmt.Sprintf("\n总计: %d 个个人需求（正向需求 %d 个，负向需求 %d 个）\n", totalNeeds, positiveNeeds, negativeNeeds))
		sb.WriteString("说明：\n")
		sb.WriteString("- 正向需求（RequestType为prefer/must且指定了TargetShiftID）：要求在指定日期上指定班次，应在固定排班后立即处理\n")
		sb.WriteString("- 负向需求（RequestType为avoid或未指定TargetShiftID）：要求不在指定日期上班，应在特殊班次填充和剩余填充时考虑\n")
	} else {
		sb.WriteString("无个人需求\n")
	}
	sb.WriteString("\n")

	// 6. 固定排班配置
	sb.WriteString("【固定排班配置】\n")
	if len(fixedShiftAssignments) > 0 {
		// 先收集所有固定排班人员ID，检查是否有未匹配的
		allFixedStaffIDs := make(map[string]bool)
		for _, staffIDs := range fixedShiftAssignments {
			for _, staffID := range staffIDs {
				allFixedStaffIDs[staffID] = true
			}
		}

		// 检查是否有未匹配的人员ID
		unmatchedIDs := make([]string, 0)
		for staffID := range allFixedStaffIDs {
			if _, exists := staffIDToName[staffID]; !exists {
				unmatchedIDs = append(unmatchedIDs, staffID)
			}
		}

		// 如果有未匹配的ID，在提示词中明确说明，避免LLM看到UUID
		if len(unmatchedIDs) > 0 {
			s.logger.Warn("Some fixed shift staff IDs not found in allStaffList",
				"unmatchedCount", len(unmatchedIDs),
				"totalFixedStaff", len(allFixedStaffIDs))
			// 注意：这里我们无法动态查询，所以只能跳过这些人员或使用ID
			// 但为了不让LLM看到UUID，我们选择不显示这些未匹配的人员
		}

		for date, staffIDs := range fixedShiftAssignments {
			sb.WriteString(fmt.Sprintf("日期 %s: %d 人\n", date, len(staffIDs)))
			staffNames := ReplaceStaffIDsInList(staffIDs, staffIDToName)
			// 过滤掉仍然是UUID的项（未匹配的人员）
			validNames := make([]string, 0, len(staffNames))
			for _, name := range staffNames {
				// 检查是否是UUID格式（如果ReplaceStaffIDsInList返回的还是UUID，说明未匹配）
				if !isUUID(name) {
					validNames = append(validNames, name)
				}
			}
			if len(validNames) > 0 {
				for _, name := range validNames {
					sb.WriteString(fmt.Sprintf("  - %s\n", name))
				}
			} else {
				sb.WriteString("  - （人员信息未找到）\n")
			}
		}
	} else {
		sb.WriteString("无固定排班配置\n")
	}
	sb.WriteString("\n")

	// 7. 任务规划要求
	sb.WriteString("【任务规划要求】\n")
	sb.WriteString("请根据以上信息，生成一个渐进式任务计划。\n")
	sb.WriteString("**重要说明**：固定排班已在任务计划生成前完成，所有固定排班人员已写入排班表并标记为占用。**绝对不要生成任何与固定排班填充相关的任务**。\n")
	sb.WriteString("**重要**：任务必须严格按照以下顺序排列，不能颠倒：\n")
	sb.WriteString("1. **正向需求填充**：处理\"要求在XXX日上XXX班\"类型的需求（RequestType为\"prefer\"或\"must\"且明确指定了TargetShiftID），将这些人员安排到指定日期和班次\n")
	sb.WriteString("2. **特殊班次填充**：根据上方【班次信息】中的排班优先级和【排班规则】中的约束数量，识别出规则复杂、人数需求少、对人员资质有严格要求的班次，优先占位确保合规，同时避开负向需求涉及的人员\n")
	sb.WriteString("3. **剩余人员填充**：填充常规班次的人员缺口，完善排班表，同样需要避开负向需求涉及的人员\n")
	sb.WriteString("\n")
	sb.WriteString("**任务描述要求**：\n")
	sb.WriteString("- 任务标题应该清晰描述任务类型和目标\n")
	sb.WriteString("- **绝对禁止**：生成任何标题或描述中包含\"固定\"、\"固定班\"、\"固定排班\"、\"fixed\"等关键词的任务\n")
	sb.WriteString("- 任务描述应该详细说明要解决的具体问题和涉及的范围\n")
	sb.WriteString("- 不需要指定任务类型字段，系统会根据描述自动识别\n")
	sb.WriteString("\n")
	sb.WriteString("**任务数量原则**：\n")
	sb.WriteString("- 根据实际需求自然分解任务，任务数量应与复杂度匹配\n")
	sb.WriteString("- 每个任务必须有明确的必要性和目标，不要为了满足数量要求而生成不必要的任务\n")
	sb.WriteString("- 简单场景可能只需要2-3个任务，复杂场景可能需要更多任务\n")
	sb.WriteString("- 重点确保每个任务都有意义，而非追求特定数量\n")
	sb.WriteString("\n")
	sb.WriteString("每个任务应该包含：\n")
	sb.WriteString("- 明确的目标和范围\n")
	sb.WriteString("- 涉及的班次、日期、人员\n")
	sb.WriteString("- 相关的规则ID\n")
	sb.WriteString("- 优先级\n")

	return sb.String()
}

// parseRequirementAssessmentResult 解析需求评估结果
func (s *ProgressiveSchedulingService) parseRequirementAssessmentResult(raw string) (*d_model.ProgressiveTaskPlan, error) {
	// 尝试提取JSON部分
	jsonStart := strings.Index(raw, "{")
	jsonEnd := strings.LastIndex(raw, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid JSON format: no JSON object found")
	}

	jsonStr := raw[jsonStart : jsonEnd+1]

	var result d_model.ProgressiveTaskPlan
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 验证和规范化任务
	for i, task := range result.Tasks {
		if task == nil {
			return nil, fmt.Errorf("task at index %d is nil", i)
		}

		// 确保任务ID和顺序正确
		if task.ID == "" {
			task.ID = fmt.Sprintf("task_%d", i+1)
		}
		if task.Order == 0 {
			task.Order = i + 1
		}
		if task.Status == "" {
			task.Status = "pending"
		}
		if task.Priority == 0 {
			task.Priority = 2 // 默认中等优先级
		}

		// 设置默认任务类型（如果未指定）
		if task.Type == "" {
			// 根据标题和描述推断类型（向后兼容旧数据）
			title := strings.ToLower(task.Title)
			description := strings.ToLower(task.Description)

			// 检查是否包含"校验"、"验证"等关键词
			validationKeywords := []string{"校验", "验证", "validation", "检查规则"}
			isValidation := false
			for _, keyword := range validationKeywords {
				if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
					isValidation = true
					break
				}
			}

			if isValidation {
				// 废弃的validation类型，默认改为ai
				task.Type = "ai"
				s.logger.Warn("Task contains validation keywords, setting type to 'ai'",
					"taskID", task.ID,
					"taskTitle", task.Title)
			} else {
				// 默认使用AI执行
				task.Type = "ai"
			}
		}
	}

	// 任务质量验证
	if validationErr := s.validateTaskPlanQuality(&result); validationErr != nil {
		s.logger.Warn("Task plan quality validation found issues", "error", validationErr, "tasksCount", len(result.Tasks))
		// 不阻塞执行，只记录警告
	}

	s.logger.Info("Parsed requirement assessment result",
		"tasksCount", len(result.Tasks),
		"summary", result.Summary)

	return &result, nil
}

// validateTaskPlanQuality 验证任务计划质量
func (s *ProgressiveSchedulingService) validateTaskPlanQuality(plan *d_model.ProgressiveTaskPlan) error {
	var issues []string

	// 1. 检查任务是否有意义（标题和描述不能为空）
	for i, task := range plan.Tasks {
		if task == nil {
			continue
		}
		if strings.TrimSpace(task.Title) == "" {
			issues = append(issues, fmt.Sprintf("任务 %d: 标题为空", i+1))
		}
		if strings.TrimSpace(task.Description) == "" {
			issues = append(issues, fmt.Sprintf("任务 %d: 描述为空", i+1))
		}
	}

	// 2. 检查是否有明显的重复任务（基于标题相似度）
	taskTitles := make(map[string]int)
	for i, task := range plan.Tasks {
		if task == nil {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(task.Title))

		// 检查完全匹配
		if existingIndex, exists := taskTitles[title]; exists {
			issues = append(issues, fmt.Sprintf("任务 %d 和任务 %d 标题完全相同: %s", existingIndex+1, i+1, task.Title))
		} else {
			taskTitles[title] = i
		}

		// 检查语义相似度（简单的字符串相似度检测）
		for existingTitle, existingIndex := range taskTitles {
			if existingIndex == i {
				continue // 跳过自己
			}
			similarity := calculateStringSimilarity(title, existingTitle)
			// 如果相似度超过80%，认为是重复任务
			if similarity > 0.8 {
				issues = append(issues, fmt.Sprintf("任务 %d 和任务 %d 标题高度相似（相似度 %.0f%%）: \"%s\" vs \"%s\"",
					existingIndex+1, i+1, similarity*100, plan.Tasks[existingIndex].Title, task.Title))
			}
		}
	}

	// 3. 验证任务顺序是否符合6个阶段要求（如果涉及相关阶段）
	// 这里只做基本检查，不强制所有阶段都存在
	hasFixedFill := false
	hasPersonalNeeds := false

	for _, task := range plan.Tasks {
		if task == nil {
			continue
		}
		title := strings.ToLower(task.Title)
		desc := strings.ToLower(task.Description)

		if strings.Contains(title, "固定") || strings.Contains(desc, "固定") {
			hasFixedFill = true
		}
		if strings.Contains(title, "个人需求") || strings.Contains(desc, "个人需求") {
			hasPersonalNeeds = true
		}
	}

	// 4. 检查任务顺序：固定排班应该在正向需求之前（如果两者都存在）
	// 注意：现在个人需求分为正向需求和负向需求，这里只检查正向需求
	if hasFixedFill && hasPersonalNeeds {
		fixedFillIndex := -1
		positiveNeedsIndex := -1
		specialShiftIndex := -1
		for i, task := range plan.Tasks {
			if task == nil {
				continue
			}
			title := strings.ToLower(task.Title)
			desc := strings.ToLower(task.Description)

			if fixedFillIndex == -1 && (strings.Contains(title, "固定") || strings.Contains(desc, "固定")) {
				fixedFillIndex = i
			}
			if positiveNeedsIndex == -1 && (strings.Contains(title, "正向需求") || strings.Contains(desc, "正向需求") ||
				(strings.Contains(title, "个人需求") && !strings.Contains(title, "负向"))) {
				positiveNeedsIndex = i
			}
			if specialShiftIndex == -1 && (strings.Contains(title, "特殊班次") || strings.Contains(desc, "特殊班次") ||
				strings.Contains(title, "夜班") || strings.Contains(title, "跨夜")) {
				specialShiftIndex = i
			}
		}
		// 检查固定排班不应出现（已预先完成）
		if fixedFillIndex >= 0 {
			issues = append(issues, fmt.Sprintf("❌ 严重错误: 检测到固定排班任务（任务 %d），但固定排班应该在任务计划生成前完成，不应生成此任务", fixedFillIndex+1))
		}
		// 检查正向需求应该在特殊班次填充之前
		if positiveNeedsIndex >= 0 && specialShiftIndex >= 0 && positiveNeedsIndex > specialShiftIndex {
			issues = append(issues, fmt.Sprintf("任务顺序问题: 正向需求填充（任务 %d）应该在特殊班次填充（任务 %d）之前", positiveNeedsIndex+1, specialShiftIndex+1))
		}
	}

	// 5. 检查规则校验是否在填充之后（如果两者都存在）
	// 这个检查比较复杂，暂时跳过，因为任务可能在不同阶段都有校验

	// 6. 检查任务描述相似度（避免语义重复）
	for i := 0; i < len(plan.Tasks); i++ {
		if plan.Tasks[i] == nil {
			continue
		}
		for j := i + 1; j < len(plan.Tasks); j++ {
			if plan.Tasks[j] == nil {
				continue
			}
			// 检查描述相似度
			desc1 := strings.ToLower(strings.TrimSpace(plan.Tasks[i].Description))
			desc2 := strings.ToLower(strings.TrimSpace(plan.Tasks[j].Description))
			if desc1 != "" && desc2 != "" {
				similarity := calculateStringSimilarity(desc1, desc2)
				// 如果描述相似度超过85%，认为是重复任务
				if similarity > 0.85 {
					issues = append(issues, fmt.Sprintf("任务 %d 和任务 %d 描述高度相似（相似度 %.0f%%）: \"%s\" vs \"%s\"",
						i+1, j+1, similarity*100, plan.Tasks[i].Title, plan.Tasks[j].Title))
				}
			}
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("任务质量验证发现问题: %s", strings.Join(issues, "; "))
	}

	return nil
}

// calculateStringSimilarity 计算两个字符串的相似度（使用简单的Jaccard相似度）
// 返回0-1之间的值，1表示完全相同，0表示完全不同
func calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// 使用字符级别的Jaccard相似度
	// 将字符串转换为字符集合
	set1 := make(map[rune]bool)
	set2 := make(map[rune]bool)

	for _, r := range s1 {
		set1[r] = true
	}
	for _, r := range s2 {
		set2[r] = true
	}

	// 计算交集和并集
	intersection := 0
	union := len(set1)

	for r := range set2 {
		if set1[r] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	// Jaccard相似度 = 交集大小 / 并集大小
	return float64(intersection) / float64(union)
}

// sortTasksByShiftPriority 按班次排班优先级对任务进行代码排序
// 排序规则：
// 1. 先按任务的 order（阶段）排序，保持LLM输出的阶段顺序
// 2. 同 order 内，按 targetShifts 中最高优先级（最小 SchedulingPriority）排序
func sortTasksByShiftPriority(plan *d_model.ProgressiveTaskPlan, shifts []*d_model.Shift) {
	if plan == nil || len(plan.Tasks) <= 1 || len(shifts) == 0 {
		return
	}

	// 构建班次ID → 优先级映射
	shiftPriorityMap := make(map[string]int)
	for _, shift := range shifts {
		shiftPriorityMap[shift.ID] = shift.SchedulingPriority
	}

	// 获取任务关联的最高班次优先级（最小值）
	getTaskShiftPriority := func(task *d_model.ProgressiveTask) int {
		if task == nil || len(task.TargetShifts) == 0 {
			return 9999 // 无关联班次的任务排最后
		}
		minPriority := 9999
		for _, shiftID := range task.TargetShifts {
			if p, ok := shiftPriorityMap[shiftID]; ok && p < minPriority {
				minPriority = p
			}
		}
		return minPriority
	}

	// 排序：先按 order（阶段），再按班次优先级
	sort.SliceStable(plan.Tasks, func(i, j int) bool {
		ti, tj := plan.Tasks[i], plan.Tasks[j]
		if ti.Order != tj.Order {
			return ti.Order < tj.Order
		}
		return getTaskShiftPriority(ti) < getTaskShiftPriority(tj)
	})

	// 重新编号 order，确保连续
	for i, task := range plan.Tasks {
		if task != nil {
			task.Order = i + 1
		}
	}
}
