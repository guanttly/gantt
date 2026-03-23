import { defineStore } from 'pinia'
import { login as loginApi, logout as logoutApi } from '@/api/login/index'
import { getMenuList, getRoleList } from '@/api/permissions'
import { ACCESS_TOKEN_KEY } from '@/enums/cacheEnum'
import router from '@/router'
import store from '@/store'
import { handleMenuList } from '@/utils/permission'
import { Storage } from '@/utils/storage'

interface UserState {
  token: string
  userInfo: Partial<API.User>
  menuList: any[]
  functionList: string[]
  roleList: any[]
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    token: Storage.get(ACCESS_TOKEN_KEY, null),
    userInfo: {},
    menuList: [],
    functionList: [],
    roleList: [],
  }),
  getters: {
    getToken(): string {
      return this.token
    },
  },
  actions: {
    /** 清空token及用户信息 */
    resetToken() {
      this.token = ''
      this.userInfo = {}
      this.menuList = []
      this.functionList = []
      this.roleList = []
      Storage.remove(ACCESS_TOKEN_KEY)
    },
    /** 登录成功保存token */
    setToken(token: string) {
      this.token = token ?? ''
      const ex = 10 * 60 * 60
      Storage.set(ACCESS_TOKEN_KEY, this.token, ex)
    },
    /** 获取用户信息 */
    async getUserInfo() {
      try {
        const authUserId = 0 // 这个值是调用业务系统获取登录用户对应的权限系统id
        const roleRes = await getRoleList(authUserId)
        this.roleList = roleRes?.filter((item: any) => item.flag) || []
        return this.userInfo
      }
      catch (error) {
        return Promise.reject(error)
      }
    },
    /** 登录 */
    async login(params: API.LoginParams) {
      try {
        const token = await loginApi(params)
        this.setToken(token)
        this.getUserInfo()
        return token
      }
      catch (error) {
        if (this.token)
          this.logout(false)
        return Promise.reject(error)
      }
    },
    /** 判断是否已登录 */
    isLogin() {
      return !!this.token
    },
    /** 登出 */
    async logout(goLogin = true) {
      await logoutApi()
      this.resetToken()
      // 退出登录时，清除定时器
      goLogin && router.replace('/login') // 登出后自动进入登录页
    },
    /** 获取用户权限信息 */
    async getAuthInfo() {
      try {
        const res = await getMenuList()
        const { menuList = [], funcList = [] } = handleMenuList(res || [], [], [])
        this.menuList = menuList
        this.functionList = funcList
        return this.menuList
      }
      catch (error) {
        return Promise.reject(error)
      }
    },
  },
})

// 在组件setup函数外使用
export function useUserStoreWithOut() {
  return useUserStore(store)
}
