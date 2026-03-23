package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	d_model "jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"

	"jusha/agent/rostering/internal/workflow/schedule_v3/utils"
)

// executeProgressiveShiftScheduling 执行渐进式班次排班（按天执行）
// 【占位信息格式统一】已移除 map 格式的 occupiedSlots 参数，统一使用 taskContext.OccupiedSlots（强类型数组）
func (e *ProgressiveTaskExecutor) executeProgressiveShiftScheduling(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	targetShift *d_model.Shift,
	allShifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
	currentDraft *d_model.ShiftScheduleDraft,
	personalNeeds map[string][]*d_model.PersonalNeed,
	shiftIndex int,
	totalShifts int,
	retryContext *d_model.ShiftRetryContext,
	failedDatesIssues map[string][]string,
) (*d_model.ShiftScheduleDraft, string, error) {

	// 获取该班次的日期列表并排序
	targetDates := task.TargetDates
	if len(targetDates) == 0 {
		if reqs, ok := staffRequirements[targetShift.ID]; ok {
			for date := range reqs {
				targetDates = append(targetDates, date)
			}
		}
	}
	// 排序日期
	for i := 0; i < len(targetDates); i++ {
		for j := i + 1; j < len(targetDates); j++ {
			if targetDates[i] > targetDates[j] {
				targetDates[i], targetDates[j] = targetDates[j], targetDates[i]
			}
		}
	}

	// 收集所有日期的LLM3冲突人员信息（用于LLM4调整）
	allConflictStaff := make(map[string][]*RuleConflictStaffInfo) // date -> ConflictStaff列表

	if len(targetDates) == 0 {
		e.logger.Warn("No target dates for progressive scheduling",
			"shiftID", targetShift.ID)
		return currentDraft, "", nil
	}

	// 获取班次人数需求（总需求）
	shiftStaffRequirements := make(map[string]int)
	if reqs, ok := staffRequirements[targetShift.ID]; ok {
		shiftStaffRequirements = reqs
	}

	// 【修复】获取当前班次的固定排班（而不是所有班次的合并数据）
	// 这样在合并时只会考虑当前班次的固定排班人员，避免跨班次数据污染
	shiftFixedAssignments := make(map[string][]string)
	shiftFixedCounts := make(map[string]int)
	if e.taskContext != nil && len(e.taskContext.FixedAssignments) > 0 {
		for _, fa := range e.taskContext.FixedAssignments {
			if fa.ShiftID == targetShift.ID {
				shiftFixedAssignments[fa.Date] = fa.StaffIDs
				shiftFixedCounts[fa.Date] = len(fa.StaffIDs)
			}
		}
	}

	// 【BUG2修复】提前构建定向重试日期集合，用于后续existingDynamicCounts计算时排除
	retryTargetDateSet := make(map[string]bool)
	isTargetedRetry := retryContext != nil && retryContext.RetryOnlyTargetDates && len(retryContext.TargetRetryDates) > 0
	if isTargetedRetry {
		for _, date := range retryContext.TargetRetryDates {
			retryTargetDateSet[date] = true
		}
	}

	// 【关键修复】获取 WorkingDraft 中已有的动态排班人数
	// 这样可以避免任务重复执行时超配
	existingDynamicCounts := make(map[string]int)
	if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
		if shiftDraft, exists := e.taskContext.WorkingDraft.Shifts[targetShift.ID]; exists && shiftDraft != nil && shiftDraft.Days != nil {
			for date, dayShift := range shiftDraft.Days {
				// 【BUG2修复】定向重试的目标日期不计入已有动态排班
				// 原因：重试时这些日期需要重新排班，如果计入existingDynamicCount
				// 会导致dynamicRequired=0，使目标日期被错误skip
				if isTargetedRetry && retryTargetDateSet[date] {
					e.logger.Debug("Excluding targeted retry date from existingDynamicCounts",
						"shiftID", targetShift.ID,
						"date", date,
						"existingCount", len(dayShift.StaffIDs))
					continue
				}
				if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
					existingDynamicCounts[date] = len(dayShift.StaffIDs)
				}
			}
		}
	}

	// 计算动态需求（减去固定排班人数 + 已有的动态排班人数）
	dynamicStaffRequirements := make(map[string]int)
	for date, totalRequired := range shiftStaffRequirements {
		fixedCount := shiftFixedCounts[date]
		existingDynamicCount := existingDynamicCounts[date]
		// 动态需求 = 总需求 - 固定排班 - 已有动态排班
		dynamicRequired := totalRequired - fixedCount - existingDynamicCount
		if dynamicRequired < 0 {
			dynamicRequired = 0 // 如果已超配，动态需求为0
		}
		dynamicStaffRequirements[date] = dynamicRequired
		if fixedCount > 0 || existingDynamicCount > 0 {
			e.logger.Debug("Adjusted staff requirements for existing assignments (executeProgressiveShiftScheduling)",
				"shiftID", targetShift.ID,
				"date", date,
				"totalRequired", totalRequired,
				"fixedCount", fixedCount,
				"existingDynamicCount", existingDynamicCount,
				"dynamicRequired", dynamicRequired)
		}
	}

	// 检查是否有检查点（用于中断恢复）
	var completedDates []string
	var draft *d_model.ShiftScheduleDraft
	startIndex := 0

	if e.taskContext != nil {
		if checkpoint := e.taskContext.GetShiftCheckpoint(targetShift.ID); checkpoint != nil {
			e.logger.Info("Found checkpoint, resuming from last completed date",
				"shiftID", targetShift.ID,
				"lastCompletedDate", checkpoint.LastCompletedDate,
				"completedDates", len(checkpoint.CompletedDates))

			completedDates = checkpoint.CompletedDates
			draft = checkpoint.DraftSnapshot

			// 【占位信息格式统一】恢复占位信息（仅使用强类型数组）
			if checkpoint.OccupiedSlotsSnapshot != nil && e.taskContext != nil {
				// 构建 shiftID -> Shift 映射
				shiftMap := make(map[string]*d_model.Shift)
				for _, s := range allShifts {
					if s != nil {
						shiftMap[s.ID] = s
					}
				}

				// 从 map 格式的检查点恢复为强类型数组
				for staffID, slots := range checkpoint.OccupiedSlotsSnapshot {
					for date, shiftID := range slots {
						shift := shiftMap[shiftID]
						shiftName := ""
						if shift != nil {
							shiftName = shift.Name
						}
						e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
							e.taskContext.OccupiedSlots,
							d_model.StaffOccupiedSlot{
								StaffID:   staffID,
								Date:      date,
								ShiftID:   shiftID,
								ShiftName: shiftName,
								Source:    "checkpoint",
							},
						)
					}
				}
			}

			// 找到下一个要处理的日期索引
			for i, date := range targetDates {
				if date == checkpoint.LastCompletedDate {
					startIndex = i + 1
					break
				}
			}
		}
	}

	if draft == nil {
		draft = d_model.NewShiftScheduleDraft()
		// 复制当前草案
		if currentDraft != nil && currentDraft.Schedule != nil {
			for date, staffIDs := range currentDraft.Schedule {
				draft.Schedule[date] = make([]string, len(staffIDs))
				copy(draft.Schedule[date], staffIDs)
			}
		}
	}

	totalDays := len(targetDates)
	var combinedReasoning strings.Builder

	// 构建关联班次排班信息（用于连班规则）
	relatedShiftsSchedule := e.buildRelatedShiftsSchedule(targetShift, allShifts, rules)

	// 【已移至前面】retryTargetDateSet和isTargetedRetry在existingDynamicCounts计算前构建
	if isTargetedRetry {
		e.logger.Info("Targeted retry mode: only reschedule specific dates",
			"shiftID", targetShift.ID,
			"shiftName", targetShift.Name,
			"targetRetryDates", retryContext.TargetRetryDates)
	}

	e.logger.Info("Starting progressive shift scheduling",
		"shiftID", targetShift.ID,
		"shiftName", targetShift.Name,
		"totalDays", totalDays,
		"startIndex", startIndex,
		"completedDates", len(completedDates),
		"isTargetedRetry", isTargetedRetry)

	// 逐天执行
	for dayIndex := startIndex; dayIndex < totalDays; dayIndex++ {
		targetDate := targetDates[dayIndex]
		currentDay := dayIndex + 1

		// 【新增】定向重排模式：跳过不在目标日期列表中的日期
		if isTargetedRetry && !retryTargetDateSet[targetDate] {
			e.logger.Debug("Skipping date not in target retry list",
				"shiftID", targetShift.ID,
				"date", targetDate,
				"targetRetryDates", retryContext.TargetRetryDates)
			// 【P0修复】保留该日期原有的排班结果到新draft中
			// 之前的bug：只标记为已完成但没有复制数据到draft，导致数据丢失
			if currentDraft != nil && currentDraft.Schedule != nil {
				if staffIDs, exists := currentDraft.Schedule[targetDate]; exists && len(staffIDs) > 0 {
					draft.Schedule[targetDate] = make([]string, len(staffIDs))
					copy(draft.Schedule[targetDate], staffIDs)
				}
			}
			if !contains(completedDates, targetDate) {
				completedDates = append(completedDates, targetDate)
			}
			continue
		}

		// 【优化】检查当日动态需求是否为0，如果为0则跳过（固定排班已满足需求）
		dynamicRequired := dynamicStaffRequirements[targetDate]
		if dynamicRequired <= 0 {
			// 标记为已完成
			completedDates = append(completedDates, targetDate)
			continue
		}

		// 【新增】定向重排模式：在重新生成之前清空该日期原有的排班
		if isTargetedRetry && retryTargetDateSet[targetDate] {
			// 清空该日期的动态排班（保留固定排班）
			delete(draft.Schedule, targetDate)
			// 【占位信息格式统一】清空该日期在 taskContext.OccupiedSlots 中的记录（允许人员被重新安排）
			if e.taskContext != nil {
				// 移除该班次在该日期的所有占位记录
				result := make([]d_model.StaffOccupiedSlot, 0, len(e.taskContext.OccupiedSlots))
				for _, slot := range e.taskContext.OccupiedSlots {
					if !(slot.ShiftID == targetShift.ID && slot.Date == targetDate) {
						result = append(result, slot)
					}
				}
				e.taskContext.OccupiedSlots = result
			}
		}

		// 发送天级进度通知：开始生成
		draftPreviewJSON, _ := json.Marshal(draft)
		e.notifyProgress(&ShiftProgressInfo{
			ShiftID:        targetShift.ID,
			ShiftName:      targetShift.Name,
			Current:        shiftIndex,
			Total:          totalShifts,
			Status:         "day_generating",
			Message:        fmt.Sprintf("正在生成 [%s] 第%d/%d天 (%s) 排班...", targetShift.Name, currentDay, totalDays, targetDate),
			CurrentDay:     currentDay,
			TotalDays:      totalDays,
			CurrentDate:    targetDate,
			CompletedDates: completedDates,
			DraftPreview:   string(draftPreviewJSON),
		})

		// ============================================================
		// 【阶段1】规则预分析 - 策划阶段
		// 让LLM先分析当日需要关注的规则和个人需求
		// ============================================================
		var ruleAnalysis *DayRuleAnalysis
		ruleAnalysis, _ = e.analyzeDayRules(
			ctx,
			targetShift,
			targetDate,
			rules,
			personalNeeds,
			staffList,
			relatedShiftsSchedule,
			shiftFixedAssignments, // 传入当前班次的固定排班
			draft,                 // 【新增】传入当前班次已完成的动态排班，用于LLM3规则冲突检查
		)
		// 注意：即使分析失败也继续，使用全部规则

		// 【收集LLM3冲突人员信息】用于LLM4调整
		if ruleAnalysis != nil {
			for _, need := range ruleAnalysis.RelevantPersonalNeeds {
				if need.NeedType == "规则冲突" {
					// 从Description中提取规则名和原因
					parts := strings.SplitN(need.Description, ": ", 2)
					conflictRule := ""
					reason := need.Description
					if len(parts) == 2 {
						conflictRule = parts[0]
						reason = parts[1]
					}
					allConflictStaff[targetDate] = append(allConflictStaff[targetDate], &RuleConflictStaffInfo{
						StaffName:    need.StaffName,
						StaffID:      need.StaffID,
						ConflictRule: conflictRule,
						Reason:       reason,
					})
				}
			}
		}

		// ============================================================
		// 【阶段2】执行排班 - 执行阶段
		// 使用过滤后的规则进行实际排班
		// ============================================================

		// 【分批模式判断】当动态需求人数超过批次阈值时，启用分批排班
		batchConfig := DefaultBatchConfig()
		useBatchMode := ShouldUseBatchMode(dynamicRequired, batchConfig)

		if useBatchMode {
			// ============================================================
			// 分批排班路径：dynamicRequired > BatchSize（默认5人）
			// 将排班需求拆分为多个小批次，逐批调用LLM
			// ============================================================

			// 使用 taskContext.OccupiedSlots（强类型）进行分批排班
			batchOccupiedSlots := &e.taskContext.OccupiedSlots

			batchScheduledIDs, batchReasoning, batchErr := e.ExecuteBatchSchedulingForDay(
				ctx,
				targetShift,
				targetDate,
				dynamicRequired,
				staffList,
				rules,
				personalNeeds,
				shiftFixedAssignments,
				batchOccupiedSlots,
				batchConfig,
			)

			if batchErr != nil {
				e.logger.Warn("Batch scheduling failed, falling back to single mode",
					"shiftID", targetShift.ID,
					"date", targetDate,
					"error", batchErr)
				// 回退标记：不使用分批模式，继续走单次模式
				useBatchMode = false
			} else {
				// 分批排班成功：将结果写入 draft 和 occupiedSlots(map)

				// 写入draft
				draft.Schedule[targetDate] = batchScheduledIDs
				if draft.UpdatedStaff == nil {
					draft.UpdatedStaff = make(map[string]bool)
				}
				for _, staffID := range batchScheduledIDs {
					draft.UpdatedStaff[staffID] = true
				}

				// 【占位信息格式统一】更新占位信息（仅使用强类型数组）
				if e.taskContext != nil {
					for _, staffID := range batchScheduledIDs {
						realID := e.taskContext.ResolveStaffID(staffID)
						e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
							e.taskContext.OccupiedSlots,
							d_model.StaffOccupiedSlot{
								StaffID:   realID,
								Date:      targetDate,
								ShiftID:   targetShift.ID,
								ShiftName: targetShift.Name,
								Source:    "draft",
							},
						)
					}
				}

				// 记录推理
				if batchReasoning != "" {
					combinedReasoning.WriteString(fmt.Sprintf("[%s] %s\n", targetDate, batchReasoning))
				}
			}
		}

		// 单次模式：人数不多（<= BatchSize）或分批模式回退
		if !useBatchMode {
			// 构建单天Prompt（传入规则分析结果）
			sysPrompt := e.buildProgressiveDaySystemPrompt()
			userPrompt := e.buildProgressiveDayPromptWithAnalysis(
				targetShift,
				targetDate,
				targetDates,
				completedDates,
				rules,
				staffList,
				dynamicStaffRequirements, // 【关键修改】使用减去固定排班后的动态需求
				draft,
				personalNeeds,
				shiftFixedAssignments, // 【修复】使用当前班次的固定排班
				relatedShiftsSchedule,
				retryContext,      // 传入重试上下文
				failedDatesIssues, // 传入失败问题点
				ruleAnalysis,      // 【新增】传入规则分析结果
			)

			// 调用LLM
			llmCallStart := time.Now()
			resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
			llmCallDuration := time.Since(llmCallStart)

			// 记录到调试文件
			e.logLLMDebug(task.Title, logging.LLMCallProgressiveDay, targetShift.Name, targetDate, sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

			if err != nil {
				e.logger.Error("Progressive day scheduling LLM call failed", "shiftID", targetShift.ID, "date", targetDate, "error", err)
				return draft, combinedReasoning.String(), fmt.Errorf("day %s scheduling failed: %w", targetDate, err)
			}

			// 解析响应
			output, err := e.parseScheduleOutput(resp.Content, fmt.Sprintf("%s_%s", task.ID, targetDate))
			if err != nil {
				e.logger.Warn("Failed to parse day scheduling response", "shiftID", targetShift.ID, "date", targetDate, "error", err)
				// 继续下一天，不中断
				continue
			}

			// 合并到草案（使用当前班次的固定排班，而不是所有班次的合并数据）
			if err := e.mergeScheduleOutput(draft, output, shiftFixedAssignments); err != nil {
				e.logger.Warn("Failed to merge day schedule output",
					"shiftID", targetShift.ID,
					"date", targetDate,
					"error", err)
			}

			// 【占位信息格式统一】更新占位信息（仅使用强类型数组）
			if output.Schedule != nil && e.taskContext != nil {
				for date, staffIDs := range output.Schedule {
					// 转换shortID为UUID（统一通过 taskContext.ResolveStaffID）
					for _, staffID := range staffIDs {
						realID := e.taskContext.ResolveStaffID(staffID)
						e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
							e.taskContext.OccupiedSlots,
							d_model.StaffOccupiedSlot{
								StaffID:   realID,
								Date:      date,
								ShiftID:   targetShift.ID,
								ShiftName: targetShift.Name,
								Source:    "draft",
							},
						)
					}
				}
			}

			// 记录推理
			if output.Reasoning != "" {
				combinedReasoning.WriteString(fmt.Sprintf("[%s] %s\n", targetDate, output.Reasoning))
			}
		} // end of !useBatchMode

		// 更新已完成日期
		completedDates = append(completedDates, targetDate)

		// ============================================================
		// 【单日排班后LLM校验】
		// 在单日排班完成后立即调用LLM校验，校验到当前日期为止的所有已排班结果
		// ============================================================
		var validationResult *RuleMatchingResult
		if e.aiFactory != nil && len(rules) > 0 {
			// 获取WorkingDraft用于校验
			var workingDraft *d_model.ScheduleDraft
			if e.taskContext != nil {
				workingDraft = e.taskContext.WorkingDraft
			}

			validationResult = e.validateDayScheduleWithLLM(
				ctx,
				targetShift,
				targetDate,
				completedDates,
				draft,
				allShifts,
				rules,
				staffList,
				workingDraft,
				shiftStaffRequirements, // 传入人数需求，用于检查人数是否满足
			)

			if validationResult != nil {
				// 【前端显示】无论成功还是失败，都发送校验结果到前端
				var validationMsg strings.Builder
				if validationResult.Passed {
					validationMsg.WriteString(fmt.Sprintf("✅ [%s] %s 排班校验通过", targetShift.Name, targetDate))
					if validationResult.MatchScore < 1.0 {
						validationMsg.WriteString(fmt.Sprintf(" (匹配度: %.2f)", validationResult.MatchScore))
					}
				} else {
					// 过滤出critical级别的问题
					criticalIssues := make([]*RuleMatchingIssue, 0)
					for _, issue := range validationResult.Issues {
						if issue.Severity == "critical" {
							criticalIssues = append(criticalIssues, issue)
						}
					}

					// 只有critical问题时才显示
					if len(criticalIssues) > 0 {
						validationMsg.WriteString(fmt.Sprintf("❌ [%s] %s 发现错误", targetShift.Name, targetDate))
						validationMsg.WriteString("\n错误：")
						for i, issue := range criticalIssues {
							if i >= 3 {
								validationMsg.WriteString(fmt.Sprintf("\n... 还有%d个错误", len(criticalIssues)-3))
								break
							}
							validationMsg.WriteString(fmt.Sprintf("\n%d. ❌ %s", i+1, issue.Description))
						}
					} else {
						// 没有critical问题，认为通过
						validationMsg.WriteString(fmt.Sprintf("✅ [%s] %s 排班校验通过", targetShift.Name, targetDate))
					}
				}

				// 发送校验结果到前端
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:        targetShift.ID,
					ShiftName:      targetShift.Name,
					Current:        shiftIndex,
					Total:          totalShifts,
					Status:         "day_validated",
					Message:        validationMsg.String(),
					CurrentDay:     currentDay,
					TotalDays:      totalDays,
					CurrentDate:    targetDate,
					CompletedDates: completedDates,
					Reasoning:      validationResult.LLMAnalysis,
				})
			}

			// 如果校验失败且需要重试，立即纠正
			if validationResult != nil && !validationResult.Passed && validationResult.NeedsRetry {
				e.logger.Warn("Day schedule validation failed, attempting correction",
					"shiftID", targetShift.ID,
					"date", targetDate,
					"issueCount", len(validationResult.Issues))

				// 【新增】向前端发送重试通知消息（通过 progress callback）
				var retryMsg strings.Builder
				retryMsg.WriteString(fmt.Sprintf("🔄 **[%s] %s 正在自动重试**\n\n", targetShift.Name, targetDate))
				retryMsg.WriteString("发现以下问题，系统正在尝试纠正：\n")
				for i, issue := range validationResult.Issues {
					if issue.Severity == "critical" {
						retryMsg.WriteString(fmt.Sprintf("  %d. ❌ %s\n", i+1, issue.Description))
					}
				}

				// 发送进度通知：校验失败，正在纠正（Message 包含详细重试信息）
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:        targetShift.ID,
					ShiftName:      targetShift.Name,
					Current:        shiftIndex,
					Total:          totalShifts,
					Status:         "day_retrying",
					Message:        retryMsg.String(),
					CurrentDay:     currentDay,
					TotalDays:      totalDays,
					CurrentDate:    targetDate,
					CompletedDates: completedDates,
				})

				// 构建纠正建议
				var correctionSuggestions strings.Builder
				correctionSuggestions.WriteString("排班校验发现问题，请根据以下问题纠正：\n\n")
				for i, issue := range validationResult.Issues {
					if issue.Severity == "critical" {
						correctionSuggestions.WriteString(fmt.Sprintf("%d. 【严重】%s\n", i+1, issue.Description))
					}
				}
				if validationResult.LLMAnalysis != "" {
					correctionSuggestions.WriteString(fmt.Sprintf("\n整体分析：%s\n", validationResult.LLMAnalysis))
				}

				// 清空当日排班，准备重新生成
				if _, exists := draft.Schedule[targetDate]; exists {
					// 【占位信息格式统一】清理占位信息（仅使用强类型数组）
					if e.taskContext != nil {
						result := make([]d_model.StaffOccupiedSlot, 0, len(e.taskContext.OccupiedSlots))
						for _, slot := range e.taskContext.OccupiedSlots {
							if !(slot.ShiftID == targetShift.ID && slot.Date == targetDate) {
								result = append(result, slot)
							}
						}
						e.taskContext.OccupiedSlots = result
					}
					// 删除当日排班
					delete(draft.Schedule, targetDate)
				}

				// 从已完成日期中移除，准备重新排班
				completedDates = completedDates[:len(completedDates)-1]

				// 构建包含纠正建议的失败问题点
				correctionIssues := make(map[string][]string)
				correctionIssues[targetDate] = []string{correctionSuggestions.String()}

				// 重新调用单日排班（传入纠正建议）
				// 重新构建Prompt（包含纠正建议）
				sysPrompt := e.buildProgressiveDaySystemPrompt()
				userPrompt := e.buildProgressiveDayPromptWithAnalysis(
					targetShift,
					targetDate,
					targetDates,
					completedDates,
					rules,
					staffList,
					dynamicStaffRequirements,
					draft,
					personalNeeds,
					shiftFixedAssignments,
					relatedShiftsSchedule,
					retryContext,
					correctionIssues, // 传入纠正建议
					ruleAnalysis,
				)

				// 调用LLM重新生成
				llmCallStart := time.Now()
				resp, err := e.aiFactory.CallWithRetryLevel(ctx, 0, sysPrompt, userPrompt, nil)
				llmCallDuration := time.Since(llmCallStart)

				// 记录LLM调试日志
				e.logLLMDebug(task.Title+"_correction", logging.LLMCallDayCorrection, targetShift.Name, targetDate, sysPrompt, userPrompt, resp.Content, llmCallDuration, err)

				if err != nil {
					e.logger.Error("Day schedule correction LLM call failed",
						"shiftID", targetShift.ID,
						"date", targetDate,
						"duration", llmCallDuration.Seconds(),
						"error", err)
					// 纠正失败，记录警告但继续下一天
					e.logger.Warn("Day schedule correction failed, continuing to next day",
						"shiftID", targetShift.ID,
						"date", targetDate)
					// 恢复已完成日期
					completedDates = append(completedDates, targetDate)
					// 继续下一天
					continue
				}

				// 解析纠正后的响应
				correctedOutput, err := e.parseScheduleOutput(resp.Content, fmt.Sprintf("%s_%s_corrected", task.ID, targetDate))
				if err != nil {
					e.logger.Warn("Failed to parse corrected day scheduling response",
						"shiftID", targetShift.ID,
						"date", targetDate,
						"error", err)
					// 恢复已完成日期
					completedDates = append(completedDates, targetDate)
					// 继续下一天
					continue
				}

				// 合并纠正后的结果
				if err := e.mergeScheduleOutput(draft, correctedOutput, shiftFixedAssignments); err != nil {
					e.logger.Warn("Failed to merge corrected day schedule output",
						"shiftID", targetShift.ID,
						"date", targetDate,
						"error", err)
				}

				// 【占位信息格式统一】更新占位信息（仅使用强类型数组）
				if correctedOutput.Schedule != nil && e.taskContext != nil {
					for date, staffIDs := range correctedOutput.Schedule {
						for _, staffID := range staffIDs {
							realID := e.taskContext.ResolveStaffID(staffID)
							e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
								e.taskContext.OccupiedSlots,
								d_model.StaffOccupiedSlot{
									StaffID:   realID,
									Date:      date,
									ShiftID:   targetShift.ID,
									ShiftName: targetShift.Name,
									Source:    "draft",
								},
							)
						}
					}
				}

				// 记录纠正推理
				if correctedOutput.Reasoning != "" {
					combinedReasoning.WriteString(fmt.Sprintf("[%s 纠正] %s\n", targetDate, correctedOutput.Reasoning))
				}

				// 重新添加到已完成日期
				completedDates = append(completedDates, targetDate)

				// 【新增】向前端发送重试成功通知（通过 progress callback）
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:        targetShift.ID,
					ShiftName:      targetShift.Name,
					Current:        shiftIndex,
					Total:          totalShifts,
					Status:         "day_retry_success",
					Message:        fmt.Sprintf("✅ **[%s] %s 重试成功**\n\n排班已自动纠正完成。", targetShift.Name, targetDate),
					CurrentDay:     currentDay,
					TotalDays:      totalDays,
					CurrentDate:    targetDate,
					CompletedDates: completedDates,
				})
			} else if validationResult != nil && !validationResult.Passed {
				// 校验失败但不需要重试（warning级别），记录日志但继续
				e.logger.Warn("Day schedule validation failed but no retry needed",
					"shiftID", targetShift.ID,
					"date", targetDate,
					"issueCount", len(validationResult.Issues))

				// 【新增】向前端发送跳过通知（warning级别问题，通过 progress callback）
				// var skipMsg strings.Builder
				// skipMsg.WriteString(fmt.Sprintf("⚠️ **[%s] %s 发现问题已跳过**\n\n", targetShift.Name, targetDate))
				// skipMsg.WriteString("以下问题将在后续处理中解决：\n")
				// for i, issue := range validationResult.Issues {
				// 	if i >= 3 {
				// 		skipMsg.WriteString(fmt.Sprintf("  ... 还有 %d 个问题\n", len(validationResult.Issues)-3))
				// 		break
				// 	}
				// 	skipMsg.WriteString(fmt.Sprintf("  %d. ⚠️ %s\n", i+1, issue.Description))
				// }
				// e.notifyProgress(&ShiftProgressInfo{
				// 	ShiftID:        targetShift.ID,
				// 	ShiftName:      targetShift.Name,
				// 	Current:        shiftIndex,
				// 	Total:          totalShifts,
				// 	Status:         "day_skipped",
				// 	Message:        skipMsg.String(),
				// 	CurrentDay:     currentDay,
				// 	TotalDays:      totalDays,
				// 	CurrentDate:    targetDate,
				// 	CompletedDates: completedDates,
				// })
			}
		}

		// 保存检查点
		if e.taskContext != nil {
			// 【占位信息格式统一】将强类型数组转换为 map 格式保存到检查点（兼容检查点格式）
			occupiedSlotsMap := d_model.ConvertOccupiedSlotsToMap(e.taskContext.OccupiedSlots)
			checkpoint := &d_model.ShiftExecutionCheckpoint{
				ShiftID:               targetShift.ID,
				ShiftName:             targetShift.Name,
				LastCompletedDate:     targetDate,
				CompletedDates:        completedDates,
				AllDates:              targetDates,
				DraftSnapshot:         draft,
				OccupiedSlotsSnapshot: occupiedSlotsMap,
				UpdatedAt:             time.Now().Format(time.RFC3339),
			}
			e.taskContext.SaveShiftCheckpoint(checkpoint)
			e.logger.Debug("Saved checkpoint",
				"shiftID", targetShift.ID,
				"date", targetDate)
		}

		// 发送天级进度通知：完成
		draftPreviewJSON, _ = json.Marshal(draft)
		statusMsg := fmt.Sprintf("✅ [%s] 第%d/%d天 (%s) 排班完成", targetShift.Name, currentDay, totalDays, targetDate)
		if validationResult != nil && !validationResult.Passed && validationResult.NeedsRetry {
			statusMsg = fmt.Sprintf("✅ [%s] 第%d/%d天 (%s) 排班完成（已纠正）", targetShift.Name, currentDay, totalDays, targetDate)
		}
		e.notifyProgress(&ShiftProgressInfo{
			ShiftID:        targetShift.ID,
			ShiftName:      targetShift.Name,
			Current:        shiftIndex,
			Total:          totalShifts,
			Status:         "day_completed",
			Message:        statusMsg,
			CurrentDay:     currentDay,
			TotalDays:      totalDays,
			CurrentDate:    targetDate,
			CompletedDates: completedDates,
			DraftPreview:   string(draftPreviewJSON),
		})

		e.logger.Info("Completed progressive day scheduling",
			"shiftID", targetShift.ID,
			"date", targetDate,
			"dayIndex", dayIndex+1,
			"totalDays", totalDays,
			"scheduledCount", len(draft.Schedule[targetDate]),
			"validationPassed", validationResult == nil || validationResult.Passed)
	}

	// 清理检查点（所有天完成）
	if e.taskContext != nil {
		e.taskContext.ClearShiftCheckpoint(targetShift.ID)
	}

	// ============================================================
	// 【LLM4: 班次排班调整执行】
	// 在整个班次排班完成后调用，结合LLM3输出的冲突人员，对排班进行调整
	// 限制：每个班次最多调用1次LLM4，避免循环调整
	// ============================================================
	maxLLM4Adjustments := 1 // LLM4调整次数上限
	llm4AdjustmentCount := 0
	if retryContext != nil {
		llm4AdjustmentCount = retryContext.LLM4AdjustmentCount
	}

	if e.aiFactory != nil && len(allConflictStaff) > 0 && llm4AdjustmentCount < maxLLM4Adjustments {
		// 获取该班次的人数需求（总需求）
		shiftStaffRequirements := make(map[string]int)
		if reqs, ok := staffRequirements[targetShift.ID]; ok {
			shiftStaffRequirements = reqs
		}

		// 发送"正在调整冲突"通知
		e.notifyProgress(&ShiftProgressInfo{
			ShiftID:   targetShift.ID,
			ShiftName: targetShift.Name,
			Current:   shiftIndex,
			Total:     totalShifts,
			Status:    "llm4_adjusting",
			Message:   fmt.Sprintf("⏳ [%s] 正在调整冲突...", targetShift.Name),
		})

		// 记录LLM4调整次数
		llm4AdjustmentCount++
		if retryContext != nil {
			retryContext.LLM4AdjustmentCount = llm4AdjustmentCount
		}

		adjustmentResult := e.adjustScheduleWithLLM4(
			ctx,
			targetShift,
			targetDates,
			draft,
			staffList,
			shiftStaffRequirements,
			allConflictStaff,      // 传入LLM3的冲突人员信息
			rules,                 // 传入排班规则（供LLM4决策用）
			relatedShiftsSchedule, // 传入相关班次排班（供LLM4避免新冲突）
		)

		if adjustmentResult != nil {
			// 如果LLM4输出了调整后的排班，更新draft
			if len(adjustmentResult.AdjustedSchedule) > 0 {
				// 【占位信息格式统一】清理旧的占位信息（仅使用强类型数组）
				if e.taskContext != nil {
					result := make([]d_model.StaffOccupiedSlot, 0, len(e.taskContext.OccupiedSlots))
					for _, slot := range e.taskContext.OccupiedSlots {
						// 保留不是该班次在目标日期的占位记录
						shouldRemove := false
						for _, date := range targetDates {
							if slot.ShiftID == targetShift.ID && slot.Date == date {
								shouldRemove = true
								break
							}
						}
						if !shouldRemove {
							result = append(result, slot)
						}
					}
					e.taskContext.OccupiedSlots = result
				}

				// 更新draft
				draft.Schedule = adjustmentResult.AdjustedSchedule

				// 【占位信息格式统一】更新占位信息（仅使用强类型数组）
				if e.taskContext != nil {
					for date, staffIDs := range adjustmentResult.AdjustedSchedule {
						for _, staffID := range staffIDs {
							e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
								e.taskContext.OccupiedSlots,
								d_model.StaffOccupiedSlot{
									StaffID:   staffID,
									Date:      date,
									ShiftID:   targetShift.ID,
									ShiftName: targetShift.Name,
									Source:    "draft",
								},
							)
						}
					}
				}

				// 【更新WorkingDraft】更新WorkingDraft中的排班信息
				if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
					if e.taskContext.WorkingDraft.Shifts == nil {
						e.taskContext.WorkingDraft.Shifts = make(map[string]*d_model.ShiftDraft)
					}
					if e.taskContext.WorkingDraft.Shifts[targetShift.ID] == nil {
						e.taskContext.WorkingDraft.Shifts[targetShift.ID] = &d_model.ShiftDraft{
							ShiftID: targetShift.ID,
							Days:    make(map[string]*d_model.DayShift),
						}
					}
					shiftDraftEntry := e.taskContext.WorkingDraft.Shifts[targetShift.ID]
					if shiftDraftEntry.Days == nil {
						shiftDraftEntry.Days = make(map[string]*d_model.DayShift)
					}

					// 构建人员ID到姓名的映射
					staffNamesMap := BuildStaffNamesMap(staffList)

					for date, staffIDs := range adjustmentResult.AdjustedSchedule {
						// 转换为姓名列表
						staffNames := MapIDsToNames(staffIDs, staffNamesMap)

						// 更新 WorkingDraft
						shiftDraftEntry.Days[date] = &d_model.DayShift{
							Staff:         staffNames,
							StaffIDs:      staffIDs,
							ActualCount:   len(staffIDs),
							RequiredCount: len(staffIDs),
							IsFixed:       false,
						}
					}
				}

				// 【LLM4调整后校验】对调整后的排班进行校验
				validationResult := e.validateSingleShift(
					ctx,
					targetShift,
					draft,
					rules,
					shiftStaffRequirements,
					staffList,
					allShifts,
				)

				if validationResult != nil && !validationResult.Passed {
					e.logger.Warn("LLM4 adjustment validation failed",
						"shiftID", targetShift.ID,
						"summary", validationResult.Summary)
				}

				// 【LLM4系统消息】发送简短修复通知
				replacedCount := len(adjustmentResult.ReplacedStaff)
				draftPreviewForLLM4, _ := json.Marshal(draft)
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:      targetShift.ID,
					ShiftName:    targetShift.Name,
					Current:      shiftIndex,
					Total:        totalShifts,
					Status:       "llm4_fixed",
					Message:      fmt.Sprintf("✅ [%s] 已修复 %d 天冲突", targetShift.Name, replacedCount),
					DraftPreview: string(draftPreviewForLLM4),
				})
			}
		}
	}

	e.logger.Info("Completed progressive shift scheduling",
		"shiftID", targetShift.ID,
		"shiftName", targetShift.Name,
		"totalDays", totalDays)

	return draft, combinedReasoning.String(), nil
}

// buildRelatedShiftsSchedule 构建关联班次的排班信息（用于连班规则）
func (e *ProgressiveTaskExecutor) buildRelatedShiftsSchedule(
	targetShift *d_model.Shift,
	allShifts []*d_model.Shift,
	rules []*d_model.Rule,
) map[string]map[string][]string {
	result := make(map[string]map[string][]string) // shiftID -> date -> staffNames

	if e.taskContext == nil || e.taskContext.WorkingDraft == nil {
		return result
	}

	// 找出规则关联的班次
	relatedShiftIDs := make(map[string]bool)
	for _, rule := range rules {
		if rule == nil || len(rule.Associations) == 0 {
			continue
		}
		ruleShiftIDs := make([]string, 0)
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == "shift" {
				ruleShiftIDs = append(ruleShiftIDs, assoc.AssociationID)
			}
		}
		if contains(ruleShiftIDs, targetShift.ID) {
			for _, shiftID := range ruleShiftIDs {
				if shiftID != targetShift.ID {
					relatedShiftIDs[shiftID] = true
				}
			}
		}
	}

	// 构建姓名映射
	staffIDToName := make(map[string]string)
	if e.taskContext.AllStaff != nil {
		staffIDToName = utils.BuildStaffIDToNameMapping(e.taskContext.AllStaff)
	}

	// 获取关联班次的排班
	workingDraft := e.taskContext.WorkingDraft
	for shiftID := range relatedShiftIDs {
		if shiftDraft, exists := workingDraft.Shifts[shiftID]; exists && shiftDraft != nil {
			result[shiftID] = make(map[string][]string)
			for date, dayShift := range shiftDraft.Days {
				if dayShift != nil && len(dayShift.Staff) > 0 {
					result[shiftID][date] = dayShift.Staff
				} else if dayShift != nil && len(dayShift.StaffIDs) > 0 {
					// 转换ID为姓名
					names := make([]string, 0, len(dayShift.StaffIDs))
					for _, staffID := range dayShift.StaffIDs {
						name := staffIDToName[staffID]
						if name == "" {
							name = e.taskContext.GetStaffName(staffID) // 禁止UUID泄露
						}
						names = append(names, name)
					}
					result[shiftID][date] = names
				}
			}
		}
	}

	return result
}

// 【已废弃】syncOccupiedSlot 函数已移除
// 占位信息格式已统一为强类型数组，直接使用 d_model.AddOccupiedSlotIfNotExists
