package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	d_model "jusha/agent/rostering/domain/model"
)

// executeAITask 使用 AI 执行任务（带工具调用）
// 支持多班次任务拆分
// 返回值：shiftDrafts（按班次组织的排班草案）、reasoning（AI推理说明）、error（错误）
// 返回班次维度的数据（每个班次独立处理）
// 【占位信息格式统一】已移除 map 格式的 occupiedSlots 参数，统一使用 taskContext.OccupiedSlots（强类型数组）
func (e *ProgressiveTaskExecutor) executeAITask(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
) (map[string]*d_model.ShiftScheduleDraft, string, error) {
	e.logger.Info("Executing task with AI", "taskID", task.ID, "title", task.Title)

	// 解析任务识别目标班次
	shiftSpecs, parsingReasoning, err := e.parseTaskTargetShifts(ctx, task, shifts)
	if err != nil {
		// 只有格式错误（JSON解析失败等）才会返回错误，应该传播给上层处理
		e.logger.Error("Failed to parse task target shifts (format error)",
			"taskID", task.ID,
			"error", err)
		// 返回错误，让上层决定如何处理（返回给前端）
		return nil, "", fmt.Errorf("failed to parse task target shifts: %w", err)
	}

	// 记录解析结果（包括空列表的情况）
	if len(shiftSpecs) == 0 {
		e.logger.Info("Task parsed successfully (empty shifts, task needs no processing)",
			"taskID", task.ID,
			"reasoning", parsingReasoning)
	} else {
		e.logger.Info("Task parsed successfully",
			"taskID", task.ID,
			"parsedShiftsCount", len(shiftSpecs),
			"reasoning", parsingReasoning)

		// 【P0修复】更新task.TargetShifts为实际解析出的班次ID
		// 原因：LLM解析出的班次ID才是实际写入数据的位置，必须更新任务定义
		// 这样后续所有使用task.TargetShifts的地方（如前端显示）都能正确工作
		parsedShiftIDs := make([]string, 0, len(shiftSpecs))
		for _, spec := range shiftSpecs {
			parsedShiftIDs = append(parsedShiftIDs, spec.ShiftID)
		}
		task.TargetShifts = parsedShiftIDs
	}

	// 如果解析出的班次为空，说明任务不需要处理，直接返回空结果
	if len(shiftSpecs) == 0 {
		e.logger.Info("Task has no target shifts, returning empty result",
			"taskID", task.ID,
			"reasoning", parsingReasoning)
		return make(map[string]*d_model.ShiftScheduleDraft), parsingReasoning, nil
	}

	// ============================================================
	// 【新增】关联班次分组识别
	// 识别哪些班次需要一起排班（共享规则和人员信息）
	// ============================================================
	shiftGroups, groupingReasoning, err := e.identifyShiftGroups(ctx, shiftSpecs, shifts, rules)
	if err != nil {
		e.logger.Warn("Failed to identify shift groups, using single group fallback",
			"taskID", task.ID,
			"error", err)
		// 失败时所有班次放入同一组（保守策略）
		shiftGroups, groupingReasoning, _ = e.fallbackToSingleGroup(shiftSpecs, "分组识别失败，采用保守策略")
	}

	e.logger.Info("Shift grouping completed",
		"taskID", task.ID,
		"groupCount", len(shiftGroups),
		"groupingReasoning", groupingReasoning)

	var allReasoning []string
	allReasoning = append(allReasoning, fmt.Sprintf("任务解析：%s", parsingReasoning))
	allReasoning = append(allReasoning, fmt.Sprintf("班次分组：%s", groupingReasoning))

	successCount := 0
	failCount := 0

	// 为每个班次创建独立的 draft
	shiftDrafts := make(map[string]*d_model.ShiftScheduleDraft) // shiftID -> draft

	// 收集失败班次信息
	failedShifts := make(map[string]*d_model.ShiftFailureInfo)
	successfulShifts := make([]string, 0)

	// ============================================================
	// 按分组处理班次
	// 每个分组内的班次共享规则和人员信息
	// ============================================================
	totalShiftsProcessed := 0
	totalShiftsToProcess := len(shiftSpecs)

	for groupIdx, group := range shiftGroups {
		// 获取分组相关的规则（组内所有班次共享）
		groupRules := e.getGroupRelatedRules(group, rules)

		// 获取分组相关的人员（组内所有班次共享）
		groupStaff := e.getGroupRelatedStaff(ctx, group, staffList)

		// 按班次优先级排序组内班次
		sortedGroupSpecs := e.sortShiftSpecsByDependency(group.Shifts)

		// 循环处理组内每个班次
		for _, spec := range sortedGroupSpecs {
			totalShiftsProcessed++

			// 查找班次详细信息
			var shift *d_model.Shift
			for _, s := range shifts {
				if s.ID == spec.ShiftID {
					shift = s
					break
				}
			}
			if shift == nil {
				e.logger.Warn("Shift not found, skipping",
					"shiftID", spec.ShiftID,
					"shiftName", spec.ShiftName)
				failCount++
				continue
			}

			// 发送进度通知
			e.notifyProgress(&ShiftProgressInfo{
				ShiftID:   shift.ID,
				ShiftName: shift.Name,
				Current:   totalShiftsProcessed,
				Total:     totalShiftsToProcess,
				Status:    "executing",
				Message:   fmt.Sprintf("正在处理班次 [%s]（分组%d，%d/%d）...", shift.Name, groupIdx+1, totalShiftsProcessed, totalShiftsToProcess),
			})

			// 获取PersonalNeeds（从任务上下文获取）
			var personalNeeds map[string][]*d_model.PersonalNeed
			if e.taskContext != nil && e.taskContext.PersonalNeeds != nil {
				personalNeeds = e.taskContext.PersonalNeeds
			}

			// 为当前班次创建独立的 currentDraft
			// 从 WorkingDraft 中提取只包含当前班次的数据，避免不同班次的数据混淆
			shiftCurrentDraft := d_model.NewShiftScheduleDraft()
			if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
				shiftDraftData := e.taskContext.WorkingDraft.Shifts[shift.ID]
				if shiftDraftData != nil && shiftDraftData.Days != nil {
					for date, dayShift := range shiftDraftData.Days {
						// 只提取动态排班（IsFixed == false），固定排班不需要传递给 AI
						if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
							if shiftCurrentDraft.Schedule == nil {
								shiftCurrentDraft.Schedule = make(map[string][]string)
							}
							// 复制人员ID列表（避免引用共享）
							shiftCurrentDraft.Schedule[date] = append([]string{}, dayShift.StaffIDs...)
						}
					}
				}
			}

			// ============================================================
			// 【改进】使用分组人员代替单班次人员过滤
			// 组内班次共享人员池，确保规则一致性
			// ============================================================
			shiftStaffList := groupStaff

			// 【重试逻辑】从配置中获取最大重试次数和AI分析开关
			cfg := e.configurator.GetConfig()
			maxRetries := 3
			if cfg.ScheduleV3.Retry.MaxShiftRetries > 0 {
				maxRetries = cfg.ScheduleV3.Retry.MaxShiftRetries
			}
			enableAIAnalysis := true
			if !cfg.ScheduleV3.Retry.EnableAIAnalysis {
				enableAIAnalysis = false
			}

			var draft *d_model.ShiftScheduleDraft
			var reasoning string
			var lastError error
			var retryContext = &d_model.ShiftRetryContext{
				RetryCount:        0,
				IsManualRetry:     false,
				FailureHistory:    make([]string, 0),
				AIRecommendations: "",
			}

			// 获取该班次的人数需求（总需求）
			shiftStaffRequirements := make(map[string]int)
			if reqs, ok := staffRequirements[shift.ID]; ok {
				shiftStaffRequirements = reqs
			}

			// 【关键修改】计算该班次每日的固定排班人数，并从需求中减去
			// 这样传给 LLM 的是"还需要安排的动态人数"，而不是"总需求人数"
			shiftFixedCounts := make(map[string]int)
			if e.taskContext != nil && len(e.taskContext.FixedAssignments) > 0 {
				for _, fa := range e.taskContext.FixedAssignments {
					if fa.ShiftID == shift.ID {
						shiftFixedCounts[fa.Date] = len(fa.StaffIDs)
					}
				}
			}

			// 【关键修复】获取 WorkingDraft 中已有的动态排班人数
			existingDynamicCounts := make(map[string]int)
			if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
				if shiftDraft, exists := e.taskContext.WorkingDraft.Shifts[shift.ID]; exists && shiftDraft != nil && shiftDraft.Days != nil {
					for date, dayShift := range shiftDraft.Days {
						if dayShift != nil && !dayShift.IsFixed && len(dayShift.StaffIDs) > 0 {
							existingDynamicCounts[date] = len(dayShift.StaffIDs)
						}
					}
				}
			}

			// 计算动态需求（减去固定排班人数和已有动态排班人数）
			dynamicStaffRequirements := make(map[string]int)
			for date, totalRequired := range shiftStaffRequirements {
				fixedCount := shiftFixedCounts[date]
				existingDynamicCount := existingDynamicCounts[date]
				dynamicRequired := totalRequired - fixedCount - existingDynamicCount
				if dynamicRequired < 0 {
					dynamicRequired = 0 // 如果已超配，动态需求为0
				}
				dynamicStaffRequirements[date] = dynamicRequired
				if fixedCount > 0 || existingDynamicCount > 0 {
					e.logger.Debug("Adjusted staff requirements for existing assignments (executeAITask)",
						"shiftID", shift.ID,
						"date", date,
						"totalRequired", totalRequired,
						"fixedCount", fixedCount,
						"existingDynamicCount", existingDynamicCount,
						"dynamicRequired", dynamicRequired)
				}
			}

			// 【优化】检查是否所有日期的动态需求都为0
			hasNonZeroRequirement := false
			for _, count := range dynamicStaffRequirements {
				if count > 0 {
					hasNonZeroRequirement = true
					break
				}
			}
			if !hasNonZeroRequirement {
				e.logger.Info("Skipping shift with all zero dynamic requirements (fixed assignments already meet all needs)",
					"shiftID", shift.ID,
					"shiftName", shift.Name,
					"totalDates", len(dynamicStaffRequirements))
				successCount++
				// 创建空的draft并添加到结果中
				shiftDrafts[shift.ID] = d_model.NewShiftScheduleDraft()
				successfulShifts = append(successfulShifts, shift.ID)
				continue
			}

			// ============================================================
			// 【改进】使用分组规则代替单班次规则过滤
			// 组内班次共享相关规则，确保跨班次约束被正确应用
			// ============================================================
			filteredRules := groupRules

			for retryAttempt := 0; retryAttempt < maxRetries; retryAttempt++ {
				retryContext.RetryCount = retryAttempt

				// 如果是重试，清理本班次的占位和检查点，确保从头开始重排
				if retryAttempt > 0 {
					e.logger.Info("Retrying shift execution",
						"shiftID", shift.ID,
						"shiftName", shift.Name,
						"retryAttempt", retryAttempt,
						"maxRetries", maxRetries)

					// 发送进度通知：重试
					e.notifyProgress(&ShiftProgressInfo{
						ShiftID:   shift.ID,
						ShiftName: shift.Name,
						Current:   totalShiftsProcessed,
						Total:     totalShiftsToProcess,
						Status:    "retrying",
						Message:   fmt.Sprintf("🔄 班次 [%s] 第%d次重试中...", shift.Name, retryAttempt),
					})

					// 【BUG1修复】重试前，用上一轮draft的结果更新shiftCurrentDraft
					// 原因：shiftCurrentDraft在循环外初始化，不会随迭代更新
					// 导致定向重试时currentDraft为空，非目标日期无法从currentDraft复制数据
					if draft != nil && draft.Schedule != nil && len(draft.Schedule) > 0 {
						// 用上一轮的draft结果重建shiftCurrentDraft
						shiftCurrentDraft = d_model.NewShiftScheduleDraft()
						shiftCurrentDraft.Schedule = make(map[string][]string)
						for date, staffIDs := range draft.Schedule {
							shiftCurrentDraft.Schedule[date] = make([]string, len(staffIDs))
							copy(shiftCurrentDraft.Schedule[date], staffIDs)
						}
					}

					// 【BUG1修复续】同步清理WorkingDraft中目标日期的数据
					// 原因：WorkingDraft在第0次迭代结尾被LLM4调整结果更新，但重试时不会自动清理
					// 导致existingDynamicCounts包含旧数据，使dynamicRequired计算为0，目标日期被错误skip
					if e.taskContext != nil && e.taskContext.WorkingDraft != nil {
						if shiftDraftEntry, exists := e.taskContext.WorkingDraft.Shifts[shift.ID]; exists && shiftDraftEntry != nil && shiftDraftEntry.Days != nil {
							if retryContext.RetryOnlyTargetDates && len(retryContext.TargetRetryDates) > 0 {
								// 定向重试：只清理目标日期
								for _, retryDate := range retryContext.TargetRetryDates {
									delete(shiftDraftEntry.Days, retryDate)
								}
							} else {
								// 全量重试：清理所有日期
								shiftDraftEntry.Days = make(map[string]*d_model.DayShift)
							}
						}
					}

					// 【P0修复】定向重试模式下，只清理目标日期的占位和草案
					// 【占位信息格式统一】已移除 map 格式的 occupiedSlots 参数
					// 之前的bug：全量清理导致非目标日期的排班数据丢失
					if retryContext.RetryOnlyTargetDates && len(retryContext.TargetRetryDates) > 0 {
						// 定向重试：只清理目标日期
						e.cleanShiftOccupiedSlots(shift, retryContext.TargetRetryDates, shiftCurrentDraft)
					} else {
						// 全量重试：清理所有日期
						e.cleanShiftOccupiedSlots(shift, task.TargetDates, shiftCurrentDraft)
					}

					// 【关键修复】清理检查点，确保重试时从头开始重排整个班次
					if e.taskContext != nil {
						e.taskContext.ClearShiftCheckpoint(shift.ID)
						e.logger.Debug("Cleared shift checkpoint for retry",
							"shiftID", shift.ID,
							"retryAttempt", retryAttempt)
					}
				}

				// ============================================================
				// 【渐进式排班】使用按天执行的方式生成排班
				// ============================================================
				// 准备失败问题点（仅重试时有）
				var failedDatesIssues map[string][]string
				if retryContext.LastValidationResult != nil {
					failedDatesIssues = e.extractIssuesByDate(retryContext.LastValidationResult)
				}

				// 创建临时子任务用于执行
				subTask := &d_model.ProgressiveTask{
					ID:           fmt.Sprintf("%s_group_%d_shift_%s", task.ID, groupIdx+1, shift.ID),
					Title:        task.Title,
					Description:  spec.Description,
					TargetShifts: []string{shift.ID},
					TargetDates:  task.TargetDates,
					TargetStaff:  task.TargetStaff,
					RuleIDs:      task.RuleIDs,
					Priority:     task.Priority,
					Status:       task.Status,
				}

				// 【占位信息格式统一】使用渐进式按天执行（已移除 map 格式的 occupiedSlots 参数）
				draft, reasoning, lastError = e.executeProgressiveShiftScheduling(
					ctx,
					subTask,
					shift,
					shifts,
					filteredRules,
					shiftStaffList,
					staffRequirements,
					shiftCurrentDraft,
					personalNeeds,
					totalShiftsProcessed, // shiftIndex
					totalShiftsToProcess, // totalShifts
					retryContext,         // 重试上下文
					failedDatesIssues,    // 失败问题点
				)

				if lastError != nil {
					e.logger.Warn("Shift execution failed",
						"shiftID", shift.ID,
						"retryAttempt", retryAttempt,
						"error", lastError)
					retryContext.FailureHistory = append(retryContext.FailureHistory,
						fmt.Sprintf("尝试%d：%s", retryAttempt+1, lastError.Error()))
					continue
				}

				// 执行后立即校验
				validationResult := e.validateSingleShift(
					ctx,
					shift,
					draft,
					rules,
					shiftStaffRequirements,
					staffList,
					shifts, // 传入所有班次信息，用于真正的时间重叠检查
				)

				if validationResult.Passed {
					// 校验通过，成功退出重试循环
					e.logger.Info("Shift execution succeeded",
						"shiftID", shift.ID,
						"shiftName", shift.Name,
						"retryAttempt", retryAttempt)
					break
				}

				// 校验失败
				e.logger.Warn("Shift validation failed",
					"shiftID", shift.ID,
					"retryAttempt", retryAttempt,
					"validationSummary", validationResult.Summary)

				// 转为语义化描述
				failureDesc := e.formatFailureToSemantic(draft, validationResult, shift.Name)
				retryContext.FailureHistory = append(retryContext.FailureHistory,
					fmt.Sprintf("尝试%d：%s", retryAttempt+1, failureDesc))

				// 构建详细的校验结果描述
				var validationDetails strings.Builder
				validationDetails.WriteString(fmt.Sprintf("🔍 校验结果（尝试 %d/%d）：\n", retryAttempt+1, maxRetries))
				if len(validationResult.StaffCountIssues) > 0 {
					validationDetails.WriteString("  📊 人数问题：\n")
					for _, issue := range validationResult.StaffCountIssues {
						validationDetails.WriteString(fmt.Sprintf("    - %s\n", issue.Description))
					}
				}
				if len(validationResult.ShiftRuleIssues) > 0 {
					validationDetails.WriteString("  ⏰ 时间冲突：\n")
					for _, issue := range validationResult.ShiftRuleIssues {
						validationDetails.WriteString(fmt.Sprintf("    - %s\n", issue.Description))
					}
				}
				if len(validationResult.RuleComplianceIssues) > 0 {
					validationDetails.WriteString("  📋 规则违反：\n")
					for idx, issue := range validationResult.RuleComplianceIssues {
						if idx >= 5 {
							validationDetails.WriteString(fmt.Sprintf("    ... 还有 %d 个规则问题\n", len(validationResult.RuleComplianceIssues)-5))
							break
						}
						validationDetails.WriteString(fmt.Sprintf("    - %s\n", issue.Description))
					}
				}

				// 发送校验失败的进度通知（包含详细校验结果）
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:   shift.ID,
					ShiftName: shift.Name,
					Current:   totalShiftsProcessed,
					Total:     totalShiftsToProcess,
					Status:    "validating",
					Message:   validationDetails.String(),
				})

				// 如果还有重试机会，根据配置决定是否调用AI分析失败原因
				if retryAttempt < maxRetries-1 {
					// 只有开启AI分析时才调用
					if enableAIAnalysis {
						recommendations := e.analyzeShiftFailureWithAI(
							ctx,
							shift,
							subTask,
							validationResult,
							retryContext.FailureHistory,
						)
						retryContext.AIRecommendations = recommendations

						// 发送AI分析结果给前端
						e.notifyProgress(&ShiftProgressInfo{
							ShiftID:   shift.ID,
							ShiftName: shift.Name,
							Current:   totalShiftsProcessed,
							Total:     totalShiftsToProcess,
							Status:    "retrying",
							Message:   fmt.Sprintf("🤖 AI纠正建议：%s", recommendations),
							Reasoning: recommendations,
						})
					} else {
						e.logger.Debug("AI failure analysis disabled by config",
							"shiftID", shift.ID,
							"retryAttempt", retryAttempt)
						retryContext.AIRecommendations = "根据校验结果调整排班方案"
					}
					retryContext.LastValidationResult = validationResult

					// 【新增】提取需要重排的目标日期和冲突班次
					// 这样重试时只重排有问题的日期，而不是全部重排
					targetRetryDates, conflictingShiftIDs := e.extractTargetRetryInfo(validationResult, shift.ID)
					if len(targetRetryDates) > 0 {
						retryContext.TargetRetryDates = targetRetryDates
						retryContext.RetryOnlyTargetDates = true
						retryContext.ConflictingShiftIDs = conflictingShiftIDs

						e.logger.Info("Setting targeted retry for specific dates",
							"shiftID", shift.ID,
							"shiftName", shift.Name,
							"targetRetryDates", targetRetryDates,
							"conflictingShiftIDs", conflictingShiftIDs)

						// 发送通知：将只重排特定日期
						e.notifyProgress(&ShiftProgressInfo{
							ShiftID:   shift.ID,
							ShiftName: shift.Name,
							Current:   totalShiftsProcessed,
							Total:     totalShiftsToProcess,
							Status:    "retrying",
							Message:   fmt.Sprintf("🎯 定向重排：仅重新安排 %v 的排班", targetRetryDates),
						})
					}
				} else {
					// 最后一次尝试失败，记录为最终失败
					lastError = fmt.Errorf("校验失败（已重试%d次）：%s", maxRetries, validationResult.Summary)
				}
			}

			// 检查最终结果
			if lastError != nil {
				e.logger.Error("Shift execution failed after max retries",
					"shiftID", shift.ID,
					"shiftName", shift.Name,
					"maxRetries", maxRetries,
					"error", lastError)
				failCount++

				// 发送进度通知：班次执行失败
				e.notifyProgress(&ShiftProgressInfo{
					ShiftID:   shift.ID,
					ShiftName: shift.Name,
					Current:   totalShiftsProcessed,
					Total:     totalShiftsToProcess,
					Status:    "failed",
					Message:   fmt.Sprintf("❌ 班次 [%s] 排班失败（已重试%d次）: %s", shift.Name, maxRetries, lastError.Error()),
				})

				// 记录失败班次信息（用于部分成功处理）
				failureInfo := &d_model.ShiftFailureInfo{
					ShiftID:          shift.ID,
					ShiftName:        shift.Name,
					FailureSummary:   lastError.Error(),
					AutoRetryCount:   maxRetries,
					ManualRetryCount: 0,
					FailureHistory:   retryContext.FailureHistory,
					LastError:        lastError.Error(),
				}
				// 如果有最后一次校验结果，保存校验问题
				if retryContext.LastValidationResult != nil {
					failureInfo.ValidationIssues = append(
						retryContext.LastValidationResult.StaffCountIssues,
						retryContext.LastValidationResult.ShiftRuleIssues...,
					)
					failureInfo.ValidationIssues = append(
						failureInfo.ValidationIssues,
						retryContext.LastValidationResult.RuleComplianceIssues...,
					)
				}
				failedShifts[shift.ID] = failureInfo

				// 继续处理其他班次，不中断整个任务
				continue
			}

			successCount++
			successfulShifts = append(successfulShifts, shift.ID)
			shiftDrafts[shift.ID] = draft

			// 发送进度通知：班次执行成功（包含预览数据和AI解释）
			previewJSON := ""
			if draft != nil && draft.Schedule != nil {
				if jsonBytes, err := json.Marshal(draft.Schedule); err == nil {
					previewJSON = string(jsonBytes)
				}
			}
			e.notifyProgress(&ShiftProgressInfo{
				ShiftID:     shift.ID,
				ShiftName:   shift.Name,
				Current:     totalShiftsProcessed,
				Total:       totalShiftsToProcess,
				Status:      "success",
				Message:     fmt.Sprintf("✅ 班次 [%s] 排班完成 (%d/%d)", shift.Name, totalShiftsProcessed, totalShiftsToProcess),
				Reasoning:   reasoning,
				PreviewData: previewJSON,
			})

			// 【P0修复】立即更新 WorkingDraft 和 occupiedSlots
			// 原因：后续班次（如下夜班）需要知道前面班次（如本部夜班、江北夜班）的排班结果
			// 这样才能正确应用连班规则（如 rule_11：下夜班人员必须前一日上本部夜班或江北夜班）
			if e.taskContext != nil && e.taskContext.WorkingDraft != nil && draft != nil && draft.Schedule != nil {
				// 更新 WorkingDraft
				if e.taskContext.WorkingDraft.Shifts == nil {
					e.taskContext.WorkingDraft.Shifts = make(map[string]*d_model.ShiftDraft)
				}
				if e.taskContext.WorkingDraft.Shifts[shift.ID] == nil {
					e.taskContext.WorkingDraft.Shifts[shift.ID] = &d_model.ShiftDraft{
						ShiftID: shift.ID,
						Days:    make(map[string]*d_model.DayShift),
					}
				}
				shiftDraftEntry := e.taskContext.WorkingDraft.Shifts[shift.ID]
				if shiftDraftEntry.Days == nil {
					shiftDraftEntry.Days = make(map[string]*d_model.DayShift)
				}

				// 构建人员ID到姓名的映射
				staffNamesMap := BuildStaffNamesMap(staffList)

				for date, staffIDs := range draft.Schedule {
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

					// 【占位信息格式统一】更新 taskContext.OccupiedSlots（强类型数组）
					if e.taskContext != nil {
						for _, staffID := range staffIDs {
							e.taskContext.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(
								e.taskContext.OccupiedSlots,
								d_model.StaffOccupiedSlot{
									StaffID:   staffID,
									Date:      date,
									ShiftID:   shift.ID,
									ShiftName: shift.Name,
									Source:    "draft",
								},
							)
						}
					}
				}
			}

			if reasoning != "" {
				allReasoning = append(allReasoning, fmt.Sprintf("【%s】%s", shift.Name, reasoning))
			}
		} // end of shiftIdx loop (组内班次循环)
	} // end of groupIdx loop (分组循环)

	// 记录日志
	e.logger.Info("AI task execution completed",
		"taskID", task.ID,
		"groupCount", len(shiftGroups),
		"totalShifts", totalShiftsToProcess,
		"successCount", successCount,
		"failCount", failCount,
		"shiftDrafts", len(shiftDrafts))

	combinedReasoning := strings.Join(allReasoning, "\n\n")

	// 返回失败信息通过特殊的方式传递（保存到执行器属性中）
	e.lastFailedShifts = failedShifts
	e.lastSuccessfulShifts = successfulShifts

	return shiftDrafts, combinedReasoning, nil
}

// executeAITaskV3 使用 AI 执行任务（V3版本，直接使用 taskContext.OccupiedSlots）
// 这是 executeAITask 的重构版本，不再需要 occupiedSlots map 参数
// 所有占位信息直接从 e.taskContext.OccupiedSlots 读写
func (e *ProgressiveTaskExecutor) executeAITaskV3(
	ctx context.Context,
	task *d_model.ProgressiveTask,
	shifts []*d_model.Shift,
	rules []*d_model.Rule,
	staffList []*d_model.Employee,
	staffRequirements map[string]map[string]int,
) (map[string]*d_model.ShiftScheduleDraft, string, error) {
	// 【占位信息格式统一】已移除 map 格式的 occupiedSlots，统一使用 taskContext.OccupiedSlots
	// 直接调用实现，不再需要转换
	return e.executeAITask(ctx, task, shifts, rules, staffList, staffRequirements)
}
