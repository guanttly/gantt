<script setup>
defineProps({
  name: {
    type: String,
    required: true,
  },
})
</script>

<template>
  <svg aria-hidden="true" fill="currentColor" class="svg-external-icon svg-icon" style="outline: none !important;">
    <use :xlink:href="`#icon-${name}`" />
  </svg>
</template>

<style scoped>
.svg-icon {
  width: 1em;
  height: 1em;
  vertical-align: -0.15em;
  fill: currentColor;
  overflow: hidden;
}
:deep(svg:focus-visible) {
  outline: none !important;
}
</style>
