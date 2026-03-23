import path from 'node:path'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'
import Inspect from 'vite-plugin-inspect'
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'
import Layouts from 'vite-plugin-vue-layouts'

export default defineConfig(({ mode }) => {
  // 加载当前模式下的环境变量
  // 第三个参数 '' 表示加载所有环境变量，不仅仅是 VITE_ 开头的
  // eslint-disable-next-line node/prefer-global/process
  const env = loadEnv(mode, process.cwd(), '')

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

      Layouts(),

      // https://github.com/antfu/vite-plugin-inspect
      Inspect(),
    ],
    server: {
      host: '0.0.0.0', // 允许通过 IP 地址访问
      port: 3000, // 根据需要修改端口
      cors: true, // 启用 CORS
      proxy: {
        // WebSocket代理 - 排班agent (rostering)
        '^/ws': {
          target: 'ws://192.168.119.128:9601',
          ws: true,
          changeOrigin: true,
        },
        // 排班 agent 的会话相关 API（需要代理到排班 agent）
        '^/api/sessions': {
          target: 'http://192.168.119.128:9601',
          changeOrigin: true,
        },
        // 管理服务的 API（统一使用 /api/management 前缀）
        '^/api/management': {
          target: 'http://192.168.119.128:9605',
          changeOrigin: true,
          rewrite: (path) => {
            return path.replace(/^\/api\/management/, '')
          },
        },
        // 其他 API 代理到管理服务 (management-service)
        '^/api': {
          target: 'http://192.168.119.128:9605',
          changeOrigin: true,
        },
      },
    },
  }
},
)
