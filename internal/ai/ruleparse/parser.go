package ruleparse

import (
"context"
"encoding/json"
"fmt"
"strings"

"gantt-saas/internal/ai"

"go.uber.org/zap"
)

const ruleParseSystemPrompt = `你是排班规则配置助手。将用户的自然语言规则描述转为标准JSON格式。

支持的规则类型:
- forbid: 排他/禁止规则, category=constraint, config示例: {"type":"exclusive_shifts","shift_ids":["A","B"],"scope":"same_day"}
- limit: 数量限制, category=constraint, config示例: {"type":"max_count","shift_id":"A","max":5,"period":"week"}
- min_rest: 最小休息, category=constraint, config示例: {"type":"min_rest","days":2}
- must: 固定排班, category=constraint, config示例: {"type":"required_together","employee_ids":["e1"],"shift_id":"A"}
- prefer: 偏好权重, category=preference, config示例: {"type":"prefer_employee","employee_id":"e1","shift_id":"A","weight":10}
- source: 人员来源, category=dependency, config示例: {"type":"staff_source","target_shift_id":"B","source_shift_id":"A"}
- order: 执行顺序, category=dependency, config示例: {"type":"execution_order","before_shift_id":"A","after_shift_id":"B"}

返回JSON: {"name":"规则名称","category":"...","sub_type":"...","config":{...},"description":"规则描述"}`

// RuleConfig 解析后的规则配置。
type RuleConfig struct {
Name        string          `json:"name"`
Category    string          `json:"category"`
SubType     string          `json:"sub_type"`
Config      json.RawMessage `json:"config"`
Description string          `json:"description"`
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
func (p *Parser) Parse(ctx context.Context, description string) (*RuleConfig, error) {
resp, err := p.provider.Chat(ctx, ai.ChatRequest{
Messages: []ai.Message{
{Role: "system", Content: ruleParseSystemPrompt},
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

p.logger.Info("rule parsed", zap.String("name", cfg.Name), zap.String("category", cfg.Category), zap.String("sub_type", cfg.SubType))
return cfg, nil
}
