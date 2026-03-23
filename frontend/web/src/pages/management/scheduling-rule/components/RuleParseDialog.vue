<script setup lang="ts">
import { ArrowRight, Warning } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref } from 'vue'
import { parseRule, batchSaveRules } from '@/api/scheduling-rule'
import {
  getCategoryText,
  getCategoryTagType,
  getSubCategoryText,
  getRuleTypeText,
  getApplyScopeText,
  getTimeScopeText,
} from '../logic'

// 冲突类型中文映射
function getConflictTypeText(conflictType: string): string {
  const map: Record<string, string> = {
    exclusive: '互斥',
    duplicate: '名称重复',
    logical: '逻辑冲突',
    resource: '资源冲突',
    time: '时间冲突',
    priority: '优先级冲突',
  }
  return map[conflictType] || conflictType
}

// 依赖类型中文映射
function getDependencyTypeText(depType: string): string {
  const map: Record<string, string> = {
    requires: '依赖',
    extends: '扩展',
    overrides: '覆盖',
    conflicts: '冲突',
    time: '时间依赖',
    source: '来源依赖',
    resource: '资源依赖',
    order: '顺序依赖',
  }
  return map[depType] || depType
}

// V4.1 适用范围类型中文映射
function getScopeTypeText(scopeType: string): string {
  const map: Record<string, string> = {
    all: '全局',
    employee: '指定员工',
    group: '指定分组',
    exclude_employee: '排除员工',
    exclude_group: '排除分组',
  }
  return map[scopeType] || scopeType
}

interface Props {
  visible: boolean
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const loading = ref(false)
const parsing = ref(false)

// 步骤管理
const currentStep = ref(0) // 0: 输入, 1: 解析结果, 2: 审核确认

// 表单数据
const formData = reactive({
  name: '',
  ruleDescription: '',
  applyScope: '',
  priority: 5,
  validFrom: '',
  validTo: '',
})

// 解析结果
const parseResult = ref<SchedulingRule.ParseRuleResponse | null>(null)

// 选中的规则（用于多规则勾选）
const selectedRuleIndices = ref<number[]>([])

// 是否显示解析结果
const showParseResult = computed(() => currentStep.value >= 1)

// 选中的规则列表
const selectedRules = computed(() => {
  if (!parseResult.value) return []
  return parseResult.value.parsedRules.filter((_, index) => selectedRuleIndices.value.includes(index))
})

// 全选/取消全选
const allSelected = computed({
  get: () => parseResult.value && selectedRuleIndices.value.length === parseResult.value.parsedRules.length,
  set: (val: boolean) => {
    if (!parseResult.value) return
    if (val) {
      selectedRuleIndices.value = parseResult.value.parsedRules.map((_, index) => index)
    } else {
      selectedRuleIndices.value = []
    }
  },
})

// 解析规则
async function handleParse() {
  if (!formData.name.trim()) {
    ElMessage.warning('请输入规则名称')
    return
  }
  if (!formData.ruleDescription.trim()) {
    ElMessage.warning('请输入规则描述')
    return
  }

  parsing.value = true
  try {
    const result = await parseRule({
      orgId: props.orgId,
      name: formData.name,
      ruleDescription: formData.ruleDescription,
      applyScope: formData.applyScope || undefined,
      priority: formData.priority,
      validFrom: formData.validFrom || undefined,
      validTo: formData.validTo || undefined,
    })
    parseResult.value = result
    // 默认全选所有规则
    selectedRuleIndices.value = result.parsedRules.map((_, index) => index)
    currentStep.value = 1 // 跳转到解析结果步骤
    ElMessage.success('解析成功')
  }
  catch (error: any) {
    // 根据错误类型给出友好提示
    if (error.message?.includes('超时') || error.code === 'ECONNABORTED') {
      ElMessage.error('AI 解析超时，请简化规则描述后重试')
    } else if (error.message?.includes('JSON')) {
      ElMessage.error('AI 返回格式异常，请重新尝试')
    } else {
      ElMessage.error(error.message || '解析失败，请稍后重试')
    }
  }
  finally {
    parsing.value = false
  }
}

// 保存解析结果（只保存选中的规则）
async function handleSave() {
  if (!parseResult.value || selectedRules.value.length === 0) {
    ElMessage.warning('请至少选择一条规则')
    return
  }

  loading.value = true
  try {
    // 只保存选中的规则
    await batchSaveRules({
      orgId: props.orgId,
      parsedRules: selectedRules.value,
      dependencies: parseResult.value.dependencies,
      conflicts: parseResult.value.conflicts,
    })
    ElMessage.success(`成功保存 ${selectedRules.value.length} 条规则`)
    emit('success')
    handleClose()
  }
  catch {
    ElMessage.error('保存失败')
  }
  finally {
    loading.value = false
  }
}

// 下一步
function handleNext() {
  if (currentStep.value === 0) {
    handleParse()
  } else if (currentStep.value === 1) {
    if (selectedRules.value.length === 0) {
      ElMessage.warning('请至少选择一条规则')
      return
    }
    currentStep.value = 2
  }
}

// 上一步
function handlePrev() {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
  // 重置表单
  formData.name = ''
  formData.ruleDescription = ''
  formData.applyScope = ''
  formData.priority = 5
  formData.validFrom = ''
  formData.validTo = ''
  parseResult.value = null
  selectedRuleIndices.value = []
  currentStep.value = 0
}

// 获取置信度颜色
function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return '#67c23a'
  if (confidence >= 0.6) return '#e6a23c'
  return '#f56c6c'
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="语义化规则解析（V4）"
    width="1000px"
    top="5vh"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 步骤条 -->
    <el-steps :active="currentStep" align-center style="margin-bottom: 24px;">
      <el-step title="输入规则" description="输入语义化规则描述" />
      <el-step title="解析结果" description="查看解析后的规则" />
      <el-step title="审核确认" description="确认并保存规则" />
    </el-steps>

    <div v-if="currentStep === 0" class="parse-form">
      <el-form label-width="120px">
        <el-form-item label="规则名称" required>
          <el-input
            v-model="formData.name"
            placeholder="请输入规则名称"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="规则描述" required>
          <el-input
            v-model="formData.ruleDescription"
            type="textarea"
            :rows="6"
            placeholder="请输入语义化规则描述，例如：&#10;• 张三每周最多工作5天&#10;• 夜班后至少休息1天&#10;• 优先安排周末休息"
            maxlength="1000"
            show-word-limit
          />
          <div class="form-tip">
            <div style="margin-bottom: 4px; font-weight: 600;">示例：</div>
            <div class="example-item">• 张三每周最多工作5天</div>
            <div class="example-item">• 夜班后至少休息1天</div>
            <div class="example-item">• 优先安排周末休息</div>
            <div class="example-item">• 禁止连续上夜班</div>
          </div>
        </el-form-item>

        <el-form-item label="应用范围">
          <el-select
            v-model="formData.applyScope"
            placeholder="可选，系统可自动识别"
            clearable
            style="width: 100%"
          >
            <el-option label="全局" value="global" />
            <el-option label="员工" value="employee" />
            <el-option label="班次" value="shift" />
            <el-option label="分组" value="group" />
          </el-select>
        </el-form-item>

        <el-form-item label="优先级">
          <el-input-number
            v-model="formData.priority"
            :min="1"
            :max="10"
            :step="1"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="有效期">
          <el-date-picker
            v-model="formData.validFrom"
            type="date"
            placeholder="开始日期（可选）"
            value-format="YYYY-MM-DD"
            style="width: 48%"
          />
          <span style="margin: 0 8px;">至</span>
          <el-date-picker
            v-model="formData.validTo"
            type="date"
            placeholder="结束日期（可选）"
            value-format="YYYY-MM-DD"
            style="width: 48%"
          />
        </el-form-item>
      </el-form>
    </div>

    <!-- 解析结果步骤 -->
    <div v-if="currentStep === 1" class="parse-result">
      <el-alert
        type="success"
        :closable="false"
        style="margin-bottom: 16px;"
      >
        <template #title>
          <div style="display: flex; align-items: center; gap: 8px;">
            <span>解析成功！共识别出 {{ parseResult?.parsedRules.length || 0 }} 条规则</span>
            <el-tag v-if="selectedRules.length > 0" type="primary" effect="plain">
              已选择 {{ selectedRules.length }} 条
            </el-tag>
          </div>
        </template>
      </el-alert>

      <!-- 解析说明 -->
      <el-card v-if="parseResult?.reasoning" shadow="never" style="margin-bottom: 16px;">
        <template #header>
          <span style="font-weight: 600;">解析说明</span>
        </template>
        <div style="white-space: pre-wrap; line-height: 1.6;">
          {{ parseResult.reasoning }}
        </div>
      </el-card>

      <!-- 解析后的规则列表（带勾选） -->
      <el-card shadow="never" style="margin-bottom: 16px;">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center;">
            <span style="font-weight: 600;">解析后的规则（{{ parseResult?.parsedRules.length || 0 }} 条）</span>
            <el-checkbox v-model="allSelected" style="margin-left: auto;">
              全选
            </el-checkbox>
          </div>
        </template>
        <div class="parsed-rules-list">
          <div
            v-for="(rule, index) in parseResult?.parsedRules"
            :key="index"
            class="parsed-rule-item"
            :class="{ 'rule-selected': selectedRuleIndices.includes(index) }"
          >
            <el-checkbox
              :model-value="selectedRuleIndices.includes(index)"
              @change="(val: boolean) => {
                if (val) {
                  if (!selectedRuleIndices.includes(index)) {
                    selectedRuleIndices.push(index)
                  }
                } else {
                  selectedRuleIndices = selectedRuleIndices.filter(i => i !== index)
                }
              }"
              style="flex-shrink: 0;"
            />
            <div class="rule-body">
              <div class="rule-header">
                <div class="rule-tags">
                  <el-tag :type="getCategoryTagType(rule.category)" effect="dark" size="small">
                    {{ getCategoryText(rule.category) }}
                  </el-tag>
                  <el-tag type="info" effect="plain" size="small">
                    {{ getSubCategoryText(rule.subCategory) }}
                  </el-tag>
                </div>
                <div class="rule-title">{{ rule.name }}</div>
              </div>
              <div class="rule-content">
                <div class="rule-meta">
                  <span><strong>类型：</strong>{{ getRuleTypeText(rule.ruleType) }}</span>
                  <span><strong>范围：</strong>{{ getApplyScopeText(rule.applyScope) }} / {{ getTimeScopeText(rule.timeScope) }}</span>
                </div>
                <div class="rule-desc"><strong>描述：</strong>{{ rule.description }}</div>
                <!-- V4.1 班次关系展示 -->
                <div v-if="rule.subjectShifts?.length || rule.objectShifts?.length || rule.targetShifts?.length" class="rule-shifts">
                  <span v-if="rule.subjectShifts?.length"><strong>主体班次：</strong>{{ rule.subjectShifts.join('、') }}</span>
                  <span v-if="rule.objectShifts?.length"><strong>客体班次：</strong>{{ rule.objectShifts.join('、') }}</span>
                  <span v-if="rule.targetShifts?.length"><strong>目标班次：</strong>{{ rule.targetShifts.join('、') }}</span>
                </div>
                <!-- V4.1 适用范围展示 -->
                <div v-if="rule.scopeType && rule.scopeType !== 'all'" class="rule-scope">
                  <span><strong>适用范围：</strong>{{ getScopeTypeText(rule.scopeType) }}</span>
                  <span v-if="rule.scopeEmployees?.length"> - {{ rule.scopeEmployees.join('、') }}</span>
                  <span v-if="rule.scopeGroups?.length"> - {{ rule.scopeGroups.join('、') }}</span>
                </div>
                <div v-if="rule.maxCount || rule.consecutiveMax || rule.minRestDays" class="rule-params">
                  <span v-if="rule.maxCount">最大次数: {{ rule.maxCount }}</span>
                  <span v-if="rule.consecutiveMax">连续最大: {{ rule.consecutiveMax }}</span>
                  <span v-if="rule.minRestDays">最少休息: {{ rule.minRestDays }}天</span>
                </div>
                <!-- 置信度展示（如果后端返回） -->
                <div v-if="(rule as any).parseConfidence !== undefined" style="margin-top: 8px;">
                  <el-progress
                    :percentage="((rule as any).parseConfidence || 0) * 100"
                    :format="(percentage: number) => `置信度: ${percentage.toFixed(1)}%`"
                    :color="getConfidenceColor((rule as any).parseConfidence || 0)"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-card>

      <!-- 依赖关系 -->
      <el-card
        v-if="parseResult?.dependencies && parseResult.dependencies.length > 0"
        shadow="never"
        style="margin-bottom: 16px;"
      >
        <template #header>
          <span style="font-weight: 600;">依赖关系（{{ parseResult.dependencies.length }} 条）</span>
        </template>
        <div class="dependencies-list">
          <div
            v-for="(dep, index) in parseResult.dependencies"
            :key="index"
            class="dependency-item"
          >
            <div class="dependency-main">
              <el-icon style="color: var(--el-color-primary);"><ArrowRight /></el-icon>
              <span style="font-weight: 600;">{{ dep.dependentRuleName }}</span>
              <span style="margin: 0 8px; color: var(--el-text-color-secondary);">→</span>
              <span>{{ dep.dependentOnRuleName }}</span>
              <el-tag type="info" size="small" effect="plain" style="margin-left: 12px;">
                {{ getDependencyTypeText(dep.dependencyType) }}
              </el-tag>
            </div>
            <div class="dependency-desc">
              {{ dep.description }}
            </div>
          </div>
        </div>
      </el-card>

      <!-- 冲突关系 -->
      <el-card
        v-if="parseResult?.conflicts && parseResult.conflicts.length > 0"
        shadow="never"
        style="margin-bottom: 16px;"
      >
        <template #header>
          <span style="font-weight: 600;">冲突关系（{{ parseResult.conflicts.length }} 条）</span>
        </template>
        <div class="conflicts-list">
          <div
            v-for="(conflict, index) in parseResult.conflicts"
            :key="index"
            class="conflict-item"
          >
            <div class="conflict-main">
              <el-icon style="color: var(--el-color-warning);"><Warning /></el-icon>
              <span style="font-weight: 600;">{{ conflict.ruleName1 }}</span>
              <span style="margin: 0 8px; color: var(--el-text-color-secondary);">↔</span>
              <span style="font-weight: 600;">{{ conflict.ruleName2 }}</span>
              <el-tag type="warning" size="small" effect="plain" style="margin-left: 12px;">
                {{ getConflictTypeText(conflict.conflictType) }}
              </el-tag>
            </div>
            <div class="conflict-desc">
              {{ conflict.description }}
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 审核确认步骤 -->
    <div v-if="currentStep === 2" class="review-step">
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 16px;"
      >
        <template #title>
          <div style="display: flex; align-items: center; gap: 8px;">
            <span>请确认以下 {{ selectedRules.length }} 条规则，确认后将保存到系统</span>
          </div>
        </template>
      </el-alert>

      <!-- 回译对比（如果后端返回） -->
      <el-card v-if="parseResult?.originalRule" shadow="never" style="margin-bottom: 16px;">
        <template #header>
          <span style="font-weight: 600;">回译对比</span>
        </template>
        <div class="back-translation">
          <div class="translation-item">
            <div class="translation-label">原始输入：</div>
            <div class="translation-content">{{ parseResult.originalRule }}</div>
          </div>
          <!-- 如果后端返回回译结果，显示对比 -->
          <div v-if="(parseResult as any).backTranslation" class="translation-item" style="margin-top: 12px;">
            <div class="translation-label">回译结果：</div>
            <div class="translation-content">{{ (parseResult as any).backTranslation }}</div>
          </div>
        </div>
      </el-card>

      <!-- 选中的规则列表 -->
      <el-card shadow="never">
        <template #header>
          <span style="font-weight: 600;">待保存的规则（{{ selectedRules.length }} 条）</span>
        </template>
        <div class="selected-rules-list">
          <div
            v-for="(rule, index) in selectedRules"
            :key="index"
            class="selected-rule-item"
          >
            <div class="rule-header">
              <div class="rule-tags">
                <el-tag :type="getCategoryTagType(rule.category)" effect="dark" size="small">
                  {{ getCategoryText(rule.category) }}
                </el-tag>
              </div>
              <div class="rule-title">{{ rule.name }}</div>
            </div>
            <div class="rule-content">
              <div class="rule-meta">
                <span><strong>类型：</strong>{{ getRuleTypeText(rule.ruleType) }}</span>
              </div>
              <div class="rule-desc"><strong>描述：</strong>{{ rule.description }}</div>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button v-if="currentStep > 0" @click="handlePrev">
        上一步
      </el-button>
      <el-button
        v-if="currentStep === 0"
        type="primary"
        :loading="parsing"
        @click="handleParse"
      >
        解析规则
      </el-button>
      <el-button
        v-else-if="currentStep === 1"
        type="primary"
        :disabled="selectedRules.length === 0"
        @click="handleNext"
      >
        下一步（已选 {{ selectedRules.length }} 条）
      </el-button>
      <el-button
        v-else-if="currentStep === 2"
        type="primary"
        :loading="loading"
        @click="handleSave"
      >
        确认保存
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.parse-form {
  max-height: calc(85vh - 200px);
  overflow-y: auto;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.5;
}

.example-item {
  color: var(--el-color-info);
  margin-left: 8px;
  line-height: 1.8;
}

.parse-result {
  max-height: calc(85vh - 200px);
  overflow-y: auto;
}

.parsed-rules-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.parsed-rule-item {
  padding: 12px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color-page);
  display: flex;
  align-items: flex-start;
  gap: 12px;
  transition: all 0.3s;

  &.rule-selected {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

.rule-body {
  flex: 1;
  min-width: 0;
}

.rule-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.rule-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.rule-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.rule-content {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

.rule-meta {
  display: flex;
  gap: 24px;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
}

.rule-desc {
  margin-bottom: 6px;
}

.rule-shifts,
.rule-scope {
  display: flex;
  gap: 16px;
  margin-bottom: 6px;
  padding: 6px 10px;
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
  font-size: 12px;
}

.rule-params {
  display: flex;
  gap: 16px;
  margin-top: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.dependencies-list,
.conflicts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dependency-item,
.conflict-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color-page);
}

.dependency-main,
.conflict-main {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.dependency-desc,
.conflict-desc {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  padding-left: 20px;
}

.review-step {
  max-height: calc(85vh - 200px);
  overflow-y: auto;
}

.back-translation {
  .translation-item {
    .translation-label {
      font-weight: 600;
      margin-bottom: 8px;
      color: var(--el-text-color-primary);
    }

    .translation-content {
      padding: 12px;
      background: var(--el-bg-color-page);
      border: 1px solid var(--el-border-color-light);
      border-radius: 4px;
      line-height: 1.6;
      white-space: pre-wrap;
    }
  }
}

.selected-rules-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.selected-rule-item {
  padding: 12px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color-page);
}
</style>
