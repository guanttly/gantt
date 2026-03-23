# Rostering MCP Server

排班管理 MCP Server，提供排班相关的工具和功能。通过服务提供者模式与 Management Service 交互，实现员工、班次、部门、分组、规则、请假和排班等管理功能。

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    Rostering MCP Server                      │
│  ┌──────────────┐         ┌────────────────────────────┐   │
│  │ MCP Server   │◄────────│   Tool Manager             │   │
│  │              │         │  - Employee Tools          │   │
│  │  HTTP/SSE    │         │  - Department Tools        │   │
│  │  Transport   │         │  - Group Tools             │   │
│  └──────────────┘         │  - Shift Tools             │   │
│                           │  - Rule Tools              │   │
│                           │  - Leave Tools             │   │
│                           │  - Scheduling Tools        │   │
│                           └──────────┬─────────────────┘   │
│                                      │                      │
│                           ┌──────────▼─────────────────┐   │
│                           │ Service Provider Interface │   │
│                           │  - Employee Service        │   │
│                           │  - Department Service      │   │
│                           │  - Group Service           │   │
│                           │  - Shift Service           │   │
│                           │  - Rule Service            │   │
│                           │  - Leave Service           │   │
│                           │  - Schedule Service        │   │
│                           └──────────┬─────────────────┘   │
└───────────────────────────────────────┼─────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
            ┌───────▼────────┐  ┌──────▼─────────┐        │
            │ Nacos (服务发现)│  │ Direct Config  │        │
            └───────┬────────┘  └────────────────┘        │
                    │                                       │
                    └──────────────┬────────────────────────┘
                                   │
                        ┌──────────▼──────────────┐
                        │  Management Service     │
                        │  REST API (Port 9605)   │
                        └─────────────────────────┘
```

## 功能特性

### 1. 员工管理 (Employee Management)
- `rostering.employee.create` - 创建员工
  - 参数：Name, EmployeeID, OrgID, UserID, DepartmentID, Phone, Email, Position, Role, Status, HireDate, Groups
- `rostering.employee.list` - 查询员工列表（支持分页、关键词搜索、部门过滤）
  - 参数：OrgID, Page, PageSize, Keyword, DepartmentID
- `rostering.employee.get` - 获取员工详情
- `rostering.employee.update` - 更新员工信息
  - 参数：ID, OrgID, Name, UserID, DepartmentID, Phone, Email, Position, Role, Status, HireDate, Groups
- `rostering.employee.delete` - 删除员工

### 2. 部门管理 (Department Management)
- `rostering.department.create` - 创建部门
  - 参数：OrgID, Name, Code, ParentID, Level, Path, Description, ManagerID, SortOrder, IsActive
- `rostering.department.list` - 查询部门列表
- `rostering.department.update` - 更新部门信息
  - 参数：ID, OrgID, Name, Code, ParentID, Description, ManagerID, SortOrder, IsActive

### 3. 分组管理 (Group Management)
- `rostering.group.create` - 创建分组
  - 参数：OrgID, Name, Code, Type, Description, ParentID, LeaderID, Attributes, Status
- `rostering.group.list` - 查询分组列表
  - 参数：OrgID, Page, PageSize, Type, Status, Keyword
- `rostering.group.get` - 获取分组详情
- `rostering.group.update` - 更新分组信息
- `rostering.group.delete` - 删除分组
- `rostering.group.get_members` - 获取分组成员
- `rostering.group.add_member` - 添加分组成员
- `rostering.group.remove_member` - 移除分组成员

### 4. 班次管理 (Shift Management)
- `rostering.shift.create` - 创建班次
  - 参数：OrgID, Name, Code, Type, StartTime, EndTime, Description, Duration, IsOvernight, Priority, IsActive
- `rostering.shift.list` - 查询班次列表
- `rostering.shift.update` - 更新班次信息
  - 参数：ID, OrgID, Name, Code, Type, StartTime, EndTime, Description, Duration, IsOvernight, Priority, IsActive

### 5. 规则管理 (Rule Management)
- `rostering.rule.create` - 创建排班规则
  - 参数：OrgID, Name, Type, Description, Status, ApplyScope, TimeScope, RuleData, ValidFrom, ValidTo, Priority, Associations
- `rostering.rule.list` - 查询规则列表
- `rostering.rule.get` - 获取规则详情
- `rostering.rule.update` - 更新规则信息
- `rostering.rule.delete` - 删除规则
- `rostering.rule.add_associations` - 添加规则关联
- `rostering.rule.get_for_employee` - 获取员工适用的规则

### 6. 请假管理 (Leave Management)
- `rostering.leave.create` - 创建请假记录
  - 参数：OrgID, EmployeeID, EmployeeName, Type, Days, StartTime, EndTime, Reason, Status
- `rostering.leave.list` - 查询请假列表
  - 参数：OrgID, Page, PageSize, EmployeeID, Type, Status, StartDate, EndDate
- `rostering.leave.get` - 获取请假详情
- `rostering.leave.update` - 更新请假信息
- `rostering.leave.delete` - 删除请假记录
- `rostering.leave.get_balance` - 获取员工假期余额

### 7. 排班管理 (Scheduling)
- `rostering.scheduling.batch_assign` - 批量排班
  - 参数：OrgID, Assignments (EmployeeID, ShiftID, Date, Notes)

## 配置说明

### 配置文件

配置文件位于 `config/mcp-servers/rostering-server.yml`：

```yaml
# 服务发现配置（使用 Nacos）
discovery:
  nacos:
    groupName: "mcp-server"  # Nacos 服务分组名

# 工具配置
tools:
  # 启用的工具列表（空数组表示启用所有工具）
  enabled_tools: []
  
  # 危险操作配置
  dangerous:
    enabled: false           # 是否启用危险操作（如删除）
    require_passcode: true   # 是否需要密码验证
    passcode: ""             # 危险操作密码

# 服务端口配置
ports:
  http_port: 9613  # MCP Server HTTP 服务端口
```

### 服务发现

服务器通过 Nacos 自动发现 management-service 实例，无需手动配置服务地址。Nacos 配置在 `config/common.yml` 中：

```yaml
discovery:
  nacos:
    server_addr: "localhost:8848"
    namespace: "public"
    group: "DEFAULT_GROUP"
```

## 目录结构

```
mcp-servers/rostering/
├── config/              # 配置相关
│   └── config.go
├── domain/              # 领域层
│   ├── model/           # 数据模型（实体定义）
│   │   ├── employee.go        # 员工模型
│   │   ├── shift.go           # 班次模型
│   │   ├── department.go      # 部门模型
│   │   ├── group.go           # 分组模型
│   │   ├── rule.go            # 规则模型
│   │   ├── leave.go           # 请假模型
│   │   └── schedule_assignment.go # 排班记录模型
│   └── service/         # 服务接口层
│       ├── provider.go        # 服务提供者接口
│       ├── employee.go        # 员工服务接口
│       ├── shift.go           # 班次服务接口
│       ├── department.go      # 部门服务接口
│       ├── group.go           # 分组服务接口
│       ├── rule.go            # 规则服务接口
│       ├── leave.go           # 请假服务接口
│       └── schedule_assignment.go # 排班服务接口
├── tool/                # MCP 工具实现
│   ├── employee/        # 员工管理工具
│   ├── department/      # 部门管理工具
│   ├── group/           # 分组管理工具
│   ├── shift/           # 班次管理工具
│   ├── rule/            # 规则管理工具
│   ├── leave/           # 请假管理工具
│   ├── scheduling/      # 排班管理工具
│   └── manager.go       # 工具管理器
├── setup.go             # 服务启动配置
├── go.mod
└── README.md
```

## 运行方式

### 开发环境

1. 确保 management-service 已启动（端口 9605）
2. 确保 Nacos 已启动（端口 8848）
3. 配置 `config/common.yml` 和 `config/mcp-servers/rostering-server.yml`
4. 运行服务：

```bash
go run cmd/mcp-servers/rostering-server/main.go
```

### 生产环境

1. 配置 Nacos 服务发现（`config/common.yml`）
2. 服务会自动从 Nacos 发现 management-service
3. 编译并运行二进制文件

## API 使用示例

### 创建员工

```json
{
  "method": "tools/call",
  "params": {
    "name": "rostering.employee.create",
    "arguments": {
      "orgId": "org-001",
      "employeeId": "E001",
      "name": "张三",
      "phone": "13800138000",
      "email": "zhangsan@example.com",
      "departmentId": "dept-001",
      "userId": "user-001",
      "position": "工程师",
      "role": "STAFF",
      "status": "ACTIVE",
      "hireDate": "2024-01-01T00:00:00Z"
    }
  }
}
```

### 创建班次

```json
{
  "method": "tools/call",
  "params": {
    "name": "rostering.shift.create",
    "arguments": {
      "orgId": "org-001",
      "name": "早班",
      "code": "MORNING",
      "type": "REGULAR",
      "startTime": "08:00",
      "endTime": "16:00",
      "description": "早班时间",
      "duration": 8.0,
      "isOvernight": false,
      "priority": 1,
      "isActive": true
    }
  }
}
```

### 创建排班规则

```json
{
  "method": "tools/call",
  "params": {
    "name": "rostering.rule.create",
    "arguments": {
      "orgId": "org-001",
      "name": "连续工作不超过7天",
      "type": "CONSECUTIVE_WORK_DAYS",
      "description": "员工连续工作不得超过7天",
      "status": "ACTIVE",
      "applyScope": {
        "departments": ["dept-001"],
        "groups": ["group-001"],
        "employees": []
      },
      "timeScope": {
        "validFrom": "2024-01-01",
        "validTo": "2024-12-31",
        "weekdays": [1, 2, 3, 4, 5, 6, 7]
      },
      "ruleData": {
        "maxValue": 7,
        "minValue": 1
      },
      "priority": 100,
      "validFrom": "2024-01-01T00:00:00Z",
      "validTo": "2024-12-31T23:59:59Z"
    }
  }
}
```

### 批量排班

```json
{
  "method": "tools/call",
  "params": {
    "name": "rostering.scheduling.batch_assign",
    "arguments": {
      "orgId": "org-001",
      "assignments": [
        {
          "date": "2025-11-15",
          "employeeId": "E001",
          "shiftId": "shift-morning",
          "notes": "正常排班"
        },
        {
          "date": "2025-11-15",
          "employeeId": "E002",
          "shiftId": "shift-evening",
          "notes": "正常排班"
        }
      ]
    }
  }
}
```

## 依赖服务

- **management-service**: 提供排班管理的核心业务逻辑和数据存储（端口 9605）
- **Nacos**: 服务发现和配置管理（端口 8848，用于生产环境）

## 技术栈

- Go 1.23+
- MCP (Model Context Protocol)
- 服务提供者模式（Service Provider Pattern）
- HTTP/REST API 通信
- Nacos 服务发现
- JSON Schema 参数验证

## 核心特性

### 服务架构
- **服务接口层** (`domain/service`): 定义业务服务接口，解耦工具层与实现
- **数据模型层** (`domain/model`): 统一的实体定义，与 management-service 保持一致
- **工具层** (`tool`): MCP 工具实现，通过服务提供者访问业务逻辑

### 字段映射
所有实体字段已与 management-service 完全同步：
- Employee: 支持 UserID, Phone, Email, Position, Role, Status, HireDate 等完整信息
- Shift: 支持 Code, Type, Duration, IsOvernight, Priority, IsActive 等扩展字段
- Department: 支持层级结构 (Code, Level, Path, ParentID, ManagerID)
- Group: 支持分组层级和属性扩展
- Rule: 完整的规则引擎支持 (ApplyScope, TimeScope, RuleData, Associations)
- Leave: 完整的请假管理字段
- ScheduleAssignment: 支持排班备注和完整时间戳

### 时间处理
- 使用 RFC3339 格式 (ISO 8601) 处理所有日期时间字段
- 支持时区信息和跨天班次
- 统一的时间解析和验证

## 开发指南

### 添加新工具
1. 在 `tool/<category>/` 创建新工具文件
2. 实现 `mcp.Tool` 接口
3. 在 `tool/manager.go` 的 `RegisterTools()` 中注册
4. 使用 `t.provider.<Service>().<Method>()` 访问服务

### 更新实体字段
1. 修改 `domain/model/<entity>.go` 模型定义
2. 同步更新相关工具的 InputSchema
3. 更新工具的参数解析和请求构造
4. 更新 README.md 中的示例

## 后续计划

- [x] 完成员工、部门、分组、班次基础管理
- [x] 实现规则引擎和请假管理
- [x] 完成服务架构迁移（Client → Service Provider）
- [x] 字段映射完全同步
- [ ] 添加批量导入导出功能
- [ ] 实现智能排班算法
- [ ] 添加数据缓存机制
- [ ] 增强错误处理和重试机制