package utils

import (
	"context"
	"encoding/json"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// CoreV3 任务上下文管理
// ============================================================

const (
	// KeyCoreV3TaskContext CoreV3任务上下文在session中的key
	KeyCoreV3TaskContext = "core_v3_task_context"

	// KeyTaskExecutionContext 任务执行上下文在session中的key（L2层）
	KeyTaskExecutionContext = "task_execution_context"
)

// CoreV3TaskContext V3排班渐进式任务执行上下文
// 使用强类型数组替代嵌套map，提供类型安全和更好的可维护性
type CoreV3TaskContext struct {
	// ========== 组织信息 ==========
	OrgID string `json:"orgID"`

	// ========== 人员数据（统一管理） ==========

	// AllStaff 所有员工（用于姓名映射、信息查询）
	AllStaff []*d_model.Employee `json:"allStaffList"`

	// CandidateStaff 候选人员（可参与排班的员工，已过滤请假等）
	CandidateStaff []*d_model.Employee `json:"staffList"`

	// ShiftMembersMap 各班次专属人员映射（shiftID → 成员列表，L3 ComputeCandidateStaff 的基础候选池）
	ShiftMembersMap map[string][]*d_model.Employee `json:"shiftMembersMap,omitempty"`

	// OccupiedSlots 人员已占位信息（强类型数组）
	OccupiedSlots []d_model.StaffOccupiedSlot `json:"occupiedSlots"`

	// ScheduleMarks 人员排班标记（用于时段冲突检查，可选）
	ScheduleMarks []d_model.StaffScheduleMark `json:"scheduleMarks,omitempty"`

	// PersonalNeeds 个人需求 (staffID -> needs)
	PersonalNeeds map[string][]*d_model.PersonalNeed `json:"personalNeeds,omitempty"`

	// ========== 班次数据 ==========

	// Shifts 涉及的班次列表
	Shifts []*d_model.Shift `json:"shifts"`

	// StaffRequirements 班次人员需求（强类型数组）
	StaffRequirements []d_model.ShiftDateRequirement `json:"staffRequirements"`

	// FixedAssignments 固定排班配置（强类型数组）
	FixedAssignments []d_model.CtxFixedShiftAssignment `json:"fixedAssignments,omitempty"`

	// ========== 规则数据 ==========
	Rules []*d_model.Rule `json:"rules"`

	// ========== 任务数据 ==========
	Task       *d_model.ProgressiveTask `json:"task"`
	TaskResult *d_model.TaskResult      `json:"taskResult,omitempty"`

	// ========== 排班草稿 ==========

	// CurrentDraft 当前班次排班草案
	CurrentDraft *d_model.ShiftScheduleDraft `json:"currentDraft,omitempty"`

	// WorkingDraft 工作草案（所有班次）
	WorkingDraft *d_model.ScheduleDraft `json:"workingDraft,omitempty"`

	// ========== 渐进式执行检查点（用于中断恢复） ==========

	// ShiftCheckpoints 班次执行检查点（shiftID -> checkpoint）
	ShiftCheckpoints map[string]*d_model.ShiftExecutionCheckpoint `json:"shiftCheckpoints,omitempty"`

	// ========== LLM调用预构建数据（运行时构建，不序列化） ==========

	// StaffInfoForAI 预构建的AI用人员信息（避免每次调用时转换）
	StaffInfoForAI []*d_model.StaffInfoForAI `json:"-"`

	// ShiftRequirementsMap 班次人员需求映射（LLM调用时直接使用）
	// 格式: shiftID -> (date -> count)
	ShiftRequirementsMap map[string]map[string]int `json:"-"`

	// FixedAssignmentsMap 固定排班映射（LLM调用时直接使用）
	// 格式: date -> staffIDs（合并所有班次）
	FixedAssignmentsMap map[string][]string `json:"-"`

	// ========== ID映射（避免UUID泄露） ==========

	// StaffIDToName 人员ID到姓名映射（staffID -> staffName，用于显示）
	StaffIDToName map[string]string `json:"-"`

	// StaffNameToID 人员姓名到ID映射（staffName -> staffID，用于LLM返回中文名时的兜底查找）
	StaffNameToID map[string]string `json:"-"`

	// StaffForwardMappings 人员ID映射（真实ID -> 短ID，如 uuid -> staff_1）
	StaffForwardMappings map[string]string `json:"-"`

	// StaffReverseMappings 人员ID反向映射（短ID -> 真实ID，如 staff_1 -> uuid）
	StaffReverseMappings map[string]string `json:"-"`

	// ShiftForwardMappings 班次ID映射（真实ID -> 短ID）
	ShiftForwardMappings map[string]string `json:"-"`

	// ShiftReverseMappings 班次ID反向映射（短ID -> 真实ID）
	ShiftReverseMappings map[string]string `json:"-"`

	// RuleForwardMappings 规则ID映射（真实ID -> 短ID）
	RuleForwardMappings map[string]string `json:"-"`

	// RuleReverseMappings 规则ID反向映射（短ID -> 真实ID）
	RuleReverseMappings map[string]string `json:"-"`
}

// GetCoreV3TaskContext 从 session 获取任务上下文
func GetCoreV3TaskContext(ctx context.Context, wctx engine.Context) (*CoreV3TaskContext, error) {
	sess := wctx.Session()
	data, found, err := wctx.SessionService().GetData(ctx, sess.ID, KeyCoreV3TaskContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get task context: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("task context not found in session")
	}

	// 通过JSON序列化/反序列化转换
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context: %w", err)
	}

	var taskCtx CoreV3TaskContext
	if err := json.Unmarshal(jsonBytes, &taskCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	// ⚠️ 重新构建ID映射（因为 json:"-" 标签导致反序列化后丢失）
	taskCtx.rebuildIDMappings()

	return &taskCtx, nil
}

// SaveCoreV3TaskContext 保存任务上下文到 session
func SaveCoreV3TaskContext(ctx context.Context, wctx engine.Context, taskCtx *CoreV3TaskContext) error {
	sess := wctx.Session()
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, KeyCoreV3TaskContext, taskCtx); err != nil {
		return fmt.Errorf("failed to save task context: %w", err)
	}
	return nil
}

// GetCoreV3TaskContextFromSession 从 session 对象直接获取（用于子工作流）
func GetCoreV3TaskContextFromSession(sess *session.Session) (*CoreV3TaskContext, error) {
	if data, ok := sess.Data[KeyCoreV3TaskContext]; ok {
		// 通过JSON序列化/反序列化转换
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", err)
		}

		var taskCtx CoreV3TaskContext
		if err := json.Unmarshal(jsonBytes, &taskCtx); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}

		// ⚠️ 重新构建ID映射（因为 json:"-" 标签导致反序列化后丢失）
		taskCtx.rebuildIDMappings()

		return &taskCtx, nil
	}
	return nil, fmt.Errorf("task context not found in session")
}

// ============================================================
// CoreV3TaskContext 辅助方法
// ============================================================

// BuildLLMCache 构建LLM调用缓存（在上下文创建时调用一次）
func (ctx *CoreV3TaskContext) BuildLLMCache() {
	// 1. 构建 StaffInfoForAI
	ctx.StaffInfoForAI = d_model.NewStaffInfoListFromEmployees(ctx.CandidateStaff)

	// 2. 构建 ShiftRequirementsMap
	ctx.ShiftRequirementsMap = d_model.ConvertRequirementsToMap(ctx.StaffRequirements)

	// 3. 构建 FixedAssignmentsMap
	ctx.FixedAssignmentsMap = d_model.ConvertFixedAssignmentsToMap(ctx.FixedAssignments)

	// 4. 构建ID映射（避免UUID泄露给LLM）
	ctx.rebuildIDMappings()
}

// rebuildIDMappings 重新构建ID映射（从session反序列化后调用）
// 因为json:"-"标签导致这些映射不会被序列化，需要每次从session读取后重新构建
func (ctx *CoreV3TaskContext) rebuildIDMappings() {
	// ⚠️ 使用 AllStaff 构建映射，确保所有班次使用一致的ID映射
	if len(ctx.AllStaff) > 0 {
		ctx.StaffIDToName = BuildStaffIDToNameMapping(ctx.AllStaff)
		ctx.StaffNameToID = BuildStaffNameToIDMapping(ctx.AllStaff)
		ctx.StaffForwardMappings, ctx.StaffReverseMappings = BuildStaffIDMappings(ctx.AllStaff)
	}
	if len(ctx.Shifts) > 0 {
		ctx.ShiftForwardMappings, ctx.ShiftReverseMappings = BuildShiftIDMappings(ctx.Shifts)
	}
	if len(ctx.Rules) > 0 {
		ctx.RuleForwardMappings, ctx.RuleReverseMappings = BuildRuleIDMappings(ctx.Rules)
	}
}

// PrepareForSerialization 序列化前准备（预留用于未来扩展）
func (ctx *CoreV3TaskContext) PrepareForSerialization() {
	// JSON tag已直接设置在主字段上，无需额外处理
}

// GetStaffByID 根据ID获取员工
func (ctx *CoreV3TaskContext) GetStaffByID(staffID string) *d_model.Employee {
	for _, staff := range ctx.AllStaff {
		if staff.ID == staffID {
			return staff
		}
	}
	return nil
}

// IsStaffOccupied 检查人员在指定日期是否已被占位
func (ctx *CoreV3TaskContext) IsStaffOccupied(staffID, date string) bool {
	return d_model.IsStaffOccupiedOnDate(ctx.OccupiedSlots, staffID, date)
}

// AddOccupiedSlot 添加占位记录
func (ctx *CoreV3TaskContext) AddOccupiedSlot(slot d_model.StaffOccupiedSlot) {
	ctx.OccupiedSlots = d_model.AddOccupiedSlotIfNotExists(ctx.OccupiedSlots, slot)
}

// GetOccupiedSlot 获取人员在指定日期的占位信息
func (ctx *CoreV3TaskContext) GetOccupiedSlot(staffID, date string) *d_model.StaffOccupiedSlot {
	return d_model.FindOccupiedSlot(ctx.OccupiedSlots, staffID, date)
}

// GetRequirement 获取班次在指定日期的人员需求
func (ctx *CoreV3TaskContext) GetRequirement(shiftID, date string) int {
	req := d_model.FindRequirement(ctx.StaffRequirements, shiftID, date)
	if req != nil {
		return req.Count
	}
	return 0
}

// GetFixedAssignment 获取班次在指定日期的固定排班人员ID列表
func (ctx *CoreV3TaskContext) GetFixedAssignment(shiftID, date string) []string {
	return d_model.FindFixedAssignment(ctx.FixedAssignments, shiftID, date)
}

// ============================================================
// 检查点管理方法
// ============================================================

// GetShiftCheckpoint 获取班次的执行检查点
func (ctx *CoreV3TaskContext) GetShiftCheckpoint(shiftID string) *d_model.ShiftExecutionCheckpoint {
	if ctx.ShiftCheckpoints == nil {
		return nil
	}
	return ctx.ShiftCheckpoints[shiftID]
}

// SaveShiftCheckpoint 保存班次执行检查点
func (ctx *CoreV3TaskContext) SaveShiftCheckpoint(checkpoint *d_model.ShiftExecutionCheckpoint) {
	if ctx.ShiftCheckpoints == nil {
		ctx.ShiftCheckpoints = make(map[string]*d_model.ShiftExecutionCheckpoint)
	}
	ctx.ShiftCheckpoints[checkpoint.ShiftID] = checkpoint
}

// ClearShiftCheckpoint 清除班次执行检查点
func (ctx *CoreV3TaskContext) ClearShiftCheckpoint(shiftID string) {
	if ctx.ShiftCheckpoints != nil {
		delete(ctx.ShiftCheckpoints, shiftID)
	}
}

// HasShiftCheckpoint 检查是否存在班次检查点
func (ctx *CoreV3TaskContext) HasShiftCheckpoint(shiftID string) bool {
	if ctx.ShiftCheckpoints == nil {
		return false
	}
	_, exists := ctx.ShiftCheckpoints[shiftID]
	return exists
}

// ResolveStaffID 统一的人员ID解析：shortID -> UUID，或中文名 -> UUID，或原样返回
// 解析优先级：
//  1. shortID映射（staff_N -> uuid）
//  2. 中文名映射（姓名 -> uuid）
//  3. 原样返回（可能本身就是UUID）
func (ctx *CoreV3TaskContext) ResolveStaffID(id string) string {
	// 1. 先尝试shortID映射
	if ctx.StaffReverseMappings != nil {
		if mappedUUID, ok := ctx.StaffReverseMappings[id]; ok {
			return mappedUUID
		}
	}
	// 2. 再尝试中文名映射
	if ctx.StaffNameToID != nil {
		if mappedUUID, ok := ctx.StaffNameToID[id]; ok {
			return mappedUUID
		}
	}
	// 3. 都找不到，原样返回（可能本身就是UUID）
	return id
}

// MaskStaffID 将人员UUID转换为shortID（用于输出给LLM的prompt，禁止UUID泄露）
// 解析优先级：
//  1. 正向映射（uuid -> staff_N）
//  2. 原样返回（可能本身就是shortID）
func (ctx *CoreV3TaskContext) MaskStaffID(uuid string) string {
	if ctx.StaffForwardMappings != nil {
		if shortID, ok := ctx.StaffForwardMappings[uuid]; ok {
			return shortID
		}
	}
	return uuid
}

// MaskShiftID 将班次UUID转换为shortID（用于输出给LLM的prompt，禁止UUID泄露）
func (ctx *CoreV3TaskContext) MaskShiftID(uuid string) string {
	if ctx.ShiftForwardMappings != nil {
		if shortID, ok := ctx.ShiftForwardMappings[uuid]; ok {
			return shortID
		}
	}
	return uuid
}

// MaskRuleID 将规则UUID转换为shortID（用于输出给LLM的prompt，禁止UUID泄露）
func (ctx *CoreV3TaskContext) MaskRuleID(uuid string) string {
	if ctx.RuleForwardMappings != nil {
		if shortID, ok := ctx.RuleForwardMappings[uuid]; ok {
			return shortID
		}
	}
	return uuid
}

// GetStaffName 根据人员UUID获取姓名（找不到返回shortID兜底，绝不泄露UUID）
func (ctx *CoreV3TaskContext) GetStaffName(uuid string) string {
	if ctx.StaffIDToName != nil {
		if name, ok := ctx.StaffIDToName[uuid]; ok {
			return name
		}
	}
	// 姓名找不到时用shortID兜底，绝不直接返回UUID
	return ctx.MaskStaffID(uuid)
}
