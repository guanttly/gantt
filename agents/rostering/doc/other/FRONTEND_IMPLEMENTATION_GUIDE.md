# 前端固定人员配置实施指南

## 概述

本文档说明如何在班次表单中添加固定人员配置功能。

## 已完成的工作

1. ✅ TypeScript类型定义（`/frontend/web/src/api/shift/model.d.ts`）
2. ✅ API函数（`/frontend/web/src/api/shift/index.ts`）
3. ✅ 后端完整实现（数据库、Repository、Service、API Handler）

## 待实施的前端工作

### 1. 修改 ShiftForm.vue

文件位置：`frontend/web/src/pages/management/shift/components/ShiftForm.vue`

#### 1.1 添加表单数据字段

```vue
<script setup lang="ts">
import { getFixedAssignments, batchCreateFixedAssignments } from '@/api/shift'

// 表单数据
const formData = reactive<ShiftFormData>({
  // ... 现有字段
  fixedAssignments: [] as Shift.FixedAssignment[]
})

// 添加固定人员
function addFixedAssignment() {
  formData.fixedAssignments.push({
    staffId: '',
    patternType: 'weekly',
    weekdays: [],
    specificDates: [],
    startDate: undefined,
    endDate: undefined
  })
}

// 移除固定人员
function removeFixedAssignment(index: number) {
  formData.fixedAssignments.splice(index, 1)
}

// 加载固定人员配置（编辑模式）
async function loadFixedAssignments(shiftId: string) {
  try {
    const assignments = await getFixedAssignments(shiftId)
    formData.fixedAssignments = assignments || []
  } catch (error) {
    console.error('加载固定人员配置失败', error)
  }
}

// 提交时保存固定人员配置
async function handleSubmit() {
  // 1. 先创建/更新班次
  let shiftId: string
  if (props.mode === 'edit') {
    await updateShift(props.shift!.id, formData)
    shiftId = props.shift!.id
  } else {
    const result = await createShift(formData)
    shiftId = result.id
  }
  
  // 2. 保存固定人员配置
  if (formData.fixedAssignments.length > 0) {
    await batchCreateFixedAssignments({
      shiftId,
      assignments: formData.fixedAssignments
    })
  }
  
  emit('success')
}
</script>
```

#### 1.2 添加UI组件

在班次基本信息表单之后添加固定人员配置区域：

```vue
<template>
  <el-dialog>
    <el-form>
      <!-- 现有的基本信息字段 -->
      
      <!-- 固定人员配置区域 -->
      <el-divider content-position="left">
        <span style="font-size: 14px; color: #606266;">
          固定人员配置（可选）
        </span>
      </el-divider>
      
      <el-form-item>
        <template #label>
          <span>固定人员</span>
          <el-tooltip
            content="配置后，排班时会自动为这些人员安排班次"
            placement="top"
          >
            <el-icon style="margin-left: 4px; color: #909399;">
              <QuestionFilled />
            </el-icon>
          </el-tooltip>
        </template>
        
        <div class="fixed-assignments-container">
          <!-- 固定人员列表 -->
          <el-card
            v-for="(item, index) in formData.fixedAssignments"
            :key="index"
            class="assignment-card"
            shadow="hover"
          >
            <template #header>
              <div class="card-header">
                <span>固定人员 {{ index + 1 }}</span>
                <el-button
                  type="danger"
                  size="small"
                  text
                  @click="removeFixedAssignment(index)"
                >
                  删除
                </el-button>
              </div>
            </template>
            
            <!-- 人员选择 -->
            <el-form-item label="人员">
              <el-select
                v-model="item.staffId"
                placeholder="选择人员"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="staff in staffList"
                  :key="staff.id"
                  :label="staff.name"
                  :value="staff.id"
                />
              </el-select>
            </el-form-item>
            
            <!-- 模式选择 -->
            <el-form-item label="配置模式">
              <el-radio-group v-model="item.patternType">
                <el-radio label="weekly">按周重复</el-radio>
                <el-radio label="specific">指定日期</el-radio>
              </el-radio-group>
            </el-form-item>
            
            <!-- 按周重复配置 -->
            <el-form-item v-if="item.patternType === 'weekly'" label="周几">
              <el-checkbox-group v-model="item.weekdays">
                <el-checkbox :label="1">周一</el-checkbox>
                <el-checkbox :label="2">周二</el-checkbox>
                <el-checkbox :label="3">周三</el-checkbox>
                <el-checkbox :label="4">周四</el-checkbox>
                <el-checkbox :label="5">周五</el-checkbox>
                <el-checkbox :label="6">周六</el-checkbox>
                <el-checkbox :label="0">周日</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <!-- 指定日期配置 -->
            <el-form-item v-if="item.patternType === 'specific'" label="日期">
              <el-date-picker
                v-model="item.specificDates"
                type="dates"
                placeholder="选择多个日期"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
            
            <!-- 生效期间 -->
            <el-form-item label="生效期间">
              <el-date-picker
                v-model="[item.startDate, item.endDate]"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期（可选）"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
          </el-card>
          
          <!-- 添加按钮 -->
          <el-button
            type="primary"
            plain
            @click="addFixedAssignment"
            icon="Plus"
            style="width: 100%; margin-top: 12px;"
          >
            添加固定人员
          </el-button>
        </div>
      </el-form-item>
    </el-form>
  </el-dialog>
</template>

<style scoped>
.fixed-assignments-container {
  width: 100%;
}

.assignment-card {
  margin-bottom: 12px;
}

.assignment-card:last-of-type {
  margin-bottom: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

### 2. 获取人员列表

需要在ShiftForm组件中获取可用人员列表：

```vue
<script setup lang="ts">
import { getEmployeeList } from '@/api/employee'

// 人员列表
const staffList = ref<Employee.EmployeeInfo[]>([])

// 加载人员列表
async function loadStaffList() {
  try {
    const result = await getEmployeeList({
      orgId: props.orgId,
      isActive: true,
      page: 1,
      size: 1000 // 获取所有人员
    })
    staffList.value = result.items
  } catch (error) {
    console.error('加载人员列表失败', error)
  }
}

// 在对话框打开时加载
watch(() => props.visible, (val) => {
  if (val) {
    loadStaffList()
    if (props.mode === 'edit' && props.shift) {
      // 加载固定人员配置
      loadFixedAssignments(props.shift.id)
    }
  }
})
</script>
```

### 3. 班次类型管理页面（可选）

文件结构：

```
frontend/web/src/pages/management/shift-type/
├── index.vue               # 列表页
├── components/
│   ├── ShiftTypeForm.vue   # 创建/编辑表单
│   └── ShiftTypeTable.vue  # 表格组件
├── type.d.ts               # 类型定义
└── logic.ts                # 业务逻辑
```

由于这是可选功能，且班次类型主要在后端使用，前端暂时可以不实现完整的管理页面。

## 测试要点

1. **创建班次时添加固定人员**
   - 测试按周重复模式
   - 测试指定日期模式
   - 测试生效时间范围

2. **编辑班次时修改固定人员**
   - 验证数据回显正确
   - 验证修改后保存成功

3. **删除固定人员配置**
   - 验证删除操作正常

4. **排班工作流验证**
   - 创建排班时，验证固定人员自动填充
   - 验证占位信息正确传递

## 注意事项

1. **数据验证**：确保至少选择了一个人员，且配置了有效的模式
2. **用户体验**：添加适当的提示和帮助文本
3. **错误处理**：妥善处理API调用失败的情况
4. **性能优化**：人员列表较多时考虑分页或搜索

## 完整实施估计

- ShiftForm.vue修改：3-4小时
- 测试和调试：2-3小时
- 班次类型管理页面（可选）：6-8小时

**总计：5-7小时（不包括班次类型管理页面）**

