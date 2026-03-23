package tool

import (
	"context"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/tool/conversation"
	"jusha/gantt/mcp/rostering/tool/department"
	"jusha/gantt/mcp/rostering/tool/employee"
	"jusha/gantt/mcp/rostering/tool/fixed_assignment"
	"jusha/gantt/mcp/rostering/tool/group"
	"jusha/gantt/mcp/rostering/tool/leave"
	"jusha/gantt/mcp/rostering/tool/rule"
	"jusha/gantt/mcp/rostering/tool/scheduling"
	"jusha/gantt/mcp/rostering/tool/shift"
	"jusha/gantt/mcp/rostering/tool/system_setting"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"

	internalRepo "jusha/gantt/mcp/rostering/internal/repository"
	internalService "jusha/gantt/mcp/rostering/internal/service"
)

// ToolManager 负责根据配置初始化并持有工具
type ToolManager struct {
	logger       logging.ILogger
	cfg          config.IRosteringConfigurator
	namingClient naming_client.INamingClient
	tools        []mcp.ITool
}

func NewToolManager(logger logging.ILogger, cfg config.IRosteringConfigurator, namingClient naming_client.INamingClient) *ToolManager {
	return &ToolManager{
		logger:       logger,
		cfg:          cfg,
		namingClient: namingClient,
	}
}

// GetTools 返回所有已初始化的工具
func (tm *ToolManager) GetTools() []mcp.ITool {
	return tm.tools
}

// Init 根据配置初始化工具
func (tm *ToolManager) Init(ctx context.Context) error {
	tm.tools = []mcp.ITool{}

	// 创建 repository provider
	repoProvider := internalRepo.NewRepositoryProviderBuilder().
		WithLogger(tm.logger).
		WithConfigurator(tm.cfg).
		WithNamingClient(tm.namingClient).
		Build()

	// 创建 service provider
	serviceProvider := internalService.NewServiceProviderBuilder().
		WithLogger(tm.logger).
		WithConfigurator(tm.cfg).
		WithRepositoryProvider(repoProvider).
		Build()

	// 注册员工管理工具
	tm.tools = append(tm.tools,
		employee.NewCreateEmployeeTool(tm.logger, serviceProvider),
		employee.NewListEmployeesTool(tm.logger, serviceProvider),
		employee.NewGetEmployeeTool(tm.logger, serviceProvider),
		employee.NewUpdateEmployeeTool(tm.logger, serviceProvider),
		employee.NewDeleteEmployeeTool(tm.logger, serviceProvider),
	)

	// 注册部门管理工具
	tm.tools = append(tm.tools,
		department.NewCreateDepartmentTool(tm.logger, serviceProvider),
		department.NewListDepartmentsTool(tm.logger, serviceProvider),
		department.NewGetDepartmentTool(tm.logger, serviceProvider),
		department.NewUpdateDepartmentTool(tm.logger, serviceProvider),
		department.NewDeleteDepartmentTool(tm.logger, serviceProvider),
		// department.NewGetDepartmentTreeTool(tm.logger, serviceProvider), // TODO: 实现树形结构工具
	)

	// 注册分组管理工具
	tm.tools = append(tm.tools,
		group.NewCreateGroupTool(tm.logger, serviceProvider),
		group.NewListGroupsTool(tm.logger, serviceProvider),
		group.NewGetGroupTool(tm.logger, serviceProvider),
		group.NewUpdateGroupTool(tm.logger, serviceProvider),
		group.NewDeleteGroupTool(tm.logger, serviceProvider),
		group.NewGetGroupMembersTool(tm.logger, serviceProvider),
		group.NewAddGroupMemberTool(tm.logger, serviceProvider),
		group.NewRemoveGroupMemberTool(tm.logger, serviceProvider),
	)

	// 注册班次管理工具
	tm.tools = append(tm.tools,
		shift.NewCreateShiftTool(tm.logger, serviceProvider),
		shift.NewListShiftsTool(tm.logger, serviceProvider),
		shift.NewGetShiftTool(tm.logger, serviceProvider),
		shift.NewUpdateShiftTool(tm.logger, serviceProvider),
		shift.NewDeleteShiftTool(tm.logger, serviceProvider),
		shift.NewSetShiftGroupsTool(tm.logger, serviceProvider),
		shift.NewAddShiftGroupTool(tm.logger, serviceProvider),
		shift.NewRemoveShiftGroupTool(tm.logger, serviceProvider),
		shift.NewGetShiftGroupsTool(tm.logger, serviceProvider),
		shift.NewGetShiftGroupMembersTool(tm.logger, serviceProvider),
		shift.NewToggleShiftStatusTool(tm.logger, serviceProvider),
		shift.NewGetWeeklyStaffTool(tm.logger, serviceProvider),
		shift.NewSetWeeklyStaffTool(tm.logger, serviceProvider),
		shift.NewCalculateStaffingTool(tm.logger, serviceProvider),
	)

	// 注册排班管理工具
	tm.tools = append(tm.tools,
		scheduling.NewBatchAssignScheduleTool(tm.logger, serviceProvider),
		scheduling.NewGetScheduleByDateRangeTool(tm.logger, serviceProvider),
		scheduling.NewGetScheduleSummaryTool(tm.logger, serviceProvider),
		scheduling.NewDeleteScheduleTool(tm.logger, serviceProvider),
	)

	// 注册请假管理工具
	tm.tools = append(tm.tools,
		leave.NewCreateLeaveTool(tm.logger, serviceProvider),
		leave.NewListLeavesTool(tm.logger, serviceProvider),
		leave.NewGetLeaveTool(tm.logger, serviceProvider),
		leave.NewUpdateLeaveTool(tm.logger, serviceProvider),
		leave.NewDeleteLeaveTool(tm.logger, serviceProvider),
		leave.NewGetLeaveBalanceTool(tm.logger, serviceProvider),
	)

	// 注册排班规则工具
	tm.tools = append(tm.tools,
		rule.NewCreateRuleTool(tm.logger, serviceProvider),
		rule.NewListRulesTool(tm.logger, serviceProvider),
		rule.NewGetRuleTool(tm.logger, serviceProvider),
		rule.NewUpdateRuleTool(tm.logger, serviceProvider),
		rule.NewDeleteRuleTool(tm.logger, serviceProvider),
		rule.NewGetRulesForEmployeeTool(tm.logger, serviceProvider),
		rule.NewGetRulesForGroupTool(tm.logger, serviceProvider),
		rule.NewGetRulesForShiftTool(tm.logger, serviceProvider),
		// 批量查询规则工具
		rule.NewGetRulesForEmployeesTool(tm.logger, serviceProvider),
		rule.NewGetRulesForShiftsTool(tm.logger, serviceProvider),
		rule.NewGetRulesForGroupsTool(tm.logger, serviceProvider),
		// V4新增工具
		rule.NewParseRuleTool(tm.logger, serviceProvider),
		rule.NewGetRuleStatisticsTool(tm.logger, serviceProvider),
		rule.NewBatchParseRulesTool(tm.logger, serviceProvider),
		rule.NewGetRuleDependenciesTool(tm.logger, serviceProvider),
		rule.NewAddRuleDependencyTool(tm.logger, serviceProvider),
		rule.NewGetRuleConflictsTool(tm.logger, serviceProvider),
		rule.NewPreviewMigrationTool(tm.logger, serviceProvider),
		rule.NewExecuteMigrationTool(tm.logger, serviceProvider),
		// V4.1新增工具 - 班次关系
		rule.NewCreateRuleWithRelationsTool(tm.logger, serviceProvider),
		rule.NewGetRulesBySubjectShiftTool(tm.logger, serviceProvider),
		rule.NewGetRulesByObjectShiftTool(tm.logger, serviceProvider),
	)

	// 注册固定人员配置工具
	tm.tools = append(tm.tools,
		fixed_assignment.NewCalculateFixedScheduleTool(tm.logger, serviceProvider),
		fixed_assignment.NewCalculateMultipleFixedSchedulesTool(tm.logger, serviceProvider),
	)

	// 注册系统设置工具
	tm.tools = append(tm.tools,
		system_setting.NewGetSystemSettingTool(tm.logger, serviceProvider),
		system_setting.NewSetSystemSettingTool(tm.logger, serviceProvider),
	)

	// 注册对话管理工具（调用管理服务）
	managementRepo := repoProvider.GetManagementRepository()
	tm.tools = append(tm.tools,
		conversation.NewCreateOrUpdateConversationTool(tm.logger, managementRepo),
	)

	tm.logger.Info("Rostering MCP tools initialized", "tool_count", len(tm.tools))
	return nil
}
