<script setup lang="ts">
import { LockClosedOutline, PersonOutline } from '@vicons/ionicons5'
import type { FormInst, FormRules } from 'naive-ui'
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NIcon, NInput, useMessage } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { RoleName } from '@/types/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const message = useMessage()

const loading = ref(false)
const form = reactive({ username: '', password: '' })
const formRef = ref<FormInst | null>(null)

const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: ['blur', 'input'] }],
  password: [{ required: true, message: '请输入密码', trigger: ['blur', 'input'] }],
}

async function handleLogin() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }

  loading.value = true
  try {
    const res = await auth.login(form.username, form.password)

    if (![RoleName.PlatformAdmin, RoleName.OrgAdmin, RoleName.DeptAdmin].includes(auth.currentRole)) {
      auth.logout()
      message.error('仅限平台管理账号登录')
      return
    }

    if (res.must_reset_pwd) {
      await router.push('/force-reset-password')
    }
    else {
      const redirect = (route.query.redirect as string) || '/'
      await router.push(redirect)
    }

    message.success('登录成功')
  }
  catch (e: any) {
    message.error(e?.response?.data?.message || '登录失败')
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-shell">
    <div class="auth-panel">
      <section class="auth-brand">
        <div class="auth-brand-content">
          <span class="auth-brand-badge">平台管理后台</span>
          <h2 class="auth-brand-title">平台统一管理</h2>
          <p class="auth-brand-lead">租户、订阅与系统配置</p>
          <p class="auth-brand-copy">
            这里集中处理平台级运营与治理事务。登录后可查看机构运营状态、维护系统配置，并管理跨租户能力。
          </p>
        </div>
      </section>

      <section class="auth-card">
        <div class="auth-card-header">
          <h1>登录管理后台</h1>
          <p>平台管理员、机构管理员、科室管理员可访问。首次登录如使用默认口令，系统会强制要求重置密码。</p>
        </div>

        <n-form
          ref="formRef"
          :model="form"
          :rules="rules"
          class="auth-form"
          @keyup.enter="handleLogin"
        >
          <n-form-item path="username">
            <n-input v-model:value="form.username" placeholder="用户名" size="large" clearable>
              <template #prefix>
                <n-icon :size="18" color="#94a3b8">
                  <person-outline />
                </n-icon>
              </template>
            </n-input>
          </n-form-item>
          <n-form-item path="password">
            <n-input v-model:value="form.password" type="password" placeholder="密码" size="large" show-password-on="click">
              <template #prefix>
                <n-icon :size="18" color="#94a3b8">
                  <lock-closed-outline />
                </n-icon>
              </template>
            </n-input>
          </n-form-item>
          <n-form-item>
            <n-button type="primary" size="large" :loading="loading" class="auth-submit" block @click="handleLogin">登录</n-button>
          </n-form-item>
        </n-form>
      </section>
    </div>
  </div>
</template>

<style scoped>
</style>
