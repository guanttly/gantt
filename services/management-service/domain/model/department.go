package model

import (
	"fmt"
	"time"
)

// Department 部门领域模型
type Department struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code"`        // 部门编码
	Name        string     `json:"name"`        // 部门名称
	ParentID    *string    `json:"parentId"`    // 父部门ID
	Level       int        `json:"level"`       // 层级
	Path        string     `json:"path"`        // 部门路径
	Description string     `json:"description"` // 部门描述
	ManagerID   *string    `json:"managerId"`   // 部门经理员工ID
	SortOrder   int        `json:"sortOrder"`   // 排序
	IsActive    bool       `json:"isActive"`    // 是否启用
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`

	// 扩展字段（不存储在数据库）
	ParentName    string        `json:"parentName,omitempty"`    // 父部门名称
	ManagerName   string        `json:"managerName,omitempty"`   // 经理姓名
	EmployeeCount int           `json:"employeeCount,omitempty"` // 员工数量
	Children      []*Department `json:"children,omitempty"`      // 子部门列表
}

// IsTopLevel 是否顶级部门
func (d *Department) IsTopLevel() bool {
	return d.ParentID == nil || *d.ParentID == ""
}

// BuildPath 构建部门路径
func (d *Department) BuildPath(parentPath string) {
	if parentPath == "" {
		d.Path = fmt.Sprintf("/%s", d.ID)
	} else {
		d.Path = fmt.Sprintf("%s/%s", parentPath, d.ID)
	}
}

// DepartmentFilter 部门查询过滤器
type DepartmentFilter struct {
	OrgID    string
	ParentID *string // nil表示查询所有，空字符串表示查询顶级部门
	Keyword  string  // 按名称、编码模糊搜索
	IsActive *bool   // 是否启用
	Page     int
	PageSize int
}

// DepartmentListResult 部门列表结果
type DepartmentListResult struct {
	Items    []*Department `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// DepartmentTree 部门树节点
type DepartmentTree struct {
	*Department
	Children []*DepartmentTree `json:"children,omitempty"`
}

// BuildDepartmentTree 构建部门树
func BuildDepartmentTree(departments []*Department) []*DepartmentTree {
	// 创建ID到部门的映射
	deptMap := make(map[string]*DepartmentTree)
	for _, dept := range departments {
		deptMap[dept.ID] = &DepartmentTree{
			Department: dept,
			Children:   make([]*DepartmentTree, 0),
		}
	}

	// 构建树形结构
	var roots []*DepartmentTree
	for _, dept := range departments {
		node := deptMap[dept.ID]
		if dept.IsTopLevel() {
			roots = append(roots, node)
		} else if parent, ok := deptMap[*dept.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}
