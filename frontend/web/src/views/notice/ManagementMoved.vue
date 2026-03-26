<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="notice-page">
    <div class="notice-card">
      <div class="notice-badge">入口调整</div>
      <h2>基础配置已经迁移到管理后台</h2>
      <p>
        员工、班次、分组、规则、组织和系统配置不再在业务端维护。
        请改用平台管理后台完成这些操作，业务端只保留排班执行、请假和 AI 协同能力。
      </p>
      <div class="notice-actions">
        <el-button type="primary" @click="router.push('/dashboard')">
          返回仪表盘
        </el-button>
        <el-button @click="router.push('/scheduling')">
          前往排班列表
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notice-page {
  min-height: calc(100vh - 132px);
  display: grid;
  place-items: center;
  padding: 32px;
}

.notice-card {
  width: min(720px, 100%);
  padding: 36px;
  border-radius: 24px;
  background: linear-gradient(180deg, #fff 0%, #f8fafc 100%);
  border: 1px solid #e2e8f0;
  box-shadow: 0 24px 60px rgb(15 23 42 / 10%);
}

.notice-badge {
  display: inline-flex;
  padding: 6px 12px;
  border-radius: 999px;
  background: #dbeafe;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 700;
}

h2 {
  margin: 18px 0 12px;
  font-size: 30px;
  line-height: 1.2;
  color: #0f172a;
}

p {
  margin: 0;
  color: #475569;
  line-height: 1.8;
}

.notice-actions {
  display: flex;
  gap: 12px;
  margin-top: 28px;
}

@media (max-width: 640px) {
  .notice-card {
    padding: 28px 22px;
  }

  .notice-actions {
    flex-direction: column;
  }
}
</style>