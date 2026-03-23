package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/workflow/wsbridge"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	common_config "jusha/mcp/pkg/config"
)

// intentService 意图识别服务实现
// 职责：初始意图识别（识别一级意图并映射到工作流）
// 实现接口：IIntentService, session.IIntentRecognizer, wsbridge.IIntentEventMapper
//
// 注意：工作流内的意图分析（如 schedule.adjust 中的调整意图分析）
// 应由 ISchedulingAIService.AnalyzeAdjustIntent 等专用方法处理
type intentService struct {
	logger           logging.ILogger
	sessionService   session.ISessionService
	configurator     config.IRosteringConfigurator
	aiFactory        *ai.AIProviderFactory
	baseConfigurator common_config.IServiceConfigurator
}

// NewIntentService 创建意图识别服务
func NewIntentService(
	logger logging.ILogger,
	sessionService session.ISessionService,
	cfg config.IRosteringConfigurator,
) (d_service.IIntentService, error) {
	if cfg == nil {
		return nil, errors.New("configurator is required for intent service")
	}
	base := cfg.(common_config.IServiceConfigurator)
	if base.GetBaseConfig().AI == nil {
		return nil, errors.New("AI configuration missing for intent detection")
	}
	factory := ai.NewAIModelFactory(context.Background(), base, logger)
	return &intentService{
		logger:           logger.With("component", "IntentService"),
		sessionService:   sessionService,
		configurator:     cfg,
		aiFactory:        factory,
		baseConfigurator: base,
	}, nil
}

// ============================================================
// 意图识别接口实现 (session.IIntentRecognizer)
// ============================================================

// Recognize 识别用户意图（仅用于初始意图识别）
// 当用户开始新对话时，识别其一级意图类型（schedule/rule/dept/general）
func (s *intentService) Recognize(ctx context.Context, req session.IntentRecognizeRequest) (*session.IntentRecognizeResponse, error) {
	s.logger.Info("Recognizing initial intent", "sessionId", req.SessionID, "messageCount", len(req.Context.Messages))

	// 获取 Session
	sess, err := s.sessionService.Get(ctx, req.SessionID)
	if err != nil {
		s.logger.Error("Failed to get session", "error", err, "sessionId", req.SessionID)
		return nil, fmt.Errorf("get session failed: %w", err)
	}

	// 检查是否有消息
	if len(sess.Messages) == 0 {
		s.logger.Warn("No messages to analyze", "sessionId", req.SessionID)
		return s.buildUnknownIntentResponse(req.UserMessage, "no_messages"), nil
	}

	// 获取配置
	cfg := s.configurator.GetConfig()
	maxHistory := cfg.Intent.MaxHistory
	if maxHistory <= 0 || maxHistory > 20 {
		maxHistory = 8
	}

	// 取最近的消息
	msgs := sess.Messages
	if len(msgs) > maxHistory {
		msgs = msgs[len(msgs)-maxHistory:]
	}

	// 使用 initial 策略进行识别
	intent, results, aiResp, err := s.recognizeInitialIntent(ctx, msgs)
	if err != nil {
		s.logger.Error("Intent recognition failed", "error", err)
		return nil, fmt.Errorf("intent recognition failed: %w", err)
	}

	if intent == nil {
		s.logger.Warn("All intents have low confidence", "sessionId", req.SessionID)
		return s.buildUnknownIntentResponse(req.UserMessage, "low_confidence"), nil
	}

	// 保存识别结果到 Session
	if err := s.saveIntentToSession(ctx, sess, intent, results, aiResp); err != nil {
		s.logger.Error("Failed to save intent to session", "error", err)
		// 不阻断流程，继续返回结果
	}

	// 构建响应
	response := &session.IntentRecognizeResponse{
		Intent: &session.Intent{
			Type:       string(intent.Type),
			Confidence: intent.Confidence,
			RawText:    req.UserMessage,
			Entities:   intent.Arguments,
			Metadata: map[string]any{
				"summary":  intent.Summary,
				"strategy": "initial",
				"model":    s.aiFactory.GetDefaultProviderNameSafe(),
			},
		},
		NeedsMoreInfo: false,
		Suggestions:   []string{},
		MissingFields: []string{},
	}

	// 添加 Session 信息
	if sess != nil {
		response.Intent.Metadata["currentState"] = sess.State
		if sess.WorkflowMeta != nil {
			response.Intent.Metadata["workflow"] = sess.WorkflowMeta.Workflow
			response.Intent.Metadata["phase"] = sess.WorkflowMeta.Phase
		}
	}

	s.logger.Info("Initial intent recognized",
		"sessionId", req.SessionID,
		"intentType", response.Intent.Type,
		"confidence", response.Intent.Confidence)

	return response, nil
}

// ValidateIntent 验证意图的有效性
func (s *intentService) ValidateIntent(ctx context.Context, intent *session.Intent) ([]string, error) {
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}

	intentType := d_model.IntentType(intent.Type)

	switch intentType {
	case d_model.IntentSchedule, d_model.IntentScheduleCreate, d_model.IntentScheduleAdjust, d_model.IntentScheduleQuery:
		return s.validateScheduleIntent(intent)

	case d_model.IntentRule, d_model.IntentRuleCreate, d_model.IntentRuleUpdate, d_model.IntentRuleQuery, d_model.IntentRuleDelete:
		return s.validateRuleIntent(intent)

	case d_model.IntentDept,
		d_model.IntentDeptStaffCreate, d_model.IntentDeptStaffUpdate, d_model.IntentDeptStaffStatus, d_model.IntentDeptStaffDelete,
		d_model.IntentDeptTeamCreate, d_model.IntentDeptTeamUpdate, d_model.IntentDeptTeamAssign,
		d_model.IntentDeptSkillCreate, d_model.IntentDeptSkillUpdate, d_model.IntentDeptSkillAssign,
		d_model.IntentDeptLocationCreate, d_model.IntentDeptLocationUpdate:
		return s.validateDeptIntent(intent)

	case d_model.IntentGeneral, d_model.IntentGeneralHelp, d_model.IntentGeneralSmallTalk:
		return []string{}, nil

	case d_model.IntentUnknown:
		return nil, fmt.Errorf("unknown intent type: %s", intent.Type)

	default:
		return nil, fmt.Errorf("unsupported intent type: %s", intent.Type)
	}
}

// SupportedIntents 返回支持的意图类型列表
func (s *intentService) SupportedIntents() []string {
	return []string{
		// 一级意图
		string(d_model.IntentSchedule),
		string(d_model.IntentRule),
		string(d_model.IntentDept),
		string(d_model.IntentGeneral),

		// 二级意图 - 排班类
		string(d_model.IntentScheduleCreate),
		string(d_model.IntentScheduleAdjust),
		string(d_model.IntentScheduleQuery),

		// 二级意图 - 规则类
		string(d_model.IntentRuleCreate),
		string(d_model.IntentRuleUpdate),
		string(d_model.IntentRuleQuery),
		string(d_model.IntentRuleDelete),

		// 二级意图 - 科室类
		string(d_model.IntentDeptStaffCreate),
		string(d_model.IntentDeptStaffUpdate),
		string(d_model.IntentDeptStaffStatus),
		string(d_model.IntentDeptStaffDelete),
		string(d_model.IntentDeptTeamCreate),
		string(d_model.IntentDeptTeamUpdate),
		string(d_model.IntentDeptTeamAssign),
		string(d_model.IntentDeptSkillCreate),
		string(d_model.IntentDeptSkillUpdate),
		string(d_model.IntentDeptSkillAssign),
		string(d_model.IntentDeptLocationCreate),
		string(d_model.IntentDeptLocationUpdate),

		// 二级意图 - 通用类
		string(d_model.IntentGeneralHelp),
		string(d_model.IntentGeneralSmallTalk),
	}
}

// ============================================================
// 事件映射接口实现 (wsbridge.IIntentEventMapper)
// ============================================================

// MapIntentToEvent 将意图类型映射到工作流事件
// 使用 domain/model/intent.go 中定义的静态映射表
func (s *intentService) MapIntentToEvent(intentType string) engine.Event {
	intent := d_model.IntentType(intentType)

	s.logger.Debug("Mapping intent to event", "intentType", intentType)

	// 使用映射表查找
	mapping := d_model.GetWorkflowMapping(intent)
	if mapping == nil {
		s.logger.Warn("No workflow mapping found", "intentType", intentType)
		return engine.Event("")
	}

	// 检查是否已实现
	if !mapping.Implemented {
		s.logger.Warn("Workflow not implemented yet",
			"intentType", intentType,
			"workflow", mapping.WorkflowName)
		return engine.Event("")
	}

	// 返回触发事件
	if mapping.Event == "" {
		s.logger.Info("Intent does not trigger workflow", "intentType", intentType)
		return engine.Event("")
	}

	// 使用 engine.EventStart 作为标准起始事件
	if mapping.Event == "start" {
		return engine.EventStart
	}
	return engine.Event(mapping.Event)
}

// MapIntentToWorkflow 将意图类型映射到完整的工作流信息
// 返回包含 WorkflowName, Event, Implemented 的映射结果
func (s *intentService) MapIntentToWorkflow(intentType string) *wsbridge.IntentWorkflowMapping {
	intent := d_model.IntentType(intentType)

	s.logger.Debug("Mapping intent to workflow", "intentType", intentType)

	// 使用映射表查找
	mapping := d_model.GetWorkflowMapping(intent)
	if mapping == nil {
		s.logger.Warn("No workflow mapping found", "intentType", intentType)
		return nil
	}

	// 转换事件
	var event engine.Event
	if mapping.Event == "start" {
		event = engine.EventStart
	} else if mapping.Event != "" {
		event = engine.Event(mapping.Event)
	}

	result := &wsbridge.IntentWorkflowMapping{
		WorkflowName: mapping.WorkflowName,
		Event:        event,
		Implemented:  mapping.Implemented,
	}

	return result
}

// ============================================================
// 核心识别方法
// ============================================================

// recognizeInitialIntent 使用 initial 策略识别初始意图
func (s *intentService) recognizeInitialIntent(ctx context.Context, msgs []session.Message) (*d_model.IntentResult, []*d_model.IntentResult, *ai.AIResponse, error) {
	cfg := s.configurator.GetConfig()

	// 获取 initial 策略配置
	strategy, ok := cfg.Intent.Strategies["initial"]
	if !ok {
		s.logger.Warn("Initial strategy not found, using default prompt")
		return s.recognizeWithDefaultPrompt(ctx, msgs)
	}

	// 构建系统提示词
	systemPrompt := strings.TrimSpace(strategy.SystemPrompt)
	if systemPrompt == "" {
		s.logger.Warn("Empty system prompt for initial strategy")
		systemPrompt = s.defaultIntentSystemPrompt()
	}

	// 替换日期占位符
	currentDate := time.Now().Format("2006-01-02")
	systemPrompt = strings.ReplaceAll(systemPrompt, "{currentDate}", currentDate)

	// 构建用户提示词（仅对话历史）
	userPrompt := s.buildUserPrompt(msgs)

	// 选择 AI 模型
	var model *common_config.AIModelProvider
	if strategy.Model != nil && strategy.Model.Provider != "" && strategy.Model.Name != "" {
		model = strategy.Model
	}

	// 调用 AI
	resp, err := s.aiFactory.CallWithModel(ctx, model, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("AI call failed: %w", err)
	}

	s.logger.Debug("AI response received", "contentLength", len(resp.Content))

	// 解析 AI 输出
	raw := strings.TrimSpace(resp.Content)
	results, parseErr := s.parseIntentJSON(raw)
	if parseErr != nil {
		s.logger.Error("Failed to parse intent JSON", "error", parseErr, "raw", raw)
		return nil, nil, &resp, fmt.Errorf("parse intent json failed: %w", parseErr)
	}

	// 选择置信度最高的意图（保留原始意图类型，不再归一化）
	mainIntent := s.selectMainIntent(results)

	return mainIntent, results, &resp, nil
}

// recognizeWithDefaultPrompt 使用默认提示词识别意图
func (s *intentService) recognizeWithDefaultPrompt(ctx context.Context, msgs []session.Message) (*d_model.IntentResult, []*d_model.IntentResult, *ai.AIResponse, error) {
	systemPrompt := s.defaultIntentSystemPrompt()
	userPrompt := s.buildUserPrompt(msgs)

	resp, err := s.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("AI call failed: %w", err)
	}

	raw := strings.TrimSpace(resp.Content)
	results, parseErr := s.parseIntentJSON(raw)
	if parseErr != nil {
		return nil, nil, &resp, fmt.Errorf("parse intent json failed: %w", parseErr)
	}

	// 保留原始意图类型，不再归一化
	mainIntent := s.selectMainIntent(results)
	return mainIntent, results, &resp, nil
}

// ============================================================
// 辅助方法
// ============================================================

// buildUnknownIntentResponse 构建未知意图响应
func (s *intentService) buildUnknownIntentResponse(rawText string, reason string) *session.IntentRecognizeResponse {
	return &session.IntentRecognizeResponse{
		Intent: &session.Intent{
			Type:       string(d_model.IntentUnknown),
			Confidence: 0.0,
			RawText:    rawText,
			Entities:   map[string]any{},
			Metadata:   map[string]any{"reason": reason},
		},
		NeedsMoreInfo: false,
		Suggestions:   []string{},
		MissingFields: []string{},
	}
}

// selectMainIntent 选择置信度最高的意图
func (s *intentService) selectMainIntent(results []*d_model.IntentResult) *d_model.IntentResult {
	if len(results) == 0 {
		return nil
	}

	var mainIntent *d_model.IntentResult
	maxConfidence := 0.0

	for _, result := range results {
		if result == nil || result.Type == d_model.IntentUnknown {
			continue
		}
		if result.Confidence > maxConfidence {
			maxConfidence = result.Confidence
			mainIntent = result
		}
	}

	return mainIntent
}

// saveIntentToSession 保存意图识别结果到 Session
func (s *intentService) saveIntentToSession(
	ctx context.Context,
	sess *session.Session,
	mainIntent *d_model.IntentResult,
	results []*d_model.IntentResult,
	_ *ai.AIResponse,
) error {
	// 获取或创建业务上下文
	var schedCtx *d_model.SchedulingContext
	if ctxRaw, ok := sess.Data[d_model.DataKeyContext]; ok {
		schedCtx, _ = ctxRaw.(*d_model.SchedulingContext)
	}
	if schedCtx == nil {
		schedCtx = &d_model.SchedulingContext{
			Normalized: map[string]any{},
			Extra:      map[string]any{},
		}
		sess.Data[d_model.DataKeyContext] = schedCtx
	}
	if schedCtx.Extra == nil {
		schedCtx.Extra = map[string]any{}
	}

	// 保存意图识别结果
	payload := &d_model.IntentDetectionPayload{
		MessageIndex: len(sess.Messages) - 1,
		Results:      results,
		Model:        s.aiFactory.GetDefaultProviderNameSafe(),
	}

	// 保留历史
	if histRaw, ok := schedCtx.Extra["intentHistory"].([]*d_model.IntentDetectionPayload); ok {
		schedCtx.Extra["intentHistory"] = append(histRaw, payload)
	} else {
		schedCtx.Extra["intentHistory"] = []*d_model.IntentDetectionPayload{payload}
	}

	// 保存主意图
	schedCtx.Extra["mainIntent"] = mainIntent

	// 如果是排班类意图，尝试从 arguments 填充业务字段
	if mainIntent.Type == d_model.IntentSchedule || mainIntent.Type == d_model.IntentScheduleCreate {
		if br, ok := mainIntent.Arguments["bizDateRange"].(string); ok {
			if _, exists := sess.Data[d_model.DataKeyBizDateRange]; !exists || sess.Data[d_model.DataKeyBizDateRange] == "" {
				sess.Data[d_model.DataKeyBizDateRange] = br
			}
		}
		if dep, ok := mainIntent.Arguments["department"].(string); ok {
			if _, exists := sess.Data[d_model.DataKeyDepartment]; !exists || sess.Data[d_model.DataKeyDepartment] == "" {
				sess.Data[d_model.DataKeyDepartment] = dep
			}
		}
		if mod, ok := mainIntent.Arguments["modality"].(string); ok {
			if _, exists := sess.Data[d_model.DataKeyModality]; !exists || sess.Data[d_model.DataKeyModality] == "" {
				sess.Data[d_model.DataKeyModality] = mod
			}
		}
	}

	// 更新 Session
	_, err := s.sessionService.Update(ctx, sess.ID, func(s *session.Session) error {
		*s = *sess
		return nil
	})
	return err
}

// validateScheduleIntent 验证排班类意图
func (s *intentService) validateScheduleIntent(intent *session.Intent) ([]string, error) {
	// 排班类意图的必填字段在工作流中收集，这里不强制验证
	return []string{}, nil
}

// validateRuleIntent 验证规则类意图
func (s *intentService) validateRuleIntent(intent *session.Intent) ([]string, error) {
	return []string{}, nil
}

// validateDeptIntent 验证科室类意图
func (s *intentService) validateDeptIntent(intent *session.Intent) ([]string, error) {
	return []string{}, nil
}

// normalizeToMainIntentType 归一化到一级意图类型
func (s *intentService) normalizeToMainIntentType(intentType d_model.IntentType) d_model.IntentType {
	switch intentType {
	case d_model.IntentScheduleCreate, d_model.IntentScheduleAdjust, d_model.IntentScheduleQuery:
		return d_model.IntentSchedule
	case d_model.IntentRuleCreate, d_model.IntentRuleUpdate, d_model.IntentRuleQuery, d_model.IntentRuleDelete:
		return d_model.IntentRule
	case d_model.IntentDeptStaffCreate, d_model.IntentDeptStaffUpdate, d_model.IntentDeptStaffStatus, d_model.IntentDeptStaffDelete,
		d_model.IntentDeptTeamCreate, d_model.IntentDeptTeamUpdate, d_model.IntentDeptTeamAssign,
		d_model.IntentDeptSkillCreate, d_model.IntentDeptSkillUpdate, d_model.IntentDeptSkillAssign,
		d_model.IntentDeptLocationCreate, d_model.IntentDeptLocationUpdate:
		return d_model.IntentDept
	case d_model.IntentGeneralHelp, d_model.IntentGeneralSmallTalk:
		return d_model.IntentGeneral
	default:
		return intentType
	}
}

// parseIntentJSON 解析 AI 返回的意图 JSON
func (s *intentService) parseIntentJSON(raw string) ([]*d_model.IntentResult, error) {
	// 尝试提取 JSON 数组 (处理 markdown 代码块)
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

	jsonStr = strings.TrimSpace(jsonStr)

	var results []*d_model.IntentResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("unmarshal intent results failed: %w", err)
	}

	return results, nil
}

// buildUserPrompt 构建用户提示词（仅对话历史）
func (s *intentService) buildUserPrompt(msgs []session.Message) string {
	var sb strings.Builder

	sb.WriteString("## 对话历史：\n")
	for i, msg := range msgs {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, role, msg.Content))
	}

	sb.WriteString("\n## 请分析：\n")
	sb.WriteString("分析用户的主要意图类型（schedule/rule/dept/general/unknown）。\n")

	return sb.String()
}

// defaultIntentSystemPrompt 默认的意图识别系统提示词
func (s *intentService) defaultIntentSystemPrompt() string {
	return `你是一个专业的意图识别助手。你的任务是分析用户的对话，识别其意图。

请返回 JSON 数组格式，每个元素包含：
- type: 意图类型 (schedule/rule/dept/general/unknown)
- confidence: 置信度 (0.0-1.0)
- summary: 简短描述
- arguments: 提取的参数 (如 bizDateRange, department, modality 等)

示例：
[
  {
    "type": "schedule",
    "confidence": 0.95,
    "summary": "用户想要创建下周的排班",
    "arguments": {
      "bizDateRange": "下周",
      "department": "急诊科"
    }
  }
]

支持的意图类型：
- schedule: 排班相关 (创建、调整、查询排班)
- rule: 规则相关 (创建、更新、查询、删除规则)
- dept: 科室管理 (员工、团队、技能、地点管理)
- general: 通用对话 (帮助、闲聊)
- unknown: 无法识别

请只返回 JSON 数组，不要包含其他内容。`
}
