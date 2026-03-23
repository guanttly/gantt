package utils

import (
	"encoding/json"
	"reflect"
)

// DeepEqualValues 比较两个值是否相等，忽略指针地址差异
// 通过序列化为 JSON 再比较来实现值层面的比较
func DeepEqualValues(a, b any) bool {
	// 如果两个值都是 nil
	if a == nil && b == nil {
		return true
	}

	// 如果其中一个是 nil
	if a == nil || b == nil {
		return false
	}

	// 先尝试直接比较（对于基本类型和已实现 equal 的类型）
	if reflect.DeepEqual(a, b) {
		return true
	}

	// 对于复杂类型，通过 JSON 序列化比较值
	aJSON, errA := json.Marshal(a)
	if errA != nil {
		// 如果序列化失败，回退到 reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}

	bJSON, errB := json.Marshal(b)
	if errB != nil {
		// 如果序列化失败，回退到 reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}

	return string(aJSON) == string(bJSON)
}

// ConfigChanged 检查配置是否发生变化的辅助函数
func ConfigChanged(oldVal, newVal any, fieldName string) (bool, string) {
	if !DeepEqualValues(oldVal, newVal) {
		return true, fieldName + " config changed"
	}
	return false, ""
}
