import type { AxiosRequestConfig } from 'axios'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { ACCESS_TOKEN_KEY } from '@/enums/cacheEnum'
import { baseApiUrl } from '@/enums/urlEnum'
import { Storage } from '@/utils/storage'

export interface RequestOptions {
  /** 当前接口权限, 不需要鉴权的接口请忽略， 格式：sys:user:add */
  permCode?: string
  /** 是否直接获取data，而忽略message等 */
  isGetDataDirectly?: boolean
  /** 请求成功是提示信息 */
  successMsg?: string
  /** 请求失败是提示信息 */
  errorMsg?: string
}

const UNKNOWN_ERROR = '未知错误，请重试'
const TIMEOUT_ERROR = '请求超时，请稍后重试'
const service = axios.create({
  timeout: 30000, // 默认30秒超时
})
// 这个是权限系统对应的业务平台id，先写死作为示例，后续实际开发可从配置中获取
const platId = 76584883768069

// 在请求头里配置token
service.interceptors.request.use(
  (config) => {
    const token = Storage.get(ACCESS_TOKEN_KEY)
    if (token && config.headers) {
      config.headers.platId = platId
      config.headers.Authorization = token
    }

    return config
  },
  (error) => {
    Promise.reject(error)
  },
)

service.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.errorCode && res.errorCode !== 0) {
      if (res.message && res.errorCode !== 401)
        ElMessage.error(res.message || UNKNOWN_ERROR)

      // Illegal token

      if (res.errorCode === 401 || res.code === 11001 || res.code === 11002) {
        window.localStorage.clear()
        window.location.reload()
      }

      // throw other
      const error = new Error(res.message || UNKNOWN_ERROR) as Error & { code: any }
      error.code = res.code
      return Promise.reject(error)
    }
    else {
      return res
    }
  },
  (error) => {
    // 处理超时错误
    if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
      ElMessage.error(TIMEOUT_ERROR)
      error.message = TIMEOUT_ERROR
      return Promise.reject(error)
    }
    
    // 处理 422 或者 500 的错误异常提示
    const errMsg = error?.response?.data?.message ?? UNKNOWN_ERROR
    if (error?.response?.data?.message)
      ElMessage.error(errMsg)
    error.message = errMsg
    return Promise.reject(error)
  },
)

export interface Response<T = any> {
  code: number
  message: string
  data: T
}

export type BaseResponse<T = any> = Promise<Response<T>>

/**
 * 将路径中重复的正斜杆替换成单个斜杆隔开的字符串
 * @param path 要处理的路径
 * @returns {string} 将/去重后的结果
 */
const uniqueSlash = (path: string) => path.replace(/(https?:\/)|(\/)+/g, '$1$2')
/**
 *
 * @param method - request methods
 * @param url - request url
 * @param data - request data or params
 */
export async function request<T = any>(config: AxiosRequestConfig, options: RequestOptions = {}): Promise<T> {
  try {
    const { successMsg, errorMsg, isGetDataDirectly = true } = options

    const fullUrl = `${baseApiUrl}${config.url}`
    config.url = uniqueSlash(fullUrl)
    const res = await service.request(config)
    successMsg && ElMessage.success(successMsg)
    errorMsg && ElMessage.error(errorMsg)
    return isGetDataDirectly ? res.data : (res as any)
  }
  catch (error: any) {
    return Promise.reject(error)
  }
}
