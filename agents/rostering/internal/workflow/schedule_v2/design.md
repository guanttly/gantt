# 排班工作流

## 流程定义

排班核心是人-岗匹配，所以所有的流程都是围绕将**符合要求的人**放入到**预定义班次（岗）**中，对于所有排班而言，核心抽象是不变的：

### Setp1: 收集排班信息

1.  时间信息：例如本周，下周等等时间范围
    
2.  班次信息：参与排班的班次，包括固定班次，特殊班次，普通班次
    
3.  人员信息：班次所要求的人员，一个人员可能满足多个班次的要求
    
4.  规则信息：不同班次有不同的排班规则，不同人员也有不同的排班需求
    

### Setp2: 梳理规则分类

当前抽象的规则可以初步分类为：

1.  班次类规则
    
2.  人员类规则
    

对于**班次类规则**：它不影响排班流程的优先级，它仅在涉及到的班次排班进行参考/执行的规则

对于**人员类规则：**它可以进一步细分为**需求类规则**和**偏好类规则**，对于需求类规则，原则上需要优先考虑，将对应的人员按照具体需求，先进行班次占位。**偏好类**也一样，不过它相对灵活，当产生规则冲突时允许被打破。

对于人员需求类规则，按照作用域来描述，分为：**常态化需求**和**临时需求。**

**常态化需求：**每次进行排班都需要考虑的需求。

**临时需求：**仅对本次排班有效，本次排班结束，该需求则被销毁。

### Setp3: 制订排班优先级

排班的成败取决于对规则的遵从程度，不同规则本身优先级也不同。但是从大方向的逻辑来看，排班普遍遵循以下优先级：

1.  固定班次：固定班次是最高优先级，它们不需要AI参与排班，而且在排班表上占据固定位置
    
2.  个人需求：不同的人对不同的班次有着自己的需求，优先对班次进行占位。
    
3.  特殊班次：由于特殊班次中的人员很多与普通班次人员重叠，通常有技能要求，因此特殊班次需要进行优先排班
    
4.  普通班次：按照既定目标，将剩余的人员按照普适性规则，填充到排班表中，直到满足校验要求
    
5.  科研班次：对于科研班次而言，优先级较低但是还是有必要进行排班的
    
6.  填充班次：完成上述排班后，有些人员的班次没达到要求，需要用类似行政/年假等班次进行填充
    

### Setp4: 梳理排班用户交互及干预节点

| 阶段 | 用户动作 | 描述 |
| --- | --- | --- |
| 信息收集 | 确认 | 1.确认排班时间<br>2.确认待排班次<br>3.确认每个班次每天的排班人数 |
| 固定班次 | 确认 | 用户确认固定班次排班情况，是否需要修改 |
| 个人需求 | 确认及补充 | 1.  用户明确常态化的个人需求，同时在这个阶段可以额外加上本次排班的临时需求<br>    <br>2.  确认排班结果是否符合预期 |
| 特殊班次 | 确认 | 确认排班结果是否符合预期 |
| 普通班次 | 确认 | 确认排班结果是否符合预期 |
| 科研班次 | 确认 | 确认排班结果是否符合预期 |
| 填充班次 | 确认 | 确认排班结果是否符合预期 |

### Setp5: 存储排班并提供统计报表

---

## V2 工作流实施说明

### 架构设计

V2版本采用**优先级驱动的阶段式工作流**架构，父工作流 `schedule_v2.create` 按照优先级顺序编排各个阶段，每个需要AI排班的阶段复用统一的 `schedule.core` 子工作流。

```
┌─────────────────────────────────────────────────────────────┐
│                  schedule_v2.create                         │
│                    (父工作流)                                │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  ┌──────────┐        ┌──────────┐       ┌──────────┐
  │信息收集   │        │个人需求   │       │固定班次   │
  │子工作流   │        │收集确认   │       │自动填充   │
  └──────────┘        └──────────┘       └──────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  ┌──────────┐        ┌──────────┐       ┌──────────┐
  │特殊班次   │        │普通班次   │       │科研班次   │
  │(循环Core) │        │(循环Core) │       │(循环Core) │
  └──────────┘        └──────────┘       └──────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
  ┌──────────┐        ┌──────────┐       ┌──────────┐
  │填充班次   │        │确认保存   │       │完成       │
  │处理      │        │子工作流   │       │          │
  └──────────┘        └──────────┘       └──────────┘
```

### 班次类型识别

使用现有的 `Shift.Type` 字段进行班次分类，**无需修改数据库结构**。

#### 工作流类型（后端）

| 类型值 | 常量名 | 说明 | 处理方式 | 优先级 |
|--------|--------|------|----------|--------|
| `fixed` | `ShiftTypeFixed` | 固定班次 | 自动填充，无需AI | 1（最高） |
| `special` | `ShiftTypeSpecial` | 特殊班次 | 优先排班，调用Core子工作流 | 3（高） |
| `normal` | `ShiftTypeNormal` | 普通班次（默认） | 常规排班，调用Core子工作流 | 4（中） |
| `research` | `ShiftTypeResearch` | 科研班次 | 较低优先级，调用Core子工作流 | 5（较低） |
| `fill` | `ShiftTypeFill` | 填充班次 | 补充排班不足 | 6（最低） |
| `leave` | `ShiftTypeLeave` | 请假班次 | 属于填充类 | 6（最低） |

#### 前端类型映射

前端管理界面使用的类型会**自动映射**到工作流类型：

| 前端类型 | 前端显示 | 映射到工作流类型 | 优先级 |
|---------|---------|----------------|--------|
| `regular` | 常规班次 | `normal` | 4（中） |
| `overtime` | 加班班次 | `special` | 3（高） |
| `standby` | 备班班次 | `special` | 3（高） |

**自动映射**：系统在 `ClassifyShiftsByType()` 函数中自动完成映射，前端无需修改。

**向后兼容**：
- ✅ 前端继续使用 `regular`/`overtime`/`standby`
- ✅ `Type` 字段为空时，默认视为 `normal` 类型
- ✅ 已有数据无需修改

详见：[前后端类型对齐说明](FRONTEND_TYPE_MAPPING.md)

### 工作流状态机

#### 父工作流状态

```go
CreateV2StateInit              // 初始化
CreateV2StateInfoCollecting    // 信息收集中
CreateV2StatePersonalNeeds     // 个人需求确认
CreateV2StateFixedShift        // 固定班次处理
CreateV2StateSpecialShift      // 特殊班次排班
CreateV2StateNormalShift       // 普通班次排班
CreateV2StateResearchShift     // 科研班次排班
CreateV2StateFillShift         // 填充班次处理
CreateV2StateConfirmSaving     // 确认保存
CreateV2StateCompleted         // 完成
CreateV2StateFailed            // 失败
CreateV2StateCancelled         // 取消
```

#### 核心事件

```go
CreateV2EventStart                    // 启动
CreateV2EventInfoCollected            // 信息收集完成
CreateV2EventPersonalNeedsConfirmed   // 个人需求确认
CreateV2EventFixedShiftConfirmed      // 固定班次确认
CreateV2EventShiftPhaseComplete       // 班次阶段完成
CreateV2EventShiftCompleted           // 单个班次完成
CreateV2EventSaveCompleted            // 保存完成
CreateV2EventUserCancel               // 用户取消
CreateV2EventSkipPhase                // 跳过阶段
```

### 核心数据结构

#### CreateV2Context - 工作流上下文

```go
type CreateV2Context struct {
    // 基础信息（来自 InfoCollect）
    StartDate         string
    EndDate           string
    SelectedShifts    []*Shift
    StaffList         []*Employee
    Rules             []*Rule
    
    // 个人需求（独立收集）
    PersonalNeeds     map[string][]*PersonalNeed
    
    // 已占位信息（累积约束）
    OccupiedSlots     map[string]map[string]string
    
    // 分阶段结果
    FixedShiftResults    *PhaseResult
    SpecialShiftResults  *PhaseResult
    NormalShiftResults   *PhaseResult
    ResearchShiftResults *PhaseResult
    FillShiftResults     *PhaseResult
    
    // 当前阶段进度
    CurrentPhase      string
    CurrentShiftIndex int
    PhaseShiftList    []*Shift
    
    // 分类后的班次
    ClassifiedShifts  map[string][]*Shift
}
```

#### PersonalNeed - 个人需求

```go
type PersonalNeed struct {
    StaffID       string   // 人员ID
    NeedType      string   // "permanent" | "temporary"
    RequestType   string   // "prefer" | "avoid" | "must"
    TargetShiftID string   // 目标班次
    TargetDates   []string // 目标日期列表
    Priority      int      // 优先级
    Source        string   // "rule" | "user"
}
```

### 约束累积机制

每个阶段完成后，结果会被记录到 `OccupiedSlots` 和 `ExistingScheduleMarks` 中，作为后续阶段的约束条件：

```go
// 人员在某日期已被分配的班次
OccupiedSlots: map[staffID]map[date]shiftID

// 人员在某日期的班次标记（含时段信息）
ExistingScheduleMarks: map[staffID]map[date][]ShiftMark
```

这确保：
1. 同一人员同一天不会被分配多个班次
2. 避免时段冲突（如跨夜班次）
3. 后续阶段只能使用未被占用的人员-日期槽位

### 核心辅助函数

| 函数名 | 功能 | 用途 |
|--------|------|------|
| `ClassifyShiftsByType()` | 按 Type 字段分类班次 | 初始化时分类所有班次 |
| `SortShiftsBySchedulingPriority()` | 按优先级排序 | 每个阶段开始前排序 |
| `ExtractPersonalNeeds()` | 从规则提取个人需求 | 个人需求阶段 |
| `BuildOccupiedSlotsMap()` | 构建已占位映射 | 约束管理 |
| `MergeOccupiedSlots()` | 合并占位信息 | 每个班次完成后 |
| `DetectUnderScheduledStaff()` | 检测排班不足人员 | 填充班次阶段 |
| `MergeScheduleDrafts()` | 合并多阶段结果 | 最终保存前 |

### 文件结构

```
schedule_v2/
├── create/
│   ├── actions.go        # Action 实现（809行）
│   ├── context.go        # 上下文数据结构（268行）
│   ├── definition.go     # 工作流定义（244行）
│   ├── helpers.go        # 辅助工具函数（459行）
│   └── main.go           # 包注册
├── core/
│   ├── actions.go        # 核心排班 Actions（711行）
│   └── definition.go     # 核心子工作流定义（165行）
├── main.go               # 总入口
└── design.md             # 本文档
```

### 使用示例

```go
// 1. 在 main.go 中导入
import (
    _ "jusha/agent/rostering/internal/workflow/schedule_v2/create"
)

// 2. 启动工作流
workflow := engine.GetWorkflow(WorkflowScheduleCreateV2)
ctx := engine.NewContext(session, services)
err := ctx.Send(context.Background(), CreateV2EventStart, nil)

// 3. 监听状态变化和用户交互
// 工作流会在每个阶段提示用户确认
```

### 灵活扩展

#### 添加新班次类型

1. 在 `create_v2.go` 中定义新的班次类型常量
2. 在 `definition.go` 中添加新状态和转换
3. 在 `actions.go` 中实现对应的 Action
4. 在 `ClassifyShiftsByType()` 中添加识别逻辑

无需修改 Core 子工作流，保持了良好的解耦。

### 待完善项（TODO标记）

1. **InfoCollect 子工作流集成**
   - 当前使用模拟数据
   - 需要实现完整的信息收集子工作流
   
2. **Core 子工作流调用**
   - 已有框架代码
   - 需要等待子工作流引擎功能完善

3. **填充班次详细逻辑**
   - `DetectUnderScheduledStaff()` 已实现
   - 需要实现填充策略和用户交互

4. **确认保存功能**
   - 需要实现预览界面
   - 保存到数据库
   - 生成统计报表

5. **规则解析细化**
   - `parseRuleToPersonalNeed()` 需要根据实际规则格式完善
   - 支持更多规则类型的解析

6. **冲突检测**
   - 在 `MergeScheduleDrafts()` 中实现完整的冲突检测逻辑

### 测试建议

1. **单元测试**：测试 helpers 中的分类、排序、合并等纯函数
2. **集成测试**：模拟完整工作流从 Init 到 Completed
3. **用户交互测试**：验证每个确认点的交互逻辑
4. **约束验证测试**：确保约束正确传递和生效
5. **边界条件测试**：空班次列表、无可用人员等场景

### 版本历史

- **V2.1 (2025-12-17)**：班次类型独立表设计
  - 将班次类型从硬编码改为数据库表管理
  - 支持动态配置类型和优先级
  - 前后端统一类型定义
  - 更灵活的扩展能力
  - 详见 `SHIFT_TYPE_MIGRATION_GUIDE.md`

- **V2 (2025-12-17)**：初始实现
  - 采用优先级驱动的阶段式架构
  - 复用现有 Shift.Type 字段
  - 实现约束累积机制
  - 支持灵活扩展

---

## 班次类型独立表设计（V2.1 新增）

### 设计理念

将班次类型从硬编码字符串改为数据库表管理，实现更灵活的配置和扩展能力。

### 核心架构

```
┌─────────────────┐         ┌──────────────────┐
│  shift_types    │         │     shifts       │
├─────────────────┤         ├──────────────────┤
│ id (PK)         │<────────│ shift_type_id(FK)│
│ org_id          │         │ name             │
│ code            │         │ start_time       │
│ name            │         │ end_time         │
│ priority        │         │ ...              │
│ workflow_phase  │         └──────────────────┘
│ color           │
│ is_ai_scheduling│
│ ...             │
└─────────────────┘
```

### 数据库表设计

```sql
CREATE TABLE shift_types (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    code VARCHAR(50) NOT NULL,              -- regular, overtime, standby
    name VARCHAR(100) NOT NULL,             -- 常规班次、加班班次
    scheduling_priority INT NOT NULL,       -- 1-100（越小越优先）
    workflow_phase VARCHAR(50) NOT NULL,    -- normal/special/research/fixed/fill
    color VARCHAR(20),
    is_ai_scheduling BOOLEAN DEFAULT TRUE,
    -- 更多配置字段...
);

ALTER TABLE shifts 
    ADD COLUMN shift_type_id VARCHAR(36),
    ADD INDEX idx_shift_type (shift_type_id);
```

### 系统内置类型

| 编码 | 名称 | 优先级 | 工作流阶段 | 说明 |
|------|------|--------|-----------|------|
| fixed | 固定班次 | 10 | fixed | 每周固定人员，无需AI |
| special | 特殊班次 | 30 | special | 有技能要求，优先排班 |
| overtime | 加班班次 | 31 | special | 节假日加班 |
| standby | 备班班次 | 32 | special | 待命或应急 |
| regular | 常规班次 | 50 | normal | 日常工作班次 |
| research | 科研班次 | 70 | research | 科研或学习时间 |
| fill | 填充班次 | 90 | fill | 补充排班不足 |

### 前后端类型对接

**前端类型（已有）**：
- `regular` - 常规班次
- `overtime` - 加班班次
- `standby` - 备班班次

**后端映射策略**：
```go
// 方案1：前端直接使用 code
shiftTypeCode := "regular"  // 前端选择

// 方案2：后端提供类型下拉列表API
GET /api/v1/shift-types?org_id=xxx
// 返回：[{id, code, name, color, priority}, ...]

// 创建班次时传递 shift_type_id
POST /api/v1/shifts
{
  "name": "白班",
  "shiftTypeId": "xxx-uuid",  // 从下拉列表获取
  ...
}
```

### 工作流集成

```go
// 动态获取班次类型并排序
func actStartSpecialShiftPhase(ctx context.Context, wctx engine.Context, payload any) error {
    v2ctx := wctx.Data().(*CreateV2Context)
    
    // 从服务获取类型信息
    shiftTypeService := getShiftTypeService(wctx)
    phaseGroups, err := ClassifyShiftsByTypeWithSort(
        ctx, 
        v2ctx.SelectedShifts, 
        shiftTypeService,
    )
    
    // 找到 special 阶段的班次（动态查询）
    for _, group := range phaseGroups {
        if group.Phase == "special" {
            v2ctx.PhaseShiftList = group.Shifts
            break
        }
    }
    
    // 循环处理该阶段的班次...
}
```

### 优势

1. **灵活配置**：管理员可在界面调整类型和优先级，无需修改代码
2. **统一定义**：前后端共享同一套类型定义
3. **易扩展**：可添加更多类型属性（颜色、图标、业务规则）
4. **组织定制**：支持不同组织自定义班次类型

### 迁移步骤

1. ✅ 数据库表创建（`shift_type_schema.sql`）
2. ✅ SDK 模型定义（`model/shift_type.go`）
3. ✅ Repository 层（`repository/shift_type_repository.go`）
4. ✅ Service 层（`service/shift_type_service.go`）
5. ✅ API Handler（`handler/shift_type_handler.go`）
6. 🔄 数据迁移脚本执行
7. 🔄 工作流代码更新（使用动态类型）
8. 🔄 前端集成（调用类型列表API）

详细迁移指南请参考：`SHIFT_TYPE_MIGRATION_GUIDE.md`