<script setup lang="ts">
import { OfficeBuilding } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const currentNodeName = computed(() => auth.currentNode?.node_name ?? '未选择')
const nodes = computed(() => auth.availableNodes)
const hasMultipleNodes = computed(() => nodes.value.length > 1)

async function handleSwitch(nodeId: string) {
  if (nodeId === auth.currentNodeId)
    return
  await auth.selectNode(nodeId)
  // 切换节点后刷新页面数据
  window.location.reload()
}
</script>

<template>
  <el-dropdown v-if="hasMultipleNodes" trigger="click" @command="handleSwitch">
    <div class="org-selector">
      <el-icon><OfficeBuilding /></el-icon>
      <span class="org-name">{{ currentNodeName }}</span>
      <el-icon class="arrow">
        <i class="el-icon-arrow-down" />
      </el-icon>
    </div>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="node in nodes"
          :key="node.node_id"
          :command="node.node_id"
          :class="{ 'is-active': node.node_id === auth.currentNodeId }"
        >
          <div class="node-option">
            <span class="node-name">{{ node.node_name }}</span>
            <span class="node-role">{{ node.role_name }}</span>
          </div>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
  <div v-else class="org-label">
    <el-icon><OfficeBuilding /></el-icon>
    <span class="org-name">{{ currentNodeName }}</span>
  </div>
</template>

<style scoped>
.org-selector,
.org-label {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 13px;
  color: var(--text-regular);
  cursor: pointer;
  transition: background 0.2s;
}

.org-selector:hover {
  background: var(--bg-hover);
}

.org-label {
  cursor: default;
}

.org-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-width: 160px;
}

.node-role {
  font-size: 12px;
  color: var(--text-placeholder);
}

:deep(.is-active) {
  color: var(--el-color-primary);
  font-weight: 600;
}
</style>
