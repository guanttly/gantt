import { createApp } from 'vue'
import { accessOperation } from '@/utils/permission'
import App from './App.vue'
import { installRouter } from './router'
import store from './store'
import { installI18n } from './utils/i18n'

// 统一样式系统 - 包含所有样式定义
import './styles/styles.scss'

// Element Plus 样式
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/display.css'

// SVG 图标
import 'virtual:svg-icons-register'

const app = createApp(App)

app.use(store)
installRouter(app)
installI18n(app)
// 全局功能权限判断方法
app.config.globalProperties.$accessOperation = accessOperation
app.mount('#app')
