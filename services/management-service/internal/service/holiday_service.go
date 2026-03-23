package service

import (
	"context"
	"fmt"
	"time"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/logging"
)

// HolidayService 节假日管理服务
type HolidayService struct {
	holidayRepo repository.IHolidayRepository
	logger      logging.ILogger
}

// NewHolidayService 创建节假日管理服务
func NewHolidayService(holidayRepo repository.IHolidayRepository, logger logging.ILogger) *HolidayService {
	return &HolidayService{
		holidayRepo: holidayRepo,
		logger:      logger.With("service", "HolidayService"),
	}
}

// InitChinaHolidays2025 初始化2025年中国法定节假日数据
// 已废弃：节假日数据应通过API导入或配置文件加载
func (s *HolidayService) InitChinaHolidays2025(ctx context.Context) error {
	s.logger.Warn("InitChinaHolidays2025 is deprecated, please use API to import holiday data")
	return fmt.Errorf("this method is deprecated, please use holiday import API")
}

// InitChinaHolidaysForYear 初始化指定年份的中国法定节假日
// 注意：需要根据实际政府公告更新节假日数据
func (s *HolidayService) InitChinaHolidaysForYear(ctx context.Context, year int) error {
	// TODO: 根据实际年份加载对应的节假日数据
	// 可以从配置文件、数据库、或者外部API获取

	if year == 2025 {
		return s.InitChinaHolidays2025(ctx)
	}

	return fmt.Errorf("no holiday data for year %d", year)
}

// GetWorkingDays 计算两个日期之间的工作日天数
func (s *HolidayService) GetWorkingDays(ctx context.Context, orgID string, startDate, endDate time.Time) (int, error) {
	// 获取日期范围内的节假日配置
	holidays, err := s.holidayRepo.ListByDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		return 0, fmt.Errorf("get holidays: %w", err)
	}

	// 构建节假日映射表
	holidayMap := make(map[string]model.HolidayType)
	for _, h := range holidays {
		dateKey := h.Date.Format("2006-01-02")
		if existing, ok := holidayMap[dateKey]; !ok || (existing != h.Type && h.OrgID != "") {
			holidayMap[dateKey] = h.Type
		}
	}

	// 计算工作日天数
	workingDays := 0
	current := startDate

	for !current.After(endDate) {
		dateKey := current.Format("2006-01-02")

		if holidayType, exists := holidayMap[dateKey]; exists {
			if holidayType == model.HolidayTypeWorkday {
				workingDays++
			}
		} else if !isWeekend(current) {
			workingDays++
		}

		current = current.AddDate(0, 0, 1)
	}

	return workingDays, nil
}
