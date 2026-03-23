package executor

import (
	"fmt"

	d_model "jusha/agent/rostering/domain/model"
)

// ============================================================
// 分批处理中的 ID 遮蔽/还原 统一入口
//
// 工程规约：所有发送给 LLM 的 prompt 中禁止出现 UUID，
// 必须在代码层面将 UUID 转为 shortID（staff_N / rule_N / shift_N），
// 解析 LLM 输出时再将 shortID 还原为 UUID。
//
// 本文件提供统一的遮蔽（mask）和还原（resolve）方法，
// 避免在各 batch 文件中重复编写 `if e.taskContext != nil` 分支。
// ============================================================

// ---------- 遮蔽：UUID → shortID（构建 prompt 时使用） ----------

// maskStaffID 将人员 UUID 转为 shortID（如 staff_1）
// fallbackIndex 用于 taskContext 不可用时的兜底编号
func (e *ProgressiveTaskExecutor) maskStaffID(uuid string, fallbackIndex int) string {
	if e.taskContext != nil {
		return e.taskContext.MaskStaffID(uuid)
	}
	return fmt.Sprintf("staff_%d", fallbackIndex)
}

// maskRuleID 将规则 UUID 转为 shortID（如 rule_1）
func (e *ProgressiveTaskExecutor) maskRuleID(uuid string, fallbackIndex int) string {
	if e.taskContext != nil {
		return e.taskContext.MaskRuleID(uuid)
	}
	return fmt.Sprintf("rule_%d", fallbackIndex)
}

// maskShiftID 将班次 UUID 转为 shortID（如 shift_1）
func (e *ProgressiveTaskExecutor) maskShiftID(uuid string, fallbackIndex int) string {
	if e.taskContext != nil {
		return e.taskContext.MaskShiftID(uuid)
	}
	return fmt.Sprintf("shift_%d", fallbackIndex)
}

// ---------- 还原：shortID → UUID（解析 LLM 输出时使用） ----------

// resolveStaffID 将 LLM 返回的 shortID/姓名 还原为 UUID
// 解析优先级：shortID映射 > 中文名映射 > 原样返回
func (e *ProgressiveTaskExecutor) resolveStaffID(idOrName string) string {
	if e.taskContext != nil {
		return e.taskContext.ResolveStaffID(idOrName)
	}
	return idOrName
}

// resolveRuleID 将 LLM 返回的 rule shortID 还原为 UUID
func (e *ProgressiveTaskExecutor) resolveRuleID(shortID string) string {
	if e.taskContext != nil && e.taskContext.RuleReverseMappings != nil {
		if uuid, ok := e.taskContext.RuleReverseMappings[shortID]; ok {
			return uuid
		}
	}
	return shortID
}

// ---------- 复合查找：从 LLM 输出中定位真实人员 ----------

// resolveStaffFromLLM 从 LLM 返回的 id+name 定位到真实的 Employee
// 查找优先级：
//  1. shortID → UUID → 在 staffList 中查找
//  2. 原始 id 直接在 staffList 中查找（兼容直接返回 UUID 的情况）
//  3. 按姓名在 staffList 中查找
func (e *ProgressiveTaskExecutor) resolveStaffFromLLM(
	llmID string,
	llmName string,
	staffList []*d_model.Employee,
) *d_model.Employee {
	// 1. 通过 resolveStaffID 将 shortID 还原为 UUID，再查找
	resolvedID := e.resolveStaffID(llmID)
	for _, staff := range staffList {
		if staff.ID == resolvedID || staff.EmployeeID == resolvedID {
			return staff
		}
	}

	// 2. 原始 id 直接查找（可能 LLM 直接返回了 UUID 或工号）
	if resolvedID != llmID {
		// 已经尝试过 resolved，跳过
	} else {
		for _, staff := range staffList {
			if staff.ID == llmID || staff.EmployeeID == llmID {
				return staff
			}
		}
	}

	// 3. 按姓名查找
	if llmName != "" {
		for _, staff := range staffList {
			if staff.Name == llmName {
				return staff
			}
		}
	}

	return nil
}
