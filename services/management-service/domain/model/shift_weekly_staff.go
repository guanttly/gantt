package model

// ShiftWeeklyStaff 班次周默认人数领域模型
type ShiftWeeklyStaff struct {
	ID         uint64 `json:"id"`
	ShiftID    string `json:"shiftId"`    // 班次ID
	Weekday    int    `json:"weekday"`    // 周几：0=周日,1=周一,...,6=周六
	StaffCount int    `json:"staffCount"` // 默认人数
}

// WeekdayStaff 单日人数配置（用于API响应）
type WeekdayStaff struct {
	Weekday     int    `json:"weekday"`     // 0-6
	WeekdayName string `json:"weekdayName"` // 周日/周一/.../周六
	StaffCount  int    `json:"staffCount"`  // 人数
	IsCustom    bool   `json:"isCustom"`    // 是否自定义（false则使用通用默认值）
}

// WeeklyStaffConfig 周默认人数配置（用于API响应）
type WeeklyStaffConfig struct {
	ShiftID      string         `json:"shiftId"`
	ShiftName    string         `json:"shiftName"`
	WeeklyConfig []WeekdayStaff `json:"weeklyConfig"` // 周配置（7项）
}

// GetWeekdayName 获取周几的中文名称
func GetWeekdayName(weekday int) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	if weekday >= 0 && weekday < len(names) {
		return names[weekday]
	}
	return ""
}

// SetWeeklyStaffRequest 设置周默认人数请求
type SetWeeklyStaffRequest struct {
	ShiftID      string `json:"shiftId"`
	WeeklyConfig []struct {
		Weekday    int `json:"weekday"`
		StaffCount int `json:"staffCount"`
	} `json:"weeklyConfig"`
}

// FormatWeeklyStaffSummary 格式化周人数摘要
// 返回格式："工作日X人/周末Y人"、"统一X人"、"未配置"
func FormatWeeklyStaffSummary(weeklyConfig []WeekdayStaff) string {
	if len(weeklyConfig) == 0 {
		return "未配置"
	}

	// 统计工作日和周末的人数
	weekdaySum := 0 // 工作日总人数 (周一到周五: 1-5)
	weekdayCount := 0
	weekendSum := 0 // 周末总人数 (周六周日: 0,6)
	weekendCount := 0
	allZero := true
	allSame := true
	firstCount := -1

	for _, wc := range weeklyConfig {
		if wc.StaffCount > 0 {
			allZero = false
		}
		if firstCount == -1 {
			firstCount = wc.StaffCount
		} else if wc.StaffCount != firstCount {
			allSame = false
		}

		// 周日(0)和周六(6)是周末
		if wc.Weekday == 0 || wc.Weekday == 6 {
			weekendSum += wc.StaffCount
			weekendCount++
		} else {
			weekdaySum += wc.StaffCount
			weekdayCount++
		}
	}

	if allZero {
		return "未配置"
	}

	if allSame && firstCount > 0 {
		return "统一" + itoa(firstCount) + "人"
	}

	// 计算工作日和周末的平均值（四舍五入）
	weekdayAvg := 0
	weekendAvg := 0
	if weekdayCount > 0 {
		weekdayAvg = (weekdaySum + weekdayCount/2) / weekdayCount
	}
	if weekendCount > 0 {
		weekendAvg = (weekendSum + weekendCount/2) / weekendCount
	}

	// 如果工作日和周末平均值相同
	if weekdayAvg == weekendAvg && weekdayAvg > 0 {
		return "统一" + itoa(weekdayAvg) + "人"
	}

	return "工作日" + itoa(weekdayAvg) + "人/周末" + itoa(weekendAvg) + "人"
}

// itoa 简单的整数转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
