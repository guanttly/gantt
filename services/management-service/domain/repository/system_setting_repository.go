package repository

import (
	"context"
	"jusha/gantt/service/management/domain/model"
)

// ISystemSettingRepository 系统设置仓储接口
type ISystemSettingRepository interface {
	// GetByKey 根据组织ID和键获取设置
	GetByKey(ctx context.Context, orgID, key string) (*model.SystemSetting, error)

	// Set 设置或更新系统设置
	Set(ctx context.Context, setting *model.SystemSetting) error

	// GetAll 获取组织的所有设置
	GetAll(ctx context.Context, orgID string) ([]*model.SystemSetting, error)

	// Delete 删除系统设置
	Delete(ctx context.Context, orgID, key string) error
}

