package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"jusha/gantt/service/management/config"
	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// StaffingCalculationServiceImpl 排班人数计算服务实现
type StaffingCalculationServiceImpl struct {
	shiftRepo        repository.IShiftRepository
	staffingRuleRepo repository.IShiftStaffingRuleRepository
	weeklyStaffRepo  repository.IShiftWeeklyStaffRepository
	weeklyVolumeRepo repository.IModalityRoomWeeklyVolumeRepository
	modalityRoomRepo repository.IModalityRoomRepository
	timePeriodRepo   repository.ITimePeriodRepository
	configurator     config.IManagementServiceConfigurator
	logger           logging.ILogger
}

// NewStaffingCalculationService 创建排班人数计算服务实例
func NewStaffingCalculationService(
	shiftRepo repository.IShiftRepository,
	staffingRuleRepo repository.IShiftStaffingRuleRepository,
	weeklyStaffRepo repository.IShiftWeeklyStaffRepository,
	weeklyVolumeRepo repository.IModalityRoomWeeklyVolumeRepository,
	modalityRoomRepo repository.IModalityRoomRepository,
	timePeriodRepo repository.ITimePeriodRepository,
	configurator config.IManagementServiceConfigurator,
	logger logging.ILogger,
) domain_service.IStaffingCalculationService {
	return &StaffingCalculationServiceImpl{
		shiftRepo:        shiftRepo,
		staffingRuleRepo: staffingRuleRepo,
		weeklyStaffRepo:  weeklyStaffRepo,
		weeklyVolumeRepo: weeklyVolumeRepo,
		modalityRoomRepo: modalityRoomRepo,
		timePeriodRepo:   timePeriodRepo,
		configurator:     configurator,
		logger:           logger,
	}
}

// CalculateStaffCount 计算排班人数（按天计算）
func (s *StaffingCalculationServiceImpl) CalculateStaffCount(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error) {
	// 1. 获取班次信息
	shift, err := s.shiftRepo.GetByID(ctx, orgID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("班次不存在: %w", err)
	}

	// 2. 获取班次的计算规则
	rule, err := s.staffingRuleRepo.GetByShiftID(ctx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("班次未配置计算规则: %w", err)
	}

	if !rule.IsActive {
		return nil, fmt.Errorf("班次计算规则未启用")
	}

	// 3. 获取时间段信息
	timePeriod, err := s.timePeriodRepo.GetByID(ctx, orgID, rule.TimePeriodID)
	if err != nil {
		return nil, fmt.Errorf("时间段不存在: %w", err)
	}

	// 4. 获取周检查量预估数据（按星期几汇总）
	volumes, err := s.weeklyVolumeRepo.GetByFilter(ctx, orgID, rule.ModalityRoomIDs, nil, []string{rule.TimePeriodID})
	if err != nil {
		return nil, fmt.Errorf("获取周检查量数据失败: %w", err)
	}

	// 5. 计算每日检查量
	dailyVolumes := make(map[int]int) // weekday -> volume
	modalityRoomVolumes := make(map[string]int)

	for _, v := range volumes {
		dailyVolumes[v.Weekday] += v.Volume
		modalityRoomVolumes[v.ModalityRoomID] += v.Volume
	}

	// 计算总量和实际有数据的天数
	totalVolume := 0
	dataDays := 0
	for _, vol := range dailyVolumes {
		totalVolume += vol
		if vol > 0 {
			dataDays++
		}
	}

	// 6. 确定人均上限
	avgReportLimit := rule.AvgReportLimit
	if avgReportLimit <= 0 {
		staffingConfig := s.configurator.GetStaffingConfig()
		avgReportLimit = staffingConfig.DefaultAvgReportLimit
	}
	if avgReportLimit <= 0 {
		avgReportLimit = 50 // 最终兜底默认值
	}

	// 7. 获取当前周人数配置
	weeklyStaffs, _ := s.weeklyStaffRepo.GetByShiftID(ctx, shiftID)
	currentStaffMap := make(map[int]int)
	for _, ws := range weeklyStaffs {
		currentStaffMap[ws.Weekday] = ws.StaffCount
	}

	// 8. 计算每日推荐人数
	dailyResults := make([]*model.DailyStaffingResult, 7)
	weeklyVolume := 0
	totalCalculatedCount := 0

	for i := 0; i < 7; i++ {
		dailyVol := dailyVolumes[i]
		weeklyVolume += dailyVol

		// 计算当日推荐人数
		var calculatedCount int
		if dailyVol > 0 {
			ratio := float64(dailyVol) / float64(avgReportLimit)
			if rule.RoundingMode == model.RoundingModeFloor {
				calculatedCount = int(math.Floor(ratio))
			} else {
				calculatedCount = int(math.Ceil(ratio))
			}
		}
		if calculatedCount < 0 {
			calculatedCount = 0
		}
		totalCalculatedCount += calculatedCount

		dailyResults[i] = &model.DailyStaffingResult{
			Weekday:         i,
			WeekdayName:     model.GetWeekdayName(i),
			DailyVolume:     dailyVol,
			CalculatedCount: calculatedCount,
			CurrentCount:    currentStaffMap[i], // 当前配置，默认为0
		}
	}

	// 9. 构建机房报告量明细
	modalityRooms, _ := s.modalityRoomRepo.BatchGet(ctx, orgID, rule.ModalityRoomIDs)
	modalityRoomMap := make(map[string]*model.ModalityRoom)
	for _, mr := range modalityRooms {
		modalityRoomMap[mr.ID] = mr
	}

	var modalityRoomSummaries []*model.ModalityRoomVolumeSummary
	for _, modalityRoomID := range rule.ModalityRoomIDs {
		summary := &model.ModalityRoomVolumeSummary{
			ModalityRoomID: modalityRoomID,
			TotalVolume:    modalityRoomVolumes[modalityRoomID],
			DataDays:       dataDays,
		}
		if dataDays > 0 {
			summary.AvgVolume = modalityRoomVolumes[modalityRoomID] / dataDays
		}
		if mr, ok := modalityRoomMap[modalityRoomID]; ok {
			summary.ModalityRoomCode = mr.Code
			summary.ModalityRoomName = mr.Name
		}
		modalityRoomSummaries = append(modalityRoomSummaries, summary)
	}

	// 10. 构建计算过程说明
	var steps []string
	steps = append(steps, fmt.Sprintf("1. 统计周检查量预估配置，共 %d 天有数据", dataDays))
	steps = append(steps, fmt.Sprintf("2. 总检查量: %d", totalVolume))
	steps = append(steps, fmt.Sprintf("3. 人均处理上限: %d", avgReportLimit))
	roundingDesc := "向上取整"
	if rule.RoundingMode == model.RoundingModeFloor {
		roundingDesc = "向下取整"
	}
	steps = append(steps, fmt.Sprintf("4. 取整方式: %s", roundingDesc))
	steps = append(steps, "5. 按天计算各日推荐人数:")
	for _, dr := range dailyResults {
		if dr.DailyVolume > 0 {
			steps = append(steps, fmt.Sprintf("   %s: %d / %d = %d人", dr.WeekdayName, dr.DailyVolume, avgReportLimit, dr.CalculatedCount))
		} else {
			steps = append(steps, fmt.Sprintf("   %s: 无数据，0人", dr.WeekdayName))
		}
	}

	return &model.StaffingCalculationPreview{
		ShiftID:          shiftID,
		ShiftName:        shift.Name,
		TimePeriodID:     rule.TimePeriodID,
		TimePeriodName:   timePeriod.Name,
		ModalityRooms:    modalityRoomSummaries,
		TotalVolume:      totalVolume,
		DataDays:         dataDays,
		WeeklyVolume:     weeklyVolume,
		AvgReportLimit:   avgReportLimit,
		RoundingMode:     rule.RoundingMode,
		CalculatedCount:  totalCalculatedCount, // 周总推荐人次
		DailyResults:     dailyResults,
		CalculationSteps: strings.Join(steps, "\n"),
	}, nil
}

// ApplyStaffCount 应用排班人数（批量设置周配置）
func (s *StaffingCalculationServiceImpl) ApplyStaffCount(ctx context.Context, orgID string, req *model.ApplyStaffCountRequest) (*model.ApplyStaffCountResult, error) {
	// 获取班次信息，确保班次存在
	_, err := s.shiftRepo.GetByID(ctx, orgID, req.ShiftID)
	if err != nil {
		return nil, fmt.Errorf("班次不存在: %w", err)
	}

	result := &model.ApplyStaffCountResult{
		ShiftID:      req.ShiftID,
		AppliedCount: req.StaffCount,
	}

	// 写入周配置
	var weeklyStaffs []*model.ShiftWeeklyStaff
	for _, weekday := range req.Weekdays {
		weeklyStaffs = append(weeklyStaffs, &model.ShiftWeeklyStaff{
			ShiftID:    req.ShiftID,
			Weekday:    weekday,
			StaffCount: req.StaffCount,
		})
	}

	if err := s.weeklyStaffRepo.BatchUpsert(ctx, req.ShiftID, weeklyStaffs); err != nil {
		s.logger.Error("Failed to apply weekly staff count", "error", err)
		return nil, fmt.Errorf("更新周默认人数失败: %w", err)
	}

	result.ApplyMode = "weekly"
	result.AffectedDays = req.Weekdays
	result.Message = fmt.Sprintf("已更新 %d 天的默认人数为 %d", len(req.Weekdays), req.StaffCount)

	s.logger.Info("Staff count applied", "shiftId", req.ShiftID, "count", req.StaffCount, "mode", result.ApplyMode)
	return result, nil
}

// GetShiftStaffingRule 获取班次的计算规则
func (s *StaffingCalculationServiceImpl) GetShiftStaffingRule(ctx context.Context, shiftID string) (*model.ShiftStaffingRule, error) {
	return s.staffingRuleRepo.GetByShiftID(ctx, shiftID)
}

// GetStaffingRuleByID 根据规则ID获取计算规则
func (s *StaffingCalculationServiceImpl) GetStaffingRuleByID(ctx context.Context, ruleID string) (*model.ShiftStaffingRule, error) {
	return s.staffingRuleRepo.GetByID(ctx, ruleID)
}

// CreateOrUpdateStaffingRule 创建或更新班次计算规则
func (s *StaffingCalculationServiceImpl) CreateOrUpdateStaffingRule(ctx context.Context, rule *model.ShiftStaffingRule) error {
	// 检查是否已存在规则
	existing, err := s.staffingRuleRepo.GetByShiftID(ctx, rule.ShiftID)
	if err == nil && existing != nil {
		// 更新现有规则
		rule.ID = existing.ID
		rule.UpdatedAt = time.Now()
		return s.staffingRuleRepo.Update(ctx, rule)
	}

	// 创建新规则
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	rule.IsActive = true
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	if rule.RoundingMode == "" {
		rule.RoundingMode = model.RoundingModeCeil
	}

	return s.staffingRuleRepo.Create(ctx, rule)
}

// DeleteStaffingRule 删除班次计算规则
func (s *StaffingCalculationServiceImpl) DeleteStaffingRule(ctx context.Context, ruleID string) error {
	return s.staffingRuleRepo.Delete(ctx, ruleID)
}

// ListStaffingRules 查询所有计算规则
func (s *StaffingCalculationServiceImpl) ListStaffingRules(ctx context.Context, orgID string) ([]*model.ShiftStaffingRule, error) {
	rules, err := s.staffingRuleRepo.List(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// 填充关联信息
	for _, rule := range rules {
		// 获取班次信息
		shift, _ := s.shiftRepo.GetByID(ctx, orgID, rule.ShiftID)
		if shift != nil {
			rule.ShiftName = shift.Name
		}

		// 获取时间段信息
		timePeriod, _ := s.timePeriodRepo.GetByID(ctx, orgID, rule.TimePeriodID)
		if timePeriod != nil {
			rule.TimePeriodName = timePeriod.Name
		}

		// 获取机房信息
		modalityRooms, _ := s.modalityRoomRepo.BatchGet(ctx, orgID, rule.ModalityRoomIDs)
		rule.ModalityRooms = modalityRooms
	}

	return rules, nil
}

// GetWeeklyStaffConfig 获取班次的周默认人数配置
func (s *StaffingCalculationServiceImpl) GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*model.WeeklyStaffConfig, error) {
	// 获取班次信息
	shift, err := s.shiftRepo.GetByID(ctx, orgID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("班次不存在: %w", err)
	}

	// 获取周配置
	weeklyStaffs, err := s.weeklyStaffRepo.GetByShiftID(ctx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("获取周默认人数失败: %w", err)
	}

	// 构建周配置映射
	weeklyMap := make(map[int]*model.ShiftWeeklyStaff)
	for _, ws := range weeklyStaffs {
		weeklyMap[ws.Weekday] = ws
	}

	// 构建完整的7天配置
	weeklyConfig := make([]model.WeekdayStaff, 7)
	for i := 0; i < 7; i++ {
		weeklyConfig[i] = model.WeekdayStaff{
			Weekday:     i,
			WeekdayName: model.GetWeekdayName(i),
			StaffCount:  0, // 默认为0
			IsCustom:    false,
		}
		if ws, ok := weeklyMap[i]; ok {
			weeklyConfig[i].StaffCount = ws.StaffCount
			weeklyConfig[i].IsCustom = true
		}
	}

	return &model.WeeklyStaffConfig{
		ShiftID:      shiftID,
		ShiftName:    shift.Name,
		WeeklyConfig: weeklyConfig,
	}, nil
}

// SetWeeklyStaffConfig 设置班次的周默认人数配置
func (s *StaffingCalculationServiceImpl) SetWeeklyStaffConfig(ctx context.Context, shiftID string, weeklyConfig []model.WeekdayStaff) error {
	var weeklyStaffs []*model.ShiftWeeklyStaff
	for _, wc := range weeklyConfig {
		if wc.IsCustom {
			weeklyStaffs = append(weeklyStaffs, &model.ShiftWeeklyStaff{
				ShiftID:    shiftID,
				Weekday:    wc.Weekday,
				StaffCount: wc.StaffCount,
			})
		}
	}

	// 先删除所有现有配置
	if err := s.weeklyStaffRepo.DeleteByShiftID(ctx, shiftID); err != nil {
		s.logger.Error("Failed to delete existing weekly staff", "error", err)
		return fmt.Errorf("删除现有周配置失败: %w", err)
	}

	// 批量创建新配置
	if len(weeklyStaffs) > 0 {
		if err := s.weeklyStaffRepo.BatchUpsert(ctx, shiftID, weeklyStaffs); err != nil {
			s.logger.Error("Failed to set weekly staff config", "error", err)
			return fmt.Errorf("设置周默认人数失败: %w", err)
		}
	}

	s.logger.Info("Weekly staff config updated", "shiftId", shiftID, "count", len(weeklyStaffs))
	return nil
}
