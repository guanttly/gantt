package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"jusha/agent/rostering/config"
	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
	"jusha/mcp/pkg/ai"
	common_config "jusha/mcp/pkg/config"
	"jusha/mcp/pkg/logging"

	"github.com/google/uuid"
)

// ============================================================
// 全局评审服务实现
// ============================================================

// globalReviewService 全局评审服务实现
type globalReviewService struct {
	logger           logging.ILogger
	configurator     config.IRosteringConfigurator
	aiFactory        *ai.AIProviderFactory
	baseConfigurator common_config.IServiceConfigurator
}

// NewGlobalReviewService 创建全局评审服务
func NewGlobalReviewService(
	logger logging.ILogger,
	cfg config.IRosteringConfigurator,
) (d_service.IGlobalReviewService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configurator is required for global review service")
	}
	base := cfg.(common_config.IServiceConfigurator)
	if base.GetBaseConfig().AI == nil {
		return nil, fmt.Errorf("AI configuration missing for global review service")
	}
	factory := ai.NewAIModelFactory(context.Background(), base, logger)
	return &globalReviewService{
		logger:           logger.With("component", "GlobalReviewService"),
		configurator:     cfg,
		aiFactory:        factory,
		baseConfigurator: base,
	}, nil
}

// ============================================================
// 评审项合并
// ============================================================

// MergeToReviewItems 将规则和个人需求合并为统一的评审项列表
func (s *globalReviewService) MergeToReviewItems(rules []*d_model.Rule, personalNeeds []*d_model.PersonalNeed) []*d_model.ReviewItem {
	items := make([]*d_model.ReviewItem, 0, len(rules)+len(personalNeeds))

	// 添加规则评审项
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		item := &d_model.ReviewItem{
			ID:          fmt.Sprintf("rule_%s", rule.ID),
			Type:        d_model.ReviewItemTypeRule,
			Name:        rule.Name,
			Description: rule.Description,
			Priority:    rule.Priority,
			SourceRule:  rule,
		}
		// 从规则关联中提取受影响的人员和班次
		if len(rule.Associations) > 0 {
			for _, assoc := range rule.Associations {
				switch assoc.AssociationType {
				case "employee":
					item.AffectedStaffIDs = append(item.AffectedStaffIDs, assoc.AssociationID)
				case "shift":
					item.AffectedShiftIDs = append(item.AffectedShiftIDs, assoc.AssociationID)
				}
			}
		}
		items = append(items, item)
	}

	// 添加个人需求评审项
	for _, need := range personalNeeds {
		if need == nil {
			continue
		}
		item := &d_model.ReviewItem{
			ID:               fmt.Sprintf("need_%s_%s", need.StaffID, uuid.New().String()[:8]),
			Type:             d_model.ReviewItemTypePersonalNeed,
			Name:             fmt.Sprintf("%s的%s需求", need.StaffName, s.getRequestTypeText(need.RequestType)),
			Description:      need.Description,
			Priority:         need.Priority,
			SourceNeed:       need,
			AffectedStaffIDs: []string{need.StaffID},
			AffectedDates:    need.TargetDates,
		}
		if need.TargetShiftID != "" {
			item.AffectedShiftIDs = []string{need.TargetShiftID}
		}
		items = append(items, item)
	}

	// 按优先级排序，相同优先级时规则优先于个人需求
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority // 数字小的优先级高
		}
		// 相同优先级，规则优先
		if items[i].Type == d_model.ReviewItemTypeRule && items[j].Type == d_model.ReviewItemTypePersonalNeed {
			return true
		}
		return false
	})

	return items
}

// getRequestTypeText 获取请求类型的文本描述
func (s *globalReviewService) getRequestTypeText(requestType string) string {
	switch requestType {
	case "prefer":
		return "偏好"
	case "avoid":
		return "回避"
	case "must":
		return "必须"
	default:
		return requestType
	}
}

// ============================================================
// 逐条评审
// ============================================================

// ReviewItemAgainstDraft 逐条评审项对照排班表评审
func (s *globalReviewService) ReviewItemAgainstDraft(
	ctx context.Context,
	item *d_model.ReviewItem,
	draft *d_model.ScheduleDraft,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) (*d_model.ModificationOpinion, error) {
	// 构建评审prompt
	systemPrompt := s.buildReviewSystemPrompt()
	userPrompt := s.buildReviewUserPrompt(item, draft, allStaffList, allShifts)

	// 调用LLM
	modelConfig := s.getModelConfig("globalReview")
	resp, err := s.aiFactory.CallWithModel(ctx, modelConfig, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("调用LLM评审失败: %w", err)
	}

	// 解析响应
	opinion, err := s.parseReviewResponse(resp.Content, item)
	if err != nil {
		s.logger.Warn("解析评审响应失败，创建默认意见",
			"error", err,
			"itemID", item.ID,
			"rawContent", resp.Content,
		)
		// 返回默认的"通过"意见
		opinion = &d_model.ModificationOpinion{
			ID:             fmt.Sprintf("opinion_%s", item.ID),
			ReviewItemID:   item.ID,
			ReviewItemType: item.Type,
			ReviewItemName: item.Name,
			IsViolated:     false,
			Status:         d_model.OpinionStatusApproved,
			Severity:       "low",
		}
	}

	return opinion, nil
}

// buildReviewSystemPrompt 构建评审系统提示词
func (s *globalReviewService) buildReviewSystemPrompt() string {
	return `你是一个专业的排班规则评审专家。你的职责是：
1. 仔细检查给定的规则或个人需求是否在排班表中被正确满足
2. 如果发现违规情况，详细说明违规的具体内容
3. 提出具体的修改建议，包括需要调整的日期、人员和班次

请以JSON格式输出评审结果：
{
  "isViolated": true/false,
  "violationDescription": "违规描述（如果违规）",
  "suggestion": "修改建议（如果违规）",
  "severity": "critical/warning/low",
  "proposedChanges": [
    {
      "date": "YYYY-MM-DD",
      "shiftId": "班次ID",
      "addStaffIds": ["需要添加的人员ID"],
      "removeStaffIds": ["需要移除的人员ID"],
      "reason": "变更原因"
    }
  ],
  "affectedStaffIds": ["受影响的人员ID列表"],
  "affectedDates": ["受影响的日期列表"]
}

注意：
- 如果规则/需求已满足，isViolated应为false，其他字段可省略
- severity说明：critical=严重违规必须修复，warning=建议修复，low=轻微问题
- proposedChanges中的修改建议应该是具体可执行的`
}

// buildReviewUserPrompt 构建评审用户提示词
func (s *globalReviewService) buildReviewUserPrompt(
	item *d_model.ReviewItem,
	draft *d_model.ScheduleDraft,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) string {
	var sb strings.Builder

	// 评审项信息
	sb.WriteString("## 待评审项\n\n")
	sb.WriteString(fmt.Sprintf("- **类型**: %s\n", s.getItemTypeText(item.Type)))
	sb.WriteString(fmt.Sprintf("- **名称**: %s\n", item.Name))
	sb.WriteString(fmt.Sprintf("- **描述**: %s\n", item.Description))
	sb.WriteString(fmt.Sprintf("- **优先级**: %d\n", item.Priority))

	// 如果是规则，添加规则详情
	if item.Type == d_model.ReviewItemTypeRule && item.SourceRule != nil {
		rule := item.SourceRule
		sb.WriteString(fmt.Sprintf("- **规则类型**: %s\n", rule.RuleType))
		if rule.MaxCount != nil {
			sb.WriteString(fmt.Sprintf("- **最大次数**: %d\n", *rule.MaxCount))
		}
		if rule.ConsecutiveMax != nil {
			sb.WriteString(fmt.Sprintf("- **连续最大天数**: %d\n", *rule.ConsecutiveMax))
		}
		if rule.MinRestDays != nil {
			sb.WriteString(fmt.Sprintf("- **最小休息天数**: %d\n", *rule.MinRestDays))
		}
	}

	// 如果是个人需求，添加需求详情
	if item.Type == d_model.ReviewItemTypePersonalNeed && item.SourceNeed != nil {
		need := item.SourceNeed
		sb.WriteString(fmt.Sprintf("- **人员**: %s (%s)\n", need.StaffName, need.StaffID))
		sb.WriteString(fmt.Sprintf("- **请求类型**: %s\n", s.getRequestTypeText(need.RequestType)))
		if need.TargetShiftName != "" {
			sb.WriteString(fmt.Sprintf("- **目标班次**: %s\n", need.TargetShiftName))
		}
		if len(need.TargetDates) > 0 {
			sb.WriteString(fmt.Sprintf("- **目标日期**: %s\n", strings.Join(need.TargetDates, ", ")))
		}
	}

	// 预先确定需要输出的范围
	relevantShiftIDs := s.getRelevantShiftIDs(item, allShifts)
	relevantDates := s.getRelevantDates(item, draft)
	relevantStaffIDs := s.getRelevantStaffIDs(item)

	// 构建人员映射（用于后续使用）
	staffMap := make(map[string]string)
	for _, staff := range allStaffList {
		if staff != nil {
			staffMap[staff.ID] = staff.Name
		}
	}

	// 只输出相关人员信息（如果有指定人员）
	sb.WriteString("\n## 人员信息\n\n")
	if len(relevantStaffIDs) > 0 {
		for _, staffID := range relevantStaffIDs {
			if name, ok := staffMap[staffID]; ok {
				sb.WriteString(fmt.Sprintf("- %s: %s（相关人员）\n", staffID, name))
			}
		}
		sb.WriteString("\n其他人员信息略...\n")
	} else {
		// 没有指定人员时，输出所有人员（但限制数量）
		count := 0
		for _, staff := range allStaffList {
			if staff != nil {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", staff.ID, staff.Name))
				count++
				if count >= 20 { // 最多输出20个人员
					sb.WriteString(fmt.Sprintf("（还有 %d 人未列出...）\n", len(allStaffList)-count))
					break
				}
			}
		}
	}

	// 班次信息（只输出相关班次）
	sb.WriteString("\n## 班次信息\n\n")
	for _, shift := range allShifts {
		if shift != nil {
			if len(relevantShiftIDs) > 0 && !s.containsString(relevantShiftIDs, shift.ID) {
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %s-%s\n", shift.Name, shift.ID, shift.StartTime, shift.EndTime))
		}
	}

	// 当前排班表（只输出与评审项相关的内容）
	sb.WriteString("\n## 当前排班表\n\n")

	if draft != nil && draft.Shifts != nil {
		for shiftID, shiftDraft := range draft.Shifts {
			if shiftDraft == nil || shiftDraft.Days == nil {
				continue
			}

			// 如果有指定班次，只输出相关班次
			if len(relevantShiftIDs) > 0 && !s.containsString(relevantShiftIDs, shiftID) {
				continue
			}

			// 获取班次名称
			shiftName := shiftID
			for _, shift := range allShifts {
				if shift != nil && shift.ID == shiftID {
					shiftName = shift.Name
					break
				}
			}
			sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", shiftName, shiftID))

			// 按日期排序输出
			dates := make([]string, 0, len(shiftDraft.Days))
			for date := range shiftDraft.Days {
				// 如果有指定日期，只输出相关日期
				if len(relevantDates) > 0 && !s.containsString(relevantDates, date) {
					continue
				}
				dates = append(dates, date)
			}
			sort.Strings(dates)

			// 如果没有相关日期，跳过此班次
			if len(dates) == 0 {
				continue
			}

			for _, date := range dates {
				dayShift := shiftDraft.Days[date]
				if dayShift == nil {
					continue
				}

				// 如果有指定人员，标记相关人员
				staffNames := make([]string, 0, len(dayShift.StaffIDs))
				for _, staffID := range dayShift.StaffIDs {
					name := staffID
					if n, ok := staffMap[staffID]; ok {
						name = n
					}
					// 标记相关人员
					if len(relevantStaffIDs) > 0 && s.containsString(relevantStaffIDs, staffID) {
						name = "【" + name + "】"
					}
					staffNames = append(staffNames, name)
				}
				sb.WriteString(fmt.Sprintf("- %s: %s (需要%d人，实际%d人)\n",
					date, strings.Join(staffNames, ", "), dayShift.RequiredCount, dayShift.ActualCount))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n请评审上述规则/需求是否在排班表中被正确满足，并以JSON格式输出结果。")

	return sb.String()
}

// getRelevantShiftIDs 获取与评审项相关的班次ID
func (s *globalReviewService) getRelevantShiftIDs(item *d_model.ReviewItem, allShifts []*d_model.Shift) []string {
	if item == nil {
		return nil
	}
	// 从评审项中获取相关班次
	if len(item.AffectedShiftIDs) > 0 {
		return item.AffectedShiftIDs
	}
	// 个人需求可能指定了目标班次
	if item.SourceNeed != nil && item.SourceNeed.TargetShiftID != "" {
		return []string{item.SourceNeed.TargetShiftID}
	}
	return nil // 返回nil表示所有班次都相关
}

// getRelevantDates 获取与评审项相关的日期
func (s *globalReviewService) getRelevantDates(item *d_model.ReviewItem, draft *d_model.ScheduleDraft) []string {
	if item == nil {
		return nil
	}
	// 从评审项中获取相关日期
	if len(item.AffectedDates) > 0 {
		return item.AffectedDates
	}
	// 个人需求可能指定了目标日期
	if item.SourceNeed != nil && len(item.SourceNeed.TargetDates) > 0 {
		return item.SourceNeed.TargetDates
	}
	return nil // 返回nil表示所有日期都相关
}

// getRelevantStaffIDs 获取与评审项相关的人员ID
func (s *globalReviewService) getRelevantStaffIDs(item *d_model.ReviewItem) []string {
	if item == nil {
		return nil
	}
	// 从评审项中获取相关人员
	if len(item.AffectedStaffIDs) > 0 {
		return item.AffectedStaffIDs
	}
	// 个人需求一定有目标人员
	if item.SourceNeed != nil && item.SourceNeed.StaffID != "" {
		return []string{item.SourceNeed.StaffID}
	}
	return nil
}

// containsString 检查字符串切片是否包含指定字符串
func (s *globalReviewService) containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// getItemTypeText 获取评审项类型的文本描述
func (s *globalReviewService) getItemTypeText(itemType d_model.ReviewItemType) string {
	switch itemType {
	case d_model.ReviewItemTypeRule:
		return "规则"
	case d_model.ReviewItemTypePersonalNeed:
		return "个人需求"
	default:
		return string(itemType)
	}
}

// parseReviewResponse 解析评审响应
func (s *globalReviewService) parseReviewResponse(content string, item *d_model.ReviewItem) (*d_model.ModificationOpinion, error) {
	// 提取JSON
	jsonStr := s.extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("无法从响应中提取JSON")
	}

	// 解析响应结构
	var resp struct {
		IsViolated           bool   `json:"isViolated"`
		ViolationDescription string `json:"violationDescription"`
		Suggestion           string `json:"suggestion"`
		Severity             string `json:"severity"`
		ProposedChanges      []struct {
			Date           string   `json:"date"`
			ShiftID        string   `json:"shiftId"`
			AddStaffIDs    []string `json:"addStaffIds"`
			RemoveStaffIDs []string `json:"removeStaffIds"`
			Reason         string   `json:"reason"`
		} `json:"proposedChanges"`
		AffectedStaffIDs []string `json:"affectedStaffIds"`
		AffectedDates    []string `json:"affectedDates"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 构建修改意见
	opinion := &d_model.ModificationOpinion{
		ID:                   fmt.Sprintf("opinion_%s", item.ID),
		ReviewItemID:         item.ID,
		ReviewItemType:       item.Type,
		ReviewItemName:       item.Name,
		IsViolated:           resp.IsViolated,
		ViolationDescription: resp.ViolationDescription,
		Suggestion:           resp.Suggestion,
		Severity:             resp.Severity,
		AffectedStaffIDs:     resp.AffectedStaffIDs,
		AffectedDates:        resp.AffectedDates,
		Status:               d_model.OpinionStatusPending,
	}

	if !resp.IsViolated {
		opinion.Status = d_model.OpinionStatusApproved
	}

	// 转换建议的修改
	if len(resp.ProposedChanges) > 0 {
		opinion.ProposedChanges = make(map[string]*d_model.ReviewScheduleChange)
		for _, change := range resp.ProposedChanges {
			opinion.ProposedChanges[change.Date] = &d_model.ReviewScheduleChange{
				ShiftID:        change.ShiftID,
				AddStaffIDs:    change.AddStaffIDs,
				RemoveStaffIDs: change.RemoveStaffIDs,
				Reason:         change.Reason,
			}
		}
	}

	return opinion, nil
}

// extractJSON 从响应中提取JSON字符串
func (s *globalReviewService) extractJSON(raw string) string {
	// 尝试找到JSON代码块
	if strings.Contains(raw, "```json") {
		start := strings.Index(raw, "```json") + 7
		end := strings.Index(raw[start:], "```")
		if end > 0 {
			return strings.TrimSpace(raw[start : start+end])
		}
	}
	if strings.Contains(raw, "```") {
		start := strings.Index(raw, "```") + 3
		// 跳过可能的语言标识
		if idx := strings.Index(raw[start:], "\n"); idx >= 0 {
			start += idx + 1
		}
		end := strings.Index(raw[start:], "```")
		if end > 0 {
			return strings.TrimSpace(raw[start : start+end])
		}
	}

	// 尝试直接找到JSON对象
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}

	return ""
}

// ReviewAllItems 批量评审所有评审项
func (s *globalReviewService) ReviewAllItems(
	ctx context.Context,
	items []*d_model.ReviewItem,
	draft *d_model.ScheduleDraft,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
	progressCallback d_model.GlobalReviewProgressCallback,
) ([]*d_model.ModificationOpinion, error) {
	opinions := make([]*d_model.ModificationOpinion, 0, len(items))
	violatedCount := 0

	for i, item := range items {
		// 发送进度回调
		if progressCallback != nil {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:            d_model.ReviewProgressItemReviewing,
				CurrentItem:     i + 1,
				TotalItems:      len(items),
				CurrentItemName: item.Name,
				CurrentItemType: item.Type,
				ViolatedCount:   violatedCount,
				Message:         fmt.Sprintf("正在评审: %s", item.Name),
			})
		}

		// 评审单项
		opinion, err := s.ReviewItemAgainstDraft(ctx, item, draft, allStaffList, allShifts)
		if err != nil {
			s.logger.Warn("评审项失败",
				"itemID", item.ID,
				"itemName", item.Name,
				"error", err,
			)
			// 创建失败意见，标记需人工评审
			opinion = &d_model.ModificationOpinion{
				ID:                   fmt.Sprintf("opinion_%s", item.ID),
				ReviewItemID:         item.ID,
				ReviewItemType:       item.Type,
				ReviewItemName:       item.Name,
				IsViolated:           true,
				ViolationDescription: fmt.Sprintf("评审失败: %v", err),
				Severity:             "warning",
				Status:               d_model.OpinionStatusManualReview,
			}
		}

		opinions = append(opinions, opinion)

		if opinion.IsViolated {
			violatedCount++
		}

		// 发送完成回调
		if progressCallback != nil {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:            d_model.ReviewProgressItemCompleted,
				CurrentItem:     i + 1,
				TotalItems:      len(items),
				CurrentItemName: item.Name,
				CurrentItemType: item.Type,
				ViolatedCount:   violatedCount,
				Message:         fmt.Sprintf("评审完成: %s (%s)", item.Name, s.getViolationStatus(opinion.IsViolated)),
			})
		}
	}

	return opinions, nil
}

// getViolationStatus 获取违规状态文本
func (s *globalReviewService) getViolationStatus(isViolated bool) string {
	if isViolated {
		return "违规"
	}
	return "通过"
}

// ============================================================
// 冲突检测
// ============================================================

// DetectConflicts 检测修改意见之间的冲突
func (s *globalReviewService) DetectConflicts(opinions []*d_model.ModificationOpinion) []*d_model.ConflictGroup {
	conflicts := make([]*d_model.ConflictGroup, 0)

	// 只检查违规的意见
	violatedOpinions := make([]*d_model.ModificationOpinion, 0)
	for _, op := range opinions {
		if op.IsViolated && op.Status == d_model.OpinionStatusPending {
			violatedOpinions = append(violatedOpinions, op)
		}
	}

	// 构建冲突检测索引：date+staffID -> []opinionID
	dateStaffIndex := make(map[string][]*d_model.ModificationOpinion)
	for _, op := range violatedOpinions {
		if op.ProposedChanges == nil {
			continue
		}
		for date, change := range op.ProposedChanges {
			// 添加操作
			for _, staffID := range change.AddStaffIDs {
				key := fmt.Sprintf("%s_%s_add", date, staffID)
				dateStaffIndex[key] = append(dateStaffIndex[key], op)
			}
			// 移除操作
			for _, staffID := range change.RemoveStaffIDs {
				key := fmt.Sprintf("%s_%s_remove", date, staffID)
				dateStaffIndex[key] = append(dateStaffIndex[key], op)
			}
		}
	}

	// 检测同一人同一天既要添加又要移除的冲突
	processedPairs := make(map[string]bool)
	for _, op := range violatedOpinions {
		if op.ProposedChanges == nil {
			continue
		}
		for date, change := range op.ProposedChanges {
			for _, staffID := range change.AddStaffIDs {
				removeKey := fmt.Sprintf("%s_%s_remove", date, staffID)
				if removeOps, ok := dateStaffIndex[removeKey]; ok {
					for _, removeOp := range removeOps {
						if removeOp.ID != op.ID {
							pairKey := fmt.Sprintf("%s_%s", op.ID, removeOp.ID)
							reversePairKey := fmt.Sprintf("%s_%s", removeOp.ID, op.ID)
							if !processedPairs[pairKey] && !processedPairs[reversePairKey] {
								processedPairs[pairKey] = true
								conflict := &d_model.ConflictGroup{
									ID:               fmt.Sprintf("conflict_%s", uuid.New().String()[:8]),
									OpinionIDs:       []string{op.ID, removeOp.ID},
									Opinions:         []*d_model.ModificationOpinion{op, removeOp},
									ConflictType:     "contradicting_rules",
									Description:      fmt.Sprintf("意见冲突: %s要求添加人员，%s要求移除同一人员", op.ReviewItemName, removeOp.ReviewItemName),
									AffectedDates:    []string{date},
									AffectedStaffIDs: []string{staffID},
								}
								conflicts = append(conflicts, conflict)

								// 标记冲突意见
								op.Status = d_model.OpinionStatusConflict
								op.ConflictingOpinionIDs = append(op.ConflictingOpinionIDs, removeOp.ID)
								removeOp.Status = d_model.OpinionStatusConflict
								removeOp.ConflictingOpinionIDs = append(removeOp.ConflictingOpinionIDs, op.ID)
							}
						}
					}
				}
			}
		}
	}

	return conflicts
}

// ============================================================
// 对论迭代
// ============================================================

// DebateOpinions 对修改意见进行对论迭代
func (s *globalReviewService) DebateOpinions(
	ctx context.Context,
	opinions []*d_model.ModificationOpinion,
	draft *d_model.ScheduleDraft,
	maxRounds int,
	progressCallback d_model.GlobalReviewProgressCallback,
) (*d_model.DebateResult, error) {
	if maxRounds <= 0 {
		maxRounds = 3
	}

	// 过滤出需要对论的意见（违规且未冲突）
	pendingOpinions := make([]*d_model.ModificationOpinion, 0)
	for _, op := range opinions {
		if op.IsViolated && op.Status == d_model.OpinionStatusPending {
			pendingOpinions = append(pendingOpinions, op)
		}
	}

	if len(pendingOpinions) == 0 {
		return &d_model.DebateResult{
			Converged:   true,
			TotalRounds: 0,
			Summary:     "无需对论的意见",
		}, nil
	}

	debateCtx := d_model.NewDebateContext(pendingOpinions, maxRounds)

	// 进行对论迭代
	for round := 1; round <= maxRounds; round++ {
		debateCtx.CurrentRound = round

		// 发送进度回调
		if progressCallback != nil {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:        d_model.ReviewProgressDebating,
				DebateRound: round,
				Message:     fmt.Sprintf("对论第%d轮（共%d轮）", round, maxRounds),
			})
		}

		// 执行一轮对论
		roundResult, err := s.executeDebateRound(ctx, debateCtx, draft)
		if err != nil {
			s.logger.Warn("对论轮次执行失败",
				"round", round,
				"error", err,
			)
			continue
		}

		debateCtx.DebateHistory = append(debateCtx.DebateHistory, roundResult)

		// 检查是否达成共识
		if s.checkConsensus(debateCtx) {
			debateCtx.Converged = true
			break
		}
	}

	// 构建对论结果
	result := s.buildDebateResult(debateCtx)
	return result, nil
}

// executeDebateRound 执行一轮对论
func (s *globalReviewService) executeDebateRound(
	ctx context.Context,
	debateCtx *d_model.DebateContext,
	draft *d_model.ScheduleDraft,
) (*d_model.DebateRound, error) {
	// 构建对论prompt
	systemPrompt := s.buildDebateSystemPrompt()
	userPrompt := s.buildDebateUserPrompt(debateCtx, draft)

	// 调用LLM
	modelConfig := s.getModelConfig("globalReview")
	resp, err := s.aiFactory.CallWithModel(ctx, modelConfig, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("调用LLM对论失败: %w", err)
	}

	// 解析响应
	round, err := s.parseDebateResponse(resp.Content, debateCtx.CurrentRound)
	if err != nil {
		return nil, fmt.Errorf("解析对论响应失败: %w", err)
	}

	// 根据裁定更新意见状态
	s.updateOpinionStatusFromDebate(debateCtx, round)

	return round, nil
}

// buildDebateSystemPrompt 构建对论系统提示词
func (s *globalReviewService) buildDebateSystemPrompt() string {
	return `你是一个排班修改方案评审专家。你需要评估每条修改意见的合理性和可行性。

评审标准：
1. **合理性**：修改是否真正解决了规则违规或需求未满足的问题？
2. **可行性**：修改是否可以执行？是否会导致其他问题？
3. **必要性**：是否有更简单的解决方案？修改是否过度？
4. **一致性**：修改是否与其他已批准的意见冲突？

对于每条修改意见，你需要：
- 分析其优缺点
- 判断是否应该通过

请以JSON格式输出评审结果：
{
  "analysis": "整体分析说明",
  "opinionVerdicts": [
    {
      "opinionId": "意见ID",
      "verdict": "approved/rejected/continue",
      "pros": "优点",
      "cons": "缺点",
      "reason": "裁定理由"
    }
  ],
  "overallVerdict": "approved/rejected/continue",
  "reason": "整体裁定理由"
}

verdict说明：
- approved: 通过，意见合理可执行
- rejected: 拒绝，意见不合理或无法执行  
- continue: 需要进一步讨论（仅在第一轮使用）`
}

// buildDebateUserPrompt 构建对论用户提示词
func (s *globalReviewService) buildDebateUserPrompt(debateCtx *d_model.DebateContext, draft *d_model.ScheduleDraft) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 对论第%d轮（共%d轮）\n\n", debateCtx.CurrentRound, debateCtx.MaxRounds))

	// 待对论的意见
	sb.WriteString("## 待评审的修改意见\n\n")
	for i, op := range debateCtx.Opinions {
		if op.Status != d_model.OpinionStatusPending {
			continue
		}
		sb.WriteString(fmt.Sprintf("### 意见%d: %s\n", i+1, op.ReviewItemName))
		sb.WriteString(fmt.Sprintf("- **意见ID**: %s\n", op.ID))
		sb.WriteString(fmt.Sprintf("- **违规描述**: %s\n", op.ViolationDescription))
		sb.WriteString(fmt.Sprintf("- **修改建议**: %s\n", op.Suggestion))
		sb.WriteString(fmt.Sprintf("- **严重程度**: %s\n", op.Severity))
		if len(op.ProposedChanges) > 0 {
			sb.WriteString("- **具体修改**:\n")
			for date, change := range op.ProposedChanges {
				sb.WriteString(fmt.Sprintf("  - %s: ", date))
				if len(change.AddStaffIDs) > 0 {
					sb.WriteString(fmt.Sprintf("添加%v ", change.AddStaffIDs))
				}
				if len(change.RemoveStaffIDs) > 0 {
					sb.WriteString(fmt.Sprintf("移除%v ", change.RemoveStaffIDs))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// 历史评审记录
	if len(debateCtx.DebateHistory) > 0 {
		sb.WriteString("## 之前的评审记录\n\n")
		for _, round := range debateCtx.DebateHistory {
			sb.WriteString(fmt.Sprintf("### 第%d轮评审\n", round.Round))
			sb.WriteString(fmt.Sprintf("- **分析**: %s\n", round.ReviewerChallenge))
			sb.WriteString(fmt.Sprintf("- **裁定**: %s\n", round.Verdict))
			sb.WriteString(fmt.Sprintf("- **理由**: %s\n\n", round.Reason))
		}
	}

	// 添加提示
	if debateCtx.CurrentRound == debateCtx.MaxRounds {
		sb.WriteString("\n**注意**：这是最后一轮评审，请对所有待定意见给出明确裁定（approved 或 rejected），不要再使用 continue。")
	}

	sb.WriteString("\n请对上述修改意见进行评审分析，并以JSON格式输出评审结果。")

	return sb.String()
}

// parseDebateResponse 解析对论响应
func (s *globalReviewService) parseDebateResponse(content string, round int) (*d_model.DebateRound, error) {
	jsonStr := s.extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("无法从响应中提取JSON")
	}

	var resp struct {
		Analysis        string `json:"analysis"`
		OpinionVerdicts []struct {
			OpinionID string `json:"opinionId"`
			Verdict   string `json:"verdict"`
			Pros      string `json:"pros"`
			Cons      string `json:"cons"`
			Reason    string `json:"reason"`
		} `json:"opinionVerdicts"`
		OverallVerdict string `json:"overallVerdict"`
		Reason         string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 转换意见裁定
	opinionVerdicts := make([]d_model.OpinionVerdict, 0, len(resp.OpinionVerdicts))
	for _, v := range resp.OpinionVerdicts {
		opinionVerdicts = append(opinionVerdicts, d_model.OpinionVerdict{
			OpinionID: v.OpinionID,
			Verdict:   v.Verdict,
			Reason:    v.Reason,
		})
	}

	return &d_model.DebateRound{
		Round:             round,
		ReviewerChallenge: resp.Analysis, // 使用 analysis 作为评审内容
		DefenderResponse:  "",            // 新格式不再需要辩护回应
		OpinionVerdicts:   opinionVerdicts,
		Verdict:           resp.OverallVerdict,
		Reason:            resp.Reason,
	}, nil
}

// updateOpinionStatusFromDebate 根据对论结果更新意见状态
func (s *globalReviewService) updateOpinionStatusFromDebate(debateCtx *d_model.DebateContext, round *d_model.DebateRound) {
	// 构建意见ID到意见的映射
	opinionMap := make(map[string]*d_model.ModificationOpinion)
	for _, op := range debateCtx.Opinions {
		opinionMap[op.ID] = op
	}

	// 首先处理单个意见的裁定
	if len(round.OpinionVerdicts) > 0 {
		for _, verdict := range round.OpinionVerdicts {
			if op, ok := opinionMap[verdict.OpinionID]; ok && op.Status == d_model.OpinionStatusPending {
				switch verdict.Verdict {
				case "approved":
					op.Status = d_model.OpinionStatusApproved
					op.ReviewComments = verdict.Reason
				case "rejected":
					op.Status = d_model.OpinionStatusRejected
					op.ReviewComments = verdict.Reason
				}
				// "continue" 保持 pending 状态
			}
		}
		return // 如果有单独裁定，不再使用整体裁定
	}

	// 回退到整体裁定（兼容旧格式）
	if round.Verdict == "approved" {
		for _, op := range debateCtx.Opinions {
			if op.Status == d_model.OpinionStatusPending {
				op.Status = d_model.OpinionStatusApproved
				op.ReviewComments = round.Reason
			}
		}
	} else if round.Verdict == "rejected" {
		for _, op := range debateCtx.Opinions {
			if op.Status == d_model.OpinionStatusPending {
				op.Status = d_model.OpinionStatusRejected
				op.ReviewComments = round.Reason
			}
		}
	}
	// continue状态保持pending，继续下一轮对论
}

// checkConsensus 检查是否达成共识
func (s *globalReviewService) checkConsensus(debateCtx *d_model.DebateContext) bool {
	// 所有意见都不再是pending状态时，达成共识
	for _, op := range debateCtx.Opinions {
		if op.Status == d_model.OpinionStatusPending {
			return false
		}
	}
	return true
}

// buildDebateResult 构建对论结果
func (s *globalReviewService) buildDebateResult(debateCtx *d_model.DebateContext) *d_model.DebateResult {
	result := &d_model.DebateResult{
		Converged:            debateCtx.Converged,
		TotalRounds:          debateCtx.CurrentRound,
		ApprovedOpinions:     make([]*d_model.ModificationOpinion, 0),
		RejectedOpinions:     make([]*d_model.ModificationOpinion, 0),
		ConflictOpinions:     make([]*d_model.ModificationOpinion, 0),
		ManualReviewOpinions: make([]*d_model.ModificationOpinion, 0),
	}

	for _, op := range debateCtx.Opinions {
		switch op.Status {
		case d_model.OpinionStatusApproved:
			result.ApprovedOpinions = append(result.ApprovedOpinions, op)
		case d_model.OpinionStatusRejected:
			result.RejectedOpinions = append(result.RejectedOpinions, op)
		case d_model.OpinionStatusConflict:
			result.ConflictOpinions = append(result.ConflictOpinions, op)
		case d_model.OpinionStatusPending, d_model.OpinionStatusManualReview:
			// 超过最大轮次仍未达成共识，标记为需人工评审
			op.Status = d_model.OpinionStatusManualReview
			result.ManualReviewOpinions = append(result.ManualReviewOpinions, op)
		}
	}

	// 生成总结
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("对论完成，共%d轮。", result.TotalRounds))
	if len(result.ApprovedOpinions) > 0 {
		sb.WriteString(fmt.Sprintf(" 通过%d条。", len(result.ApprovedOpinions)))
	}
	if len(result.RejectedOpinions) > 0 {
		sb.WriteString(fmt.Sprintf(" 拒绝%d条。", len(result.RejectedOpinions)))
	}
	if len(result.ConflictOpinions) > 0 {
		sb.WriteString(fmt.Sprintf(" 冲突%d条需人工处理。", len(result.ConflictOpinions)))
	}
	if len(result.ManualReviewOpinions) > 0 {
		sb.WriteString(fmt.Sprintf(" 未达共识%d条需人工评审。", len(result.ManualReviewOpinions)))
	}
	result.Summary = sb.String()

	return result
}

// ============================================================
// 批量修改
// ============================================================

// ApplyApprovedOpinions 应用通过的修改意见到草案
func (s *globalReviewService) ApplyApprovedOpinions(
	ctx context.Context,
	draft *d_model.ScheduleDraft,
	approvedOpinions []*d_model.ModificationOpinion,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) (*d_model.ScheduleDraft, error) {
	if len(approvedOpinions) == 0 {
		return draft, nil
	}

	// 构建修改prompt
	systemPrompt := s.buildModifySystemPrompt()
	userPrompt := s.buildModifyUserPrompt(draft, approvedOpinions, allStaffList, allShifts)

	// 调用LLM
	modelConfig := s.getModelConfig("globalReview")
	resp, err := s.aiFactory.CallWithModel(ctx, modelConfig, systemPrompt, userPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("调用LLM修改草案失败: %w", err)
	}

	// 解析响应
	modifiedDraft, err := s.parseModifyResponse(resp.Content, draft)
	if err != nil {
		s.logger.Warn("解析修改响应失败，返回原草案",
			"error", err,
		)
		return draft, nil
	}

	return modifiedDraft, nil
}

// buildModifySystemPrompt 构建修改系统提示词
func (s *globalReviewService) buildModifySystemPrompt() string {
	return `你是一个专业的排班调整专家。你的职责是根据评审通过的修改意见，对排班草案进行调整。

请以JSON格式输出需要修改的内容（只输出有变化的部分）：
{
  "changes": [
    {
      "shiftId": "班次ID",
      "date": "日期YYYY-MM-DD",
      "staffIds": ["人员ID1", "人员ID2", ...]
    }
  ],
  "summary": "修改说明"
}

注意：
- 严格按照修改意见执行调整
- 只输出有变化的日期，无变化的部分不要输出
- staffIds 是修改后该日期的完整人员列表`
}

// buildModifyUserPrompt 构建修改用户提示词
func (s *globalReviewService) buildModifyUserPrompt(
	draft *d_model.ScheduleDraft,
	approvedOpinions []*d_model.ModificationOpinion,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
) string {
	var sb strings.Builder

	// 收集相关的班次ID、日期和人员ID
	relevantShiftIDs := make(map[string]bool)
	relevantDates := make(map[string]bool)
	relevantStaffIDs := make(map[string]bool)

	for _, op := range approvedOpinions {
		for _, shiftID := range op.AffectedShiftIDs {
			relevantShiftIDs[shiftID] = true
		}
		for _, date := range op.AffectedDates {
			relevantDates[date] = true
		}
		for _, staffID := range op.AffectedStaffIDs {
			relevantStaffIDs[staffID] = true
		}
		for date, change := range op.ProposedChanges {
			relevantDates[date] = true
			if change != nil {
				relevantShiftIDs[change.ShiftID] = true
				for _, id := range change.AddStaffIDs {
					relevantStaffIDs[id] = true
				}
				for _, id := range change.RemoveStaffIDs {
					relevantStaffIDs[id] = true
				}
			}
		}
	}

	// 只输出相关人员信息
	sb.WriteString("## 人员信息\n\n")
	staffCount := 0
	for _, staff := range allStaffList {
		if staff != nil {
			if len(relevantStaffIDs) > 0 && !relevantStaffIDs[staff.ID] {
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s: %s\n", staff.ID, staff.Name))
			staffCount++
		}
	}
	if staffCount == 0 && len(allStaffList) > 0 {
		// 没有匹配时输出前20个
		for i, staff := range allStaffList {
			if staff != nil {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", staff.ID, staff.Name))
				if i >= 19 {
					sb.WriteString(fmt.Sprintf("- ... (共%d人，已省略部分)\n", len(allStaffList)))
					break
				}
			}
		}
	}

	// 只输出相关班次信息
	sb.WriteString("\n## 班次信息\n\n")
	for _, shift := range allShifts {
		if shift != nil {
			if len(relevantShiftIDs) > 0 && !relevantShiftIDs[shift.ID] {
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", shift.Name, shift.ID))
		}
	}

	// 只输出相关班次和日期的排班
	sb.WriteString("\n## 当前排班草案\n\n")
	if draft != nil && draft.Shifts != nil {
		for shiftID, shiftDraft := range draft.Shifts {
			if shiftDraft == nil || shiftDraft.Days == nil {
				continue
			}
			// 过滤相关班次
			if len(relevantShiftIDs) > 0 && !relevantShiftIDs[shiftID] {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n", shiftID))
			dates := make([]string, 0, len(shiftDraft.Days))
			for date := range shiftDraft.Days {
				// 过滤相关日期
				if len(relevantDates) > 0 && !relevantDates[date] {
					continue
				}
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

	// 需要执行的修改
	sb.WriteString("\n## 需要执行的修改\n\n")
	for i, op := range approvedOpinions {
		sb.WriteString(fmt.Sprintf("### 修改%d: %s\n", i+1, op.ReviewItemName))
		sb.WriteString(fmt.Sprintf("- 建议: %s\n", op.Suggestion))
		if len(op.ProposedChanges) > 0 {
			sb.WriteString("- 具体变更:\n")
			for date, change := range op.ProposedChanges {
				sb.WriteString(fmt.Sprintf("  - %s (%s): ", date, change.ShiftID))
				if len(change.AddStaffIDs) > 0 {
					sb.WriteString(fmt.Sprintf("添加%v ", change.AddStaffIDs))
				}
				if len(change.RemoveStaffIDs) > 0 {
					sb.WriteString(fmt.Sprintf("移除%v ", change.RemoveStaffIDs))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n请根据上述修改意见调整排班草案，输出完整的修改后排班。")

	return sb.String()
}

// parseModifyResponse 解析修改响应
func (s *globalReviewService) parseModifyResponse(content string, originalDraft *d_model.ScheduleDraft) (*d_model.ScheduleDraft, error) {
	jsonStr := s.extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("无法从响应中提取JSON")
	}

	// 尝试新格式（只包含变更）
	var newResp struct {
		Changes []struct {
			ShiftID  string   `json:"shiftId"`
			Date     string   `json:"date"`
			StaffIDs []string `json:"staffIds"`
		} `json:"changes"`
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &newResp); err == nil && len(newResp.Changes) > 0 {
		// 使用新格式：基于原始草案应用变更
		return s.applyChangesToDraft(originalDraft, newResp.Changes, newResp.Summary), nil
	}

	// 兼容旧格式（完整排班）
	var resp struct {
		Shifts  map[string]map[string][]string `json:"shifts"`
		Summary string                         `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 构建新的草案
	newDraft := &d_model.ScheduleDraft{
		StartDate:  originalDraft.StartDate,
		EndDate:    originalDraft.EndDate,
		Shifts:     make(map[string]*d_model.ShiftDraft),
		Summary:    resp.Summary,
		StaffStats: originalDraft.StaffStats,
		Conflicts:  originalDraft.Conflicts,
	}

	for shiftID, dates := range resp.Shifts {
		shiftDraft := &d_model.ShiftDraft{
			ShiftID: shiftID,
			Days:    make(map[string]*d_model.DayShift),
		}

		// 获取原始优先级
		if originalShift, ok := originalDraft.Shifts[shiftID]; ok {
			shiftDraft.Priority = originalShift.Priority
		}

		for date, staffIDs := range dates {
			dayShift := &d_model.DayShift{
				StaffIDs:    staffIDs,
				ActualCount: len(staffIDs),
			}
			// 获取原始需求人数
			if originalShift, ok := originalDraft.Shifts[shiftID]; ok {
				if originalDay, ok := originalShift.Days[date]; ok {
					dayShift.RequiredCount = originalDay.RequiredCount
					dayShift.IsFixed = originalDay.IsFixed
				}
			}
			shiftDraft.Days[date] = dayShift
		}

		newDraft.Shifts[shiftID] = shiftDraft
	}

	return newDraft, nil
}

// applyChangesToDraft 将变更应用到草案（新格式）
func (s *globalReviewService) applyChangesToDraft(
	originalDraft *d_model.ScheduleDraft,
	changes []struct {
		ShiftID  string   `json:"shiftId"`
		Date     string   `json:"date"`
		StaffIDs []string `json:"staffIds"`
	},
	summary string,
) *d_model.ScheduleDraft {
	// 深拷贝原始草案
	newDraft := &d_model.ScheduleDraft{
		StartDate:  originalDraft.StartDate,
		EndDate:    originalDraft.EndDate,
		Shifts:     make(map[string]*d_model.ShiftDraft),
		Summary:    summary,
		StaffStats: originalDraft.StaffStats,
		Conflicts:  originalDraft.Conflicts,
	}

	// 复制原始草案的所有班次
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
			// 复制日班次
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

	// 应用变更
	for _, change := range changes {
		shiftDraft, ok := newDraft.Shifts[change.ShiftID]
		if !ok {
			// 创建新的班次草案
			shiftDraft = &d_model.ShiftDraft{
				ShiftID: change.ShiftID,
				Days:    make(map[string]*d_model.DayShift),
			}
			newDraft.Shifts[change.ShiftID] = shiftDraft
		}

		// 更新或创建日班次
		dayShift, ok := shiftDraft.Days[change.Date]
		if !ok {
			dayShift = &d_model.DayShift{}
			shiftDraft.Days[change.Date] = dayShift
		}

		dayShift.StaffIDs = change.StaffIDs
		dayShift.ActualCount = len(change.StaffIDs)
	}

	return newDraft
}

// ============================================================
// 完整流程
// ============================================================

// ExecuteGlobalReview 执行完整的全局评审流程
func (s *globalReviewService) ExecuteGlobalReview(
	ctx context.Context,
	rules []*d_model.Rule,
	personalNeeds []*d_model.PersonalNeed,
	draft *d_model.ScheduleDraft,
	allStaffList []*d_model.Employee,
	allShifts []*d_model.Shift,
	maxDebateRounds int,
	progressCallback d_model.GlobalReviewProgressCallback,
) (*d_model.GlobalReviewResult, error) {
	startTime := time.Now()
	result := &d_model.GlobalReviewResult{}

	// 1. 合并规则和个人需求为评审项
	items := s.MergeToReviewItems(rules, personalNeeds)
	result.TotalItems = len(items)

	if len(items) == 0 {
		result.Summary = "无评审项"
		result.ExecutionTime = time.Since(startTime).Seconds()
		return result, nil
	}

	// 2. 逐条评审收集修改意见
	opinions, err := s.ReviewAllItems(ctx, items, draft, allStaffList, allShifts, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("评审失败: %w", err)
	}
	result.AllOpinions = opinions
	result.ReviewedItems = len(opinions)

	// 统计违规数
	for _, op := range opinions {
		if op.IsViolated {
			result.ViolatedItems++
		}
	}

	// 3. 检测冲突
	conflicts := s.DetectConflicts(opinions)
	for _, conflict := range conflicts {
		for _, op := range conflict.Opinions {
			result.ManualReviewItems = append(result.ManualReviewItems, op)
		}
	}

	// 4. 对论迭代（仅针对非冲突的违规意见）
	if progressCallback != nil {
		progressCallback(&d_model.GlobalReviewProgress{
			Type:    d_model.ReviewProgressDebating,
			Message: "开始对论迭代",
		})
	}

	debateResult, err := s.DebateOpinions(ctx, opinions, draft, maxDebateRounds, progressCallback)
	if err != nil {
		s.logger.Warn("对论迭代失败", "error", err)
	} else {
		result.DebateResult = debateResult
		// 添加需人工处理的项目
		result.ManualReviewItems = append(result.ManualReviewItems, debateResult.ConflictOpinions...)
		result.ManualReviewItems = append(result.ManualReviewItems, debateResult.ManualReviewOpinions...)
	}

	result.NeedsManualReview = len(result.ManualReviewItems) > 0

	// 5. 应用通过的修改意见
	if debateResult != nil && len(debateResult.ApprovedOpinions) > 0 {
		if progressCallback != nil {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:    d_model.ReviewProgressModifying,
				Message: fmt.Sprintf("应用%d条通过的修改意见", len(debateResult.ApprovedOpinions)),
			})
		}

		modifiedDraft, err := s.ApplyApprovedOpinions(ctx, draft, debateResult.ApprovedOpinions, allStaffList, allShifts)
		if err != nil {
			s.logger.Warn("应用修改失败", "error", err)
		} else {
			result.ModifiedDraft = modifiedDraft
		}
	} else {
		result.ModifiedDraft = draft
	}

	// 6. 生成总结
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("全局评审完成。共评审%d项，违规%d项。", result.TotalItems, result.ViolatedItems))
	if debateResult != nil {
		sb.WriteString(fmt.Sprintf(" 对论%d轮，通过%d项修改。",
			debateResult.TotalRounds, len(debateResult.ApprovedOpinions)))
	}
	if result.NeedsManualReview {
		sb.WriteString(fmt.Sprintf(" %d项需人工处理。", len(result.ManualReviewItems)))
	}
	result.Summary = sb.String()

	// 发送完成回调
	if progressCallback != nil {
		if result.NeedsManualReview {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:    d_model.ReviewProgressNeedsManual,
				Message: result.Summary,
			})
		} else {
			progressCallback(&d_model.GlobalReviewProgress{
				Type:    d_model.ReviewProgressCompleted,
				Message: result.Summary,
			})
		}
	}

	result.ExecutionTime = time.Since(startTime).Seconds()
	return result, nil
}

// ============================================================
// 辅助方法
// ============================================================

// getModelConfig 获取模型配置
func (s *globalReviewService) getModelConfig(taskType string) *common_config.AIModelProvider {
	// 尝试获取特定任务的模型名称
	cfg := s.configurator.GetConfig()
	if cfg.ScheduleV3.TaskModels != nil {
		if modelName, ok := cfg.ScheduleV3.TaskModels[taskType]; ok && modelName != "" {
			// 使用模型名称构建AIModelProvider
			// 从基础AI配置中获取provider
			baseAI := s.baseConfigurator.GetBaseConfig().AI
			if baseAI != nil {
				return &common_config.AIModelProvider{
					Provider: baseAI.ChatModel.Provider,
					Name:     modelName,
				}
			}
		}
	}
	// 回退到全局聊天模型
	baseAI := s.baseConfigurator.GetBaseConfig().AI
	if baseAI != nil {
		return &baseAI.ChatModel
	}
	return nil
}
