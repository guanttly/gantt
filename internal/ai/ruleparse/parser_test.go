package ruleparse

import (
	"context"
	"strings"
	"testing"

	"gantt-saas/internal/ai"

	"go.uber.org/zap"
)

type fakeProvider struct {
	content     string
	lastRequest ai.ChatRequest
}

func (p *fakeProvider) Chat(_ context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	p.lastRequest = req
	return &ai.ChatResponse{Content: p.content}, nil
}

func (p *fakeProvider) ChatStream(_ context.Context, _ ai.ChatRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil
}

func (p *fakeProvider) Name() string { return "fake" }

func TestParseBatchFromContent_NormalizesParsedRulesAlias(t *testing.T) {
	parser := NewParser(&fakeProvider{}, zap.NewNop())
	content := strings.TrimSpace(`
解析如下：
{
	"parsed_rules": [{
		"name": "Night Source",
		"type": "source",
		"category": "dependency",
		"sub_type": "source",
		"config": {"type":"staff_source","target_shift_id":"night","source_shift_id":"day"},
		"description": "night depends on day",
		"subject_shifts": ["night"],
		"object_shifts": ["day"]
	}],
	"dependencies": [{
		"dependent_rule_name": "Night Source",
		"dependent_on_rule_name": "Base Rule",
		"dependency_type": "source"
	}],
	"conflicts": [{
		"rule_name_1": "Night Source",
		"rule_name_2": "Other Rule",
		"conflict_type": "exclusive"
	}],
	"reasoning": "done"
}
谢谢。
`)

	result, err := parser.ParseBatchFromContent(content)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("len(result.Rules) = %d, want 1", len(result.Rules))
	}
	if len(result.ParsedRules) != 1 {
		t.Fatalf("len(result.ParsedRules) = %d, want 1", len(result.ParsedRules))
	}
	if result.Rules[0].RuleType != "source" {
		t.Fatalf("result.Rules[0].RuleType = %q, want %q", result.Rules[0].RuleType, "source")
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].DependentOnRuleName != "Base Rule" {
		t.Fatalf("dependencies = %+v, want one dependency on Base Rule", result.Dependencies)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].RuleName2 != "Other Rule" {
		t.Fatalf("conflicts = %+v, want one conflict with Other Rule", result.Conflicts)
	}
	if result.Reasoning != "done" {
		t.Fatalf("reasoning = %q, want %q", result.Reasoning, "done")
	}
}

func TestParseBatchFromContent_SkipsIncompleteRules(t *testing.T) {
	parser := NewParser(&fakeProvider{}, zap.NewNop())
	content := `{
		"rules": [
			{"name":"complete","rule_type":"preferred","category":"preference","sub_type":"prefer","config":{},"description":"ok"},
			{"name":"missing-category","sub_type":"prefer","config":{},"description":"bad"}
		],
		"reasoning": "filtered"
	}`

	result, err := parser.ParseBatchFromContent(content)
	if err != nil {
		t.Fatalf("ParseBatchFromContent() error = %v", err)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("len(result.Rules) = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].Name != "complete" {
		t.Fatalf("result.Rules[0].Name = %q, want %q", result.Rules[0].Name, "complete")
	}
}

func TestParseBatch_ExtractsJSONFromWrappedProviderOutput(t *testing.T) {
	provider := &fakeProvider{content: "prefix\n{\"rules\":[{\"name\":\"Night Prefer\",\"rule_type\":\"preferred\",\"category\":\"preference\",\"sub_type\":\"prefer\",\"config\":{},\"description\":\"prefer night\"}],\"reasoning\":\"ok\"}\nsuffix"}
	parser := NewParser(provider, zap.NewNop())

	result, err := parser.ParseBatch(context.Background(), "prefer night")
	if err != nil {
		t.Fatalf("ParseBatch() error = %v", err)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("len(result.Rules) = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].Name != "Night Prefer" {
		t.Fatalf("result.Rules[0].Name = %q, want %q", result.Rules[0].Name, "Night Prefer")
	}
	if result.Rules[0].RuleType != "preferred" {
		t.Fatalf("result.Rules[0].RuleType = %q, want %q", result.Rules[0].RuleType, "preferred")
	}
}

func TestParseBatch_IncludesShiftCatalogInPrompt(t *testing.T) {
	provider := &fakeProvider{content: `{"rules":[{"name":"Night Prefer","rule_type":"preferred","category":"preference","sub_type":"prefer","config":{},"description":"prefer night"}],"reasoning":"ok"}`}
	parser := NewParser(provider, zap.NewNop())

	_, err := parser.ParseBatch(context.Background(), "优先安排夜班", ParseOptions{ShiftCatalog: []ShiftCatalogItem{{Code: "NIGHT", Name: "夜班", Aliases: []string{"夜", "晚夜"}}}})
	if err != nil {
		t.Fatalf("ParseBatch() error = %v", err)
	}
	if len(provider.lastRequest.Messages) == 0 {
		t.Fatal("provider.lastRequest.Messages is empty")
	}
	systemPrompt := provider.lastRequest.Messages[0].Content
	if !strings.Contains(systemPrompt, "code=NIGHT") {
		t.Fatalf("system prompt = %q, want shift code", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "名称=夜班") {
		t.Fatalf("system prompt = %q, want shift name", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "别名=夜 / 晚夜") {
		t.Fatalf("system prompt = %q, want shift aliases", systemPrompt)
	}
}
