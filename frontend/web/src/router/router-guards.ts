import type { Router } from 'vue-router'
// import { useUserStore } from '@/store/user'

// const whiteList = ['/login', '/404'] // 不重定向白名单

// 创建所有的路由守卫
export function setupRouterGuard(router: Router) {
  createGlobalGuards(router)
  // 这里后续有需要的话还可以加其他更细化的守卫
}
// 创建全局路由守卫，主要是：页面的登录鉴权以及路由权限控制
export function createGlobalGuards(router: Router) {
  router.beforeEach(async (to, from, next) => {
    next()

    // const userStore = useUserStore()
    // const hasToken = userStore.isLogin()
    // const toPath = to.path
    // if (hasToken) {
    //   if (toPath === '/login') {
    //     next({ path: '/' }) // 若存在token,访问login重定向为 /（主页）
    //   }
    //   else {
    //     const hasInfo = userStore.menuList.length
    //     if (hasInfo) {
    //       const isAllow = userStore.menuList.some(menuItem => menuItem.path === toPath) // 判断to.path是否在用户拥有的菜单权限列表中
    //       if (!isAllow && !['/', '/404'].includes(toPath))
    //         next('/404')
    //       else
    //         next()
    //     }
    //     else {
    //       const token = userStore.token
    //       if (!token)
    //         window.location.reload()
    //       await userStore.getAuthInfo()
    //       const isAllow = userStore.menuList.some(menuItem => menuItem.path === toPath) // 判断to.path是否在用户拥有的菜单权限列表中
    //       // 获取系统所有路由
    //       const allRoutes = router.getRoutes()
    //       console.warn('系统所有路由', router.options.routes, allRoutes, userStore.menuList)

    //       // 三种情况不拦截路由，1、获取用户信息失败，放过让用户继续刷新页面；2、正常有权限的路由页面；3、根路由或者404页面
    //       if (router.options.routes.length === 2 || isAllow || ['/', '/404'].includes(toPath))
    //         next()
    //       else
    //         next('/404')
    //     }
    //   }
    // }
    // else {
    //   if (whiteList.includes(toPath)) {
    //     next() // 白名单中直接进入
    //   }
    //   else {
    //     next(`/login?redirect=${toPath}`)
    //   }
    // }
  })

  router.afterEach(() => {
  })

  router.onError((error) => {
    console.warn(error, '路由错误')
  })
}
