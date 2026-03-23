package model

import "time"

// Holiday 节假日配置领域模型
type Holiday struct {
	ID          uint64      `json:"id"`
	OrgID       string      `json:"orgId"`
	Name        string      `json:"name"`
	Date        time.Time   `json:"date"`
	Type        HolidayType `json:"type"`
	Description string      `json:"description"`
	Year        int         `json:"year"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// HolidayType 节假日类型
type HolidayType string

const (
	HolidayTypeHoliday HolidayType = "holiday" // 法定节假日（不上班）
	HolidayTypeWorkday HolidayType = "workday" // 调休补班（周末上班）
	HolidayTypeCustom  HolidayType = "custom"  // 组织自定义假期
)

// IsHoliday 是否为休息日
func (h *Holiday) IsHoliday() bool {
	return h.Type == HolidayTypeHoliday || h.Type == HolidayTypeCustom
}

// IsWorkday 是否为工作日（调休补班）
func (h *Holiday) IsWorkday() bool {
	return h.Type == HolidayTypeWorkday
}
