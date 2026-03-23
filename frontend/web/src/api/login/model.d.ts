declare namespace API {
  interface LoginParams {
    username: string
    password: string
  }
  /** 基本信息 */
  interface User {
    id: number
    userName: string // 用户名
    headPic: string // 头像
    sysUserId: number // 对应权限系统中的用户ID
    phone: string // 手机号
  }
}
