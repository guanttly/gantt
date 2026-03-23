/**
 * 部门管理 - 类型定义
 */

/**
 * 部门信息
 */
export interface DepartmentInfo {
  /** 部门ID */
  id: string
  /** 组织ID */
  orgId: string
  /** 部门编码 */
  code: string
  /** 部门名称 */
  name: string
  /** 父部门ID */
  parentId?: string
  /** 父部门名称 */
  parentName?: string
  /** 层级（1为顶级） */
  level: number
  /** 部门路径 */
  path?: string
  /** 描述 */
  description?: string
  /** 部门经理ID */
  managerId?: string
  /** 部门经理姓名 */
  managerName?: string
  /** 排序 */
  sortOrder: number
  /** 是否启用 */
  isActive: boolean
  /** 员工数量 */
  employeeCount?: number
  /** 创建时间 */
  createdAt?: string
  /** 更新时间 */
  updatedAt?: string
}

/**
 * 部门树形结构
 */
export interface DepartmentTree extends DepartmentInfo {
  /** 子部门 */
  children?: DepartmentTree[]
}

/**
 * 部门列表查询参数
 */
export interface DepartmentListParams {
  /** 组织ID（必填） */
  orgId: string
  /** 父部门ID（用于查询子部门） */
  parentId?: string
  /** 关键字（名称或编码） */
  keyword?: string
  /** 是否启用 */
  isActive?: boolean
  /** 页码 */
  page?: number
  /** 每页数量 */
  size?: number
}

/**
 * 部门列表结果
 */
export interface DepartmentListResult {
  /** 数据列表 */
  departments: DepartmentInfo[]
  /** 总数 */
  total: number
  /** 当前页 */
  page: number
  /** 每页数量 */
  pageSize: number
}

/**
 * 创建部门请求
 */
export interface CreateDepartmentRequest {
  /** 组织ID */
  orgId: string
  /** 部门编码 */
  code: string
  /** 部门名称 */
  name: string
  /** 父部门ID */
  parentId?: string
  /** 描述 */
  description?: string
  /** 部门经理ID */
  managerId?: string
  /** 排序 */
  sortOrder?: number
}

/**
 * 更新部门请求
 */
export interface UpdateDepartmentRequest {
  /** 组织ID */
  orgId?: string
  /** 部门名称 */
  name?: string
  /** 描述 */
  description?: string
  /** 部门经理ID */
  managerId?: string
  /** 排序 */
  sortOrder?: number
  /** 是否启用 */
  isActive?: boolean
}

/**
 * 移动部门请求
 */
export interface MoveDepartmentRequest {
  /** 组织ID */
  orgId: string
  /** 部门ID */
  departmentId: string
  /** 新父部门ID */
  newParentId?: string
}
