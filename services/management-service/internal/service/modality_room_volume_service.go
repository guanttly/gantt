package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// ModalityRoomVolumeServiceImpl 机房报告量服务实现
type ModalityRoomVolumeServiceImpl struct {
	volumeRepo       repository.IModalityRoomVolumeRepository
	modalityRoomRepo repository.IModalityRoomRepository
	timePeriodRepo   repository.ITimePeriodRepository
	logger           logging.ILogger
}

// NewModalityRoomVolumeService 创建机房报告量服务实例
func NewModalityRoomVolumeService(
	volumeRepo repository.IModalityRoomVolumeRepository,
	modalityRoomRepo repository.IModalityRoomRepository,
	timePeriodRepo repository.ITimePeriodRepository,
	logger logging.ILogger,
) domain_service.IModalityRoomVolumeService {
	return &ModalityRoomVolumeServiceImpl{
		volumeRepo:       volumeRepo,
		modalityRoomRepo: modalityRoomRepo,
		timePeriodRepo:   timePeriodRepo,
		logger:           logger,
	}
}

// CreateVolume 创建报告量记录
func (s *ModalityRoomVolumeServiceImpl) CreateVolume(ctx context.Context, volume *model.ModalityRoomVolume) error {
	if volume.ID == "" {
		volume.ID = uuid.New().String()
	}

	volume.CreatedAt = time.Now()
	volume.UpdatedAt = time.Now()

	if err := s.volumeRepo.Create(ctx, volume); err != nil {
		s.logger.Error("Failed to create volume", "error", err)
		return fmt.Errorf("创建报告量记录失败: %w", err)
	}

	s.logger.Info("Volume created", "id", volume.ID)
	return nil
}

// UpdateVolume 更新报告量记录
func (s *ModalityRoomVolumeServiceImpl) UpdateVolume(ctx context.Context, volume *model.ModalityRoomVolume) error {
	volume.UpdatedAt = time.Now()

	if err := s.volumeRepo.Update(ctx, volume); err != nil {
		s.logger.Error("Failed to update volume", "error", err)
		return fmt.Errorf("更新报告量记录失败: %w", err)
	}

	s.logger.Info("Volume updated", "id", volume.ID)
	return nil
}

// DeleteVolume 删除报告量记录
func (s *ModalityRoomVolumeServiceImpl) DeleteVolume(ctx context.Context, volumeID string) error {
	if err := s.volumeRepo.Delete(ctx, volumeID); err != nil {
		s.logger.Error("Failed to delete volume", "error", err)
		return fmt.Errorf("删除报告量记录失败: %w", err)
	}

	s.logger.Info("Volume deleted", "id", volumeID)
	return nil
}

// GetVolume 获取报告量记录
func (s *ModalityRoomVolumeServiceImpl) GetVolume(ctx context.Context, volumeID string) (*model.ModalityRoomVolume, error) {
	volume, err := s.volumeRepo.GetByID(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("获取报告量记录失败: %w", err)
	}
	return volume, nil
}

// ListVolumes 查询报告量列表
func (s *ModalityRoomVolumeServiceImpl) ListVolumes(ctx context.Context, filter *model.ModalityRoomVolumeFilter) (*model.ModalityRoomVolumeListResult, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	result, err := s.volumeRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list volumes", "error", err)
		return nil, fmt.Errorf("查询报告量列表失败: %w", err)
	}

	// 填充冗余字段（机房名称、时间段名称）
	if len(result.Items) > 0 {
		// 收集所有机房ID和时间段ID
		modalityRoomIDs := make(map[string]bool)
		timePeriodIDs := make(map[string]bool)
		for _, v := range result.Items {
			modalityRoomIDs[v.ModalityRoomID] = true
			timePeriodIDs[v.TimePeriodID] = true
		}

		// 批量获取机房信息
		var modalityRoomIDList []string
		for id := range modalityRoomIDs {
			modalityRoomIDList = append(modalityRoomIDList, id)
		}
		modalityRooms, _ := s.modalityRoomRepo.BatchGet(ctx, filter.OrgID, modalityRoomIDList)
		modalityRoomMap := make(map[string]*model.ModalityRoom)
		for _, mr := range modalityRooms {
			modalityRoomMap[mr.ID] = mr
		}

		// 批量获取时间段信息
		timePeriods, _ := s.timePeriodRepo.GetActiveTimePeriods(ctx, filter.OrgID)
		timePeriodMap := make(map[string]*model.TimePeriod)
		for _, tp := range timePeriods {
			timePeriodMap[tp.ID] = tp
		}

		// 填充冗余字段
		for _, v := range result.Items {
			if mr, ok := modalityRoomMap[v.ModalityRoomID]; ok {
				v.ModalityRoomCode = mr.Code
				v.ModalityRoomName = mr.Name
			}
			if tp, ok := timePeriodMap[v.TimePeriodID]; ok {
				v.TimePeriodName = tp.Name
			}
		}
	}

	return result, nil
}

// ImportFromExcel 从Excel导入报告量数据
func (s *ModalityRoomVolumeServiceImpl) ImportFromExcel(ctx context.Context, orgID string, reader io.Reader) (*model.VolumeImportResult, error) {
	result := &model.VolumeImportResult{
		ErrorDetails: []string{},
	}

	// 读取Excel文件
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("读取Excel文件失败: %w", err)
	}
	defer f.Close()

	// 获取第一个工作表
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel文件为空或只有表头")
	}

	// 预加载机房和时间段数据
	modalityRooms, _ := s.modalityRoomRepo.GetActiveModalityRooms(ctx, orgID)
	modalityRoomCodeMap := make(map[string]*model.ModalityRoom)
	for _, mr := range modalityRooms {
		modalityRoomCodeMap[mr.Code] = mr
	}

	timePeriods, _ := s.timePeriodRepo.GetActiveTimePeriods(ctx, orgID)
	timePeriodNameMap := make(map[string]*model.TimePeriod)
	for _, tp := range timePeriods {
		timePeriodNameMap[tp.Name] = tp
	}

	// 解析数据行（跳过表头）
	var volumes []*model.ModalityRoomVolume
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		result.TotalRows++

		if len(row) < 4 {
			result.FailedRows++
			result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("第%d行: 列数不足", i+1))
			continue
		}

		modalityRoomCode := row[0]
		dateStr := row[1]
		timePeriodName := row[2]
		volumeStr := row[3]

		// 验证机房
		modalityRoom, ok := modalityRoomCodeMap[modalityRoomCode]
		if !ok {
			result.FailedRows++
			result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("第%d行: 机房编码 %s 不存在", i+1, modalityRoomCode))
			continue
		}

		// 解析日期
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// 尝试其他日期格式
			date, err = time.Parse("2006/01/02", dateStr)
			if err != nil {
				result.FailedRows++
				result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("第%d行: 日期格式错误 %s", i+1, dateStr))
				continue
			}
		}

		// 验证时间段
		timePeriod, ok := timePeriodNameMap[timePeriodName]
		if !ok {
			result.FailedRows++
			result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("第%d行: 时间段 %s 不存在", i+1, timePeriodName))
			continue
		}

		// 解析报告量
		reportVolume, err := strconv.Atoi(volumeStr)
		if err != nil {
			result.FailedRows++
			result.ErrorDetails = append(result.ErrorDetails, fmt.Sprintf("第%d行: 报告量格式错误 %s", i+1, volumeStr))
			continue
		}

		volumes = append(volumes, &model.ModalityRoomVolume{
			ID:             uuid.New().String(),
			OrgID:          orgID,
			ModalityRoomID: modalityRoom.ID,
			Date:           date,
			TimePeriodID:   timePeriod.ID,
			ReportVolume:   reportVolume,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
	}

	// 批量导入
	if len(volumes) > 0 {
		if err := s.volumeRepo.BatchUpsert(ctx, volumes); err != nil {
			s.logger.Error("Failed to batch upsert volumes", "error", err)
			return nil, fmt.Errorf("批量导入失败: %w", err)
		}
		result.SuccessRows = len(volumes)
	}

	s.logger.Info("Volume import completed", "total", result.TotalRows, "success", result.SuccessRows, "failed", result.FailedRows)
	return result, nil
}

// ExportTemplate 导出Excel模板
func (s *ModalityRoomVolumeServiceImpl) ExportTemplate(ctx context.Context, orgID string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "报告量导入模板"
	f.SetSheetName("Sheet1", sheetName)

	// 设置表头
	headers := []string{"机房编码", "机房名称(仅参考)", "日期(YYYY-MM-DD)", "时间段名称", "报告量"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetRowStyle(sheetName, 1, 1, headerStyle)

	// 获取机房和时间段数据，添加示例行
	modalityRooms, _ := s.modalityRoomRepo.GetActiveModalityRooms(ctx, orgID)
	timePeriods, _ := s.timePeriodRepo.GetActiveTimePeriods(ctx, orgID)

	row := 2
	today := time.Now().Format("2006-01-02")
	for _, mr := range modalityRooms {
		for _, tp := range timePeriods {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), mr.Code)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), mr.Name)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), today)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), tp.Name)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), 0)
			row++
		}
		if row > 10 { // 限制示例行数
			break
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "B", 20)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 15)
	f.SetColWidth(sheetName, "E", "E", 10)

	// 写入buffer
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, fmt.Errorf("生成Excel模板失败: %w", err)
	}

	return buf.Bytes(), nil
}

// GetLatestWeekVolumes 获取最近一周有数据的报告量
func (s *ModalityRoomVolumeServiceImpl) GetLatestWeekVolumes(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string) ([]*model.ModalityRoomVolume, error) {
	return s.volumeRepo.GetLatestWeekVolumes(ctx, orgID, modalityRoomIDs, timePeriodID)
}

// GetVolumesSummary 获取报告量汇总
func (s *ModalityRoomVolumeServiceImpl) GetVolumesSummary(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string, startDate, endDate time.Time) ([]*model.ModalityRoomVolumeSummary, error) {
	return s.volumeRepo.GetVolumesSummary(ctx, orgID, modalityRoomIDs, timePeriodID, startDate, endDate)
}
