<script setup lang="ts">
import type { FormInstance } from 'element-plus'
import { Md5 } from 'ts-md5'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { validatePassword } from '../../utils/validate'

const _emit = defineEmits(['close'])
const t = useI18n().t

const show = ref(false)
const ruleForm = ref<FormInstance>()
const defaultForm = {
  userOldPwd: '',
  userPwd: '',
  userPwdConfirm: '',
}
const form = ref({ ...defaultForm })
function showFunc() {
  show.value = true
  form.value = { ...defaultForm }
}
function validateNewPassword(rule: any, value: string, callback: any) {
  if (value !== form.value.userPwd) {
    callback(t('message.login.verify_confirm_pwd'))
    return false
  }
  else {
    callback()
    return true
  }
}

const rules = {
  userOldPwd: [
    { required: true, trigger: 'blur', message: '必填项' },
    { trigger: 'blur', validator: validatePassword },
  ],
  userPwd: [
    { required: true, trigger: 'blur', message: '必填项' },
    { trigger: 'blur', validator: validatePassword },
  ],
  userPwdConfirm: [
    { required: true, trigger: 'blur', message: '必填项' },
    { trigger: 'blur', validator: validateNewPassword },
  ],
}
function submit() {
  if (ruleForm.value) {
    ruleForm.value.validate((valid) => {
      if (valid) {
        // const params = {
        //   newPassword: Md5.hashStr(form.value.userPwd),
        //   oldPassword: Md5.hashStr(form.value.userOldPwd),
        // }
        // updatePassword(params).then((res: any) => {
        //   ElMessage({
        //     type: 'success',
        //     message: t('message.login.res_update_success'),
        //   })
        //   // layerDom.value && layerDom.value.close() // 关闭修改密码弹窗
        //   setTimeout(() => {
        //     userStore.logout() // 自动登出，让用户在登录页面用新密码重新登录
        //   }, 500)
        // })
      }
    })
  }
}
function close() {
  show.value = false
}
defineExpose({
  showFunc,
})
</script>

<template>
  <el-dialog v-model="show" title="修改密码" width="30%" center destroy-on-close>
    <el-form
      ref="ruleForm"
      :model="form"
      :rules="rules"
      label-width="120px"
      style="margin-right: 30px"
    >
      <el-form-item label="旧密码" prop="userOldPwd">
        <el-input
          v-model.trim="form.userOldPwd"
          placeholder="请输入"
          maxlength="20"
          show-password
        />
      </el-form-item>
      <el-form-item label="新密码" prop="userPwd">
        <el-input
          v-model.trim="form.userPwd"
          placeholder="请输入"
          maxlength="20"
          show-password
        />
      </el-form-item>
      <el-form-item
        label="确认新密码"
        prop="userPwdConfirm"
      >
        <el-input
          v-model.trim="form.userPwdConfirm"
          placeholder="请输入"
          maxlength="20"
          show-password
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <div>
        <el-button type="primary" @click="submit">
          确认
        </el-button>
        <el-button @click="close">
          取消
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped></style>
