package service

import (
	"context"
	"fmt"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// RuleStatisticsService 规则统计服务实现
type RuleStatisticsService struct {
	logger   logging.ILogger
	ruleRepo repository.ISchedulingRuleRepository
}

// NewRuleStatisticsService 创建规则统计服务
func NewRuleStatisticsService(
	logger logging.ILogger,
	ruleRepo repository.ISchedulingRuleRepository,
) domain_service.IRuleStatisticsService {
	return &RuleStatisticsService{
		logger:   logger,
		ruleRepo: ruleRepo,
	}
}

// GetRuleStatistics 获取规则统计信息
func (s *RuleStatisticsService) GetRuleStatistics(ctx context.Context, orgID string) (*domain_service.RuleStatistics, error) {
	// 获取所有规则
	filter := &model.SchedulingRuleFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 10000,
	}
	rules, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取规则列表失败: %w", err)
	}

	stats := &domain_service.RuleStatistics{}

	for _, rule := range rules.Items {
		stats.Total++

		// 按分类统计
		switch rule.Category {
		case "constraint":
			stats.Constraint++
		case "preference":
			stats.Preference++
		case "dependency":
			stats.Dependency++
		}

		// 按版本统计
		switch rule.Version {
		case "v3", "":
			stats.V3++
		case "v4":
			stats.V4++
		}

		// 按状态统计
		if rule.IsActive {
			stats.Active++
		} else {
			stats.Inactive++
		}

		// 按来源统计
		switch rule.SourceType {
		case "manual":
			stats.Manual++
		case "llm_parsed":
			stats.LLMParsed++
		case "migrated":
			stats.Migrated++
		}

		// 按子分类统计
		switch rule.SubCategory {
		case "forbid":
			stats.Forbid++
		case "must":
			stats.Must++
		case "limit":
			stats.Limit++
		case "prefer":
			stats.Prefer++
		case "suggest":
			stats.Suggest++
		}
	}

	return stats, nil
}
