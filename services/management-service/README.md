# Management Service (管理服务)

基于DDD(领域驱动设计)的智能排班管理服务

## 项目结构

```
management-service/
├── domain/                  # 领域层(核心业务逻辑)
│   ├── model/              # 领域实体
│   │   ├── employee.go     # 员工实体
│   │   ├── group.go        # 分组实体
│   │   ├── shift.go        # 班次实体
│   │   └── leave.go        # 假期实体
│   ├── repository/         # 仓储接口(定义数据访问契约)
│   │   ├── employee_repository.go
│   │   ├── group_repository.go
│   │   ├── shift_repository.go
│   │   └── leave_repository.go
│   └── service/            # 领域服务接口(定义业务逻辑契约)
│       ├── employee.go
│       ├── group.go
│       ├── shift.go
│       └── leave.go
├── application/            # 应用层(业务逻辑实现)
│   └── service/           # 领域服务实现
│       └── employee_service.go
├── internal/              # 基础设施层
│   ├── repository/       # 仓储实现(GORM数据访问)
│   │   └── employee_repository.go
│   ├── wiring/           # 依赖注入容器
│   │   └── container.go
│   └── port/             # 端口适配器
│       └── http/         # HTTP API适配器
│           └── handler.go
├── config/               # 配置管理
│   ├── config.go        # 配置定义和加载
│   ├── common.yml       # 通用配置(从../../config复制)
│   └── management-service.yml # 服务特定配置
├── setup.go             # 服务组装和启动
├── go.mod               # Go模块定义
└── README.md            # 本文档
```

## DDD架构说明

### 依赖方向
```
HTTP Handler (internal/port/http)
    ↓
Application Service (application/service)
    ↓
Domain Service Interface (domain/service) ← Repository Interface (domain/repository)
    ↑                                            ↑
    └────────────────────────────────────────────┘
                        ↑
                Repository Impl (internal/repository)
```

### 关键原则

1. **领域层纯粹性**: `domain/`目录只包含接口和实体,不包含实现
2. **依赖倒置**: 应用层和基础设施层都依赖于领域层的接口
3. **仓储抽象**: Service通过Repository接口访问数据,不直接使用GORM
4. **清晰分层**: 每一层都有明确的职责和边界

## 编译

```powershell
# 编译服务
cd services/management-service
go build -o bin/management-service.exe ./cmd

# 或者从项目根目录编译
cd /path/to/project
go build -o services/management-service/bin/management-service.exe ./cmd/services/management-service
```

## 配置

### 配置文件

服务需要两个配置文件:

1. `config/common.yml` - 通用配置(端口、日志、超时等)
2. `config/management-service.yml` - 服务特定配置(数据库连接等)

### 示例配置 (management-service.yml)

```yaml
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  database: "management_db"
  username: "root"
  password: "password"
  max_idle: 10
  max_open: 100
```

## 运行

```powershell
# 确保配置文件存在
# 从服务目录运行
cd services/management-service
./bin/management-service.exe

# 服务默认监听在 :8080
```

## API端点

### 健康检查
- `GET /health` - 服务健康状态检查

### 员工管理 (Employees)
- `GET /api/v1/employees` - 查询员工列表
- `POST /api/v1/employees` - 创建员工
- `GET /api/v1/employees/{id}` - 获取员工详情
- `PUT /api/v1/employees/{id}` - 更新员工信息
- `DELETE /api/v1/employees/{id}` - 删除员工
- `PATCH /api/v1/employees/{id}/status` - 更新员工状态

### 分组管理 (Groups)
- `GET /api/v1/groups` - 查询分组列表
- `POST /api/v1/groups` - 创建分组
- `GET /api/v1/groups/{id}` - 获取分组详情
- `PUT /api/v1/groups/{id}` - 更新分组
- `DELETE /api/v1/groups/{id}` - 删除分组
- `GET /api/v1/groups/{id}/members` - 获取分组成员
- `POST /api/v1/groups/{id}/members` - 添加分组成员
- `DELETE /api/v1/groups/{id}/members/{employeeId}` - 移除分组成员

### 班次管理 (Shifts)
- `GET /api/v1/shifts` - 查询班次列表
- `POST /api/v1/shifts` - 创建班次
- `GET /api/v1/shifts/{id}` - 获取班次详情
- `PUT /api/v1/shifts/{id}` - 更新班次
- `DELETE /api/v1/shifts/{id}` - 删除班次

### 请假管理 (Leaves)
- `GET /api/v1/leaves` - 查询假期记录列表
- `POST /api/v1/leaves` - 创建假期申请
- `GET /api/v1/leaves/{id}` - 获取假期记录详情
- `PUT /api/v1/leaves/{id}` - 更新假期记录
- `DELETE /api/v1/leaves/{id}` - 删除假期记录
- `POST /api/v1/leaves/{id}/approve` - 批准假期
- `POST /api/v1/leaves/{id}/reject` - 拒绝假期
- `POST /api/v1/leaves/{id}/cancel` - 取消假期

## 开发

### 添加新功能

1. **定义领域实体**: 在`domain/model/`中创建新实体
2. **定义仓储接口**: 在`domain/repository/`中定义数据访问接口
3. **定义服务接口**: 在`domain/service/`中定义业务逻辑接口
4. **实现应用服务**: 在`application/service/`中实现业务逻辑
5. **实现仓储**: 在`internal/repository/`中实现数据访问
6. **添加HTTP端点**: 在`internal/port/http/handler.go`中添加API端点
7. **注册依赖**: 在`internal/wiring/container.go`中注册新的依赖

### 运行测试

```powershell
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./domain/model
go test ./application/service
```

## 数据库

### 表结构

服务使用GORM自动创建以下数据库表:

- `employees` - 员工信息
- `groups` - 分组信息
- `shifts` - 班次定义
- `shift_assignments` - 班次分配
- `leave_records` - 假期记录
- `leave_balances` - 假期余额

### 初始化数据库

```sql
CREATE DATABASE management_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## 技术栈

- **Go**: 1.23.2
- **Web框架**: Gorilla Mux
- **ORM**: GORM
- **数据库**: MySQL
- **服务发现**: Nacos (可选)
- **日志**: slog (标准库)

## 参考

- 项目参考: `agents/rostering/` (DDD架构参考)
- 数据模型参考: `mcp-servers/context-server/` (领域模型参考)
- 模块名称: `jusha/gantt/service/management`



## Getting started

To make it easy for you to get started with GitLab, here's a list of recommended next steps.

Already a pro? Just edit this README.md and make it your own. Want to make it easy? [Use the template at the bottom](#editing-this-readme)!

## Add your files

- [ ] [Create](https://docs.gitlab.com/ee/user/project/repository/web_editor.html#create-a-file) or [upload](https://docs.gitlab.com/ee/user/project/repository/web_editor.html#upload-a-file) files
- [ ] [Add files using the command line](https://docs.gitlab.com/ee/gitlab-basics/add-file.html#add-a-file-using-the-command-line) or push an existing Git repository with the following command:

```
cd existing_repo
git remote add origin http://192.168.20.3/inno-tech-build/plato/ai-scheduling/services/management-service.git
git branch -M main
git push -uf origin main
```

## Integrate with your tools

- [ ] [Set up project integrations](http://192.168.20.3/inno-tech-build/plato/ai-scheduling/services/management-service/-/settings/integrations)

## Collaborate with your team

- [ ] [Invite team members and collaborators](https://docs.gitlab.com/ee/user/project/members/)
- [ ] [Create a new merge request](https://docs.gitlab.com/ee/user/project/merge_requests/creating_merge_requests.html)
- [ ] [Automatically close issues from merge requests](https://docs.gitlab.com/ee/user/project/issues/managing_issues.html#closing-issues-automatically)
- [ ] [Enable merge request approvals](https://docs.gitlab.com/ee/user/project/merge_requests/approvals/)
- [ ] [Set auto-merge](https://docs.gitlab.com/ee/user/project/merge_requests/merge_when_pipeline_succeeds.html)

## Test and Deploy

Use the built-in continuous integration in GitLab.

- [ ] [Get started with GitLab CI/CD](https://docs.gitlab.com/ee/ci/quick_start/index.html)
- [ ] [Analyze your code for known vulnerabilities with Static Application Security Testing (SAST)](https://docs.gitlab.com/ee/user/application_security/sast/)
- [ ] [Deploy to Kubernetes, Amazon EC2, or Amazon ECS using Auto Deploy](https://docs.gitlab.com/ee/topics/autodevops/requirements.html)
- [ ] [Use pull-based deployments for improved Kubernetes management](https://docs.gitlab.com/ee/user/clusters/agent/)
- [ ] [Set up protected environments](https://docs.gitlab.com/ee/ci/environments/protected_environments.html)

***

# Editing this README

When you're ready to make this README your own, just edit this file and use the handy template below (or feel free to structure it however you want - this is just a starting point!). Thanks to [makeareadme.com](https://www.makeareadme.com/) for this template.

## Suggestions for a good README

Every project is different, so consider which of these sections apply to yours. The sections used in the template are suggestions for most open source projects. Also keep in mind that while a README can be too long and detailed, too long is better than too short. If you think your README is too long, consider utilizing another form of documentation rather than cutting out information.

## Name
Choose a self-explaining name for your project.

## Description
Let people know what your project can do specifically. Provide context and add a link to any reference visitors might be unfamiliar with. A list of Features or a Background subsection can also be added here. If there are alternatives to your project, this is a good place to list differentiating factors.

## Badges
On some READMEs, you may see small images that convey metadata, such as whether or not all the tests are passing for the project. You can use Shields to add some to your README. Many services also have instructions for adding a badge.

## Visuals
Depending on what you are making, it can be a good idea to include screenshots or even a video (you'll frequently see GIFs rather than actual videos). Tools like ttygif can help, but check out Asciinema for a more sophisticated method.

## Installation
Within a particular ecosystem, there may be a common way of installing things, such as using Yarn, NuGet, or Homebrew. However, consider the possibility that whoever is reading your README is a novice and would like more guidance. Listing specific steps helps remove ambiguity and gets people to using your project as quickly as possible. If it only runs in a specific context like a particular programming language version or operating system or has dependencies that have to be installed manually, also add a Requirements subsection.

## Usage
Use examples liberally, and show the expected output if you can. It's helpful to have inline the smallest example of usage that you can demonstrate, while providing links to more sophisticated examples if they are too long to reasonably include in the README.

## Support
Tell people where they can go to for help. It can be any combination of an issue tracker, a chat room, an email address, etc.

## Roadmap
If you have ideas for releases in the future, it is a good idea to list them in the README.

## Contributing
State if you are open to contributions and what your requirements are for accepting them.

For people who want to make changes to your project, it's helpful to have some documentation on how to get started. Perhaps there is a script that they should run or some environment variables that they need to set. Make these steps explicit. These instructions could also be useful to your future self.

You can also document commands to lint the code or run tests. These steps help to ensure high code quality and reduce the likelihood that the changes inadvertently break something. Having instructions for running tests is especially helpful if it requires external setup, such as starting a Selenium server for testing in a browser.

## Authors and acknowledgment
Show your appreciation to those who have contributed to the project.

## License
For open source projects, say how it is licensed.

## Project status
If you have run out of energy or time for your project, put a note at the top of the README saying that development has slowed down or stopped completely. Someone may choose to fork your project or volunteer to step in as a maintainer or owner, allowing your project to keep going. You can also make an explicit request for maintainers.
