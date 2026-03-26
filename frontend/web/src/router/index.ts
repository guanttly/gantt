import type { App } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { setupGuards } from './guards'
import { allRoutes } from './routes'

const router = createRouter({
  routes: allRoutes,
  history: createWebHistory(),
})

export function installRouter(app: App<Element>) {
  app.use(router)
  setupGuards(router)
}

export default router
