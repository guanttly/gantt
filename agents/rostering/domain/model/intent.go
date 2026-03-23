package model

// IntentType 系统支持的通用意图枚举
type IntentType string

const (
	// 一级意图 - 主分类
	IntentSchedule IntentType = "schedule" // 排班类
	IntentRule     IntentType = "rule"     // 规则类
	IntentDept     IntentType = "dept"     // 科室类
	IntentGeneral  IntentType = "general"  // 通用类
	IntentUnknown  IntentType = "unknown"  // 未识别

	// 二级意图 - 排班类
	IntentScheduleCreate IntentType = "schedule.create" // 创建/发起排班
	IntentScheduleAdjust IntentType = "schedule.adjust" // 调整已有排班（调班/换班）
	IntentScheduleQuery  IntentType = "schedule.query"  // 查询排班

	// 二级意图 - 规则类
	IntentRuleCreate IntentType = "rule.create" // 创建排班规则
	IntentRuleUpdate IntentType = "rule.update" // 更新排班规则
	IntentRuleQuery  IntentType = "rule.query"  // 查询排班规则
	IntentRuleDelete IntentType = "rule.delete" // 删除排班规则

	// 二级意图 - 科室类
	IntentDeptStaffCreate    IntentType = "dept.staff.create"    // 新增员工
	IntentDeptStaffUpdate    IntentType = "dept.staff.update"    // 更新员工信息
	IntentDeptStaffStatus    IntentType = "dept.staff.status"    // 更新员工状态
	IntentDeptStaffDelete    IntentType = "dept.staff.delete"    // 删除员工
	IntentDeptTeamCreate     IntentType = "dept.team.create"     // 新增团队
	IntentDeptTeamUpdate     IntentType = "dept.team.update"     // 更新团队信息
	IntentDeptTeamAssign     IntentType = "dept.team.assign"     // 人员团队分配
	IntentDeptSkillCreate    IntentType = "dept.skill.create"    // 新增技能
	IntentDeptSkillUpdate    IntentType = "dept.skill.update"    // 更新技能信息
	IntentDeptSkillAssign    IntentType = "dept.skill.assign"    // 人员技能分配
	IntentDeptLocationCreate IntentType = "dept.location.create" // 新增地点
	IntentDeptLocationUpdate IntentType = "dept.location.update" // 更新地点信息

	// 二级意图 - 通用类
	IntentGeneralHelp      IntentType = "general.help"      // 帮助/指引
	IntentGeneralSmallTalk IntentType = "general.smalltalk" // 闲聊（可忽略业务处理）
)

// IntentResult 单条意图识别结果
type IntentResult struct {
	Type       IntentType     `json:"type" label:"操作类型"`
	Confidence float64        `json:"confidence" label:"置信度"`
	Summary    string         `json:"summary,omitempty" label:"说明"`
	Arguments  map[string]any `json:"arguments,omitempty" label:"参数"` // 结构化槽位，如 bizDateRange/department/modality/ruleKey/staffId 等
	Raw        string         `json:"raw,omitempty" label:"原始内容"`     // AI 原始片段（可选）
}

// IntentDetectionPayload 用于写入 Context.Extra 中的统一结构
type IntentDetectionPayload struct {
	MessageIndex int             `json:"messageIndex"`
	Results      []*IntentResult `json:"results"`
	Model        string          `json:"model"`
}

// SubIntentAnalysisResult 二级意图分析结果
type SubIntentAnalysisResult struct {
	MainIntent   IntentType      `json:"mainIntent"`       // 一级意图
	SubIntents   []*IntentResult `json:"subIntents"`       // 二级意图列表
	PlanID       string          `json:"planId,omitempty"` // 执行计划ID（仅科室类）
	MessageIndex int             `json:"messageIndex"`
	Model        string          `json:"model"`
}

// ============================================================
// 意图到工作流的映射
// ============================================================

// IntentWorkflowMapping 意图到工作流的映射配置
type IntentWorkflowMapping struct {
	WorkflowName string // 工作流名称（如 "schedule.create", "schedule.adjust"）
	Event        string // 触发事件（如 "start"）
	Implemented  bool   // 是否已实现
}

// intentWorkflowMappings 意图类型到工作流的静态映射表
var intentWorkflowMappings = map[IntentType]*IntentWorkflowMapping{
	// 排班类意图
	IntentSchedule:       {WorkflowName: "schedule_v2.create", Event: "start", Implemented: true},
	IntentScheduleCreate: {WorkflowName: "schedule_v2.create", Event: "start", Implemented: true},
	IntentScheduleAdjust: {WorkflowName: "schedule.adjust", Event: "start", Implemented: true},
	IntentScheduleQuery:  {WorkflowName: "schedule.query", Event: "start", Implemented: false},

	// 规则类意图
	IntentRule:       {WorkflowName: "rule.create", Event: "start", Implemented: false},
	IntentRuleCreate: {WorkflowName: "rule.create", Event: "start", Implemented: false},
	IntentRuleUpdate: {WorkflowName: "rule.update", Event: "start", Implemented: false},
	IntentRuleQuery:  {WorkflowName: "rule.query", Event: "start", Implemented: false},
	IntentRuleDelete: {WorkflowName: "rule.delete", Event: "start", Implemented: false},

	// 科室类意图 - 暂不实现
	IntentDept:               {WorkflowName: "", Event: "", Implemented: false},
	IntentDeptStaffCreate:    {WorkflowName: "dept.staff.create", Event: "start", Implemented: false},
	IntentDeptStaffUpdate:    {WorkflowName: "dept.staff.update", Event: "start", Implemented: false},
	IntentDeptStaffStatus:    {WorkflowName: "dept.staff.status", Event: "start", Implemented: false},
	IntentDeptStaffDelete:    {WorkflowName: "dept.staff.delete", Event: "start", Implemented: false},
	IntentDeptTeamCreate:     {WorkflowName: "dept.team.create", Event: "start", Implemented: false},
	IntentDeptTeamUpdate:     {WorkflowName: "dept.team.update", Event: "start", Implemented: false},
	IntentDeptTeamAssign:     {WorkflowName: "dept.team.assign", Event: "start", Implemented: false},
	IntentDeptSkillCreate:    {WorkflowName: "dept.skill.create", Event: "start", Implemented: false},
	IntentDeptSkillUpdate:    {WorkflowName: "dept.skill.update", Event: "start", Implemented: false},
	IntentDeptSkillAssign:    {WorkflowName: "dept.skill.assign", Event: "start", Implemented: false},
	IntentDeptLocationCreate: {WorkflowName: "dept.location.create", Event: "start", Implemented: false},
	IntentDeptLocationUpdate: {WorkflowName: "dept.location.update", Event: "start", Implemented: false},

	// 通用类意图 - 不触发工作流
	IntentGeneral:          {WorkflowName: "", Event: "", Implemented: true},
	IntentGeneralHelp:      {WorkflowName: "", Event: "", Implemented: true},
	IntentGeneralSmallTalk: {WorkflowName: "", Event: "", Implemented: true},

	// 未知意图 - 不触发工作流
	IntentUnknown: {WorkflowName: "", Event: "", Implemented: true},
}

// GetWorkflowMapping 根据意图类型获取工作流映射
// 返回 nil 表示该意图类型未配置映射
func GetWorkflowMapping(intentType IntentType) *IntentWorkflowMapping {
	if mapping, ok := intentWorkflowMappings[intentType]; ok {
		return mapping
	}
	return nil
}

// GetWorkflowEvent 根据意图类型获取触发事件
// 返回空字符串表示该意图不触发工作流
func GetWorkflowEvent(intentType IntentType) string {
	mapping := GetWorkflowMapping(intentType)
	if mapping == nil || !mapping.Implemented || mapping.Event == "" {
		return ""
	}
	return mapping.Event
}

// IsIntentImplemented 检查意图类型对应的工作流是否已实现
func IsIntentImplemented(intentType IntentType) bool {
	mapping := GetWorkflowMapping(intentType)
	return mapping != nil && mapping.Implemented
}
