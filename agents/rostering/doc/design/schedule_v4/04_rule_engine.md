# 04. 确定性规则引擎详细设计

> **开发负责人**: Agent-2  
> **依赖**: Agent-1 (数据模型)  
> **被依赖**: Agent-3 (V4工作流), Agent-4 (规则解析服务-模拟验证)  
> **包路径**: `agents/rostering/internal/engine/`

## 1. 设计目标

**用代码替代 V3 中 4 个 LLM 调用**，实现 100% 确定性规则计算：

| V3 LLM 调用 | V4 引擎组件 | 输入 | 输出 |
|-------------|-----------|------|------|
| LLM-1 人员过滤 | `CandidateFilter` | Staff + Leaves + Needs | AvailableCandidates |
| LLM-2 规则过滤 | `RuleMatcher` | Rules + ShiftID + Associations | MatchedRules |
| LLM-3 冲突检测 | `ConstraintChecker` | Candidates + Rules + Draft | Eligible/Excluded |
| LLM-5 校验 | `ScheduleValidator` | Schedule + Rules | ValidationResult |
| （新增） | `PreferenceScorer` | Candidates + PreferenceRules | ScoredCandidates |
| （新增） | `DependencyResolver` | Shifts + Dependencies | ExecutionOrder |

## 2. 类型定义

**文件**: `internal/engine/types.go`

```go
package engine

import (
    "time"
    d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 引擎输入类型
// ============================================================

// SchedulingInput 排班引擎输入（单班次+单日期）
type SchedulingInput struct {
    // 基础数据
    OrgID             string
    ShiftID           string
    ShiftName         string
    ShiftStartTime    string              // HH:MM
    ShiftEndTime      string              // HH:MM
    ShiftIsOvernight  bool
    Date              time.Time
    RequiredCount     int                 // 该日该班次需要的人数
    
    // 全量数据（L1 传入）
    AllStaff          []*d_model.Employee
    AllRules          []*d_model.Rule     // 包含 V4 分类字段
    AllShifts         []*d_model.Shift
    
    // 个人需求与请假
    PersonalNeeds     map[string][]*PersonalNeed  // staffID -> needs
    StaffLeaves       map[string][]*d_model.LeaveRecord
    
    // 固定排班
    FixedAssignments  []d_model.CtxFixedShiftAssignment
    
    // 当前排班草稿（全局，用于跨班次检查）
    GlobalDraft       *d_model.ScheduleDraft
    
    // 当前班次的已有排班（该班次已完成的天）
    CurrentShiftDraft *d_model.ShiftScheduleDraft
    
    // 占位信息
    OccupiedSlots     []d_model.StaffOccupiedSlot
    
    // V4 依赖/冲突关系
    RuleDependencies  []*d_model.RuleDependency
    RuleConflicts     []*d_model.RuleConflict
    ShiftDependencies []*d_model.ShiftDependency
}

// PersonalNeed 个人需求（从 create.PersonalNeed 简化）
type PersonalNeed struct {
    StaffID         string
    StaffName       string
    NeedType        string   // permanent / temporary
    RequestType     string   // prefer / avoid / must
    TargetShiftID   string
    TargetDates     []string // YYYY-MM-DD
    Description     string
}

// ============================================================
// 引擎输出类型
// ============================================================

// SchedulingContext 排班上下文（引擎输出，传给 Executor）
type SchedulingContext struct {
    ShiftID            string
    ShiftName          string
    Date               time.Time
    DateStr            string                  // YYYY-MM-DD
    RequiredCount      int
    
    // 规则匹配结果
    MatchedRules       *MatchedRules
    
    // 候选人结果
    EligibleCandidates []*CandidateStatus      // 通过所有硬约束
    ExcludedCandidates []*CandidateStatus      // 被排除的（附原因）
    ExclusionReasons   []*ExclusionRecord       // 所有排除记录
    
    // 约束详情
    ConstraintDetails  []*ConstraintDetail
    
    // 偏好评分
    PreferenceScores   *PreferenceScoreResult
    
    // LLM 摘要（结构化，直接用于构建 Prompt）
    LLMBrief           *LLMBrief
}

// ============================================================
// 规则匹配结果
// ============================================================

// MatchedRules 匹配后的分类规则
type MatchedRules struct {
    ConstraintRules []*ClassifiedRule  // 约束型（必须遵守）
    PreferenceRules []*ClassifiedRule  // 偏好型（尽量满足）
    DependencyRules []*ClassifiedRule  // 依赖型（影响执行顺序）
    AllMatched      []*ClassifiedRule  // 全部匹配的规则
}

// ClassifiedRule 分类后的规则
type ClassifiedRule struct {
    Rule         *d_model.Rule
    Category     d_model.RuleCategory
    SubCategory  d_model.RuleSubCategory
    
    // 该规则的关联对象（按角色分类）
    Targets      []d_model.RuleAssociation  // role="target" 的关联
    Sources      []d_model.RuleAssociation  // role="source" 的关联
    References   []d_model.RuleAssociation  // role="reference" 的关联
}

// ============================================================
// 候选人状态
// ============================================================

// CandidateStatus 候选人状态
type CandidateStatus struct {
    StaffID          string
    StaffName        string
    Groups           []string               // 所属分组
    IsEligible       bool                   // 是否通过所有硬约束
    ViolatedRules    []*RuleViolation       // 违反的硬约束列表
    Warnings         []*RuleWarning         // 警告（接近限制但未违反）
    ConstraintScores map[string]float64     // 各约束的"剩余空间" (0.0~1.0)
    PreferenceScore  float64               // 综合偏好评分 (0.0~1.0)
    
    // 排班状态统计
    WeeklyShiftCount    int                // 本周已排总班次数
    CurrentShiftCount   int                // 当前班次本周已排次数
    ConsecutiveDays     int                // 当前连续排班天数
    LastShiftDate       string             // 最近一次排班日期
}

// RuleViolation 规则违反
type RuleViolation struct {
    RuleID    string
    RuleName  string
    RuleType  string
    IsHard    bool     // true=硬约束违反(排除), false=软约束(警告)
    Message   string
}

// RuleWarning 规则警告
type RuleWarning struct {
    RuleID    string
    RuleName  string
    Message   string
}

// ExclusionRecord 排除记录（可观测性）
type ExclusionRecord struct {
    StaffID    string
    StaffName  string
    Reason     string   // 人可读的排除原因
    Phase      string   // "leave" / "occupied" / "personal_need" / "constraint"
    RuleID     string   // 触发排除的规则ID（如有）
    RuleName   string
}

// ============================================================
// 约束检查详情
// ============================================================

// ConstraintDetail 约束检查详情
type ConstraintDetail struct {
    RuleID       string
    RuleName     string
    RuleType     string
    StaffID      string
    StaffName    string
    Passed       bool
    CurrentValue int      // 当前值（如已排次数）
    LimitValue   int      // 限制值（如最大次数）
    Message      string
}

// ============================================================
// 偏好评分结果
// ============================================================

// PreferenceScoreResult 偏好评分结果
type PreferenceScoreResult struct {
    Scores       map[string]float64    // staffID -> 综合偏好评分
    Details      map[string][]*PreferenceDetail  // staffID -> 各偏好项评分
}

// PreferenceDetail 偏好评分详情
type PreferenceDetail struct {
    RuleID     string
    RuleName   string
    Score      float64  // 0.0~1.0
    Reason     string
}

// ============================================================
// 校验结果
// ============================================================

// ValidationResult 排班校验结果
type ValidationResult struct {
    IsValid      bool               `json:"isValid"`
    Score        float64            `json:"score"`         // 0-100 质量评分
    Violations   []*ValidationItem  `json:"violations"`    // 硬约束违反
    Warnings     []*ValidationItem  `json:"warnings"`      // 软约束/偏好未满足
    Summary      string             `json:"summary"`
}

// GlobalValidationResult 全局校验结果
type GlobalValidationResult struct {
    IsValid       bool               `json:"isValid"`
    OverallScore  float64            `json:"overallScore"`   // 全局质量评分
    ShiftScores   map[string]float64 `json:"shiftScores"`    // 各班次评分
    Violations    []*ValidationItem  `json:"violations"`
    Warnings      []*ValidationItem  `json:"warnings"`
    StaffFairness *FairnessReport    `json:"staffFairness"`  // 公平性报告
    Summary       string             `json:"summary"`
}

// ValidationItem 校验项
type ValidationItem struct {
    RuleID       string   `json:"ruleId"`
    RuleName     string   `json:"ruleName"`
    RuleType     string   `json:"ruleType"`
    Category     string   `json:"category"`
    StaffIDs     []string `json:"staffIds"`
    Date         string   `json:"date"`
    ShiftID      string   `json:"shiftId"`
    Message      string   `json:"message"`
    Severity     string   `json:"severity"`    // error / warning / info
    AutoFixable  bool     `json:"autoFixable"` // 是否可代码自动修复
}

// FairnessReport 公平性报告
type FairnessReport struct {
    StdDeviation float64                   // 排班次数标准差（越小越公平）
    MaxDiff      int                       // 最大差异（最多排和最少排的差值）
    Details      map[string]*StaffFairness // staffID -> 详情
}

// StaffFairness 人员公平性
type StaffFairness struct {
    StaffID       string
    StaffName     string
    TotalShifts   int      // 总排班次数
    NightShifts   int      // 夜班次数
    WeekendShifts int      // 周末排班次数
}

// ============================================================
// LLM 摘要（结构化，直接用于构建 Prompt）
// ============================================================

// LLMBrief 传递给 LLM 的结构化摘要
type LLMBrief struct {
    // 候选人列表（已通过硬约束，附偏好评分）
    Candidates          []*LLMCandidateBrief
    // 硬约束摘要（已验证，告知 LLM 边界）
    HardConstraints     []*LLMConstraintBrief
    // 软偏好摘要
    SoftPreferences     []*LLMPreferenceBrief
    // 排除人员及原因（透明化）
    ExcludedWithReasons []*LLMExclusionBrief
}

// LLMCandidateBrief 候选人摘要
type LLMCandidateBrief struct {
    ShortID          string   // S1, S2, ...
    RealID           string   // 真实ID（不传给LLM，用于结果映射）
    Name             string
    PreferenceScore  float64  // 推荐度 0.0~1.0
    ConstraintMargin float64  // 约束余量 0.0~1.0（越高越宽松）
    WeeklyCount      int      // 本周已排次数
    ConsecutiveDays  int      // 当前连续天数
    Note             string   // 附加说明
}

// LLMConstraintBrief 约束摘要
type LLMConstraintBrief struct {
    RuleShortID  string   // R1, R2, ...
    Description  string   // 简洁描述
    Type         string   // maxCount/consecutiveMax/...
    LimitValue   int      // 限制值
}

// LLMPreferenceBrief 偏好摘要
type LLMPreferenceBrief struct {
    Description  string
    Weight       int     // 权重 1-10
}

// LLMExclusionBrief 排除摘要
type LLMExclusionBrief struct {
    Name    string
    Reason  string
}

// ============================================================
// 依赖解析类型
// ============================================================

// DependencyEdge 依赖边
type DependencyEdge struct {
    From string  // 被依赖的节点（先执行）
    To   string  // 依赖者节点（后执行）
    Type string  // 依赖类型
}

// DependencyGraph 依赖图
type DependencyGraph struct {
    Nodes []string
    Edges []DependencyEdge
}
```

## 3. 引擎入口

**文件**: `internal/engine/engine.go`

```go
package engine

import (
    "context"
    "fmt"
    "time"

    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// IRuleEngine 规则引擎接口
type IRuleEngine interface {
    // PrepareSchedulingContext 为单班次单日期准备排班上下文
    // 替代 V3 的 LLM-1 + LLM-2 + LLM-3
    PrepareSchedulingContext(ctx context.Context, input *SchedulingInput) (*SchedulingContext, error)
    
    // ValidateSchedule 校验单班次单日期排班结果
    // 替代 V3 的 LLM-5
    ValidateSchedule(ctx context.Context, schedule *ScheduleResult, matchedRules *MatchedRules, globalDraft *d_model.ScheduleDraft) (*ValidationResult, error)
    
    // ValidateGlobal 全局校验（所有班次所有日期）
    ValidateGlobal(ctx context.Context, draft *d_model.ScheduleDraft, allRules []*d_model.Rule, allStaff []*d_model.Employee) (*GlobalValidationResult, error)
}

// ScheduleResult LLM 排班结果（用于校验输入）
type ScheduleResult struct {
    ShiftID    string
    Date       string              // YYYY-MM-DD
    StaffIDs   []string            // LLM 选择的人员ID列表
}

// RuleEngine 确定性规则引擎实现
type RuleEngine struct {
    logger             logging.ILogger
    candidateFilter    *CandidateFilter
    ruleMatcher        *RuleMatcher
    constraintChecker  *ConstraintChecker
    preferenceScorer   *PreferenceScorer
    validator          *ScheduleValidator
    dependencyResolver *DependencyResolver
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine(logger logging.ILogger) *RuleEngine {
    l := logger.With("component", "RuleEngine")
    return &RuleEngine{
        logger:             l,
        candidateFilter:    NewCandidateFilter(l),
        ruleMatcher:        NewRuleMatcher(l),
        constraintChecker:  NewConstraintChecker(l),
        preferenceScorer:   NewPreferenceScorer(l),
        validator:          NewScheduleValidator(l),
        dependencyResolver: NewDependencyResolver(l),
    }
}

// PrepareSchedulingContext 为单个班次单个日期准备排班上下文
func (e *RuleEngine) PrepareSchedulingContext(
    ctx context.Context,
    input *SchedulingInput,
) (*SchedulingContext, error) {
    startTime := time.Now()
    dateStr := input.Date.Format("2006-01-02")
    
    // Step 1: 规则匹配（替代 LLM-2）
    matchedRules := e.ruleMatcher.MatchRules(input.AllRules, input.ShiftID, input.Date)
    
    // Step 2: 候选人过滤（替代 LLM-1）
    candidates, exclusionReasons := e.candidateFilter.Filter(
        input.AllStaff,
        input.StaffLeaves,
        input.PersonalNeeds,
        input.FixedAssignments,
        input.OccupiedSlots,
        input.ShiftID,
        dateStr,
    )
    
    // Step 3: 约束检查（替代 LLM-3）
    constraintResult := e.constraintChecker.CheckAll(
        candidates,
        matchedRules,
        input.GlobalDraft,
        input.CurrentShiftDraft,
        input.ShiftID,
        input.Date,
    )
    
    // Step 4: 偏好评分
    preferenceResult := e.preferenceScorer.Score(
        constraintResult.EligibleCandidates,
        matchedRules.PreferenceRules,
        input.GlobalDraft,
        input.Date,
    )
    
    // Step 5: 构建 LLM 摘要
    llmBrief := e.buildLLMBrief(constraintResult, preferenceResult, matchedRules, exclusionReasons)
    
    e.logger.Info("Scheduling context prepared",
        "shiftID", input.ShiftID,
        "date", dateStr,
        "totalStaff", len(input.AllStaff),
        "available", len(candidates),
        "eligible", len(constraintResult.EligibleCandidates),
        "excluded", len(constraintResult.ExcludedCandidates),
        "matchedRules", len(matchedRules.AllMatched),
        "duration", time.Since(startTime),
    )
    
    return &SchedulingContext{
        ShiftID:            input.ShiftID,
        ShiftName:          input.ShiftName,
        Date:               input.Date,
        DateStr:            dateStr,
        RequiredCount:      input.RequiredCount,
        MatchedRules:       matchedRules,
        EligibleCandidates: constraintResult.EligibleCandidates,
        ExcludedCandidates: constraintResult.ExcludedCandidates,
        ExclusionReasons:   exclusionReasons,
        ConstraintDetails:  constraintResult.Details,
        PreferenceScores:   preferenceResult,
        LLMBrief:           llmBrief,
    }, nil
}

// buildLLMBrief 构建 LLM 结构化摘要
func (e *RuleEngine) buildLLMBrief(
    constraints *ConstraintCheckResult,
    preferences *PreferenceScoreResult,
    rules *MatchedRules,
    exclusions []*ExclusionRecord,
) *LLMBrief {
    brief := &LLMBrief{}
    
    // 候选人
    for i, c := range constraints.EligibleCandidates {
        score := 0.5
        if preferences != nil {
            if s, ok := preferences.Scores[c.StaffID]; ok {
                score = s
            }
        }
        // 计算约束余量（所有约束评分的平均值）
        margin := 1.0
        if len(c.ConstraintScores) > 0 {
            total := 0.0
            for _, s := range c.ConstraintScores {
                total += s
            }
            margin = total / float64(len(c.ConstraintScores))
        }
        
        brief.Candidates = append(brief.Candidates, &LLMCandidateBrief{
            ShortID:          fmt.Sprintf("S%d", i+1),
            RealID:           c.StaffID,
            Name:             c.StaffName,
            PreferenceScore:  score,
            ConstraintMargin: margin,
            WeeklyCount:      c.WeeklyShiftCount,
            ConsecutiveDays:  c.ConsecutiveDays,
        })
    }
    
    // 硬约束摘要
    for i, r := range rules.ConstraintRules {
        limit := 0
        if r.Rule.MaxCount != nil {
            limit = *r.Rule.MaxCount
        } else if r.Rule.ConsecutiveMax != nil {
            limit = *r.Rule.ConsecutiveMax
        } else if r.Rule.MinRestDays != nil {
            limit = *r.Rule.MinRestDays
        }
        brief.HardConstraints = append(brief.HardConstraints, &LLMConstraintBrief{
            RuleShortID: fmt.Sprintf("R%d", i+1),
            Description: r.Rule.Name,
            Type:        r.Rule.RuleType,
            LimitValue:  limit,
        })
    }
    
    // 偏好摘要
    for _, r := range rules.PreferenceRules {
        brief.SoftPreferences = append(brief.SoftPreferences, &LLMPreferenceBrief{
            Description: r.Rule.Name + ": " + r.Rule.Description,
            Weight:      r.Rule.Priority,
        })
    }
    
    // 排除摘要
    for _, ex := range exclusions {
        brief.ExcludedWithReasons = append(brief.ExcludedWithReasons, &LLMExclusionBrief{
            Name:   ex.StaffName,
            Reason: ex.Reason,
        })
    }
    
    return brief
}
```

## 4. 候选人过滤器

**文件**: `internal/engine/candidate_filter.go`

```go
package engine

import (
    "time"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// CandidateFilter 候选人过滤器（替代 LLM-1）
type CandidateFilter struct {
    logger logging.ILogger
}

func NewCandidateFilter(logger logging.ILogger) *CandidateFilter {
    return &CandidateFilter{logger: logger.With("sub", "CandidateFilter")}
}

// Filter 过滤候选人
// 返回: (可用候选人列表, 排除记录列表)
func (f *CandidateFilter) Filter(
    allStaff []*d_model.Employee,
    leaves map[string][]*d_model.LeaveRecord,
    personalNeeds map[string][]*PersonalNeed,
    fixedAssignments []d_model.CtxFixedShiftAssignment,
    occupiedSlots []d_model.StaffOccupiedSlot,
    shiftID string,
    dateStr string,
) ([]*d_model.Employee, []*ExclusionRecord) {
    candidates := make([]*d_model.Employee, 0, len(allStaff))
    exclusions := make([]*ExclusionRecord, 0)
    
    for _, staff := range allStaff {
        excluded := false
        
        // 检查1: 请假
        if f.hasLeaveOnDate(leaves[staff.ID], dateStr) {
            exclusions = append(exclusions, &ExclusionRecord{
                StaffID:   staff.ID,
                StaffName: staff.Name,
                Reason:    "请假",
                Phase:     "leave",
            })
            excluded = true
        }
        
        // 检查2: 个人需求（avoid 类型）
        if !excluded {
            if reason := f.hasAvoidNeed(personalNeeds[staff.ID], shiftID, dateStr); reason != "" {
                exclusions = append(exclusions, &ExclusionRecord{
                    StaffID:   staff.ID,
                    StaffName: staff.Name,
                    Reason:    reason,
                    Phase:     "personal_need",
                })
                excluded = true
            }
        }
        
        // 检查3: 已在该日期该班次有固定排班（不需要动态分配）
        if !excluded {
            if f.isFixedAssigned(fixedAssignments, staff.ID, shiftID, dateStr) {
                // 固定排班人员不算排除，但也不进入候选池
                excluded = true
            }
        }
        
        // 检查4: 已在该日期被其他班次占位（时段冲突由 ConstraintChecker 处理）
        // 注意：这里不做时段冲突检查，只排除 "完全被占位" 的情况
        // 时段重叠但不完全冲突的情况留给 ConstraintChecker
        
        if !excluded {
            candidates = append(candidates, staff)
        }
    }
    
    f.logger.Debug("Candidate filter completed",
        "total", len(allStaff),
        "available", len(candidates),
        "excluded", len(exclusions),
        "date", dateStr,
    )
    
    return candidates, exclusions
}

// hasLeaveOnDate 检查员工在指定日期是否请假
func (f *CandidateFilter) hasLeaveOnDate(leaves []*d_model.LeaveRecord, dateStr string) bool {
    if len(leaves) == 0 {
        return false
    }
    targetDate, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        return false
    }
    for _, leave := range leaves {
        if leave == nil {
            continue
        }
        startDate, _ := time.Parse("2006-01-02", leave.StartDate)
        endDate, _ := time.Parse("2006-01-02", leave.EndDate)
        if !targetDate.Before(startDate) && !targetDate.After(endDate) {
            return true
        }
    }
    return false
}

// hasAvoidNeed 检查员工是否有回避需求
func (f *CandidateFilter) hasAvoidNeed(needs []*PersonalNeed, shiftID, dateStr string) string {
    for _, need := range needs {
        if need == nil || need.RequestType != "avoid" {
            continue
        }
        // 检查日期匹配
        for _, d := range need.TargetDates {
            if d == dateStr {
                // 检查班次匹配（空表示所有班次）
                if need.TargetShiftID == "" || need.TargetShiftID == shiftID {
                    return need.Description
                }
            }
        }
    }
    return ""
}

// isFixedAssigned 检查员工是否已固定排班
func (f *CandidateFilter) isFixedAssigned(
    assignments []d_model.CtxFixedShiftAssignment,
    staffID, shiftID, dateStr string,
) bool {
    for _, fa := range assignments {
        if fa.ShiftID == shiftID && fa.Date == dateStr {
            for _, sid := range fa.StaffIDs {
                if sid == staffID {
                    return true
                }
            }
        }
    }
    return false
}
```

## 5. 规则匹配器

**文件**: `internal/engine/rule_matcher.go`

```go
package engine

import (
    "time"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// RuleMatcher 规则匹配器（替代 LLM-2）
type RuleMatcher struct {
    logger logging.ILogger
}

func NewRuleMatcher(logger logging.ILogger) *RuleMatcher {
    return &RuleMatcher{logger: logger.With("sub", "RuleMatcher")}
}

// MatchRules 匹配与指定班次相关的规则，并按分类组织
func (m *RuleMatcher) MatchRules(
    allRules []*d_model.Rule,
    shiftID string,
    date time.Time,
) *MatchedRules {
    result := &MatchedRules{
        ConstraintRules: make([]*ClassifiedRule, 0),
        PreferenceRules: make([]*ClassifiedRule, 0),
        DependencyRules: make([]*ClassifiedRule, 0),
        AllMatched:      make([]*ClassifiedRule, 0),
    }
    
    for _, rule := range allRules {
        if rule == nil || !rule.IsActive {
            continue
        }
        
        // 检查规则有效期
        if !m.isRuleValidOnDate(rule, date) {
            continue
        }
        
        // 检查规则是否与当前班次相关
        if !m.isRuleRelatedToShift(rule, shiftID) {
            continue
        }
        
        // 分类规则
        classified := m.classifyRule(rule)
        result.AllMatched = append(result.AllMatched, classified)
        
        switch d_model.RuleCategory(classified.Category) {
        case d_model.RuleCategoryConstraint:
            result.ConstraintRules = append(result.ConstraintRules, classified)
        case d_model.RuleCategoryPreference:
            result.PreferenceRules = append(result.PreferenceRules, classified)
        case d_model.RuleCategoryDependency:
            result.DependencyRules = append(result.DependencyRules, classified)
        default:
            // 未分类的规则（V3遗留）默认当约束处理
            result.ConstraintRules = append(result.ConstraintRules, classified)
        }
    }
    
    m.logger.Debug("Rule matching completed",
        "shiftID", shiftID,
        "totalRules", len(allRules),
        "matched", len(result.AllMatched),
        "constraints", len(result.ConstraintRules),
        "preferences", len(result.PreferenceRules),
        "dependencies", len(result.DependencyRules),
    )
    
    return result
}

// isRuleValidOnDate 检查规则在指定日期是否有效
func (m *RuleMatcher) isRuleValidOnDate(rule *d_model.Rule, date time.Time) bool {
    if rule.ValidFrom != nil && date.Before(*rule.ValidFrom) {
        return false
    }
    if rule.ValidTo != nil && date.After(*rule.ValidTo) {
        return false
    }
    return true
}

// isRuleRelatedToShift 检查规则是否与指定班次相关
// 匹配逻辑:
// 1. 无 Associations → 全局规则，匹配所有班次
// 2. 有 Associations → 检查是否包含 shiftID (任意 role)
// 3. ApplyScope=="global" → 匹配所有班次
func (m *RuleMatcher) isRuleRelatedToShift(rule *d_model.Rule, shiftID string) bool {
    // 全局规则
    if rule.ApplyScope == "global" {
        return true
    }
    
    // 无关联 → 全局
    if len(rule.Associations) == 0 {
        return true
    }
    
    // 检查 Associations 是否包含该班次
    for _, assoc := range rule.Associations {
        if assoc.AssociationType == "shift" && assoc.AssociationID == shiftID {
            return true
        }
    }
    
    return false
}

// classifyRule 分类规则
func (m *RuleMatcher) classifyRule(rule *d_model.Rule) *ClassifiedRule {
    classified := &ClassifiedRule{
        Rule:       rule,
        Targets:    make([]d_model.RuleAssociation, 0),
        Sources:    make([]d_model.RuleAssociation, 0),
        References: make([]d_model.RuleAssociation, 0),
    }
    
    // 使用 V4 字段（如果有）
    if rule.Category != "" {
        classified.Category = d_model.RuleCategory(rule.Category)
        classified.SubCategory = d_model.RuleSubCategory(rule.SubCategory)
    } else {
        // V3 兼容：根据 RuleType 推断分类
        classified.Category, classified.SubCategory = d_model.RuleTypeToDefaultCategory(rule.RuleType)
    }
    
    // 按 Role 分类 Associations
    for _, assoc := range rule.Associations {
        role := assoc.Role
        if role == "" {
            role = "target" // V3 默认
        }
        switch d_model.AssociationRole(role) {
        case d_model.AssociationRoleTarget:
            classified.Targets = append(classified.Targets, assoc)
        case d_model.AssociationRoleSource:
            classified.Sources = append(classified.Sources, assoc)
        case d_model.AssociationRoleReference:
            classified.References = append(classified.References, assoc)
        default:
            classified.Targets = append(classified.Targets, assoc)
        }
    }
    
    return classified
}
```

## 6. 约束检查器

**文件**: `internal/engine/constraint_checker.go`

```go
package engine

import (
    "fmt"
    "time"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// ConstraintChecker 约束检查器（替代 LLM-3）
type ConstraintChecker struct {
    logger logging.ILogger
}

func NewConstraintChecker(logger logging.ILogger) *ConstraintChecker {
    return &ConstraintChecker{logger: logger.With("sub", "ConstraintChecker")}
}

// ConstraintCheckResult 约束检查结果
type ConstraintCheckResult struct {
    EligibleCandidates []*CandidateStatus
    ExcludedCandidates []*CandidateStatus
    Details            []*ConstraintDetail
}

// CheckAll 检查所有候选人的所有约束
func (c *ConstraintChecker) CheckAll(
    candidates []*d_model.Employee,
    rules *MatchedRules,
    globalDraft *d_model.ScheduleDraft,
    shiftDraft *d_model.ShiftScheduleDraft,
    shiftID string,
    date time.Time,
) *ConstraintCheckResult {
    result := &ConstraintCheckResult{
        EligibleCandidates: make([]*CandidateStatus, 0),
        ExcludedCandidates: make([]*CandidateStatus, 0),
        Details:            make([]*ConstraintDetail, 0),
    }
    
    dateStr := date.Format("2006-01-02")
    
    for _, staff := range candidates {
        status := &CandidateStatus{
            StaffID:          staff.ID,
            StaffName:        staff.Name,
            IsEligible:       true,
            ViolatedRules:    make([]*RuleViolation, 0),
            Warnings:         make([]*RuleWarning, 0),
            ConstraintScores: make(map[string]float64),
        }
        
        // 提取分组名称
        if len(staff.Groups) > 0 {
            for _, g := range staff.Groups {
                if g != nil {
                    status.Groups = append(status.Groups, g.Name)
                }
            }
        }
        
        // 计算排班统计
        c.computeScheduleStats(status, globalDraft, shiftDraft, shiftID, date)
        
        // 逐条检查约束
        for _, rule := range rules.ConstraintRules {
            violation := c.checkSingleConstraint(staff, rule, globalDraft, shiftDraft, shiftID, date)
            if violation != nil {
                status.IsEligible = false
                status.ViolatedRules = append(status.ViolatedRules, violation)
                result.Details = append(result.Details, &ConstraintDetail{
                    RuleID:    rule.Rule.ID,
                    RuleName:  rule.Rule.Name,
                    RuleType:  rule.Rule.RuleType,
                    StaffID:   staff.ID,
                    StaffName: staff.Name,
                    Passed:    false,
                    Message:   violation.Message,
                })
            }
            
            // 计算约束余量评分
            score := c.computeConstraintScore(staff, rule, globalDraft, shiftDraft, shiftID, date)
            status.ConstraintScores[rule.Rule.ID] = score
            
            // 接近限制的警告
            if score > 0 && score < 0.3 {
                status.Warnings = append(status.Warnings, &RuleWarning{
                    RuleID:   rule.Rule.ID,
                    RuleName: rule.Rule.Name,
                    Message:  fmt.Sprintf("约束余量仅 %.0f%%", score*100),
                })
            }
        }
        
        if status.IsEligible {
            result.EligibleCandidates = append(result.EligibleCandidates, status)
        } else {
            result.ExcludedCandidates = append(result.ExcludedCandidates, status)
        }
    }
    
    c.logger.Debug("Constraint check completed",
        "date", dateStr,
        "shiftID", shiftID,
        "candidates", len(candidates),
        "eligible", len(result.EligibleCandidates),
        "excluded", len(result.ExcludedCandidates),
    )
    
    return result
}

// checkSingleConstraint 检查单个约束
func (c *ConstraintChecker) checkSingleConstraint(
    staff *d_model.Employee,
    rule *ClassifiedRule,
    globalDraft *d_model.ScheduleDraft,
    shiftDraft *d_model.ShiftScheduleDraft,
    shiftID string,
    date time.Time,
) *RuleViolation {
    switch rule.Rule.RuleType {
    case "maxCount":
        return c.checkMaxCount(staff.ID, rule, globalDraft, shiftID, date)
    case "consecutiveMax":
        return c.checkConsecutiveMax(staff.ID, rule, globalDraft, shiftDraft, shiftID, date)
    case "minRestDays":
        return c.checkMinRestDays(staff.ID, rule, globalDraft, shiftID, date)
    case "exclusive":
        return c.checkExclusive(staff.ID, rule, globalDraft, shiftID, date)
    case "forbidden_day":
        return c.checkForbiddenDay(staff.ID, rule, date)
    case "required_together":
        // required_together 在校验阶段检查，排班阶段不排除候选人
        return nil
    default:
        return nil
    }
}

// checkMaxCount 检查最大次数约束
func (c *ConstraintChecker) checkMaxCount(
    staffID string, rule *ClassifiedRule,
    draft *d_model.ScheduleDraft, shiftID string, date time.Time,
) *RuleViolation {
    if rule.Rule.MaxCount == nil {
        return nil
    }
    maxCount := *rule.Rule.MaxCount
    startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)
    
    currentCount := c.countStaffShiftInRange(draft, staffID, shiftID, startDate, endDate)
    
    if currentCount >= maxCount {
        return &RuleViolation{
            RuleID:   rule.Rule.ID,
            RuleName: rule.Rule.Name,
            RuleType: "maxCount",
            IsHard:   true,
            Message:  fmt.Sprintf("已达最大次数限制 %d/%d", currentCount, maxCount),
        }
    }
    return nil
}

// checkConsecutiveMax 检查连续天数约束
func (c *ConstraintChecker) checkConsecutiveMax(
    staffID string, rule *ClassifiedRule,
    globalDraft *d_model.ScheduleDraft, shiftDraft *d_model.ShiftScheduleDraft,
    shiftID string, date time.Time,
) *RuleViolation {
    if rule.Rule.ConsecutiveMax == nil {
        return nil
    }
    maxConsecutive := *rule.Rule.ConsecutiveMax
    
    // 从当前日期往前回溯，计算连续排班天数
    consecutiveDays := 0
    checkDate := date.AddDate(0, 0, -1)
    for i := 0; i < maxConsecutive+1; i++ {
        if c.hasStaffShiftOnDate(globalDraft, shiftDraft, staffID, shiftID, checkDate) {
            consecutiveDays++
            checkDate = checkDate.AddDate(0, 0, -1)
        } else {
            break
        }
    }
    
    if consecutiveDays >= maxConsecutive {
        return &RuleViolation{
            RuleID:   rule.Rule.ID,
            RuleName: rule.Rule.Name,
            RuleType: "consecutiveMax",
            IsHard:   true,
            Message:  fmt.Sprintf("已连续排班%d天，达到上限%d天", consecutiveDays, maxConsecutive),
        }
    }
    return nil
}

// checkMinRestDays 检查最少休息天数
func (c *ConstraintChecker) checkMinRestDays(
    staffID string, rule *ClassifiedRule,
    draft *d_model.ScheduleDraft, shiftID string, date time.Time,
) *RuleViolation {
    if rule.Rule.MinRestDays == nil {
        return nil
    }
    minRest := *rule.Rule.MinRestDays
    
    // 检查前 minRest 天内是否有关联班次排班
    relatedShiftIDs := c.getRelatedShiftIDs(rule)
    if len(relatedShiftIDs) == 0 {
        relatedShiftIDs = []string{shiftID}
    }
    
    for i := 1; i <= minRest; i++ {
        checkDate := date.AddDate(0, 0, -i)
        for _, relShiftID := range relatedShiftIDs {
            if c.hasStaffInDraft(draft, staffID, relShiftID, checkDate) {
                return &RuleViolation{
                    RuleID:   rule.Rule.ID,
                    RuleName: rule.Rule.Name,
                    RuleType: "minRestDays",
                    IsHard:   true,
                    Message:  fmt.Sprintf("距上次排班仅%d天，需至少休息%d天", i, minRest),
                }
            }
        }
    }
    return nil
}

// checkExclusive 检查排他约束
func (c *ConstraintChecker) checkExclusive(
    staffID string, rule *ClassifiedRule,
    draft *d_model.ScheduleDraft, shiftID string, date time.Time,
) *RuleViolation {
    // 获取排他的班次（role=reference 或 role=target 中排除当前班次的）
    exclusiveShiftIDs := make([]string, 0)
    for _, ref := range rule.References {
        if ref.AssociationType == "shift" && ref.AssociationID != shiftID {
            exclusiveShiftIDs = append(exclusiveShiftIDs, ref.AssociationID)
        }
    }
    for _, tgt := range rule.Targets {
        if tgt.AssociationType == "shift" && tgt.AssociationID != shiftID {
            exclusiveShiftIDs = append(exclusiveShiftIDs, tgt.AssociationID)
        }
    }
    
    for _, exShiftID := range exclusiveShiftIDs {
        if c.hasStaffInDraft(draft, staffID, exShiftID, date) {
            return &RuleViolation{
                RuleID:   rule.Rule.ID,
                RuleName: rule.Rule.Name,
                RuleType: "exclusive",
                IsHard:   true,
                Message:  "同日已排排他班次",
            }
        }
    }
    return nil
}

// checkForbiddenDay 检查禁止日期
func (c *ConstraintChecker) checkForbiddenDay(
    staffID string, rule *ClassifiedRule, date time.Time,
) *RuleViolation {
    weekday := date.Weekday()
    // 从 RuleData 中解析禁止日期（简化处理，实际需更完善的解析）
    // TODO: 根据实际 RuleData 格式实现
    _ = weekday
    _ = staffID
    return nil
}

// computeConstraintScore 计算约束剩余空间评分 (0.0~1.0)
func (c *ConstraintChecker) computeConstraintScore(
    staff *d_model.Employee, rule *ClassifiedRule,
    draft *d_model.ScheduleDraft, shiftDraft *d_model.ShiftScheduleDraft,
    shiftID string, date time.Time,
) float64 {
    switch rule.Rule.RuleType {
    case "maxCount":
        if rule.Rule.MaxCount == nil {
            return 1.0
        }
        max := float64(*rule.Rule.MaxCount)
        startDate, endDate := c.getTimeScopeRange(rule.Rule.TimeScope, date)
        current := float64(c.countStaffShiftInRange(draft, staff.ID, shiftID, startDate, endDate))
        if max == 0 {
            return 0.0
        }
        return (max - current) / max
    case "consecutiveMax":
        if rule.Rule.ConsecutiveMax == nil {
            return 1.0
        }
        max := float64(*rule.Rule.ConsecutiveMax)
        // 简化：计算连续天数占比
        return 1.0 - 0.0 // TODO: 实际计算
    default:
        return 1.0
    }
}

// computeScheduleStats 计算排班统计
func (c *ConstraintChecker) computeScheduleStats(
    status *CandidateStatus,
    globalDraft *d_model.ScheduleDraft,
    shiftDraft *d_model.ShiftScheduleDraft,
    shiftID string,
    date time.Time,
) {
    if globalDraft == nil {
        return
    }
    // 计算本周已排总次数
    weekStart, weekEnd := c.getTimeScopeRange("same_week", date)
    status.WeeklyShiftCount = c.countAllShiftsInRange(globalDraft, status.StaffID, weekStart, weekEnd)
    status.CurrentShiftCount = c.countStaffShiftInRange(globalDraft, status.StaffID, shiftID, weekStart, weekEnd)
    
    // 计算连续天数
    consecutiveDays := 0
    checkDate := date.AddDate(0, 0, -1)
    for {
        found := false
        if globalDraft.Shifts != nil {
            for _, sd := range globalDraft.Shifts {
                if sd != nil && sd.Schedule != nil {
                    dateStr := checkDate.Format("2006-01-02")
                    if staffIDs, ok := sd.Schedule[dateStr]; ok {
                        for _, sid := range staffIDs {
                            if sid == status.StaffID {
                                found = true
                                break
                            }
                        }
                    }
                }
                if found { break }
            }
        }
        if found {
            consecutiveDays++
            checkDate = checkDate.AddDate(0, 0, -1)
        } else {
            break
        }
    }
    status.ConsecutiveDays = consecutiveDays
}

// ============================================================
// 辅助方法（数据查询）
// ============================================================

func (c *ConstraintChecker) getTimeScopeRange(timeScope string, date time.Time) (time.Time, time.Time) {
    switch timeScope {
    case "same_day":
        return date, date
    case "same_week":
        weekday := int(date.Weekday())
        if weekday == 0 { weekday = 7 }
        start := date.AddDate(0, 0, -(weekday - 1))
        end := start.AddDate(0, 0, 6)
        return start, end
    case "same_month":
        start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
        end := start.AddDate(0, 1, -1)
        return start, end
    default:
        // 默认同周
        weekday := int(date.Weekday())
        if weekday == 0 { weekday = 7 }
        start := date.AddDate(0, 0, -(weekday - 1))
        end := start.AddDate(0, 0, 6)
        return start, end
    }
}

func (c *ConstraintChecker) countStaffShiftInRange(
    draft *d_model.ScheduleDraft, staffID, shiftID string,
    startDate, endDate time.Time,
) int {
    if draft == nil || draft.Shifts == nil {
        return 0
    }
    shiftDraft, ok := draft.Shifts[shiftID]
    if !ok || shiftDraft == nil || shiftDraft.Schedule == nil {
        return 0
    }
    count := 0
    for dateStr, staffIDs := range shiftDraft.Schedule {
        d, err := time.Parse("2006-01-02", dateStr)
        if err != nil { continue }
        if !d.Before(startDate) && !d.After(endDate) {
            for _, sid := range staffIDs {
                if sid == staffID {
                    count++
                }
            }
        }
    }
    return count
}

func (c *ConstraintChecker) countAllShiftsInRange(
    draft *d_model.ScheduleDraft, staffID string,
    startDate, endDate time.Time,
) int {
    if draft == nil || draft.Shifts == nil { return 0 }
    count := 0
    for _, shiftDraft := range draft.Shifts {
        if shiftDraft == nil || shiftDraft.Schedule == nil { continue }
        for dateStr, staffIDs := range shiftDraft.Schedule {
            d, err := time.Parse("2006-01-02", dateStr)
            if err != nil { continue }
            if !d.Before(startDate) && !d.After(endDate) {
                for _, sid := range staffIDs {
                    if sid == staffID { count++ }
                }
            }
        }
    }
    return count
}

func (c *ConstraintChecker) hasStaffShiftOnDate(
    globalDraft *d_model.ScheduleDraft, shiftDraft *d_model.ShiftScheduleDraft,
    staffID, shiftID string, date time.Time,
) bool {
    dateStr := date.Format("2006-01-02")
    // 先查全局草稿
    if c.hasStaffInDraft(globalDraft, staffID, shiftID, date) {
        return true
    }
    // 再查当前班次草稿
    if shiftDraft != nil && shiftDraft.Schedule != nil {
        if staffIDs, ok := shiftDraft.Schedule[dateStr]; ok {
            for _, sid := range staffIDs {
                if sid == staffID { return true }
            }
        }
    }
    return false
}

func (c *ConstraintChecker) hasStaffInDraft(
    draft *d_model.ScheduleDraft, staffID, shiftID string, date time.Time,
) bool {
    if draft == nil || draft.Shifts == nil { return false }
    dateStr := date.Format("2006-01-02")
    if sd, ok := draft.Shifts[shiftID]; ok && sd != nil && sd.Schedule != nil {
        if staffIDs, ok := sd.Schedule[dateStr]; ok {
            for _, sid := range staffIDs {
                if sid == staffID { return true }
            }
        }
    }
    return false
}

func (c *ConstraintChecker) getRelatedShiftIDs(rule *ClassifiedRule) []string {
    ids := make([]string, 0)
    for _, a := range rule.Targets {
        if a.AssociationType == "shift" { ids = append(ids, a.AssociationID) }
    }
    for _, a := range rule.Sources {
        if a.AssociationType == "shift" { ids = append(ids, a.AssociationID) }
    }
    return ids
}
```

## 7. 依赖解析器

**文件**: `internal/engine/dependency_resolver.go`

```go
package engine

import (
    "fmt"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// DependencyResolver 依赖解析器
type DependencyResolver struct {
    logger logging.ILogger
}

func NewDependencyResolver(logger logging.ILogger) *DependencyResolver {
    return &DependencyResolver{logger: logger.With("sub", "DependencyResolver")}
}

// IDependencyResolver 依赖解析接口
type IDependencyResolver interface {
    ResolveShiftOrder(shifts []*d_model.Shift, deps []*d_model.ShiftDependency) ([]string, error)
    DetectCircularDependency(edges []DependencyEdge) ([][]string, error)
}

// ResolveShiftOrder 计算班次执行顺序（拓扑排序）
// 返回按依赖关系排序后的班次ID列表
// 无依赖关系的班次按 SchedulingPriority 排序
func (r *DependencyResolver) ResolveShiftOrder(
    shifts []*d_model.Shift,
    deps []*d_model.ShiftDependency,
) ([]string, error) {
    // 构建节点集合
    nodes := make([]string, 0, len(shifts))
    priorityMap := make(map[string]int)
    for _, s := range shifts {
        nodes = append(nodes, s.ID)
        priorityMap[s.ID] = s.SchedulingPriority
    }
    
    // 构建边集合
    edges := make([]DependencyEdge, 0, len(deps))
    for _, dep := range deps {
        edges = append(edges, DependencyEdge{
            From: dep.DependentShiftID,  // 被依赖的（先排）
            To:   dep.DependsOnShiftID,  // 依赖者（后排）
            Type: dep.DependencyType,
        })
    }
    
    // 拓扑排序
    sorted, err := topologicalSort(nodes, edges, priorityMap)
    if err != nil {
        return nil, fmt.Errorf("班次依赖存在循环: %w", err)
    }
    
    r.logger.Info("Shift execution order resolved",
        "totalShifts", len(shifts),
        "dependencies", len(deps),
        "order", sorted,
    )
    
    return sorted, nil
}

// DetectCircularDependency 检测循环依赖，返回所有环路
func (r *DependencyResolver) DetectCircularDependency(edges []DependencyEdge) ([][]string, error) {
    // 使用 DFS 检测环
    graph := make(map[string][]string)
    for _, e := range edges {
        graph[e.From] = append(graph[e.From], e.To)
    }
    
    visited := make(map[string]int) // 0=未访问, 1=访问中, 2=已完成
    cycles := make([][]string, 0)
    path := make([]string, 0)
    
    var dfs func(node string)
    dfs = func(node string) {
        visited[node] = 1
        path = append(path, node)
        
        for _, next := range graph[node] {
            if visited[next] == 1 {
                // 找到环
                cycleStart := -1
                for i, n := range path {
                    if n == next {
                        cycleStart = i
                        break
                    }
                }
                if cycleStart >= 0 {
                    cycle := make([]string, len(path)-cycleStart)
                    copy(cycle, path[cycleStart:])
                    cycles = append(cycles, cycle)
                }
            } else if visited[next] == 0 {
                dfs(next)
            }
        }
        
        path = path[:len(path)-1]
        visited[node] = 2
    }
    
    for node := range graph {
        if visited[node] == 0 {
            dfs(node)
        }
    }
    
    return cycles, nil
}

// topologicalSort 拓扑排序（Kahn 算法）
// 同层级节点按 priorityMap 排序
func topologicalSort(nodes []string, edges []DependencyEdge, priorityMap map[string]int) ([]string, error) {
    inDegree := make(map[string]int)
    graph := make(map[string][]string)
    
    // 初始化入度
    for _, node := range nodes {
        inDegree[node] = 0
    }
    
    // 构建图
    for _, edge := range edges {
        graph[edge.From] = append(graph[edge.From], edge.To)
        inDegree[edge.To]++
    }
    
    // 初始队列：入度为0的节点（按优先级排序）
    queue := make([]string, 0)
    for _, node := range nodes {
        if inDegree[node] == 0 {
            queue = append(queue, node)
        }
    }
    sortByPriority(queue, priorityMap)
    
    result := make([]string, 0, len(nodes))
    for len(queue) > 0 {
        // 取队首
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)
        
        // 更新邻居入度
        newNodes := make([]string, 0)
        for _, neighbor := range graph[node] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                newNodes = append(newNodes, neighbor)
            }
        }
        // 同层按优先级排序后加入队列
        sortByPriority(newNodes, priorityMap)
        queue = append(queue, newNodes...)
    }
    
    // 检查是否有环
    if len(result) != len(nodes) {
        return nil, fmt.Errorf("存在循环依赖，已排序 %d/%d 个节点", len(result), len(nodes))
    }
    
    return result, nil
}

// sortByPriority 按 SchedulingPriority 升序排序
func sortByPriority(ids []string, priorityMap map[string]int) {
    // 简单插入排序（数量少）
    for i := 1; i < len(ids); i++ {
        key := ids[i]
        j := i - 1
        for j >= 0 && priorityMap[ids[j]] > priorityMap[key] {
            ids[j+1] = ids[j]
            j--
        }
        ids[j+1] = key
    }
}
```

## 8. 排班校验器

**文件**: `internal/engine/schedule_validator.go`

```go
package engine

import (
    "fmt"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// ScheduleValidator 确定性排班校验器（替代 LLM-5）
type ScheduleValidator struct {
    logger  logging.ILogger
    checker *ConstraintChecker
}

func NewScheduleValidator(logger logging.ILogger) *ScheduleValidator {
    return &ScheduleValidator{
        logger:  logger.With("sub", "ScheduleValidator"),
        checker: NewConstraintChecker(logger),
    }
}

// Validate 校验单班次排班结果
func (v *ScheduleValidator) Validate(
    result *ScheduleResult,
    rules *MatchedRules,
    globalDraft *d_model.ScheduleDraft,
) *ValidationResult {
    vr := &ValidationResult{
        IsValid:    true,
        Score:      100.0,
        Violations: make([]*ValidationItem, 0),
        Warnings:   make([]*ValidationItem, 0),
    }
    
    // 1. 检查人数约束
    // （由调用方在外部检查 RequiredCount vs len(StaffIDs)）
    
    // 2. 检查每个被选人的硬约束
    for _, staffID := range result.StaffIDs {
        for _, rule := range rules.ConstraintRules {
            // 复用 ConstraintChecker 的单条规则检查
            // 这里检查的是"假如该人员被安排到该班次"后是否违规
            // 需要在 globalDraft 中临时添加该人员后检查
            _ = staffID
            _ = rule
            // TODO: 实现详细校验（与 ConstraintChecker 复用逻辑）
        }
    }
    
    // 3. 检查重复分配（同一人是否出现多次）
    seen := make(map[string]bool)
    for _, staffID := range result.StaffIDs {
        if seen[staffID] {
            vr.IsValid = false
            vr.Violations = append(vr.Violations, &ValidationItem{
                Message:  fmt.Sprintf("人员 %s 重复分配", staffID),
                Severity: "error",
            })
            vr.Score -= 20
        }
        seen[staffID] = true
    }
    
    // 4. 检查偏好满足度
    for _, rule := range rules.PreferenceRules {
        // 偏好未满足不影响 IsValid，只影响评分
        _ = rule
    }
    
    if vr.Score < 0 {
        vr.Score = 0
    }
    vr.Summary = v.buildSummary(vr)
    return vr
}

// ValidateGlobal 全局校验
func (v *ScheduleValidator) ValidateGlobal(
    draft *d_model.ScheduleDraft,
    allRules []*d_model.Rule,
    allStaff []*d_model.Employee,
) *GlobalValidationResult {
    gvr := &GlobalValidationResult{
        IsValid:      true,
        OverallScore: 100.0,
        ShiftScores:  make(map[string]float64),
        Violations:   make([]*ValidationItem, 0),
        Warnings:     make([]*ValidationItem, 0),
    }
    
    // 1. 全局约束检查（跨班次）
    // 2. 公平性分析
    gvr.StaffFairness = v.analyzeFairness(draft, allStaff)
    
    // 3. 生成摘要
    gvr.Summary = fmt.Sprintf("全局评分: %.1f, 公平性标准差: %.2f",
        gvr.OverallScore, gvr.StaffFairness.StdDeviation)
    
    return gvr
}

// analyzeFairness 分析排班公平性
func (v *ScheduleValidator) analyzeFairness(
    draft *d_model.ScheduleDraft,
    allStaff []*d_model.Employee,
) *FairnessReport {
    report := &FairnessReport{
        Details: make(map[string]*StaffFairness),
    }
    
    if draft == nil || draft.Shifts == nil {
        return report
    }
    
    // 统计每个人的排班次数
    for _, staff := range allStaff {
        sf := &StaffFairness{
            StaffID:   staff.ID,
            StaffName: staff.Name,
        }
        for _, shiftDraft := range draft.Shifts {
            if shiftDraft == nil || shiftDraft.Schedule == nil {
                continue
            }
            for _, staffIDs := range shiftDraft.Schedule {
                for _, sid := range staffIDs {
                    if sid == staff.ID {
                        sf.TotalShifts++
                    }
                }
            }
        }
        report.Details[staff.ID] = sf
    }
    
    // 计算标准差和最大差异
    if len(report.Details) > 0 {
        sum := 0.0
        min, max := 999999, 0
        for _, sf := range report.Details {
            sum += float64(sf.TotalShifts)
            if sf.TotalShifts < min { min = sf.TotalShifts }
            if sf.TotalShifts > max { max = sf.TotalShifts }
        }
        report.MaxDiff = max - min
        // TODO: 计算标准差
    }
    
    return report
}

func (v *ScheduleValidator) buildSummary(result *ValidationResult) string {
    if result.IsValid {
        return fmt.Sprintf("校验通过，评分 %.1f", result.Score)
    }
    return fmt.Sprintf("校验失败，%d 个违反，%d 个警告，评分 %.1f",
        len(result.Violations), len(result.Warnings), result.Score)
}
```

## 9. 偏好评分器

**文件**: `internal/engine/preference_scorer.go`

```go
package engine

import (
    "time"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/mcp/pkg/logging"
)

// PreferenceScorer 偏好评分器
type PreferenceScorer struct {
    logger logging.ILogger
}

func NewPreferenceScorer(logger logging.ILogger) *PreferenceScorer {
    return &PreferenceScorer{logger: logger.With("sub", "PreferenceScorer")}
}

// Score 为每个合格候选人计算偏好评分
func (p *PreferenceScorer) Score(
    candidates []*CandidateStatus,
    preferenceRules []*ClassifiedRule,
    draft *d_model.ScheduleDraft,
    date time.Time,
) *PreferenceScoreResult {
    result := &PreferenceScoreResult{
        Scores:  make(map[string]float64),
        Details: make(map[string][]*PreferenceDetail),
    }
    
    for _, candidate := range candidates {
        totalScore := 0.0
        totalWeight := 0.0
        details := make([]*PreferenceDetail, 0)
        
        for _, rule := range preferenceRules {
            weight := float64(rule.Rule.Priority)
            if weight <= 0 { weight = 5 }
            
            score := p.scorePreference(candidate, rule, draft, date)
            totalScore += score * weight
            totalWeight += weight
            
            details = append(details, &PreferenceDetail{
                RuleID:   rule.Rule.ID,
                RuleName: rule.Rule.Name,
                Score:    score,
            })
        }
        
        // 加入公平性评分：排班少的人得分更高
        fairnessScore := p.computeFairnessScore(candidate)
        fairnessWeight := 3.0 // 公平性权重
        totalScore += fairnessScore * fairnessWeight
        totalWeight += fairnessWeight
        
        finalScore := 0.5 // 默认
        if totalWeight > 0 {
            finalScore = totalScore / totalWeight
        }
        
        result.Scores[candidate.StaffID] = finalScore
        result.Details[candidate.StaffID] = details
        candidate.PreferenceScore = finalScore
    }
    
    return result
}

// scorePreference 单条偏好规则评分
func (p *PreferenceScorer) scorePreference(
    candidate *CandidateStatus,
    rule *ClassifiedRule,
    draft *d_model.ScheduleDraft,
    date time.Time,
) float64 {
    // 基础评分逻辑
    switch rule.Rule.RuleType {
    case "preferred":
        // 如果规则偏好某员工/分组/时间，匹配则高分
        return p.checkPreferredMatch(candidate, rule, date)
    default:
        return 0.5
    }
}

// checkPreferredMatch 检查偏好匹配
func (p *PreferenceScorer) checkPreferredMatch(
    candidate *CandidateStatus,
    rule *ClassifiedRule,
    date time.Time,
) float64 {
    // 检查员工是否在规则关联的对象中
    for _, assoc := range rule.Targets {
        if assoc.AssociationType == "employee" && assoc.AssociationID == candidate.StaffID {
            return 0.9
        }
        if assoc.AssociationType == "group" {
            for _, g := range candidate.Groups {
                if g == assoc.AssociationID {
                    return 0.8
                }
            }
        }
    }
    return 0.5
}

// computeFairnessScore 公平性评分（排班少的人得分高）
func (p *PreferenceScorer) computeFairnessScore(candidate *CandidateStatus) float64 {
    // 本周排班越少，公平性评分越高
    count := candidate.WeeklyShiftCount
    switch {
    case count == 0:
        return 1.0
    case count == 1:
        return 0.8
    case count == 2:
        return 0.6
    case count == 3:
        return 0.4
    case count == 4:
        return 0.2
    default:
        return 0.1
    }
}
```

## 10. 测试要求

每个组件至少覆盖以下测试场景：

### CandidateFilter
- 请假人员过滤
- 个人需求(avoid)过滤
- 固定排班人员排除
- 空列表边界

### RuleMatcher
- 全局规则匹配所有班次
- Association 精确匹配
- 规则有效期过滤
- V3 兼容（无 Category 字段的规则）
- V4 Role 分类

### ConstraintChecker
- `maxCount`: 未达限制 / 刚好达限 / 已超限
- `consecutiveMax`: 连续 0/1/max/max+1 天
- `minRestDays`: 间隔 0/1/min/min+1 天
- `exclusive`: 无冲突 / 有冲突
- 多规则组合
- 空草稿

### DependencyResolver
- 线性依赖: A→B→C
- 菱形依赖: A→B→D, A→C→D
- 无依赖（按优先级排序）
- 循环依赖检测: A→B→A
- 空依赖列表

### ScheduleValidator
- 全通过
- 硬约束违反
- 人员重复
- 人数不足/超配
