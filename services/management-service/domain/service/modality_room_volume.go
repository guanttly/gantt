package service

import (
	"context"
	"io"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// IModalityRoomVolumeService 检查量管理服务接口
type IModalityRoomVolumeService interface {
	// CreateVolume 创建检查量记录
	CreateVolume(ctx context.Context, volume *model.ModalityRoomVolume) error

	// UpdateVolume 更新检查量记录
	UpdateVolume(ctx context.Context, volume *model.ModalityRoomVolume) error

	// DeleteVolume 删除检查量记录
	DeleteVolume(ctx context.Context, volumeID string) error

	// GetVolume 获取检查量记录
	GetVolume(ctx context.Context, volumeID string) (*model.ModalityRoomVolume, error)

	// ListVolumes 查询检查量列表
	ListVolumes(ctx context.Context, filter *model.ModalityRoomVolumeFilter) (*model.ModalityRoomVolumeListResult, error)

	// ImportFromExcel 从Excel导入检查量数据
	// 返回导入结果，包含成功/失败数量和错误详情
	ImportFromExcel(ctx context.Context, orgID string, reader io.Reader) (*model.VolumeImportResult, error)

	// ExportTemplate 导出Excel模板
	// 返回Excel模板的字节数据
	ExportTemplate(ctx context.Context, orgID string) ([]byte, error)

	// GetLatestWeekVolumes 获取最近一周有数据的检查量
	GetLatestWeekVolumes(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string) ([]*model.ModalityRoomVolume, error)

	// GetVolumesSummary 获取检查量汇总
	GetVolumesSummary(ctx context.Context, orgID string, modalityRoomIDs []string, timePeriodID string, startDate, endDate time.Time) ([]*model.ModalityRoomVolumeSummary, error)
}
