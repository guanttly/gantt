package service

import (
	"context"

	"jusha/gantt/service/management/domain/model"
)

// ISystemSettingService 系统设置领域服务接口
type ISystemSettingService interface {
	// GetSetting 获取系统设置（如果不存在返回默认值）
	GetSetting(ctx context.Context, orgID, key string) (string, error)

	// SetSetting 设置系统设置
	SetSetting(ctx context.Context, orgID, key, value, description string) error

	// GetAllSettings 获取组织的所有设置
	GetAllSettings(ctx context.Context, orgID string) ([]*model.SystemSetting, error)

	// DeleteSetting 删除系统设置
	DeleteSetting(ctx context.Context, orgID, key string) error
}

