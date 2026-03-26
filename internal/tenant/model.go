package tenant

import (
	"time"
)

const (
	NodeTypeOrganization = "organization"
	NodeTypeCampus       = "campus"
	NodeTypeDepartment   = "department"
	NodeTypeCustom       = "custom"
)

const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
)

// OrgNode 组织树节点模型。
type OrgNode struct {
	ID           string    `gorm:"primaryKey;size:64" json:"id"`
	ParentID     *string   `gorm:"size:64;index:idx_parent" json:"parent_id"`
	NodeType     string    `gorm:"size:32;not null" json:"node_type"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Code         string    `gorm:"size:64;not null" json:"code"`
	Path         string    `gorm:"size:512;not null;index:idx_path" json:"path"`
	Depth        int       `gorm:"not null;default:0" json:"depth"`
	IsLoginPoint bool      `gorm:"not null;default:false" json:"is_login_point"`
	Status       string    `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Parent   *OrgNode  `gorm:"foreignKey:ParentID" json:"-"`
	Children []OrgNode `gorm:"foreignKey:ParentID" json:"-"`
}

// TableName 指定表名。
func (OrgNode) TableName() string {
	return "org_nodes"
}

// IsActive 节点是否启用。
func (n *OrgNode) IsActive() bool {
	return n.Status == StatusActive
}

// OrgNodeTree 组织树节点含子节点列表。
type OrgNodeTree struct {
	OrgNode
	Children []*OrgNodeTree `json:"children,omitempty"`
}

// TenantModel 嵌入到所有业务 Model 中，自动包含 org_node_id 列。
type TenantModel struct {
	OrgNodeID string `gorm:"size:64;not null;index:idx_org_node" json:"org_node_id"`
}
