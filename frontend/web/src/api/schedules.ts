// 排班管理 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import type {
  AdjustAssignmentRequest,
  CreateScheduleRequest,
  MyScheduleAssignment,
  ScheduleAssignment,
  ScheduleChange,
  SchedulePlan,
  ScheduleSummary,
  ValidationResult,
} from '@/types/scheduling'
import client from './client'

// 重新导出类型，方便其他模块直接从 api/schedules 导入
export type {
  AdjustAssignmentRequest,
  CreateScheduleRequest,
  MyScheduleAssignment,
  ScheduleAssignment,
  ScheduleChange,
  SchedulePlan,
  ScheduleSummary,
  ValidationResult,
}

// ==================== 排班计划 CRUD ====================

/** 创建排班计划 */
export function createSchedule(data: CreateScheduleRequest) {
  return client.post<SchedulePlan>('/app/scheduling/plans/', data).then(r => r.data)
}

/** 排班列表 */
export function listSchedules(params?: ListParams) {
  return client.get<PaginatedResponse<SchedulePlan>>('/app/scheduling/plans/', { params }).then(r => r.data)
}

/** 获取排班详情 */
export function getSchedule(id: string) {
  return client.get<SchedulePlan>(`/app/scheduling/plans/${id}`).then(r => r.data)
}

/** 删除排班 */
export function deleteSchedule(id: string) {
  return client.delete(`/app/scheduling/plans/${id}`)
}

/** 生成排班（调用引擎） */
export function generateSchedule(id: string) {
  return client.put(`/app/scheduling/plans/${id}/execute`).then(r => r.data)
}

/** 获取排班分配 */
export function getAssignments(id: string) {
  return client.get<ScheduleAssignment[]>(`/app/scheduling/plans/${id}/assignments`).then(r => r.data)
}

/** 调整排班分配 */
export function adjustAssignments(id: string, data: AdjustAssignmentRequest) {
  return client.put(`/app/scheduling/plans/${id}/adjust`, data).then(r => r.data)
}

/** 验证排班 */
export function validateSchedule(id: string) {
  return client.post<ValidationResult>(`/app/scheduling/plans/${id}/validate`).then(r => r.data)
}

/** 发布排班 */
export function publishSchedule(id: string) {
  return client.put(`/app/scheduling/plans/${id}/publish`).then(r => r.data)
}

/** 获取排班变更记录 */
export function getScheduleChanges(id: string) {
  return client.get<ScheduleChange[]>(`/app/scheduling/plans/${id}/changes`).then(r => r.data)
}

/** 获取排班统计 */
export function getScheduleSummary(id: string) {
  return client.get<ScheduleSummary>(`/app/scheduling/plans/${id}/summary`).then(r => r.data)
}

// ==================== 旧排班分配 API（甘特图直接操作） ====================

export interface DirectAssignmentItem {
  employee_id: string
  shift_id: string
  date: string
  notes?: string
}

/** 批量分配排班（直接操作） */
export function batchAssignSchedule(data: { assignments: DirectAssignmentItem[] }) {
  return client.post('/scheduling/assignments/batch', data)
}

/** 查询日期范围内的排班 */
export function getScheduleByDateRange(params: { start_date: string, end_date: string }) {
  return client.get<ScheduleAssignment[]>('/scheduling/assignments', { params }).then(r => r.data)
}

/** 查询员工排班 */
export function getEmployeeSchedule(params: { employee_id: string, start_date: string, end_date: string }) {
  return client.get<ScheduleAssignment[]>('/scheduling/assignments/employee', { params }).then(r => r.data)
}

/** 查询当前登录员工的已发布排班 */
export function getMySchedule(params: { start_date: string, end_date: string }) {
  return client.get<MyScheduleAssignment[]>('/app/scheduling/assignments/self', { params }).then(r => r.data)
}

/** 删除排班分配 */
export function deleteScheduleAssignment(id: string) {
  return client.delete(`/scheduling/assignments/${id}`)
}

/** 批量删除排班 */
export function batchDeleteSchedule(data: { employee_ids: string[], dates: string[] }) {
  return client.post('/scheduling/assignments/batch/delete', data)
}

// ==================== Session 排班 API（AI 工作流） ====================

/** 获取最近会话 */
export function getRecentSession() {
  return client.get('/sessions/recent').then(r => r.data)
}

/** 创建会话 */
export function createSession(data: { agent_type: string, title?: string, locale?: string }) {
  return client.post('/sessions/', data).then(r => r.data)
}

/** 获取会话消息 */
export function getSessionMessages(sessionId: string) {
  return client.get(`/sessions/${sessionId}/messages`).then(r => r.data)
}

/** 获取会话快照 */
export function getSessionSnapshot(sessionId: string) {
  return client.get(`/sessions/${sessionId}/snapshot`).then(r => r.data)
}

/** 发送消息 */
export function sendSessionMessage(sessionId: string, content: string) {
  return client.post(`/sessions/${sessionId}/messages`, { content }).then(r => r.data)
}

/** 定稿会话 */
export function finalizeSession(sessionId: string) {
  return client.post(`/sessions/${sessionId}/finalize`).then(r => r.data)
}

/** 发送工作流命令 */
export function sendWorkflowCommand(sessionId: string, event: string, payload?: Record<string, unknown>) {
  return client.post(`/sessions/${sessionId}/workflow`, { event, payload }).then(r => r.data)
}

/** 列出对话历史 */
export function listConversations() {
  return client.get('/sessions/conversations').then(r => r.data)
}

/** 排班导出 */
export function exportSchedule(params: { start_date: string, end_date: string, format?: string }) {
  return client.get('/scheduling/export', { params, responseType: 'blob' })
}
