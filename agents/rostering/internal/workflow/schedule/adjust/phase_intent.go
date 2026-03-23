package adjust

import (
	"context"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ========== 阶段3: 收集意图 ==========

// actScheduleAdjustCollectIntent 收集用户调整意图
func actScheduleAdjustCollectIntent(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Collecting user intent for adjustment")

	// 构建意图收集消息，提供快捷操作
	actions := []session.WorkflowAction{
		{
			ID:    "quick_swap",
			Event: session.WorkflowEvent(EventAdjustQuickSwap),
			Type:  session.ActionTypeWorkflow,
			Label: "🔄 换班",
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "date",
					Label:       "日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择日期",
				},
				{
					Name:        "staff1",
					Label:       "员工 A",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择要换班的员工",
				},
				{
					Name:        "staff2",
					Label:       "员工 B",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择要换班的员工",
				},
			},
		},
		{
			ID:    "quick_replace",
			Event: session.WorkflowEvent(EventAdjustQuickReplace),
			Type:  session.ActionTypeWorkflow,
			Label: "👤 替换员工",
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "date",
					Label:       "日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择日期",
				},
				{
					Name:        "oldStaff",
					Label:       "原员工",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择要被替换的员工",
				},
				{
					Name:        "newStaff",
					Label:       "新员工",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择替换的员工",
				},
			},
		},
		{
			ID:    "quick_add",
			Event: session.WorkflowEvent(EventAdjustQuickAdd),
			Type:  session.ActionTypeWorkflow,
			Label: "➕ 添加员工",
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "date",
					Label:       "日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择日期",
				},
				{
					Name:        "staff",
					Label:       "员工",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择要添加的员工",
				},
			},
		},
		{
			ID:    "quick_remove",
			Event: session.WorkflowEvent(EventAdjustQuickRemove),
			Type:  session.ActionTypeWorkflow,
			Label: "➖ 移除员工",
			Style: session.ActionStyleSecondary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "date",
					Label:       "日期",
					Type:        session.FieldTypeDate,
					Required:    true,
					Placeholder: "选择日期",
				},
				{
					Name:        "staff",
					Label:       "员工",
					Type:        session.FieldTypeSelect,
					Required:    true,
					Options:     buildStaffOptions(adjustCtx.StaffList),
					Placeholder: "选择要移除的员工",
				},
			},
		},
		{
			ID:    "ai_adjust",
			Event: session.WorkflowEvent(EventAdjustIntentSubmitted),
			Type:  session.ActionTypeWorkflow,
			Label: "🤖 AI 调整",
			Style: session.ActionStylePrimary,
			Fields: []session.WorkflowActionField{
				{
					Name:        "intent",
					Label:       "调整意图",
					Type:        session.FieldTypeText,
					Required:    true,
					Placeholder: "例如：周三张三请假，需要找人替班",
				},
			},
		},
		{
			ID:    "regenerate",
			Event: session.WorkflowEvent(EventAdjustRegenerateStart),
			Type:  session.ActionTypeWorkflow,
			Label: "🔄 重排班次",
			Style: session.ActionStyleWarning,
		},
	}

	// 如果有历史记录，添加撤销按钮
	if adjustCtx.CanUndo() {
		actions = append(actions, session.WorkflowAction{
			ID:    "undo",
			Event: session.WorkflowEvent(EventAdjustUndo),
			Label: "↩️ 撤销",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStyleSecondary,
		})
	}

	// 如果可以重做，添加重做按钮
	if adjustCtx.CanRedo() {
		actions = append(actions, session.WorkflowAction{
			ID:    "redo",
			Event: session.WorkflowEvent(EventAdjustRedo),
			Label: "↪️ 重做",
			Type:  session.ActionTypeWorkflow,
			Style: session.ActionStyleSecondary,
		})
	}

	// 添加完成按钮
	actions = append(actions, session.WorkflowAction{
		ID:    "finish",
		Event: session.WorkflowEvent(EventAdjustFinish),
		Label: "✅ 完成调整",
		Type:  session.ActionTypeWorkflow,
		Style: session.ActionStylePrimary,
	})

	// 添加取消按钮
	actions = append(actions, session.WorkflowAction{
		ID:    "cancel",
		Event: session.WorkflowEvent(EventAdjustUserCancelled),
		Label: "❌ 取消",
		Type:  session.ActionTypeWorkflow,
		Style: session.ActionStyleSecondary,
	})

	// 使用 SetWorkflowMetaWithActions 同时设置消息和按钮
	if err := session.SetWorkflowMetaWithActions(ctx, wctx.SessionService(), sess.ID,
		"🛠️ 请选择调整操作，或直接告诉我您的需求：", actions); err != nil {
		return fmt.Errorf("failed to set workflow meta: %w", err)
	}

	return nil
}

// actScheduleAdjustAnalyzeIntent 分析用户输入的意图
func actScheduleAdjustAnalyzeIntent(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Analyzing user intent")

	// 从 payload 获取用户输入
	var userIntent string
	if payload != nil {
		var req struct {
			Intent string `json:"intent,omitempty"`
		}
		if err := ParsePayload(payload, &req); err == nil {
			userIntent = req.Intent
		}
	}

	// 如果 payload 中没有，尝试从会话消息获取
	if userIntent == "" {
		if len(sess.Messages) > 0 {
			lastMsg := sess.Messages[len(sess.Messages)-1]
			if lastMsg.Role == session.RoleUser {
				userIntent = lastMsg.Content
			}
		}
	}

	if userIntent == "" {
		// 没有意图输入，标记需要回退（AfterAct 会处理）
		adjustCtx.ParsedIntent = nil
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 保存用户原始输入
	adjustCtx.UserIntent = userIntent

	// 调用 SchedulingAIService 分析调整意图
	svc, ok := engine.GetService[d_service.ISchedulingAIService](wctx, engine.ServiceKeySchedulingAI)
	if !ok {
		logger.Warn("SchedulingAIService not found, using fallback parser")
		// 降级到简单规则解析
		adjustCtx.ParsedIntent = parseAdjustIntent(userIntent, adjustCtx)
	} else {
		// 调用 AI 分析调整意图
		adjustIntent, err := svc.AnalyzeAdjustIntent(ctx, userIntent, sess.Messages)
		if err != nil {
			logger.Error("Adjust intent analysis failed", "error", err)
			// 降级到简单规则解析
			adjustCtx.ParsedIntent = parseAdjustIntent(userIntent, adjustCtx)
		} else if adjustIntent != nil {
			// 直接使用返回的 AdjustIntent
			adjustCtx.ParsedIntent = adjustIntent
			logger.Info("Intent analyzed by AI",
				"type", adjustCtx.ParsedIntent.Type,
				"confidence", adjustCtx.ParsedIntent.Confidence)
		} else {
			// AI 返回空结果，降级到规则解析
			adjustCtx.ParsedIntent = parseAdjustIntent(userIntent, adjustCtx)
		}
	}

	if adjustCtx.ParsedIntent != nil {
		adjustCtx.AddLog(fmt.Sprintf("识别到调整意图: %s", adjustCtx.ParsedIntent.Type))
	}

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterAnalyzeIntent 意图分析后触发下一步
func actScheduleAdjustAfterAnalyzeIntent(ctx context.Context, wctx engine.Context, payload any) error {
	sess := wctx.Session()
	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	if adjustCtx.ParsedIntent == nil {
		return wctx.Send(ctx, EventAdjustBackToIntent, nil)
	}
	return wctx.Send(ctx, EventAdjustIntentAnalyzed, nil)
}

// parseAdjustIntent 解析调整意图（简单实现）
func parseAdjustIntent(text string, _ *d_model.ScheduleAdjustContext) *d_model.AdjustIntent {
	intent := &d_model.AdjustIntent{
		RawText: text,
	}

	// 简单的关键词匹配
	if containsAny(text, "换班", "调换", "交换") {
		intent.Type = d_model.AdjustIntentSwap
	} else if containsAny(text, "替班", "替换", "换人") {
		intent.Type = d_model.AdjustIntentReplace
	} else if containsAny(text, "加班", "增加", "添加") {
		intent.Type = d_model.AdjustIntentAdd
	} else if containsAny(text, "请假", "移除", "删除") {
		intent.Type = d_model.AdjustIntentRemove
	} else if containsAny(text, "调整", "修改", "更改") {
		intent.Type = d_model.AdjustIntentModify
	} else {
		intent.Type = d_model.AdjustIntentOther
	}

	return intent
}

// containsAny 检查文本是否包含任一关键词
func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if containsString(text, kw) {
			return true
		}
	}
	return false
}

// containsString 检查文本是否包含子串
func containsString(text, substr string) bool {
	return len(text) >= len(substr) && findString(text, substr) >= 0
}

// findString 查找子串位置
func findString(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// buildStaffOptions 构建员工选项列表
func buildStaffOptions(staffList []*d_model.Staff) []session.FieldOption {
	options := make([]session.FieldOption, 0, len(staffList))
	for _, staff := range staffList {
		options = append(options, session.FieldOption{
			Label: staff.Name,
			Value: staff.ID,
		})
	}
	return options
}
