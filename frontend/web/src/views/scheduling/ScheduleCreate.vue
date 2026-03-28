<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createSchedule } from '@/api/schedules'
import { listGroups } from '@/api/groups'
import type { Group } from '@/api/groups'

const router = useRouter()
const loading = ref(false)
const groups = ref<Group[]>([])

const form = reactive({
  name: '',
  start_date: '',
  end_date: '',
  group_id: '' as string | undefined,
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
    const payload: Record<string, unknown> = {
      name: form.name,
      start_date: form.start_date,
      end_date: form.end_date,
    }
    if (form.group_id) {
      payload.group_id = form.group_id
    }
    const res = await createSchedule(payload as any)
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

async function loadGroups() {
  try {
    const result = await listGroups()
    groups.value = Array.isArray(result) ? result : result.items
  }
  catch {
    // 分组加载失败不阻断页面
  }
}

onMounted(loadGroups)
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
        <el-form-item label="排班分组">
          <el-select v-model="form.group_id" clearable placeholder="可选，按分组排班" style="width: 100%">
            <el-option v-for="g in groups" :key="g.id" :label="`${g.name}（${g.member_count}人）`" :value="g.id" />
          </el-select>
          <div v-if="form.group_id" style="font-size: 12px; color: #909399; margin-top: 4px;">
            选择分组后，仅该分组内的成员参与排班
          </div>
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
