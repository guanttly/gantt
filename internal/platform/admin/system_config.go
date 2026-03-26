package admin

import (
	"encoding/json"
	"net/http"

	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"gorm.io/gorm"
)

// SystemConfig 系统配置模型。
type SystemConfig struct {
	ID    string `gorm:"primaryKey;size:64" json:"id"`
	Key   string `gorm:"size:128;not null;uniqueIndex:uk_config_key" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

// TableName 指定表名。
func (SystemConfig) TableName() string {
	return "system_configs"
}

// SystemConfigHandler 系统配置处理器。
type SystemConfigHandler struct {
	db *gorm.DB
}

// NewSystemConfigHandler 创建系统配置处理器。
func NewSystemConfigHandler(db *gorm.DB) *SystemConfigHandler {
	return &SystemConfigHandler{db: db}
}

// AutoMigrate 自动迁移表结构。
func (h *SystemConfigHandler) AutoMigrate() error {
	return h.db.AutoMigrate(&SystemConfig{})
}

// GetConfig 获取系统配置。
// GET /api/v1/admin/system/config
func (h *SystemConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := tenant.SkipTenantGuard(r.Context())
	var configs []SystemConfig
	if err := h.db.WithContext(ctx).Find(&configs).Error; err != nil {
		response.InternalError(w, "获取系统配置失败")
		return
	}

	configMap := make(map[string]string)
	for _, c := range configs {
		configMap[c.Key] = c.Value
	}
	response.OK(w, configMap)
}

// UpdateConfigInput 更新系统配置输入。
type UpdateConfigInput struct {
	Configs map[string]string `json:"configs"`
}

// UpdateConfig 更新系统配置。
// PUT /api/v1/admin/system/config
func (h *SystemConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var input UpdateConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}

	ctx := tenant.SkipTenantGuard(r.Context())
	for key, value := range input.Configs {
		result := h.db.WithContext(ctx).
			Where("`key` = ?", key).
			Assign(SystemConfig{Value: value}).
			FirstOrCreate(&SystemConfig{Key: key, Value: value})
		if result.Error != nil {
			response.InternalError(w, "更新系统配置失败")
			return
		}
	}
	response.OK(w, input.Configs)
}
