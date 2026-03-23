# Rostering Go SDK

基于 MCP (Model Context Protocol) 的智能排班系统 Go 客户端 SDK，提供完整的排班管理、员工管理、部门管理、班次管理、规则管理等功能。

## 项目简介

本项目是一个 Go 语言实现的排班系统客户端 SDK，通过 MCP 协议与 Rostering MCP Server 通信，提供了简洁易用的 API 接口用于企业排班管理。项目采用清晰的分层架构，遵循领域驱动设计（DDD）原则。

## 主要特性

### 核心功能模块

- **员工管理 (Employee Management)**
  - 创建/更新/删除员工
  - 查询员工列表（支持分页、关键词搜索、部门过滤）
  - 获取员工详情
  - 完整的员工信息管理（UserID, Phone, Email, Position, Role, Status, HireDate等）

- **部门管理 (Department Management)**
  - 创建/更新部门
  - 查询部门列表
  - 支持部门层级结构（Code, Level, Path, ParentID）
  - 部门经理分配

- **分组管理 (Group Management)**
  - 创建/更新/删除分组
  - 查询分组列表（支持类型、状态过滤）
  - 分组成员管理（添加/移除成员）
  - 获取分组成员列表
  - 支持分组层级和扩展属性

- **班次管理 (Shift Management)**
  - 创建/更新/删除班次
  - 查询班次列表
  - 班次状态切换
  - 班次分组关联
  - 支持班次类型、时长、跨天、优先级等属性

- **规则管理 (Rule Management)**
  - 创建/更新/删除排班规则
  - 查询规则列表
  - 规则关联管理（班次、部门、分组、员工）
  - 获取员工适用规则
  - 支持复杂规则配置（ApplyScope, TimeScope, RuleData）

- **请假管理 (Leave Management)**
  - 创建/更新/删除请假记录
  - 查询请假列表
  - 获取假期余额

- **排班管理 (Scheduling)**
  - 批量排班分配
  - 按日期范围查询排班
  - 获取排班汇总统计
  - 删除排班记录

## 项目结构

```
sdk/rostering/
├── client/                    # 客户端实现层
│   ├── rostering-client.go   # Rostering客户端实现
│   ├── factory.go            # 工厂方法
│   ├── data-client.go        # [已废弃] 旧数据服务客户端
│   ├── context-client.go     # [已废弃] 旧上下文客户端
│   └── relational-graph-client.go  # [已废弃] 旧关系图谱客户端
├── domain/                    # 领域层（接口定义）
│   ├── rostering-client.go   # Rostering客户端接口
│   ├── data-client.go        # [已废弃] 旧数据客户端接口
│   ├── context-client.go     # [已废弃] 旧上下文接口
│   └── relational-graph-client.go  # [已废弃] 旧关系图谱接口
├── model/                     # 数据模型层
│   ├── employee.go           # 员工模型
│   ├── department.go         # 部门模型
│   ├── group.go              # 分组模型
│   ├── shift.go              # 班次模型
│   ├── rule.go               # 规则模型
│   ├── leave.go              # 请假模型
│   ├── scheduling.go         # 排班模型
│   └── page.go               # 分页模型
├── tool/                      # 工具定义层
│   └── define.go             # MCP工具名称常量定义
├── go.mod                     # Go模块定义
├── go.sum                     # Go依赖校验
└── README.md                  # 项目文档
```

## 技术栈

- **语言**: Go 1.23.2+
- **协议**: MCP (Model Context Protocol)
- **依赖管理**: Go Modules
- **核心依赖**: 
  - `jusha/mcp/pkg` - MCP 协议实现

## 快速开始

### 环境要求

- Go 1.23.2 或更高版本
- 访问 Rostering MCP Server 的网络权限

### 安装

```bash
go get jusha/agent/sdk/rostering
```

### 基本使用

```go
package main

import (
    "context"
    "jusha/mcp/pkg/logging"
    
    "jusha/agent/sdk/rostering/client"
    "jusha/agent/sdk/rostering/model"
    "jusha/mcp/pkg/mcp"
)

func main() {
    // 创建工具总线（假设已有 MCP 连接）
    toolBus := mcp.NewToolBus(...)
    logger := slog.Default()
    
    // 创建Rostering客户端
    rosteringClient := client.CreateRosteringClient(toolBus, logger)
    
    ctx := context.Background()
    
    // 示例1: 创建员工
    employeeID, err := rosteringClient.CreateEmployee(ctx, model.EmployeeCreateRequest{
        OrgID:      "org-001",
        EmployeeID: "E001",
        Name:       "张三",
        Phone:      strPtr("13800138000"),
        Email:      strPtr("zhangsan@example.com"),
        Position:   strPtr("工程师"),
        Role:       strPtr("STAFF"),
        Status:     strPtr("ACTIVE"),
    })
    if err != nil {
        logger.Error("Failed to create employee", "error", err)
        return
    }
    logger.Info("Employee created", "id", employeeID)
    
    // 示例2: 查询员工列表
    employees, err := rosteringClient.ListEmployees(ctx, model.EmployeeListRequest{
        OrgID:    "org-001",
        Page:     1,
        PageSize: 20,
        Keyword:  "张",
    })
    if err != nil {
        logger.Error("Failed to list employees", "error", err)
        return
    }
    logger.Info("Employees retrieved", "count", employees.Total)
    
    // 示例3: 批量排班
    err = rosteringClient.BatchAssignSchedule(ctx, model.BatchAssignRequest{
        OrgID: "org-001",
        Assignments: []*model.ScheduleAssignment{
            {
                Date:       "2025-11-15",
                EmployeeID: "E001",
                ShiftID:    "shift-morning",
                Notes:      "正常排班",
            },
            {
                Date:       "2025-11-15",
                EmployeeID: "E002",
                ShiftID:    "shift-evening",
                Notes:      "正常排班",
            },
        },
    })
    if err != nil {
        logger.Error("Failed to batch assign schedule", "error", err)
        return
    }
    logger.Info("Schedule assigned successfully")
}

func strPtr(s string) *string {
    return &s
}
```

### 高级用法示例

#### 批量创建排班

```go
batchReq := model.ScheduleBatchRequest{
    Schedules: []model.ScheduleUpsertRequest{
        {
            UserID:    "user123",
            WorkDate:  "2025-10-29",
            ShiftCode: "MORNING",
        },
        {
            UserID:    "user456",
            WorkDate:  "2025-10-29",
            ShiftCode: "AFTERNOON",
        },
    },
}

response, err := dataClient.BatchUpsertSchedules(ctx, batchReq)
```

#### 团队成员管理

```go
// 创建团队
team, err := dataClient.CreateTeam(ctx, model.CreateTeamRequest{
    Name:        "开发一组",
    Description: "研发中心开发一组",
    OrgID:       "org001",
})

// 分配团队成员
_, err = dataClient.AssignTeamMembers(ctx, model.TeamMemberAssignRequest{
    TeamID:    team.ID,
    StaffIDs:  []string{"staff001", "staff002", "staff003"},
})
```

## 架构设计

### 分层架构

1. **Domain Layer (领域层)**: 定义业务接口和领域概念
2. **Client Layer (客户端层)**: 实现具体的业务逻辑和 MCP 通信
3. **Model Layer (模型层)**: 定义数据传输对象 (DTO)
4. **Tool Layer (工具层)**: MCP 工具定义和常量
5. **Utils Layer (工具层)**: 通用工具和 MCP 工具总线

### Anti-Corruption Layer (防腐层)

`IToolBus` 接口作为防腐层，隔离了 MCP 协议的具体实现，使得业务逻辑不直接依赖 MCP 底层细节。

### 依赖注入

通过工厂方法和接口注入，保持代码的可测试性和可维护性。

## 配置说明

工具总线支持以下配置项：

- `DefaultTimeout`: 默认超时时间（默认: 30s）
- `RetryCount`: 重试次数（默认: 3）
- `RetryDelay`: 重试延迟（默认: 1s）
- `EnableHealthCheck`: 启用健康检查（默认: true）

## 开发指南

### 添加新功能

1. 在 `domain/` 中定义接口
2. 在 `model/` 中定义数据模型
3. 在 `tool/` 中定义工具常量
4. 在 `client/` 中实现业务逻辑

### 代码规范

- 遵循 Go 官方代码规范
- 使用结构化日志 (`slog`)
- 所有公共方法必须有注释
- 使用上下文 (`context.Context`) 传递请求范围的值

## 故障排查

### 常见问题

1. **连接 MCP 服务失败**
   - 检查网络连通性
   - 验证 MCP 服务器配置
   - 查看工具总线健康检查日志

2. **工具执行超时**
   - 调整 `DefaultTimeout` 配置
   - 检查后端服务性能
   - 优化查询参数

3. **数据序列化错误**
   - 验证请求/响应模型定义
   - 检查 JSON 标签是否正确
   - 查看详细错误日志

## 版本历史

### v2.0.0 (2025-11-10) - 重大架构升级 ⚠️ 破坏性变更

**重大变更:**
- ❌ 移除 `IDataServerClient`, `IContextServerClient`, `IRelationalGraphClient` 接口
- ❌ 移除 `CreateDataServerClient()`, `CreateContextServerClient()`, `CreateRelationalGraphClient()` 工厂方法
- ❌ 移除旧的实体模型: `Staff`, `Team`, `Schedule`, 以及部分 `Shift`, `Rule`, `Leave` 旧字段

**新特性:**
- ✨ 全新的 `IRosteringClient` 统一接口
- 🔄 从旧的 data-server/context-server/relational-graph-server 迁移到 rostering MCP server
- 📦 完整的实体模型同步（Employee, Department, Group, Shift, Rule, Leave, ScheduleAssignment）
- 🎯 新增完整的 CRUD 操作支持
- 🏗️ 简化的架构设计，一个客户端解决所有排班管理需求

**模型字段变更:**
- `Staff` → `Employee`: 新增 `UserID`, `Phone`, `Email`, `Position`, `Role`, `Status`, `HireDate` 等字段
- `Team` → `Group`: 新增 `Type`, `Status`, `LeaderID`, `Attributes` 等字段
- `Schedule` → `ScheduleAssignment`: 字段结构重新设计
- `Shift`: 新增 `Code`, `Type`, `Duration`, `IsOvernight`, `Priority`, `IsActive` 等字段
- `Rule`: 重构为 `ApplyScope`, `TimeScope`, `RuleData` 结构
- `Leave`: 新增 `Type`, `Days` 字段，调整时间字段语义

### v0.0.1-dev.20251112  - 初始开发版本
- 实现核心排班管理功能
- 支持人员、团队、班次管理
- 集成 MCP 协议通信

## 迁移指南

### 从旧版本迁移到 v2.0.0

#### 1. 更新客户端创建方式

**旧版本:**
```go
dataClient := client.CreateDataServerClient(toolBus, logger)
contextClient := client.CreateContextServerClient(toolBus, logger)
graphClient := client.CreateRelationalGraphClient(toolBus, logger)
```

**新版本:**
```go
// 统一使用 RosteringClient
rosteringClient := client.CreateRosteringClient(toolBus, logger)
```

#### 2. 更新方法调用

**旧版本:**
```go
// 旧的Staff模型和方法
staffList, err := dataClient.ListStaff(ctx, orgID, nil, "", 1, 20)
```

**新版本:**
```go
// 新的Employee模型和方法
employees, err := rosteringClient.ListEmployees(ctx, model.EmployeeListRequest{
    OrgID:    orgID,
    Page:     1,
    PageSize: 20,
})
```

#### 3. 模型字段更新

**Employee (原 Staff)**
- `Department` → `DepartmentID` (类型改为 `*int64`)
- 新增: `UserID`, `Phone`, `Email`, `Position`, `Role`, `Status`, `HireDate`, `Groups`

**Shift**
- 新增: `Code`, `Type`, `Duration`, `IsOvernight`, `Priority`, `IsActive`
- 移除: `ParentID`, `Level` (不再支持班次层级)

**Rule**
- 完全重构: 使用 `ApplyScope`, `TimeScope`, `RuleData` 结构化配置
- 新增: `Associations` 关联管理

#### 4. 完整迁移示例

```go
// 旧版本: 创建员工
staffID, err := dataClient.CreateStaff(ctx, model.StaffCreateRequest{
    OrgID:    "org-001",
    UserID:   "user001",
    Name:     "张三",
    Role:     "STAFF",
    TeamID:   &teamID,
    Position: "工程师",
})

// 新版本: 创建员工
employeeID, err := rosteringClient.CreateEmployee(ctx, model.EmployeeCreateRequest{
    OrgID:        "org-001",
    EmployeeID:   "E001",
    Name:         "张三",
    UserID:       strPtr("user001"),
    DepartmentID: int64Ptr(1),
    Position:     strPtr("工程师"),
    Role:         strPtr("STAFF"),
    Status:       strPtr("ACTIVE"),
})
```

## 贡献指南

欢迎提交 Issue 和 Pull Request。

## 许可证

内部项目，版权归属于巨杉科技。

## 联系方式

- 项目维护: 研发中心
- Git 仓库: http://192.168.20.3/dsd-department/pds/solutions-project/js-agent/sdk/rostering-go
