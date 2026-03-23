package model

import (
	"time"
)

// SystemSetting 系统设置领域模型
type SystemSetting struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"orgId"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SystemSettingKey 系统设置键常量
const (
	// SettingKeyContinuousScheduling 连续排班配置键
	SettingKeyContinuousScheduling = "continuous_scheduling"
)

// SystemSettingDefaultValues 系统设置默认值
var SystemSettingDefaultValues = map[string]string{
	SettingKeyContinuousScheduling: "true", // 默认开启连续排班
}

// GetDefaultValue 获取设置的默认值
func GetSystemSettingDefaultValue(key string) string {
	if val, ok := SystemSettingDefaultValues[key]; ok {
		return val
	}
	return ""
}

