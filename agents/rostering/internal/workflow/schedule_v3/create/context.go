package create

import (
	"time"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// CreateV3Context - 排班创建工作流 V3 上下文
// 存储整个工作流的状态和数据
// ============================================================

// CreateV3Context 创建工作流的核心上下文
type CreateV3Context struct {
	// ========== 基础信息（来自 InfoCollect 子工作流） ==========

	// StartDate 排班周期开始日期 (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// EndDate 排班周期结束日期 (YYYY-MM-DD)
	EndDate string `json:"endDate"`

	// SelectedShifts 用户选定的所有班次列表
	SelectedShifts []*d_model.Shift `json:"selectedShifts"`

	// StaffRequirements 人员配置需求 - 新强类型数组（替换嵌套map）
	StaffRequirements []d_model.ShiftDateRequirement `json:"staffRequirements"`

	// AllStaff 所有员工列表（用于姓名映射、信息检索，候选人员在L3动态计算）
	AllStaff []*d_model.Employee `json:"allStaffList"`

	// ShiftMembersMap 各班次专属人员映射（shiftID → 成员列表，L3 ComputeCandidateStaff 的基础候选池）
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

	// AdjustmentScope 调整范围: "plan" | "task"
	AdjustmentScope string `json:"adjustmentScope,omitempty"`

	// ========== 渐进式任务计划（V3核心） ==========

	// ProgressiveTaskPlan 渐进式任务计划
	ProgressiveTaskPlan *d_model.ProgressiveTaskPlan `json:"progressiveTaskPlan"`

	// CurrentTaskIndex 当前任务索引
	CurrentTaskIndex int `json:"currentTaskIndex"`

	// TaskResults 任务执行结果 (task_id -> TaskResult)
	TaskResults map[string]*d_model.TaskResult `json:"taskResults"`

	// ========== 已占位信息（累积约束） ==========

	// OccupiedSlots 已占位的人员-日期-班次 - 新强类型数组
	OccupiedSlots []d_model.StaffOccupiedSlot `json:"occupiedSlots"`

	// ExistingScheduleMarks 已有排班标记（用于时段冲突检查） - 新强类型数组
	ExistingScheduleMarks []d_model.StaffScheduleMark `json:"existingScheduleMarks"`

	// ========== 固定排班缓存（P1优化：避免重复获取） ==========

	// FixedAssignments 固定排班配置 - 新强类型数组
	FixedAssignments []d_model.CtxFixedShiftAssignment `json:"fixedAssignments"`

	// ========== 变更追踪（V3核心改造） ==========

	// BaselineSchedule 基准排班（固定排班填充后的状态）
	BaselineSchedule *d_model.ScheduleDraft `json:"baselineSchedule"`

	// ChangeBatches 变更历史（按任务记录）
	ChangeBatches []*d_model.ScheduleChangeBatch `json:"changeBatches"`

	// ========== 最终结果 ==========

	// WorkingDraft 工作草稿（当前排班状态，保持向后兼容的JSON名称）
	WorkingDraft *d_model.ScheduleDraft `json:"finalScheduleDraft"`

	// SavedScheduleID 保存后的排班ID（保存成功后填充）
	SavedScheduleID string `json:"savedScheduleId"`

	// ========== 统计信息 ==========

	// CompletedTaskCount 已完成的任务总数
	CompletedTaskCount int `json:"completedTaskCount"`

	// FailedTaskCount 失败的任务总数
	FailedTaskCount int `json:"failedTaskCount"`

	// SkippedTaskCount 跳过的任务总数
	SkippedTaskCount int `json:"skippedTaskCount"`

	// ========== 最终校验与补充任务（渐进式校验） ==========

	// SupplementRound 当前补充任务轮次（最大2轮）
	SupplementRound int `json:"supplementRound"`

	// NeedsManualIntervention 是否需要人工介入（补充轮次用尽仍有缺员）
	NeedsManualIntervention bool `json:"needsManualIntervention"`

	// ========== 全局评审结果（规则和个人需求校验） ==========

	// GlobalReviewResult 全局评审执行结果
	GlobalReviewResult *d_model.GlobalReviewResult `json:"globalReviewResult,omitempty"`

	// ========== 组织信息 ==========

	// OrgID 组织ID
	OrgID string `json:"orgId,omitempty"`

	// StaffList 候选人员列表（向后兼容，等同于 AllStaff 的子集）
	StaffList []*d_model.Employee `json:"staffList,omitempty"`
}

// ============================================================
// PersonalNeed - 个人需求定义（复用V2的结构）
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
// 上下文初始化和辅助方法
// ============================================================

// NewCreateV3Context 创建新的工作流上下文
func NewCreateV3Context() *CreateV3Context {
	return &CreateV3Context{
		PersonalNeeds:          make(map[string][]*PersonalNeed),
		OccupiedSlots:          make([]d_model.StaffOccupiedSlot, 0),
		ExistingScheduleMarks:  make([]d_model.StaffScheduleMark, 0),
		StaffRequirements:      make([]d_model.ShiftDateRequirement, 0),
		FixedAssignments:       make([]d_model.CtxFixedShiftAssignment, 0),
		AllStaff:               make([]*d_model.Employee, 0),
		ShiftMembersMap:        make(map[string][]*d_model.Employee),
		StaffLeaves:            make(map[string][]*d_model.LeaveRecord),
		TaskResults:            make(map[string]*d_model.TaskResult),
		CurrentTaskIndex:       0,
		CompletedTaskCount:     0,
		FailedTaskCount:        0,
		SkippedTaskCount:       0,
		PersonalNeedsConfirmed: false,
	}
}

// IsStaffOccupied 检查人员在指定日期是否已被占位
func (ctx *CreateV3Context) IsStaffOccupied(staffID, date string) bool {
	return d_model.IsStaffOccupiedOnDate(ctx.OccupiedSlots, staffID, date)
}

// OccupySlot 占位：记录人员在指定日期被分配到某班次
func (ctx *CreateV3Context) OccupySlot(staffID, date, shiftID string) {
	ctx.OccupiedSlots = d_model.AddOccupiedSlot(ctx.OccupiedSlots, d_model.StaffOccupiedSlot{
		StaffID: staffID,
		Date:    date,
		ShiftID: shiftID,
	})
}

// GetOccupiedShift 获取人员在指定日期已被分配的班次ID
func (ctx *CreateV3Context) GetOccupiedShift(staffID, date string) string {
	slot := d_model.FindOccupiedSlot(ctx.OccupiedSlots, staffID, date)
	if slot != nil {
		return slot.ShiftID
	}
	return ""
}

// PrepareForSerialization 准备序列化（预留用于未来扩展）
func (ctx *CreateV3Context) PrepareForSerialization() {
	// JSON tag已直接设置在主字段上，使用强类型数组
	// 前端需要适配新的数组格式
}

// GetRequirement 获取指定班次和日期的人员需求
func (ctx *CreateV3Context) GetRequirement(shiftID, date string) int {
	req := d_model.FindRequirement(ctx.StaffRequirements, shiftID, date)
	if req != nil {
		return req.Count
	}
	return 0
}

// GetFixedStaffForShiftDate 获取指定班次和日期的固定排班人员ID列表
func (ctx *CreateV3Context) GetFixedStaffForShiftDate(shiftID, date string) []string {
	return d_model.FindFixedAssignment(ctx.FixedAssignments, shiftID, date)
}

// GetPersonalNeedsForStaff 获取指定人员的所有需求
func (ctx *CreateV3Context) GetPersonalNeedsForStaff(staffID string) []*PersonalNeed {
	if needs, ok := ctx.PersonalNeeds[staffID]; ok {
		return needs
	}
	return nil
}

// AddPersonalNeed 添加个人需求
func (ctx *CreateV3Context) AddPersonalNeed(need *PersonalNeed) {
	if need == nil {
		return
	}
	if ctx.PersonalNeeds == nil {
		ctx.PersonalNeeds = make(map[string][]*PersonalNeed)
	}
	ctx.PersonalNeeds[need.StaffID] = append(ctx.PersonalNeeds[need.StaffID], need)
}

// IncrementTaskProgress 增加当前任务进度
func (ctx *CreateV3Context) IncrementTaskProgress() {
	ctx.CurrentTaskIndex++
}

// IsAllTasksComplete 判断所有任务是否完成
func (ctx *CreateV3Context) IsAllTasksComplete() bool {
	if ctx.ProgressiveTaskPlan == nil || len(ctx.ProgressiveTaskPlan.Tasks) == 0 {
		return false
	}
	return ctx.CurrentTaskIndex >= len(ctx.ProgressiveTaskPlan.Tasks)
}

// GetCurrentTask 获取当前任务
func (ctx *CreateV3Context) GetCurrentTask() *d_model.ProgressiveTask {
	if ctx.ProgressiveTaskPlan == nil || ctx.CurrentTaskIndex >= len(ctx.ProgressiveTaskPlan.Tasks) {
		return nil
	}
	return ctx.ProgressiveTaskPlan.Tasks[ctx.CurrentTaskIndex]
}

// GetTaskByID 根据任务ID获取任务
func (ctx *CreateV3Context) GetTaskByID(taskID string) *d_model.ProgressiveTask {
	if ctx.ProgressiveTaskPlan == nil {
		return nil
	}
	for _, task := range ctx.ProgressiveTaskPlan.Tasks {
		if task.ID == taskID {
			return task
		}
	}
	return nil
}

// BuildFullSchedulePreview 构建全班次排班预览数据
// 渐进式排班每次都应该展示全部班次的当前状态
// 返回格式与前端 MultiShiftScheduleDialog.vue 的接口匹配
func (ctx *CreateV3Context) BuildFullSchedulePreview() map[string]any {
	if ctx.WorkingDraft == nil || ctx.WorkingDraft.Shifts == nil {
		return map[string]any{
			"title":         "排班预览",
			"shifts":        map[string]any{},
			"shiftInfoList": []map[string]any{},
			"startDate":     ctx.StartDate,
			"endDate":       ctx.EndDate,
		}
	}

	// 构建班次ID到信息的映射
	shiftInfoMap := make(map[string]*d_model.Shift)
	for _, shift := range ctx.SelectedShifts {
		shiftInfoMap[shift.ID] = shift
	}

	shiftData := make(map[string]any)
	shiftInfoList := make([]map[string]any, 0)

	// 遍历所有选中的班次（全班次预览）
	for _, shift := range ctx.SelectedShifts {
		shiftID := shift.ID
		shiftDraft, exists := ctx.WorkingDraft.Shifts[shiftID]

		// 构建该班次的排班数据
		days := make(map[string]map[string]any)

		// 使用日期范围内的所有日期
		dates := ctx.getAllDatesInRange()
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
					// 日期存在但没有排班数据
					days[date] = map[string]any{
						"staff":         []string{},
						"staffIds":      []string{},
						"requiredCount": ctx.GetRequirement(shiftID, date),
						"actualCount":   0,
					}
				}
			} else {
				// 班次不存在
				days[date] = map[string]any{
					"staff":         []string{},
					"staffIds":      []string{},
					"requiredCount": ctx.GetRequirement(shiftID, date),
					"actualCount":   0,
				}
			}
		}

		// 添加班次数据
		shiftData[shiftID] = map[string]any{
			"shiftId":   shiftID,
			"shiftName": shift.Name,
			"priority":  shift.SchedulingPriority,
			"days":      days,
		}
		shiftInfoList = append(shiftInfoList, map[string]any{
			"id":   shiftID,
			"name": shift.Name,
		})
	}

	// 获取当前任务信息（用于显示标题）
	title := "排班预览"
	if task := ctx.GetCurrentTask(); task != nil {
		title = task.Title + " - 排班预览"
	}

	return map[string]any{
		"title":         title,
		"shifts":        shiftData,
		"shiftInfoList": shiftInfoList,
		"startDate":     ctx.StartDate,
		"endDate":       ctx.EndDate,
	}
}

// getAllDatesInRange 获取日期范围内的所有日期
func (ctx *CreateV3Context) getAllDatesInRange() []string {
	if ctx.StartDate == "" || ctx.EndDate == "" {
		return []string{}
	}

	// 解析日期
	startTime, err := parseDate(ctx.StartDate)
	if err != nil {
		return []string{}
	}
	endTime, err := parseDate(ctx.EndDate)
	if err != nil {
		return []string{}
	}

	// 生成日期列表
	dates := make([]string, 0)
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
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
