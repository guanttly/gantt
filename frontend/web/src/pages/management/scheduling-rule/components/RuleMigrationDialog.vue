<template>
  <el-dialog
    v-model="dialogVisible"
    title="V3 规则迁移"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-loading="loading">
      <!-- 预览迁移 -->
      <div v-if="step === 'preview'">
        <el-alert
          type="info"
          :closable="false"
          style="margin-bottom: 20px"
        >
          <template #default>
            <div>
              <p>迁移预览：共发现 <strong>{{ preview?.totalV3Rules || 0 }}</strong> 个 V3 规则</p>
              <p>可迁移：<strong>{{ preview?.migratableRules?.length || 0 }}</strong> 个</p>
              <p>无法迁移：<strong>{{ preview?.unmigratableRules?.length || 0 }}</strong> 个</p>
            </div>
          </template>
        </el-alert>

        <!-- 可迁移规则列表 -->
        <div v-if="preview?.migratableRules?.length">
          <h3>可迁移规则</h3>
          <el-table
            :data="preview.migratableRules"
            style="margin-bottom: 20px"
          >
            <el-table-column
              type="selection"
              width="55"
            />
            <el-table-column
              prop="ruleName"
              label="规则名称"
            />
            <el-table-column
              prop="ruleType"
              label="规则类型"
            />
            <el-table-column
              prop="category"
              label="分类"
            >
              <template #default="{ row }">
                <el-tag>{{ row.category }}/{{ row.subCategory }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column
              prop="suggestions"
              label="建议"
            >
              <template #default="{ row }">
                <el-tooltip
                  v-if="row.suggestions?.length"
                  :content="row.suggestions.join('; ')"
                >
                  <el-icon><Warning /></el-icon>
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 无法迁移规则列表 -->
        <div v-if="preview?.unmigratableRules?.length">
          <h3>无法迁移规则</h3>
          <el-table
            :data="preview.unmigratableRules"
            style="margin-bottom: 20px"
          >
            <el-table-column
              prop="ruleName"
              label="规则名称"
            />
            <el-table-column
              prop="reason"
              label="原因"
            />
            <el-table-column
              prop="suggestions"
              label="建议"
            >
              <template #default="{ row }">
                <span>{{ row.suggestions?.join('; ') }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 迁移选项 -->
        <el-form
          :model="migrationOptions"
          label-width="120px"
        >
          <el-form-item label="自动分类">
            <el-switch v-model="migrationOptions.autoClassify" />
          </el-form-item>
          <el-form-item label="填充默认值">
            <el-switch v-model="migrationOptions.fillDefaults" />
          </el-form-item>
          <el-form-item label="仅预览">
            <el-switch v-model="migrationOptions.dryRun" />
          </el-form-item>
        </el-form>
      </div>

      <!-- 执行迁移 -->
      <div v-else-if="step === 'executing'">
        <el-progress
          :percentage="progress"
          :status="progress === 100 ? 'success' : undefined"
        />
        <p style="margin-top: 10px; text-align: center">
          {{ progress === 100 ? '迁移完成' : '正在迁移...' }}
        </p>
      </div>

      <!-- 迁移结果 -->
      <div v-else-if="step === 'result'">
        <el-alert
          :type="result?.failedCount === 0 ? 'success' : 'warning'"
          :closable="false"
          style="margin-bottom: 20px"
        >
          <template #default>
            <div>
              <p>迁移完成：成功 <strong>{{ result?.successCount || 0 }}</strong> 个，失败 <strong>{{ result?.failedCount || 0 }}</strong> 个</p>
            </div>
          </template>
        </el-alert>

        <div v-if="result?.failedRules?.length">
          <h3>失败规则</h3>
          <el-table :data="result.failedRules">
            <el-table-column
              prop="ruleName"
              label="规则名称"
            />
            <el-table-column
              prop="error"
              label="错误信息"
            />
          </el-table>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button
        v-if="step === 'preview'"
        type="primary"
        :loading="executing"
        @click="handleExecute"
      >
        执行迁移
      </el-button>
      <el-button
        v-if="step === 'result'"
        type="primary"
        @click="handleClose"
      >
        完成
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { Warning } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'

interface Props {
  visible: boolean
  orgId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val),
})

const loading = ref(false)
const executing = ref(false)
const step = ref<'preview' | 'executing' | 'result'>('preview')
const progress = ref(0)

const preview = ref<any>(null)
const result = ref<any>(null)

const migrationOptions = reactive({
  autoClassify: true,
  fillDefaults: true,
  dryRun: false,
})

// 当对话框打开时，加载预览
watch(() => props.visible, (val) => {
  if (val) {
    loadPreview()
  }
})

async function loadPreview() {
  loading.value = true
  step.value = 'preview'
  try {
    // TODO: 调用预览迁移 API
    // const res = await previewMigration({ orgId: props.orgId })
    // preview.value = res
    ElMessage.warning('预览迁移 API 尚未实现')
  }
  catch (error: any) {
    ElMessage.error(`加载预览失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
}

async function handleExecute() {
  if (!preview.value?.migratableRules?.length) {
    ElMessage.warning('没有可迁移的规则')
    return
  }

  executing.value = true
  step.value = 'executing'
  progress.value = 0

  try {
    const ruleIDs = preview.value.migratableRules.map((r: any) => r.ruleId)
    // TODO: 调用执行迁移 API
    // const res = await executeMigration({
    //   orgId: props.orgId,
    //   ruleIds: ruleIDs,
    //   ...migrationOptions,
    // })
    // result.value = res
    // progress.value = 100

    // 模拟进度
    const interval = setInterval(() => {
      progress.value += 10
      if (progress.value >= 100) {
        clearInterval(interval)
        step.value = 'result'
        executing.value = false
        ElMessage.success('迁移完成')
        emit('success')
      }
    }, 200)
  }
  catch (error: any) {
    ElMessage.error(`迁移失败: ${error.message}`)
    executing.value = false
    step.value = 'preview'
  }
}

function handleClose() {
  dialogVisible.value = false
  step.value = 'preview'
  progress.value = 0
  preview.value = null
  result.value = null
}
</script>

<style scoped lang="scss">
h3 {
  margin: 20px 0 10px;
  font-size: 16px;
  font-weight: 600;
}
</style>
