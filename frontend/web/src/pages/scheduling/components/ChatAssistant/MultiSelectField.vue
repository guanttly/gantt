<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { computed, ref, watch } from 'vue'

interface FieldOption {
  label: string
  value: any
  description?: string
  disabled?: boolean
  icon?: string
  extra?: Record<string, any>
}

interface Props {
  modelValue: any[]
  options: FieldOption[]
  label?: string
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  label: '请选择',
  placeholder: '搜索...',
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
}>()

const searchKeyword = ref('')
const selectedValues = ref<Set<any>>(new Set(props.modelValue || []))

// 过滤后的选项
const filteredOptions = computed(() => {
  if (!searchKeyword.value)
    return props.options

  const keyword = searchKeyword.value.toLowerCase()
  return props.options.filter(
    opt =>
      opt.label.toLowerCase().includes(keyword)
      || opt.description?.toLowerCase().includes(keyword),
  )
})

// 是否全选
const isAllSelected = computed(() => {
  const availableOptions = filteredOptions.value.filter(opt => !opt.disabled)
  if (availableOptions.length === 0)
    return false
  return availableOptions.every(opt => selectedValues.value.has(opt.value))
})

// 是否半选
const isIndeterminate = computed(() => {
  const availableOptions = filteredOptions.value.filter(opt => !opt.disabled)
  if (availableOptions.length === 0)
    return false
  const selectedCount = availableOptions.filter(opt => selectedValues.value.has(opt.value)).length
  return selectedCount > 0 && selectedCount < availableOptions.length
})

// 已选数量
const selectedCount = computed(() => selectedValues.value.size)

// 切换选项
function toggleOption(value: any) {
  if (selectedValues.value.has(value)) {
    selectedValues.value.delete(value)
  }
  else {
    selectedValues.value.add(value)
  }
  emitChange()
}

// 全选/取消全选
function toggleSelectAll() {
  const availableOptions = filteredOptions.value.filter(opt => !opt.disabled)
  if (isAllSelected.value) {
    // 取消全选
    availableOptions.forEach(opt => selectedValues.value.delete(opt.value))
  }
  else {
    // 全选
    availableOptions.forEach(opt => selectedValues.value.add(opt.value))
  }
  emitChange()
}

// 一键启用/禁用（快捷操作）
function toggleAll() {
  if (selectedValues.value.size === props.options.length) {
    selectedValues.value.clear()
  }
  else {
    props.options.forEach(opt => selectedValues.value.add(opt.value))
  }
  emitChange()
}

function emitChange() {
  emit('update:modelValue', Array.from(selectedValues.value))
}

// 监听外部值变化
watch(
  () => props.modelValue,
  (newVal) => {
    selectedValues.value = new Set(newVal || [])
  },
  { deep: true },
)
</script>

<template>
  <div class="multi-select-field">
    <!-- 头部工具栏 -->
    <div class="multi-select-header">
      <div class="header-left">
        <span class="label">{{ label }}</span>
        <span class="count-badge">已选 {{ selectedCount }}/{{ options.length }}</span>
      </div>
      <div class="header-right">
        <el-button link size="small" @click="toggleAll">
          {{ selectedValues.size === options.length ? '取消全选' : '全选' }}
        </el-button>
      </div>
    </div>

    <!-- 搜索框 -->
    <div class="search-box">
      <el-input
        v-model="searchKeyword"
        :prefix-icon="Search"
        :placeholder="placeholder"
        clearable
        size="small"
      />
    </div>

    <!-- 选项列表 -->
    <div class="options-list">
      <!-- 全选控制 -->
      <div v-if="filteredOptions.length > 0" class="select-all-item">
        <el-checkbox
          :model-value="isAllSelected"
          :indeterminate="isIndeterminate"
          @change="toggleSelectAll"
        >
          当前筛选结果全选 ({{ filteredOptions.length }})
        </el-checkbox>
      </div>

      <!-- 选项卡片 -->
      <div
        v-for="option in filteredOptions"
        :key="option.value"
        class="option-item"
        :class="{ 'is-selected': selectedValues.has(option.value), 'is-disabled': option.disabled }"
        @click="!option.disabled && toggleOption(option.value)"
      >
        <el-checkbox
          :model-value="selectedValues.has(option.value)"
          :disabled="option.disabled"
          @click.stop
          @change="toggleOption(option.value)"
        />
        <div class="option-content">
          <div class="option-main">
            <span v-if="option.icon" class="option-icon">{{ option.icon }}</span>
            <span class="option-label">{{ option.label }}</span>
            <el-tag v-if="option.extra?.type" size="small" type="info">
              {{ option.extra.type }}
            </el-tag>
          </div>
          <div v-if="option.description" class="option-description">
            {{ option.description }}
          </div>
          <!-- 额外信息展示 -->
          <div v-if="option.extra" class="option-extra">
            <span v-if="option.extra.color" class="color-badge" :style="{ backgroundColor: option.extra.color }" />
            <span v-if="option.extra.duration" class="duration-text">
              时长: {{ option.extra.duration }}分钟
            </span>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredOptions.length === 0" class="empty-state">
        <el-empty description="没有找到匹配的选项" :image-size="80" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.multi-select-field {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.multi-select-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.count-badge {
  padding: 2px 8px;
  font-size: 12px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 12px;
}

.search-box {
  padding: 0 12px;
}

.options-list {
  max-height: 400px;
  overflow-y: auto;
  padding: 0 12px;
}

.select-all-item {
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  margin-bottom: 8px;
}

.option-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.option-item:hover:not(.is-disabled) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.option-item.is-selected {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.option-item.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.option-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.option-icon {
  font-size: 18px;
}

.option-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.option-description {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.option-extra {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.color-badge {
  display: inline-block;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  border: 1px solid var(--el-border-color);
}

.duration-text {
  padding: 2px 6px;
  background: var(--el-fill-color);
  border-radius: 3px;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}
</style>
