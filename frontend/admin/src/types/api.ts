// 通用 API 响应类型定义

/** 分页响应 */
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  size: number
}

/** 列表查询参数 */
export interface ListParams {
  page?: number
  size?: number
  [key: string]: unknown
}

/** 通用状态响应 */
export interface StatusResponse {
  status: string
}
