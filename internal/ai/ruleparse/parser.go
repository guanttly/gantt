package ruleparse

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gantt-saas/internal/ai"

	"go.uber.org/zap"
)

const ruleParseSystemPrompt = `你是排班规则解析专家。将用户自然语言规则解析为结构化 JSON。

你必须尽量补齐以下维度：
- rule_type: exclusive/combinable/required_together/periodic/maxCount/forbidden_day/preferred/source/order/min_rest
- category: constraint/preference/dependency
- sub_type: forbid/limit/must/prefer/combinable/source/order/min_rest
- apply_scope: global/specific
- time_scope: same_day/same_week/same_month/custom
- time_offset_days: 跨日关系偏移，前一天=-1，后一天=1
- rule_data: 规则语义化说明
- priority: 默认可给 100
- source_type: 对 AI 解析结果固定输出 llm_parsed
- version: 固定输出 v4
- subject_shifts/object_shifts/target_shifts: 按班次角色拆分
- scope_type: all/employee/group/exclude_employee/exclude_group
- scope_employees/scope_groups: 适用对象名称
- config: 保留兼容当前系统的约束配置

输出要求：
- 所有说明性文本必须使用简体中文，包括 description、rule_data、reasoning、dependency description、conflict description。
- 枚举值和字段名保持约定英文，不要翻译成中文。
- 严禁输出 UUID、数据库主键或看起来像随机 ID 的字符串作为班次/员工/分组标识。
- 如果上文提供了现有班次清单，subject_shifts/object_shifts/target_shifts 必须优先使用班次短代号或英文 code，不要使用 UUID。
- 不要编造不存在的班次 code；如果无法从给定清单中确认映射，相关数组留空，并在 reasoning 中用中文说明未确认的原始提法。
- 如果没有依赖或冲突，直接返回空数组 []，不要写 Empty、None、Not applicable、Using... 等英文说明。
- 不要在 JSON 中添加注释、括号补充说明、Markdown 列表或任何 JSON 之外的文本。

返回 JSON，不要输出任何额外文本。`

const ruleParseBatchSystemPrompt = `你是排班规则解析专家。用户会提供一段包含一条或多条排班规则的自然语言描述，你需要将每条规则分别解析为结构化 JSON。

必须输出完整规则维度，而不是只输出 name/category/sub_type/config。

语言要求：
- 所有说明性文本必须使用简体中文，包括 description、rule_data、reasoning、dependencies[].description、conflicts[].description。
- 字段名和枚举值保持约定英文，不要翻译。
- 严禁输出 UUID、数据库主键或看起来像随机 ID 的字符串作为班次/员工/分组标识。
- 如果上文提供了现有班次清单，subject_shifts/object_shifts/target_shifts 必须优先使用班次短代号或英文 code，不要使用 UUID。
- 不要编造不存在的班次 code；如果无法从给定清单中确认映射，相关数组留空，并在 reasoning 中用中文说明未确认的原始提法。
- 如果某个数组为空，直接输出 []，不要写 Empty、None、Not applicable、Using... 等英文说明。
- 不要输出 JSON 以外的任何分析文字、Markdown、注释或括号说明。

严格返回如下 JSON：
{
	"rules": [{
		"name": "规则名称",
		"rule_type": "exclusive|maxCount|required_together|preferred|source|order|min_rest|...",
		"category": "constraint|preference|dependency",
		"sub_type": "forbid|limit|must|prefer|combinable|source|order|min_rest",
		"apply_scope": "global|specific",
		"time_scope": "same_day|same_week|same_month|custom",
		"time_offset_days": -1,
		"rule_data": "语义化描述",
		"priority": 100,
		"source_type": "llm_parsed",
		"version": "v4",
		"description": "规则描述",
		"config": {},
		"subject_shifts": ["班次A"],
		"object_shifts": ["班次B"],
		"target_shifts": ["班次C"],
		"scope_type": "all|employee|group|exclude_employee|exclude_group",
		"scope_employees": ["员工名"],
		"scope_groups": ["分组名"]
	}],
	"dependencies": [{
		"dependent_rule_name": "规则A",
		"dependent_on_rule_name": "规则B",
		"dependency_type": "time|source|resource|order",
		"description": "依赖说明"
	}],
	"conflicts": [{
		"rule_name_1": "规则A",
		"rule_name_2": "规则B",
		"conflict_type": "exclusive|resource|time|frequency|duplicate",
		"description": "冲突说明"
	}],
	"reasoning": "简要说明解析思路"
}`

// RuleConfig 解析后的规则配置。
type RuleConfig struct {
	Name            string          `json:"name"`
	RuleType        string          `json:"rule_type,omitempty"`
	Type            string          `json:"type,omitempty"`
	Category        string          `json:"category"`
	SubType         string          `json:"sub_type"`
	ApplyScope      string          `json:"apply_scope,omitempty"`
	TimeScope       string          `json:"time_scope,omitempty"`
	TimeOffsetDays  *int            `json:"time_offset_days,omitempty"`
	RuleData        string          `json:"rule_data,omitempty"`
	Priority        int             `json:"priority,omitempty"`
	SourceType      string          `json:"source_type,omitempty"`
	ParseConfidence *float64        `json:"parse_confidence,omitempty"`
	Version         string          `json:"version,omitempty"`
	Config          json.RawMessage `json:"config"`
	Description     string          `json:"description"`
	SubjectShifts   []string        `json:"subject_shifts,omitempty"`
	ObjectShifts    []string        `json:"object_shifts,omitempty"`
	TargetShifts    []string        `json:"target_shifts,omitempty"`
	ScopeType       string          `json:"scope_type,omitempty"`
	ScopeEmployees  []string        `json:"scope_employees,omitempty"`
	ScopeGroups     []string        `json:"scope_groups,omitempty"`
}

type RuleDependencyInfo struct {
	DependentRuleName   string `json:"dependent_rule_name"`
	DependentOnRuleName string `json:"dependent_on_rule_name"`
	DependencyType      string `json:"dependency_type"`
	Description         string `json:"description,omitempty"`
}

type RuleConflictInfo struct {
	RuleName1    string `json:"rule_name_1"`
	RuleName2    string `json:"rule_name_2"`
	ConflictType string `json:"conflict_type"`
	Description  string `json:"description,omitempty"`
}

type ShiftCatalogItem struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
}

type ParseOptions struct {
	ShiftCatalog []ShiftCatalogItem `json:"shift_catalog,omitempty"`
}

// ParseBatchResult 批量解析结果。
type ParseBatchResult struct {
	Rules        []RuleConfig         `json:"rules,omitempty"`
	ParsedRules  []RuleConfig         `json:"parsed_rules,omitempty"`
	Dependencies []RuleDependencyInfo `json:"dependencies,omitempty"`
	Conflicts    []RuleConflictInfo   `json:"conflicts,omitempty"`
	Reasoning    string               `json:"reasoning"`
}

// Parser 规则解析器。
type Parser struct {
	provider ai.Provider
	logger   *zap.Logger
}

// NewParser 创建规则解析器。
func NewParser(provider ai.Provider, logger *zap.Logger) *Parser {
	return &Parser{provider: provider, logger: logger.Named("ruleparse")}
}

// Parse 将自然语言描述转为规则配置 JSON。
func (p *Parser) Parse(ctx context.Context, description string, opts ...ParseOptions) (*RuleConfig, error) {
	options := firstParseOptions(opts)
	resp, err := p.provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: buildPrompt(ruleParseSystemPrompt, options)},
			{Role: "user", Content: description},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("rule parse failed: %w", err)
	}

	cfg := &RuleConfig{}
	content := strings.TrimSpace(resp.Content)
	if idx := strings.Index(content, "{"); idx != -1 {
		if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
			content = content[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(content), cfg); err != nil {
		p.logger.Warn("rule parse JSON failed", zap.Error(err), zap.String("content", resp.Content))
		return nil, fmt.Errorf("rule parse: invalid JSON response: %w", err)
	}

	if cfg.Name == "" || cfg.Category == "" || cfg.SubType == "" {
		return nil, fmt.Errorf("rule parse: incomplete result (name=%s, category=%s, sub_type=%s)", cfg.Name, cfg.Category, cfg.SubType)
	}
	if cfg.RuleType == "" {
		cfg.RuleType = cfg.Type
	}

	p.logger.Info("rule parsed", zap.String("name", cfg.Name), zap.String("category", cfg.Category), zap.String("sub_type", cfg.SubType))
	return cfg, nil
}

// ParseBatch 将包含多条规则的自然语言描述批量转为规则配置。
func (p *Parser) ParseBatch(ctx context.Context, description string, opts ...ParseOptions) (*ParseBatchResult, error) {
	options := firstParseOptions(opts)
	resp, err := p.provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: buildPrompt(ruleParseBatchSystemPrompt, options)},
			{Role: "user", Content: description},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("rule batch parse failed: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	if idx := strings.Index(content, "{"); idx != -1 {
		if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
			content = content[idx : endIdx+1]
		}
	}

	result := &ParseBatchResult{}
	if err := json.Unmarshal([]byte(content), result); err != nil {
		p.logger.Warn("rule batch parse JSON failed", zap.Error(err), zap.String("content", resp.Content))
		return nil, fmt.Errorf("rule batch parse: invalid JSON response: %w", err)
	}

	result.normalize()
	if len(result.Rules) == 0 {
		return nil, fmt.Errorf("rule batch parse: no rules extracted")
	}

	// 校验每条规则的必填字段
	valid := make([]RuleConfig, 0, len(result.Rules))
	for _, r := range result.Rules {
		if r.Name == "" || r.Category == "" || r.SubType == "" {
			p.logger.Warn("rule batch parse: skipping incomplete rule", zap.String("name", r.Name))
			continue
		}
		if r.RuleType == "" {
			r.RuleType = r.Type
		}
		valid = append(valid, r)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("rule batch parse: all rules incomplete")
	}
	result.Rules = valid

	p.logger.Info("rules batch parsed", zap.Int("count", len(result.Rules)))
	return result, nil
}

// ParseBatchStream 流式调用 LLM 解析多条规则，返回 StreamChunk channel。
// 调用方负责收集完整文本后调用 ParseBatchFromContent 解析 JSON。
func (p *Parser) ParseBatchStream(ctx context.Context, description string, opts ...ParseOptions) (<-chan ai.StreamChunk, error) {
	options := firstParseOptions(opts)
	return p.provider.ChatStream(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: buildPrompt(ruleParseBatchSystemPrompt, options)},
			{Role: "user", Content: description},
		},
		Temperature: 0.1,
	})
}

// ParseBatchFromContent 从 LLM 完整输出文本中解析批量规则结果。
func (p *Parser) ParseBatchFromContent(content string) (*ParseBatchResult, error) {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "{"); idx != -1 {
		if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
			content = content[idx : endIdx+1]
		}
	}

	result := &ParseBatchResult{}
	if err := json.Unmarshal([]byte(content), result); err != nil {
		p.logger.Warn("rule batch parse JSON failed", zap.Error(err), zap.String("content", content))
		return nil, fmt.Errorf("rule batch parse: invalid JSON response: %w", err)
	}

	result.normalize()
	if len(result.Rules) == 0 {
		return nil, fmt.Errorf("rule batch parse: no rules extracted")
	}

	valid := make([]RuleConfig, 0, len(result.Rules))
	for _, r := range result.Rules {
		if r.Name == "" || r.Category == "" || r.SubType == "" {
			p.logger.Warn("rule batch parse: skipping incomplete rule", zap.String("name", r.Name))
			continue
		}
		if r.RuleType == "" {
			r.RuleType = r.Type
		}
		valid = append(valid, r)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("rule batch parse: all rules incomplete")
	}
	result.Rules = valid

	p.logger.Info("rules batch parsed", zap.Int("count", len(result.Rules)))
	return result, nil
}

func (r *ParseBatchResult) normalize() {
	if len(r.Rules) == 0 && len(r.ParsedRules) > 0 {
		r.Rules = r.ParsedRules
	}
	if len(r.ParsedRules) == 0 && len(r.Rules) > 0 {
		r.ParsedRules = r.Rules
	}
}

func firstParseOptions(opts []ParseOptions) ParseOptions {
	if len(opts) == 0 {
		return ParseOptions{}
	}
	return opts[0]
}

func buildPrompt(base string, options ParseOptions) string {
	shiftCatalogPrompt := buildShiftCatalogPrompt(options.ShiftCatalog)
	if shiftCatalogPrompt == "" {
		return base
	}
	return base + "\n\n" + shiftCatalogPrompt
}

func buildShiftCatalogPrompt(items []ShiftCatalogItem) string {
	normalized := normalizeShiftCatalog(items)
	if len(normalized) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("现有班次清单如下。涉及班次时，只能使用下列 code，严禁编造新 code：\n")
	for _, item := range normalized {
		builder.WriteString("- code=")
		builder.WriteString(item.Code)
		builder.WriteString("，名称=")
		builder.WriteString(item.Name)
		if len(item.Aliases) > 0 {
			builder.WriteString("，别名=")
			builder.WriteString(strings.Join(item.Aliases, " / "))
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func normalizeShiftCatalog(items []ShiftCatalogItem) []ShiftCatalogItem {
	seen := make(map[string]struct{})
	normalized := make([]ShiftCatalogItem, 0, len(items))
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		name := strings.TrimSpace(item.Name)
		if code == "" || name == "" {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		aliases := make([]string, 0, len(item.Aliases))
		aliasSeen := make(map[string]struct{})
		for _, alias := range item.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if _, exists := aliasSeen[alias]; exists {
				continue
			}
			aliasSeen[alias] = struct{}{}
			aliases = append(aliases, alias)
		}
		normalized = append(normalized, ShiftCatalogItem{Code: code, Name: name, Aliases: aliases})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Code < normalized[j].Code
	})
	return normalized
}
