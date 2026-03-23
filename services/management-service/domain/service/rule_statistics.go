package service

import (
	"context"
)

// IRuleStatisticsService 规则统计服务接口
type IRuleStatisticsService interface {
	// GetRuleStatistics 获取规则统计信息
	GetRuleStatistics(ctx context.Context, orgID string) (*RuleStatistics, error)
}

// RuleStatistics 规则统计信息
type RuleStatistics struct {
	Total      int `json:"total"`      // 总规则数
	Constraint int `json:"constraint"` // 约束规则数
	Preference int `json:"preference"` // 偏好规则数
	Dependency int `json:"dependency"` // 依赖规则数
	V3         int `json:"v3"`         // V3 规则数
	V4         int `json:"v4"`         // V4 规则数
	Active     int `json:"active"`     // 启用规则数
	Inactive   int `json:"inactive"`   // 禁用规则数
	// 按来源统计
	Manual      int `json:"manual"`      // 手动创建
	LLMParsed   int `json:"llmParsed"`   // LLM 解析
	Migrated    int `json:"migrated"`    // 迁移
	// 按子分类统计
	Forbid      int `json:"forbid"`      // 禁止型
	Must        int `json:"must"`        // 必须型
	Limit       int `json:"limit"`       // 限制型
	Prefer      int `json:"prefer"`      // 优先型
	Suggest     int `json:"suggest"`     // 建议型
}
