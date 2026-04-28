package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AppConfigEntry stores application-level settings as key/value pairs.
type AppConfigEntry struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	AppCode   string    `gorm:"size:64;not null;uniqueIndex:uk_app_config_key,priority:1" json:"app_code"`
	Key       string    `gorm:"size:128;not null;uniqueIndex:uk_app_config_key,priority:2" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AppConfigEntry) TableName() string { return "app_configs" }

// AppWorkflowConfig stores platform-managed workflow metadata for an application.
type AppWorkflowConfig struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	AppCode     string    `gorm:"size:64;not null;uniqueIndex:uk_app_workflow,priority:1" json:"app_code"`
	WorkflowKey string    `gorm:"size:96;not null;uniqueIndex:uk_app_workflow,priority:2" json:"workflow_key"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Version     string    `gorm:"size:32;not null;default:''" json:"version"`
	Description string    `gorm:"type:text" json:"description"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AppWorkflowConfig) TableName() string { return "app_workflow_configs" }

// AIModelConfig stores model settings for one application workflow node.
type AIModelConfig struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	AppCode        string    `gorm:"size:64;not null;uniqueIndex:uk_ai_model_node,priority:1" json:"app_code"`
	WorkflowKey    string    `gorm:"size:96;not null;uniqueIndex:uk_ai_model_node,priority:2" json:"workflow_key"`
	NodeKey        string    `gorm:"size:96;not null;uniqueIndex:uk_ai_model_node,priority:3" json:"node_key"`
	Provider       string    `gorm:"size:32;not null;default:''" json:"provider"`
	Model          string    `gorm:"size:128;not null;default:''" json:"model"`
	TimeoutSeconds int       `gorm:"not null;default:60" json:"timeout_seconds"`
	Temperature    *float64  `json:"temperature"`
	MaxTokens      int       `gorm:"not null;default:0" json:"max_tokens"`
	Enabled        bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AIModelConfig) TableName() string { return "ai_model_configs" }

type AppConfigService struct {
	db *gorm.DB
}

func NewAppConfigService(db *gorm.DB) *AppConfigService {
	return &AppConfigService{db: db}
}

func (s *AppConfigService) AutoMigrate() error {
	return s.db.AutoMigrate(&AppConfigEntry{}, &AppWorkflowConfig{}, &AIModelConfig{})
}

type AppConfigHandler struct {
	svc *AppConfigService
}

func NewAppConfigHandler(svc *AppConfigService) *AppConfigHandler {
	return &AppConfigHandler{svc: svc}
}

func (h *AppConfigHandler) AutoMigrate() error { return h.svc.AutoMigrate() }

type AppConfigView struct {
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Settings    map[string]string    `json:"settings"`
	Workflows   []WorkflowConfigView `json:"workflows"`
}

type WorkflowConfigView struct {
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Enabled     bool               `json:"enabled"`
	Nodes       []WorkflowNodeView `json:"nodes"`
	Edges       []WorkflowEdgeView `json:"edges"`
}

type WorkflowNodeView struct {
	Key          string            `json:"key"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	Description  string            `json:"description"`
	Configurable bool              `json:"configurable"`
	Position     WorkflowPosition  `json:"position"`
	ModelConfig  AIModelConfigView `json:"model_config"`
}

type WorkflowEdgeView struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type WorkflowPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type AIModelConfigView struct {
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Temperature    *float64 `json:"temperature"`
	MaxTokens      int      `json:"max_tokens"`
	Enabled        bool     `json:"enabled"`
}

type appDefinition struct {
	Code        string
	Name        string
	Description string
	Settings    map[string]string
	Workflows   []workflowDefinition
}

type workflowDefinition struct {
	Key         string
	Name        string
	Version     string
	Description string
	Enabled     bool
	SortOrder   int
	Nodes       []workflowNodeDefinition
	Edges       []WorkflowEdgeView
}

type workflowNodeDefinition struct {
	Key                   string
	Name                  string
	Kind                  string
	Description           string
	Configurable          bool
	Position              WorkflowPosition
	DefaultProvider       string
	DefaultModel          string
	DefaultTimeoutSeconds int
	DefaultTemperature    *float64
	DefaultMaxTokens      int
}

var legacySchedulingSettingKeys = []string{
	"schedule_auto_publish",
	"schedule_lock_days",
}

func floatPtr(v float64) *float64 { return &v }

func defaultAppDefinitions() []appDefinition {
	return []appDefinition{
		{
			Code:        ai.AppScheduling,
			Name:        "智能排班",
			Description: "面向科室排班、规则解析与排班助手的应用配置。",
			Settings: map[string]string{
				"schedule_auto_publish": "false",
				"schedule_lock_days":    "3",
			},
			Workflows: []workflowDefinition{
				{
					Key:         ai.WorkflowAIChat,
					Name:        "AI 对话",
					Version:     "v1",
					Description: "识别用户意图并给出排班助手回复。",
					Enabled:     true,
					SortOrder:   10,
					Nodes: []workflowNodeDefinition{
						{Key: ai.NodeIntentClassify, Name: "意图识别", Kind: "llm", Description: "快速判断创建、调整、查询或普通对话意图。", Configurable: true, Position: WorkflowPosition{X: 1, Y: 1}, DefaultModel: "qwen-turbo", DefaultTimeoutSeconds: 20, DefaultTemperature: floatPtr(0.1), DefaultMaxTokens: 256},
						{Key: ai.NodeChatReply, Name: "对话回复", Kind: "llm", Description: "处理无法路由到固定动作的普通对话。", Configurable: true, Position: WorkflowPosition{X: 2, Y: 1}, DefaultTimeoutSeconds: 120, DefaultTemperature: floatPtr(0.7), DefaultMaxTokens: 2048},
					},
					Edges: []WorkflowEdgeView{{From: ai.NodeIntentClassify, To: ai.NodeChatReply}},
				},
				{
					Key:         ai.WorkflowScheduleCreate,
					Name:        "创建排班",
					Version:     "v1",
					Description: "加载规则、筛选候选人、候选建议、确定性兜底并保存草稿。",
					Enabled:     true,
					SortOrder:   20,
					Nodes: []workflowNodeDefinition{
						{Key: "load_rules", Name: "加载规则", Kind: "system", Description: "读取当前组织节点的生效排班规则。", Position: WorkflowPosition{X: 1, Y: 1}},
						{Key: "filter_candidates", Name: "筛选候选人", Kind: "system", Description: "按请假、规则和分组过滤候选员工。", Position: WorkflowPosition{X: 2, Y: 1}},
						{Key: "phase_zero", Name: "固定占位", Kind: "system", Description: "写入固定班次与已锁定安排。", Position: WorkflowPosition{X: 3, Y: 1}},
						{Key: "phase_one", Name: "规则占位", Kind: "system", Description: "执行必须类规则和依赖规则。", Position: WorkflowPosition{X: 4, Y: 1}},
						{Key: ai.NodeAISelect, Name: "候选建议", Kind: "llm", Description: "仅对剩余空缺给出建议分配，实际落位仍要经过候选约束、规则校验与兜底逻辑。", Configurable: true, Position: WorkflowPosition{X: 5, Y: 1}, DefaultTimeoutSeconds: 180, DefaultTemperature: floatPtr(0.3), DefaultMaxTokens: 4096},
						{Key: "phase_two", Name: "兜底填充", Kind: "system", Description: "未被采纳或未覆盖的空缺由确定性算法补齐。", Position: WorkflowPosition{X: 6, Y: 1}},
						{Key: "full_validation", Name: "全量校验", Kind: "system", Description: "对草稿进行完整规则校验。", Position: WorkflowPosition{X: 7, Y: 1}},
						{Key: "save_draft", Name: "保存草稿", Kind: "system", Description: "保存生成结果并推送通知。", Position: WorkflowPosition{X: 8, Y: 1}},
					},
					Edges: []WorkflowEdgeView{{From: "load_rules", To: "filter_candidates"}, {From: "filter_candidates", To: "phase_zero"}, {From: "phase_zero", To: "phase_one"}, {From: "phase_one", To: ai.NodeAISelect}, {From: ai.NodeAISelect, To: "phase_two"}, {From: "phase_two", To: "full_validation"}, {From: "full_validation", To: "save_draft"}},
				},
				{
					Key:         ai.WorkflowRuleParse,
					Name:        "规则解析",
					Version:     "v1",
					Description: "将自然语言规则解析为结构化规则配置。",
					Enabled:     true,
					SortOrder:   30,
					Nodes: []workflowNodeDefinition{
						{Key: ai.NodeRuleParse, Name: "单条解析", Kind: "llm", Description: "解析单条自然语言排班规则。", Configurable: true, Position: WorkflowPosition{X: 1, Y: 1}, DefaultTimeoutSeconds: 120, DefaultTemperature: floatPtr(0.1)},
						{Key: ai.NodeRuleBatchParse, Name: "批量解析", Kind: "llm", Description: "批量解析多条规则、依赖与冲突关系。", Configurable: true, Position: WorkflowPosition{X: 2, Y: 1}, DefaultTimeoutSeconds: 180, DefaultTemperature: floatPtr(0.1)},
					},
					Edges: []WorkflowEdgeView{{From: ai.NodeRuleParse, To: ai.NodeRuleBatchParse}},
				},
				{
					Key:         "schedule.adjust",
					Name:        "调整排班",
					Version:     "v1",
					Description: "加载已有草稿、应用调整、校验并保存变更。",
					Enabled:     true,
					SortOrder:   40,
					Nodes: []workflowNodeDefinition{
						{Key: "load_schedule", Name: "加载草稿", Kind: "system", Description: "读取现有排班和上下文。", Position: WorkflowPosition{X: 1, Y: 1}},
						{Key: "apply_edit", Name: "应用调整", Kind: "system", Description: "执行手动增删换班。", Position: WorkflowPosition{X: 2, Y: 1}},
						{Key: "validation", Name: "规则校验", Kind: "system", Description: "校验调整后的排班。", Position: WorkflowPosition{X: 3, Y: 1}},
						{Key: "save_change", Name: "保存变更", Kind: "system", Description: "保存调整记录和草稿。", Position: WorkflowPosition{X: 4, Y: 1}},
					},
					Edges: []WorkflowEdgeView{{From: "load_schedule", To: "apply_edit"}, {From: "apply_edit", To: "validation"}, {From: "validation", To: "save_change"}},
				},
			},
		},
	}
}

func defaultAppsByCode() map[string]appDefinition {
	defs := defaultAppDefinitions()
	result := make(map[string]appDefinition, len(defs))
	for _, app := range defs {
		result[app.Code] = app
	}
	return result
}

func (s *AppConfigService) ListApps(ctx context.Context) ([]AppConfigView, error) {
	ctx = tenant.SkipTenantGuard(ctx)

	settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	legacySettings, err := s.loadLegacySchedulingSettings(ctx)
	if err != nil {
		return nil, err
	}
	workflows, err := s.loadWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	models, err := s.loadModelConfigs(ctx)
	if err != nil {
		return nil, err
	}

	defs := defaultAppDefinitions()
	result := make([]AppConfigView, 0, len(defs))
	for _, app := range defs {
		view := AppConfigView{
			Code:        app.Code,
			Name:        app.Name,
			Description: app.Description,
			Settings:    mergeSettings(mergeSettings(app.Settings, legacySettings[app.Code]), settings[app.Code]),
			Workflows:   make([]WorkflowConfigView, 0, len(app.Workflows)),
		}
		for _, wf := range app.Workflows {
			savedWorkflow := workflows[app.Code+"/"+wf.Key]
			wfView := WorkflowConfigView{
				Key:         wf.Key,
				Name:        firstNonEmpty(savedWorkflow.Name, wf.Name),
				Version:     firstNonEmpty(savedWorkflow.Version, wf.Version),
				Description: firstNonEmpty(savedWorkflow.Description, wf.Description),
				Enabled:     wf.Enabled,
				Nodes:       make([]WorkflowNodeView, 0, len(wf.Nodes)),
				Edges:       wf.Edges,
			}
			if savedWorkflow.ID != "" {
				wfView.Enabled = savedWorkflow.Enabled
			}
			for _, node := range wf.Nodes {
				modelKey := app.Code + "/" + wf.Key + "/" + node.Key
				wfView.Nodes = append(wfView.Nodes, WorkflowNodeView{
					Key:          node.Key,
					Name:         node.Name,
					Kind:         node.Kind,
					Description:  node.Description,
					Configurable: node.Configurable,
					Position:     node.Position,
					ModelConfig:  modelViewFromDefinition(node, models[modelKey]),
				})
			}
			view.Workflows = append(view.Workflows, wfView)
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *AppConfigService) loadLegacySchedulingSettings(ctx context.Context) (map[string]map[string]string, error) {
	var rows []SystemConfig
	if err := s.db.WithContext(ctx).Where("`key` IN ?", legacySchedulingSettingKeys).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]map[string]string{}, nil
	}
	result := map[string]map[string]string{
		ai.AppScheduling: {},
	}
	for _, row := range rows {
		result[ai.AppScheduling][row.Key] = row.Value
	}
	return result, nil
}

func (s *AppConfigService) loadSettings(ctx context.Context) (map[string]map[string]string, error) {
	var rows []AppConfigEntry
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]map[string]string)
	for _, row := range rows {
		if result[row.AppCode] == nil {
			result[row.AppCode] = make(map[string]string)
		}
		result[row.AppCode][row.Key] = row.Value
	}
	return result, nil
}

func (s *AppConfigService) loadWorkflows(ctx context.Context) (map[string]AppWorkflowConfig, error) {
	var rows []AppWorkflowConfig
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]AppWorkflowConfig, len(rows))
	for _, row := range rows {
		result[row.AppCode+"/"+row.WorkflowKey] = row
	}
	return result, nil
}

func (s *AppConfigService) loadModelConfigs(ctx context.Context) (map[string]AIModelConfig, error) {
	var rows []AIModelConfig
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]AIModelConfig, len(rows))
	for _, row := range rows {
		result[row.AppCode+"/"+row.WorkflowKey+"/"+row.NodeKey] = row
	}
	return result, nil
}

func mergeSettings(defaults map[string]string, saved map[string]string) map[string]string {
	result := make(map[string]string, len(defaults)+len(saved))
	for key, value := range defaults {
		result[key] = value
	}
	for key, value := range saved {
		result[key] = value
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func modelViewFromDefinition(node workflowNodeDefinition, saved AIModelConfig) AIModelConfigView {
	view := AIModelConfigView{
		Provider:       node.DefaultProvider,
		Model:          node.DefaultModel,
		TimeoutSeconds: node.DefaultTimeoutSeconds,
		Temperature:    node.DefaultTemperature,
		MaxTokens:      node.DefaultMaxTokens,
		Enabled:        node.Configurable,
	}
	if view.TimeoutSeconds == 0 && node.Configurable {
		view.TimeoutSeconds = 60
	}
	if saved.ID == "" {
		return view
	}
	view.Provider = saved.Provider
	view.Model = saved.Model
	view.TimeoutSeconds = saved.TimeoutSeconds
	view.Temperature = saved.Temperature
	view.MaxTokens = saved.MaxTokens
	view.Enabled = saved.Enabled
	return view
}

func (s *AppConfigService) SaveSettings(ctx context.Context, appCode string, settings map[string]string) (map[string]string, error) {
	ctx = tenant.SkipTenantGuard(ctx)
	if err := s.validateApp(appCode); err != nil {
		return nil, err
	}
	for key, value := range settings {
		var row AppConfigEntry
		err := s.db.WithContext(ctx).Where("app_code = ? AND `key` = ?", appCode, key).First(&row).Error
		if err == gorm.ErrRecordNotFound {
			row = AppConfigEntry{ID: uuid.New().String(), AppCode: appCode, Key: key, Value: value}
			if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			if err := s.db.WithContext(ctx).Model(&row).Update("value", value).Error; err != nil {
				return nil, err
			}
		}
	}
	if appCode == ai.AppScheduling {
		if err := s.clearLegacySchedulingSettings(ctx); err != nil {
			return nil, err
		}
	}
	return settings, nil
}

func (s *AppConfigService) clearLegacySchedulingSettings(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("`key` IN ?", legacySchedulingSettingKeys).Delete(&SystemConfig{}).Error
}

func (s *AppConfigService) SaveWorkflow(ctx context.Context, appCode, workflowKey string, input WorkflowConfigView) (*WorkflowConfigView, error) {
	ctx = tenant.SkipTenantGuard(ctx)
	wf, err := s.findWorkflow(appCode, workflowKey)
	if err != nil {
		return nil, err
	}
	name := firstNonEmpty(input.Name, wf.Name)
	version := firstNonEmpty(input.Version, wf.Version)
	description := firstNonEmpty(input.Description, wf.Description)

	var row AppWorkflowConfig
	dbErr := s.db.WithContext(ctx).Where("app_code = ? AND workflow_key = ?", appCode, workflowKey).First(&row).Error
	if dbErr == gorm.ErrRecordNotFound {
		row = AppWorkflowConfig{ID: uuid.New().String(), AppCode: appCode, WorkflowKey: workflowKey, Name: name, Version: version, Description: description, Enabled: input.Enabled, SortOrder: wf.SortOrder}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if dbErr != nil {
		return nil, dbErr
	} else {
		row.Name = name
		row.Version = version
		row.Description = description
		row.Enabled = input.Enabled
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	}
	return &WorkflowConfigView{Key: workflowKey, Name: row.Name, Version: row.Version, Description: row.Description, Enabled: row.Enabled}, nil
}

func (s *AppConfigService) SaveNodeModel(ctx context.Context, appCode, workflowKey, nodeKey string, input AIModelConfigView) (*AIModelConfigView, error) {
	ctx = tenant.SkipTenantGuard(ctx)
	node, err := s.findNode(appCode, workflowKey, nodeKey)
	if err != nil {
		return nil, err
	}
	if !node.Configurable {
		return nil, gorm.ErrRecordNotFound
	}
	if input.TimeoutSeconds <= 0 {
		input.TimeoutSeconds = node.DefaultTimeoutSeconds
	}
	if input.TimeoutSeconds <= 0 {
		input.TimeoutSeconds = 60
	}

	var row AIModelConfig
	dbErr := s.db.WithContext(ctx).Where("app_code = ? AND workflow_key = ? AND node_key = ?", appCode, workflowKey, nodeKey).First(&row).Error
	if dbErr == gorm.ErrRecordNotFound {
		row = AIModelConfig{ID: uuid.New().String(), AppCode: appCode, WorkflowKey: workflowKey, NodeKey: nodeKey}
	} else if dbErr != nil {
		return nil, dbErr
	}
	row.Provider = input.Provider
	row.Model = input.Model
	row.TimeoutSeconds = input.TimeoutSeconds
	row.Temperature = input.Temperature
	row.MaxTokens = input.MaxTokens
	row.Enabled = input.Enabled

	if dbErr == gorm.ErrRecordNotFound {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}

	view := modelViewFromDefinition(node, row)
	return &view, nil
}

func (s *AppConfigService) ResolveNodeModel(ctx context.Context, appCode, workflowKey, nodeKey string) (*ai.NodeModelConfig, error) {
	ctx = tenant.SkipTenantGuard(ctx)
	var row AIModelConfig
	if err := s.db.WithContext(ctx).Where("app_code = ? AND workflow_key = ? AND node_key = ?", appCode, workflowKey, nodeKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ai.NodeModelConfig{
		AppCode:     row.AppCode,
		WorkflowKey: row.WorkflowKey,
		NodeKey:     row.NodeKey,
		Provider:    row.Provider,
		Model:       row.Model,
		Timeout:     time.Duration(row.TimeoutSeconds) * time.Second,
		Temperature: row.Temperature,
		MaxTokens:   row.MaxTokens,
		Enabled:     row.Enabled,
	}, nil
}

func (s *AppConfigService) validateApp(appCode string) error {
	if _, ok := defaultAppsByCode()[appCode]; !ok {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *AppConfigService) findWorkflow(appCode, workflowKey string) (workflowDefinition, error) {
	apps := defaultAppsByCode()
	app, ok := apps[appCode]
	if !ok {
		return workflowDefinition{}, gorm.ErrRecordNotFound
	}
	for _, wf := range app.Workflows {
		if wf.Key == workflowKey {
			return wf, nil
		}
	}
	return workflowDefinition{}, gorm.ErrRecordNotFound
}

func (s *AppConfigService) findNode(appCode, workflowKey, nodeKey string) (workflowNodeDefinition, error) {
	wf, err := s.findWorkflow(appCode, workflowKey)
	if err != nil {
		return workflowNodeDefinition{}, err
	}
	for _, node := range wf.Nodes {
		if node.Key == nodeKey {
			return node, nil
		}
	}
	return workflowNodeDefinition{}, gorm.ErrRecordNotFound
}

// ListApps GET /api/v1/admin/app-config/apps
func (h *AppConfigHandler) ListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.svc.ListApps(r.Context())
	if err != nil {
		response.InternalError(w, "获取应用配置失败")
		return
	}
	response.OK(w, apps)
}

type updateAppSettingsInput struct {
	Settings map[string]string `json:"settings"`
}

// UpdateSettings PUT /api/v1/admin/app-config/apps/{appCode}/settings
func (h *AppConfigHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	appCode := chi.URLParam(r, "appCode")
	var input updateAppSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	settings, err := h.svc.SaveSettings(r.Context(), appCode, input.Settings)
	if err != nil {
		handleAppConfigError(w, err)
		return
	}
	response.OK(w, settings)
}

// UpdateWorkflow PUT /api/v1/admin/app-config/apps/{appCode}/workflows/{workflowKey}
func (h *AppConfigHandler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	appCode := chi.URLParam(r, "appCode")
	workflowKey := chi.URLParam(r, "workflowKey")
	var input WorkflowConfigView
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	wf, err := h.svc.SaveWorkflow(r.Context(), appCode, workflowKey, input)
	if err != nil {
		handleAppConfigError(w, err)
		return
	}
	response.OK(w, wf)
}

// UpdateNodeModel PUT /api/v1/admin/app-config/apps/{appCode}/workflows/{workflowKey}/nodes/{nodeKey}/model
func (h *AppConfigHandler) UpdateNodeModel(w http.ResponseWriter, r *http.Request) {
	appCode := chi.URLParam(r, "appCode")
	workflowKey := chi.URLParam(r, "workflowKey")
	nodeKey := chi.URLParam(r, "nodeKey")
	var input AIModelConfigView
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "请求参数格式错误")
		return
	}
	model, err := h.svc.SaveNodeModel(r.Context(), appCode, workflowKey, nodeKey, input)
	if err != nil {
		handleAppConfigError(w, err)
		return
	}
	response.OK(w, model)
}

func handleAppConfigError(w http.ResponseWriter, err error) {
	if err == gorm.ErrRecordNotFound {
		response.NotFound(w, "应用、工作流或节点不存在")
		return
	}
	response.InternalError(w, "保存应用配置失败")
}
