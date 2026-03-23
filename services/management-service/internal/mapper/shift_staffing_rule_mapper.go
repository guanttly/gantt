package mapper

import (
	"encoding/json"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// ShiftStaffingRuleEntityToModel 将班次计算规则实体转换为领域模型
func ShiftStaffingRuleEntityToModel(e *entity.ShiftStaffingRuleEntity) *model.ShiftStaffingRule {
	if e == nil {
		return nil
	}

	// 解析 ModalityRoomIDs JSON
	var modalityRoomIDs []string
	if e.ModalityRoomIDs != "" {
		_ = json.Unmarshal([]byte(e.ModalityRoomIDs), &modalityRoomIDs)
	}

	return &model.ShiftStaffingRule{
		ID:              e.ID,
		ShiftID:         e.ShiftID,
		ModalityRoomIDs: modalityRoomIDs,
		TimePeriodID:    e.TimePeriodID,
		AvgReportLimit:  e.AvgReportLimit,
		RoundingMode:    model.RoundingMode(e.RoundingMode),
		IsActive:        e.IsActive,
		Description:     e.Description,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

// ShiftStaffingRuleModelToEntity 将班次计算规则领域模型转换为实体
func ShiftStaffingRuleModelToEntity(m *model.ShiftStaffingRule) *entity.ShiftStaffingRuleEntity {
	if m == nil {
		return nil
	}

	// 序列化 ModalityRoomIDs 为 JSON
	modalityRoomIDsJSON, _ := json.Marshal(m.ModalityRoomIDs)

	return &entity.ShiftStaffingRuleEntity{
		ID:              m.ID,
		ShiftID:         m.ShiftID,
		ModalityRoomIDs: string(modalityRoomIDsJSON),
		TimePeriodID:    m.TimePeriodID,
		AvgReportLimit:  m.AvgReportLimit,
		RoundingMode:    string(m.RoundingMode),
		IsActive:        m.IsActive,
		Description:     m.Description,
		// 注意：不设置 CreatedAt 和 UpdatedAt，让 GORM 通过 autoCreateTime/autoUpdateTime 自动处理
	}
}

// ShiftStaffingRuleEntitiesToModels 批量转换实体为领域模型
func ShiftStaffingRuleEntitiesToModels(entities []*entity.ShiftStaffingRuleEntity) []*model.ShiftStaffingRule {
	if entities == nil {
		return nil
	}
	models := make([]*model.ShiftStaffingRule, len(entities))
	for i, e := range entities {
		models[i] = ShiftStaffingRuleEntityToModel(e)
	}
	return models
}
