<script setup lang="ts">
import SvgIcon from '@/components/svg-icon/index.vue'

defineProps({
  icon: {
    type: String,
    default: '',
  },
  title: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <SvgIcon v-if="icon" :name="icon" class="menu-icon" />
  <span class="title-text">{{ title }}</span>
</template>

<style lang="scss" scoped>
.title-text {
  margin-left: 8px;
}
.menu-icon {
  font-size: 24px;
  color: var(--system-menu-text-color);
}
</style>
