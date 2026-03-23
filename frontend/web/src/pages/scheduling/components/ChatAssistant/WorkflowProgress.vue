<script setup lang="ts">
import { computed } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'

interface WorkflowStep {
  key: string
  label: string
  icon: string
  description: string
}

const props = defineProps<{
  currentPhase?: string
  workflow?: string
}>()

// 排班创建工作流的步骤定义
const scheduleCreationSteps: WorkflowStep[] = [
  {
    key: 'confirming_period',
    label: '确认周期',
    icon: 'calendar',
    description: '设置排班时间范围',
  },
  {
    key: 'confirming_shifts',
    label: '选择班次',
    icon: 'clock',
    description: '选择需要排班的班次',
  },
  {
    key: 'confirming_staff_count',
    label: '确认人数',
    icon: 'users',
    description: '设置每个班次的人数需求',
  },
  {
    key: 'retrieving_staff',
    label: '检索人员',
    icon: 'search',
    description: '查询可用人员',
  },
  {
    key: 'retrieving_rules',
    label: '加载规则',
    icon: 'clipboard',
    description: '获取排班规则',
  },
  {
    key: 'generating_schedule',
    label: '生成排班',
    icon: 'gear',
    description: 'AI 自动生成排班表',
  },
  {
    key: 'previewing_draft',
    label: '预览草案',
    icon: 'eye',
    description: '查看并确认排班结果',
  },
  {
    key: 'saving_schedule',
    label: '保存排班',
    icon: 'save',
    description: '保存到数据库',
  },
  {
    key: 'completed',
    label: '完成',
    icon: 'check-circle',
    description: '排班创建成功',
  },
]

// V3 工作流步骤映射（将V3状态映射到V2步骤）
const v3PhaseToStepMap: Record<string, string> = {
  '_schedule_v3_create_confirm_period_': 'confirming_period',
  '_schedule_v3_create_confirm_shifts_': 'confirming_shifts',
  '_schedule_v3_create_confirm_staff_count_': 'confirming_staff_count',
  '_schedule_v3_create_personal_needs_': 'retrieving_staff', // 个人需求阶段映射到检索人员
  '_schedule_v3_create_requirement_assessment_': 'retrieving_rules', // 需求评估映射到加载规则
  '_schedule_v3_create_plan_review_': 'generating_schedule', // 计划预览映射到生成排班
  '_schedule_v3_create_progressive_task_': 'generating_schedule', // 渐进式任务执行映射到生成排班
  '_schedule_v3_create_task_review_': 'generating_schedule', // 任务审核映射到生成排班
  '_schedule_v3_create_confirm_saving_': 'saving_schedule', // 确认保存映射到保存排班
  '_completed_': 'completed', // 完成状态
}

// 当前步骤索引
const currentStepIndex = computed(() => {
  if (!props.currentPhase)
    return -1
  
  // 如果是V3工作流，先映射到V2步骤
  let phaseKey = props.currentPhase
  if (props.workflow === 'schedule_v3.create' && v3PhaseToStepMap[props.currentPhase]) {
    phaseKey = v3PhaseToStepMap[props.currentPhase]
  }
  
  return scheduleCreationSteps.findIndex(step => phaseKey?.includes(step.key))
})

// 当前步骤
const currentStep = computed(() => {
  if (currentStepIndex.value >= 0)
    return scheduleCreationSteps[currentStepIndex.value]
  return null
})

// 判断步骤状态
function getStepStatus(index: number): 'wait' | 'process' | 'finish' | 'error' {
  if (currentStepIndex.value < 0)
    return 'wait'
  if (index < currentStepIndex.value)
    return 'finish'
  if (index === currentStepIndex.value)
    return 'process'
  return 'wait'
}

// 判断是否显示工作流进度
const showProgress = computed(() => {
  // 支持V2和V3工作流
  const isV2Workflow = props.workflow === 'schedule_create' || props.workflow === 'schedule_v2.create'
  const isV3Workflow = props.workflow === 'schedule_v3.create'
  return (isV2Workflow || isV3Workflow) && currentStepIndex.value >= 0
})
</script>

<template>
  <div v-if="showProgress" class="workflow-progress">
    <div class="progress-header">
      <div class="current-step">
        <span class="step-icon"><SvgIcon :name="currentStep?.icon || 'calendar'" size="24px" /></span>
        <span class="step-label">{{ currentStep?.label }}</span>
      </div>
      <div class="step-description">
        {{ currentStep?.description }}
      </div>
    </div>

    <div class="progress-steps">
      <el-steps :active="currentStepIndex" align-center finish-status="success">
        <el-step
          v-for="(step, index) in scheduleCreationSteps"
          :key="step.key"
          :title="step.label"
          :status="getStepStatus(index)"
        />
      </el-steps>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.workflow-progress {
  padding: 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px;
  margin-bottom: 16px;
  color: #fff;

  .progress-header {
    margin-bottom: 16px;

    .current-step {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 18px;
      font-weight: 600;
      margin-bottom: 8px;

      .step-icon {
        font-size: 24px;
      }
    }

    .step-description {
      font-size: 14px;
      opacity: 0.9;
      padding-left: 32px;
    }
  }

  .progress-steps {
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    border-radius: 8px;
    padding: 16px;

    :deep(.el-steps) {
      .el-step__title {
        color: rgba(255, 255, 255, 0.8);
        font-size: 12px;
      }

      .el-step__icon {
        border-color: rgba(255, 255, 255, 0.5);
        color: rgba(255, 255, 255, 0.8);
      }

      .el-step__icon.is-text {
        border-color: rgba(255, 255, 255, 0.5);
        background: rgba(255, 255, 255, 0.1);
      }

      .el-step.is-process {
        .el-step__title {
          color: #fff;
          font-weight: 600;
        }

        .el-step__icon {
          border-color: #fff;
          color: #fff;
          background: rgba(255, 255, 255, 0.2);
        }
      }

      .el-step.is-finish {
        .el-step__title {
          color: #67c23a;
        }

        .el-step__icon {
          border-color: #67c23a;
          color: #67c23a;
        }
      }

      .el-step__line {
        background-color: rgba(255, 255, 255, 0.3);
      }

      .el-step.is-finish .el-step__line {
        background-color: #67c23a;
      }
    }
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .workflow-progress {
    background: linear-gradient(135deg, #434343 0%, #000000 100%);
  }
}
</style>
