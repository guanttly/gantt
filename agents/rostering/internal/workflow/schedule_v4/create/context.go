package create

import (
	"time"

	d_model "jusha/agent/rostering/domain/model"

	"jusha/agent/rostering/internal/workflow/schedule_v4/executor"
)

// ============================================================
// CreateV4Context - 排班创建工作流 V4 上下文
// V4特色：基于确定性规则引擎，减少LLM调用
// ============================================================

// KeyCreateV4Context 上下文存储键
const KeyCreateV4Context = "createV4Context"

// CreateV4Context 创建工作流V4的核心上下文
type CreateV4Context struct {
	// ========== 基础信息（来自信息收集阶段） ==========

	// OrgID 组织ID
	OrgID string `json:"orgId,omitempty"`

	// StartDate 排班周期开始日期 (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// EndDate 排班周期结束日期 (YYYY-MM-DD)
	EndDate string `json:"endDate"`

	// SelectedShifts 用户选定的所有班次列表
	SelectedShifts []*d_model.Shift `json:"selectedShifts"`

	// StaffRequirements 人员配置需求 - 强类型数组
	StaffRequirements []d_model.ShiftDateRequirement `json:"staffRequirements"`

	// AllStaff 所有员工列表
	AllStaff []*d_model.Employee `json:"allStaffList"`

	// ShiftMembersMap 每个班次对应的分组成员列表（shiftID -> 员工列表）
	// 由 LoadScheduleBasicContext 赋値，供确定性引擎按班次过滤候选人池使用
	ShiftMembersMap map[string][]*d_model.Employee `json:"shiftMembersMap,omitempty"`

	// StaffLeaves 人员请假信息 (staff_id -> 请假记录列表)
	StaffLeaves map[string][]*d_model.LeaveRecord `json:"staffLeaves"`

	// Rules 所有排班规则
	Rules []*d_model.Rule `json:"rules"`

	// ========== 个人需求（独立收集阶段） ==========

	// PersonalNeeds 个人需求列表 (staff_id -> 需求列表)
	PersonalNeeds map[string][]*PersonalNeed `json:"personalNeeds"`

	// PersonalNeedsConfirmed 个人需求是否已确认
	PersonalNeedsConfirmed bool `json:"personalNeedsConfirmed"`

	// TemporaryNeedsText 临时需求原始文本（用户粘贴）
	TemporaryNeedsText string `json:"temporaryNeedsText,omitempty"`

	// TemporaryRules 临时规则列表（从临时需求解析而来，结构与常态规则一致）
	// 这些规则仅在当前排班周期有效，不持久化到数据库
	TemporaryRules []*d_model.Rule `json:"temporaryRules,omitempty"`

	// ========== V4核心：规则组织（确定性预计算） ==========

	// RuleOrganization 规则组织结果（包含依赖关系、执行顺序等）
	RuleOrganization *executor.RuleOrganization `json:"ruleOrganization,omitempty"`

	// ========== 已占位信息（累积约束） ==========

	// OccupiedSlots 已占位的人员-日期-班次 - 强类型数组
	OccupiedSlots []d_model.StaffOccupiedSlot `json:"occupiedSlots"`

	// ExistingScheduleMarks 已有排班标记 - 强类型数组
	ExistingScheduleMarks []d_model.StaffScheduleMark `json:"existingScheduleMarks"`

	// ========== 固定排班缓存 ==========

	// FixedAssignments 固定排班配置 - 强类型数组
	FixedAssignments []d_model.CtxFixedShiftAssignment `json:"fixedAssignments"`

	// ========== 变更追踪 ==========

	// BaselineSchedule 基准排班（固定排班填充后的状态）
	BaselineSchedule *d_model.ScheduleDraft `json:"baselineSchedule"`

	// ChangeBatches 变更历史（按任务记录）
	ChangeBatches []*d_model.ScheduleChangeBatch `json:"changeBatches"`

	// ========== 最终结果 ==========

	// WorkingDraft 工作草稿（当前排班状态）
	WorkingDraft *d_model.ScheduleDraft `json:"finalScheduleDraft"`

	// SavedScheduleID 保存后的排班ID
	SavedScheduleID string `json:"savedScheduleId"`

	// ========== V4特有：校验结果 ==========

	// ValidationResult 校验结果
	ValidationResult *ValidationResult `json:"validationResult,omitempty"`

	// ========== 统计信息 ==========

	// LLMCallCount LLM调用次数（V4目标：最少化）
	LLMCallCount int `json:"llmCallCount"`

	// SchedulingDuration 排班耗时（毫秒）
	SchedulingDuration int64 `json:"schedulingDuration"`
}

// ValidationResult 校验结果
type ValidationResult struct {
	// IsValid 是否通过校验
	IsValid bool `json:"isValid"`

	// Violations 违规列表
	Violations []*Violation `json:"violations,omitempty"`

	// Warnings 警告列表
	Warnings []*Warning `json:"warnings,omitempty"`

	// UncheckedRules 无法确定性校验的规则（需 LLM 辅助）
	UncheckedRules []*UncheckedRule `json:"uncheckedRules,omitempty"`

	// LLMValidationDone LLM辅助校验是否已执行
	LLMValidationDone bool `json:"llmValidationDone,omitempty"`

	// Summary 校验摘要
	Summary string `json:"summary"`
}

// UncheckedRule 无法确定性校验的规则（需 LLM 辅助判断）
type UncheckedRule struct {
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	RuleType    string `json:"ruleType"`
	Description string `json:"description"` // 规则原始描述
	Reason      string `json:"reason"`      // 为什么无法确定性校验
}

// Violation 违规项
type Violation struct {
	RuleID           string   `json:"ruleId"`
	RuleName         string   `json:"ruleName"`
	Date             string   `json:"date"`
	ShiftID          string   `json:"shiftId"`
	ShiftName        string   `json:"shiftName"`
	StaffID          string   `json:"staffId,omitempty"`
	StaffName        string   `json:"staffName,omitempty"`
	Description      string   `json:"description"`
	Severity         string   `json:"severity"`                   // error/warning
	IssueType        string   `json:"issueType,omitempty"`        // V4.2 新增: 问题类型
	AffectedStaffIDs []string `json:"affectedStaffIds,omitempty"` // V4.2 新增: 涉事的所有人员ID
}

// Warning 警告项
type Warning struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	Severity    string `json:"severity,omitempty"` // "warning"=需关注 / "info"=一般提示
}

// PersonalNeed 个人需求（复用V3的结构）
type PersonalNeed struct {
	StaffID         string   `json:"staffId"`
	StaffName       string   `json:"staffName"`
	NeedType        string   `json:"needType"`    // permanent/temporary
	RequestType     string   `json:"requestType"` // prefer/avoid/must
	TargetShiftID   string   `json:"targetShiftId,omitempty"`
	TargetShiftName string   `json:"targetShiftName,omitempty"`
	TargetDates     []string `json:"targetDates,omitempty"`
	Description     string   `json:"description"`
	Priority        int      `json:"priority"`
	RuleID          string   `json:"ruleId,omitempty"`
	Source          string   `json:"source"` // rule/user
	Confirmed       bool     `json:"confirmed"`
}

// ============================================================
// 上下文初始化和辅助方法
// ============================================================

// NewCreateV4Context 创建新的V4工作流上下文
func NewCreateV4Context() *CreateV4Context {
	return &CreateV4Context{
		PersonalNeeds:         make(map[string][]*PersonalNeed),
		OccupiedSlots:         make([]d_model.StaffOccupiedSlot, 0),
		ExistingScheduleMarks: make([]d_model.StaffScheduleMark, 0),
		StaffRequirements:     make([]d_model.ShiftDateRequirement, 0),
		FixedAssignments:      make([]d_model.CtxFixedShiftAssignment, 0),
		AllStaff:              make([]*d_model.Employee, 0),
		ShiftMembersMap:       make(map[string][]*d_model.Employee),
		StaffLeaves:           make(map[string][]*d_model.LeaveRecord),
		ChangeBatches:         make([]*d_model.ScheduleChangeBatch, 0),
		LLMCallCount:          0,
	}
}

// IsStaffOccupied 检查人员在指定日期是否已被占位
func (ctx *CreateV4Context) IsStaffOccupied(staffID, date string) bool {
	return d_model.IsStaffOccupiedOnDate(ctx.OccupiedSlots, staffID, date)
}

// OccupySlot 占位：记录人员在指定日期被分配到某班次
func (ctx *CreateV4Context) OccupySlot(staffID, date, shiftID string) {
	ctx.OccupiedSlots = d_model.AddOccupiedSlot(ctx.OccupiedSlots, d_model.StaffOccupiedSlot{
		StaffID: staffID,
		Date:    date,
		ShiftID: shiftID,
	})
}

// GetOccupiedShift 获取人员在指定日期已被分配的班次ID
func (ctx *CreateV4Context) GetOccupiedShift(staffID, date string) string {
	slot := d_model.FindOccupiedSlot(ctx.OccupiedSlots, staffID, date)
	if slot != nil {
		return slot.ShiftID
	}
	return ""
}

// GetRequirement 获取指定班次和日期的人员需求
func (ctx *CreateV4Context) GetRequirement(shiftID, date string) int {
	req := d_model.FindRequirement(ctx.StaffRequirements, shiftID, date)
	if req != nil {
		return req.Count
	}
	return 0
}

// GetFixedStaffForShiftDate 获取指定班次和日期的固定排班人员ID列表
func (ctx *CreateV4Context) GetFixedStaffForShiftDate(shiftID, date string) []string {
	return d_model.FindFixedAssignment(ctx.FixedAssignments, shiftID, date)
}

// GetPersonalNeedsForStaff 获取指定人员的所有需求
func (ctx *CreateV4Context) GetPersonalNeedsForStaff(staffID string) []*PersonalNeed {
	if needs, ok := ctx.PersonalNeeds[staffID]; ok {
		return needs
	}
	return nil
}

// AddPersonalNeed 添加个人需求
func (ctx *CreateV4Context) AddPersonalNeed(need *PersonalNeed) {
	if need == nil {
		return
	}
	if ctx.PersonalNeeds == nil {
		ctx.PersonalNeeds = make(map[string][]*PersonalNeed)
	}
	ctx.PersonalNeeds[need.StaffID] = append(ctx.PersonalNeeds[need.StaffID], need)
}

// GetAllDatesInRange 获取日期范围内的所有日期
func (ctx *CreateV4Context) GetAllDatesInRange() []string {
	if ctx.StartDate == "" || ctx.EndDate == "" {
		return []string{}
	}

	startTime, err := time.Parse("2006-01-02", ctx.StartDate)
	if err != nil {
		return []string{}
	}
	endTime, err := time.Parse("2006-01-02", ctx.EndDate)
	if err != nil {
		return []string{}
	}

	dates := make([]string, 0)
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}

// BuildFullSchedulePreview 构建全班次排班预览数据
func (ctx *CreateV4Context) BuildFullSchedulePreview() map[string]any {
	if ctx.WorkingDraft == nil || ctx.WorkingDraft.Shifts == nil {
		return map[string]any{
			"title":         "排班预览",
			"shifts":        map[string]any{},
			"shiftInfoList": []map[string]any{},
			"startDate":     ctx.StartDate,
			"endDate":       ctx.EndDate,
		}
	}

	shiftData := make(map[string]any)
	shiftInfoList := make([]map[string]any, 0)

	for _, shift := range ctx.SelectedShifts {
		shiftID := shift.ID
		shiftDraft, exists := ctx.WorkingDraft.Shifts[shiftID]

		days := make(map[string]map[string]any)
		dates := ctx.GetAllDatesInRange()
		for _, date := range dates {
			if exists && shiftDraft != nil && shiftDraft.Days != nil {
				dayShift, dateExists := shiftDraft.Days[date]
				if dateExists && dayShift != nil {
					days[date] = map[string]any{
						"staff":         dayShift.Staff,
						"staffIds":      dayShift.StaffIDs,
						"requiredCount": dayShift.RequiredCount,
						"actualCount":   dayShift.ActualCount,
					}
				} else {
					days[date] = map[string]any{
						"staff":         []string{},
						"staffIds":      []string{},
						"requiredCount": ctx.GetRequirement(shiftID, date),
						"actualCount":   0,
					}
				}
			} else {
				days[date] = map[string]any{
					"staff":         []string{},
					"staffIds":      []string{},
					"requiredCount": ctx.GetRequirement(shiftID, date),
					"actualCount":   0,
				}
			}
		}

		shiftData[shiftID] = map[string]any{
			"shiftId":   shiftID,
			"shiftName": shift.Name,
			"priority":  shift.SchedulingPriority,
			"days":      days,
		}
		shiftInfoList = append(shiftInfoList, map[string]any{
			"id":          shiftID,
			"name":        shift.Name,
			"startTime":   shift.StartTime,
			"endTime":     shift.EndTime,
			"isOvernight": shift.IsOvernight,
			"type":        shift.Type,
		})
	}

	// 构建有序员工列表（按工号排序，用于导出时人员行排序）
	staffList := make([]map[string]any, 0, len(ctx.AllStaff))
	for _, emp := range ctx.AllStaff {
		if emp == nil {
			continue
		}
		staffList = append(staffList, map[string]any{
			"id":         emp.ID,
			"name":       emp.Name,
			"employeeId": emp.EmployeeID,
		})
	}

	return map[string]any{
		"title":         "排班预览 (V4)",
		"shifts":        shiftData,
		"shiftInfoList": shiftInfoList,
		"staffList":     staffList,
		"startDate":     ctx.StartDate,
		"endDate":       ctx.EndDate,
	}
}

// ============================================================
// Payload 结构体定义
// ============================================================

// PeriodConfirmPayload 时间确认的 Payload
type PeriodConfirmPayload struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// ShiftsConfirmPayload 班次确认的 Payload
type ShiftsConfirmPayload struct {
	ShiftIDs []string `json:"shiftIds"`
}

// StaffCountConfirmPayload 人数配置确认的 Payload
type StaffCountConfirmPayload struct {
	Requirements []d_model.ShiftDateRequirement `json:"requirements,omitempty"`
}

// PersonalNeedsConfirmPayload 个人需求确认的 Payload
type PersonalNeedsConfirmPayload struct {
	Confirmed bool `json:"confirmed"`
}

// ReviewConfirmPayload 审核确认的 Payload
type ReviewConfirmPayload struct {
	Action string `json:"action"` // confirm/modify/cancel
}
