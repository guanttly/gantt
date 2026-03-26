package websocket

// ScheduleProgressMessage 排班进度消息。
type ScheduleProgressMessage struct {
	Type       string  `json:"type"`
	ScheduleID string  `json:"schedule_id"`
	Step       string  `json:"step"`
	Progress   float64 `json:"progress"`
	Message    string  `json:"message"`
}

// ScheduleCompleteMessage 排班完成消息。
type ScheduleCompleteMessage struct {
	Type             string `json:"type"`
	ScheduleID       string `json:"schedule_id"`
	AssignmentsCount int    `json:"assignments_count"`
	ViolationsCount  int    `json:"violations_count"`
}

// NewProgressMessage 创建进度消息。
func NewProgressMessage(scheduleID, step string, progress float64, message string) *ScheduleProgressMessage {
	return &ScheduleProgressMessage{
		Type:       "schedule_progress",
		ScheduleID: scheduleID,
		Step:       step,
		Progress:   progress,
		Message:    message,
	}
}

// NewCompleteMessage 创建完成消息。
func NewCompleteMessage(scheduleID string, assignmentsCount, violationsCount int) *ScheduleCompleteMessage {
	return &ScheduleCompleteMessage{
		Type:             "schedule_complete",
		ScheduleID:       scheduleID,
		AssignmentsCount: assignmentsCount,
		ViolationsCount:  violationsCount,
	}
}
