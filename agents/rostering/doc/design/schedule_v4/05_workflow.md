# 05. V4 工作流详细设计

> **开发负责人**: Agent-3  
> **依赖**: Agent-1 (数据模型), Agent-2 (规则引擎)  
> **被依赖**: Agent-5 (前端)  
> **包路径**: `agents/rostering/internal/workflow/schedule_v4/`

## 1. 设计目标

V4 工作流继承 V3 的状态机架构和渐进式排班思路，核心改造点：

| 改造项 | V3 | V4 |
|-------|----|----|
| 规则预分析 | 3 个 LLM 并行调用 | `engine.PrepareSchedulingContext()` 代码计算 |
| Prompt 构建 | 自然语言拼接规则 | 结构化 JSON + 模板化 Prompt |
| 排班执行 | LLM-4 + LLM-5 (排班+校验) | LLM 仅做排班决策 + 引擎校验 |
| 全局校验 | LLM 全局评审 | 引擎 `ValidateGlobal()` + LLM 补充建议 |
| 班次顺序 | SchedulingPriority 简单排序 | 拓扑排序（ShiftDependency） |

## 2. 目录结构

```
agents/rostering/internal/workflow/schedule_v4/
├── create/                            # 创建工作流
│   ├── context.go                     # CreateV4Context
│   ├── definition.go                  # 状态机定义
│   ├── actions_init.go                # 初始化阶段 actions
│   ├── actions_collect.go             # 信息收集阶段 actions
│   ├── actions_plan.go                # 计划生成阶段 actions
│   ├── actions_execute.go             # 任务执行阶段 actions
│   ├── actions_review.go             # 审核阶段 actions
│   └── actions_save.go               # 保存阶段 actions
├── executor/                          # V4 执行器
│   ├── types.go                       # 类型定义
│   ├── executor.go                    # V4TaskExecutor
│   ├── prompt_builder.go             # 结构化 Prompt 构建器
│   └── result_parser.go              # LLM 结果解析
└── state/                             # 状态定义
    └── constants.go                   # 状态/事件常量
```

## 3. 状态常量定义

**文件**: `internal/workflow/schedule_v4/state/constants.go`

```go
package state

import "jusha/mcp/pkg/workflow/engine"

// ============================================================
// 工作流名称
// ============================================================
const WorkflowScheduleCreateV4 = "schedule.create.v4"

// ============================================================
// 状态定义
// ============================================================
const (
    // 阶段0: 初始化
    V4StateInit               engine.State = "v4.init"
    
    // 阶段1: 信息收集（复用V3）
    V4StateInfoCollecting     engine.State = "v4.info_collecting"
    V4StateConfirmPeriod      engine.State = "v4.confirm_period"
    V4StateConfirmShifts      engine.State = "v4.confirm_shifts"
    V4StateConfirmStaffCount  engine.State = "v4.confirm_staff_count"
    
    // 阶段2: 个人需求（复用V3）
    V4StatePersonalNeeds      engine.State = "v4.personal_needs"
    
    // 阶段3: 需求评估 & 计划生成
    // V4改动: 增加规则解析验证步骤
    V4StateRuleValidation     engine.State = "v4.rule_validation"
    V4StatePlanGeneration     engine.State = "v4.plan_generation"
    V4StatePlanReview         engine.State = "v4.plan_review"
    
    // 阶段4: 渐进式任务执行（核心改造）
    V4StateProgressiveTask    engine.State = "v4.progressive_task"
    V4StateTaskReview         engine.State = "v4.task_review"
    V4StateWaitingAdjustment  engine.State = "v4.waiting_adjustment"
    V4StateTaskFailed         engine.State = "v4.task_failed"
    
    // 阶段5: 全局校验（V4改造）
    V4StateGlobalValidation   engine.State = "v4.global_validation"
    V4StateGlobalReviewManual engine.State = "v4.global_review_manual"
    
    // 阶段6: 保存
    V4StateConfirmSaving      engine.State = "v4.confirm_saving"
    V4StateCompleted          engine.State = "v4.completed"
    V4StateCancelled          engine.State = "v4.cancelled"
    V4StateFailed             engine.State = "v4.failed"
)

// ============================================================
// 事件定义
// ============================================================
const (
    // 通用事件
    V4EventStart              engine.Event = "v4.start"
    V4EventUserCancel         engine.Event = "v4.user_cancel"
    V4EventError              engine.Event = "v4.error"
    V4EventUserModify         engine.Event = "v4.user_modify"
    
    // 信息收集事件
    V4EventInfoCollected      engine.Event = "v4.info_collected"
    V4EventPeriodConfirmed    engine.Event = "v4.period_confirmed"
    V4EventShiftsConfirmed    engine.Event = "v4.shifts_confirmed"
    V4EventStaffCountConfirmed engine.Event = "v4.staff_count_confirmed"
    
    // 个人需求事件
    V4EventPersonalNeedsConfirmed engine.Event = "v4.personal_needs_confirmed"
    V4EventSkipPhase              engine.Event = "v4.skip_phase"
    
    // 规则验证事件（V4新增）
    V4EventRuleValidationComplete engine.Event = "v4.rule_validation_complete"
    V4EventRuleValidationFailed   engine.Event = "v4.rule_validation_failed"
    
    // 计划事件
    V4EventPlanGenerated      engine.Event = "v4.plan_generated"
    V4EventPlanConfirmed      engine.Event = "v4.plan_confirmed"
    V4EventPlanAdjust         engine.Event = "v4.plan_adjust"
    
    // 任务事件
    V4EventTaskCompleted      engine.Event = "v4.task_completed"
    V4EventAllTasksComplete   engine.Event = "v4.all_tasks_complete"
    V4EventTaskFailed         engine.Event = "v4.task_failed"
    V4EventEnterTaskReview    engine.Event = "v4.enter_task_review"
    V4EventTaskReviewConfirmed engine.Event = "v4.task_review_confirmed"
    V4EventTaskReviewAdjust   engine.Event = "v4.task_review_adjust"
    V4EventTaskFailedContinue engine.Event = "v4.task_failed_continue"
    
    // 全局校验事件（V4改造）
    V4EventGlobalValidationComplete engine.Event = "v4.global_validation_complete"
    V4EventGlobalValidationFailed   engine.Event = "v4.global_validation_failed"
    V4EventGlobalReviewManualDone   engine.Event = "v4.global_review_manual_done"
    
    // 保存事件
    V4EventSaveCompleted      engine.Event = "v4.save_completed"
)
```

## 4. V4 工作流上下文

**文件**: `internal/workflow/schedule_v4/create/context.go`

```go
package create

import (
    "time"
    d_model "jusha/agent/rostering/domain/model"
    "jusha/agent/rostering/internal/engine"
)

// CreateV4Context V4 排班工作流上下文
// 继承 V3 字段，增加 V4 专有字段
type CreateV4Context struct {
    // ========== 基础信息（与V3相同） ==========
    OrgID              string                              `json:"orgId,omitempty"`
    StartDate          string                              `json:"startDate"`
    EndDate            string                              `json:"endDate"`
    SelectedShifts     []*d_model.Shift                    `json:"selectedShifts"`
    StaffRequirements  []d_model.ShiftDateRequirement      `json:"staffRequirements"`
    AllStaff           []*d_model.Employee                 `json:"allStaffList"`
    StaffLeaves        map[string][]*d_model.LeaveRecord   `json:"staffLeaves"`
    Rules              []*d_model.Rule                     `json:"rules"`
    
    // ========== 个人需求（与V3相同） ==========
    PersonalNeeds           map[string][]*engine.PersonalNeed `json:"personalNeeds"`
    PersonalNeedsConfirmed  bool                              `json:"personalNeedsConfirmed"`
    
    // ========== V4 规则分析结果（新增） ==========
    
    // RuleDependencies 规则依赖关系（从数据库加载）
    RuleDependencies   []*d_model.RuleDependency           `json:"ruleDependencies"`
    
    // RuleConflicts 规则冲突关系（从数据库加载）
    RuleConflicts      []*d_model.RuleConflict             `json:"ruleConflicts"`
    
    // ShiftDependencies 班次依赖关系（从数据库加载）
    ShiftDependencies  []*d_model.ShiftDependency          `json:"shiftDependencies"`
    
    // ShiftExecutionOrder 班次执行顺序（依赖解析后）
    ShiftExecutionOrder []string                            `json:"shiftExecutionOrder"`
    
    // RuleValidationResult 规则预校验结果
    RuleValidationResult *RulePreValidationResult           `json:"ruleValidationResult"`
    
    // ========== 渐进式任务（与V3相同） ==========
    ProgressiveTaskPlan *d_model.ProgressiveTaskPlan        `json:"progressiveTaskPlan"`
    CurrentTaskIndex    int                                 `json:"currentTaskIndex"`
    TaskResults         map[string]*d_model.TaskResult      `json:"taskResults"`
    
    // ========== 占位 & 固定排班（与V3相同） ==========
    OccupiedSlots       []d_model.StaffOccupiedSlot         `json:"occupiedSlots"`
    FixedAssignments    []d_model.CtxFixedShiftAssignment   `json:"fixedAssignments"`
    
    // ========== V4 引擎上下文缓存 ==========
    
    // EngineContexts 引擎计算的排班上下文缓存
    // key: "{shiftID}_{date}" 
    // 每次执行前由引擎重新计算，不持久化
    EngineContexts      map[string]*engine.SchedulingContext `json:"-"`
    
    // GlobalValidationResult 全局校验结果（V4 引擎计算）
    GlobalValidationResult *engine.GlobalValidationResult    `json:"globalValidationResult"`
    
    // ========== 结果（与V3相同） ==========
    BaselineSchedule   *d_model.ScheduleDraft               `json:"baselineSchedule"`
    WorkingDraft       *d_model.ScheduleDraft               `json:"finalScheduleDraft"`
    ChangeBatches      []*d_model.ScheduleChangeBatch       `json:"changeBatches"`
    SavedScheduleID    string                               `json:"savedScheduleId"`
    
    // ========== 统计信息 ==========
    CompletedTaskCount  int                                 `json:"completedTaskCount"`
    FailedTaskCount     int                                 `json:"failedTaskCount"`
    SkippedTaskCount    int                                 `json:"skippedTaskCount"`
    SupplementRound     int                                 `json:"supplementRound"`
}

// RulePreValidationResult 规则预校验结果（在排班前检查规则数据质量）
type RulePreValidationResult struct {
    IsValid           bool                  `json:"isValid"`
    ParsedRuleCount   int                   `json:"parsedRuleCount"`
    UnparsedRuleCount int                   `json:"unparsedRuleCount"`
    CircularDeps      [][]string            `json:"circularDeps,omitempty"`
    ConflictWarnings  []string              `json:"conflictWarnings,omitempty"`
    Errors            []string              `json:"errors,omitempty"`
}

// GetEngineContextKey 获取引擎上下文缓存key
func GetEngineContextKey(shiftID, dateStr string) string {
    return shiftID + "_" + dateStr
}
```

## 5. 状态机定义

**文件**: `internal/workflow/schedule_v4/create/definition.go`

```go
package create

import (
    "jusha/mcp/pkg/workflow/engine"
    . "jusha/agent/rostering/internal/workflow/schedule_v4/state"
)

func init() {
    engine.Register(GetScheduleCreateV4WorkflowDefinition())
}

func GetScheduleCreateV4WorkflowDefinition() *engine.WorkflowDefinition {
    return &engine.WorkflowDefinition{
        Name:         WorkflowScheduleCreateV4,
        InitialState: V4StateInit,
        Transitions:  buildCreateV4Transitions(),
    }
}

func buildCreateV4Transitions() []engine.Transition {
    return []engine.Transition{
        // ============================================================
        // 阶段 0: 初始化
        // ============================================================
        {
            From:       V4StateInit,
            Event:      V4EventStart,
            To:         V4StateInfoCollecting,
            StateLabel: "收集排班信息",
            Act:        actStartInfoCollect,
        },
        {From: V4StateInit, Event: V4EventUserCancel, To: V4StateCancelled, Act: actUserCancel},

        // ============================================================
        // 阶段 1: 信息收集（复用 V3 逻辑，仅改状态名）
        // ============================================================
        {From: V4StateInfoCollecting, Event: V4EventInfoCollected, To: V4StateConfirmPeriod, Act: actOnInfoCollected},
        {From: V4StateConfirmPeriod, Event: V4EventPeriodConfirmed, To: V4StateConfirmShifts, Act: actOnPeriodConfirmed},
        {From: V4StateConfirmPeriod, Event: V4EventUserModify, To: V4StateConfirmPeriod, Act: actModifyPeriod},
        {From: V4StateConfirmShifts, Event: V4EventShiftsConfirmed, To: V4StateConfirmStaffCount, Act: actOnShiftsConfirmed},
        {From: V4StateConfirmShifts, Event: V4EventUserModify, To: V4StateConfirmShifts, Act: actModifyShifts},
        {From: V4StateConfirmStaffCount, Event: V4EventStaffCountConfirmed, To: V4StatePersonalNeeds, Act: actOnStaffCountConfirmed},

        // ============================================================
        // 阶段 2: 个人需求（复用 V3）
        // ============================================================
        {From: V4StatePersonalNeeds, Event: V4EventPersonalNeedsConfirmed, To: V4StateRuleValidation, Act: actOnPersonalNeedsConfirmed},
        {From: V4StatePersonalNeeds, Event: V4EventSkipPhase, To: V4StateRuleValidation, Act: actSkipPersonalNeeds},

        // ============================================================
        // 阶段 3: 规则预校验（V4 新增阶段）
        // 目的: 在排班前检查规则数据质量
        // - 检查所有规则的 Category/SubCategory 是否已填充
        // - 检查 ShiftDependency 有无循环
        // - 检查 RuleConflict 有无矛盾
        // - 加载依赖关系并计算班次执行顺序
        // ============================================================
        {
            From:       V4StateRuleValidation,
            Event:      V4EventRuleValidationComplete,
            To:         V4StatePlanGeneration,
            StateLabel: "规则验证通过，生成任务计划",
            Act:        actOnRuleValidationComplete,
        },
        {
            From:       V4StateRuleValidation,
            Event:      V4EventRuleValidationFailed,
            To:         V4StateFailed,
            StateLabel: "规则数据存在问题",
            Act:        actOnRuleValidationFailed,
        },

        // ============================================================
        // 阶段 3.5: 计划生成（V4 改造）
        // V3: 完全由 LLM 生成任务计划
        // V4: 引擎计算班次执行顺序 + LLM 辅助分组优化
        // ============================================================
        {From: V4StatePlanGeneration, Event: V4EventPlanGenerated, To: V4StatePlanReview, Act: actOnPlanGenerated},
        {From: V4StatePlanReview, Event: V4EventPlanConfirmed, To: V4StateProgressiveTask,
            StateLabel: "正在执行V4排班任务...", Act: actOnPlanConfirmed, AfterAct: actAfterPlanConfirmed},
        {From: V4StatePlanReview, Event: V4EventPlanAdjust, To: V4StateWaitingAdjustment, Act: actOnPlanAdjust},

        // ============================================================
        // 阶段 4: 渐进式任务执行（V4 核心改造）
        // 
        // V3 单个 Shift+Date 执行流程:
        //   LLM-1(人员过滤) + LLM-2(规则过滤) + LLM-3(冲突检测)
        //   → LLM-4(排班决策) → LLM-5(校验)
        //
        // V4 单个 Shift+Date 执行流程:
        //   engine.PrepareSchedulingContext()   ← 替代 LLM-1/2/3
        //   → StructuredPrompt(LLM-4)          ← 传入引擎结果
        //   → engine.ValidateSchedule()        ← 替代 LLM-5
        //   → [如失败] AutoFix or Retry
        // ============================================================
        {
            From: V4StateProgressiveTask, Event: V4EventTaskCompleted,
            To: V4StateProgressiveTask,
            Act: actOnTaskCompleted, AfterAct: actAfterTaskCompleted,
        },
        {From: V4StateProgressiveTask, Event: V4EventEnterTaskReview, To: V4StateTaskReview, Act: actEnterTaskReview},
        {From: V4StateProgressiveTask, Event: V4EventAllTasksComplete, To: V4StateGlobalValidation, Act: actOnAllTasksComplete},
        {
            From: V4StateProgressiveTask, Event: V4EventTaskFailed,
            To: V4StateTaskFailed, StateLabel: "任务失败，等待处理",
            Act: actOnTaskFailed,
        },

        // 任务审核
        {From: V4StateTaskReview, Event: V4EventTaskReviewConfirmed, To: V4StateProgressiveTask,
            Act: actOnTaskReviewConfirmed, AfterAct: actSpawnNextTaskOrComplete},
        {From: V4StateTaskReview, Event: V4EventTaskReviewAdjust, To: V4StateWaitingAdjustment, Act: actOnTaskReviewAdjust},

        // 调整
        {From: V4StateWaitingAdjustment, Event: V4EventUserModify, To: V4StatePlanReview, Act: actOnPlanAdjusted},

        // 失败处理
        {From: V4StateTaskFailed, Event: V4EventTaskFailedContinue, To: V4StateProgressiveTask,
            Act: actOnTaskFailedContinue, AfterAct: actSpawnNextTaskOrComplete},

        // ============================================================
        // 阶段 5: 全局校验（V4 改造）
        //
        // V3: LLM 做全局评审（单次大 LLM 调用，不稳定）
        // V4: engine.ValidateGlobal() 确定性计算
        //     + 公平性报告
        //     + LLM 仅在发现人工需介入时提供建议
        // ============================================================
        {
            From:       V4StateGlobalValidation,
            Event:      V4EventGlobalValidationComplete,
            To:         V4StateConfirmSaving,
            StateLabel: "全局校验通过",
            Act:        actOnGlobalValidationComplete,
        },
        {
            From:  V4StateGlobalValidation,
            Event: V4EventGlobalValidationFailed,
            To:    V4StateGlobalReviewManual,
            Act:   actOnGlobalValidationFailed,
        },
        {From: V4StateGlobalReviewManual, Event: V4EventGlobalReviewManualDone, To: V4StateConfirmSaving, Act: actOnGlobalReviewManualDone},
        {From: V4StateGlobalReviewManual, Event: V4EventUserModify, To: V4StateGlobalReviewManual, Act: actModifyGlobalReview},

        // ============================================================
        // 阶段 6: 保存
        // ============================================================
        {From: V4StateConfirmSaving, Event: V4EventSaveCompleted, To: V4StateCompleted, StateLabel: "V4排班已保存 ✅", Act: actOnSaveCompleted},
        {From: V4StateConfirmSaving, Event: V4EventUserModify, To: V4StateConfirmSaving, Act: actModifyBeforeSave},

        // 全局错误/取消
        {From: engine.State("*"), Event: V4EventError, To: V4StateFailed, Act: actHandleError},
        {From: engine.State("*"), Event: V4EventUserCancel, To: V4StateCancelled, Act: actUserCancel},
    }
}
```

## 6. V4 执行器

**文件**: `internal/workflow/schedule_v4/executor/types.go`

```go
package executor

import (
    d_model "jusha/agent/rostering/domain/model"
    "jusha/agent/rostering/internal/engine"
)

// IV4TaskExecutor V4 任务执行器接口
type IV4TaskExecutor interface {
    // ExecuteTask 执行单个渐进式任务
    // V4改动: 接收 V4TaskContext，内部使用规则引擎
    ExecuteTask(ctx context.Context, taskCtx *V4TaskContext) (*d_model.TaskResult, error)
    
    // SetProgressCallback 设置进度回调
    SetProgressCallback(callback ProgressCallback)
}

// ProgressCallback 进度回调
type ProgressCallback func(info *ProgressInfo)

// ProgressInfo 进度信息
type ProgressInfo struct {
    TaskID        string
    ShiftID       string
    ShiftName     string
    Date          string
    Status        string  // "preparing" / "scheduling" / "validating" / "completed" / "failed"
    Message       string
    Progress      float64 // 0.0~1.0
}

// V4TaskContext V4 任务执行上下文
type V4TaskContext struct {
    // 任务定义
    Task              *d_model.ProgressiveTask
    
    // 所有数据（L1 传入）
    AllStaff          []*d_model.Employee
    AllRules          []*d_model.Rule
    AllShifts         []*d_model.Shift
    StaffLeaves       map[string][]*d_model.LeaveRecord
    PersonalNeeds     map[string][]*engine.PersonalNeed
    FixedAssignments  []d_model.CtxFixedShiftAssignment
    OccupiedSlots     []d_model.StaffOccupiedSlot
    StaffRequirements []d_model.ShiftDateRequirement
    
    // V4 依赖/冲突
    RuleDependencies  []*d_model.RuleDependency
    RuleConflicts     []*d_model.RuleConflict
    ShiftDependencies []*d_model.ShiftDependency
    
    // 当前排班草稿
    GlobalDraft       *d_model.ScheduleDraft
    CurrentDraft      *d_model.ShiftScheduleDraft
    
    // 规则引擎实例
    Engine            engine.IRuleEngine
}
```

**文件**: `internal/workflow/schedule_v4/executor/executor.go`

```go
package executor

import (
    "context"
    "fmt"
    "time"

    d_model "jusha/agent/rostering/domain/model"
    "jusha/agent/rostering/internal/engine"
    "jusha/mcp/pkg/ai"
    "jusha/mcp/pkg/logging"
)

// V4TaskExecutor V4 任务执行器
type V4TaskExecutor struct {
    logger           logging.ILogger
    ruleEngine       engine.IRuleEngine
    aiFactory        *ai.AIProviderFactory
    promptBuilder    *PromptBuilder
    resultParser     *ResultParser
    progressCallback ProgressCallback
}

// NewV4TaskExecutor 创建 V4 任务执行器
func NewV4TaskExecutor(
    logger logging.ILogger,
    ruleEngine engine.IRuleEngine,
    aiFactory *ai.AIProviderFactory,
) *V4TaskExecutor {
    return &V4TaskExecutor{
        logger:        logger.With("component", "V4TaskExecutor"),
        ruleEngine:    ruleEngine,
        aiFactory:     aiFactory,
        promptBuilder: NewPromptBuilder(logger),
        resultParser:  NewResultParser(logger),
    }
}

func (e *V4TaskExecutor) SetProgressCallback(callback ProgressCallback) {
    e.progressCallback = callback
}

// ExecuteTask 执行单个 V4 任务
func (e *V4TaskExecutor) ExecuteTask(
    ctx context.Context,
    taskCtx *V4TaskContext,
) (*d_model.TaskResult, error) {
    task := taskCtx.Task
    startTime := time.Now()
    
    e.logger.Info("V4 task execution started",
        "taskID", task.ID,
        "title", task.Title,
        "shifts", len(task.TargetShifts),
        "dates", len(task.TargetDates))
    
    result := &d_model.TaskResult{
        TaskID:         task.ID,
        Success:        true,
        ShiftSchedules: make(map[string]*d_model.ShiftScheduleDraft),
    }
    
    // 遍历每个班次 x 日期
    for _, shiftID := range task.TargetShifts {
        shift := e.findShift(taskCtx.AllShifts, shiftID)
        if shift == nil {
            e.logger.Warn("Shift not found, skipping", "shiftID", shiftID)
            continue
        }
        
        for _, dateStr := range task.TargetDates {
            date, err := time.Parse("2006-01-02", dateStr)
            if err != nil {
                continue
            }
            
            // ============================================
            // V4 核心流程（替代 V3 的 5 个 LLM 调用）
            // ============================================
            
            e.notifyProgress(&ProgressInfo{
                TaskID: task.ID, ShiftID: shiftID, ShiftName: shift.Name,
                Date: dateStr, Status: "preparing", Progress: 0.1,
            })
            
            // Step 1: 引擎准备上下文（替代 LLM-1/2/3）
            schedCtx, err := e.prepareContext(ctx, taskCtx, shift, date)
            if err != nil {
                e.logger.Error("Engine context preparation failed",
                    "shiftID", shiftID, "date", dateStr, "error", err)
                result.Success = false
                continue
            }
            
            e.notifyProgress(&ProgressInfo{
                TaskID: task.ID, ShiftID: shiftID, ShiftName: shift.Name,
                Date: dateStr, Status: "scheduling",
                Message: fmt.Sprintf("候选人: %d, 匹配规则: %d",
                    len(schedCtx.EligibleCandidates), len(schedCtx.MatchedRules.AllMatched)),
                Progress: 0.4,
            })
            
            // Step 2: 构建结构化 Prompt + LLM 排班决策
            staffIDs, err := e.callLLMForScheduling(ctx, schedCtx)
            if err != nil {
                e.logger.Error("LLM scheduling failed",
                    "shiftID", shiftID, "date", dateStr, "error", err)
                result.Success = false
                continue
            }
            
            e.notifyProgress(&ProgressInfo{
                TaskID: task.ID, ShiftID: shiftID, ShiftName: shift.Name,
                Date: dateStr, Status: "validating", Progress: 0.7,
            })
            
            // Step 3: 引擎校验结果（替代 LLM-5）
            schedResult := &engine.ScheduleResult{
                ShiftID:  shiftID,
                Date:     dateStr,
                StaffIDs: staffIDs,
            }
            validation, err := e.ruleEngine.ValidateSchedule(
                ctx, schedResult, schedCtx.MatchedRules, taskCtx.GlobalDraft)
            if err != nil {
                e.logger.Error("Validation failed",
                    "shiftID", shiftID, "date", dateStr, "error", err)
            }
            
            // Step 4: 根据校验结果决定是否重试
            if validation != nil && !validation.IsValid {
                // 自动修复尝试
                staffIDs = e.tryAutoFix(schedCtx, validation, staffIDs)
            }
            
            // Step 5: 写入结果
            e.mergeResult(result, shiftID, dateStr, staffIDs)
            
            e.notifyProgress(&ProgressInfo{
                TaskID: task.ID, ShiftID: shiftID, ShiftName: shift.Name,
                Date: dateStr, Status: "completed", Progress: 1.0,
                Message: fmt.Sprintf("分配 %d 人", len(staffIDs)),
            })
        }
    }
    
    e.logger.Info("V4 task execution completed",
        "taskID", task.ID,
        "success", result.Success,
        "duration", time.Since(startTime))
    
    return result, nil
}

// prepareContext 调用规则引擎准备上下文
func (e *V4TaskExecutor) prepareContext(
    ctx context.Context,
    taskCtx *V4TaskContext,
    shift *d_model.Shift,
    date time.Time,
) (*engine.SchedulingContext, error) {
    requiredCount := e.getRequiredCount(taskCtx.StaffRequirements, shift.ID, date.Format("2006-01-02"))
    
    input := &engine.SchedulingInput{
        OrgID:             "",
        ShiftID:           shift.ID,
        ShiftName:         shift.Name,
        ShiftStartTime:    shift.StartTime,
        ShiftEndTime:      shift.EndTime,
        ShiftIsOvernight:  shift.IsOvernight,
        Date:              date,
        RequiredCount:     requiredCount,
        AllStaff:          taskCtx.AllStaff,
        AllRules:          taskCtx.AllRules,
        AllShifts:         taskCtx.AllShifts,
        PersonalNeeds:     taskCtx.PersonalNeeds,
        StaffLeaves:       taskCtx.StaffLeaves,
        FixedAssignments:  taskCtx.FixedAssignments,
        GlobalDraft:       taskCtx.GlobalDraft,
        CurrentDraft:      nil, // TODO: extract current shift draft
        OccupiedSlots:     taskCtx.OccupiedSlots,
        RuleDependencies:  taskCtx.RuleDependencies,
        RuleConflicts:     taskCtx.RuleConflicts,
        ShiftDependencies: taskCtx.ShiftDependencies,
    }
    
    return e.ruleEngine.PrepareSchedulingContext(ctx, input)
}

// callLLMForScheduling 调用 LLM 进行排班决策（V4 唯一的 LLM 调用）
func (e *V4TaskExecutor) callLLMForScheduling(
    ctx context.Context,
    schedCtx *engine.SchedulingContext,
) ([]string, error) {
    // 构建结构化 Prompt
    prompt := e.promptBuilder.Build(schedCtx)
    
    // 调用 LLM
    provider := e.aiFactory.GetDefaultProvider()
    resp, err := provider.Chat(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("LLM 调用失败: %w", err)
    }
    
    // 解析结果
    staffIDs, err := e.resultParser.Parse(resp, schedCtx.LLMBrief)
    if err != nil {
        return nil, fmt.Errorf("结果解析失败: %w", err)
    }
    
    return staffIDs, nil
}

// tryAutoFix 尝试自动修复校验失败的结果
func (e *V4TaskExecutor) tryAutoFix(
    schedCtx *engine.SchedulingContext,
    validation *engine.ValidationResult,
    originalStaffIDs []string,
) []string {
    fixed := make([]string, 0, len(originalStaffIDs))
    
    for _, item := range validation.Violations {
        if item.AutoFixable {
            e.logger.Info("Auto-fixing violation",
                "ruleID", item.RuleID, "message", item.Message)
            // TODO: 实现各类自动修复策略
        }
    }
    
    // 如果没有自动修复成功，返回原结果
    if len(fixed) == 0 {
        return originalStaffIDs
    }
    return fixed
}

// ============================================================
// 辅助方法
// ============================================================

func (e *V4TaskExecutor) findShift(shifts []*d_model.Shift, id string) *d_model.Shift {
    for _, s := range shifts {
        if s.ID == id { return s }
    }
    return nil
}

func (e *V4TaskExecutor) getRequiredCount(reqs []d_model.ShiftDateRequirement, shiftID, dateStr string) int {
    for _, r := range reqs {
        if r.ShiftID == shiftID && r.Date == dateStr {
            return r.RequiredCount
        }
    }
    return 1 // 默认1人
}

func (e *V4TaskExecutor) mergeResult(result *d_model.TaskResult, shiftID, dateStr string, staffIDs []string) {
    if _, ok := result.ShiftSchedules[shiftID]; !ok {
        result.ShiftSchedules[shiftID] = d_model.NewShiftScheduleDraft()
    }
    result.ShiftSchedules[shiftID].Schedule[dateStr] = staffIDs
}

func (e *V4TaskExecutor) notifyProgress(info *ProgressInfo) {
    if e.progressCallback != nil {
        e.progressCallback(info)
    }
}
```

## 7. 结构化 Prompt 构建器

**文件**: `internal/workflow/schedule_v4/executor/prompt_builder.go`

```go
package executor

import (
    "encoding/json"
    "fmt"
    "strings"

    "jusha/agent/rostering/internal/engine"
    "jusha/mcp/pkg/logging"
)

// PromptBuilder 结构化 Prompt 构建器
type PromptBuilder struct {
    logger logging.ILogger
}

func NewPromptBuilder(logger logging.ILogger) *PromptBuilder {
    return &PromptBuilder{logger: logger}
}

// Build 构建排班 Prompt
// 核心原则:
// 1. 只传引擎已验证的候选人（不传全量人员）
// 2. 硬约束已过滤，只告知 LLM 边界（不需要 LLM 重新判断）
// 3. 偏好评分已计算，作为参考（不强制）
// 4. 使用短ID（S1/S2/R1/R2），减少 token
func (b *PromptBuilder) Build(schedCtx *engine.SchedulingContext) string {
    brief := schedCtx.LLMBrief
    
    var sb strings.Builder
    
    // System 部分
    sb.WriteString("你是排班助手。根据以下信息为指定班次和日期选择排班人员。\n\n")
    
    // 任务信息
    sb.WriteString(fmt.Sprintf("## 任务\n班次: %s\n日期: %s\n需要人数: %d\n\n",
        schedCtx.ShiftName, schedCtx.DateStr, schedCtx.RequiredCount))
    
    // 候选人列表（已通过硬约束检查）
    sb.WriteString("## 可选人员（已通过规则校验）\n")
    sb.WriteString("| 编号 | 姓名 | 推荐度 | 本周已排 | 连续天数 | 备注 |\n")
    sb.WriteString("|------|------|--------|----------|----------|------|\n")
    for _, c := range brief.Candidates {
        note := c.Note
        if note == "" { note = "-" }
        sb.WriteString(fmt.Sprintf("| %s | %s | %.0f%% | %d | %d | %s |\n",
            c.ShortID, c.Name, c.PreferenceScore*100,
            c.WeeklyCount, c.ConsecutiveDays, note))
    }
    sb.WriteString("\n")
    
    // 硬约束边界（告知LLM）
    if len(brief.HardConstraints) > 0 {
        sb.WriteString("## 硬约束（已预检，请勿违反）\n")
        for _, r := range brief.HardConstraints {
            sb.WriteString(fmt.Sprintf("- %s: %s (限制值: %d)\n", r.RuleShortID, r.Description, r.LimitValue))
        }
        sb.WriteString("\n")
    }
    
    // 偏好建议
    if len(brief.SoftPreferences) > 0 {
        sb.WriteString("## 偏好（尽量满足）\n")
        for _, p := range brief.SoftPreferences {
            sb.WriteString(fmt.Sprintf("- %s (权重: %d/10)\n", p.Description, p.Weight))
        }
        sb.WriteString("\n")
    }
    
    // 已排除人员（透明化，让 LLM 理解为何某些人不可用）
    if len(brief.ExcludedWithReasons) > 0 {
        sb.WriteString("## 已排除人员\n")
        for _, ex := range brief.ExcludedWithReasons {
            sb.WriteString(fmt.Sprintf("- %s: %s\n", ex.Name, ex.Reason))
        }
        sb.WriteString("\n")
    }
    
    // 输出格式要求
    sb.WriteString("## 输出要求\n")
    sb.WriteString(fmt.Sprintf("从可选人员中选择 %d 人。", schedCtx.RequiredCount))
    sb.WriteString("优先选择推荐度高、本周排班少的人员，兼顾公平性。\n")
    sb.WriteString("请以 JSON 格式输出:\n")
    sb.WriteString("```json\n")
    sb.WriteString(`{"selected": ["S1", "S3"], "reason": "简要说明选择理由"}`)
    sb.WriteString("\n```\n")
    
    prompt := sb.String()
    
    b.logger.Debug("Prompt built",
        "candidates", len(brief.Candidates),
        "constraints", len(brief.HardConstraints),
        "promptLength", len(prompt))
    
    return prompt
}

// BuildBatch 批量构建（多日期同班次）
// 适用于同一班次连续多天的场景，减少 LLM 调用次数
func (b *PromptBuilder) BuildBatch(contexts []*engine.SchedulingContext) string {
    if len(contexts) == 1 {
        return b.Build(contexts[0])
    }
    
    var sb strings.Builder
    sb.WriteString("你是排班助手。请为以下多个日期的同一班次选择排班人员。\n\n")
    sb.WriteString(fmt.Sprintf("## 班次: %s\n\n", contexts[0].ShiftName))
    
    for i, ctx := range contexts {
        sb.WriteString(fmt.Sprintf("### 日期 %d: %s (需 %d 人)\n", i+1, ctx.DateStr, ctx.RequiredCount))
        
        // 候选人
        briefJSON, _ := json.Marshal(ctx.LLMBrief.Candidates)
        sb.WriteString(fmt.Sprintf("候选人: %s\n\n", string(briefJSON)))
    }
    
    sb.WriteString("## 输出要求\n")
    sb.WriteString("对每个日期分别选人。JSON格式:\n")
    sb.WriteString("```json\n")
    sb.WriteString(`{"dates": {"2025-01-06": {"selected": ["S1"]}, "2025-01-07": {"selected": ["S2"]}}}`)
    sb.WriteString("\n```\n")
    
    return sb.String()
}
```

## 8. LLM 结果解析器

**文件**: `internal/workflow/schedule_v4/executor/result_parser.go`

```go
package executor

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"

    "jusha/agent/rostering/internal/engine"
    "jusha/mcp/pkg/logging"
)

// ResultParser LLM 结果解析器
type ResultParser struct {
    logger logging.ILogger
}

func NewResultParser(logger logging.ILogger) *ResultParser {
    return &ResultParser{logger: logger}
}

// schedulingResponse LLM 排班响应
type schedulingResponse struct {
    Selected []string `json:"selected"`
    Reason   string   `json:"reason"`
}

// Parse 解析 LLM 排班结果，返回真实 StaffID 列表
func (p *ResultParser) Parse(
    llmResponse string,
    brief *engine.LLMBrief,
) ([]string, error) {
    // 提取 JSON 块
    jsonStr := p.extractJSON(llmResponse)
    if jsonStr == "" {
        return nil, fmt.Errorf("LLM 响应中未找到 JSON 块")
    }
    
    // 解析 JSON
    var resp schedulingResponse
    if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
        return nil, fmt.Errorf("JSON 解析失败: %w, raw: %s", err, jsonStr)
    }
    
    if len(resp.Selected) == 0 {
        return nil, fmt.Errorf("LLM 未选择任何人员")
    }
    
    // 构建短ID到真实ID的映射
    shortToReal := make(map[string]string)
    for _, c := range brief.Candidates {
        shortToReal[c.ShortID] = c.RealID
        // 也支持直接使用姓名
        shortToReal[c.Name] = c.RealID
    }
    
    // 映射短ID到真实ID
    realIDs := make([]string, 0, len(resp.Selected))
    for _, shortID := range resp.Selected {
        shortID = strings.TrimSpace(shortID)
        if realID, ok := shortToReal[shortID]; ok {
            realIDs = append(realIDs, realID)
        } else {
            p.logger.Warn("Unknown short ID from LLM, skipping",
                "shortID", shortID)
        }
    }
    
    if len(realIDs) == 0 {
        return nil, fmt.Errorf("所有 LLM 选择的 ID 都无法映射到真实人员")
    }
    
    p.logger.Debug("LLM result parsed",
        "selected", resp.Selected,
        "realIDs", realIDs,
        "reason", resp.Reason)
    
    return realIDs, nil
}

// extractJSON 从 LLM 响应中提取 JSON 块
func (p *ResultParser) extractJSON(text string) string {
    // 尝试 ```json...``` 块
    re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return strings.TrimSpace(matches[1])
    }
    
    // 尝试裸 JSON
    re2 := regexp.MustCompile(`(?s)\{.*\}`)
    match := re2.FindString(text)
    if match != "" {
        return match
    }
    
    return ""
}
```

## 9. V3→V4 执行流程对比

```
V3 单班次单日期执行流程:
┌─────────────────────────────────────────────────────────┐
│ analyzeDayRules()                                       │
│  ├── goroutine 1: LLM-1 filterPersonalNeeds    ~3s     │
│  ├── goroutine 2: LLM-2 filterRelevantRules    ~3s     │
│  └── goroutine 3: LLM-3 filterRuleConflictStaff ~3s    │
│                                                         │
│ executeScheduleForDay()                                 │
│  └── LLM-4 排班决策                            ~5s     │
│                                                         │
│ validateSchedule()                                      │
│  └── LLM-5 校验                                ~3s     │
│                                                         │
│ 总计: ~11s, 5 LLM 调用, ~4000 tokens                    │
└─────────────────────────────────────────────────────────┘

V4 单班次单日期执行流程:
┌─────────────────────────────────────────────────────────┐
│ engine.PrepareSchedulingContext()              ~5ms     │
│  ├── RuleMatcher.MatchRules()                           │
│  ├── CandidateFilter.Filter()                           │
│  ├── ConstraintChecker.CheckAll()                       │
│  ├── PreferenceScorer.Score()                           │
│  └── buildLLMBrief()                                    │
│                                                         │
│ PromptBuilder.Build() + LLM 排班决策           ~5s     │
│                                                         │
│ engine.ValidateSchedule()                      ~1ms     │
│                                                         │
│ 总计: ~5s, 1 LLM 调用, ~800 tokens                     │
└─────────────────────────────────────────────────────────┘

改进:
- LLM 调用: 5 → 1 (减少 80%)
- 延迟: 11s → 5s (减少 55%)
- Token: 4000 → 800 (减少 80%)
- 确定性: 3/5 → 4/5 步骤确定性执行
```

## 10. Action 实现要点

### actOnRuleValidationComplete（V4 新增）

```go
// actOnRuleValidationComplete 规则预校验完成
// 在排班前验证:
// 1. 所有规则是否有 Category（V4 必须字段）
// 2. ShiftDependency 有无循环依赖
// 3. 加载依赖关系并计算班次执行顺序
func actOnRuleValidationComplete(wfCtx engine.WorkflowContext) error {
    c := wfCtx.GetContext().(*CreateV4Context)
    
    // 1. 检查规则分类完整性
    result := &RulePreValidationResult{IsValid: true}
    for _, rule := range c.Rules {
        if rule.Category != "" {
            result.ParsedRuleCount++
        } else {
            result.UnparsedRuleCount++
        }
    }
    
    // 2. 检查循环依赖
    ruleEngine := engine.NewRuleEngine(wfCtx.Logger())
    depResolver := ruleEngine.GetDependencyResolver()
    
    edges := make([]engine.DependencyEdge, 0)
    for _, dep := range c.ShiftDependencies {
        edges = append(edges, engine.DependencyEdge{
            From: dep.DependentShiftID,
            To:   dep.DependsOnShiftID,
        })
    }
    
    cycles, _ := depResolver.DetectCircularDependency(edges)
    if len(cycles) > 0 {
        result.IsValid = false
        result.CircularDeps = cycles
    }
    
    // 3. 计算班次执行顺序
    if result.IsValid {
        order, err := depResolver.ResolveShiftOrder(c.SelectedShifts, c.ShiftDependencies)
        if err != nil {
            result.IsValid = false
            result.Errors = append(result.Errors, err.Error())
        } else {
            c.ShiftExecutionOrder = order
        }
    }
    
    c.RuleValidationResult = result
    return nil
}
```

### actOnPlanConfirmed（V4 改造）

```go
// actOnPlanConfirmed V4 计划确认后
// V3: 直接按 task 顺序执行
// V4: 使用 ShiftExecutionOrder 确定班次执行顺序
func actOnPlanConfirmed(wfCtx engine.WorkflowContext) error {
    c := wfCtx.GetContext().(*CreateV4Context)
    
    // 按 ShiftExecutionOrder 重排任务
    if len(c.ShiftExecutionOrder) > 0 {
        reorderedTasks := reorderTasks(c.ProgressiveTaskPlan.Tasks, c.ShiftExecutionOrder)
        c.ProgressiveTaskPlan.Tasks = reorderedTasks
    }
    
    c.CurrentTaskIndex = 0
    c.TaskResults = make(map[string]*d_model.TaskResult)
    c.EngineContexts = make(map[string]*engine.SchedulingContext)
    
    return nil
}
```
