package service

// 共享辅助函数

// boolPtr 返回 bool 指针
func boolPtr(b bool) *bool {
	return &b
}

// stringPtr 返回 string 指针
func stringPtr(s string) *string {
	return &s
}
