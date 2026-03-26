<script setup lang="ts">
import { Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { forceResetPassword } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const loading = ref(false)

const form = reactive({
  newPassword: '',
  confirmPassword: '',
})

const rules = {
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        if (value !== form.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        }
        else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

const formRef = ref()

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }

  loading.value = true
  try {
    await forceResetPassword({ new_password: form.newPassword })
    auth.mustResetPwd = false
    ElMessage.success('密码重置成功')
    await router.push('/')
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '密码重置失败')
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="reset-page">
    <div class="reset-container">
      <div class="reset-header">
        <h1 class="reset-title">
          重置密码
        </h1>
        <p class="reset-subtitle">
          首次登录需要重置默认密码，密码至少 8 位，需包含大写、小写字母和数字
        </p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        class="reset-form"
        @keyup.enter="handleSubmit"
      >
        <el-form-item prop="newPassword">
          <el-input
            v-model="form.newPassword"
            type="password"
            placeholder="新密码"
            size="large"
            show-password
            :prefix-icon="Lock"
          />
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="确认新密码"
            size="large"
            show-password
            :prefix-icon="Lock"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="reset-btn"
            @click="handleSubmit"
          >
            确认重置
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.reset-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.reset-container {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgb(0 0 0 / 15%);
}

.reset-header {
  margin-bottom: 32px;
  text-align: center;
}

.reset-title {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 600;
  color: #1a1a2e;
}

.reset-subtitle {
  margin: 0;
  font-size: 14px;
  color: #6c757d;
  line-height: 1.6;
}

.reset-form {
  margin-top: 24px;
}

.reset-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
}
</style>
