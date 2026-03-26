package audit

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// AuditLog 审计日志模型。
type AuditLog struct {
	ID           string    `gorm:"primaryKey;size:64" json:"id"`
	OrgNodeID    *string   `gorm:"size:64;index:idx_audit_org_time" json:"org_node_id"`
	UserID       string    `gorm:"size:64;not null;index:idx_audit_user" json:"user_id"`
	Username     string    `gorm:"size:64;not null" json:"username"`
	Action       string    `gorm:"size:64;not null;index:idx_audit_action" json:"action"`
	ResourceType string    `gorm:"size:64;not null" json:"resource_type"`
	ResourceID   *string   `gorm:"size:64" json:"resource_id"`
	Detail       JSONMap   `gorm:"type:json" json:"detail"`
	IP           string    `gorm:"size:45" json:"ip"`
	UserAgent    string    `gorm:"size:256" json:"user_agent"`
	StatusCode   int       `gorm:"not null" json:"status_code"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 指定表名。
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditEntry 审计日志记录条目（中间件使用）。
type AuditEntry struct {
	OrgNodeID    string
	UserID       string
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	Detail       map[string]any
	IP           string
	UserAgent    string
	StatusCode   int
}

// JSONMap 用于 GORM JSON 列的 map 类型。
type JSONMap map[string]any

// Scan 实现 sql.Scanner 接口。
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONMap.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口。
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	data, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
