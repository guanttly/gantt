<script lang="ts" setup>
import type { FormInstance } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import router from '@/router'
import { useUserStore } from '@/store/user'
import { getRsaVal } from '@/utils/encrypt'
import storage from '@/utils/storage'

const userStore = useUserStore()
const systemName = ref('巨鲨AI智能排班系统')

interface FormState {
  loginName: string
  password: string
  remember: boolean
}
interface RememberedUsersState {
  users: Array<string>
}
const formState = reactive<FormState>({
  loginName: 'rieman-admin',
  password: 'jusha1996',
  remember: false,
})
const rules = {
  loginName: [{ required: true, trigger: 'blur', message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}
const rememberedUsersState = reactive<RememberedUsersState>({
  users: [],
})

const ruleFormRef = ref<FormInstance>()
const loading = ref(false)

function handleRememberUser() {
  const storedUsers = storage.get('rememberedUsers') || []
  const usersSet = new Set<string>(storedUsers)

  if (formState.remember) {
    if (formState.loginName)
      usersSet.add(formState.loginName)
  }
  else {
    usersSet.delete(formState.loginName)
  }
  storage.set('rememberedUsers', Array.from(usersSet))
  rememberedUsersState.users = Array.from(usersSet)
}

function getRememberedUsers() {
  const storedUsers = storage.get('rememberedUsers') || []
  rememberedUsersState.users = storedUsers
  if (storedUsers.length > 0 && !formState.loginName) {
    formState.loginName = storedUsers[0]
    formState.remember = true
  }
}
function setRememberedUser(loginName: string) {
  formState.loginName = loginName
  formState.password = ''
  formState.remember = true
  ruleFormRef.value?.clearValidate(['password'])
}

async function onClickLogin(formEl: FormInstance | undefined) {
  if (!formEl || loading.value)
    return
  await formEl.validate(async (valid: boolean) => {
    if (valid) {
      loading.value = true
      try {
        const password = await getRsaVal(formState.password)
        await userStore.login({
          username: formState.loginName,
          password,
        })
        handleRememberUser()
        // 登录成功后跳转
        router.push('/')
        ElMessage.success('登录成功')
      }
      catch (error: any) {
        ElMessage.error(error?.message || '登录失败')
      }
      finally {
        loading.value = false
      }
    }
  })
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter')
    onClickLogin(ruleFormRef.value)
}

onMounted(() => {
  getRememberedUsers()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <img class="system-logo" src="@/assets/images/logo.jpg">
        <div class="system-name">
          {{ systemName || '' }}
        </div>
      </div>
      <el-form
        ref="ruleFormRef"
        :model="formState"
        :rules="rules"
        label-width="0"
        size="large"
        :validate-on-rule-change="false"
        @submit.prevent="onClickLogin(ruleFormRef)"
      >
        <el-form-item prop="loginName">
          <el-input
            v-if="rememberedUsersState.users?.length === 0"
            v-model="formState.loginName"
            type="text"
            placeholder="请输入用户名"
            maxlength="20"
            :prefix-icon="User"
            size="large"
            @blur="() => ruleFormRef?.validateField('loginName')"
          />
          <el-dropdown v-else style="width: 100%">
            <el-input
              v-model="formState.loginName"
              placeholder="请输入用户名"
              maxlength="20"
              :prefix-icon="User"
              @blur="() => ruleFormRef?.validateField('loginName')"
            />
            <template #dropdown>
              <el-scrollbar max-height="192px">
                <el-dropdown-menu>
                  <el-dropdown-item v-for="(user, index) in rememberedUsersState.users" :key="`dw-${index}`" @click="setRememberedUser(user)">
                    {{ user }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </el-scrollbar>
            </template>
          </el-dropdown>
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model.trim="formState.password"
            type="password"
            placeholder="请输入密码" maxlength="20"
            :show-password="formState.password.length <= 20"
            :prefix-icon="Lock"
            size="large"
            @blur="() => ruleFormRef?.validateField('password')"
            @keyup.enter="onClickLogin(ruleFormRef)"
          />
        </el-form-item>
        <el-form-item prop="remember">
          <el-checkbox v-model="formState.remember" label="记住用户名" size="large" />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="login-button"
            :loading="loading"
            @click="onClickLogin(ruleFormRef)"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100%;
  // Change to radial gradient from center
  background: radial-gradient(circle at center, #f5f7fa 0%, #c3cfe2 100%);
}

.login-box {
  width: 400px;
  padding: 40px;
  background-color: var(--system-page-background, #fff); // Use theme page background
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.login-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 30px;

  .system-logo {
    width: 60px; // Adjust size as needed
    height: auto;
    margin-bottom: 15px;
  }

  .system-name {
    font-size: 24px;
    font-weight: 500;
    color: var(--system-page-color, #303133); // Use theme text color
  }
}

:deep(.el-input__wrapper) {
  padding: 5px 15px; // Adjust input padding
}

:deep(.el-input__prefix-inner) {
  font-size: 20px; // Adjust icon size
  margin-right: 8px;
}

:deep(.el-input .el-input__icon) {
  font-size: 18px; // Adjust clear/password icon size
}

:deep(.el-checkbox.el-checkbox--large) {
  color: var(--system-page-tip-color, rgba(0, 0, 0, 0.45)); // Use theme tip color
  height: auto; // Adjust checkbox vertical alignment if needed
}

.el-form-item {
  margin-bottom: 25px; // Increase spacing between form items
}

.login-button {
  width: 100%;
  // Use theme primary color for button background
  background-color: var(--system-primary-color, #409eff);
  border-color: var(--system-primary-color, #409eff);
  color: var(--system-primary-text-color, #fff); // Use theme primary text color

  &:hover {
    opacity: 0.9;
  }
}
</style>
