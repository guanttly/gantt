declare namespace ModalityRoom {
  /** 机房信息 */
  interface Info {
    id: string
    orgId: string
    code: string
    name: string
    description?: string
    location?: string
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
    location?: string
    sortOrder?: number
  }

  /** 更新请求 */
  interface UpdateRequest {
    orgId: string
    name?: string
    description?: string
    location?: string
    sortOrder?: number
    isActive?: boolean
  }

  /** 周检查量配置项 */
  interface WeeklyVolumeItem {
    weekday: number // 周几：0=周日,1=周一,...,6=周六
    timePeriodId: string
    timePeriodName?: string
    scanTypeId: string
    scanTypeName?: string
    volume: number
  }

  /** 周检查量列表响应 */
  interface WeeklyVolumeListResult {
    items: WeeklyVolumeItem[]
  }
}
