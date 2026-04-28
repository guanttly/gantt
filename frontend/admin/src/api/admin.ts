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

// ======== 应用配置 ========

export interface AIModelConfigView {
  provider: string
  model: string
  timeout_seconds: number
  temperature?: number | null
  max_tokens: number
  enabled: boolean
}

export interface WorkflowPosition {
  x: number
  y: number
}

export interface WorkflowEdgeView {
  from: string
  to: string
}

export interface WorkflowNodeView {
  key: string
  name: string
  kind: string
  description: string
  configurable: boolean
  position: WorkflowPosition
  model_config: AIModelConfigView
}

export interface WorkflowConfigView {
  key: string
  name: string
  version: string
  description: string
  enabled: boolean
  nodes: WorkflowNodeView[]
  edges: WorkflowEdgeView[]
}

export interface AppConfigView {
  code: string
  name: string
  description: string
  settings: Record<string, string>
  workflows: WorkflowConfigView[]
}

export function listAppConfigs() {
  return client.get<AppConfigView[]>('/admin/app-config/apps').then(r => r.data)
}

export function updateAppSettings(appCode: string, settings: Record<string, string>) {
  return client.put<Record<string, string>>(`/admin/app-config/apps/${appCode}/settings`, { settings }).then(r => r.data)
}

export function updateAppWorkflow(appCode: string, workflowKey: string, workflow: Partial<WorkflowConfigView>) {
  return client.put<WorkflowConfigView>(`/admin/app-config/apps/${appCode}/workflows/${workflowKey}`, workflow).then(r => r.data)
}

export function updateWorkflowNodeModel(appCode: string, workflowKey: string, nodeKey: string, model: AIModelConfigView) {
  return client.put<AIModelConfigView>(`/admin/app-config/apps/${appCode}/workflows/${workflowKey}/nodes/${nodeKey}/model`, model).then(r => r.data)
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
