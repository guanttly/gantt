// 组织节点组合函数
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function useOrgNode() {
  const auth = useAuthStore()

  const currentNodeId = computed(() => auth.currentNodeId)
  const currentNodePath = computed(() => auth.currentNodePath)
  const currentNodeName = computed(() => auth.currentNode?.node_name ?? '')
  const currentRole = computed(() => auth.currentRole)
  const availableNodes = computed(() => auth.availableNodes)

  /** 是否是机构级管理员 */
  const isOrgAdmin = computed(() => auth.hasRole('org_admin'))

  /** 是否是科室管理员 */
  const isDeptAdmin = computed(() => auth.hasRole('dept_admin'))

  /** 是否是排班员 */
  const isScheduler = computed(() => auth.hasRole('scheduler'))

  return {
    currentNodeId,
    currentNodePath,
    currentNodeName,
    currentRole,
    availableNodes,
    isOrgAdmin,
    isDeptAdmin,
    isScheduler,
    switchNode: auth.selectNode,
  }
}
