package schedule_v2

// 导入工作流定义包，触发 init() 函数进行工作流注册
import (
	_ "jusha/agent/rostering/internal/workflow/schedule_v2/core"
	_ "jusha/agent/rostering/internal/workflow/schedule_v2/create"
)
