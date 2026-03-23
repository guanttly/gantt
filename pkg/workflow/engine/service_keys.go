// Package engine 提供工作流引擎核心功能
package engine

import "fmt"

// ============================================================
// ServiceKey 服务键名类型定义
// 用于统一管理服务注册表中的键名，避免硬编码字符串
// ============================================================

// ServiceKey 服务键名类型
type ServiceKey string

// 服务键名常量定义
// 按业务领域分组，便于管理和维护
const (
	// ========== 数据服务 ==========

	// ServiceKeyRostering 排班数据服务
	// 提供排班、员工、班次、规则等数据的 CRUD 操作
	ServiceKeyRostering ServiceKey = "rosteringService"

	// ========== AI 服务 ==========

	// ServiceKeySchedulingAI 排班 AI 服务
	// 提供 AI 驱动的排班计划生成、Todo 执行等功能
	ServiceKeySchedulingAI ServiceKey = "schedulingAIService"

	// ========== 意图服务 ==========

	// ServiceKeyIntent 意图识别服务
	// 提供用户意图识别和工作流映射功能
	ServiceKeyIntent ServiceKey = "intentService"

	// ========== 渐进式排班服务 ==========

	// ServiceKeyProgressiveScheduling 渐进式排班服务
	// 提供渐进式排班任务计划和需求评估功能
	ServiceKeyProgressiveScheduling ServiceKey = "progressiveSchedulingService"

	// ServiceKeyToolCalling 工具调用服务
	// 用于 AI Function Calling
	ServiceKeyToolCalling ServiceKey = "toolcalling"

	// ServiceKeyProgressiveTask 渐进式任务执行服务
	// 用于通用渐进式任务编排
	ServiceKeyProgressiveTask ServiceKey = "progressive_task"

	// ServiceKeyAIFactory AI 工厂服务
	// 用于创建和管理 AI Provider 实例
	ServiceKeyAIFactory ServiceKey = "aiFactory"

	// ServiceKeyBridge WebSocket 桥接服务
	// 用于向前端广播实时消息
	ServiceKeyBridge ServiceKey = "bridge"
)

// String 返回服务键名的字符串表示
func (k ServiceKey) String() string {
	return string(k)
}

// ============================================================
// 泛型服务获取函数
// 提供类型安全的服务获取方式，避免运行时类型断言错误
// ============================================================

// GetService 从服务注册表获取指定类型的服务
// 返回服务实例和是否存在的标志
//
// 使用示例:
//
//	svc, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
//	if !ok {
//	    return fmt.Errorf("rostering service not found")
//	}
func GetService[T any](ctx Context, key ServiceKey) (T, bool) {
	var zero T
	svc, ok := ctx.Services().Get(key.String())
	if !ok {
		return zero, false
	}
	typed, ok := svc.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// MustGetService 从服务注册表获取指定类型的服务
// 如果服务不存在或类型不匹配，将 panic
//
// 使用示例:
//
//	svc := engine.MustGetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
//	result, err := svc.GenerateShiftTodoPlan(ctx, ...)
func MustGetService[T any](ctx Context, key ServiceKey) T {
	svc, ok := GetService[T](ctx, key)
	if !ok {
		panic(fmt.Sprintf("service not found or type mismatch: %s", key))
	}
	return svc
}

// ============================================================
// 服务注册辅助函数
// ============================================================

// RegisterService 向服务注册表注册服务
// 使用 ServiceKey 常量而非字符串，保证类型安全
//
// 使用示例:
//
//	engine.RegisterService(registry, engine.ServiceKeyRostering, rosteringService)
func RegisterService(registry IServiceRegistry, key ServiceKey, service any) {
	registry.Register(key.String(), service)
}
