package entity

import (
	"time"
)

// SchedulingRuleEntity 排班规则数据库实体（对应scheduling_rules表）
// 统一的排班规则表，支持全局规则和针对特定对象的规则
// 示例：
// - 全局班次规则：本部穿刺班排他（当天排了就不能再排其他班）
// - 全局班次规则：CT/MRI审核上下午班可合并（同一人可以排上午+下午）
// - 针对员工规则：王晨隔周上夜班（通过关联表关联到王晨）
// - 针对班次规则：夜班最多连续2天（通过关联表关联到夜班班次）
type SchedulingRuleEntity struct {
	ID             string `gorm:"primaryKey;type:varchar(64)"`
	OrgID          string `gorm:"index;type:varchar(64);not null"`
	Name           string `gorm:"type:varchar(128);not null"` // 规则名称："本部穿刺班排他规则"
	Description    string `gorm:"type:text"`                  // 规则说明
	RuleType       string `gorm:"type:varchar(32);not null"`  // 规则类型：exclusive/combinable/required_together/periodic/maxCount/forbidden_day/preferred
	ApplyScope     string `gorm:"type:varchar(32);not null"`  // 作用范围：global（全局）/specific（指定对象，需通过关联表关联）
	TimeScope      string `gorm:"type:varchar(32);not null"`  // 时间范围：same_day（同一天）/same_week（同一周）/same_month（同一月）/custom（自定义）
	TimeOffsetDays *int   `gorm:"type:int"`                   // 时间偏移天数
	RuleData       string `gorm:"type:varchar(512)"`          // 规则说明文本：例如"隔周上夜班"、"不能连续超过2天"等

	// 数值型规则参数（根据规则类型使用）
	MaxCount       *int `gorm:"type:int"` // 最大次数：用于 maxCount 类型，如"每周最多3次"
	ConsecutiveMax *int `gorm:"type:int"` // 连续最大天数：用于 maxCount 类型，如"连续不超过2天"
	IntervalDays   *int `gorm:"type:int"` // 间隔天数：用于 periodic 类型，如"间隔7天"、"隔周=14天"
	MinRestDays    *int `gorm:"type:int"` // 最少休息天数：用于休息规则，如"夜班后至少休息1天"

	Priority  int        `gorm:"default:0"`    // 规则优先级（数字越大优先级越高）
	IsActive  bool       `gorm:"default:true"` // 是否启用
	ValidFrom *time.Time `gorm:"type:date"`    // 规则生效开始日期（可选）
	ValidTo   *time.Time `gorm:"type:date"`    // 规则生效结束日期（可选）
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"`
	DeletedAt *time.Time `gorm:"index"` // 软删除

	// V4新增字段：规则分类
	Category       string `gorm:"type:varchar(32);index"` // 规则分类: constraint/preference/dependency
	SubCategory    string `gorm:"type:varchar(32);index"` // 规则子分类: forbid/must/limit/prefer/suggest/source/resource/order
	OriginalRuleID string `gorm:"type:varchar(64);index"` // 原始规则ID（如果是从语义化规则解析出来的）

	// SourceType 规则来源类型: manual/llm_parsed/migrated
	SourceType string `gorm:"type:varchar(32);index"`

	// ParseConfidence LLM 解析置信度 (0.0-1.0)
	ParseConfidence *float64 `gorm:"type:decimal(3,2)"`

	// Version 规则版本号（V3=空或"v3", V4="v4"）
	Version string `gorm:"type:varchar(8);index;default:'v4'"` // 默认 v4

}

// TableName 表名
func (SchedulingRuleEntity) TableName() string {
	return "scheduling_rules"
}

// SchedulingRuleAssociationEntity 排班规则关联表（对应scheduling_rule_associations表）
// 当规则的 ApplyScope 为 specific 时，通过此表关联具体的员工或班次
// 每个关联一条记录，支持一个规则关联多个对象
type SchedulingRuleAssociationEntity struct {
	ID              string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID           string    `gorm:"index;type:varchar(64);not null"`
	RuleID          string    `gorm:"index;type:varchar(64);not null"`   // 规则ID
	AssociationType string    `gorm:"type:varchar(32);not null;index"`   // 关联类型：employee（员工）/shift（班次）
	AssociationID   string    `gorm:"type:varchar(64);not null;index"`   // 关联对象ID（员工ID或班次ID）
	Role            string    `gorm:"type:varchar(32);default:'target'"` // V4新增：关联角色 target/source/reference
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

// TableName 表名
func (SchedulingRuleAssociationEntity) TableName() string {
	return "scheduling_rule_associations"
}

// RuleDependencyEntity 规则依赖关系表（V4新增）
type RuleDependencyEntity struct {
	ID                string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID             string    `gorm:"index;type:varchar(64);not null"`
	DependentRuleID   string    `gorm:"index;type:varchar(64);not null"` // 被依赖的规则ID（需要先执行）
	DependentOnRuleID string    `gorm:"index;type:varchar(64);not null"` // 依赖的规则ID（后执行）
	DependencyType    string    `gorm:"type:varchar(32);not null"`       // 依赖类型: time/source/resource/order
	Description       string    `gorm:"type:text"`                       // 依赖关系描述
	CreatedAt         time.Time `gorm:"autoCreateTime"`
}

// TableName 表名
func (RuleDependencyEntity) TableName() string {
	return "rule_dependencies"
}

// RuleConflictEntity 规则冲突关系表（V4新增）
type RuleConflictEntity struct {
	ID                 string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID              string    `gorm:"index;type:varchar(64);not null"`
	RuleID1            string    `gorm:"index;type:varchar(64);not null"`
	RuleID2            string    `gorm:"index;type:varchar(64);not null"`
	ConflictType       string    `gorm:"type:varchar(32);not null"` // 冲突类型: exclusive/resource/time/frequency
	Description        string    `gorm:"type:text"`                 // 冲突描述
	ResolutionPriority int       `gorm:"type:int"`                  // 解决优先级（数字越小越优先）
	CreatedAt          time.Time `gorm:"autoCreateTime"`
}

// TableName 表名
func (RuleConflictEntity) TableName() string {
	return "rule_conflicts"
}

// ShiftDependencyEntity 班次依赖关系表（V4新增）
type ShiftDependencyEntity struct {
	ID                 string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID              string    `gorm:"index;type:varchar(64);not null"`
	DependentShiftID   string    `gorm:"index;type:varchar(64);not null"` // 被依赖的班次ID（需要先排）
	DependentOnShiftID string    `gorm:"index;type:varchar(64);not null"` // 依赖的班次ID（后排）
	DependencyType     string    `gorm:"type:varchar(32);not null"`       // 依赖类型: time/source/resource
	RuleID             string    `gorm:"index;type:varchar(64)"`          // 产生此依赖关系的规则ID
	Description        string    `gorm:"type:text"`                       // 依赖关系描述
	CreatedAt          time.Time `gorm:"autoCreateTime"`
}

// TableName 表名
func (ShiftDependencyEntity) TableName() string {
	return "shift_dependencies"
}

// RuleApplyScopeEntity 规则适用范围表（V4.1新增）
// 用于存储规则的适用范围（员工/分组）
// 示例：规则"王晨每周最多3次夜班" -> scope_type:employee, scope_id:王晨的ID
type RuleApplyScopeEntity struct {
	ID        string    `gorm:"primaryKey;type:varchar(64)"`
	OrgID     string    `gorm:"index;type:varchar(64);not null"`
	RuleID    string    `gorm:"index;type:varchar(64);not null"` // 规则ID
	ScopeType string    `gorm:"type:varchar(32);not null;index"` // 范围类型: all/employee/group/exclude_employee/exclude_group
	ScopeID   string    `gorm:"type:varchar(64);index"`          // 范围对象ID
	ScopeName string    `gorm:"type:varchar(128)"`               // 范围对象名称（冗余）
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName 表名
func (RuleApplyScopeEntity) TableName() string {
	return "rule_apply_scopes"
}

// ScopeType 范围类型常量
const (
	ScopeTypeAll             = "all"              // 全局
	ScopeTypeEmployee        = "employee"         // 员工
	ScopeTypeGroup           = "group"            // 分组
	ScopeTypeExcludeEmployee = "exclude_employee" // 排除员工
	ScopeTypeExcludeGroup    = "exclude_group"    // 排除分组
)
