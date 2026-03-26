<script setup lang="ts">
import { LockClosedOutline } from '@vicons/ionicons5'
import type { FormInst, FormRules } from 'naive-ui'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NIcon, NInput, useMessage } from 'naive-ui'
import { forceResetPassword } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const message = useMessage()
const loading = ref(false)

const form = reactive({ newPassword: '', confirmPassword: '' })
const formRef = ref<FormInst | null>(null)

function isStrongPassword(password: string) {
  return password.length >= 8
    && /[A-Z]/.test(password)
    && /[a-z]/.test(password)
    && /\d/.test(password)
}

const rules: FormRules = {
  newPassword: [
    { required: true, message: '请输入新密码', trigger: ['blur', 'input'] },
    { min: 8, message: '密码至少 8 位', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string) => {
        if (!value) {
          return true
        }
        if (!isStrongPassword(value)) {
          return new Error('密码强度不足：至少 8 位，需包含大写、小写和数字')
        }
        return true
      },
      trigger: ['blur', 'input'],
    },
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: ['blur', 'input'] },
    {
      validator: (_rule: unknown, value: string) => {
        if (value !== form.newPassword)
          return new Error('两次密码不一致')
        return true
      },
      trigger: ['blur', 'input'],
    },
  ],
}

async function handleSubmit() {
  try { await formRef.value?.validate() }
  catch { return }

  loading.value = true
  try {
    await forceResetPassword({ new_password: form.newPassword })
    auth.mustResetPwd = false
    message.success('密码重置成功')
    await router.push('/')
  }
  catch {}
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-shell">
    <div class="auth-panel">
      <section class="auth-brand">
        <span class="auth-brand-badge">安全重置流程</span>
        <div>
          <h2 class="auth-brand-title">首次登录需要替换默认密码</h2>
          <p class="auth-brand-copy">
            为避免平台管理员默认口令被长期保留，系统会在首次登录后强制进入安全重置流程。
          </p>
        </div>
      </section>

      <section class="auth-card">
        <div class="auth-card-header">
          <h1>重置登录密码</h1>
          <p>密码至少 8 位，建议同时包含大写字母、小写字母、数字和特殊字符。</p>
        </div>
        <n-form ref="formRef" :model="form" :rules="rules" class="auth-form" @keyup.enter="handleSubmit">
          <n-form-item path="newPassword">
            <n-input v-model:value="form.newPassword" type="password" placeholder="新密码" size="large" show-password-on="click">
              <template #prefix>
                <n-icon :size="18" color="#94a3b8">
                  <lock-closed-outline />
                </n-icon>
              </template>
            </n-input>
          </n-form-item>
          <n-form-item path="confirmPassword">
            <n-input v-model:value="form.confirmPassword" type="password" placeholder="确认新密码" size="large" show-password-on="click">
              <template #prefix>
                <n-icon :size="18" color="#94a3b8">
                  <lock-closed-outline />
                </n-icon>
              </template>
            </n-input>
          </n-form-item>
          <n-form-item>
            <n-button type="primary" size="large" :loading="loading" class="auth-submit" block @click="handleSubmit">确认重置</n-button>
          </n-form-item>
        </n-form>
      </section>
    </div>
  </div>
</template>

<style scoped>
</style>
