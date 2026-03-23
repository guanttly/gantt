package service

import (
	"context"
)

// ISystemSettingService 系统设置服务接口
type ISystemSettingService interface {
	// GetSetting 获取系统设置值（如果不存在返回默认值）
	GetSetting(ctx context.Context, orgID, key string) (string, error)

	// SetSetting 设置系统设置值
	SetSetting(ctx context.Context, orgID, key, value string) error
}

