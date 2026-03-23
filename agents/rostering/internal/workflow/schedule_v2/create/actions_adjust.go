package create

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jusha/mcp/pkg/workflow/engine"

	d_model "jusha/agent/rostering/domain/model"

	"jusha/agent/rostering/internal/workflow/schedule_v2/adjust"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ============================================================
// 调整完成处理
// ============================================================

// actOnShiftAdjusted 处理班次调整完成事件
func actOnShiftAdjusted(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("CreateV2: Shift adjustment completed", "sessionID", sess.ID)

	createCtx, err := loadCreateV2Context(ctx, wctx)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// 从 payload 解析子工作流结果
	result, ok := payload.(*engine.SubWorkflowResult)
	if !ok {
		logger.Error("CreateV2: Invalid payload type for shift adjusted event",
			"payloadType", fmt.Sprintf("%T", payload))
		return fmt.Errorf("invalid payload type for shift adjusted event")
	}

	if !result.Success {
		logger.Error("CreateV2: Adjustment sub-workflow failed", "error", result.Error)
		errorMsg := "调整排班失败，请重试或提出其他调整需求。"
		if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, errorMsg); err != nil {
			logger.Warn("Failed to send error message", "error", err)
		}
		// 保持在等待调整状态，允许用户重试
		return nil
	}

	logger.Info("CreateV2: Parsing adjustment result",
		"outputIsNil", result.Output == nil,
		"outputKeys", func() int {
			if result.Output != nil {
				return len(result.Output)
			}
			return 0
		}())

	// 获取调整结果
	var resultDraft *d_model.ShiftScheduleDraft
	if result.Output != nil {
		if draftRaw, ok := result.Output["result_draft"]; ok {
			logger.Info("CreateV2: Found result_draft in output",
				"draftType", fmt.Sprintf("%T", draftRaw))
			if draft, ok := draftRaw.(*d_model.ShiftScheduleDraft); ok {
				resultDraft = draft
				logger.Info("CreateV2: Successfully extracted result_draft from output (direct type)")
			} else if draftMap, ok := draftRaw.(map[string]any); ok {
				// 尝试从 map 反序列化
				logger.Info("CreateV2: Attempting to deserialize result_draft from map")
				jsonBytes, err := json.Marshal(draftMap)
				if err == nil {
					var draft d_model.ShiftScheduleDraft
					if err := json.Unmarshal(jsonBytes, &draft); err == nil {
						resultDraft = &draft
						logger.Info("CreateV2: Successfully deserialized result_draft from map")
					} else {
						logger.Error("CreateV2: Failed to unmarshal result_draft", "error", err)
					}
				} else {
					logger.Error("CreateV2: Failed to marshal draftMap", "error", err)
				}
			} else {
				logger.Warn("CreateV2: result_draft has unexpected type", "type", fmt.Sprintf("%T", draftRaw))
			}
		} else {
			logger.Warn("CreateV2: result_draft not found in output", "availableKeys", func() []string {
				if result.Output != nil {
					keys := make([]string, 0, len(result.Output))
					for k := range result.Output {
						keys = append(keys, k)
					}
					return keys
				}
				return nil
			}())
		}
	}

	// 如果从 output 中获取失败，尝试从 session 获取
	if resultDraft == nil {
		logger.Info("CreateV2: Attempting to get result_draft from session")
		// 首先尝试从 shift_scheduling_context 获取
		shiftCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, "shift_scheduling_context")
		if err == nil && found {
			if shiftCtxMap, ok := shiftCtxData.(map[string]any); ok {
				if draftRaw, ok := shiftCtxMap["result_draft"]; ok {
					if draft, ok := draftRaw.(*d_model.ShiftScheduleDraft); ok {
						resultDraft = draft
						logger.Info("CreateV2: Found result_draft in shift_scheduling_context")
					}
				}
			}
		}
		// 如果还是找不到，尝试从 adjust context 获取
		if resultDraft == nil {
			adjustCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, adjust.DataKeyAdjustV2Context)
			if err == nil && found {
				if adjustCtx, ok := adjustCtxData.(*adjust.AdjustV2Context); ok {
					resultDraft = adjustCtx.ResultDraft
					if resultDraft != nil {
						logger.Info("CreateV2: Found result_draft in adjust context")
					}
				}
			}
		}
	}

	if resultDraft == nil {
		logger.Error("CreateV2: No result draft from adjustment sub-workflow",
			"outputIsNil", result.Output == nil,
			"outputKeys", func() []string {
				if result.Output != nil {
					keys := make([]string, 0, len(result.Output))
					for k := range result.Output {
						keys = append(keys, k)
					}
					return keys
				}
				return nil
			}())
		return fmt.Errorf("no result draft from adjustment sub-workflow")
	}

	logger.Info("CreateV2: Successfully retrieved result_draft",
		"scheduleCount", len(resultDraft.Schedule),
		"hasSchedule", resultDraft.Schedule != nil)

	// 提取临时需求并合并到 CreateV2Context.PersonalNeeds
	var temporaryNeeds []*d_model.PersonalNeed
	if result.Output != nil {
		if needsRaw, ok := result.Output["temporary_needs"]; ok {
			logger.Info("CreateV2: Found temporary_needs in adjustment result")
			if needs, ok := needsRaw.([]*d_model.PersonalNeed); ok {
				temporaryNeeds = needs
			} else if needsArray, ok := needsRaw.([]any); ok {
				// 尝试从 []any 转换
				temporaryNeeds = make([]*d_model.PersonalNeed, 0, len(needsArray))
				for _, item := range needsArray {
					if need, ok := item.(*d_model.PersonalNeed); ok {
						temporaryNeeds = append(temporaryNeeds, need)
					} else if needMap, ok := item.(map[string]any); ok {
						// 尝试从 map 反序列化
						jsonBytes, err := json.Marshal(needMap)
						if err == nil {
							var need d_model.PersonalNeed
							if err := json.Unmarshal(jsonBytes, &need); err == nil {
								temporaryNeeds = append(temporaryNeeds, &need)
							}
						}
					}
				}
			}
		}
	}

	// 合并临时需求到 CreateV2Context.PersonalNeeds
	if len(temporaryNeeds) > 0 {
		logger.Info("CreateV2: Merging temporary needs into PersonalNeeds",
			"temporaryNeedsCount", len(temporaryNeeds))
		if createCtx.PersonalNeeds == nil {
			createCtx.PersonalNeeds = make(map[string][]*PersonalNeed)
		}
		for _, need := range temporaryNeeds {
			if need != nil && (need.StaffID != "" || need.StaffName != "") {
				// 如果 StaffID 为空但 StaffName 不为空，尝试从 AllStaffList 中查找
				if need.StaffID == "" && need.StaffName != "" {
					for _, staff := range createCtx.AllStaffList {
						if staff != nil && staff.Name == need.StaffName {
							need.StaffID = staff.ID
							break
						}
					}
				}
				// 转换为 create 包中的 PersonalNeed 类型
				createNeed := &PersonalNeed{
					StaffID:         need.StaffID,
					StaffName:       need.StaffName,
					NeedType:        need.NeedType,
					RequestType:     need.RequestType,
					TargetShiftID:   need.TargetShiftID,
					TargetShiftName: need.TargetShiftName,
					TargetDates:     need.TargetDates,
					Description:     need.Description,
					Priority:        need.Priority,
					RuleID:          need.RuleID,
					Source:          need.Source,
					Confirmed:       need.Confirmed,
				}
				// 使用 StaffID 作为 key，如果 StaffID 为空则使用 StaffName
				key := need.StaffID
				if key == "" {
					key = need.StaffName
				}
				if createCtx.PersonalNeeds[key] == nil {
					createCtx.PersonalNeeds[key] = make([]*PersonalNeed, 0)
				}
				createCtx.PersonalNeeds[key] = append(createCtx.PersonalNeeds[key], createNeed)
			}
		}
		logger.Info("CreateV2: Temporary needs merged successfully",
			"totalStaffWithNeeds", len(createCtx.PersonalNeeds),
			"mergedCount", len(temporaryNeeds))
	}

	// 获取当前班次（用于后续检查和日志）
	if createCtx.CurrentShiftIndex >= len(createCtx.PhaseShiftList) {
		return fmt.Errorf("current shift index out of range")
	}
	currentShift := createCtx.PhaseShiftList[createCtx.CurrentShiftIndex]
	shiftID := currentShift.ID

	// 检查排班方案是否为空
	if len(resultDraft.Schedule) == 0 {
		logger.Error("CreateV2: Result draft schedule is empty!",
			"shiftID", shiftID,
			"shiftName", currentShift.Name,
			"scheduleCount", len(resultDraft.Schedule))
		return fmt.Errorf("result draft schedule is empty, adjustment may have failed")
	}

	// 获取调整总结和变化列表
	var adjustSummary string
	var adjustChanges []d_model.AdjustScheduleChange
	if result.Output != nil {
		if summaryRaw, ok := result.Output["adjust_summary"]; ok {
			if summaryStr, ok := summaryRaw.(string); ok {
				adjustSummary = summaryStr
			}
		}
		if changesRaw, ok := result.Output["adjust_changes"]; ok {
			if changes, ok := changesRaw.([]d_model.AdjustScheduleChange); ok {
				adjustChanges = changes
			} else if changesArr, ok := changesRaw.([]any); ok {
				// 尝试从 []any 转换
				adjustChanges = make([]d_model.AdjustScheduleChange, 0, len(changesArr))
				for _, item := range changesArr {
					if changeMap, ok := item.(map[string]any); ok {
						change := d_model.AdjustScheduleChange{}
						if date, ok := changeMap["date"].(string); ok {
							change.Date = date
						}
						if added, ok := changeMap["added"].([]any); ok {
							change.Added = make([]string, 0, len(added))
							for _, id := range added {
								if idStr, ok := id.(string); ok {
									change.Added = append(change.Added, idStr)
								}
							}
						}
						if removed, ok := changeMap["removed"].([]any); ok {
							change.Removed = make([]string, 0, len(removed))
							for _, id := range removed {
								if idStr, ok := id.(string); ok {
									change.Removed = append(change.Removed, idStr)
								}
							}
						}
						adjustChanges = append(adjustChanges, change)
					}
				}
			}
		}
	}

	// 如果从 output 中获取失败，尝试从 adjust context 获取
	if adjustSummary == "" || len(adjustChanges) == 0 {
		adjustCtxData, found, err := wctx.SessionService().GetData(ctx, sess.ID, adjust.DataKeyAdjustV2Context)
		if err == nil && found {
			if adjustCtx, ok := adjustCtxData.(*adjust.AdjustV2Context); ok {
				if adjustSummary == "" {
					adjustSummary = adjustCtx.AdjustSummary
				}
				if len(adjustChanges) == 0 {
					adjustChanges = adjustCtx.AdjustChanges
				}
			}
		}
	}

	// 获取当前班次（已在前面定义，这里不再重复定义）

	// 保存到对应阶段的 PhaseResult
	var phaseResult *PhaseResult
	switch createCtx.CurrentPhase {
	case PhaseSpecialShift:
		if createCtx.SpecialShiftResults == nil {
			createCtx.SpecialShiftResults = &PhaseResult{
				PhaseName:      PhaseSpecialShift,
				ShiftType:      ShiftTypeSpecial,
				ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
				StartTime:      time.Now().Format(time.RFC3339),
			}
		}
		phaseResult = createCtx.SpecialShiftResults
	case PhaseNormalShift:
		if createCtx.NormalShiftResults == nil {
			createCtx.NormalShiftResults = &PhaseResult{
				PhaseName:      PhaseNormalShift,
				ShiftType:      ShiftTypeNormal,
				ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
				StartTime:      time.Now().Format(time.RFC3339),
			}
		}
		phaseResult = createCtx.NormalShiftResults
	case PhaseResearchShift:
		if createCtx.ResearchShiftResults == nil {
			createCtx.ResearchShiftResults = &PhaseResult{
				PhaseName:      PhaseResearchShift,
				ShiftType:      ShiftTypeResearch,
				ScheduleDrafts: make(map[string]*d_model.ShiftScheduleDraft),
				StartTime:      time.Now().Format(time.RFC3339),
			}
		}
		phaseResult = createCtx.ResearchShiftResults
	default:
		return fmt.Errorf("unknown phase: %s", createCtx.CurrentPhase)
	}

	if phaseResult != nil {
		if phaseResult.ScheduleDrafts == nil {
			phaseResult.ScheduleDrafts = make(map[string]*d_model.ShiftScheduleDraft)
		}
		// 创建副本保存（避免引用问题）
		copyDraft := d_model.NewShiftScheduleDraft()
		if resultDraft.Schedule != nil {
			for date, staffIDs := range resultDraft.Schedule {
				if staffIDs != nil {
					copyDraft.Schedule[date] = append([]string{}, staffIDs...)
				}
			}
		}
		if resultDraft.UpdatedStaff != nil {
			for staffID := range resultDraft.UpdatedStaff {
				copyDraft.UpdatedStaff[staffID] = true
			}
		}
		phaseResult.ScheduleDrafts[shiftID] = copyDraft

		// 更新已占位信息（如果是重排，需要重新计算）
		MergeOccupiedSlots(createCtx.OccupiedSlots, copyDraft, shiftID)

		// 更新现有排班标记（用于时段冲突检查）
		allDrafts := make(map[string]*d_model.ShiftScheduleDraft)
		if createCtx.FixedShiftResults != nil && createCtx.FixedShiftResults.ScheduleDrafts != nil {
			for k, v := range createCtx.FixedShiftResults.ScheduleDrafts {
				allDrafts[k] = v
			}
		}
		if createCtx.SpecialShiftResults != nil && createCtx.SpecialShiftResults.ScheduleDrafts != nil {
			for k, v := range createCtx.SpecialShiftResults.ScheduleDrafts {
				allDrafts[k] = v
			}
		}
		if createCtx.NormalShiftResults != nil && createCtx.NormalShiftResults.ScheduleDrafts != nil {
			for k, v := range createCtx.NormalShiftResults.ScheduleDrafts {
				allDrafts[k] = v
			}
		}
		if createCtx.ResearchShiftResults != nil && createCtx.ResearchShiftResults.ScheduleDrafts != nil {
			for k, v := range createCtx.ResearchShiftResults.ScheduleDrafts {
				allDrafts[k] = v
			}
		}

		// 重新构建 ExistingScheduleMarks
		shiftMap := make(map[string]*d_model.Shift)
		for _, shift := range createCtx.SelectedShifts {
			shiftMap[shift.ID] = shift
		}
		createCtx.ExistingScheduleMarks = BuildExistingScheduleMarks(allDrafts, shiftMap)
	}

	// 保存上下文
	if err := saveCreateV2Context(ctx, wctx, createCtx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// 调整完成后，状态已经自动转换到 CreateV2StateShiftReview（通过状态转换定义）
	// 需要将调整后的结果保存到 shift_scheduling_context，以便 enterShiftReviewState 可以显示
	// 构建 ShiftSchedulingContext 以便 enterShiftReviewState 可以读取结果
	shiftCtx := d_model.NewShiftSchedulingContext(
		currentShift,
		createCtx.StartDate,
		createCtx.EndDate,
		string(WorkflowScheduleCreateV2),
	)
	shiftCtx.ResultDraft = resultDraft
	// 保存变化信息到 metadata，供详情对话框使用
	shiftCtxMetadata := map[string]any{
		"adjust_summary": adjustSummary,
		"adjust_changes": adjustChanges,
	}
	// 保存到 session，以便 enterShiftReviewState 可以读取
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "shift_scheduling_context", shiftCtx); err != nil {
		logger.Warn("Failed to save adjusted result to shift_scheduling_context", "error", err)
	}
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, "shift_scheduling_adjust_metadata", shiftCtxMetadata); err != nil {
		logger.Warn("Failed to save adjust metadata", "error", err)
	}

	// 构建成功消息，包含AI总结和变化列表
	var successMsg strings.Builder
	successMsg.WriteString(fmt.Sprintf("✅ 班次【%s】调整完成！\n\n", currentShift.Name))

	// 添加AI总结
	if adjustSummary != "" {
		successMsg.WriteString("**调整说明**：\n")
		successMsg.WriteString(adjustSummary)
		successMsg.WriteString("\n\n")
	}

	// 添加变化列表
	if len(adjustChanges) > 0 {
		// 构建员工ID到姓名的映射
		staffNameMap := make(map[string]string)
		for _, staff := range createCtx.AllStaffList {
			staffNameMap[staff.ID] = staff.Name
		}

		successMsg.WriteString("**主要变化**：\n")
		for _, change := range adjustChanges {
			changeParts := make([]string, 0)
			if len(change.Removed) > 0 {
				removedNames := make([]string, 0, len(change.Removed))
				for _, id := range change.Removed {
					if name, ok := staffNameMap[id]; ok {
						removedNames = append(removedNames, name)
					} else {
						removedNames = append(removedNames, id)
					}
				}
				changeParts = append(changeParts, fmt.Sprintf("移除%s", strings.Join(removedNames, "、")))
			}
			if len(change.Added) > 0 {
				addedNames := make([]string, 0, len(change.Added))
				for _, id := range change.Added {
					if name, ok := staffNameMap[id]; ok {
						addedNames = append(addedNames, name)
					} else {
						addedNames = append(addedNames, id)
					}
				}
				changeParts = append(changeParts, fmt.Sprintf("新增%s", strings.Join(addedNames, "、")))
			}
			if len(changeParts) > 0 {
				successMsg.WriteString(fmt.Sprintf("- %s：%s\n", change.Date, strings.Join(changeParts, "，")))
			}
		}
		successMsg.WriteString("\n")
	}

	if _, err := wctx.SessionService().AddAssistantMessage(ctx, sess.ID, successMsg.String()); err != nil {
		logger.Warn("Failed to send success message", "error", err)
	}

	// 调整完成后，状态已经自动转换到 CreateV2StateShiftReview（通过状态转换定义）
	// 这里直接调用 enterShiftReviewState 来显示审核界面
	// 注意：不需要再发送 CreateV2EventEnterShiftReview 事件，因为状态已经转换了
	logger.Info("CreateV2: Adjustment completed, entering review state for user confirmation")
	return enterShiftReviewState(ctx, wctx, createCtx)
}
