<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'

interface Props {
  ruleDependencies: SchedulingRule.RuleDependencyInfo[]
  ruleConflicts: SchedulingRule.RuleConflictInfo[]
  rules: SchedulingRule.ClassifiedRuleInfo[]
  ruleExecutionOrder?: string[]
}

const props = defineProps<Props>()

const chartContainer = ref<HTMLElement>()
let chartInstance: echarts.ECharts | null = null

// 构建图表数据
const chartOption = computed<EChartsOption>(() => {
  const nodes: any[] = []
  const edges: any[] = []
  const ruleMap = new Map<string, SchedulingRule.ClassifiedRuleInfo>()

  // 创建规则节点映射
  props.rules.forEach((rule) => {
    ruleMap.set(rule.ruleId, rule)
    nodes.push({
      id: rule.ruleId,
      name: rule.ruleName,
      category: rule.category,
      symbolSize: 50,
      label: {
        show: true,
        formatter: (params: any) => {
          const rule = ruleMap.get(params.data.id)
          if (!rule) return params.data.name
          return `${params.data.name}\n[${getCategoryText(rule.category)}]`
        },
      },
      itemStyle: {
        color: getCategoryColor(rule.category),
      },
    })
  })

  // 添加依赖关系边
  props.ruleDependencies.forEach((dep) => {
    edges.push({
      source: dep.dependentRuleID,
      target: dep.dependentOnRuleID,
      label: {
        show: true,
        formatter: dep.dependencyType,
        fontSize: 10,
      },
      lineStyle: {
        color: '#409EFF',
        width: 2,
        type: 'solid',
        curveness: 0.2,
      },
      arrow: {
        show: true,
        type: 'triangle',
        length: 10,
      },
    })
  })

  // 添加冲突关系边（用红色虚线表示）
  props.ruleConflicts.forEach((conflict) => {
    edges.push({
      source: conflict.ruleID1,
      target: conflict.ruleID2,
      label: {
        show: true,
        formatter: '冲突',
        fontSize: 10,
        color: '#F56C6C',
      },
      lineStyle: {
        color: '#F56C6C',
        width: 2,
        type: 'dashed',
        curveness: 0.3,
      },
    })
  })

  // 如果有执行顺序，按顺序排列节点
  let layout = 'force'
  if (props.ruleExecutionOrder && props.ruleExecutionOrder.length > 0) {
    layout = 'circular'
    // 按执行顺序重新排列节点
    const orderedNodes = props.ruleExecutionOrder.map((ruleId, index) => {
      const node = nodes.find(n => n.id === ruleId)
      if (node) {
        node.x = Math.cos((index * 2 * Math.PI) / props.ruleExecutionOrder!.length) * 200
        node.y = Math.sin((index * 2 * Math.PI) / props.ruleExecutionOrder!.length) * 200
      }
      return node
    }).filter(Boolean)
    nodes.splice(0, nodes.length, ...orderedNodes)
  }

  return {
    title: {
      text: '规则依赖关系图',
      left: 'center',
      textStyle: {
        fontSize: 16,
        fontWeight: 'bold',
      },
    },
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        if (params.dataType === 'node') {
          const rule = ruleMap.get(params.data.id)
          if (!rule) return params.data.name
          return `
            <div style="padding: 8px;">
              <div style="font-weight: bold; margin-bottom: 4px;">${params.data.name}</div>
              <div>分类: ${getCategoryText(rule.category)} / ${rule.subCategory}</div>
              <div>类型: ${rule.ruleType}</div>
              <div>优先级: ${rule.priority}</div>
              ${rule.description ? `<div>描述: ${rule.description}</div>` : ''}
            </div>
          `
        }
        else if (params.dataType === 'edge') {
          return `
            <div style="padding: 8px;">
              <div style="font-weight: bold;">${params.data.label?.formatter || '关系'}</div>
            </div>
          `
        }
        return ''
      },
    },
    legend: {
      data: ['约束规则', '偏好规则', '依赖规则'],
      bottom: 10,
    },
    series: [
      {
        type: 'graph',
        layout: layout,
        data: nodes,
        links: edges,
        categories: [
          { name: '约束规则', itemStyle: { color: '#F56C6C' } },
          { name: '偏好规则', itemStyle: { color: '#E6A23C' } },
          { name: '依赖规则', itemStyle: { color: '#409EFF' } },
        ],
        roam: true,
        label: {
          show: true,
          position: 'right',
          formatter: '{b}',
        },
        labelLayout: {
          hideOverlap: true,
        },
        lineStyle: {
          color: 'source',
          curveness: 0.2,
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: {
            width: 4,
          },
        },
        force: {
          repulsion: 1000,
          gravity: 0.1,
          edgeLength: 150,
          layoutAnimation: true,
        },
      },
    ],
  }
})

// 获取分类颜色
function getCategoryColor(category: string): string {
  switch (category) {
    case 'constraint':
      return '#F56C6C'
    case 'preference':
      return '#E6A23C'
    case 'dependency':
      return '#409EFF'
    default:
      return '#909399'
  }
}

// 获取分类文本
function getCategoryText(category: string): string {
  switch (category) {
    case 'constraint':
      return '约束'
    case 'preference':
      return '偏好'
    case 'dependency':
      return '依赖'
    default:
      return category
  }
}

// 初始化图表
function initChart() {
  if (!chartContainer.value) return

  if (chartInstance) {
    chartInstance.dispose()
  }

  chartInstance = echarts.init(chartContainer.value)
  chartInstance.setOption(chartOption.value)
}

// 监听数据变化
watch(
  () => [props.ruleDependencies, props.ruleConflicts, props.rules, props.ruleExecutionOrder],
  () => {
    if (chartInstance) {
      chartInstance.setOption(chartOption.value)
    }
  },
  { deep: true },
)

onMounted(() => {
  initChart()
  window.addEventListener('resize', () => {
    chartInstance?.resize()
  })
})
</script>

<template>
  <div ref="chartContainer" style="width: 100%; height: 600px;" />
</template>

<style lang="scss" scoped>
// 图表容器样式
</style>
