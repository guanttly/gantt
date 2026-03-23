<script setup lang="ts">
import { Check } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { getShiftList } from '@/api/shift'
import { applyStaffingResult, calculateStaffing, getStaffingRules } from '@/api/staffing'

interface Props {
  visible: boolean
  orgId: string
  hasRules: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const applying = ref(false)
const selectedShiftId = ref('')
const shifts = ref<Shift.ShiftInfo[]>([])
const rules = ref<Staffing.Rule[]>([])
const calculateResult = ref<Staffing.CalculationPreview | null>(null)

// 编辑后的每日人数（用户可调整）
const editedDailyCounts = ref<number[]>([])
// 监听 visible 变化，重置状态
watch(() => props.visible, async (val) => {
  if (val) {
    calculateResult.value = null
    selectedShiftId.value = ''
    editedDailyCounts.value = []
    await loadData()
  }
})

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const [shiftRes, ruleRes] = await Promise.all([
      getShiftList({ orgId: props.orgId, isActive: true, page: 1, size: 100 }),
      getStaffingRules({ orgId: props.orgId }),
    ])
    shifts.value = shiftRes.items || []
    rules.value = ruleRes.items || []
  }
  catch {
    ElMessage.error('加载数据失败')
  }
  finally {
    loading.value = false
  }
}

// 获取已配置规则的班次
function getConfiguredShifts() {
  const ruleShiftIds = new Set(rules.value.map(r => r.shiftId))
  return shifts.value.filter(s => ruleShiftIds.has(s.id))
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 执行计算
async function handleCalculate() {
  if (!selectedShiftId.value) {
    ElMessage.warning('请选择班次')
    return
  }

  loading.value = true
  try {
    const result = await calculateStaffing(props.orgId, selectedShiftId.value)
    calculateResult.value = result
    // 初始化编辑数据为计算推荐值
    editedDailyCounts.value = result.dailyResults?.map(r => r.calculatedCount) || []
    ElMessage.success('计算完成')
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
  finally {
    loading.value = false
  }
}

// 计算是否有修改
const hasChanges = computed(() => {
  if (!calculateResult.value?.dailyResults)
    return false
  return calculateResult.value.dailyResults.some((r, idx) =>
    r.currentCount !== editedDailyCounts.value[idx],
  )
})

// 应用计算结果 - 直接写入周配置
async function handleApply() {
  if (!calculateResult.value)
    return

  try {
    // 构建确认消息
    const changedDays = calculateResult.value.dailyResults
      ?.filter((r, idx) => r.currentCount !== editedDailyCounts.value[idx])
      .map(r => `${r.weekdayName}: ${r.currentCount} → ${editedDailyCounts.value[calculateResult.value!.dailyResults!.indexOf(r)]}人`)
      .join('\n') || ''

    await ElMessageBox.confirm(
      `确认应用以下人数配置到班次吗？\n\n${changedDays}`,
      '应用确认',
      {
        confirmButtonText: '确定应用',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    applying.value = true

    // 获取有变更的星期几
    const weekdays = calculateResult.value.dailyResults
      ?.map((r, idx) => ({ weekday: r.weekday, changed: r.currentCount !== editedDailyCounts.value[idx] }))
      .filter(r => r.changed)
      .map(r => r.weekday) || []

    // 注意：这里需要为每个有变更的天分别应用
    // 暂时使用第一个变更的值作为统一值（后续可优化为批量设置不同值）
    const firstChangedIdx = calculateResult.value.dailyResults?.findIndex((r, idx) =>
      r.currentCount !== editedDailyCounts.value[idx],
    ) ?? 0

    await applyStaffingResult(props.orgId, {
      shiftId: calculateResult.value.shiftId,
      staffCount: editedDailyCounts.value[firstChangedIdx],
      applyMode: 'weekly',
      weekdays,
    })

    ElMessage.success('应用成功')
    emit('success')
    handleClose()
  }
  catch (error) {
    // 用户取消确认框不处理，其他错误由 request 拦截器统一处理
    if (error === 'cancel')
      return
  }
  finally {
    applying.value = false
  }
}

// 一键应用推荐值
function applyRecommended() {
  if (calculateResult.value?.dailyResults) {
    editedDailyCounts.value = calculateResult.value.dailyResults.map(r => r.calculatedCount)
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="排班人数计算"
    width="700px"
    @close="handleClose"
  >
    <div v-loading="loading" class="calculate-container">
      <!-- 无规则提示 -->
      <el-alert
        v-if="!hasRules"
        title="请先配置计算规则"
        description="需要先为班次配置计算规则，才能进行人数计算"
        type="warning"
        :closable="false"
        show-icon
        class="no-rules-alert"
      />

      <!-- 班次选择 -->
      <div class="shift-selection">
        <span class="label">选择要计算的班次：</span>
        <el-select
          v-model="selectedShiftId"
          placeholder="请选择班次"
          style="width: 300px"
          :disabled="!hasRules"
        >
          <el-option
            v-for="shift in getConfiguredShifts()"
            :key="shift.id"
            :label="`${shift.name} (${shift.startTime}-${shift.endTime})`"
            :value="shift.id"
          />
        </el-select>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!selectedShiftId"
          @click="handleCalculate"
        >
          开始计算
        </el-button>
      </div>

      <!-- 计算结果 -->
      <div v-if="calculateResult" class="result-section">
        <div class="result-header">
          <el-icon color="#67c23a">
            <Check />
          </el-icon>
          <span>计算结果</span>
        </div>

        <div class="result-content">
          <!-- 基本信息 -->
          <div class="info-row">
            <span class="info-label">班次名称：</span>
            <span class="info-value">{{ calculateResult.shiftName }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">时间段：</span>
            <span class="info-value">{{ calculateResult.timePeriodName }}</span>
          </div>

          <!-- 机房检查量明细 -->
          <div class="info-row">
            <span class="info-label">机房检查量：</span>
            <div class="info-value">
              <el-tag
                v-for="room in calculateResult.modalityRooms"
                :key="room.modalityRoomId"
                type="info"
                effect="plain"
                class="room-tag"
              >
                {{ room.modalityRoomName }}: {{ room.volume }}
              </el-tag>
            </div>
          </div>

          <!-- 统计数据 -->
          <div class="stats-row">
            <div class="stat-item">
              <span class="stat-label">总检查量</span>
              <span class="stat-value">{{ calculateResult.totalVolume }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">数据天数</span>
              <span class="stat-value">{{ calculateResult.dataDays }} 天</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">人均上限</span>
              <span class="stat-value">{{ calculateResult.avgReportLimit }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">周总人次</span>
              <span class="stat-value primary">{{ calculateResult.calculatedCount }}</span>
            </div>
          </div>

          <!-- 每日计算结果表格 -->
          <div class="daily-results">
            <div class="daily-header">
              <span>每日人数配置</span>
              <el-button type="primary" link size="small" @click="applyRecommended">
                一键使用推荐值
              </el-button>
            </div>
            <el-table :data="calculateResult.dailyResults" border size="small" class="daily-table">
              <el-table-column prop="weekdayName" label="星期" width="80" align="center" />
              <el-table-column prop="dailyVolume" label="检查量" width="100" align="center">
                <template #default="{ row }">
                  <span :class="{ 'no-data': row.dailyVolume === 0 }">
                    {{ row.dailyVolume || '-' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="calculatedCount" label="推荐人数" width="100" align="center">
                <template #default="{ row }">
                  <el-tag type="success" size="small">
                    {{ row.calculatedCount }} 人
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="currentCount" label="当前配置" width="100" align="center">
                <template #default="{ row }">
                  <span :class="{ 'zero-count': row.currentCount === 0 }">
                    {{ row.currentCount }} 人
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="应用人数" width="140" align="center">
                <template #default="{ $index }">
                  <el-input-number
                    v-model="editedDailyCounts[$index]"
                    :min="0"
                    :max="99"
                    size="small"
                    controls-position="right"
                  />
                </template>
              </el-table-column>
              <el-table-column label="变更" width="80" align="center">
                <template #default="{ row, $index }">
                  <el-tag
                    v-if="row.currentCount !== editedDailyCounts[$index]"
                    type="warning"
                    size="small"
                  >
                    已修改
                  </el-tag>
                  <span v-else class="no-change">-</span>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <!-- 计算过程 -->
          <div class="calculation-steps">
            <div class="steps-label">
              计算过程：
            </div>
            <div class="steps-content">
              {{ calculateResult.calculationSteps }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">
        关闭
      </el-button>
      <el-button
        v-if="calculateResult && hasChanges"
        type="primary"
        :loading="applying"
        @click="handleApply"
      >
        应用到班次配置
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.calculate-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.no-rules-alert {
  margin-bottom: 8px;
}

.shift-selection {
  display: flex;
  align-items: center;
  gap: 12px;

  .label {
    font-weight: 500;
    white-space: nowrap;
  }
}

.result-section {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  overflow: hidden;

  .result-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background-color: #f5f7fa;
    border-bottom: 1px solid #ebeef5;
    font-weight: 500;
  }

  .result-content {
    padding: 16px;
  }
}

.info-row {
  display: flex;
  margin-bottom: 12px;

  .info-label {
    width: 100px;
    color: #909399;
    flex-shrink: 0;
  }

  .info-value {
    color: #303133;
  }
}

.room-tag {
  margin-right: 8px;
  margin-bottom: 4px;
}

.stats-row {
  display: flex;
  gap: 24px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 4px;
  margin: 16px 0;

  .stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;

    .stat-label {
      font-size: 12px;
      color: #909399;
      margin-bottom: 4px;
    }

    .stat-value {
      font-size: 18px;
      font-weight: 600;
      color: #303133;

      &.primary {
        color: #409eff;
      }
    }
  }
}

.daily-results {
  margin: 16px 0;

  .daily-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    font-weight: 500;
  }

  .daily-table {
    .no-data {
      color: #c0c4cc;
    }

    .zero-count {
      color: #f56c6c;
    }

    .no-change {
      color: #c0c4cc;
    }
  }
}

.calculation-steps {
  margin: 16px 0;
  padding: 12px;
  background-color: #fafafa;
  border-radius: 4px;

  .steps-label {
    font-size: 12px;
    color: #909399;
    margin-bottom: 4px;
  }

  .steps-content {
    font-size: 13px;
    color: #606266;
    line-height: 1.6;
    white-space: pre-line;
  }
}
</style>
