package mapper

import (
	"encoding/json"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/internal/entity"
)

// GroupEntityToModel 将实体转换为领域模型
func GroupEntityToModel(e *entity.GroupEntity) (*model.Group, error) {
	if e == nil {
		return nil, nil
	}

	group := &model.Group{
		ID:          e.ID,
		OrgID:       e.OrgID,
		Name:        e.Name,
		Code:        e.Code,
		Type:        model.GroupType(e.Type),
		Description: e.Description,
		ParentID:    e.ParentID,
		LeaderID:    e.LeaderID,
		Status:      model.GroupStatus(e.Status),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		DeletedAt:   e.DeletedAt,
	}

	// 解析Attributes JSON
	if e.Attributes != "" {
		if err := json.Unmarshal([]byte(e.Attributes), &group.Attributes); err != nil {
			return nil, err
		}
	}

	return group, nil
}

// GroupModelToEntity 将领域模型转换为实体
func GroupModelToEntity(m *model.Group) (*entity.GroupEntity, error) {
	if m == nil {
		return nil, nil
	}

	group := &entity.GroupEntity{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Name:        m.Name,
		Code:        m.Code,
		Type:        string(m.Type),
		Description: m.Description,
		ParentID:    m.ParentID,
		LeaderID:    m.LeaderID,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
	}

	// 序列化Attributes为JSON
	if m.Attributes != nil {
		attrsJSON, err := json.Marshal(m.Attributes)
		if err != nil {
			return nil, err
		}
		group.Attributes = string(attrsJSON)
	}

	return group, nil
}

// GroupEntitiesToModels 批量转换实体到领域模型
func GroupEntitiesToModels(entities []*entity.GroupEntity) ([]*model.Group, error) {
	if entities == nil {
		return nil, nil
	}

	models := make([]*model.Group, 0, len(entities))
	for _, e := range entities {
		m, err := GroupEntityToModel(e)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}

	return models, nil
}

// GroupMemberEntityToModel 将成员实体转换为领域模型
func GroupMemberEntityToModel(e *entity.GroupMemberEntity) *model.GroupMember {
	if e == nil {
		return nil
	}

	return &model.GroupMember{
		ID:         e.ID,
		GroupID:    e.GroupID,
		EmployeeID: e.EmployeeID,
		Role:       e.Role,
		JoinedAt:   e.JoinedAt,
		LeftAt:     e.LeftAt,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

// GroupMemberModelToEntity 将成员领域模型转换为实体
func GroupMemberModelToEntity(m *model.GroupMember) *entity.GroupMemberEntity {
	if m == nil {
		return nil
	}

	return &entity.GroupMemberEntity{
		ID:         m.ID,
		GroupID:    m.GroupID,
		EmployeeID: m.EmployeeID,
		Role:       m.Role,
		JoinedAt:   m.JoinedAt,
		LeftAt:     m.LeftAt,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

// GroupMemberEntitiesToModels 批量转换成员实体到领域模型
func GroupMemberEntitiesToModels(entities []*entity.GroupMemberEntity) []*model.GroupMember {
	if entities == nil {
		return nil
	}

	models := make([]*model.GroupMember, 0, len(entities))
	for _, e := range entities {
		models = append(models, GroupMemberEntityToModel(e))
	}

	return models
}
