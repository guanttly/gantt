<script setup lang="ts">
// @ts-expect-error element-plus locale has no type declaration
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <RouterView />
  </el-config-provider>
</template>

<style scoped></style>
