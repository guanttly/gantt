package rule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// EffectiveRuleSet 某节点的生效规则集。
type EffectiveRuleSet struct {
	Rules     []Rule            // 合并后的生效规则
	SourceMap map[string]string // ruleID → 来源节点名称（用于 UI 展示）
}

// ComputeEffectiveRules 计算某节点的生效规则集。
// 沿组织树路径从根到叶，逐层合并规则；下级覆盖上级同类型规则。
func (s *Service) ComputeEffectiveRules(ctx context.Context, nodeID string) (*EffectiveRuleSet, error) {
	// 1. 查询当前节点信息
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("查询组织节点失败: %w", err)
	}

	// 2. 提取祖先节点 ID 列表（按 path 分割，从根到叶）
	ancestorIDs := extractAncestorIDs(node.Path)

	// 3. 批量查询所有祖先节点的规则
	allRules, err := s.repo.ListByNodeIDs(ctx, ancestorIDs)
	if err != nil {
		return nil, fmt.Errorf("查询规则列表失败: %w", err)
	}

	// 4. 构建节点 ID → 深度的映射（用于确定覆盖优先级）
	depthMap := make(map[string]int)
	for i, id := range ancestorIDs {
		depthMap[id] = i
	}

	// 5. 构建节点 ID → 名称的映射（用于来源标记）
	nodeNameMap := make(map[string]string)
	for _, id := range ancestorIDs {
		n, err := s.nodeRepo.GetByID(ctx, id)
		if err == nil {
			nodeNameMap[id] = n.Name
		}
	}

	// 6. 合并规则：同组内下级覆盖上级
	merged := mergeRules(allRules, depthMap)

	// 7. 构建来源映射
	sourceMap := make(map[string]string)
	for _, r := range merged {
		if name, ok := nodeNameMap[r.OrgNodeID]; ok {
			sourceMap[r.ID] = name
		}
	}

	return &EffectiveRuleSet{Rules: merged, SourceMap: sourceMap}, nil
}

// extractAncestorIDs 从物化路径提取祖先节点 ID 列表。
// 路径格式: "/rootID/parentID/currentID/" → ["rootID", "parentID", "currentID"]
func extractAncestorIDs(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ruleGroupKey 用于规则分组的键。
type ruleGroupKey struct {
	Category string
	SubType  string
	ConfigID string // 从 config 中提取的关键标识（如 shift_id 组合）
}

// mergeRules 合并规则：
// 1. 按 (category, sub_type, config关联标识) 分组
// 2. 同组内 override_rule_id 不为空的直接覆盖对应上级规则
// 3. 同组内保留 depth 最大的（最下级节点的规则）
func mergeRules(rules []Rule, depthMap map[string]int) []Rule {
	// 按 ID 索引规则（用于 override 查找）
	ruleByID := make(map[string]*Rule)
	for i := range rules {
		ruleByID[rules[i].ID] = &rules[i]
	}

	// 收集被覆盖的规则 ID
	overriddenIDs := make(map[string]bool)
	for _, r := range rules {
		if r.OverrideRuleID != nil && *r.OverrideRuleID != "" {
			overriddenIDs[*r.OverrideRuleID] = true
		}
	}

	// 按分组键聚合：同 category+sub_type 且无显式覆盖关系的规则
	groups := make(map[ruleGroupKey][]Rule)
	var standalone []Rule

	for _, r := range rules {
		// 如果已被覆盖，跳过
		if overriddenIDs[r.ID] {
			continue
		}
		// 如果是覆盖规则，直接保留
		if r.OverrideRuleID != nil && *r.OverrideRuleID != "" {
			standalone = append(standalone, r)
			continue
		}

		key := ruleGroupKey{
			Category: r.Category,
			SubType:  r.SubType,
			ConfigID: extractConfigID(r),
		}
		groups[key] = append(groups[key], r)
	}

	// 每组保留 depth 最大的规则
	result := make([]Rule, 0, len(groups)+len(standalone))
	for _, groupRules := range groups {
		best := groupRules[0]
		bestDepth := depthMap[best.OrgNodeID]
		for _, r := range groupRules[1:] {
			d := depthMap[r.OrgNodeID]
			if d > bestDepth {
				best = r
				bestDepth = d
			}
		}
		result = append(result, best)
	}

	// 加入显式覆盖的规则
	result = append(result, standalone...)

	return result
}

// extractConfigID 从规则 config 中提取关键标识用于分组。
// 不同子类型有不同的分组策略。
func extractConfigID(r Rule) string {
	var cfg map[string]interface{}
	if err := parseJSON(r.Config, &cfg); err != nil {
		return ""
	}

	switch r.SubType {
	case SubTypeForbid:
		// 排他班次按 shift_ids 排序后拼接
		if ids, ok := cfg["shift_ids"]; ok {
			return fmt.Sprintf("%v", ids)
		}
	case SubTypeLimit:
		if id, ok := cfg["shift_id"]; ok {
			return fmt.Sprintf("%v", id)
		}
	case SubTypePrefer:
		empID, _ := cfg["employee_id"]
		shiftID, _ := cfg["shift_id"]
		return fmt.Sprintf("%v_%v", empID, shiftID)
	case SubTypeMust:
		if ids, ok := cfg["employee_ids"]; ok {
			return fmt.Sprintf("%v", ids)
		}
	case SubTypeSource:
		target, _ := cfg["target_shift_id"]
		return fmt.Sprintf("%v", target)
	case SubTypeOrder:
		before, _ := cfg["before_shift_id"]
		after, _ := cfg["after_shift_id"]
		return fmt.Sprintf("%v_%v", before, after)
	case SubTypeMinRest:
		return "min_rest" // 全局唯一
	}

	return ""
}

// parseJSON 辅助函数：解析 JSON。
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
