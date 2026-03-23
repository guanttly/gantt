package model

// WeekdayStaffConfig 单日人数配置
type WeekdayStaffConfig struct {
	Weekday     int    `json:"weekday"`     // 0=周日,1=周一,...,6=周六
	WeekdayName string `json:"weekdayName"` // 周日/周一/.../周六
	StaffCount  int    `json:"staffCount"`  // 人数
	IsCustom    bool   `json:"isCustom"`    // 是否自定义配置
}

// ShiftWeeklyStaffConfig 班次周人数配置
type ShiftWeeklyStaffConfig struct {
	ShiftID      string               `json:"shiftId"`
	ShiftName    string               `json:"shiftName"`
	WeeklyConfig []WeekdayStaffConfig `json:"weeklyConfig"` // 7项配置
}

// SetWeeklyStaffRequest 设置周人数请求
type SetWeeklyStaffRequest struct {
	ShiftID      string `json:"shiftId"`
	WeeklyConfig []struct {
		Weekday    int `json:"weekday"`
		StaffCount int `json:"staffCount"`
	} `json:"weeklyConfig"`
}

// DailyStaffingResult 单日排班人数计算结果
type DailyStaffingResult struct {
	Weekday         int    `json:"weekday"`         // 周几：0=周日,1=周一,...,6=周六
	WeekdayName     string `json:"weekdayName"`     // 周几名称
	DailyVolume     int    `json:"dailyVolume"`     // 当日检查量
	CalculatedCount int    `json:"calculatedCount"` // 计算推荐人数
	CurrentCount    int    `json:"currentCount"`    // 当前配置人数
}

// StaffingCalculationPreview 排班人数计算预览
type StaffingCalculationPreview struct {
	ShiftID          string                 `json:"shiftId"`
	ShiftName        string                 `json:"shiftName"`
	TimePeriodID     string                 `json:"timePeriodId"`
	TimePeriodName   string                 `json:"timePeriodName"`
	TotalVolume      int                    `json:"totalVolume"`      // 总检查量
	DataDays         int                    `json:"dataDays"`         // 实际有数据的天数
	WeeklyVolume     int                    `json:"weeklyVolume"`     // 周检查量
	AvgReportLimit   int                    `json:"avgReportLimit"`   // 人均上限
	RoundingMode     string                 `json:"roundingMode"`     // 取整方式
	CalculatedCount  int                    `json:"calculatedCount"`  // 周总推荐人次
	DailyResults     []*DailyStaffingResult `json:"dailyResults"`     // 每日计算结果
	CalculationSteps string                 `json:"calculationSteps"` // 计算过程说明
}
