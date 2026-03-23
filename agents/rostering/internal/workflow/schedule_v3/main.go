// Package schedule_v3 排班工作流 V3
//
// 渐进式排班流程：LLM评估所有需求，生成渐进式任务计划，分阶段执行
package schedule_v3

import (
	_ "jusha/agent/rostering/internal/workflow/schedule_v3/core"  // 注册 core 子工作流
	_ "jusha/agent/rostering/internal/workflow/schedule_v3/create" // 注册 create 主工作流
)
