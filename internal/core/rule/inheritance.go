package rule

import (
	"context"
	"fmt"
)

// EffectiveRuleSet 某节点的生效规则集。
type EffectiveRuleSet struct {
	Rules     []Rule
	SourceMap map[string]string
}

// ComputeEffectiveRules 计算某节点的生效规则集。
// 纯科室侧模型下，仅返回当前节点启用中的本级规则。
func (s *Service) ComputeEffectiveRules(ctx context.Context, nodeID string) (*EffectiveRuleSet, error) {
	rules, err := s.repo.ListByNodeID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("查询规则列表失败: %w", err)
	}

	sourceMap := make(map[string]string, len(rules))
	for _, r := range rules {
		sourceMap[r.ID] = "本级"
	}

	return &EffectiveRuleSet{Rules: rules, SourceMap: sourceMap}, nil
}
