package auth

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// User 用户账户模型。
type User struct {
	ID           string    `gorm:"primaryKey;size:64" json:"id"`
	Username     string    `gorm:"size:64;not null;uniqueIndex:uk_username" json:"username"`
	Email        string    `gorm:"size:128;not null;uniqueIndex:uk_email" json:"email"`
	Phone        *string   `gorm:"size:20;index:idx_phone" json:"phone,omitempty"`
	PasswordHash string    `gorm:"size:256;not null" json:"-"`
	Status       string    `gorm:"size:16;not null;default:active" json:"status"`
	MustResetPwd bool      `gorm:"not null;default:false" json:"must_reset_pwd"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }

// Role 角色模型。
type Role struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	Name        string    `gorm:"size:64;not null;uniqueIndex:uk_role_name" json:"name"`
	DisplayName string    `gorm:"size:64;not null" json:"display_name"`
	Permissions JSONArray `gorm:"type:json;not null" json:"permissions"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Role) TableName() string { return "roles" }

// UserNodeRole 用户-组织节点-角色关联。
type UserNodeRole struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"size:64;not null;index:idx_unr_user" json:"user_id"`
	OrgNodeID string    `gorm:"size:64;not null;index:idx_unr_node" json:"org_node_id"`
	RoleID    string    `gorm:"size:64;not null" json:"role_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
	Role *Role `gorm:"foreignKey:RoleID" json:"-"`
}

func (UserNodeRole) TableName() string { return "user_node_roles" }

// JSONArray 用于 GORM JSON 列的字符串切片。
type JSONArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 数据。
func (j *JSONArray) Scan(value interface{}) error {
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
		return fmt.Errorf("JSONArray.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口，序列化为 JSON 写入数据库。
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	data, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// UserInfo 用户公开信息（API 返回）。
type UserInfo struct {
	ID           string  `json:"id"`
	Username     string  `json:"username"`
	Email        string  `json:"email"`
	Phone        *string `json:"phone,omitempty"`
	MustResetPwd bool    `json:"must_reset_pwd"`
}

// NodeRoleInfo 用户关联的节点角色信息。
type NodeRoleInfo struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
	NodePath string `json:"node_path"`
	RoleName string `json:"role_name"`
}

// ToInfo 将 User 转换为公开信息。
func (u *User) ToInfo() UserInfo {
	return UserInfo{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Phone:        u.Phone,
		MustResetPwd: u.MustResetPwd,
	}
}
