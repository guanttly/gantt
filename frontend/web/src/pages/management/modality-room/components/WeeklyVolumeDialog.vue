<script setup lang="ts">
import { CopyDocument, Delete, Plus, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { getWeeklyVolumes, saveWeeklyVolumes } from '@/api/modality-room'
import { getActiveScanTypes } from '@/api/scan-type'
import { getActiveTimePeriods } from '@/api/time-period'

interface Props {
  visible: boolean
  modalityRoom: ModalityRoom.Info | null
  orgId: string
}

// 每个时间段+星期 已添加的检查类型集合
type AddedScanTypes = Record<string, Record<number, Set<string>>> // timePeriodId -> weekday -> Set<scanTypeId>

// 内部数据结构：按时间段 -> 星期 -> 检查类型 组织
interface VolumeMatrix {
  [timePeriodId: string]: {
    [weekday: number]: {
      [scanTypeId: string]: number
    }
  }
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const saving = ref(false)

// 基础数据
const timePeriods = ref<TimePeriod.Info[]>([])
const scanTypes = ref<ScanType.Info[]>([])

// 矩阵数据
const volumeMatrix = ref<VolumeMatrix>({})

// 每个时间段+星期 已添加的检查类型
const addedScanTypes = ref<AddedScanTypes>({})

// 折叠面板展开的时间段
const activeTimePeriods = ref<string[]>([])

// 星期配置（周一开始）
const weekdays = [
  { value: 1, label: '周一', short: '一' },
  { value: 2, label: '周二', short: '二' },
  { value: 3, label: '周三', short: '三' },
  { value: 4, label: '周四', short: '四' },
  { value: 5, label: '周五', short: '五' },
  { value: 6, label: '周六', short: '六', isWeekend: true },
  { value: 0, label: '周日', short: '日', isWeekend: true },
]

// 对话框标题
const dialogTitle = computed(() => {
  return props.modalityRoom ? `${props.modalityRoom.name} - 检查量配置` : '检查量配置'
})

// 按 startTime 排序时间段（考虑跨天）
const sortedTimePeriods = computed(() => {
  return [...timePeriods.value].sort((a, b) => {
    const getMinutes = (time: string, isCrossDay: boolean) => {
      const [h, m] = time.split(':').map(Number)
      let minutes = h * 60 + m
      if (isCrossDay && h < 12) {
        minutes += 24 * 60
      }
      return minutes
    }
    return getMinutes(a.startTime, a.isCrossDay) - getMinutes(b.startTime, b.isCrossDay)
  })
})

// 获取某天已添加的检查类型列表
function getAddedScanTypesForDay(timePeriodId: string, weekday: number): ScanType.Info[] {
  const added = addedScanTypes.value[timePeriodId]?.[weekday]
  if (!added || added.size === 0)
    return []
  return scanTypes.value.filter(st => added.has(st.id))
}

// 获取某天可添加的检查类型列表（未添加的）
function getAvailableScanTypesForDay(timePeriodId: string, weekday: number): ScanType.Info[] {
  const added = addedScanTypes.value[timePeriodId]?.[weekday]
  if (!added)
    return scanTypes.value
  return scanTypes.value.filter(st => !added.has(st.id))
}

// 统计时间段内所有天的检查类型数量（去重）
function getAddedScanTypesCountForPeriod(timePeriodId: string): number {
  const allScanTypeIds = new Set<string>()
  const periodData = addedScanTypes.value[timePeriodId]
  if (periodData) {
    for (const weekday of weekdays) {
      const dayAdded = periodData[weekday.value]
      if (dayAdded) {
        for (const id of dayAdded) {
          allScanTypeIds.add(id)
        }
      }
    }
  }
  return allScanTypeIds.size
}

// 计算每个时间段的配置统计
const timePeriodStats = computed(() => {
  const stats: Record<string, { scanTypeCount: number, configuredCount: number }> = {}

  for (const tp of timePeriods.value) {
    const scanTypeCount = getAddedScanTypesCountForPeriod(tp.id)
    let configuredCount = 0

    const periodData = addedScanTypes.value[tp.id]
    if (periodData) {
      const tpData = volumeMatrix.value[tp.id]
      if (tpData) {
        for (const weekday of weekdays) {
          const dayAdded = periodData[weekday.value]
          const dayData = tpData[weekday.value]
          if (dayAdded && dayData) {
            for (const scanTypeId of dayAdded) {
              if (dayData[scanTypeId] && dayData[scanTypeId] > 0) {
                configuredCount++
              }
            }
          }
        }
      }
    }
    stats[tp.id] = { scanTypeCount, configuredCount }
  }
  return stats
})

// 监听 visible 变化
watch(() => props.visible, async (val) => {
  if (val && props.modalityRoom) {
    await loadData()
  }
})

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const [tpRes, stRes, volRes] = await Promise.all([
      getActiveTimePeriods(props.orgId),
      getActiveScanTypes(props.orgId),
      getWeeklyVolumes(props.modalityRoom!.id, props.orgId),
    ])

    timePeriods.value = tpRes || []
    scanTypes.value = stRes || []

    // 初始化矩阵结构和已添加的检查类型
    const matrix: VolumeMatrix = {}
    const added: AddedScanTypes = {}

    for (const tp of timePeriods.value) {
      matrix[tp.id] = {}
      added[tp.id] = {}
      for (const wd of weekdays) {
        matrix[tp.id][wd.value] = {}
        added[tp.id][wd.value] = new Set()
      }
    }

    // 填充已有数据，并自动标记已添加的检查类型
    for (const item of volRes.items || []) {
      if (matrix[item.timePeriodId]?.[item.weekday]) {
        matrix[item.timePeriodId][item.weekday][item.scanTypeId] = item.volume
        // volume > 0 的检查类型自动标记为已添加
        if (item.volume > 0) {
          added[item.timePeriodId]?.[item.weekday]?.add(item.scanTypeId)
        }
      }
    }

    volumeMatrix.value = matrix
    addedScanTypes.value = added

    // 默认展开第一个有数据的时间段，如果都没数据则展开第一个
    const firstConfigured = sortedTimePeriods.value.find(tp => timePeriodStats.value[tp.id]?.scanTypeCount > 0)
    activeTimePeriods.value = [firstConfigured?.id || sortedTimePeriods.value[0]?.id].filter(Boolean)
  }
  catch {
    ElMessage.error('加载数据失败')
  }
  finally {
    loading.value = false
  }
}

// 获取单元格的值
function getVolume(timePeriodId: string, weekday: number, scanTypeId: string): number {
  return volumeMatrix.value[timePeriodId]?.[weekday]?.[scanTypeId] || 0
}

// 设置单元格的值
function setVolume(timePeriodId: string, weekday: number, scanTypeId: string, value: number) {
  if (!volumeMatrix.value[timePeriodId]) {
    volumeMatrix.value[timePeriodId] = {}
  }
  if (!volumeMatrix.value[timePeriodId][weekday]) {
    volumeMatrix.value[timePeriodId][weekday] = {}
  }
  volumeMatrix.value[timePeriodId][weekday][scanTypeId] = value
}

// 添加检查类型（只添加到指定的那一天）
function handleAddScanType(timePeriodId: string, weekday: number, scanTypeId: string) {
  if (!scanTypeId)
    return

  if (!addedScanTypes.value[timePeriodId]) {
    addedScanTypes.value[timePeriodId] = {}
  }
  if (!addedScanTypes.value[timePeriodId][weekday]) {
    addedScanTypes.value[timePeriodId][weekday] = new Set()
  }
  addedScanTypes.value[timePeriodId][weekday].add(scanTypeId)

  // 初始化该检查类型在该天的数据为0
  if (!volumeMatrix.value[timePeriodId][weekday]) {
    volumeMatrix.value[timePeriodId][weekday] = {}
  }
  volumeMatrix.value[timePeriodId][weekday][scanTypeId] = 0
}

// 删除检查类型（只删除指定那一天的，无需确认）
function handleRemoveScanType(timePeriodId: string, weekday: number, scanTypeId: string) {
  addedScanTypes.value[timePeriodId]?.[weekday]?.delete(scanTypeId)

  // 清除该检查类型在该天的数据
  if (volumeMatrix.value[timePeriodId]?.[weekday]) {
    delete volumeMatrix.value[timePeriodId][weekday][scanTypeId]
  }
}

// 复制周一到全周（同时复制检查类型列表和数值）
async function handleCopyMondayToWeek(timePeriodId: string) {
  const mondayAdded = addedScanTypes.value[timePeriodId]?.[1]
  if (!mondayAdded || mondayAdded.size === 0) {
    ElMessage.warning('请先配置周一的检查类型')
    return
  }

  const mondayData = volumeMatrix.value[timePeriodId]?.[1]
  if (!mondayData || [...mondayAdded].every(id => !mondayData[id] || mondayData[id] === 0)) {
    ElMessage.warning('请先配置周一的数据')
    return
  }

  try {
    await ElMessageBox.confirm('将周一的配置复制到周二至周日，会覆盖现有配置，确定继续吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    // 复制检查类型列表和数值到其他天
    for (const wd of weekdays) {
      if (wd.value !== 1) {
        // 复制检查类型列表
        addedScanTypes.value[timePeriodId][wd.value] = new Set(mondayAdded)
        // 复制数值
        volumeMatrix.value[timePeriodId][wd.value] = { ...mondayData }
      }
    }

    ElMessage.success('已复制周一配置到全周')
  }
  catch {
    // 用户取消
  }
}

// 清空时间段配置（清空所有天的检查类型列表和数值）
async function handleClearTimePeriod(timePeriodId: string) {
  const hasData = getAddedScanTypesCountForPeriod(timePeriodId) > 0
  if (!hasData) {
    ElMessage.warning('暂无配置可清空')
    return
  }

  try {
    await ElMessageBox.confirm('确定清空该时间段的所有配置吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    for (const wd of weekdays) {
      addedScanTypes.value[timePeriodId][wd.value] = new Set()
      volumeMatrix.value[timePeriodId][wd.value] = {}
    }

    ElMessage.success('已清空')
  }
  catch {
    // 用户取消
  }
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 保存
async function handleSave() {
  saving.value = true
  try {
    const items: Array<{
      weekday: number
      timePeriodId: string
      scanTypeId: string
      volume: number
    }> = []

    for (const timePeriodId of Object.keys(volumeMatrix.value)) {
      for (const wd of weekdays) {
        const dayAdded = addedScanTypes.value[timePeriodId]?.[wd.value]
        const dayData = volumeMatrix.value[timePeriodId][wd.value]
        if (dayAdded && dayData) {
          for (const scanTypeId of dayAdded) {
            const volume = dayData[scanTypeId]
            if (volume > 0) {
              items.push({
                weekday: wd.value,
                timePeriodId,
                scanTypeId,
                volume,
              })
            }
          }
        }
      }
    }

    await saveWeeklyVolumes(props.modalityRoom!.id, props.orgId, items)
    ElMessage.success('保存成功')
    emit('success')
    handleClose()
  }
  catch {
    // 错误由拦截器处理
  }
  finally {
    saving.value = false
  }
}

// 计算某天某时间段的总检查量
function getDayTotal(timePeriodId: string, weekday: number): number {
  const dayData = volumeMatrix.value[timePeriodId]?.[weekday]
  const dayAdded = addedScanTypes.value[timePeriodId]?.[weekday]
  if (!dayData || !dayAdded)
    return 0
  let total = 0
  for (const scanTypeId of dayAdded) {
    total += dayData[scanTypeId] || 0
  }
  return total
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="1400px"
    top="5vh"
    @close="handleClose"
  >
    <div v-loading="loading" class="volume-config">
      <!-- 空状态提示 -->
      <el-alert
        v-if="timePeriods.length === 0 || scanTypes.length === 0"
        type="warning"
        :closable="false"
        show-icon
      >
        <template #title>
          <span v-if="timePeriods.length === 0">请先配置时间段</span>
          <span v-else-if="scanTypes.length === 0">请先配置检查类型</span>
        </template>
      </el-alert>

      <!-- 折叠面板按时间段分组 -->
      <el-collapse v-else v-model="activeTimePeriods" class="time-period-collapse">
        <el-collapse-item
          v-for="tp in sortedTimePeriods"
          :key="tp.id"
          :name="tp.id"
        >
          <!-- 面板标题 -->
          <template #title>
            <div class="collapse-header">
              <span class="time-period-name">{{ tp.name }}</span>
              <span class="time-period-time">{{ tp.startTime }} - {{ tp.endTime }}</span>
              <el-tag v-if="tp.isCrossDay" type="warning" size="small" class="cross-day-tag">
                跨天
              </el-tag>
              <!-- 配置状态 -->
              <el-tag
                v-if="timePeriodStats[tp.id]?.scanTypeCount === 0"
                type="info"
                size="small"
                class="status-tag"
              >
                <el-icon><WarningFilled /></el-icon>
                未配置
              </el-tag>
              <el-tag
                v-else
                type="success"
                size="small"
                class="status-tag"
              >
                {{ timePeriodStats[tp.id]?.scanTypeCount }} 个检查类型
              </el-tag>
            </div>
          </template>

          <!-- 面板内容 -->
          <div class="time-period-content">
            <!-- 工具栏 -->
            <div class="period-toolbar">
              <el-button
                size="small"
                :icon="CopyDocument"
                :disabled="getAddedScanTypesCountForPeriod(tp.id) === 0"
                @click="handleCopyMondayToWeek(tp.id)"
              >
                复制周一到全周
              </el-button>
              <el-button
                size="small"
                type="danger"
                :icon="Delete"
                :disabled="getAddedScanTypesCountForPeriod(tp.id) === 0"
                @click="handleClearTimePeriod(tp.id)"
              >
                清空全部
              </el-button>
            </div>

            <!-- 7天卡片 - 始终显示 -->
            <div class="weekday-cards">
              <div
                v-for="wd in weekdays"
                :key="wd.value"
                class="weekday-card"
                :class="{ weekend: wd.isWeekend }"
              >
                <div class="card-header">
                  <span class="weekday-name">{{ wd.label }}</span>
                  <span class="day-total">合计: {{ getDayTotal(tp.id, wd.value) }}</span>
                </div>
                <div class="card-body">
                  <div
                    v-for="st in getAddedScanTypesForDay(tp.id, wd.value)"
                    :key="st.id"
                    class="scan-type-row"
                  >
                    <span class="scan-type-name" :title="st.name">{{ st.name }}</span>
                    <el-input-number
                      :model-value="getVolume(tp.id, wd.value, st.id)"
                      :min="0"
                      :max="9999"
                      size="small"
                      controls-position="right"
                      class="volume-input"
                      @update:model-value="(val: number | undefined) => setVolume(tp.id, wd.value, st.id, val || 0)"
                    />
                    <el-button
                      type="danger"
                      link
                      size="small"
                      class="remove-btn"
                      @click.stop="handleRemoveScanType(tp.id, wd.value, st.id)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <!-- 空状态提示 -->
                  <div v-if="getAddedScanTypesForDay(tp.id, wd.value).length === 0" class="empty-day">
                    <span class="empty-text">暂无检查类型</span>
                  </div>
                </div>
                <!-- 卡片底部添加按钮 -->
                <div class="card-footer">
                  <el-dropdown
                    trigger="click"
                    :disabled="getAvailableScanTypesForDay(tp.id, wd.value).length === 0"
                    @command="(cmd: string) => handleAddScanType(tp.id, wd.value, cmd)"
                  >
                    <el-button
                      type="primary"
                      link
                      size="small"
                      class="add-btn"
                      :disabled="getAvailableScanTypesForDay(tp.id, wd.value).length === 0"
                    >
                      <el-icon><Plus /></el-icon>
                      添加检查
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item
                          v-for="st in getAvailableScanTypesForDay(tp.id, wd.value)"
                          :key="st.id"
                          :command="st.id"
                        >
                          {{ st.name }}
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>

      <!-- 提示信息 -->
      <div v-if="timePeriods.length > 0 && scanTypes.length > 0" class="tips">
        <ul>
          <li>点击每日卡片底部「添加检查」选择检查类型</li>
          <li>检查量为 0 的记录将不会被保存</li>
          <li>可使用「复制周一到全周」快速配置</li>
        </ul>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button
        type="primary"
        :loading="saving"
        :disabled="timePeriods.length === 0 || scanTypes.length === 0"
        @click="handleSave"
      >
        保存
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.volume-config {
  max-height: 70vh;
  overflow-y: auto;
}

.time-period-collapse {
  :deep(.el-collapse-item__header) {
    height: 48px;
    line-height: 48px;
    font-size: 14px;
  }
}

.collapse-header {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;

  .time-period-name {
    font-weight: 600;
    color: #303133;
  }

  .time-period-time {
    color: #909399;
    font-size: 13px;
  }

  .cross-day-tag {
    margin-left: 4px;
  }

  .status-tag {
    margin-left: auto;
    margin-right: 12px;

    .el-icon {
      margin-right: 4px;
    }
  }
}

.time-period-content {
  padding: 0 4px;
}

.period-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px dashed #ebeef5;
}

.weekday-cards {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 10px;
  align-items: stretch;
}

.weekday-card {
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
  background: #fff;
  display: flex;
  flex-direction: column;

  &.weekend {
    background: #fef0f0;
    border-color: #fab6b6;

    .card-header {
      background: #fde2e2;
    }
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: #f5f7fa;
    border-bottom: 1px solid #e4e7ed;

    .weekday-name {
      font-weight: 600;
      font-size: 13px;
      color: #303133;
    }

    .day-total {
      font-size: 11px;
      color: #909399;
    }
  }

  .card-body {
    padding: 10px;
    flex: 1;
    overflow-y: auto;
  }

  .card-footer {
    padding: 8px 10px;
    border-top: 1px dashed #ebeef5;
    text-align: center;
    margin-top: auto;

    .add-btn {
      font-size: 12px;
    }
  }

  .empty-day {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 60px;

    .empty-text {
      font-size: 12px;
      color: #c0c4cc;
    }
  }

  .scan-type-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5px 0;
    gap: 6px;

    &:not(:last-child) {
      border-bottom: 1px dashed #ebeef5;
    }

    .scan-type-name {
      font-size: 12px;
      color: #606266;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      min-width: 50px;
    }

    .volume-input {
      width: 80px;
      flex-shrink: 0;

      :deep(.el-input__inner) {
        text-align: center;
      }
    }

    .remove-btn {
      flex-shrink: 0;
      padding: 2px;
    }
  }
}

.tips {
  margin-top: 16px;
  padding: 10px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  color: #909399;

  ul {
    margin: 0;
    padding-left: 16px;

    li {
      line-height: 1.8;
    }
  }
}
</style>
