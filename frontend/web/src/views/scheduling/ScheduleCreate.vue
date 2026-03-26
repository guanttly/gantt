<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createSchedule } from '@/api/schedules'

const router = useRouter()
const loading = ref(false)

const form = reactive({
  name: '',
  start_date: '',
  end_date: '',
})

const rules = {
  name: [{ required: true, message: '请输入排班名称', trigger: 'blur' }],
  start_date: [{ required: true, message: '请选择开始日期', trigger: 'change' }],
  end_date: [{ required: true, message: '请选择结束日期', trigger: 'change' }],
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
    const res = await createSchedule(form)
    ElMessage.success('排班计划创建成功')
    await router.push(`/scheduling/${res.id}`)
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '创建失败')
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="create-page">
    <div class="create-card">
      <h2 class="page-title">
        创建排班计划
      </h2>

      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px" style="max-width: 500px">
        <el-form-item label="排班名称" prop="name">
          <el-input v-model="form.name" placeholder="如：2026年4月第1周排班" />
        </el-form-item>
        <el-form-item label="开始日期" prop="start_date">
          <el-date-picker v-model="form.start_date" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="结束日期" prop="end_date">
          <el-date-picker v-model="form.end_date" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleSubmit">
            创建
          </el-button>
          <el-button @click="router.back()">
            取消
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped>
.create-page {
  height: 100%;
  overflow-y: auto;
  padding: 32px;
}

.create-card {
  max-width: 640px;
  margin: 0 auto;
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  border: 1px solid #e5e7eb;
}

.page-title {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 32px;
  color: #1a1a2e;
}
</style>
