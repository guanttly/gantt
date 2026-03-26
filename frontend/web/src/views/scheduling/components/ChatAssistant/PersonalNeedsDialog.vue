<script setup lang="ts">
import { Calendar, Connection, User } from '@element-plus/icons-vue'
import { ElButton, ElCollapse, ElCollapseItem, ElDialog, ElEmpty, ElTag } from 'element-plus'
import { computed, ref, watch } from 'vue'

interface NeedItem {
  id: string
  staffId: string
  staffName: string
  date?: string
  dayOfWeek?: number
  shiftId?: string
  shiftName?: string
  type: 'permanent' | 'temporary'
  description: string
}

interface StaffNeedsGroup {
  staffId: string
  staffName: string
  needs: NeedItem[]
}

interface PersonalNeedsData {
  totalNeeds: number
  permanentCount: number
  temporaryCount: number
  needsByStaff: StaffNeedsGroup[]
}

const props = defineProps<{
  visible: boolean
  data: PersonalNeedsData | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
}>()

const activeNames = ref<string[]>([])
const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

const stats = computed(() => {
  if (!props.data)
    return { totalNeeds: 0, permanentCount: 0, temporaryCount: 0, staffCount: 0 }
  return {
    totalNeeds: props.data.totalNeeds,
    permanentCount: props.data.permanentCount,
    temporaryCount: props.data.temporaryCount,
    staffCount: props.data.needsByStaff.length,
  }
})

watch(() => props.visible, (newVal) => {
  if (newVal && props.data?.needsByStaff?.length) {
    activeNames.value = [props.data.needsByStaff[0].staffId]
  }
})

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :before-close="handleClose"
    :model-value="visible"
    title="个人需求详情"
    top="5vh"
    width="700px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="data" class="needs-content">
      <div class="stats-overview">
        <div class="stat-card">
          <div class="stat-icon" style="background: #ecf5ff; color: #409eff;">
            <el-icon :size="22">
              <Calendar />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.totalNeeds }}
            </div>
            <div class="stat-label">
              需求总数
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #f0f9ff; color: #67c23a;">
            <el-icon :size="22">
              <Connection />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.permanentCount }}
            </div>
            <div class="stat-label">
              固定需求
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #fef0e6; color: #e6a23c;">
            <el-icon :size="22">
              <Calendar />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.temporaryCount }}
            </div>
            <div class="stat-label">
              临时需求
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" style="background: #fef0f0; color: #f56c6c;">
            <el-icon :size="22">
              <User />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              {{ stats.staffCount }}
            </div>
            <div class="stat-label">
              涉及人员
            </div>
          </div>
        </div>
      </div>

      <div class="needs-section">
        <div class="section-title">
          按人员分组
        </div>
        <ElCollapse v-model="activeNames" class="needs-collapse">
          <ElCollapseItem
            v-for="group in data.needsByStaff"
            :key="group.staffId"
            :name="group.staffId"
          >
            <template #title>
              <div class="group-header">
                <div class="group-info">
                  <div class="staff-avatar">
                    <el-icon><User /></el-icon>
                  </div>
                  <span class="staff-name">{{ group.staffName }}</span>
                </div>
                <ElTag effect="plain" type="primary">
                  {{ group.needs.length }} 条需求
                </ElTag>
              </div>
            </template>

            <div v-if="group.needs.length > 0" class="needs-list">
              <div
                v-for="need in group.needs"
                :key="need.id"
                class="need-item"
                :class="{ 'need-permanent': need.type === 'permanent', 'need-temporary': need.type === 'temporary' }"
              >
                <div class="need-header">
                  <ElTag :type="need.type === 'permanent' ? 'success' : 'warning'" effect="plain" size="small">
                    {{ need.type === 'permanent' ? '固定' : '临时' }}
                  </ElTag>
                  <span v-if="need.shiftName" class="need-shift">{{ need.shiftName }}</span>
                  <span v-if="need.date" class="need-date">{{ need.date }}</span>
                  <span v-if="need.dayOfWeek !== undefined" class="need-day">
                    每{{ WEEKDAY_NAMES[need.dayOfWeek] }}
                  </span>
                </div>
                <div class="need-description">
                  {{ need.description }}
                </div>
              </div>
            </div>
            <ElEmpty v-else :image-size="60" description="暂无需求" />
          </ElCollapseItem>
        </ElCollapse>
      </div>
    </div>
    <ElEmpty v-else description="暂无个人需求数据" />

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.needs-content { padding: 8px 0; }

.stats-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}

.stat-card { display: flex; align-items: center; gap: 10px; padding: 14px; background: var(--el-bg-color); border: 1px solid var(--el-border-color-light); border-radius: 8px; }
.stat-icon { width: 44px; height: 44px; border-radius: 8px; display: flex; align-items: center; justify-content: center; }
.stat-value { font-size: 22px; font-weight: 600; }
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }

.needs-section { margin-top: 24px; max-height: 550px; overflow-y: auto; }
.section-title { font-size: 15px; font-weight: 600; margin-bottom: 16px; padding-bottom: 8px; border-bottom: 2px solid var(--el-border-color-lighter); }

.needs-collapse {
  border: none;
  :deep(.el-collapse-item) { margin-bottom: 12px; border: 1px solid var(--el-border-color-light); border-radius: 8px; overflow: hidden; }
  :deep(.el-collapse-item__header) { height: auto; padding: 14px 16px; border: none; }
  :deep(.el-collapse-item__content) { padding: 16px; }
}

.group-header { display: flex; align-items: center; justify-content: space-between; width: 100%; padding-right: 12px; }
.group-info { display: flex; align-items: center; gap: 10px; }
.staff-avatar { width: 32px; height: 32px; border-radius: 50%; background: var(--el-color-primary-light-9); color: var(--el-color-primary); display: flex; align-items: center; justify-content: center; }
.staff-name { font-size: 14px; font-weight: 600; }

.needs-list { display: flex; flex-direction: column; gap: 10px; }
.need-item { padding: 12px; border-radius: 6px; border: 1px solid var(--el-border-color-lighter); }
.need-permanent { border-left: 3px solid #67c23a; background: rgba(103, 194, 58, 0.04); }
.need-temporary { border-left: 3px solid #e6a23c; background: rgba(230, 162, 60, 0.04); }
.need-header { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.need-shift { font-size: 13px; color: var(--el-text-color-secondary); }
.need-date { font-size: 13px; color: var(--el-text-color-secondary); }
.need-day { font-size: 13px; color: var(--el-color-primary); font-weight: 500; }
.need-description { font-size: 13px; color: var(--el-text-color-regular); line-height: 1.6; }
</style>
