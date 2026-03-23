package schedule

// 导入工作流定义包，触发 init() 函数进行工作流注册
import (
	_ "jusha/agent/rostering/internal/workflow/schedule/adjust"
	_ "jusha/agent/rostering/internal/workflow/schedule/collectstaffcount"
	_ "jusha/agent/rostering/internal/workflow/schedule/confirmsave"
	_ "jusha/agent/rostering/internal/workflow/schedule/create"
	_ "jusha/agent/rostering/internal/workflow/schedule/infocollect"
	_ "jusha/agent/rostering/internal/workflow/schedule/regenerate"
)
