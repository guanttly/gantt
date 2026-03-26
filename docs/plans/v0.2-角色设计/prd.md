# AI 智能排班平台 — 人员与权限体系重构 PRD

> **文档版本**: 1.0  
> **文档日期**: 2026-03-26  
> **文档状态**: 设计中  
> **关联文档**: v0.2 重构 PRD、重构总体方案

---

## 一、背景与目标

### 1.1 背景

当前系统在 v0.2 重构方案中已定义了平台管理员、机构管理员、科室管理员、排班负责人、普通员工五种角色，并实现了基础的 JWT + RBAC 认证体系。但在实际落地过程中，以下问题需要进一步明确：

1. **平台与机构的职责边界不清晰**：班次管理、规则管理等核心配置究竟应该在平台侧完成还是在排班应用侧完成，缺乏明确划分。
2. **平台管理员与机构员工的关系需要重新定义**：平台管理员应当是独立账号体系，与机构员工解耦。
3. **排班作为"应用"的定位未落实**：排班应收敛为一个功能应用，平台则聚焦于组织管理、人员配置、规则配置等基础能力。

### 1.2 目标

基于用户提出的五条设计原则，重新梳理人员与权限体系，完成以下目标：

1. 平台管理员仅管理平台事项及机构的创建，不参与具体排班操作。
2. 机构创建时附带创建一个机构管理员账号（后台登录用），默认密码 + 强制修改机制。
3. 平台管理员可管理机构内的规则配置、人员配置与分组、以及下一级管理员。
4. 下一级管理员可管理本级组织内的规则配置，支持继承/禁用上级排班规则，并调整人员分组。
5. 平台管理员账号独立于机构员工体系，但预留员工绑定平台账号的扩展点。
6. 班次管理、规则管理等配置逻辑从排班应用上移到平台层，排班应用仅负责排班执行与排班查看。

---

## 二、核心设计原则

1. **平台即管理面，排班即应用面**：平台负责"谁能排班、按什么规则排"，排班应用负责"执行排班、展示排班"。
2. **账号体系双轨制**：平台管理员是独立账号（`platform_users` 表），机构员工是业务账号（`employees` 表），两套体系独立运行，未来通过绑定关系打通。
3. **配置下沉、规则继承**：上级配置自动对下级生效，下级可选择继承、覆盖或禁用。
4. **最小权限原则**：每个角色只能看到和操作其职责范围内的数据。

---

## 三、角色体系重新定义

### 3.1 角色清单

| 角色 | 归属体系 | 职责范围 | 登录入口 |
|------|---------|---------|---------|
| 平台超级管理员 | 平台账号 | 平台全局管理、机构创建 | 平台管理后台 |
| 机构管理员 | 平台账号 | 本机构内全部管理事项 | 平台管理后台（机构视角） |
| 科室/部门管理员 | 平台账号 | 本级组织内管理事项 | 平台管理后台（科室视角） |
| 排班负责人 | 业务账号（员工） | 执行排班、调整排班 | 排班应用 |
| 普通员工 | 业务账号（员工） | 查看个人排班、提交偏好/请假 | 排班应用 |

### 3.2 角色关系图

```
平台超级管理员
  │
  ├── 创建机构 → 同时创建「机构管理员」账号
  │
  └── 机构管理员
        │
        ├── 管理本机构下的规则、班次、人员、分组
        ├── 创建下级组织节点（院区/科室）
        ├── 为下级节点指派「科室管理员」
        │
        └── 科室管理员
              │
              ├── 管理本科室的规则（继承/覆盖/禁用上级规则）
              ├── 调整本科室人员分组
              └── （不可创建更下级组织节点，除非平台放开层级限制）

--- 以下为排班应用内的角色 ---

排班负责人 ← 由管理员在平台侧从员工中指派
  └── 执行排班、审核排班、处理调班

普通员工
  └── 查看排班、提交偏好、申请请假
```

### 3.3 平台管理员与机构员工的关系

**当前实现（V1）**：两套账号体系完全独立。

- 平台管理员账号存储在 `platform_users` 表中，使用独立的用户名/密码体系。
- 机构员工存储在 `employees` 表中，属于业务数据。
- 两者之间没有关联关系。

**未来扩展（V2 预留）**：支持员工绑定平台账号。

- 在 `platform_users` 表中增加 `bound_employee_id` 字段（可空），指向 `employees` 表。
- 绑定后，平台管理员在查看排班数据时可以识别"这个员工就是我自己"。
- 本期不实现，但数据模型和 API 预留扩展点。

---

## 四、账号生命周期

### 4.1 平台超级管理员

系统初始化时通过 Seed 脚本自动创建，用户名和默认密码从配置文件读取。首次登录强制修改密码。平台超级管理员账号不可被删除，只能修改密码。

### 4.2 机构管理员

**创建流程**：

```
平台超级管理员 → 创建机构
  ├── 填写机构基本信息（名称、编码、联系方式等）
  ├── 填写机构管理员信息（用户名、邮箱）
  ├── 系统自动生成默认密码（或使用预设规则如 OrgCode@2026）
  └── 创建完成
        ├── 机构节点写入 org_nodes 表
        ├── 管理员账号写入 platform_users 表（must_reset_pwd = true）
        └── 分配 org_admin 角色，绑定到该机构节点
```

**首次登录**：

```
机构管理员使用默认密码登录
  → 系统检测 must_reset_pwd = true
  → 强制跳转到「修改密码」页面
  → 新密码需满足强度要求（≥8位，含大小写 + 数字）
  → 修改成功后 must_reset_pwd = false
  → 跳转到管理后台首页（机构视角）
```

**密码重置**：

平台超级管理员可以对任意机构管理员执行「重置密码」操作。重置后密码恢复为默认值，`must_reset_pwd` 置为 `true`，下次登录时强制修改。

### 4.3 科室管理员

由机构管理员在本机构内创建，流程与机构管理员创建类似。科室管理员的权限范围限定在其所属的组织节点及其子节点内。

### 4.4 排班应用内的角色（排班负责人、普通员工）

这两个角色不直接拥有平台管理后台的登录权限。它们的身份来源于 `employees` 表，通过排班应用的独立登录入口访问系统。管理员在平台侧可以将某个员工标记为"排班负责人"，该标记影响排班应用内的功能权限。

---

## 五、功能职责划分 — 平台 vs 排班应用

### 5.1 整体划分原则

| 层面 | 平台管理后台（管理面） | 排班应用（应用面） |
|------|---------------------|------------------|
| 定位 | 配置中心、管控中心 | 执行中心、展示中心 |
| 用户 | 平台管理员、机构管理员、科室管理员 | 排班负责人、普通员工 |
| 核心动作 | 配置规则、管理人员、管理班次、管理组织 | 执行排班、查看排班、调整排班 |

### 5.2 功能归属明细

#### 5.2.1 组织管理（平台独有）

| 功能 | 操作角色 | 说明 |
|------|---------|------|
| 创建顶级机构 | 平台超级管理员 | 含创建机构管理员账号 |
| 编辑/停用机构 | 平台超级管理员 | 停用后该机构所有数据冻结 |
| 创建子节点（院区/科室） | 机构管理员 | 在本机构下创建下级组织 |
| 编辑/停用子节点 | 机构管理员 / 科室管理员 | 各自管理权限范围内的节点 |
| 查看组织树全景 | 平台超级管理员 | 跨机构全局视图 |
| 查看本机构组织树 | 机构管理员 | 仅看到本机构及下级 |

#### 5.2.2 人员管理（平台负责配置，排班应用读取）

| 功能 | 操作角色 | 归属 | 说明 |
|------|---------|------|------|
| 员工档案 CRUD | 机构管理员 / 科室管理员 | 平台 | 姓名、工号、职位、角色标签等 |
| 员工批量导入导出 | 机构管理员 | 平台 | Excel 导入导出 |
| 人员分组管理 | 机构管理员 / 科室管理员 | 平台 | 班组、项目组等分组 |
| 分组成员调整 | 科室管理员 | 平台 | 在本科室范围内调整 |
| 指派排班负责人 | 机构管理员 / 科室管理员 | 平台 | 从员工中选择，赋予排班应用内的排班权限 |
| 查看员工列表 | 排班负责人 | 排班应用 | 只读，用于排班选人 |

#### 5.2.3 班次管理（从排班应用上移到平台）

| 功能 | 操作角色 | 归属 | 说明 |
|------|---------|------|------|
| 班次模板 CRUD | 机构管理员 / 科室管理员 | **平台** | 名称、编码、时段、时长、颜色标识等 |
| 班次优先级与依赖配置 | 机构管理员 | **平台** | 班次间的拓扑关系 |
| 班次启用/停用 | 机构管理员 / 科室管理员 | **平台** | 控制哪些班次可用于排班 |
| 查看可用班次列表 | 排班负责人 | 排班应用 | 只读，创建排班时选择班次 |

#### 5.2.4 规则管理（从排班应用上移到平台）

| 功能 | 操作角色 | 归属 | 说明 |
|------|---------|------|------|
| 创建/编辑排班规则 | 机构管理员 | **平台** | 约束规则、偏好规则、依赖规则 |
| 规则继承查看 | 科室管理员 | **平台** | 查看从上级继承的规则（只读） |
| 规则覆盖（创建本级副本） | 科室管理员 | **平台** | 编辑继承规则时自动创建本级副本 |
| 规则禁用 | 科室管理员 | **平台** | 禁用从上级继承的某条规则（本级不生效） |
| 规则启用/恢复继承 | 科室管理员 | **平台** | 取消本级覆盖，恢复使用上级规则 |
| 查看生效规则集 | 排班负责人 | 排班应用 | 只读，排班前确认当前生效的规则 |

#### 5.2.5 排班操作（排班应用独有）

| 功能 | 操作角色 | 说明 |
|------|---------|------|
| 创建排班计划 | 排班负责人 | 选择周期、班次、人数需求 → 触发排班引擎 |
| 甘特图/表格视图查看 | 排班负责人 / 普通员工 | 排班负责人看全局，员工看个人 |
| 手动调整排班 | 排班负责人 | 拖拽/编辑 + 冲突实时检测 |
| 审核发布排班 | 排班负责人 | 草稿 → 已发布 |
| 查看个人排班 | 普通员工 | 仅看自己的排班 |
| 提交排班偏好 | 普通员工 | 偏好时段、休息需求等 |
| 提交请假申请 | 普通员工 | 请假后自动影响排班候选人过滤 |
| AI 对话排班（可选） | 排班负责人 | 自然语言查询/调整排班 |

#### 5.2.6 未来拓展的便利服务（排班应用内）

排班应用作为员工日常使用的入口，后续可以逐步增加以下便利服务（本期不实现，列出以明确产品方向）：

| 服务 | 说明 | 优先级 |
|------|------|--------|
| 换班申请 | 员工间发起换班请求，双方确认 + 管理员审批 | P1 |
| 加班记录 | 自动识别超时排班并记录加班工时 | P1 |
| 通知中心 | 排班变更通知、审批结果通知 | P1 |
| 排班统计个人看板 | 个人工作量、出勤率、加班时长等 | P2 |
| 排班日历同步 | 导出排班到个人日历（iCal） | P2 |
| 团队动态 | 同科室今日值班一览 | P3 |

---

## 六、规则继承与禁用机制（增强）

### 6.1 现有继承模型回顾

v0.2 方案中已定义的规则继承模型：子节点默认继承上级所有规则，可通过创建本级副本进行覆盖。

### 6.2 新增：规则禁用能力

在现有继承+覆盖的基础上，增加"禁用"操作。场景示例：机构级规则要求"夜班后必须休息 1 天"，但某急诊科室因人手紧张需要临时取消此限制。

**数据模型扩展**：

在现有的 `scheduling_rules` 表基础上新增字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `disabled` | BOOLEAN | 是否被本级禁用，默认 false |
| `disabled_by` | VARCHAR(64) | 禁用操作人 ID |
| `disabled_at` | DATETIME | 禁用时间 |
| `disabled_reason` | TEXT | 禁用原因（必填） |

**生效规则计算逻辑更新**：

```
对于科室 D 的排班：
  1. 收集 D 的所有祖先节点的规则（沿 path 向上遍历）
  2. 按从上到下的顺序合并：机构规则 → 院区规则 → 科室规则
  3. 同类型规则：下级覆盖上级
  4. 检查本级是否有禁用标记：disabled = true 的规则从生效集中移除
  5. 最终得到该科室的「生效规则集」
```

**禁用规则的展示**：

- 管理后台的规则列表中，被禁用的规则以灰色 + 删除线样式展示。
- 禁用原因以 tooltip 形式展示。
- 提供"恢复启用"操作按钮。

### 6.3 规则操作权限矩阵

| 操作 | 平台超级管理员 | 机构管理员 | 科室管理员 |
|------|:---:|:---:|:---:|
| 创建机构级规则 | ✓ | ✓ | ✗ |
| 编辑机构级规则 | ✓ | ✓ | ✗ |
| 删除机构级规则 | ✓ | ✓ | ✗ |
| 查看继承规则 | ✓ | ✓ | ✓ |
| 覆盖继承规则（创建本级副本） | ✓ | ✓ | ✓ |
| 禁用继承规则 | ✗ | ✓ | ✓ |
| 恢复被禁用的规则 | ✓ | ✓ | ✓ |
| 创建本级新规则 | ✓ | ✓ | ✓ |

---

## 七、排班应用权限管理（Application-level Permission）

### 7.1 问题定义

当前设计中，排班应用内的"排班负责人"和"普通员工"角色仅通过 `employees.scheduling_role` 字段做简单区分，缺少以下关键能力：

1. **谁来分配应用权限**：管理员在平台侧如何指定哪些员工拥有排班操作权限。
2. **权限粒度不足**：仅有 scheduler / employee 两档，无法覆盖"只读排班 + 可审批请假"等组合场景。
3. **权限生命周期管理**：权限的授予、回收、临时授权、到期自动回收等机制缺失。
4. **批量操作**：无法对一个分组/科室的员工批量授予应用角色。

### 7.2 设计方案

#### 7.2.1 应用角色体系

排班应用内定义独立的应用角色（App Role），与平台管理角色分离。应用角色由平台管理员/机构管理员在平台侧进行分配，员工在排班应用内看到的功能菜单和操作按钮由其应用角色决定。

| 应用角色 | 标识 | 能力说明 |
|---------|------|---------|
| 排班管理员 | `app:schedule_admin` | 创建排班计划、执行排班、审核发布、手动调整、查看全部排班、审批请假 |
| 排班负责人 | `app:scheduler` | 创建排班计划、执行排班、手动调整、查看本组/本科室排班 |
| 请假审批人 | `app:leave_approver` | 审批请假申请、查看本科室请假记录 |
| 普通员工 | `app:employee` | 查看个人排班、提交排班偏好、申请请假（默认角色，所有员工自动拥有） |

设计说明：

- 应用角色可**叠加**，一个员工可以同时拥有 `app:scheduler` + `app:leave_approver`。
- `app:employee` 是基础角色，所有员工自动拥有，无需手动分配。
- 应用角色与组织节点绑定，即"张三在心内科是排班负责人，但在急诊科只是普通员工"。

#### 7.2.2 权限分配流程

```
平台管理后台 → 人员管理 → 选择员工
  │
  ├── 方式一：单个员工分配
  │     └── 进入员工详情 → 「应用权限」Tab → 选择角色 → 选择生效节点 → 保存
  │
  ├── 方式二：批量分配
  │     └── 员工列表 → 勾选多人 → 「批量设置应用角色」→ 选择角色 → 选择生效节点 → 确认
  │
  └── 方式三：按分组分配
        └── 分组管理 → 选择分组 → 「设置分组默认应用角色」→ 新成员加入分组时自动继承
```

#### 7.2.3 权限生命周期

| 阶段 | 行为 | 说明 |
|------|------|------|
| 授予 | 管理员在平台侧分配应用角色 | 立即生效，员工下次打开排班应用时看到新权限 |
| 临时授权 | 分配时可设置有效期（可选） | 到期后自动回收，用于临时代班等场景 |
| 回收 | 管理员手动移除应用角色 | 立即生效 |
| 自动回收 | 员工离职/停用时自动清除所有应用角色 | 由 employees.status 变更触发 |
| 节点变更 | 员工调动到其他科室 | 原科室的应用角色不自动迁移，需管理员在新科室重新分配 |

#### 7.2.4 分组默认角色

分组（employee_groups）可以设置"默认应用角色"。当员工被加入该分组时，自动获得该分组绑定的应用角色。当员工被移出分组时，对应的应用角色自动回收（仅回收由分组授予的角色，不影响手动授予的）。

### 7.3 数据模型

#### 7.3.1 新增表：employee_app_roles（员工应用角色）

```sql
CREATE TABLE employee_app_roles (
    id              VARCHAR(64) PRIMARY KEY,
    employee_id     VARCHAR(64) NOT NULL,              -- → employees.id
    org_node_id     VARCHAR(64) NOT NULL,              -- 生效的组织节点
    app_role        VARCHAR(64) NOT NULL,              -- app:schedule_admin / app:scheduler / app:leave_approver
    source          ENUM('manual', 'group') NOT NULL DEFAULT 'manual',  -- 权限来源
    source_group_id VARCHAR(64) DEFAULT NULL,          -- 若 source=group，记录来源分组 ID
    granted_by      VARCHAR(64) NOT NULL,              -- 授权人（platform_users.id）
    granted_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME DEFAULT NULL,             -- NULL 表示永不过期
    
    UNIQUE KEY uk_emp_node_role (employee_id, org_node_id, app_role),
    INDEX idx_employee (employee_id),
    INDEX idx_node (org_node_id),
    INDEX idx_expires (expires_at),
    INDEX idx_source_group (source_group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 7.3.2 新增表：group_default_app_roles（分组默认应用角色）

```sql
CREATE TABLE group_default_app_roles (
    id              VARCHAR(64) PRIMARY KEY,
    group_id        VARCHAR(64) NOT NULL,              -- → employee_groups.id
    org_node_id     VARCHAR(64) NOT NULL,              -- 生效的组织节点
    app_role        VARCHAR(64) NOT NULL,              -- 默认授予的应用角色
    created_by      VARCHAR(64) NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_group_node_role (group_id, org_node_id, app_role),
    INDEX idx_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 7.3.3 调整：employees 表

将原方案中的 `scheduling_role` 字段**废弃**（或保留为向后兼容只读字段），应用角色统一由 `employee_app_roles` 表管理。

```sql
ALTER TABLE employees
    -- 废弃 scheduling_role，改用 employee_app_roles 表
    -- 保留字段但不再作为权限判断依据，仅供数据迁移期间使用
    MODIFY COLUMN scheduling_role ENUM('scheduler', 'employee') NOT NULL DEFAULT 'employee'
        COMMENT 'DEPRECATED: 请使用 employee_app_roles 表';
```

### 7.4 应用权限 API

#### 7.4.1 平台侧管理 API（管理员操作）

```
# --- 员工应用角色管理 ---
GET    /api/v1/platform/employees/:id/app-roles              # 查看员工的应用角色列表
POST   /api/v1/platform/employees/:id/app-roles              # 为员工分配应用角色
DELETE /api/v1/platform/employees/:id/app-roles/:roleId      # 移除员工的某个应用角色
POST   /api/v1/platform/employees/batch-app-roles            # 批量分配应用角色

# --- 分组默认角色管理 ---
GET    /api/v1/platform/groups/:id/default-app-roles         # 查看分组的默认应用角色
POST   /api/v1/platform/groups/:id/default-app-roles         # 设置分组默认应用角色
DELETE /api/v1/platform/groups/:id/default-app-roles/:roleId # 移除分组默认应用角色

# --- 应用角色查询（全局视图） ---
GET    /api/v1/platform/app-roles/summary                    # 按节点统计各角色人数
GET    /api/v1/platform/app-roles/expiring                   # 即将过期的临时授权列表
```

**分配应用角色请求体**：

```json
{
  "app_role": "app:scheduler",
  "org_node_id": "dept-001",
  "expires_at": null
}
```

**批量分配请求体**：

```json
{
  "employee_ids": ["emp-001", "emp-002", "emp-003"],
  "app_role": "app:scheduler",
  "org_node_id": "dept-001",
  "expires_at": "2026-06-30T23:59:59Z"
}
```

#### 7.4.2 排班应用侧 API（鉴权查询）

```
GET    /api/v1/app/scheduling/auth/my-roles                  # 当前员工在当前节点的应用角色列表
GET    /api/v1/app/scheduling/auth/permissions                # 当前员工的具体权限列表（由角色展开）
```

**响应示例**：

```json
{
  "employee_id": "emp-001",
  "org_node_id": "dept-001",
  "org_node_name": "心内科",
  "app_roles": [
    {
      "role": "app:scheduler",
      "source": "manual",
      "granted_at": "2026-03-01T10:00:00Z",
      "expires_at": null
    },
    {
      "role": "app:leave_approver",
      "source": "group",
      "source_group_name": "一线班组",
      "granted_at": "2026-02-15T08:00:00Z",
      "expires_at": null
    }
  ],
  "permissions": [
    "schedule:create",
    "schedule:edit",
    "schedule:view:all",
    "leave:approve",
    "leave:view:node",
    "schedule:view:self",
    "leave:create:self",
    "preference:edit:self"
  ]
}
```

### 7.5 应用角色 → 权限映射

```go
var appRolePermissions = map[string][]string{
    "app:schedule_admin": {
        "schedule:create", "schedule:edit", "schedule:publish",
        "schedule:view:all", "schedule:adjust",
        "leave:approve", "leave:view:node",
        "ai:chat",
    },
    "app:scheduler": {
        "schedule:create", "schedule:edit",
        "schedule:view:node", "schedule:adjust",
        "ai:chat",
    },
    "app:leave_approver": {
        "leave:approve", "leave:view:node",
    },
    "app:employee": {
        "schedule:view:self",
        "leave:create:self", "leave:view:self",
        "preference:edit:self",
    },
}
```

由于应用角色可叠加，最终权限为所有角色权限的**并集**。

### 7.6 权限检查流程（排班应用内）

```
员工请求排班应用 API
  → JWT 验证（app_token，由排班应用独立签发）
  → 提取 employee_id + org_node_id
  → 查询 employee_app_roles 表，获取当前节点下的所有有效角色
       ├── 过滤已过期的角色（expires_at < now）
       └── 合并 app:employee 基础角色
  → 展开角色为权限列表（并集）
  → 检查请求所需权限是否在权限列表中
  → 通过 → 执行业务逻辑
  → 不通过 → 返回 403
```

### 7.7 前端管理界面

#### 7.7.1 平台管理后台 — 员工详情页增加「应用权限」Tab

**Tab 内容**：

- 展示该员工在各组织节点下的应用角色列表，包括角色名称、生效节点、来源（手动/分组）、授权时间、过期时间。
- 操作按钮：新增角色、编辑（修改过期时间）、移除。
- 来源为"分组"的角色，操作列展示"由 XX 分组授予"并禁用直接移除按钮（需先从分组中移出）。

#### 7.7.2 平台管理后台 — 分组详情页增加「默认应用角色」配置

**配置内容**：

- 展示当前分组绑定的默认应用角色列表。
- 新增/移除默认角色后，系统弹出确认框提示"将对现有 N 名成员立即生效"。
- 新增默认角色时，对分组内已有成员自动补发 `employee_app_roles` 记录。
- 移除默认角色时，自动清除分组内成员中 `source=group` 且 `source_group_id` 匹配的记录。

#### 7.7.3 平台管理后台 — 应用权限总览页

新增独立页面 `/admin/app-permissions`，提供全局视图：

- **按节点统计**：树形展示各组织节点下各应用角色的人数。
- **即将过期**：列出未来 7 天内即将过期的临时授权，支持批量续期。
- **快速筛选**：按角色类型筛选员工列表，支持导出。

#### 7.7.4 排班应用 — 权限感知 UI

排班应用前端根据 `/auth/permissions` 接口返回的权限列表，动态控制界面元素：

| 权限 | UI 表现 |
|------|--------|
| `schedule:create` | 显示"创建排班计划"按钮 |
| `schedule:edit` | 甘特图/表格视图中启用拖拽编辑 |
| `schedule:publish` | 显示"审核发布"按钮 |
| `schedule:view:all` | 显示全科室排班 |
| `schedule:view:node` | 显示本科室排班 |
| `schedule:view:self` | 仅显示个人排班 |
| `leave:approve` | 显示请假审批入口 |
| `leave:create:self` | 显示请假申请入口 |
| `ai:chat` | 显示 AI 对话入口 |

### 7.8 过期自动回收机制

通过定时任务（Cron Job）每小时扫描一次 `employee_app_roles` 表，清除已过期的记录：

```go
// 定时任务：清理过期的应用角色
func (s *AppRoleService) CleanExpiredRoles(ctx context.Context) (int64, error) {
    result := s.db.WithContext(ctx).
        Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
        Delete(&EmployeeAppRole{})
    return result.RowsAffected, result.Error
}
```

同时，在每次权限查询时也做实时过滤（双重保障）。

### 7.9 应用权限操作的权限控制

| 操作 | 平台超级管理员 | 机构管理员 | 科室管理员 |
|------|:---:|:---:|:---:|
| 为任意节点的员工分配应用角色 | ✓ | ✗ | ✗ |
| 为本机构内员工分配应用角色 | ✓ | ✓ | ✗ |
| 为本科室内员工分配应用角色 | ✓ | ✓ | ✓ |
| 分配 `app:schedule_admin` 角色 | ✓ | ✓ | ✗ |
| 分配 `app:scheduler` 角色 | ✓ | ✓ | ✓ |
| 分配 `app:leave_approver` 角色 | ✓ | ✓ | ✓ |
| 设置分组默认角色 | ✓ | ✓ | ✓（本科室分组） |
| 查看应用权限总览 | ✓ | ✓（本机构） | ✓（本科室） |

---

## 八、数据模型变更

### 8.1 新增表：platform_users（平台账号）

> 注：此表对应原方案中的 `users` 表，重命名以明确其"平台账号"定位。

```sql
CREATE TABLE platform_users (
    id              VARCHAR(64) PRIMARY KEY,
    username        VARCHAR(64) NOT NULL UNIQUE,
    email           VARCHAR(128) NOT NULL UNIQUE,
    phone           VARCHAR(20) DEFAULT NULL,
    password_hash   VARCHAR(256) NOT NULL,
    status          ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    must_reset_pwd  BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- V2 预留：绑定机构员工
    bound_employee_id VARCHAR(64) DEFAULT NULL,
    
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_email (email),
    INDEX idx_phone (phone),
    INDEX idx_bound_employee (bound_employee_id)
);
```

### 8.2 调整表：user_node_roles

保持原有结构不变，`user_id` 指向 `platform_users.id`。

```sql
CREATE TABLE user_node_roles (
    id          VARCHAR(64) PRIMARY KEY,
    user_id     VARCHAR(64) NOT NULL,          -- → platform_users.id
    org_node_id VARCHAR(64) NOT NULL,          -- → org_nodes.id
    role_id     VARCHAR(64) NOT NULL,          -- → roles.id
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_user_node_role (user_id, org_node_id, role_id),
    INDEX idx_user (user_id),
    INDEX idx_node (org_node_id)
);
```

### 8.3 调整表：scheduling_rules（增加禁用字段）

```sql
ALTER TABLE scheduling_rules
    ADD COLUMN disabled         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN disabled_by      VARCHAR(64) DEFAULT NULL,
    ADD COLUMN disabled_at      DATETIME DEFAULT NULL,
    ADD COLUMN disabled_reason  TEXT DEFAULT NULL;
```

### 8.4 调整表：employees（增加应用登录凭证）

```sql
ALTER TABLE employees
    ADD COLUMN scheduling_role    ENUM('scheduler', 'employee') NOT NULL DEFAULT 'employee'
        COMMENT 'DEPRECATED: 向后兼容字段，实际权限由 employee_app_roles 表管理',
    ADD COLUMN app_password_hash  VARCHAR(256) DEFAULT NULL,
    ADD COLUMN app_must_reset_pwd BOOLEAN NOT NULL DEFAULT TRUE;
```

说明：`scheduling_role` 字段保留为向后兼容，但实际应用权限判断已迁移到 `employee_app_roles` 表。`app_password_hash` 和 `app_must_reset_pwd` 为排班应用独立的登录凭证（与平台账号体系分离）。

### 8.5 新增表：employee_app_roles（员工应用角色）

详见第七章 7.3.1 节。此表用于管理员工在排班应用内的角色与权限，支持角色叠加、临时授权、分组自动授予等能力。

### 8.6 新增表：group_default_app_roles（分组默认应用角色）

详见第七章 7.3.2 节。此表用于为分组设置默认应用角色，新成员加入分组时自动继承。

---

## 九、API 设计

### 9.1 平台管理后台 API（管理面）

#### 9.1.1 机构管理

```
POST   /api/v1/admin/organizations              # 创建机构（含创建机构管理员）
GET    /api/v1/admin/organizations              # 机构列表
GET    /api/v1/admin/organizations/:id          # 机构详情
PUT    /api/v1/admin/organizations/:id          # 编辑机构
PUT    /api/v1/admin/organizations/:id/suspend  # 停用机构
PUT    /api/v1/admin/organizations/:id/activate # 恢复机构
```

**创建机构请求体**：

```json
{
  "name": "XX 市第一人民医院",
  "code": "hospital-001",
  "contact_name": "张三",
  "contact_phone": "13800138000",
  "admin": {
    "username": "hospital001_admin",
    "email": "admin@hospital001.com"
  }
}
```

**创建机构响应体**：

```json
{
  "organization": {
    "id": "org_xxx",
    "name": "XX 市第一人民医院",
    "code": "hospital-001",
    "status": "active"
  },
  "admin": {
    "id": "user_xxx",
    "username": "hospital001_admin",
    "default_password": "Hospital001@2026",
    "must_reset_pwd": true
  }
}
```

#### 9.1.2 平台账号管理

```
GET    /api/v1/admin/platform-users              # 平台账号列表
POST   /api/v1/admin/platform-users              # 创建平台账号（下级管理员）
PUT    /api/v1/admin/platform-users/:id          # 编辑账号
PUT    /api/v1/admin/platform-users/:id/reset-pwd # 重置密码
PUT    /api/v1/admin/platform-users/:id/disable   # 禁用账号
```

#### 9.1.3 班次管理（上移到平台）

```
GET    /api/v1/platform/shifts                    # 班次列表（按当前节点过滤）
POST   /api/v1/platform/shifts                    # 创建班次
PUT    /api/v1/platform/shifts/:id                # 编辑班次
DELETE /api/v1/platform/shifts/:id                # 删除班次
PUT    /api/v1/platform/shifts/:id/toggle         # 启用/停用
GET    /api/v1/platform/shifts/dependencies       # 班次依赖关系
POST   /api/v1/platform/shifts/dependencies       # 配置依赖关系
```

#### 9.1.4 规则管理（上移到平台）

```
GET    /api/v1/platform/rules                     # 规则列表（含继承规则标记）
POST   /api/v1/platform/rules                     # 创建规则
PUT    /api/v1/platform/rules/:id                 # 编辑规则
DELETE /api/v1/platform/rules/:id                 # 删除规则
PUT    /api/v1/platform/rules/:id/disable         # 禁用继承规则
PUT    /api/v1/platform/rules/:id/enable          # 恢复启用
GET    /api/v1/platform/rules/effective            # 查看当前节点生效规则集
```

#### 9.1.5 人员管理

```
GET    /api/v1/platform/employees                 # 员工列表
POST   /api/v1/platform/employees                 # 创建员工
PUT    /api/v1/platform/employees/:id             # 编辑员工
DELETE /api/v1/platform/employees/:id             # 删除员工
POST   /api/v1/platform/employees/import          # 批量导入
GET    /api/v1/platform/employees/export          # 导出
PUT    /api/v1/platform/employees/:id/set-scheduler # 指派为排班负责人
```

#### 9.1.6 分组管理

```
GET    /api/v1/platform/groups                    # 分组列表
POST   /api/v1/platform/groups                    # 创建分组
PUT    /api/v1/platform/groups/:id                # 编辑分组
DELETE /api/v1/platform/groups/:id                # 删除分组
GET    /api/v1/platform/groups/:id/members        # 分组成员
POST   /api/v1/platform/groups/:id/members        # 添加成员
DELETE /api/v1/platform/groups/:id/members/:eid   # 移除成员
```

### 9.2 排班应用 API（应用面）

排班应用的 API 前缀改为 `/api/v1/app/scheduling`，与平台管理 API 明确区分。

```
# --- 排班操作 ---
POST   /api/v1/app/scheduling/plans                # 创建排班计划
GET    /api/v1/app/scheduling/plans                # 排班计划列表
GET    /api/v1/app/scheduling/plans/:id            # 排班计划详情
PUT    /api/v1/app/scheduling/plans/:id/execute    # 执行排班（触发引擎）
PUT    /api/v1/app/scheduling/plans/:id/adjust     # 手动调整
PUT    /api/v1/app/scheduling/plans/:id/publish    # 发布排班

# --- 排班查看 ---
GET    /api/v1/app/scheduling/view/gantt           # 甘特图数据
GET    /api/v1/app/scheduling/view/table           # 表格数据
GET    /api/v1/app/scheduling/view/my              # 个人排班
GET    /api/v1/app/scheduling/view/stats           # 统计看板

# --- 只读引用（从平台读取） ---
GET    /api/v1/app/scheduling/ref/shifts           # 可用班次列表（只读）
GET    /api/v1/app/scheduling/ref/rules            # 生效规则集（只读）
GET    /api/v1/app/scheduling/ref/employees        # 可排班人员列表（只读）
GET    /api/v1/app/scheduling/ref/groups           # 分组列表（只读）

# --- 员工自助 ---
POST   /api/v1/app/scheduling/preferences          # 提交排班偏好
POST   /api/v1/app/scheduling/leaves               # 提交请假申请
GET    /api/v1/app/scheduling/leaves/my             # 我的请假记录

# --- AI 对话（可选） ---
POST   /api/v1/app/scheduling/ai/chat              # AI 对话排班
```

---

## 十、权限模型更新

### 10.1 RBAC 权限定义更新

```go
var rolePermissions = map[string][]string{
    "platform_admin": {"*"},
    "org_admin": {
        "org:*",
        "platform_user:manage:child",    // 管理下级管理员
        "employee:*",
        "shift:*",
        "rule:*",
        "group:*",
        "schedule:read",                 // 可查看排班结果但不直接操作
        "leave:*",
        "ai:*",
    },
    "dept_admin": {
        "employee:read", "employee:update:own_node",
        "shift:read", "shift:create:own_node", "shift:update:own_node",
        "rule:read", "rule:create:own_node", "rule:update:own_node", "rule:disable:inherited",
        "group:*:own_node",
        "schedule:read:own_node",
        "leave:read:own_node",
    },
    "scheduler": {
        "schedule:*:own_node",
        "employee:read:own_node",
        "shift:read:own_node",
        "rule:read:own_node",
        "group:read:own_node",
        "leave:read:own_node",
    },
    "employee": {
        "schedule:read:self",
        "leave:create:self",
        "preference:*:self",
    },
}
```

### 10.2 权限检查增强

新增 `own_node` 权限修饰符：限制操作范围仅限当前登录节点及其子节点。

```go
// 权限检查示例
func RequirePermission(permission string) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := GetClaims(r.Context())
            
            // 基础权限检查
            if !HasPermission(claims.RoleName, permission) {
                http.Error(w, "forbidden", 403)
                return
            }
            
            // own_node 范围检查
            if strings.Contains(permission, "own_node") {
                targetNodeID := extractTargetNode(r)
                if !isDescendantOf(targetNodeID, claims.OrgNodeID) {
                    http.Error(w, "forbidden: out of scope", 403)
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 十一、前端页面结构

### 11.1 平台管理后台（Admin Frontend）

```
/admin
  ├── /login                          # 平台管理员登录
  ├── /force-reset-password           # 强制修改密码
  ├── /dashboard                      # 平台概览（机构数量、活跃度等）
  │
  ├── /organizations                  # 机构管理（仅平台超级管理员可见）
  │     ├── /list                     # 机构列表
  │     └── /create                   # 创建机构
  │
  ├── /org-tree                       # 组织树管理
  │     └── /nodes                    # 节点列表与操作
  │
  ├── /employees                      # 人员管理
  │     ├── /list                     # 员工列表
  │     ├── /import                   # 批量导入
  │     └── /groups                   # 分组管理
  │
  ├── /shifts                         # 班次管理
  │     ├── /list                     # 班次列表
  │     └── /dependencies             # 依赖关系配置
  │
  ├── /rules                          # 规则管理
  │     ├── /list                     # 规则列表（含继承/覆盖/禁用状态）
  │     └── /effective                # 生效规则预览
  │
  ├── /platform-users                 # 平台账号管理
  │     └── /list                     # 账号列表
  │
  └── /settings                       # 系统设置
        ├── /ai-config                # AI 模型配置
        └── /subscription             # 订阅与配额
```

### 11.2 排班应用（Scheduling App Frontend）

```
/app
  ├── /login                          # 员工登录（排班应用入口）
  │
  ├── /schedule                       # 排班中心
  │     ├── /gantt                    # 甘特图视图
  │     ├── /table                    # 表格视图
  │     ├── /create                   # 创建排班计划（排班负责人）
  │     └── /my                       # 我的排班（普通员工）
  │
  ├── /leave                          # 请假管理
  │     ├── /apply                    # 申请请假
  │     └── /my                       # 我的请假
  │
  ├── /preference                     # 排班偏好设置
  │
  ├── /stats                          # 统计看板
  │
  └── /ai-chat                        # AI 对话排班（可选）
```

---

## 十二、迁移方案

### 12.1 从现有架构迁移

由于原有系统中班次管理和规则管理的 API 在 Management Service 内（路径为 `/api/v1/shifts`、`/api/v1/rules`），需要进行以下迁移：

| 迁移项 | 原路径 | 新路径 | 变更说明 |
|--------|-------|--------|---------|
| 班次 CRUD | `/api/v1/shifts` | `/api/v1/platform/shifts` | 移入平台管理 API 命名空间 |
| 规则 CRUD | `/api/v1/rules` | `/api/v1/platform/rules` | 移入平台管理 API 命名空间 |
| 排班操作 | `/api/v1/schedules` | `/api/v1/app/scheduling/plans` | 移入排班应用 API 命名空间 |
| 员工 CRUD | `/api/v1/employees` | `/api/v1/platform/employees` | 移入平台管理 API 命名空间 |
| 分组 CRUD | `/api/v1/groups` | `/api/v1/platform/groups` | 移入平台管理 API 命名空间 |

### 12.2 数据迁移

1. `users` 表重命名为 `platform_users`。
2. `employees` 表新增 `scheduling_role`、`app_password_hash`、`app_must_reset_pwd` 字段。
3. `scheduling_rules` 表新增 `disabled`、`disabled_by`、`disabled_at`、`disabled_reason` 字段。
4. 编写向后兼容的 API 适配层（过渡期内旧路径 302 重定向到新路径）。

### 12.3 前端迁移

1. 现有 `frontend/admin` 保持为平台管理后台，增加班次管理和规则管理页面。
2. 现有 `frontend/web`（排班应用前端）移除班次管理和规则管理的编辑功能，改为只读引用。
3. 排班应用前端新增独立的员工登录页面。

---

## 十三、验收标准

| 编号 | 验收项 | 通过条件 |
|------|--------|---------|
| AC-01 | 平台超级管理员创建机构 | 创建机构时自动创建机构管理员账号，返回默认密码 |
| AC-02 | 首次登录强制改密 | 使用默认密码登录后，系统强制跳转修改密码页面，不允许跳过 |
| AC-03 | 密码重置 | 平台管理员重置机构管理员密码后，该账号下次登录需强制改密 |
| AC-04 | 班次管理在平台侧 | 班次的创建、编辑、删除操作只能在平台管理后台完成 |
| AC-05 | 规则管理在平台侧 | 规则的创建、编辑、删除、禁用操作只能在平台管理后台完成 |
| AC-06 | 排班应用只读引用 | 排班应用中的班次和规则列表为只读，不可编辑 |
| AC-07 | 规则继承正确 | 机构级规则自动对下级科室生效 |
| AC-08 | 规则禁用正确 | 科室管理员禁用某条继承规则后，排班引擎的生效规则集中不包含该规则 |
| AC-09 | 规则恢复正确 | 恢复被禁用的规则后，该规则重新出现在生效规则集中 |
| AC-10 | 权限隔离 | 机构管理员看不到其他机构的数据，科室管理员看不到其他科室的数据 |
| AC-11 | 排班负责人指派 | 平台侧指派排班负责人后，该员工在排班应用内获得排班操作权限 |
| AC-12 | 平台账号独立性 | 平台管理员账号与机构员工账号互不影响，各自独立登录 |
| AC-13 | 应用角色分配 | 管理员为员工分配 `app:scheduler` 角色后，该员工登录排班应用可看到"创建排班"按钮 |
| AC-14 | 应用角色叠加 | 同时拥有 `app:scheduler` + `app:leave_approver` 的员工，可同时操作排班和审批请假 |
| AC-15 | 批量分配应用角色 | 勾选多名员工批量分配应用角色，全部员工立即生效 |
| AC-16 | 分组默认角色 | 分组设置默认角色后，新加入成员自动获得该角色；移出成员时自动回收分组授予的角色 |
| AC-17 | 临时授权到期回收 | 设置有效期的应用角色到期后，员工在排班应用内不再拥有对应权限 |
| AC-18 | 应用权限感知 UI | 无 `schedule:create` 权限的员工在排班应用中看不到"创建排班"按钮 |
| AC-19 | 应用权限总览 | 管理后台的应用权限总览页正确展示各节点各角色人数统计 |
| AC-20 | 员工停用联动 | 员工状态设为停用后，其所有应用角色自动清除 |

---

## 十四、风险与约束

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| API 路径变更导致前端大面积修改 | 开发工期增加 | 提供过渡期 302 重定向，前端分批迁移 |
| 两套登录体系增加用户认知负担 | 用户体验 | 平台后台和排班应用使用不同域名/路径，入口明确区分 |
| 规则禁用可能导致排班质量下降 | 业务风险 | 禁用操作必填原因，且在排班结果中标记"因规则禁用可能存在风险" |
| 员工绑定平台账号（V2）的设计可能影响当前表结构 | 技术债务 | 仅预留 `bound_employee_id` 字段，不实现逻辑 |
| 应用角色叠加导致权限组合爆炸 | 权限管理复杂度 | 应用角色控制在 4 种以内；提供权限总览页辅助管理 |
| 分组默认角色与手动授予角色冲突 | 数据一致性 | 通过 `source` 字段区分来源，分组回收只影响分组授予的角色 |

---

## 十五、里程碑

| 阶段 | 交付内容 | 预计工期 |
|------|---------|---------|
| M1 | 平台账号体系 + 机构创建含管理员 + 强制改密 | 1 周 |
| M2 | 班次管理、规则管理迁移到平台 API + 规则禁用功能 | 1.5 周 |
| M2.5 | 应用权限模型（`employee_app_roles` 表 + 分配/回收 API + 分组默认角色） | 1 周 |
| M3 | 排班应用 API 命名空间重构 + 只读引用接口 + 应用内权限检查中间件 | 1 周 |
| M4 | 前端管理后台增加班次/规则/应用权限管理页面 | 1.5 周 |
| M5 | 排班应用前端移除编辑功能 + 员工独立登录 + 权限感知 UI | 1 周 |
| M6 | 集成测试 + 验收 | 0.5 周 |

**总计预估工期：7.5 周**