import type { App } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import { setupRouterGuard } from './router-guards'
import { allRoutes } from './routes'

const router = createRouter({
  routes: allRoutes,
  history: createWebHashHistory(),
})

export function installRouter(app: App<Element>) {
  app.use(router)
  setupRouterGuard(router)
}

export default router
