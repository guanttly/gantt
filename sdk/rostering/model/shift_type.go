package model

import "time"

// ShiftType 班次类型领域模型
type ShiftType struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"orgId"`
	Code        string     `json:"code"`        // 类型编码（如 regular, overtime）
	Name        string     `json:"name"`        // 类型名称（如 常规班次）
	Description string     `json:"description"` // 类型描述

	// 排班优先级配置
	SchedulingPriority int    `json:"schedulingPriority"` // 排班优先级（1-100，越小越优先）
	WorkflowPhase      string `json:"workflowPhase"`      // 工作流阶段（normal, special, research, fixed, fill）

	// 显示配置
	Color     string `json:"color,omitempty"`     // 显示颜色
	Icon      string `json:"icon,omitempty"`      // 图标名称
	SortOrder int    `json:"sortOrder,omitempty"` // 显示排序

	// 业务配置
	IsAIScheduling      bool `json:"isAiScheduling"`      // 是否需要AI排班
	IsFixedSchedule     bool `json:"isFixedSchedule"`     // 是否固定排班
	IsOvertime          bool `json:"isOvertime"`          // 是否算加班
	RequiresSpecialSkill bool `json:"requiresSpecialSkill"` // 是否需要特殊技能

	// 状态与审计
	IsActive  bool       `json:"isActive"`
	IsSystem  bool       `json:"isSystem"`  // 是否系统内置（不可删除）
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// CreateShiftTypeRequest 创建班次类型请求
type CreateShiftTypeRequest struct {
	OrgID              string `json:"orgId" binding:"required"`
	Code               string `json:"code" binding:"required"`
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	SchedulingPriority int    `json:"schedulingPriority" binding:"min=1,max=100"`
	WorkflowPhase      string `json:"workflowPhase" binding:"required,oneof=normal special research fixed fill"`
	Color              string `json:"color"`
	Icon               string `json:"icon"`
	SortOrder          int    `json:"sortOrder"`
	IsAIScheduling     bool   `json:"isAiScheduling"`
	IsFixedSchedule    bool   `json:"isFixedSchedule"`
	IsOvertime         bool   `json:"isOvertime"`
	RequiresSpecialSkill bool   `json:"requiresSpecialSkill"`
}

// UpdateShiftTypeRequest 更新班次类型请求
type UpdateShiftTypeRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	SchedulingPriority int    `json:"schedulingPriority" binding:"min=1,max=100"`
	WorkflowPhase      string `json:"workflowPhase" binding:"oneof=normal special research fixed fill"`
	Color              string `json:"color"`
	Icon               string `json:"icon"`
	SortOrder          int    `json:"sortOrder"`
	IsAIScheduling     bool   `json:"isAiScheduling"`
	IsFixedSchedule    bool   `json:"isFixedSchedule"`
	IsOvertime         bool   `json:"isOvertime"`
	RequiresSpecialSkill bool   `json:"requiresSpecialSkill"`
	IsActive           bool   `json:"isActive"`
}

// ListShiftTypesRequest 查询班次类型列表请求
type ListShiftTypesRequest struct {
	OrgID          string
	WorkflowPhase  string // 筛选工作流阶段
	IsActive       *bool  // 筛选是否启用
	IncludeSystem  bool   // 是否包含系统内置类型
	Page           int
	PageSize       int
}

// ShiftTypeStats 班次类型统计信息
type ShiftTypeStats struct {
	TypeID      string `json:"typeId"`
	TypeName    string `json:"typeName"`
	ShiftCount  int    `json:"shiftCount"`  // 使用该类型的班次数量
	UsageRate   float64 `json:"usageRate"`   // 使用率
}

// WorkflowPhaseInfo 工作流阶段信息
type WorkflowPhaseInfo struct {
	Phase       string `json:"phase"`       // 阶段标识
	Name        string `json:"name"`        // 阶段名称
	Priority    int    `json:"priority"`    // 阶段优先级
	Description string `json:"description"` // 阶段描述
}

// GetWorkflowPhases 获取所有工作流阶段定义
func GetWorkflowPhases() []WorkflowPhaseInfo {
	return []WorkflowPhaseInfo{
		{Phase: "fixed", Name: "固定班次", Priority: 1, Description: "每周固定人员，无需AI排班"},
		{Phase: "special", Name: "特殊班次", Priority: 3, Description: "有特殊要求，优先排班"},
		{Phase: "normal", Name: "普通班次", Priority: 4, Description: "常规工作班次"},
		{Phase: "research", Name: "科研班次", Priority: 5, Description: "科研或学习时间"},
		{Phase: "fill", Name: "填充班次", Priority: 6, Description: "补充排班不足"},
	}
}

