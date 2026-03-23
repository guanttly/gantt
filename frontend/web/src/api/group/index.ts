// 分组管理模块相关的接口请求方法
import { request } from '@/utils/request'

const prefix = '/v1'

// ==================== 分组管理 ====================

/**
 * 查询分组列表
 */
export function getGroupList(params: Group.ListParams) {
  return request<Group.ListData>({
    url: `${prefix}/groups`,
    method: 'get',
    params,
  })
}

/**
 * 获取分组详情
 */
export function getGroupDetail(id: string, orgId: string) {
  return request<Group.GroupInfo>({
    url: `${prefix}/groups/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建分组
 */
export function createGroup(data: Group.CreateRequest) {
  return request<Group.GroupInfo>({
    url: `${prefix}/groups`,
    method: 'post',
    data,
  })
}

/**
 * 更新分组信息
 */
export function updateGroup(id: string, data: Group.UpdateRequest) {
  return request<Group.GroupInfo>({
    url: `${prefix}/groups/${id}`,
    method: 'put',
    data,
  })
}

/**
 * 删除分组
 */
export function deleteGroup(id: string, orgId: string) {
  return request({
    url: `${prefix}/groups/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/**
 * 获取分组树形结构
 */
export function getGroupTree(orgId: string) {
  return request<Group.TreeNode[]>({
    url: `${prefix}/groups/tree`,
    method: 'get',
    params: { orgId },
  })
}

// ==================== 分组成员管理 ====================

/**
 * 查询分组成员列表
 */
export function getGroupMembers(params: Group.MemberListParams) {
  return request<Group.MemberListData>({
    url: `${prefix}/groups/${params.groupId}/members`,
    method: 'get',
    params: {
      orgId: params.orgId,
      page: params.page,
      size: params.size,
    },
  })
}

/**
 * 添加分组成员
 */
export function addGroupMember(data: Group.AddMemberRequest) {
  return request({
    url: `${prefix}/groups/${data.groupId}/members`,
    method: 'post',
    data,
  })
}

/**
 * 移除分组成员
 */
export function removeGroupMember(data: Group.RemoveMemberRequest) {
  return request({
    url: `${prefix}/groups/${data.groupId}/members/${data.employeeId}`,
    method: 'delete',
    params: { orgId: data.orgId },
  })
}

/**
 * 批量添加分组成员
 */
export function batchAddGroupMembers(data: Group.BatchAddMembersRequest) {
  return request({
    url: `${prefix}/groups/${data.groupId}/members/batch`,
    method: 'post',
    data,
  })
}
