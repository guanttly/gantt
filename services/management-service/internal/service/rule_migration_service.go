package service

import (
	"context"
	"fmt"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// RuleMigrationService 规则迁移服务实现
type RuleMigrationService struct {
	logger             logging.ILogger
	ruleRepo           repository.ISchedulingRuleRepository
	ruleDependencyRepo repository.IRuleDependencyRepository
	ruleConflictRepo   repository.IRuleConflictRepository
}

// NewRuleMigrationService 创建规则迁移服务
func NewRuleMigrationService(
	logger logging.ILogger,
	ruleRepo repository.ISchedulingRuleRepository,
	ruleDependencyRepo repository.IRuleDependencyRepository,
	ruleConflictRepo repository.IRuleConflictRepository,
) domain_service.IRuleMigrationService {
	return &RuleMigrationService{
		logger:             logger,
		ruleRepo:           ruleRepo,
		ruleDependencyRepo: ruleDependencyRepo,
		ruleConflictRepo:   ruleConflictRepo,
	}
}

// PreviewMigration 预览迁移
func (s *RuleMigrationService) PreviewMigration(ctx context.Context, orgID string) (*domain_service.MigrationPreview, error) {
	// 1. 获取所有 V3 规则（Version 为空或 "v3"）
	filter := &model.SchedulingRuleFilter{
		OrgID:    orgID,
		Version:  stringPtr("v3"),
		Page:     1,
		PageSize: 10000, // 获取所有规则
	}

	rules, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取 V3 规则失败: %w", err)
	}

	preview := &domain_service.MigrationPreview{
		TotalV3Rules:      len(rules.Items),
		MigratableRules:   make([]*domain_service.MigratableRule, 0),
		UnmigratableRules: make([]*domain_service.UnmigratableRule, 0),
		Warnings:          make([]string, 0),
	}

	// 2. 分析每个规则，判断是否可迁移
	for _, rule := range rules.Items {
		migratable := s.analyzeRuleMigratability(rule)
		if migratable.CanMigrate {
			preview.MigratableRules = append(preview.MigratableRules, &domain_service.MigratableRule{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				RuleType:    string(rule.RuleType),
				ApplyScope:  string(rule.ApplyScope),
				TimeScope:   string(rule.TimeScope),
				Category:    migratable.Category,
				SubCategory: migratable.SubCategory,
				SourceType:  "migrated",
				Version:     "v4",
				Suggestions: migratable.Suggestions,
			})
		} else {
			preview.UnmigratableRules = append(preview.UnmigratableRules, &domain_service.UnmigratableRule{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Reason:      migratable.Reason,
				Suggestions: migratable.Suggestions,
			})
		}
	}

	// 3. 估算迁移时间（每个规则约 100ms）
	preview.EstimatedTime = time.Duration(len(preview.MigratableRules)) * 100 * time.Millisecond

	return preview, nil
}

// RuleMigratability 规则可迁移性分析结果
type RuleMigratability struct {
	CanMigrate  bool
	Category    string
	SubCategory string
	Reason      string
	Suggestions []string
}

// analyzeRuleMigratability 分析规则可迁移性
func (s *RuleMigrationService) analyzeRuleMigratability(rule *model.SchedulingRule) *RuleMigratability {
	result := &RuleMigratability{
		CanMigrate:  true,
		Suggestions: make([]string, 0),
	}

	// 根据规则类型推断分类
	result.Category = inferCategoryFromRuleType(rule.RuleType)
	result.SubCategory = inferSubCategoryFromRuleType(rule.RuleType)

	// 检查规则是否有效
	if !rule.IsActive {
		result.CanMigrate = false
		result.Reason = "规则已禁用"
		result.Suggestions = append(result.Suggestions, "请先启用规则再迁移")
		return result
	}

	// 检查规则类型是否支持
	if !isRuleTypeSupported(rule.RuleType) {
		result.CanMigrate = false
		result.Reason = fmt.Sprintf("规则类型 %s 不支持迁移", rule.RuleType)
		result.Suggestions = append(result.Suggestions, "请手动创建 V4 规则")
		return result
	}

	// 添加迁移建议
	if rule.Category == "" {
		result.Suggestions = append(result.Suggestions, fmt.Sprintf("建议分类为: %s/%s", result.Category, result.SubCategory))
	}

	return result
}

// isRuleTypeSupported 检查规则类型是否支持迁移
func isRuleTypeSupported(ruleType model.RuleType) bool {
	supportedTypes := []model.RuleType{
		model.RuleTypeExclusive,
		model.RuleTypeCombinable,
		model.RuleTypeRequiredTogether,
		model.RuleTypePeriodic,
		model.RuleTypeMaxCount,
		model.RuleTypeForbiddenDay,
		model.RuleTypePreferred,
	}

	for _, t := range supportedTypes {
		if ruleType == t {
			return true
		}
	}
	return false
}

// ExecuteMigration 执行迁移
func (s *RuleMigrationService) ExecuteMigration(ctx context.Context, orgID string, plan *domain_service.MigrationPlan) (*domain_service.MigrationResult, error) {
	if plan.OrgID != orgID {
		return nil, fmt.Errorf("组织ID不匹配")
	}

	migrationID := fmt.Sprintf("migration_%d", time.Now().Unix())
	result := &domain_service.MigrationResult{
		MigrationID:  migrationID,
		TotalRules:   len(plan.RuleIDs),
		SuccessCount: 0,
		FailedCount:  0,
		FailedRules:  make([]*domain_service.FailedMigrationRule, 0),
		CreatedAt:    time.Now(),
	}

	// 如果是 DryRun，只返回预览结果
	if plan.DryRun {
		s.logger.Info("Dry run mode, skipping actual migration", "migrationId", migrationID)
		return result, nil
	}

	// 逐个迁移规则
	for _, ruleID := range plan.RuleIDs {
		if err := s.migrateSingleRule(ctx, orgID, ruleID, plan); err != nil {
			result.FailedCount++
			result.FailedRules = append(result.FailedRules, &domain_service.FailedMigrationRule{
				RuleID: ruleID,
				Error:  err.Error(),
			})
			s.logger.Warn("迁移规则失败", "ruleId", ruleID, "error", err)
		} else {
			result.SuccessCount++
		}
	}

	s.logger.Info("迁移完成", "migrationId", migrationID, "success", result.SuccessCount, "failed", result.FailedCount)
	return result, nil
}

// migrateSingleRule 迁移单个规则
func (s *RuleMigrationService) migrateSingleRule(ctx context.Context, orgID, ruleID string, plan *domain_service.MigrationPlan) error {
	// 1. 获取规则
	rule, err := s.ruleRepo.GetByID(ctx, orgID, ruleID)
	if err != nil {
		return fmt.Errorf("获取规则失败: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("规则不存在")
	}

	// 2. 检查是否为 V3 规则
	if rule.Version != "" && rule.Version != "v3" {
		return fmt.Errorf("规则不是 V3 规则，无法迁移")
	}

	// 3. 填充 V4 字段
	if plan.FillDefaults || rule.Category == "" {
		rule.Category = inferCategoryFromRuleType(rule.RuleType)
		rule.SubCategory = inferSubCategoryFromRuleType(rule.RuleType)
	}

	rule.SourceType = "migrated"
	rule.Version = "v4"

	// 4. 更新规则
	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return fmt.Errorf("更新规则失败: %w", err)
	}

	s.logger.Info("规则迁移成功", "ruleId", ruleID, "category", rule.Category, "subCategory", rule.SubCategory)
	return nil
}

// RollbackMigration 回滚迁移
func (s *RuleMigrationService) RollbackMigration(ctx context.Context, orgID string, migrationID string) error {
	// TODO: 实现回滚逻辑
	// 1. 根据 migrationID 查找迁移记录
	// 2. 将迁移的规则恢复为 V3（Version="v3", SourceType="manual"）
	// 3. 清除 V4 字段（Category, SubCategory）
	s.logger.Warn("回滚功能尚未实现", "migrationId", migrationID)
	return fmt.Errorf("回滚功能尚未实现")
}

// GetMigrationStatus 获取迁移状态
func (s *RuleMigrationService) GetMigrationStatus(ctx context.Context, orgID string, migrationID string) (*domain_service.MigrationStatus, error) {
	// TODO: 实现状态查询逻辑
	// 1. 根据 migrationID 查找迁移记录
	// 2. 返回迁移状态和进度
	s.logger.Warn("状态查询功能尚未实现", "migrationId", migrationID)
	return &domain_service.MigrationStatus{
		MigrationID: migrationID,
		Status:      "completed",
		Progress:    1.0,
	}, nil
}

// 辅助函数：从规则类型推断分类
func inferCategoryFromRuleType(ruleType model.RuleType) string {
	switch ruleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay, model.RuleTypeMaxCount,
		model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return model.CategoryConstraint
	case model.RuleTypePreferred, model.RuleTypeCombinable:
		return model.CategoryPreference
	default:
		return model.CategoryConstraint
	}
}

// 辅助函数：从规则类型推断子分类
func inferSubCategoryFromRuleType(ruleType model.RuleType) string {
	switch ruleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay:
		return model.SubCategoryForbid
	case model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return model.SubCategoryMust
	case model.RuleTypeMaxCount:
		return model.SubCategoryLimit
	case model.RuleTypePreferred:
		return model.SubCategoryPrefer
	case model.RuleTypeCombinable:
		return model.SubCategorySuggest
	default:
		return model.SubCategoryLimit
	}
}
