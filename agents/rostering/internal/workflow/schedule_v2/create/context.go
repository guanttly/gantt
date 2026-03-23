package create

import (
	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// CreateV2Context - 排班创建工作流 V2 上下文
// 存储整个工作流的状态和数据
// ============================================================

// CreateV2Context 创建工作流的核心上下文
type CreateV2Context struct {
	// ========== 基础信息（来自 InfoCollect 子工作流） ==========

	// StartDate 排班周期开始日期 (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// EndDate 排班周期结束日期 (YYYY-MM-DD)
	EndDate string `json:"endDate"`

	// SelectedShifts 用户选定的所有班次列表
	SelectedShifts []*d_model.Shift `json:"selectedShifts"`

	// StaffRequirements 人员配置需求 (shift_id -> date -> 人数)
	StaffRequirements map[string]map[string]int `json:"staffRequirements"`

	// StaffList 班次关联的人员列表（用于AI排班）
	StaffList []*d_model.Employee `json:"staffList"`
	
	// AllStaffList 所有员工列表（用于信息检索，如固定班次预览）
	AllStaffList []*d_model.Employee `json:"allStaffList"`

	// StaffLeaves 人员请假信息 (staff_id -> 请假记录列表)
	StaffLeaves map[string][]*d_model.LeaveRecord `json:"staffLeaves"`

	// Rules 所有排班规则
	Rules []*d_model.Rule `json:"rules"`

	// ========== 个人需求（独立收集阶段） ==========

	// PersonalNeeds 个人需求列表 (staff_id -> 需求列表)
	PersonalNeeds map[string][]*PersonalNeed `json:"personalNeeds"`

	// PersonalNeedsConfirmed 个人需求是否已确认
	PersonalNeedsConfirmed bool `json:"personalNeedsConfirmed"`

	// ========== 已占位信息（累积约束） ==========

	// OccupiedSlots 已占位的人员-日期-班次映射
	// 格式: staff_id -> date (YYYY-MM-DD) -> shift_id
	// 用于确保同一人员在同一天不会被分配多个班次
	OccupiedSlots map[string]map[string]string `json:"occupiedSlots"`

	// ExistingScheduleMarks 已有排班标记（用于时段冲突检查）
	// 格式: staff_id -> date -> []ShiftMark
	ExistingScheduleMarks map[string]map[string][]*d_model.ShiftMark `json:"existingScheduleMarks"`

	// ========== 分阶段结果 ==========

	// FixedShiftResults 固定班次处理结果
	FixedShiftResults *PhaseResult `json:"fixedShiftResults"`

	// SpecialShiftResults 特殊班次排班结果
	SpecialShiftResults *PhaseResult `json:"specialShiftResults"`

	// NormalShiftResults 普通班次排班结果
	NormalShiftResults *PhaseResult `json:"normalShiftResults"`

	// ResearchShiftResults 科研班次排班结果
	ResearchShiftResults *PhaseResult `json:"researchShiftResults"`

	// FillShiftResults 填充班次处理结果
	FillShiftResults *PhaseResult `json:"fillShiftResults"`

	// ========== 当前阶段进度跟踪 ==========

	// CurrentPhase 当前处理的阶段
	CurrentPhase string `json:"currentPhase"`

	// CurrentShiftIndex 当前处理的班次索引（在阶段班次列表中）
	CurrentShiftIndex int `json:"currentShiftIndex"`

	// PhaseShiftList 当前阶段的班次列表
	PhaseShiftList []*d_model.Shift `json:"phaseShiftList"`

	// TotalShiftsInPhase 当前阶段的总班次数
	TotalShiftsInPhase int `json:"totalShiftsInPhase"`

	// ========== 分类后的班次列表 ==========

	// ClassifiedShifts 按类型分类的班次
	// Key: shift type (fixed, special, normal, research, fill)
	// Value: 该类型的班次列表
	ClassifiedShifts map[string][]*d_model.Shift `json:"classifiedShifts"`

	// ========== 统计信息 ==========

	// CompletedShiftCount 已完成排班的班次总数
	CompletedShiftCount int `json:"completedShiftCount"`

	// SkippedShiftCount 被跳过的班次总数
	SkippedShiftCount int `json:"skippedShiftCount"`

	// FailedShiftCount 失败的班次总数
	FailedShiftCount int `json:"failedShiftCount"`

	// PhaseStatistics 各阶段统计信息
	PhaseStatistics map[string]*PhaseStatistics `json:"phaseStatistics"`

	// ========== 最终结果 ==========

	// FinalScheduleDraft 最终排班草案（汇总所有阶段结果）
	FinalScheduleDraft *d_model.ScheduleDraft `json:"finalScheduleDraft"`

	// SavedScheduleID 保存后的排班ID（保存成功后填充）
	SavedScheduleID string `json:"savedScheduleId"`
}

// ============================================================
// PersonalNeed - 个人需求定义
// ============================================================

// PersonalNeed 个人需求（常态化或临时）
type PersonalNeed struct {
	// StaffID 人员ID
	StaffID string `json:"staffId"`

	// StaffName 人员姓名
	StaffName string `json:"staffName"`

	// NeedType 需求类型: "permanent" (常态化) | "temporary" (临时)
	NeedType string `json:"needType"`

	// RequestType 请求类型: "prefer" (偏好) | "avoid" (回避) | "must" (必须)
	RequestType string `json:"requestType"`

	// TargetShiftID 目标班次ID（如果指定）
	TargetShiftID string `json:"targetShiftId,omitempty"`

	// TargetShiftName 目标班次名称
	TargetShiftName string `json:"targetShiftName,omitempty"`

	// TargetDates 目标日期列表 (YYYY-MM-DD)
	// 为空表示整个周期都生效
	TargetDates []string `json:"targetDates,omitempty"`

	// Description 需求描述
	Description string `json:"description"`

	// Priority 优先级 (1-10, 数字越小优先级越高)
	Priority int `json:"priority"`

	// RuleID 关联的规则ID（如果来自规则系统）
	RuleID string `json:"ruleId,omitempty"`

	// Source 来源: "rule" (从规则提取) | "user" (用户补充)
	Source string `json:"source"`

	// Confirmed 是否已确认
	Confirmed bool `json:"confirmed"`
}

// ============================================================
// PhaseResult - 阶段执行结果
// ============================================================

// PhaseResult 单个阶段的执行结果
type PhaseResult struct {
	// PhaseName 阶段名称
	PhaseName string `json:"phaseName"`

	// ShiftType 班次类型
	ShiftType string `json:"shiftType"`

	// ProcessedShifts 处理的班次列表
	ProcessedShifts []string `json:"processedShifts"` // shift IDs

	// ScheduleDrafts 各班次的排班草案
	// Key: shift_id
	ScheduleDrafts map[string]*d_model.ShiftScheduleDraft `json:"scheduleDrafts"`

	// CompletedCount 成功完成的班次数
	CompletedCount int `json:"completedCount"`

	// SkippedCount 跳过的班次数
	SkippedCount int `json:"skippedCount"`

	// FailedCount 失败的班次数
	FailedCount int `json:"failedCount"`

	// StartTime 阶段开始时间
	StartTime string `json:"startTime"`

	// EndTime 阶段结束时间
	EndTime string `json:"endTime"`

	// Duration 阶段耗时（秒）
	Duration float64 `json:"duration"`

	// Summary 阶段总结
	Summary string `json:"summary"`

	// Errors 错误信息列表
	Errors []string `json:"errors,omitempty"`
}

// ============================================================
// PhaseStatistics - 阶段统计信息
// ============================================================

// PhaseStatistics 阶段统计信息
type PhaseStatistics struct {
	// PhaseName 阶段名称
	PhaseName string `json:"phaseName"`

	// TotalShifts 总班次数
	TotalShifts int `json:"totalShifts"`

	// CompletedShifts 完成的班次数
	CompletedShifts int `json:"completedShifts"`

	// SkippedShifts 跳过的班次数
	SkippedShifts int `json:"skippedShifts"`

	// FailedShifts 失败的班次数
	FailedShifts int `json:"failedShifts"`

	// TotalStaffAssigned 分配的总人次
	TotalStaffAssigned int `json:"totalStaffAssigned"`

	// AverageTimePerShift 每班次平均耗时（秒）
	AverageTimePerShift float64 `json:"averageTimePerShift"`
}

// ============================================================
// 上下文初始化和辅助方法
// ============================================================

// NewCreateV2Context 创建新的工作流上下文
func NewCreateV2Context() *CreateV2Context {
	return &CreateV2Context{
		PersonalNeeds:          make(map[string][]*PersonalNeed),
		OccupiedSlots:          make(map[string]map[string]string),
		ExistingScheduleMarks:  make(map[string]map[string][]*d_model.ShiftMark),
		ClassifiedShifts:       make(map[string][]*d_model.Shift),
		PhaseStatistics:        make(map[string]*PhaseStatistics),
		StaffRequirements:      make(map[string]map[string]int),
		StaffLeaves:            make(map[string][]*d_model.LeaveRecord),
		CurrentShiftIndex:      0,
		CompletedShiftCount:    0,
		SkippedShiftCount:      0,
		FailedShiftCount:       0,
		PersonalNeedsConfirmed: false,
	}
}

// IsStaffOccupied 检查人员在指定日期是否已被占位
func (ctx *CreateV2Context) IsStaffOccupied(staffID, date string) bool {
	if staffDates, ok := ctx.OccupiedSlots[staffID]; ok {
		_, occupied := staffDates[date]
		return occupied
	}
	return false
}

// OccupySlot 占位：记录人员在指定日期被分配到某班次
func (ctx *CreateV2Context) OccupySlot(staffID, date, shiftID string) {
	if ctx.OccupiedSlots[staffID] == nil {
		ctx.OccupiedSlots[staffID] = make(map[string]string)
	}
	ctx.OccupiedSlots[staffID][date] = shiftID
}

// GetOccupiedShift 获取人员在指定日期已被分配的班次ID
func (ctx *CreateV2Context) GetOccupiedShift(staffID, date string) string {
	if staffDates, ok := ctx.OccupiedSlots[staffID]; ok {
		return staffDates[date]
	}
	return ""
}

// AddPersonalNeed 添加个人需求
func (ctx *CreateV2Context) AddPersonalNeed(need *PersonalNeed) {
	if ctx.PersonalNeeds[need.StaffID] == nil {
		ctx.PersonalNeeds[need.StaffID] = make([]*PersonalNeed, 0)
	}
	ctx.PersonalNeeds[need.StaffID] = append(ctx.PersonalNeeds[need.StaffID], need)
}

// ============================================================
// Payload 结构体定义（用于工作流事件）
// ============================================================

// TemporaryNeedItem 单个临时需求项
type TemporaryNeedItem struct {
	StaffID       string   `json:"staffId"`                 // 人员ID（必填）
	RequestType   string   `json:"requestType"`             // 请求类型：prefer/must/avoid（必填）
	TargetShiftID string   `json:"targetShiftId,omitempty"` // 目标班次ID（可选）
	TargetDates   []string `json:"targetDates,omitempty"`   // 目标日期列表（可选）
	Description   string   `json:"description"`             // 需求描述（必填）
	Priority      int      `json:"priority"`                // 优先级（1-10，必填）
}

// AddTemporaryNeedPayload 添加临时需求的 Payload（支持批量添加）
type AddTemporaryNeedPayload struct {
	Needs []TemporaryNeedItem `json:"needs"` // 临时需求列表（至少一条）
}

// PeriodConfirmPayload 时间确认的 Payload
type PeriodConfirmPayload struct {
	StartDate string `json:"startDate"` // 开始日期（YYYY-MM-DD）
	EndDate   string `json:"endDate"`   // 结束日期（YYYY-MM-DD）
}

// ShiftsConfirmPayload 班次确认的 Payload
type ShiftsConfirmPayload struct {
	ShiftIDs []string `json:"shiftIds"` // 班次ID列表
}

// GetPersonalNeedsForStaff 获取指定人员的所有需求
func (ctx *CreateV2Context) GetPersonalNeedsForStaff(staffID string) []*PersonalNeed {
	if needs, ok := ctx.PersonalNeeds[staffID]; ok {
		return needs
	}
	return nil
}

// IncrementPhaseProgress 增加当前阶段进度
func (ctx *CreateV2Context) IncrementPhaseProgress() {
	ctx.CurrentShiftIndex++
}

// IsPhaseComplete 判断当前阶段是否完成
func (ctx *CreateV2Context) IsPhaseComplete() bool {
	return ctx.CurrentShiftIndex >= ctx.TotalShiftsInPhase
}

// ResetPhaseProgress 重置阶段进度（开始新阶段时调用）
func (ctx *CreateV2Context) ResetPhaseProgress(phase string, shifts []*d_model.Shift) {
	ctx.CurrentPhase = phase
	ctx.PhaseShiftList = shifts
	ctx.TotalShiftsInPhase = len(shifts)
	ctx.CurrentShiftIndex = 0
}
