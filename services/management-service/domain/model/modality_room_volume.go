package model

import (
	"time"
)

// ModalityRoomVolume 机房检查量领域模型
type ModalityRoomVolume struct {
	ID               string    `json:"id"`
	OrgID            string    `json:"orgId"`
	ModalityRoomID   string    `json:"modalityRoomId"`   // 机房ID
	ModalityRoomCode string    `json:"modalityRoomCode"` // 机房编码（冗余，方便展示）
	ModalityRoomName string    `json:"modalityRoomName"` // 机房名称（冗余，方便展示）
	Date             time.Time `json:"date"`             // 日期
	TimePeriodID     string    `json:"timePeriodId"`     // 时间段ID
	TimePeriodName   string    `json:"timePeriodName"`   // 时间段名称（冗余，方便展示）
	ReportVolume     int       `json:"reportVolume"`     // 报告量/检查量
	Notes            string    `json:"notes"`            // 备注
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ModalityRoomVolumeFilter 检查量查询过滤器
type ModalityRoomVolumeFilter struct {
	OrgID           string
	ModalityRoomID  string    // 机房ID
	ModalityRoomIDs []string  // 多个机房ID
	TimePeriodID    string    // 时间段ID
	StartDate       time.Time // 开始日期
	EndDate         time.Time // 结束日期
	Page            int
	PageSize        int
}

// ModalityRoomVolumeListResult 检查量列表结果
type ModalityRoomVolumeListResult struct {
	Items    []*ModalityRoomVolume `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

// ModalityRoomVolumeSummary 机房检查量汇总（用于计算预览）
type ModalityRoomVolumeSummary struct {
	ModalityRoomID   string `json:"modalityRoomId"`
	ModalityRoomCode string `json:"modalityRoomCode"`
	ModalityRoomName string `json:"modalityRoomName"`
	TotalVolume      int    `json:"totalVolume"` // 总检查量
	DataDays         int    `json:"dataDays"`    // 有数据的天数
	AvgVolume        int    `json:"avgVolume"`   // 日均检查量
}

// VolumeImportRequest Excel导入请求
type VolumeImportRequest struct {
	OrgID string `json:"orgId"`
}

// VolumeImportResult Excel导入结果
type VolumeImportResult struct {
	TotalRows    int      `json:"totalRows"`    // 总行数
	SuccessRows  int      `json:"successRows"`  // 成功行数
	FailedRows   int      `json:"failedRows"`   // 失败行数
	ErrorDetails []string `json:"errorDetails"` // 错误详情
}
