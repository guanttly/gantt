// 根路由
export const RootRoute = [{
  path: '/',
  name: 'Root',
  component: () => import('@/layouts/modern.vue'),
  meta: {
    title: () => '首页',
    hideMenu: true,
    public: true,
  },
  redirect: '/workspace', // 重定向到工作台
}]
// 登录路由
export const LoginRoute = {
  path: '/login',
  name: 'Login',
  component: () => import('@/pages/login/index.vue'),
  meta: {
    title: () => '登录',
    noLayout: true,
    hideMenu: true,
    public: true,
  },
}
// import.meta.glob() 直接引入所有的路由模块 Vite独有的功能
const routeModules = (import.meta as any).glob('./modules/**/*.ts', { eager: true })
const routeModuleList: any[] = []
// 加入到路由集合中
Object.keys(routeModules).forEach((key) => {
  const mod = (routeModules as any)[key].default || {}
  const modList = Array.isArray(mod) ? [...mod] : [mod]
  routeModuleList.push(...modList)
})

export const allRoutes = [LoginRoute, ...RootRoute, ...routeModuleList]
