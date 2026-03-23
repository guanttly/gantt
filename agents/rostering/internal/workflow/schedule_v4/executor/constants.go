package executor

// 规则类型常量（与 SDK/API 的 ruleType 取值一致，排班执行器内用于强约束分组与依赖方向）
const (
	RuleTypeRequiredTogether = "required_together" // 必须同排
	RuleTypeExclusive        = "exclusive"         // 互斥
)

// 依赖类型常量（ShiftDependency.DependencyType / RuleDependency.DependencyType）
// 除 time/source/resource/order 外，班次依赖也会用规则类型名表示语义依赖：required_together
const (
	DependencyTypeTime             = "time"     // 时间依赖（前一日/前一周）
	DependencyTypeSource           = "source"   // 人员来源依赖
	DependencyTypeResource         = "resource" // 资源预留依赖
	DependencyTypeOrder            = "order"    // 顺序依赖
	DependencyTypeRequiredTogether = RuleTypeRequiredTogether
)
