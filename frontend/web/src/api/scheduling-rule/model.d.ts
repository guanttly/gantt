// 排班规则管理模块相关的类型定义

declare namespace SchedulingRule {
  /** 规则类型（V4 后端值） */
  type RuleType = 'exclusive' | 'combinable' | 'required_together' | 'periodic' | 'maxCount' | 'forbidden_day' | 'preferred'

  /** 应用范围（V4 后端值） */
  type ApplyScope = 'global' | 'specific'

  /** 时间范围（V4 后端值） */
  type TimeScope = 'same_day' | 'same_week' | 'same_month' | 'custom'

  // ============================================================================
  // V3 兼容类型（向后兼容）
  // @deprecated V3: 迁移完成后可移除
  // ============================================================================

  /** V3 规则类型（旧值，用于向后兼容） */
  type V3RuleType = 'max_shifts' | 'consecutive_shifts' | 'rest_days' | 'forbidden_pattern' | 'preferred_pattern'

  /** V3 应用范围（旧值，用于向后兼容） */
  type V3ApplyScope = 'global' | 'group' | 'employee' | 'shift'

  /** V3 时间范围（旧值，用于向后兼容） */
  type V3TimeScope = 'daily' | 'weekly' | 'monthly' | 'custom'

  /** 规则信息（V4 扩展） */
  interface RuleInfo {
    id: string
    orgId: string
    name: string // 规则名称
    ruleType: RuleType
    applyScope: ApplyScope
    timeScope: TimeScope
    timeOffsetDays?: number
    priority: number // 优先级，数字越大优先级越高
    isActive: boolean
    ruleData: string // 规则的语义化描述
    description?: string
    createdAt: string
    updatedAt: string
    // 关联统计信息（列表接口返回）
    associationCount?: number // 总关联数量
    employeeCount?: number // 关联的员工数量
    shiftCount?: number // 关联的班次数量
    groupCount?: number // 关联的分组数量
    // V4新增字段
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string
    sourceType?: 'manual' | 'llm_parsed' | 'migrated'
    parseConfidence?: number
    version?: 'v3' | 'v4'
    // V4.1新增字段
    associations?: AssociationData[] // 规则关联列表（包含班次/员工/分组关联）
    applyScopes?: ApplyScopeInfo[] // 适用范围列表
  }

  /** 规则关联数据（后端返回） */
  interface AssociationData {
    id: string
    ruleId: string
    associationType: 'employee' | 'shift' | 'group' // 关联类型
    associationId: string // 关联对象ID
    role?: RelationRole // 关联角色: subject/object/target
    createdAt?: string
  }

  /** 适用范围信息（V4.1新增） */
  interface ApplyScopeInfo {
    id: string
    ruleId: string
    scopeType: ScopeType
    scopeId?: string
    scopeName?: string
    createdAt?: string
  }

  /** 查询规则列表参数（V4 扩展） */
  interface ListParams {
    orgId: string
    ruleType?: RuleType
    applyScope?: ApplyScope
    timeScope?: TimeScope
    isActive?: boolean
    keyword?: string
    // V4新增筛选字段
    category?: Category
    subCategory?: SubCategory
    sourceType?: 'manual' | 'llm_parsed' | 'migrated'
    version?: 'v3' | 'v4'
    page?: number
    size?: number
  }

  /** 规则列表数据 */
  interface ListData {
    items: RuleInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建规则请求（V4 扩展） */
  interface CreateRequest {
    orgId: string
    name: string
    ruleType: RuleType
    applyScope: ApplyScope
    timeScope: TimeScope
    timeOffsetDays?: number
    priority: number
    ruleData: string // 规则的语义化描述
    description?: string
    // V4新增字段
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string
    sourceType?: 'manual' | 'llm_parsed' | 'migrated'
    parseConfidence?: number
    version?: 'v3' | 'v4'
    // V4.1新增：结构化的关联数据和适用范围
    associations?: AssociationInput[]
    applyScopes?: ApplyScopeInput[]
  }

  /** 关联输入（V4.1新增：统一替代原来的 ShiftRelations） */
  interface AssociationInput {
    associationType: 'employee' | 'shift' | 'group'
    associationId: string
    role?: RelationRole // 角色：subject/object/target 等
  }

  /** 适用范围输入（V4.1新增） */
  interface ApplyScopeInput {
    scopeType: ScopeType // 范围类型: all/employee/group/exclude_employee/exclude_group
    scopeId?: string // 范围对象ID（当scopeType不为all时必填）
    scopeName?: string // 范围对象名称（冗余，便于展示）
  }

  /** 关系角色（V4.1新增） */
  type RelationRole = 'subject' | 'object' | 'target'

  /** 范围类型（V4.1新增） */
  type ScopeType = 'all' | 'employee' | 'group' | 'exclude_employee' | 'exclude_group'

  // RelationType 已废弃，不再使用

  /** 更新规则请求（V4 扩展） */
  interface UpdateRequest {
    name?: string
    priority?: number
    ruleData?: string // 规则的语义化描述
    description?: string
    timeOffsetDays?: number
    // V4新增字段
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string
    sourceType?: 'manual' | 'llm_parsed' | 'migrated'
    parseConfidence?: number
    version?: 'v3' | 'v4'
    // V4.1新增：结构化的关联数据和适用范围
    associations?: AssociationInput[]
    applyScopes?: ApplyScopeInput[]
  }

  // ==================== 规则关联 ====================

  /** 规则关联信息（V4 扩展） */
  interface AssociationInfo {
    ruleId: string
    targetType: 'employee' | 'group' | 'shift'
    targetId: string
    targetName?: string // 关联目标的名称
    role?: 'target' | 'source' | 'reference' // V4新增：关联角色
    createdAt: string
  }

  /** 查询规则关联参数 */
  interface GetAssociationsParams {
    orgId: string
    ruleId: string
  }

  /** 规则关联列表数据 */
  interface AssociationListData {
    associations: AssociationInfo[]
  }

  /** 创建规则关联请求 */
  interface CreateAssociationRequest {
    orgId: string
    ruleId: string
    targetType: 'employee' | 'group' | 'shift'
    targetId: string
    role?: 'target' | 'source' | 'reference' // V4新增：关联角色
  }

  /** 批量创建规则关联请求 */
  interface BatchCreateAssociationRequest {
    orgId: string
    ruleId: string
    associations: Array<{
      targetType: 'employee' | 'group' | 'shift'
      targetId: string
    }>
  }

  /** 批量删除规则关联请求 */
  interface BatchDeleteAssociationRequest {
    orgId: string
    associations: Array<{
      ruleId: string
      targetType: string
      targetId: string
    }>
  }

  /** 按员工查询规则参数 */
  interface RulesByEmployeeParams {
    orgId: string
    employeeId: string
  }

  /** 按班次查询规则参数 */
  interface RulesByShiftParams {
    orgId: string
    shiftId: string
  }

  /** 按分组查询规则参数 */
  interface RulesByGroupParams {
    orgId: string
    groupId: string
  }

  // ==================== 规则验证 ====================

  /** 规则验证请求 */
  interface ValidationRequest {
    orgId: string
    employeeId: string
    shiftId: string
    date: string
  }

  /** 规则验证数据 */
  interface ValidationData {
    isValid: boolean
    violations: Array<{
      ruleId: string
      ruleName: string
      message: string
    }>
  }

  // ==================== V4 规则组织 ====================

  /** 规则分类 */
  type Category = 'constraint' | 'preference' | 'dependency'

  /** 规则子分类 */
  type SubCategory = 'forbid' | 'must' | 'limit' | 'prefer' | 'suggest' | 'source' | 'resource' | 'order'

  /** 规则信息（V4扩展） */
  interface RuleInfoV4 extends RuleInfo {
    category?: Category
    subCategory?: SubCategory
    originalRuleId?: string // 原始规则ID（如果是从语义化规则解析出来的）
  }

  /** 解析规则请求 */
  interface ParseRuleRequest {
    orgId: string
    name: string
    ruleDescription: string // 用户输入的语义化规则
    applyScope?: string // 可选，系统可自动识别
    priority: number
    validFrom?: string
    validTo?: string
  }

  /** 解析后的规则 */
  interface ParsedRule {
    name: string
    category: Category
    subCategory: SubCategory
    ruleType: RuleType
    applyScope: ApplyScope
    timeScope: TimeScope
    timeOffsetDays?: number
    description: string
    ruleData: string
    maxCount?: number
    consecutiveMax?: number
    intervalDays?: number
    minRestDays?: number
    priority: number
    validFrom?: string
    validTo?: string
    associations?: Array<{
      associationType: 'employee' | 'shift' | 'group'
      associationId: string
      role?: 'target' | 'source' | 'reference'
    }>
    dependencies?: string[] // 依赖的其他规则ID
    conflicts?: string[] // 冲突的其他规则ID
    // V4.1新增：结构化的班次关系
    subjectShifts?: string[] // 主体班次名称列表
    objectShifts?: string[] // 客体班次名称列表
    targetShifts?: string[] // 目标班次名称列表（单目标规则）
    // V4.1新增：适用范围
    scopeType?: ScopeType // all/employee/group/exclude_employee/exclude_group
    scopeEmployees?: string[] // 员工名称列表
    scopeGroups?: string[] // 分组名称列表
  }

  /** 规则依赖关系 */
  interface RuleDependency {
    dependentRuleName: string // 被依赖的规则（需要先执行）
    dependentOnRuleName: string // 依赖的规则（后执行）
    dependencyType: 'time' | 'source' | 'resource' | 'order'
    description: string
  }

  /** 规则冲突关系 */
  interface RuleConflict {
    ruleName1: string
    ruleName2: string
    conflictType: 'exclusive' | 'resource' | 'time' | 'frequency'
    description: string
  }

  /** 解析规则响应 */
  interface ParseRuleResponse {
    originalRule: string // 原始规则描述
    parsedRules: ParsedRule[] // 解析后的规则列表
    dependencies: RuleDependency[] // 识别出的依赖关系
    conflicts: RuleConflict[] // 识别出的冲突关系
    reasoning: string // 解析说明
  }

  /** 批量保存规则请求 */
  interface BatchSaveRulesRequest {
    orgId: string
    parsedRules: ParsedRule[]
    dependencies: RuleDependency[]
    conflicts: RuleConflict[]
  }

  /** 批量保存规则响应 */
  interface BatchSaveRulesResponse {
    rules: RuleInfoV4[]
    dependencies: RuleDependency[]
    conflicts: RuleConflict[]
  }

  /** 分类后的规则信息 */
  interface ClassifiedRuleInfo {
    ruleId: string
    ruleName: string
    category: Category
    subCategory: SubCategory
    ruleType: RuleType
    dependencies: string[]
    conflicts: string[]
    priority: number
    description: string
  }

  /** 班次依赖关系信息 */
  interface ShiftDependencyInfo {
    dependentShiftID: string // 被依赖的班次（需要先排）
    dependentOnShiftID: string // 依赖的班次（后排）
    dependencyType: 'time' | 'source' | 'resource'
    ruleID: string // 产生此依赖关系的规则ID
    description: string
  }

  /** 规则依赖关系信息 */
  interface RuleDependencyInfo {
    dependentRuleID: string // 被依赖的规则（需要先执行）
    dependentOnRuleID: string // 依赖的规则（后执行）
    dependencyType: 'time' | 'source' | 'resource' | 'order'
    description: string
  }

  /** 规则冲突关系信息 */
  interface RuleConflictInfo {
    ruleID1: string // 冲突的规则1
    ruleID2: string // 冲突的规则2
    conflictType: 'exclusive' | 'resource' | 'time' | 'frequency'
    description: string
    resolutionPriority: number // 解决优先级
  }

  /** 规则组织结果 */
  interface RuleOrganizationResult {
    constraintRules: ClassifiedRuleInfo[] // 约束型规则
    preferenceRules: ClassifiedRuleInfo[] // 偏好型规则
    dependencyRules: ClassifiedRuleInfo[] // 依赖型规则
    shiftDependencies: ShiftDependencyInfo[] // 班次依赖关系
    ruleDependencies: RuleDependencyInfo[] // 规则依赖关系
    ruleConflicts: RuleConflictInfo[] // 规则冲突关系
    shiftExecutionOrder: string[] // 班次执行顺序（按依赖关系排序）
    ruleExecutionOrder: string[] // 规则执行顺序（按依赖关系排序）
  }
}
