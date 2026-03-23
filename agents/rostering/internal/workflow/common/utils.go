package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 辅助函数
// ============================================================

// formatList 格式化列表为逗号分隔的字符串
func FormatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) <= 3 {
		return fmt.Sprintf("%s", items)
	}
	// 超过3个，显示前3个加省略号
	return fmt.Sprintf("%s 等 %d 个", items[0], len(items))
}

// parsePayload 解析 payload 到指定结构
func ParsePayload(payload any, target any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return nil
}

// ============================================================
// 日期解析工具
// ============================================================

// parseDateRange 解析日期范围（支持自然语言）
// 输入示例: "下周", "本周", "2025-11-18到2025-11-24", "11月18日到24日"
// 返回: startDate, endDate (格式: YYYY-MM-DD)
func ParseDateRange(input string) (string, string, error) {
	if input == "" {
		return "", "", fmt.Errorf("empty date range")
	}

	input = strings.TrimSpace(input)

	// 1. 检查是否是标准格式: YYYY-MM-DD 到 YYYY-MM-DD
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}(到|至|~|-)\d{4}-\d{2}-\d{2}$`, input); matched {
		parts := regexp.MustCompile(`(到|至|~|-)`).Split(input, -1)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
		}
	}

	// 2. 处理自然语言
	now := time.Now()

	switch {
	case strings.Contains(input, "下周"):
		return GetNextWeekRange(now)
	case strings.Contains(input, "本周") || strings.Contains(input, "这周"):
		return GetCurrentWeekRange(now)
	case strings.Contains(input, "下个月"):
		return GetNextMonthRange(now)
	case strings.Contains(input, "本月") || strings.Contains(input, "这个月"):
		return GetCurrentMonthRange(now)
	case strings.Contains(input, "明天"):
		tomorrow := now.AddDate(0, 0, 1)
		date := tomorrow.Format("2006-01-02")
		return date, date, nil
	case strings.Contains(input, "今天"):
		date := now.Format("2006-01-02")
		return date, date, nil
	}

	// 3. 尝试解析相对日期: "X天后", "X周后"
	if days := ParseRelativeDays(input); days > 0 {
		start := now.AddDate(0, 0, days)
		end := start.AddDate(0, 0, 6) // 默认一周
		return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
	}

	return "", "", fmt.Errorf("unable to parse date range: %s", input)
}

// getNextWeekRange 获取下周的日期范围（周一到周日）
func GetNextWeekRange(now time.Time) (string, string, error) {
	// 计算下周一
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日
	}
	daysUntilNextMonday := 8 - weekday
	nextMonday := now.AddDate(0, 0, daysUntilNextMonday)

	// 下周日
	nextSunday := nextMonday.AddDate(0, 0, 6)

	return nextMonday.Format("2006-01-02"), nextSunday.Format("2006-01-02"), nil
}

// getCurrentWeekRange 获取本周的日期范围（周一到周日）
func GetCurrentWeekRange(now time.Time) (string, string, error) {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日
	}
	daysFromMonday := weekday - 1
	monday := now.AddDate(0, 0, -daysFromMonday)
	sunday := monday.AddDate(0, 0, 6)

	return monday.Format("2006-01-02"), sunday.Format("2006-01-02"), nil
}

// getNextMonthRange 获取下个月的日期范围（1号到月末）
func GetNextMonthRange(now time.Time) (string, string, error) {
	firstDay := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	return firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"), nil
}

// getCurrentMonthRange 获取本月的日期范围（1号到月末）
func GetCurrentMonthRange(now time.Time) (string, string, error) {
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	return firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"), nil
}

// parseRelativeDays 解析相对天数（X天后、X周后）
func ParseRelativeDays(input string) int {
	// 匹配 "3天后", "2周后"
	re := regexp.MustCompile(`(\d+)(天|周)后`)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 3 {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		if unit == "天" {
			return num
		} else if unit == "周" {
			return num * 7
		}
	}
	return 0
}

// parseHourFromTime 从时间字符串（如 "08:00"）中解析小时数
func ParseHourFromTime(timeStr string) int {
	parts := strings.Split(timeStr, ":")
	if len(parts) > 0 {
		if hour, err := strconv.Atoi(parts[0]); err == nil {
			return hour
		}
	}
	return 0
}

// ============================================================

// ============================================================
// 类型转换辅助函数
// ============================================================

// boolPtr 返回布尔值的指针
func BoolPtr(b bool) *bool {
	return &b
}
