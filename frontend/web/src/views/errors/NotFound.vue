<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="error-page">
    <div class="error-content">
      <h1 class="error-code">
        404
      </h1>
      <p class="error-text">
        抱歉，您访问的页面不存在
      </p>
      <el-button type="primary" @click="router.push('/')">
        返回首页
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.error-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f5f7fa;
}

.error-content {
  text-align: center;
}

.error-code {
  font-size: 120px;
  font-weight: 800;
  color: #e5e7eb;
  margin: 0;
  line-height: 1;
}

.error-text {
  font-size: 18px;
  color: #6b7280;
  margin: 16px 0 32px;
}
</style>
