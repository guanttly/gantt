// 这里封装的是对接组内公共权限系统的相关方法
import { useUserStore } from '@/store/user'

// 将用户有权限的菜单列表加工一下，1、将菜单树状结构变成一维数组 2、提取菜单权限中的功能权限列表
export function handleMenuList(list: any, menuList: Array<any>, funcList: Array<any>) {
  const len = list.length
  for (let i = 0; i < len; i++) {
    const item = list[i]
    item.route = item.path
    if (item.children && item.children.length) {
      handleMenuList(item.children, menuList, funcList)
    }

    else {
      menuList.push(item)
      funcList.push(...item.meta.keyWords)
    }
  }
  const finalMenuList = JSON.parse(JSON.stringify(menuList))
  menuList.forEach((item: any) => {
    const parts = item.path.split('/').filter(Boolean)
    for (let i = 1; i < parts.length; i++) {
      const segment = `/${parts.slice(0, i).join('/')}`
      const isExist = finalMenuList.some((item: any) => item.path === segment)
      if (!isExist)
        finalMenuList.push({ name: 'add', path: segment })
    }
  })
  return { menuList: finalMenuList, funcList }
}
// 功能权限判断
export function accessOperation(operation: string): boolean {
  const userStore = useUserStore()
  let operations = []
  operations = userStore.functionList
  return operations.includes(operation)
}
