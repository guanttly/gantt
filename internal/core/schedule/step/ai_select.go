package step

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/core/rule/checker"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AISelectStep 使用 AI 为填充阶段选择最优候选人。
type AISelectStep struct {
	Provider      ai.Provider
	Selector      ai.ProviderSelector
	ModelResolver ai.NodeModelResolver
	Logger        *zap.Logger
}

func (s *AISelectStep) Name() string { return "ai_select" }

func (s *AISelectStep) Execute(ctx context.Context, state *ScheduleState) error {
	logger := s.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	if s.Provider == nil {
		logger.Warn("AI provider 未配置，跳过 AI 选人步骤")
		return nil
	}

	// 收集空缺：遍历需求表，找出尚未填满的班次-日期组合
	type vacancy struct {
		Date    string `json:"date"`
		ShiftID string `json:"shift_id"`
		Needed  int    `json:"needed"`
	}
	var vacancies []vacancy

	if state.Config == nil || len(state.Config.Requirements) == 0 {
		return nil
	}

	for shiftID, dateCounts := range state.Config.Requirements {
		for dateStr, needed := range dateCounts {
			assigned := state.CountAssigned(shiftID, dateStr)
			if assigned < needed {
				vacancies = append(vacancies, vacancy{
					Date:    dateStr,
					ShiftID: shiftID,
					Needed:  needed - assigned,
				})
			}
		}
	}

	if len(vacancies) == 0 {
		logger.Info("没有空缺，跳过 AI 选人步骤")
		return nil
	}

	// 收集所有候选人 ID（去重）
	candidateSet := make(map[string]bool)
	for _, candidates := range state.Candidates {
		for _, id := range candidates {
			candidateSet[id] = true
		}
	}
	var candidateIDs []string
	for id := range candidateSet {
		candidateIDs = append(candidateIDs, id)
	}

	vacancyJSON, _ := json.Marshal(vacancies)
	candidateJSON, _ := json.Marshal(candidateIDs)

	prompt := fmt.Sprintf(
		"Vacancies:\n%s\n\nCandidate employee IDs:\n%s\n\nReturn JSON array: [{\"employee_id\":\"...\",\"shift_id\":\"...\",\"date\":\"YYYY-MM-DD\"}]",
		string(vacancyJSON),
		string(candidateJSON),
	)

	systemPrompt := "You are a scheduling engine. Given vacancies and candidates, produce optimal shift assignments as a JSON array. Each employee can only work one shift per day. Distribute workload evenly."

	provider := s.Provider
	req := ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	}
	ctx, cancel, provider, req := ai.ApplyNodeModelConfig(ctx, provider, s.Selector, s.ModelResolver, ai.AppScheduling, ai.WorkflowScheduleCreate, ai.NodeAISelect, req, 180*time.Second, logger)
	defer cancel()

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		logger.Error("AI 选人调用失败", zap.Error(err))
		return nil // 不中断 pipeline，后续 PhaseTwo 会兜底
	}

	// 解析 AI 返回的分配方案
	var aiAssignments []struct {
		EmployeeID string `json:"employee_id"`
		ShiftID    string `json:"shift_id"`
		Date       string `json:"date"`
	}

	content := resp.Content
	// 尝试提取 JSON 数组
	start := -1
	end := -1
	for i, c := range content {
		if c == '[' {
			start = i
			break
		}
	}
	if start >= 0 {
		depth := 0
		for i := start; i < len(content); i++ {
			if content[i] == '[' {
				depth++
			} else if content[i] == ']' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
	}

	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(content[start:end]), &aiAssignments); err != nil {
			logger.Warn("AI 返回内容解析失败", zap.Error(err), zap.String("content", content))
			return nil
		}
	} else {
		logger.Warn("AI 返回内容中未找到 JSON 数组", zap.String("content", content))
		return nil
	}

	// 构建已有排班 checker.Assignment 列表
	checkerAssignments := make([]checker.Assignment, 0, len(state.Assignments))
	for _, a := range state.Assignments {
		d, _ := time.Parse("2006-01-02", a.Date)
		checkerAssignments = append(checkerAssignments, checker.Assignment{
			EmployeeID: a.EmployeeID,
			ShiftID:    a.ShiftID,
			Date:       d,
		})
	}

	// 验证 AI 分配并添加到 state
	accepted := 0
	for _, aa := range aiAssignments {
		key := aa.ShiftID + "|" + aa.Date
		if !containsCandidate(state.Candidates[key], aa.EmployeeID) {
			continue
		}
		if state.IsOccupied(aa.EmployeeID, aa.Date) {
			continue
		}
		if needed := requiredCount(state, aa.ShiftID, aa.Date); needed > 0 && state.CountAssigned(aa.ShiftID, aa.Date) >= needed {
			continue
		}

		linkedPlans, ok := planRequiredTogetherAssignments(state, aa.EmployeeID, aa.ShiftID, aa.Date)
		if !ok {
			continue
		}

		plannedAssignments := make([]Assignment, 0, len(linkedPlans)+1)
		for _, linkedPlan := range linkedPlans {
			plannedAssignments = append(plannedAssignments, Assignment{
				ID:         uuid.New().String(),
				ScheduleID: state.ScheduleID,
				EmployeeID: aa.EmployeeID,
				ShiftID:    linkedPlan.ShiftID,
				Date:       linkedPlan.Date,
				Source:     SourceRule,
			})
		}
		plannedAssignments = append(plannedAssignments, Assignment{
			ID:         uuid.New().String(),
			ScheduleID: state.ScheduleID,
			EmployeeID: aa.EmployeeID,
			ShiftID:    aa.ShiftID,
			Date:       aa.Date,
			Source:     SourceAI,
		})

		nextAssignments, violations := validatePlannedAssignments(ctx, state, checkerAssignments, plannedAssignments)
		if len(violations) > 0 {
			state.Violations = append(state.Violations, violations...)
			continue
		}

		state.Assignments = append(state.Assignments, plannedAssignments...)
		checkerAssignments = nextAssignments
		accepted++
	}

	logger.Info("AI 选人完成",
		zap.Int("ai_suggested", len(aiAssignments)),
		zap.Int("accepted", accepted),
		zap.Int("rejected", len(aiAssignments)-accepted),
	)

	return nil
}
