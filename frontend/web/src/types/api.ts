// 通用 API 响应类型定义

/** 标准 API 成功响应 */
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

/** 分页请求参数 */
export interface PaginationParams {
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/** 分页响应 */
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export type ListResponse<T> = PaginatedResponse<T> | T[]

/** 列表查询参数（分页+搜索） */
export interface ListParams extends PaginationParams {
  keyword?: string
  [key: string]: unknown
}

/** 通用 ID 响应 */
export interface IdResponse {
  id: string
}

/** 通用状态响应 */
export interface StatusResponse {
  status: string
}

/** API 错误响应 */
export interface ApiError {
  code: number
  message: string
  details?: Record<string, string[]>
}
