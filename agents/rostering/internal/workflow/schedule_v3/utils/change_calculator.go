package utils

import (
	"fmt"
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 变更计算器 - 计算两个状态之间的变更
// ============================================================

// ComputeChangeBatch 计算任务执行前后的变更批次
//
// 参数:
//   - taskID: 任务ID
//   - taskTitle: 任务标题
//   - taskIndex: 任务序号（从1开始）
//   - beforeDraft: 变更前的排班草稿（通常是 WorkingDraft）
//   - afterSchedules: 变更后的排班数据（任务返回的结果）
//   - staffList: 人员列表（用于姓名映射）
//   - allStaffList: 所有人员列表（用于姓名映射补充）
//   - selectedShifts: 选定的班次列表（用于班次名称映射）
//
// 返回:
//   - *d_model.ScheduleChangeBatch: 变更批次
func ComputeChangeBatch(
	taskID, taskTitle string,
	taskIndex int,
	beforeDraft *d_model.ScheduleDraft,
	afterSchedules map[string]*d_model.ShiftScheduleDraft,
	staffList []*d_model.Employee,
	allStaffList []*d_model.Employee,
	selectedShifts []*d_model.Shift,
) *d_model.ScheduleChangeBatch {
	// 初始化变更批次
	batch := &d_model.ScheduleChangeBatch{
		TaskID:    taskID,
		TaskTitle: taskTitle,
		TaskIndex: taskIndex,
		Changes:   make([]*d_model.ScheduleChange, 0),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 构建姓名映射
	staffNamesMap := BuildStaffNamesMap(staffList)
	if allStaffList != nil {
		// 合并所有人员列表的姓名映射
		allStaffNamesMap := BuildStaffNamesMap(allStaffList)
		for id, name := range allStaffNamesMap {
			staffNamesMap[id] = name
		}
	}
	shiftNamesMap := BuildShiftNamesMap(selectedShifts)

	// 【V3增强】构建班次ID到时长的映射（用于工时计算）
	shiftDurationMap := make(map[string]float64)
	for _, shift := range selectedShifts {
		if shift != nil {
			shiftDurationMap[shift.ID] = GetShiftDurationHours(shift)
		}
	}

	// 收集所有需要比较的 (shiftID, date) 组合
	keysMap := make(map[string]map[string]bool) // shiftID -> date -> true

	// 从 beforeDraft 收集
	if beforeDraft != nil && beforeDraft.Shifts != nil {
		for shiftID, shiftDraft := range beforeDraft.Shifts {
			if shiftDraft.Days == nil {
				continue
			}
			for date := range shiftDraft.Days {
				if keysMap[shiftID] == nil {
					keysMap[shiftID] = make(map[string]bool)
				}
				keysMap[shiftID][date] = true
			}
		}
	}

	// 从 afterSchedules 收集
	for shiftID, shiftSchedule := range afterSchedules {
		if shiftSchedule.Schedule == nil {
			continue
		}
		for date := range shiftSchedule.Schedule {
			if keysMap[shiftID] == nil {
				keysMap[shiftID] = make(map[string]bool)
			}
			keysMap[shiftID][date] = true
		}
	}

	// 遍历每个 (shiftID, date) 组合，比较变更
	for shiftID, dates := range keysMap {
		for date := range dates {
			// 获取 before 状态
			var beforeIDs []string
			if beforeDraft != nil && beforeDraft.Shifts != nil {
				if shiftDraft, ok := beforeDraft.Shifts[shiftID]; ok && shiftDraft.Days != nil {
					if dayShift, ok := shiftDraft.Days[date]; ok && dayShift != nil {
						beforeIDs = dayShift.StaffIDs
					}
				}
			}

			// 获取 after 状态
			var afterIDs []string
			if afterSchedule, ok := afterSchedules[shiftID]; ok && afterSchedule.Schedule != nil {
				if ids, ok := afterSchedule.Schedule[date]; ok {
					afterIDs = ids
				}
			}

			// 判断变更类型
			changeType := determineChangeType(beforeIDs, afterIDs)
			if changeType == "" {
				// 无变更，跳过
				continue
			}

			// 转换ID为姓名
			beforeNames := MapIDsToNames(beforeIDs, staffNamesMap)
			afterNames := MapIDsToNames(afterIDs, staffNamesMap)

			// 获取班次名称
			shiftName := shiftNamesMap[shiftID]
			if shiftName == "" {
				shiftName = shiftID // 降级处理
			}

			// 创建变更记录
			change := &d_model.ScheduleChange{
				Date:        date,
				ShiftID:     shiftID,
				ShiftName:   shiftName,
				ChangeType:  string(changeType),
				BeforeIDs:   beforeIDs,
				AfterIDs:    afterIDs,
				BeforeNames: beforeNames,
				AfterNames:  afterNames,
				// 兼容旧字段
				OldStaff:      beforeIDs,
				OldStaffNames: beforeNames,
				NewStaff:      afterIDs,
				NewStaffNames: afterNames,
			}

			// 【V3增强】计算工时变化
			shiftDuration := shiftDurationMap[shiftID]
			if shiftDuration > 0 {
				change.WorkloadChanges = CalculateWorkloadChanges(
					beforeIDs, afterIDs, shiftDuration, staffNamesMap)
			}

			batch.Changes = append(batch.Changes, change)
		}
	}

	// 【V3增强】计算总工时变化
	batch.TotalWorkloadDelta = CalculateTotalWorkloadDelta(batch.Changes)

	return batch
}

// determineChangeType 判断变更类型
func determineChangeType(beforeIDs, afterIDs []string) string {
	beforeEmpty := len(beforeIDs) == 0
	afterEmpty := len(afterIDs) == 0

	if beforeEmpty && afterEmpty {
		// 前后都为空，无变更
		return ""
	}

	if beforeEmpty && !afterEmpty {
		// 前空后有，新增
		return "add"
	}

	if !beforeEmpty && afterEmpty {
		// 前有后空，删除
		return "remove"
	}

	// 前后都有，检查是否相同
	if SlicesEqual(beforeIDs, afterIDs) {
		// 内容相同，无变更
		return ""
	}

	// 内容不同，修改
	return "modify"
}

// ============================================================
// 辅助函数：格式化变更批次为可读文本
// ============================================================

// FormatChangeBatchSummary 格式化变更批次摘要
func FormatChangeBatchSummary(batch *d_model.ScheduleChangeBatch) string {
	if batch == nil || len(batch.Changes) == 0 {
		return "本次无需排班"
	}

	stats := batch.GetStats()
	summary := ""

	if stats.AddCount > 0 {
		summary += fmt.Sprintf("🆕 新增：%d 条\n", stats.AddCount)
	}
	if stats.ModifyCount > 0 {
		summary += fmt.Sprintf("✏️ 修改：%d 条\n", stats.ModifyCount)
	}
	if stats.RemoveCount > 0 {
		summary += fmt.Sprintf("🗑️ 删除：%d 条\n", stats.RemoveCount)
	}
	summary += fmt.Sprintf("📊 涉及：%d 个班次，%d 天\n", len(stats.AffectedShifts), len(stats.AffectedDates))
	summary += fmt.Sprintf("👥 总人次：%d", stats.TotalStaffSlots)

	return summary
}

// ============================================================
// 辅助函数：工时变化计算
// ============================================================

// CalculateWorkloadChanges 计算工时变化
// 针对一个班次变更，计算每个人员的工时变化
func CalculateWorkloadChanges(
	beforeIDs []string,
	afterIDs []string,
	shiftDuration float64,
	staffNamesMap map[string]string,
) map[string]*d_model.WorkloadChange {
	changes := make(map[string]*d_model.WorkloadChange)

	// 收集所有相关人员
	allStaffIDs := make(map[string]bool)
	for _, id := range beforeIDs {
		allStaffIDs[id] = true
	}
	for _, id := range afterIDs {
		allStaffIDs[id] = true
	}

	// 计算每个人员的工时变化
	for staffID := range allStaffIDs {
		before := 0.0
		after := 0.0

		// 检查是否在before中
		for _, id := range beforeIDs {
			if id == staffID {
				before = shiftDuration
				break
			}
		}

		// 检查是否在after中
		for _, id := range afterIDs {
			if id == staffID {
				after = shiftDuration
				break
			}
		}

		// 计算变化量
		delta := after - before

		// 只记录有变化的人员
		if delta != 0 {
			staffName := staffNamesMap[staffID]
			if staffName == "" {
				staffName = staffID
			}

			changes[staffID] = &d_model.WorkloadChange{
				StaffID:        staffID,
				StaffName:      staffName,
				WorkloadBefore: before,
				WorkloadAfter:  after,
				WorkloadDelta:  delta,
			}
		}
	}

	return changes
}

// CalculateTotalWorkloadDelta 计算总工时变化
// 汇总所有变更的工时变化
func CalculateTotalWorkloadDelta(changes []*d_model.ScheduleChange) float64 {
	totalDelta := 0.0

	for _, change := range changes {
		if change.WorkloadChanges != nil {
			for _, wc := range change.WorkloadChanges {
				totalDelta += wc.WorkloadDelta
			}
		}
	}

	return totalDelta
}
