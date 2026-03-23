<script setup lang="ts">
/**
 * SvgIcon 组件 - 统一管理项目中使用的 SVG 图标
 * 替代项目中的 emoji 使用
 */

defineProps<{
  name: string
  size?: string | number
  color?: string
}>()

// 使用 Vite 的 import.meta.glob 动态导入 SVG 资源
const svgModules = import.meta.glob('../assets/icons/svg/*.svg', { eager: true, as: 'url' })

function getIconUrl(iconName: string): string {
  const key = `../assets/icons/svg/${iconName}.svg`
  return (svgModules[key] as string) || ''
}
</script>

<template>
  <span
    class="svg-icon"
    :style="{
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      width: typeof size === 'number' ? `${size}px` : (size || '1em'),
      height: typeof size === 'number' ? `${size}px` : (size || '1em'),
      color: color || 'currentColor',
      flexShrink: 0,
    }"
  >
    <img
      :src="getIconUrl(name)"
      :alt="name"
      :style="{
        width: '100%',
        height: '100%',
        objectFit: 'contain',
      }"
    >
  </span>
</template>
