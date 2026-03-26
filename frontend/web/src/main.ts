import { createApp } from 'vue'
import { ElLoadingDirective } from 'element-plus'
import App from './App.vue'
import { installRouter } from './router'
import pinia from './stores'
import { installI18n } from './utils/i18n'

// 统一样式系统
import './styles/styles.scss'

// Element Plus 样式
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/display.css'

// SVG 图标
import 'virtual:svg-icons-register'

const app = createApp(App)

app.use(pinia)
installRouter(app)
installI18n(app)
app.directive('loading', ElLoadingDirective)
app.mount('#app')
