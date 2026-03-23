package utils

import (
	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 变更预览生成器 - 为前端生成预览数据（V3改进：使用强类型结构体）
// ============================================================

// BuildChangePreview 构建变更预览数据（用于前端展示）
//
// 参数:
//   - batch: 变更批次
//
// 返回:
//   - *d_model.ChangeDetailPreview: 前端需要的预览数据结构（强类型）
func BuildChangePreview(batch *d_model.ScheduleChangeBatch) *d_model.ChangeDetailPreview {
	if batch == nil {
		return &d_model.ChangeDetailPreview{
			TaskID:    "",
			TaskTitle: "",
			TaskIndex: 0,
			Timestamp: "",
			Shifts:    make([]*d_model.ShiftChangePreview, 0),
		}
	}

	// 按班次分组组织变更数据
	shiftsMap := make(map[string]*d_model.ShiftChangePreview)

	for _, change := range batch.Changes {
		// 确保班次存在
		if shiftsMap[change.ShiftID] == nil {
			shiftsMap[change.ShiftID] = &d_model.ShiftChangePreview{
				ShiftID:   change.ShiftID,
				ShiftName: change.ShiftName,
				Changes:   make([]*d_model.DateChangePreview, 0),
			}
		}

		// 添加日期变更
		dateChange := &d_model.DateChangePreview{
			Date:        change.Date,
			ChangeType:  string(change.ChangeType),
			BeforeIDs:   change.BeforeIDs,
			AfterIDs:    change.AfterIDs,
			BeforeNames: change.BeforeNames,
			AfterNames:  change.AfterNames,
		}

		shiftsMap[change.ShiftID].Changes = append(shiftsMap[change.ShiftID].Changes, dateChange)
	}

	// 将map转换为数组（保证输出格式一致性）
	shifts := make([]*d_model.ShiftChangePreview, 0, len(shiftsMap))
	for _, shiftPreview := range shiftsMap {
		shifts = append(shifts, shiftPreview)
	}

	// 构建最终结果
	result := &d_model.ChangeDetailPreview{
		TaskID:    batch.TaskID,
		TaskTitle: batch.TaskTitle,
		TaskIndex: batch.TaskIndex,
		Timestamp: batch.Timestamp,
		Shifts:    shifts,
	}

	return result
}
