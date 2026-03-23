declare namespace ScanType {
  /** 检查类型信息 */
  interface Info {
    id: string
    orgId: string
    code: string
    name: string
    description?: string
    isActive: boolean
    sortOrder: number
    createdAt: string
    updatedAt: string
  }

  /** 查询参数 */
  interface ListParams {
    orgId: string
    keyword?: string
    isActive?: boolean
    page?: number
    pageSize?: number
  }

  /** 列表响应 */
  interface ListResult {
    items: Info[]
    total: number
    page: number
    pageSize: number
  }

  /** 创建请求 */
  interface CreateRequest {
    orgId: string
    code: string
    name: string
    description?: string
    sortOrder?: number
  }

  /** 更新请求 */
  interface UpdateRequest {
    orgId: string
    name?: string
    description?: string
    sortOrder?: number
    isActive?: boolean
  }
}
