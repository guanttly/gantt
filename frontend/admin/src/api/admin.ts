// 平台管理 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import client from './client'

// ======== 仪表盘 ========

export interface AdminDashboard {
  total_orgs: number
  active_orgs: number
  total_users: number
  active_users_30d: number
  schedules_generated_30d: number
  subscription_breakdown: Record<string, number>
}

export function getAdminDashboard() {
  return client.get<AdminDashboard>('/admin/dashboard').then(r => r.data)
}

// ======== 系统配置 ========

export interface SystemConfig {
  [key: string]: string
}

export function getSystemConfig() {
  return client.get<SystemConfig>('/admin/system/config').then(r => r.data)
}

export function updateSystemConfig(data: SystemConfig) {
  return client.put('/admin/system/config', { configs: data }).then(r => r.data)
}

// ======== 订阅管理 ========

export interface Subscription {
  id: string
  org_node_id: string
  org_name: string
  plan: string
  status: 'active' | 'expired' | 'cancelled'
  start_date: string
  end_date: string
  created_at: string
  updated_at: string
}

export interface CreateSubscriptionRequest {
  org_node_id: string
  plan: string
  start_date: string
  end_date: string
  status?: Subscription['status']
}

export function listSubscriptions(params?: ListParams) {
  return client.get<PaginatedResponse<Subscription>>('/admin/subscriptions/', { params }).then(r => r.data)
}

export function createSubscription(data: CreateSubscriptionRequest) {
  return client.post<Subscription>('/admin/subscriptions/', data).then(r => r.data)
}

export function getSubscription(id: string) {
  return client.get<Subscription>(`/admin/subscriptions/${id}`).then(r => r.data)
}

export function updateSubscription(id: string, data: Partial<CreateSubscriptionRequest>) {
  return client.put<Subscription>(`/admin/subscriptions/${id}`, data).then(r => r.data)
}

// ======== 审计日志 ========

export interface AuditLog {
  id: string
  org_node_id?: string
  user_id: string
  username: string
  action: string
  resource_type: string
  resource_id?: string
  detail: Record<string, unknown> | null
  ip: string
  status_code: number
  created_at: string
}

export function listAuditLogs(params?: ListParams) {
  return client.get<PaginatedResponse<AuditLog>>('/admin/audit-logs/', { params }).then(r => r.data)
}
