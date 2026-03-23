<script lang="ts" setup>
defineProps({
  type: {
    type: String,
    default: 'router-link',
  },
  to: {
    type: String,
    default: '',
  },
})
function linkProps(to: any) {
  return {
    to,
  }
}
</script>

<template>
  <component :is="type" v-bind="linkProps(to)">
    <slot />
  </component>
</template>

<style lang="">
</style>
