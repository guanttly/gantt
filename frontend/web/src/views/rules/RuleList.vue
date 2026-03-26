<script setup lang="ts">
import type { Rule, RuleCategory, RuleType } from '@/types/rule'
import { Delete, Edit, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { reactive, ref } from 'vue'
import { createRule, deleteRule, listRules, updateRule } from '@/api/rules'
import { usePagination } from '@/composables/usePagination'
import { RULE_CATEGORY_OPTIONS, RULE_TYPE_OPTIONS } from '@/types/rule'

const { loading, items, total, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Rule>({
  fetchFn: listRules,
})

const categoryMap: Record<string, { label: string, type: string }> = {
  constraint: { label: '约束', type: 'danger' },
  preference: { label: '偏好', type: 'info' },
  dependency: { label: '依赖', type: 'warning' },
  hard: { label: '硬约束', type: 'danger' },
  soft: { label: '软约束', type: 'warning' },
}

// ======== 表单弹窗 ========

const dialogVisible = ref(false)
const dialogTitle = ref('新增规则')
const formLoading = ref(false)
const editingId = ref<string | null>(null)

const ruleTypes = RULE_TYPE_OPTIONS
const categoryOptions = RULE_CATEGORY_OPTIONS

const form = reactive({
  name: '',
  type: ruleTypes[0].value as RuleType,
  category: categoryOptions[0].value as RuleCategory,
  priority: 100,
  enabled: true,
  config: '{}',
  description: '',
})

const rules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择规则类型', trigger: 'change' }],
}

const formRef = ref()

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增规则'
  Object.assign(form, { name: '', type: ruleTypes[0].value, category: categoryOptions[0].value, priority: 100, enabled: true, config: '{}', description: '' })
  dialogVisible.value = true
}

function handleEdit(row: Rule) {
  editingId.value = row.id
  dialogTitle.value = '编辑规则'
  Object.assign(form, {
    name: row.name,
    type: row.type,
    category: row.category,
    priority: row.priority,
    enabled: row.enabled,
    config: JSON.stringify(row.config, null, 2),
    description: row.description || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }

  let config: Record<string, unknown>
  try {
    config = JSON.parse(form.config)
  }
  catch {
    ElMessage.error('配置 JSON 格式不正确')
    return
  }

  formLoading.value = true
  try {
    const payload = { ...form, config } as any
    if (editingId.value) {
      await updateRule(editingId.value, payload)
      ElMessage.success('更新成功')
    }
    else {
      await createRule(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  }
  finally {
    formLoading.value = false
  }
}

async function handleDelete(row: Rule) {
  await ElMessageBox.confirm(`确定删除规则「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteRule(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索规则"
        clearable
        style="width: 240px"
        :prefix-icon="Search"
      />
      <el-button type="primary" :icon="Plus" @click="handleAdd">
        新增规则
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border stripe style="width: 100%">
      <el-table-column prop="name" label="名称" width="200" />
      <el-table-column prop="type" label="类型" width="160">
        <template #default="{ row }">
          {{ ruleTypes.find(t => t.value === row.type)?.label || row.type }}
        </template>
      </el-table-column>
      <el-table-column prop="category" label="约束等级" width="100">
        <template #default="{ row }">
          <el-tag :type="(categoryMap[row.category]?.type as any)" size="small">
            {{ categoryMap[row.category]?.label || row.category }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column prop="enabled" label="启用" width="80">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
            {{ row.enabled ? '是' : '否' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="inherited" label="来源" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.inherited" type="warning" size="small">
            继承: {{ row.source_node_name }}
          </el-tag>
          <el-tag v-else type="" size="small">
            本节点
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" />
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button :icon="Edit" link type="primary" :disabled="row.inherited" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button :icon="Delete" link type="danger" :disabled="row.inherited" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="page-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="currentPageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="如：最大连续工作5天" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="选择规则类型" style="width: 100%">
            <el-option v-for="t in ruleTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="约束等级" prop="category">
          <el-radio-group v-model="form.category">
            <el-radio v-for="category in categoryOptions" :key="category.value" :value="category.value">
              {{ category.label }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="配置(JSON)">
          <el-input v-model="form.config" type="textarea" :rows="4" placeholder="如：{ max_days: 5 }" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="formLoading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow: hidden;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
