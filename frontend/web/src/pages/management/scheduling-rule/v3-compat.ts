// ============================================================================
// V3 兼容层
// @deprecated V3: 此文件用于兼容 V3 前端旧枚举值，迁移完成后可移除
// ============================================================================

/**
 * V3 前端旧枚举值 → V4 后端值映射
 * @deprecated V3: 迁移完成后可移除
 */

// V3 规则类型 → V4 规则类型映射
export const V3_TO_V4_RULE_TYPE: Record<string, string> = {
  // V3 旧值 → V4 新值
  'max_shifts': 'maxCount',           // 最大班次数 → maxCount
  'consecutive_shifts': 'maxCount',   // 连续班次 → maxCount (需要结合 ConsecutiveMax)
  'rest_days': 'maxCount',            // 休息日 → maxCount (需要结合 MinRestDays)
  'forbidden_pattern': 'forbidden_day', // 禁止模式 → forbidden_day
  'preferred_pattern': 'preferred',   // 偏好模式 → preferred
  // V4 值保持不变
  'exclusive': 'exclusive',
  'combinable': 'combinable',
  'required_together': 'required_together',
  'periodic': 'periodic',
  'maxCount': 'maxCount',
  'forbidden_day': 'forbidden_day',
  'preferred': 'preferred',
}

// V3 应用范围 → V4 应用范围映射
export const V3_TO_V4_APPLY_SCOPE: Record<string, string> = {
  // V3 旧值 → V4 新值
  'global': 'global',     // 全局 → global (不变)
  'group': 'specific',    // 分组 → specific
  'employee': 'specific', // 员工 → specific
  'shift': 'specific',    // 班次 → specific
  // V4 值保持不变
  'specific': 'specific',
}

// V3 时间范围 → V4 时间范围映射
export const V3_TO_V4_TIME_SCOPE: Record<string, string> = {
  // V3 旧值 → V4 新值
  'daily': 'same_day',     // 每日 → same_day
  'weekly': 'same_week',   // 每周 → same_week
  'monthly': 'same_month', // 每月 → same_month
  'custom': 'custom',      // 自定义 → custom (不变)
  // V4 值保持不变
  'same_day': 'same_day',
  'same_week': 'same_week',
  'same_month': 'same_month',
}

// V4 规则类型 → V3 前端旧值映射（用于显示）
export const V4_TO_V3_RULE_TYPE: Record<string, string> = {
  'maxCount': 'max_shifts',
  'forbidden_day': 'forbidden_pattern',
  'preferred': 'preferred_pattern',
  'exclusive': 'exclusive',
  'combinable': 'combinable',
  'required_together': 'required_together',
  'periodic': 'periodic',
}

// V4 应用范围 → V3 前端旧值映射（用于显示）
export const V4_TO_V3_APPLY_SCOPE: Record<string, string> = {
  'global': 'global',
  'specific': 'employee', // 默认映射到 employee（因为 V3 中 specific 最常见的是 employee）
}

// V4 时间范围 → V3 前端旧值映射（用于显示）
export const V4_TO_V3_TIME_SCOPE: Record<string, string> = {
  'same_day': 'daily',
  'same_week': 'weekly',
  'same_month': 'monthly',
  'custom': 'custom',
}

/**
 * 将 V3 规则类型转换为 V4 规则类型
 * @deprecated V3: 迁移完成后可移除
 */
export function normalizeRuleType(v3Value: string): string {
  return V3_TO_V4_RULE_TYPE[v3Value] || v3Value
}

/**
 * 将 V3 应用范围转换为 V4 应用范围
 * @deprecated V3: 迁移完成后可移除
 */
export function normalizeApplyScope(v3Value: string): string {
  return V3_TO_V4_APPLY_SCOPE[v3Value] || v3Value
}

/**
 * 将 V3 时间范围转换为 V4 时间范围
 * @deprecated V3: 迁移完成后可移除
 */
export function normalizeTimeScope(v3Value: string): string {
  return V3_TO_V4_TIME_SCOPE[v3Value] || v3Value
}

/**
 * 将 V4 规则类型转换为 V3 前端旧值（用于向后兼容显示）
 * @deprecated V3: 迁移完成后可移除
 */
export function denormalizeRuleType(v4Value: string): string {
  return V4_TO_V3_RULE_TYPE[v4Value] || v4Value
}

/**
 * 将 V4 应用范围转换为 V3 前端旧值（用于向后兼容显示）
 * @deprecated V3: 迁移完成后可移除
 */
export function denormalizeApplyScope(v4Value: string): string {
  return V4_TO_V3_APPLY_SCOPE[v4Value] || v4Value
}

/**
 * 将 V4 时间范围转换为 V3 前端旧值（用于向后兼容显示）
 * @deprecated V3: 迁移完成后可移除
 */
export function denormalizeTimeScope(v4Value: string): string {
  return V4_TO_V3_TIME_SCOPE[v4Value] || v4Value
}
