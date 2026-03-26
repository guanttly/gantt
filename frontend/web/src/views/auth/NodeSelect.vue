<script setup lang="ts">
import { OfficeBuilding } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const loading = ref(false)
const selectedNodeId = ref<string>('')

const nodes = computed(() => auth.availableNodes)

async function handleSelect() {
  if (!selectedNodeId.value) {
    ElMessage.warning('请选择一个组织节点')
    return
  }

  loading.value = true
  try {
    await auth.selectNode(selectedNodeId.value)
    await router.push('/')
    ElMessage.success('切换成功')
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '切换失败')
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="node-select-page">
    <div class="node-select-container">
      <h2 class="title">
        选择组织节点
      </h2>
      <p class="desc">
        您有多个组织节点的权限，请选择要进入的节点
      </p>

      <div class="node-list">
        <div
          v-for="node in nodes"
          :key="node.node_id"
          class="node-card"
          :class="{ selected: selectedNodeId === node.node_id }"
          @click="selectedNodeId = node.node_id"
        >
          <el-icon class="node-icon" :size="24">
            <OfficeBuilding />
          </el-icon>
          <div class="node-info">
            <div class="node-name">
              {{ node.node_name }}
            </div>
            <div class="node-path">
              {{ node.node_path }}
            </div>
            <div class="node-role">
              角色：{{ node.role_name }}
            </div>
          </div>
        </div>
      </div>

      <el-button
        type="primary"
        size="large"
        :loading="loading"
        :disabled="!selectedNodeId"
        class="confirm-btn"
        @click="handleSelect"
      >
        进入
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.node-select-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.node-select-container {
  width: 480px;
  padding: 48px 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgb(0 0 0 / 15%);
}

.title {
  text-align: center;
  font-size: 24px;
  font-weight: 700;
  color: #1a1a2e;
  margin: 0 0 8px;
}

.desc {
  text-align: center;
  font-size: 14px;
  color: #6b7280;
  margin: 0 0 32px;
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 32px;
}

.node-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  border: 2px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.node-card:hover {
  border-color: #a78bfa;
  background: #faf5ff;
}

.node-card.selected {
  border-color: #7c3aed;
  background: #f5f3ff;
}

.node-icon {
  color: #7c3aed;
  flex-shrink: 0;
}

.node-info {
  flex: 1;
  min-width: 0;
}

.node-name {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a2e;
}

.node-path {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
}

.node-role {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}

.confirm-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  border-radius: 8px;
}
</style>
