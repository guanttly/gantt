<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { getActiveModalityRooms } from '@/api/modality-room'
import { getShiftList } from '@/api/shift'
import { createStaffingRule, updateStaffingRule } from '@/api/staffing'
import { getActiveTimePeriods } from '@/api/time-period'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  rule: Staffing.Rule | null
  orgId: string
  existingShiftIds: string[] // 已有规则的班次ID列表
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const formRef = ref<FormInstance>()
const loading = ref(false)
const shifts = ref<Shift.ShiftInfo[]>([])
const modalityRooms = ref<ModalityRoom.Info[]>([])
const timePeriods = ref<TimePeriod.Info[]>([])
const loadingData = ref(false)

// 表单数据
const formData = reactive<Staffing.RuleRequest>({
  shiftId: '',
  modalityRoomIds: [],
  timePeriodId: '',
  avgReportLimit: 50,
  roundingMode: 'ceil',
  description: '',
})

// 表单验证规则
const rules: FormRules = {
  shiftId: [
    { required: true, message: '请选择班次', trigger: 'change' },
  ],
  modalityRoomIds: [
    { required: true, type: 'array', min: 1, message: '请选择至少一个机房', trigger: 'change' },
  ],
  timePeriodId: [
    { required: true, message: '请选择时间段', trigger: 'change' },
  ],
  avgReportLimit: [
    { required: true, message: '请输入人均报告上限', trigger: 'blur' },
    { type: 'number', min: 1, message: '人均报告上限必须大于0', trigger: 'blur' },
  ],
  roundingMode: [
    { required: true, message: '请选择取整方式', trigger: 'change' },
  ],
}

// 对话框标题
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '新增计算规则' : '编辑计算规则'
})

// 可选班次列表（排除已有规则的班次）
const availableShifts = computed(() => {
  if (props.mode === 'edit') {
    return shifts.value
  }
  return shifts.value.filter(s => !props.existingShiftIds.includes(s.id))
})

// 取整方式选项
const roundingModeOptions = [
  { label: '向上取整', value: 'ceil' },
  { label: '向下取整', value: 'floor' },
]

// 加载基础数据
async function loadData() {
  loadingData.value = true
  try {
    const [shiftRes, roomRes, periodRes] = await Promise.all([
      getShiftList({ orgId: props.orgId, isActive: true, page: 1, size: 100 }),
      getActiveModalityRooms(props.orgId),
      getActiveTimePeriods(props.orgId),
    ])
    shifts.value = shiftRes.items || []
    modalityRooms.value = roomRes || []
    timePeriods.value = periodRes || []
  }
  catch {
    ElMessage.error('加载基础数据失败')
  }
  finally {
    loadingData.value = false
  }
}

// 监听 visible 变化，初始化表单
watch(() => props.visible, async (val) => {
  if (val) {
    await loadData()

    if (props.mode === 'edit' && props.rule) {
      Object.assign(formData, {
        shiftId: props.rule.shiftId,
        modalityRoomIds: props.rule.modalityRoomIds || [],
        timePeriodId: props.rule.timePeriodId,
        avgReportLimit: props.rule.avgReportLimit,
        roundingMode: props.rule.roundingMode,
        description: props.rule.description || '',
      })
    }
    else {
      resetForm()
    }
  }
})

// 重置表单
function resetForm() {
  Object.assign(formData, {
    shiftId: '',
    modalityRoomIds: [],
    timePeriodId: '',
    avgReportLimit: 50,
    roundingMode: 'ceil',
    description: '',
  })
  formRef.value?.clearValidate()
}

// 关闭对话框
function handleClose() {
  emit('update:visible', false)
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value)
    return

  try {
    await formRef.value.validate()

    loading.value = true

    if (props.mode === 'create') {
      await createStaffingRule(props.orgId, formData)
      ElMessage.success('创建成功')
    }
    else {
      await updateStaffingRule(props.rule!.id, props.orgId, formData)
      ElMessage.success('更新成功')
    }

    emit('success')
    handleClose()
  }
  catch {
    // 表单验证失败(error === false)或请求错误(由拦截器处理)
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="600px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      v-loading="loadingData"
      :model="formData"
      :rules="rules"
      label-width="120px"
    >
      <el-form-item label="班次" prop="shiftId">
        <el-select
          v-model="formData.shiftId"
          placeholder="请选择班次"
          style="width: 100%"
          :disabled="mode === 'edit'"
        >
          <el-option
            v-for="shift in availableShifts"
            :key="shift.id"
            :label="`${shift.name} (${shift.startTime}-${shift.endTime})`"
            :value="shift.id"
          />
        </el-select>
        <div v-if="mode === 'create' && availableShifts.length === 0" class="form-tip warning">
          所有班次都已配置规则
        </div>
      </el-form-item>

      <el-form-item label="关联机房" prop="modalityRoomIds">
        <el-select
          v-model="formData.modalityRoomIds"
          multiple
          placeholder="请选择关联的机房"
          style="width: 100%"
        >
          <el-option
            v-for="room in modalityRooms"
            :key="room.id"
            :label="room.name"
            :value="room.id"
          />
        </el-select>
        <div class="form-tip">
          选择用于计算该班次检查量的机房
        </div>
      </el-form-item>

      <el-form-item label="时间段" prop="timePeriodId">
        <el-select
          v-model="formData.timePeriodId"
          placeholder="请选择时间段"
          style="width: 100%"
        >
          <el-option
            v-for="period in timePeriods"
            :key="period.id"
            :label="`${period.name} (${period.startTime}-${period.endTime})`"
            :value="period.id"
          />
        </el-select>
        <div class="form-tip">
          选择与该班次对应的检查量统计时间段
        </div>
      </el-form-item>

      <el-form-item label="人均报告上限" prop="avgReportLimit">
        <el-input-number
          v-model="formData.avgReportLimit"
          :min="1"
          :max="9999"
          :step="10"
          style="width: 100%"
        />
        <div class="form-tip">
          每人每天可处理的报告数量上限
        </div>
      </el-form-item>

      <el-form-item label="取整方式" prop="roundingMode">
        <el-radio-group v-model="formData.roundingMode">
          <el-radio
            v-for="opt in roundingModeOptions"
            :key="opt.value"
            :value="opt.value"
          >
            {{ opt.label }}
          </el-radio>
        </el-radio-group>
        <div class="form-tip">
          向上取整确保人员充足，向下取整节约人力
        </div>
      </el-form-item>

      <el-form-item label="规则说明" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="2"
          placeholder="可选，描述该规则的用途"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">
        取消
      </el-button>
      <el-button
        type="primary"
        :loading="loading"
        :disabled="mode === 'create' && availableShifts.length === 0"
        @click="handleSubmit"
      >
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;

  &.warning {
    color: #e6a23c;
  }
}
</style>
