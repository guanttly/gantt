package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gantt-saas/internal/common/response"
	"gantt-saas/internal/infra/config"
	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SystemConfig 系统配置模型。
type SystemConfig struct {
	ID    string `gorm:"primaryKey;size:64" json:"id"`
	Key   string `gorm:"size:128;not null;uniqueIndex:uk_config_key" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

var systemConfigAllowedKeys = map[string]struct{}{
	"ai_enabled":  {},
	"ai_provider": {},
	"ai_model":    {},
	"ai_api_key":  {},
	"ai_base_url": {},
	"system_name": {},
	"system_logo": {},
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
		if isAllowedSystemConfigKey(c.Key) {
			configMap[c.Key] = c.Value
		}
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
	filtered := filterSystemConfigs(input.Configs)

	ctx := tenant.SkipTenantGuard(r.Context())
	for key, value := range filtered {
		var existing SystemConfig
		err := h.db.WithContext(ctx).Where("`key` = ?", key).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			cfg := SystemConfig{ID: uuid.New().String(), Key: key, Value: value}
			if err := h.db.WithContext(ctx).Create(&cfg).Error; err != nil {
				response.InternalError(w, "创建系统配置失败")
				return
			}
		} else if err != nil {
			response.InternalError(w, "查询系统配置失败")
			return
		} else {
			if err := h.db.WithContext(ctx).Model(&existing).Update("value", value).Error; err != nil {
				response.InternalError(w, "更新系统配置失败")
				return
			}
		}
	}
	response.OK(w, filtered)
}

func isAllowedSystemConfigKey(key string) bool {
	_, ok := systemConfigAllowedKeys[key]
	return ok
}

func filterSystemConfigs(configs map[string]string) map[string]string {
	filtered := make(map[string]string)
	for key, value := range configs {
		if isAllowedSystemConfigKey(key) {
			filtered[key] = value
		}
	}
	return filtered
}

// LoadAIConfigFromDB 从数据库 system_configs 加载 AI 配置，合并到已有的 AIConfig 上。
// 数据库配置优先于文件配置。
func LoadAIConfigFromDB(db *gorm.DB, aiCfg *config.AIConfig) error {
	ctx := tenant.SkipTenantGuard(context.Background())
	var configs []SystemConfig
	if err := db.WithContext(ctx).Find(&configs).Error; err != nil {
		return err
	}

	m := make(map[string]string, len(configs))
	for _, c := range configs {
		m[c.Key] = c.Value
	}

	// 如果数据库没有 AI 配置，保持文件配置不变
	provider := m["ai_provider"]
	if provider == "" {
		return nil
	}

	apiKey := m["ai_api_key"]
	baseURL := m["ai_base_url"]
	model := m["ai_model"]
	enabled := m["ai_enabled"]

	if enabled != "" && enabled != "true" && enabled != "1" {
		return nil
	}

	baseURL = strings.TrimRight(baseURL, "/")

	providerCfg := &config.AIProviderConfig{
		Enabled: true,
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: 60 * time.Second,
	}

	aiCfg.Enabled = true
	aiCfg.DefaultProvider = provider

	switch provider {
	case "openai":
		aiCfg.OpenAI = providerCfg
	case "bailian":
		aiCfg.Bailian = providerCfg
	case "ollama":
		aiCfg.Ollama = providerCfg
	}

	return nil
}
