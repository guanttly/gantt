package utils

import (
	"fmt"
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/workflow/session"
)

func BuildShiftSelectActions(
	shifts []*model.Shift,
	shiftScheduleCounts map[string]int,
	selectionEvent session.WorkflowEvent,
	cancelEvent session.WorkflowEvent,
) []session.WorkflowAction {
	// 构建班次选择消息
	actions := make([]session.WorkflowAction, 0, len(shifts)+1)

	for _, shift := range shifts {
		// 获取该班次的排班数量
		scheduleCount := 0
		if shiftScheduleCounts != nil {
			scheduleCount = shiftScheduleCounts[shift.ID]
		}

		// 根据是否有排班显示不同的标签
		var label string
		var style session.WorkflowActionStyle
		if scheduleCount > 0 {
			label = fmt.Sprintf("🏢 %s（%d条排班）", shift.Name, scheduleCount)
			style = session.ActionStylePrimary
		} else {
			label = fmt.Sprintf("🏢 %s（无排班）", shift.Name)
			style = session.ActionStyleSecondary
		}

		actions = append(actions, session.WorkflowAction{
			ID:    fmt.Sprintf("shift_%s", shift.ID),
			Event: selectionEvent,
			Label: label,
			Type:  session.ActionTypeWorkflow,
			Style: style,
			Payload: map[string]any{
				"shiftId": shift.ID,
			},
		})
	}

	// 添加取消按钮
	actions = append(actions, session.WorkflowAction{
		ID:    "cancel",
		Event: cancelEvent,
		Label: "取消",
		Type:  session.ActionTypeWorkflow,
		Style: session.ActionStyleDanger,
	})

	return actions
}

func BuildPeriodActions(startDate, endDate string, confirmEvent, cancelEvent session.WorkflowEvent) []session.WorkflowAction {
	return []session.WorkflowAction{
		{
			ID:    "confirm_period",
			Type:  session.ActionTypeWorkflow,
			Label: "确认周期",
			Event: confirmEvent,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"startDate": startDate,
				"endDate":   endDate,
			},
		},
		{
			ID:    "cancel",
			Type:  session.ActionTypeWorkflow,
			Label: "取消",
			Event: cancelEvent,
			Style: session.ActionStyleSecondary,
		},
	}
}

func BuildStaffCountActions(
	minCount, maxCount int,
	confirmEvent, cancelEvent session.WorkflowEvent,
) []session.WorkflowAction {
	actions := make([]session.WorkflowAction, 0, maxCount-minCount+2)

	for i := minCount; i <= maxCount; i++ {
		actions = append(actions, session.WorkflowAction{
			ID:    fmt.Sprintf("staff_count_%d", i),
			Type:  session.ActionTypeWorkflow,
			Label: fmt.Sprintf("%d 人", i),
			Event: confirmEvent,
			Style: session.ActionStylePrimary,
			Payload: map[string]any{
				"staffCount": i,
			},
		})
	}

	// 添加取消按钮
	actions = append(actions, session.WorkflowAction{
		ID:    "cancel",
		Type:  session.ActionTypeWorkflow,
		Label: "取消",
		Event: cancelEvent,
		Style: session.ActionStyleDanger,
	})

	return actions
}
