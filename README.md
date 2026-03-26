# AI 智能排班系统 (Gantt Scheduling System)

<div align="center">

**基于 AI 驱动的微服务架构智能排班系统**

[![Go Version](https://img.shields.io/badge/Go-1.23.2-blue.svg)](https://golang.org)
[![Vue Version](https://img.shields.io/badge/Vue-3.5.13-green.svg)](https://vuejs.org)
[![MCP Protocol](https://img.shields.io/badge/Protocol-MCP-orange.svg)]()
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)]()

</div>

## 📋 目录

- [项目概述](#项目概述)
- [核心特性](#核心特性)
- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [模块说明](#模块说明)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [开发指南](#开发指南)
- [部署说明](#部署说明)
- [API 文档](#api-文档)

## 🎯 项目概述

AI 智能排班系统是一个面向医疗放射科等复杂排班场景的企业级解决方案，采用微服务架构，整合了 AI 大模型、知识图谱、向量数据库等先进技术，提供智能化、自动化的排班管理能力。

### 适用场景

- 🏥 **医疗机构**：放射科、急诊科等科室的医护人员排班
- 🏭 **制造企业**：多班次、多岗位的生产人员排班
- 🏪 **零售服务**：连锁门店的员工排班管理
- 🚀 **其他场景**：任何需要复杂规则约束的排班场景

### 核心价值

- ✅ **智能化**：基于 AI 的自然语言交互，降低使用门槛
- ✅ **规则化**：支持复杂的排班规则、冲突检测和智能推荐
- ✅ **可扩展**：微服务架构，支持横向扩展和功能定制
- ✅ **高性能**：向量检索、图数据库、缓存等技术保障高效响应

## 🚀 核心特性

### 1. AI 驱动的自然语言交互

- 🤖 **意图识别**：支持多层级意图识别（一级意图 → 二级意图）
- 💬 **对话式交互**：通过自然语言描述即可创建、修改、查询排班
- 🧠 **上下文感知**：自动收集历史排班、人员信息等上下文数据
- 🔄 **工作流引擎**：基于 FSM 的多意图顺序执行与状态追踪

### 2. 知识图谱与规则引擎

- 📊 **关系建模**：基于 Neo4j 构建人员、团队、技能、角色等实体关系
- 🔍 **冲突检测**：自动识别排班冲突和规则违反
- 🎯 **兼容性分析**：计算人员间的工作兼容性和团队协同效果
- 📝 **规则解析**：支持中文自然语言规则输入并自动解析

### 3. 智能排班算法

- 🎲 **智能推荐**：基于规则、技能、历史数据的智能人员筛选
- ⚖️ **负载均衡**：确保人员工作量的公平分配
- 📈 **优化求解**：支持多目标优化的排班方案生成
- 🔄 **动态调整**：实时调整排班以应对突发情况

### 4. 上下文与记忆管理

- 💾 **持久化记忆**：对话历史和业务数据的持久化存储
- 🔎 **语义检索**：基于向量数据库（Milvus）的语义相似度搜索
- 🏷️ **标签系统**：支持多标签分类和按标签过滤检索
- 🤖 **自动向量化**：支持单条和批量的 Embedding 生成

### 5. 数据管理与查询

- 📅 **排班管理**：完整的排班 CRUD 操作，支持批量操作
- 👥 **人员管理**：人员档案、技能、角色、团队等信息管理
- 🏢 **团队管理**：团队创建、成员分配、班次配置
- 📊 **请假管理**：请假记录创建、查询和人员可用性检查

### 6. 实时通信与监控

- 🌐 **WebSocket 支持**：实时消息推送和双向通信
- 📊 **健康检查**：完善的服务健康检查和监控
- 🔄 **服务发现**：基于 Nacos 的服务注册与发现
- 📝 **结构化日志**：统一的日志规范和链路追踪

## 🏗️ 系统架构

### 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端层 (Frontend)                        │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Vue 3 + Element Plus + TypeScript                       │  │
│  │  - 排班管理界面                                          │  │
│  │  - 可视化排班表 (Gantt Chart)                           │  │
│  │  - AI 对话交互界面                                       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTP/WS
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      业务服务层 (Services)                       │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Rostering Agent (排班智能体)                          │    │
│  │  - AI 意图识别                                         │    │
│  │  - V4 排班工作流                                       │    │
│  │  - 上下文聚合                                          │    │
│  │  - 业务编排                                            │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Management Service (管理服务)                         │    │
│  │  - 组织/人员/班次/规则等管理接口                       │    │
│  └────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                        │ MCP Protocol
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                   MCP 服务层 (MCP Servers)                       │
│                                                                  │
│  ┌──────────────┐  ┌──────────────────────────────────────┐  │
│  │ Context      │  │ Rostering Server                     │  │
│  │ Server       │  │                                      │  │
│  │              │  │ - 排班领域 MCP 工具                 │  │
│  │ - 记忆管理   │  │ - 规则/班次/人员协同编排           │  │
│  │ - 对话管理   │  │ - 面向排班工作流的工具暴露         │  │
│  │ - 语义检索   │  │                                      │  │
│  │ - 向量化     │  │                                      │  │
│  └──────────────┘  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                     基础设施层 (Infrastructure)                  │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │  MySQL   │ │  Redis   │ │  Milvus  │ │  Neo4j   │          │
│  │  关系型  │ │  缓存    │ │  向量库  │ │  图数据库│          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │  MinIO   │ │  Nacos   │ │ AI Model │                        │
│  │  对象存储│ │ 服务发现 │ │ (多模型) │                        │
│  └──────────┘ └──────────┘ └──────────┘                        │
└─────────────────────────────────────────────────────────────────┘
```

### MCP 协议架构

本系统采用 Model Context Protocol (MCP) 作为服务间通信协议，提供标准化的工具调用能力：

```
Rostering Agent (编排层)
    │
    │ MCP Client
    │
    ├──────────┬──────────────┐
    ▼          ▼              ▼
  Context   Rostering      AI Models
  Server    Server         (Provider)
    │          │
    │ MCP Tools│
    │          │
  [记忆/对话] [排班领域工具]
```

## 🛠️ 技术栈

### 后端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.23.2 | 主要开发语言 |
| Go-Kit | - | 微服务框架 |
| GORM | - | ORM 框架 |
| Gin | - | HTTP 框架 |
| Gorilla WebSocket | - | WebSocket 支持 |

### 前端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue | 3.5.13 | 前端框架 |
| TypeScript | 5.9.2 | 类型系统 |
| Element Plus | 2.8.3 | UI 组件库 |
| Vue Router | 4.5.1 | 路由管理 |
| Pinia | 3.0.3 | 状态管理 |
| ECharts | 6.0.0 | 数据可视化 |
| Vis-Timeline | 8.3.1 | 甘特图组件 |
| Vite | 5.4.14 | 构建工具 |

### 数据存储

| 技术 | 用途 |
|------|------|
| MySQL | 关系型数据（排班、人员、团队等） |
| Redis | 缓存、会话管理 |
| Milvus | 向量数据库（语义检索） |
| Neo4j | 图数据库（知识图谱） |
| MinIO | 对象存储（文件管理） |

### AI 与算法

| 技术 | 用途 |
|------|------|
| OpenAI API | 大语言模型（GPT-3.5/4） |
| Ollama | 本地模型部署 |
| 阿里百炼 | 企业级 AI 服务 |
| Embedding Models | 文本向量化（维度 1024） |

### 基础设施

| 技术 | 用途 |
|------|------|
| Nacos | 服务注册与发现、配置管理 |
| Docker | 容器化部署 |
| Git | 版本控制 |

## 📦 模块说明

### 核心服务模块

#### 1. Rostering Agent (排班智能体)

**路径**：`agents/rostering/` 与 `cmd/agents/rostering/`  
**端口**：从配置读取

当前核心业务编排服务，负责 V4 排班流程的调度和 AI 意图识别。

**主要功能**：
- AI 驱动的意图识别（一级/二级意图）
- 工作流引擎（V4 排班工作流）
- 上下文聚合与管理
- MCP 客户端集成
- WebSocket 实时通信
- 排班算法编排

**领域模型**：
- Schedule（排班实体）
- Staff（人员实体）
- Session（会话模型）
- Intent（意图模型）
- Workflow（工作流）

#### 2. Context Server (上下文服务)

**路径**：`mcp-servers/context-server/`  
**端口**：9613 (HTTP)

面向多智能体的上下文与记忆管理微服务。

**主要功能**：
- 记忆管理（CRUD + 标签系统）
- 对话管理（会话创建、消息追加、历史查询）
- 语义检索（基于 Milvus 的向量检索）
- 自动向量化（单条/批量 Embedding）
- 上下文构建（融合历史与语义）

**MCP 工具**：
```
memory.create              # 创建记忆
memory.read                # 读取记忆
memory.update              # 更新记忆
memory.delete              # 删除记忆
memory.search              # 标签搜索
memory.semantic_search     # 语义搜索
memory.embedding_upsert    # 单条向量化
memory.bulk_embedding_upsert # 批量向量化
conversation.new           # 创建对话
conversation.append        # 追加消息
conversation.history       # 查询历史
memory.build_context       # 构建上下文
```

#### 3. Rostering Server (排班 MCP 服务)

**路径**：`mcp-servers/rostering/` 与 `cmd/mcp-servers/rostering-server/`  
**端口**：从配置读取

当前排班 MCP 服务，向智能体暴露排班领域工具，承接排班相关的工具调用与领域操作。

#### 4. Management Service (管理服务)

**路径**：`services/management-service/` 与 `cmd/services/management-service/`

当前管理服务提供组织、人员、班次、规则等管理接口，已不是“预留模块”。

### 前端模块

#### Web 前端

**路径**：`frontend/web/`  
**技术栈**：Vue 3 + TypeScript + Element Plus

**主要功能**：
- 排班管理界面（创建、编辑、查看）
- 可视化排班表（基于 Vis-Timeline 的甘特图）
- AI 对话交互界面
- 人员管理界面
- 团队配置界面
- 统计报表与数据可视化

### 公共包模块

#### agent PKG

**路径**：`pkg/`  
**模块**：`jusha/mcp/pkg`

为所有服务提供的通用组件包。

**主要模块**：

```
pkg/
├── ai/                 # AI 模型集成
│   ├── factory.go     # AI 提供商工厂
│   ├── openai.go      # OpenAI 集成
│   ├── ollama.go      # Ollama 集成
│   └── bailian.go     # 阿里百炼集成
├── mcp/                # MCP 协议实现
│   ├── server.go      # MCP 服务器
│   ├── client.go      # MCP 客户端
│   ├── registry.go    # 工具注册表
│   └── transport/     # 传输层（HTTP/SSE/Stdio）
├── config/             # 配置管理
│   ├── loader.go      # 配置加载器
│   └── manager.go     # 配置管理器
├── database/           # 数据库封装
│   └── mysql.go       # MySQL 连接池
├── discovery/          # 服务发现
│   ├── nacos_client.go    # Nacos 客户端
│   └── nacos_registrar.go # 服务注册
├── logging/            # 日志系统
├── client/             # 服务客户端
├── middleware/         # 中间件
├── utils/              # 工具函数
└── version/            # 版本管理
```

## 🚀 快速开始

### 环境要求

#### 必需组件

| 组件 | 版本要求 | 说明 |
|------|---------|------|
| Go | 1.23.2+ | 后端开发语言 |
| Node.js | 18.0+ | 前端构建工具 |
| pnpm | 10.14.0+ | 前端包管理器 |
| MySQL | 8.0+ | 关系型数据库 |
| Redis | 6.0+ | 缓存数据库 |
| Milvus | 2.3+ | 向量数据库 |
| Neo4j | 5.0+ | 图数据库 |
| Nacos | 2.0+ | 服务发现与配置中心 |

#### 可选组件

| 组件 | 用途 |
|------|------|
| MinIO | 对象存储（文件管理） |
| Docker | 容器化部署 |

### 安装步骤

#### 1. 克隆项目

本仓库当前以单仓多模块方式维护，直接克隆仓库即可：

```bash
# 克隆主项目
git clone git@192.168.20.3:inno-tech-build/plato/gantt/app.git
cd app
```

#### 2. 配置环境变量

由于项目依赖私有 GitLab 仓库，需要配置 Go 模块访问：

**Linux/MacOS：**
```bash
export GOPROXY="direct"
export GOPRIVATE="192.168.20.3"
export GONOPROXY="192.168.20.3"
export GONOSUMDB="192.168.20.3"
```

**Windows PowerShell：**
```powershell
$env:GOPROXY = "direct"
$env:GOPRIVATE = "192.168.20.3"
$env:GONOPROXY = "192.168.20.3"
$env:GONOSUMDB = "192.168.20.3"
```

配置 Git URL 重写（必需）：
```bash
git config --global url."git@192.168.20.3:".insteadOf "https:///"
```

#### 3. 安装依赖

**后端依赖：**
```bash
# 主项目依赖
go mod download

# 主要子模块依赖
cd agents/rostering && go mod download && cd ../..
cd pkg && go mod download && cd ..
```

**前端依赖：**
```bash
cd frontend/web
pnpm install
```

#### 4. 配置文件

当前后端只使用一个配置文件：

```bash
cp config/config.example.yml config/config.yml
vi config/config.yml
```

主要配置项：
- 数据库连接（MySQL）
- Redis 连接
- JWT 密钥与过期时间
- AI 模型配置（OpenAI/Ollama/百炼）
- 服务端口与日志级别

#### 5. 初始化数据库

```bash
# 创建数据库
mysql -u root -p
CREATE DATABASE ingestiondb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE contextdb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE scheduling_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# 数据库迁移会在服务启动时自动执行
```

#### 6. 启动服务

**方式一：使用 VS Code 任务（推荐）**

项目当前使用单体入口，可以直接运行：

```bash
go run ./cmd/server
```

**方式二：手动启动**

```bash
# 直接从仓库根目录启动后端
go run ./cmd/server
```

**启动前端：**

```bash
cd frontend/web
pnpm dev
```

#### 7. 验证安装

访问以下端点验证服务是否正常运行：

- **前端界面**：http://localhost:5173
- **后端健康检查**：http://localhost:8080/healthz

## ⚙️ 配置说明

### 配置文件结构

```
config/
├── config.yml                          # 当前运行配置
└── config.example.yml                  # 配置样例
```

### 核心配置项

#### config.yml（当前运行配置）

```yaml
server:
  port: 8080

database:
  host: "localhost"
  port: 3306
  name: "gantt_saas"
  user: "root"
  password: "your_password_here"

redis:
  addr: "localhost:6379"
  password: ""

log:
  level: "info"
  format: "json"
  output: "stdout"

jwt:
  secret: "change-me"

ai:
  enabled: false
  openai:
    api_key: "sk-xxx"
    model: "gpt-4o-mini"
  retry:
    max_shift_retries: 3
    enable_ai_analysis: true

# 服务端口
ports:
  http_port: 9601

# 意图识别配置
intent:
  maxHistory: 8
```

### 环境变量

支持通过环境变量覆盖配置文件中的设置：

```bash
# 数据库
export MYSQL_HOST=192.168.40.238
export MYSQL_PASSWORD=jusha1996
export REDIS_HOST=192.168.40.206

# AI 配置
export OPENAI_API_KEY=sk-xxx
export AI_PROVIDER=openai

# 服务发现
export NACOS_ADDRESSES=192.168.40.238:8848
```

## 💻 开发指南

### 项目结构

每个服务遵循统一的 DDD（领域驱动设计）结构：

```
service-name/
├── cmd/                    # 可执行入口
├── config/                 # 配置管理
├── domain/                 # 领域层
│   ├── model/             # 领域模型
│   ├── repository/        # 仓储接口
│   └── service/           # 领域服务
├── internal/              # 内部实现
│   ├── application/       # 应用层
│   ├── infrastructure/    # 基础设施层
│   └── interfaces/        # 接口层（HTTP/gRPC）
└── test/                  # 测试
```

### 开发规范

#### 代码风格

- 遵循 Go 官方代码规范
- 使用 `golint`、`go vet` 进行代码检查
- 前端遵循 ESLint 配置

#### 命名规范

- **包名**：小写单数形式，如 `model`、`service`
- **接口**：`I` 前缀或 `-er` 后缀，如 `IRepository`、`Handler`
- **结构体**：大驼峰，如 `ScheduleService`
- **函数**：大驼峰（公开）/ 小驼峰（私有）
- **常量**：大写下划线，如 `MAX_RETRY_COUNT`

#### 提交规范

遵循 Conventional Commits：

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型（type）：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具链相关

示例：
```
feat(scheduling): 添加智能人员推荐功能

实现基于技能匹配和历史数据的智能人员推荐算法

Closes #123
```

### 测试指南

#### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/application/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 集成测试

```bash
# 运行集成测试（需要启动依赖服务）
go test -tags=integration ./test/integration/...
```

#### 前端测试

```bash
cd frontend/web
pnpm test           # 运行单元测试
pnpm test:ui        # 运行 UI 测试
```

### 调试指南

#### VS Code 调试配置

项目已配置 `.vscode/launch.json`，可直接在 VS Code 中调试：

1. 设置断点
2. 按 F5 或选择"运行和调试"
3. 选择要调试的服务

#### 日志调试

```go
import "jusha/mcp/pkg/logging"

// 使用结构化日志
logging.Info("处理排班请求", 
    "session_id", sessionID,
    "user_id", userID,
    "action", "create_schedule")

logging.Error("数据库查询失败",
    "error", err,
    "query", sql)
```

### 多模块仓库说明

当前仓库不是旧的 Git Submodule 拆分形态，主要采用单仓多 Go module 的组织方式。

常用模块：

```bash
# 根模块
go test ./...

# pkg 模块
cd pkg && go test ./... && cd ..

# rostering agent 模块
cd agents/rostering && go test ./... && cd ../..
```

### MCP 工具开发

#### 创建新工具

1. **定义工具结构**

```go
type MyTool struct {
    Name        string
    Description string
    InputSchema any
}
```

2. **实现工具逻辑**

```go
func (t *MyTool) Execute(args map[string]any) (any, error) {
    // 参数验证
    param, ok := args["param"].(string)
    if !ok {
        return nil, errors.New("invalid parameter")
    }
    
    // 业务逻辑
    result := doSomething(param)
    
    return result, nil
}
```

3. **注册工具**

```go
registry.RegisterTool("my_tool", &MyTool{
    Name:        "my_tool",
    Description: "工具描述",
    InputSchema: schema,
})
```

## 🚢 部署说明

### Docker 部署（推荐）

#### 1. 构建镜像

```bash
# 构建所有服务镜像
docker-compose build

# 或单独构建
docker build -t context-server:latest -f Dockerfile .
```

#### 2. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 3. 健康检查

```bash
# 检查服务状态
docker-compose ps

# 访问健康检查端点
curl http://localhost:8080/healthz
```

### Kubernetes 部署

#### 1. 准备配置

```bash
# 创建命名空间
kubectl create namespace gantt-scheduling

# 创建 ConfigMap
kubectl create configmap app-config --from-file=config/config.yml -n gantt-scheduling

# 创建 Secret（敏感信息）
kubectl create secret generic db-credentials \
  --from-literal=mysql-password=xxx \
  --from-literal=redis-password=xxx \
  -n gantt-scheduling
```

#### 2. 部署服务

```bash
# 部署所有服务
kubectl apply -f k8s/ -n gantt-scheduling

# 查看部署状态
kubectl get pods -n gantt-scheduling
kubectl get svc -n gantt-scheduling

# 查看日志
kubectl logs -f deployment/management-service -n gantt-scheduling
```

#### 3. 扩容

```bash
# 水平扩容
kubectl scale deployment/management-service --replicas=3 -n gantt-scheduling
```

### 生产环境检查清单

- [ ] 数据库备份策略已配置
- [ ] Redis 持久化已启用
- [ ] 日志收集已配置（ELK/Loki）
- [ ] 监控告警已配置（Prometheus/Grafana）
- [ ] 服务限流已启用
- [ ] HTTPS 证书已配置
- [ ] 防火墙规则已设置
- [ ] 定时任务已配置
- [ ] 数据加密已启用
- [ ] 审计日志已开启

## 📚 API 文档

### RESTful API

#### Scheduling Service API

**基础路径**：`http://localhost:9601/api/v1`

##### 会话管理

```http
POST /sessions
# 创建新会话
Content-Type: application/json

{
  "user_id": "user123",
  "metadata": {}
}

# 响应
{
  "session_id": "sess_xxx",
  "createdAt": "2025-01-01T00:00:00Z"
}
```

```http
POST /sessions/{session_id}/messages
# 发送消息
Content-Type: application/json

{
  "message": "帮我为明天排班，CT组需要3个人",
  "context": {}
}

# 响应
{
  "intent": "schedule_create",
  "response": "好的，我来为您安排明天的CT组排班...",
  "data": {...}
}
```

##### 排班管理

```http
GET /schedules?date=2025-01-01&team_id=1
# 查询排班

POST /schedules
# 创建排班
Content-Type: application/json

{
  "staff_id": 1,
  "shiftId": 1,
  "date": "2025-01-01",
  "team_id": 1
}

PUT /schedules/{id}
# 更新排班

DELETE /schedules/{id}
# 删除排班
```

##### 人员管理

```http
GET /staff
# 列出所有人员

GET /staff/{id}
# 获取人员详情

POST /staff
# 创建人员

GET /staff/search?q=张三
# 搜索人员
```

### WebSocket API

**连接地址**：`ws://localhost:9602/ws`

#### 消息格式

```json
{
  "type": "message",
  "session_id": "sess_xxx",
  "data": {
    "message": "用户输入的消息"
  }
}
```

#### 服务器推送

```json
{
  "type": "intent_detected",
  "data": {
    "intent": "schedule_create",
    "confidence": 0.95
  }
}

{
  "type": "workflow_progress",
  "data": {
    "step": "gathering_context",
    "progress": 50
  }
}

{
  "type": "result",
  "data": {
    "schedules": [...]
  }
}
```

### MCP 工具 API

MCP 工具通过内部 HTTP 调用，不直接对外暴露。工具列表请参考各服务的 README。

## 🤝 贡献指南

欢迎贡献代码、报告问题或提出建议！

### 报告问题

1. 在 Issue Tracker 中搜索是否已存在类似问题
2. 如果没有，创建新 Issue
3. 提供详细的问题描述、复现步骤、环境信息

### 提交代码

1. Fork 项目
2. 创建特性分支：`git checkout -b feature/my-feature`
3. 提交更改：`git commit -m 'feat: add some feature'`
4. 推送到分支：`git push origin feature/my-feature`
5. 提交 Pull Request

### Pull Request 检查清单

- [ ] 代码遵循项目规范
- [ ] 添加了必要的测试
- [ ] 测试全部通过
- [ ] 更新了相关文档
- [ ] 提交信息符合规范

## 📄 许可证

本项目为专有软件，版权所有 © 2025 聚煞科技。未经授权不得使用、复制或分发。

## 📞 联系方式

- **项目维护者**：Gantt 团队
- **技术支持**：support@example.com
- **文档**：https://docs.example.com

## 🙏 致谢

感谢以下开源项目：

- [Go](https://golang.org/)
- [Vue.js](https://vuejs.org/)
- [Element Plus](https://element-plus.org/)
- [Go-Kit](https://gokit.io/)
- [Milvus](https://milvus.io/)
- [Neo4j](https://neo4j.com/)

---

**最后更新**：2025-10-27  
**版本**：v0.0.1-dev
