<script setup lang="ts">
import { InfoFilled, MagicStick, Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { calculateStaffing, getShiftWeeklyStaff, getStaffingRules, updateShiftWeeklyStaff } from '@/api/staffing'
import {
  createDefaultWeeklyConfig,
  getWeekdayDisplayName,
  getWeekdayName,
  getWeekends,
  getWorkdays,
  getWeeklyConfigInDisplayOrder,
  isWeekend,
  WEEKDAY_DISPLAY_ORDER,
  WEEKDAY_NAMES,
} from '../logic'

interface Props {
  visible: boolean
  orgId: string
  shiftIds: string[] // 支持单个或批量班次
  shiftNames?: string[] // 班次名称（用于显示）
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const saving = ref(false)
const calculating = ref(false)
const hasRule = ref(false) // 是否有计算规则

// 周人数配置（7项，索引对应 weekday 值：0=周日, 1=周一, ..., 6=周六）
const weeklyConfig = reactive<Staffing.DayConfig[]>(createDefaultWeeklyConfig())

// 按展示顺序排序的周配置（从周一开始，用于 UI 显示）
const weeklyConfigDisplay = computed(() => getWeeklyConfigInDisplayOrder(weeklyConfig))

// 统一设置值
const uniformValue = ref(0)

// 对话框标题
const dialogTitle = computed(() => {
  if (props.shiftIds.length === 1) {
    return props.shiftNames?.[0] ? `配置人数 - ${props.shiftNames[0]}` : '配置人数'
  }
  return `批量配置人数（已选 ${props.shiftIds.length} 个班次）`
})

// 是否为批量模式
const isBatchMode = computed(() => props.shiftIds.length > 1)

// 监听 visible 变化，加载数据
watch(() => props.visible, async (val) => {
  if (val && props.shiftIds.length > 0) {
    await loadData()
  }
})

// 加载数据
async function loadData() {
  loading.value = true
  hasRule.value = false

  try {
    // 单班次模式：加载现有配置
    if (!isBatchMode.value && props.shiftIds[0]) {
      const shiftId = props.shiftIds[0]

      // 获取周人数配置
      try {
        const config = await getShiftWeeklyStaff(shiftId, props.orgId)
        if (config?.weeklyConfig) {
          // 用返回的配置更新
          config.weeklyConfig.forEach((item) => {
            if (item.weekday >= 0 && item.weekday < 7) {
              weeklyConfig[item.weekday].staffCount = item.staffCount
              weeklyConfig[item.weekday].isCustom = item.isCustom
            }
          })
        }
      }
      catch {
        // 如果获取失败，使用默认配置
        resetConfig()
      }

      // 检查是否有计算规则
      try {
        const rules = await getStaffingRules({ orgId: props.orgId, shiftId })
        hasRule.value = (rules?.items?.length ?? 0) > 0
      }
      catch {
        hasRule.value = false
      }
    }
    else {
      // 批量模式：使用默认配置
      resetConfig()
    }
  }
  catch (error) {
    console.error('Failed to load data:', error)
    ElMessage.error('加载数据失败')
  }
  finally {
    loading.value = false
  }
}

// 重置为默认配置
function resetConfig() {
  weeklyConfig.forEach((item, index) => {
    item.staffCount = 0
    item.isCustom = false
    item.weekdayName = WEEKDAY_NAMES[index]
  })
  uniformValue.value = 0
}

// 统一设置所有日期
function applyUniformValue() {
  if (uniformValue.value < 0) {
    ElMessage.warning('人数不能为负数')
    return
  }
  weeklyConfig.forEach((item) => {
    item.staffCount = uniformValue.value
  })
}

// 仅设置工作日
function applyWorkdays() {
  if (uniformValue.value < 0) {
    ElMessage.warning('人数不能为负数')
    return
  }
  getWorkdays().forEach((weekday) => {
    weeklyConfig[weekday].staffCount = uniformValue.value
  })
}

// 仅设置周末
function applyWeekends() {
  if (uniformValue.value < 0) {
    ElMessage.warning('人数不能为负数')
    return
  }
  getWeekends().forEach((weekday) => {
    weeklyConfig[weekday].staffCount = uniformValue.value
  })
}

// 根据规则计算（仅单班次可用）
async function handleCalculate() {
  if (isBatchMode.value) {
    ElMessage.warning('批量模式不支持自动计算')
    return
  }

  if (!hasRule.value) {
    ElMessage.warning('该班次尚未配置计算规则，请先在人数计算页面配置')
    return
  }

  calculating.value = true
  try {
    const result = await calculateStaffing(props.orgId, props.shiftIds[0])
    if (result?.dailyResults) {
      // 用计算结果填充配置
      result.dailyResults.forEach((item) => {
        if (item.weekday >= 0 && item.weekday < 7) {
          weeklyConfig[item.weekday].staffCount = item.calculatedCount
        }
      })
      ElMessage.success('已填充计算推荐值')
    }
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
  finally {
    calculating.value = false
  }
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 保存配置
async function handleSave() {
  // 验证
  const hasAnyConfig = weeklyConfig.some(item => item.staffCount > 0)
  if (!hasAnyConfig) {
    try {
      await ElMessageBox.confirm(
        '所有日期的人数都为 0，确定要保存吗？',
        '提示',
        { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
      )
    }
    catch {
      return
    }
  }

  saving.value = true
  try {
    // 构建保存数据
    const saveData: Staffing.WeeklyStaff = {
      shiftId: '', // 会在循环中设置
      weeklyConfig: weeklyConfig.map(item => ({
        weekday: item.weekday,
        staffCount: item.staffCount,
        isCustom: item.staffCount > 0,
      })),
    }

    // 循环保存所有班次
    for (const shiftId of props.shiftIds) {
      saveData.shiftId = shiftId
      await updateShiftWeeklyStaff(shiftId, props.orgId, saveData)
    }

    ElMessage.success(isBatchMode.value ? `已保存 ${props.shiftIds.length} 个班次的人数配置` : '保存成功')
    emit('success')
    handleClose()
  }
  catch {
    // 错误已由 request 拦截器统一处理
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="1000px"
    @close="handleClose"
  >
    <div v-loading="loading" class="weekly-staff-container">
      <!-- 批量模式提示 -->
      <el-alert
        v-if="isBatchMode"
        title="批量配置模式"
        :description="`将统一配置 ${shiftIds.length} 个班次的周人数，保存后所有选中班次将使用相同的配置。`"
        type="info"
        :closable="false"
        show-icon
        class="batch-alert"
      />

      <!-- 快捷操作区 -->
      <div class="quick-actions">
        <el-input-number
          v-model="uniformValue"
          :min="0"
          :max="99"
          size="default"
          placeholder="人数"
          controls-position="right"
          style="width: 120px"
        />
        <el-button-group>
          <el-button @click="applyUniformValue">
            统一设置
          </el-button>
          <el-button @click="applyWorkdays">
            仅工作日
          </el-button>
          <el-button @click="applyWeekends">
            仅周末
          </el-button>
        </el-button-group>
        <el-button :icon="Refresh" @click="resetConfig">
          重置
        </el-button>
        <el-tooltip
          v-if="!isBatchMode"
          :content="hasRule ? '根据计算规则自动填充推荐人数' : '请先在人数计算页面配置规则'"
          placement="top"
        >
          <el-button
            type="primary"
            :icon="MagicStick"
            :loading="calculating"
            :disabled="!hasRule"
            @click="handleCalculate"
          >
            自动计算
          </el-button>
        </el-tooltip>
      </div>

      <!-- 周人数配置卡片（从周一开始显示） -->
      <div class="weekday-cards">
        <div
          v-for="(item, displayIndex) in weeklyConfigDisplay"
          :key="item.weekday"
          class="weekday-card"
          :class="{ weekend: isWeekend(item.weekday) }"
        >
          <div class="weekday-name">
            {{ getWeekdayDisplayName(displayIndex) }}
          </div>
          <el-input-number
            v-model="item.staffCount"
            :min="0"
            :max="99"
            size="large"
            controls-position="right"
            class="staff-input"
          />
          <div class="unit">
            人
          </div>
        </div>
      </div>

      <!-- 配置说明 -->
      <div class="config-tips">
        <el-icon style="color: #909399; margin-right: 4px">
          <InfoFilled />
        </el-icon>
        <span>工作流排班时将根据日期的星期几读取对应的人数配置</span>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button type="primary" :loading="saving" @click="handleSave">
        保存
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.weekly-staff-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.batch-alert {
  margin-bottom: 8px;
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.weekday-cards {
  display: flex;
  gap: 8px;
  justify-content: space-between;

  .weekday-card {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 16px 8px;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    background-color: #fff;
    transition: all 0.2s;

    &:hover {
      border-color: #409eff;
      box-shadow: 0 2px 12px rgba(64, 158, 255, 0.1);
    }

    &.weekend {
      background-color: #fef0f0;
      border-color: #fde2e2;

      &:hover {
        border-color: #f56c6c;
        box-shadow: 0 2px 12px rgba(245, 108, 108, 0.1);
      }

      .weekday-name {
        color: #f56c6c;
      }
    }

    .weekday-name {
      font-size: 14px;
      font-weight: 500;
      color: #606266;
      margin-bottom: 12px;
    }

    .staff-input {
      width: 80px;

      :deep(.el-input__wrapper) {
        padding: 0 2rem 0 0;
      }

      :deep(.el-input__inner) {
        text-align: center;
        font-size: 18px;
        font-weight: 600;
      }
    }

    .unit {
      margin-top: 8px;
      font-size: 12px;
      color: #909399;
    }
  }
}

.config-tips {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: #909399;
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}
</style>
