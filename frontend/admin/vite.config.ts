import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/admin/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': new URL('./src', import.meta.url).pathname,
    },
  },
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://192.168.119.128:8080',
        changeOrigin: true,
      },
    },
  },
})
