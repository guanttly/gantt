// import { registeredphone } from '@/api/user'
/**
 * Created by PanJiaChen on 16/11/18.
 */

/**
 * @param {string} path
 * @returns {boolean}
 */
export function isExternal(path: string) {
  return /^(https?:|mailto:|tel:)/.test(path)
}

/**
 *
 * @param {string} str
 * @param {number} minLength
 * @param {number} maxLength
 */
export function validField(str: string, minLength: number, maxLength: number) {
  if (str == null) { return false }
  return str.length >= minLength && str.length <= maxLength
}

/**
 * 验证密码：只能输入6-20个字母、数字、下划线
 * @param {string} str
 */
export function validPwd(str: string) {
  let patrn = /^(\w){6,20}$/
  if (!patrn.exec(str))
    return false
  return true
}

/**
 * @param {string} str
 * @returns {boolean}
 */
export function validUsername(str: string) {
  return true
}

//
/**
 * 判断字符是否为空的方法
 * @param {*} obj
 */
export function checkStringIsEmpty(obj: any) {
  if (typeof obj === 'undefined' || obj == null || obj === '') {
    return true
  }
  else {
    return false
  }
}

/**
 * 验证用户名合法性
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validateLoginUserName(rule: any, value: string, callback: Function) {
  if (!validField(value, 2, 20)) {
    callback(new Error('账号长度应为2~20位'))
    return false
  }
  else if (value.includes(' ')) {
    callback(new Error('用户名不允许包含空格'))
  }
  else {
    callback()
    return true
  }
}

/**
 * 验证用户名合法性
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validateUserName(rule: any, value: string, callback: Function) {
  if (!validField(value, 2, 20)) {
    callback(new Error('用户名长度应为2~20位'))
    return false
  }
  else if (value.includes(' ')) {
    callback(new Error('用户名不允许包含空格'))
  }
  else {
    callback()
    return true
  }
}

/**
 * 验证密码合法性
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validatePassword(rule: any, value: string, callback: Function) {
  if (!validPwd(value)) {
    callback(new Error('只能输入6-20个字母、数字、下划线'))
    return false
  }
  else {
    callback()
    return true
  }
}

export function validatePhoneNumberReg(value: string) {
  return /^1[3-9]\d[0-9*]{4}\d{4}$/.test(value)
}

/**
 * 验证手机号是否合法
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validatePhone(rule: any, value: string, callback: Function) {
  const reg = /^1[3-9]\d{9}$/
  if (value === '' || value === undefined || value === null) {
    callback()
    return false
  }
  else {
    if ((!reg.test(value)) && value !== '') {
      callback(new Error('请输入正确的手机号码'))
      return false
    }
    else {
      callback()
      return true
    }
  }
}

/**
 * 验证码是否合法
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validatephoneVerification(rule: any, value: string, callback: Function) {
  if (value.length !== 6) {
    callback(new Error('请输入6位验证码'))
    return false
  }
  else {
    callback()
    return true
  }
}

/**
 * 验证邮箱是否合法
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validateEmail(rule: any, value: string, callback: Function) {
  const reg = /^[A-Z0-9\u4E00-\u9FA5]+@[\w-]+(\.[\w-]+)+$/i
  if (value === '' || value === undefined || value === null) {
    callback(new Error('邮箱不允许为空'))
    return false
  }
  else {
    if ((!reg.test(value)) && value !== '') {
      callback(new Error('请输入正确的邮箱'))
      return false
    }
    else {
      callback()
      return true
    }
  }
}

/**
 * 名称长度属性限制 2~20位
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validateNameAttrLength(rule: any, value: string, callback: Function) {
  if (!validField(value, 2, 20)) {
    callback(new Error('输入长度应为2~20位'))
    return false
  }
  else {
    callback()
    return true
  }
}

/**
 * 直播间名称长度限制 6~20位
 * @param {*} rule
 * @param {*} value
 * @param {*} callback
 */
export function validateLiveRoomLength(rule: any, value: string, callback: Function) {
  if (!validField(value, 6, 20)) {
    callback(new Error('直播间名称请限制为6~20位'))
    return false
  }
  else {
    callback()
    return true
  }
}

export function validateInputContent(rule: any, value: string, callback: Function) {
  if (!validField(value, 2, 20)) {
    callback(new Error('输入内容请控制在2~20位'))
    return false
  }
  else {
    callback()
    return true
  }
}

export function validateDemoContent(rule: any, value: string, callback: Function) {
  if (!validField(value, 0, 200)) {
    callback(new Error('输入内容请不要超过200位'))
    return false
  }
  else {
    callback()
    return true
  }
}
