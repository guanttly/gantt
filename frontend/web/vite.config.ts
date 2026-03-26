import path from 'node:path'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { defineConfig, loadEnv } from 'vite'
import Inspect from 'vite-plugin-inspect'
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'
import Layouts from 'vite-plugin-vue-layouts'

export default defineConfig(({ mode }) => {
  // 加载当前模式下的环境变量
  // 仅加载 VITE_ 前缀变量，避免将保留变量注入前端环境
  // eslint-disable-next-line node/prefer-global/process
  const env = loadEnv(mode, process.cwd())

  // 支持通过环境变量配置 base 路径，默认为根路径
  const base = env.VITE_BASE_PATH || '/'

  return {
    base, // 配置应用的基础路径
    resolve: {
      alias: {
        '@/': `${path.resolve(__dirname, 'src')}/`,
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          // 只注入 Element Plus 变量覆盖
          additionalData: `@use "@/styles/element-vars.scss" as *;`,
        },
      },
    },
    plugins: [
      vue(),
      [createSvgIconsPlugin({
        // 指定需要缓存的图标文件夹
        // eslint-disable-next-line node/prefer-global/process
        iconDirs: [path.resolve(process.cwd(), './src/assets/icons/svg')],
        // 指定symbolId格式
        symbolId: 'icon-[name]',
      })],

      // https://github.com/intlify/bundle-tools/tree/main/packages/vite-plugin-vue-i18n
      VueI18nPlugin({
        runtimeOnly: true,
        compositionOnly: true,
        include: [path.resolve(__dirname, 'locales/**')],
      }),

      Components({
        dts: false,
        resolvers: [ElementPlusResolver({ importStyle: false })],
      }),

      Layouts(),

      // https://github.com/antfu/vite-plugin-inspect
      mode === 'development' ? Inspect() : null,
    ].filter(Boolean),
    server: {
      host: '0.0.0.0', // 允许通过 IP 地址访问
      port: 3000, // 根据需要修改端口
      cors: true, // 启用 CORS
      proxy: {
        // WebSocket 代理
        '^/ws': {
          target: 'ws://localhost:8080',
          ws: true,
          changeOrigin: true,
        },
        // API 代理
        '^/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
      // 仅剩 vis-timeline 与 xlsx 导出两个按需加载的专用大块，保留告警意义不大
      chunkSizeWarningLimit: 650,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('xlsx-js-style') || id.includes('/node_modules/xlsx/'))
              return 'xlsx-export'

            if (
              id.includes('/node_modules/vis-timeline/')
              || id.includes('/node_modules/vis-data/')
              || id.includes('/node_modules/vis-util/')
              || id.includes('/node_modules/moment/')
            ) {
              return 'vis-timeline'
            }

            if (id.includes('/node_modules/markdown-it/') || id.includes('/node_modules/highlight.js/'))
              return 'chat-renderer'

            return undefined
          },
        },
      },
    },
  }
},
)
