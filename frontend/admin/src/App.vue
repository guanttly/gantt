<script setup lang="ts">
import type { GlobalThemeOverrides } from 'naive-ui'
import { NConfigProvider, NDialogProvider, NGlobalStyle, NMessageProvider } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0f766e',
    primaryColorHover: '#0c655f',
    primaryColorPressed: '#0a5752',
    primaryColorSuppl: '#0f766e',
    borderRadius: '12px',
    borderRadiusSmall: '10px',
    fontFamily: "'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif",
  },
  Input: {
    heightLarge: '48px',
    borderHover: '1px solid #b9c6d3',
    borderFocus: '1px solid #0f766e',
    boxShadowFocus: '0 0 0 3px rgba(15, 118, 110, 0.12)',
  },
  Button: {
    heightLarge: '48px',
    borderRadiusLarge: '12px',
    fontWeight: '600',
  },
}
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-dialog-provider>
      <n-message-provider placement="top" :max="3">
        <n-global-style />
        <router-view />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>
