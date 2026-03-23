package adjust

import (
	"context"
	"fmt"

	"jusha/mcp/pkg/workflow/engine"
	"jusha/mcp/pkg/workflow/session"

	d_model "jusha/agent/rostering/domain/model"

	. "jusha/agent/rostering/internal/workflow/common"
	. "jusha/agent/rostering/internal/workflow/state/schedule"
)

// ========== 快捷操作处理 ==========

// actScheduleAdjustQuickSwap 快速换班操作
func actScheduleAdjustQuickSwap(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Processing quick swap")

	// 解析换班参数
	var req struct {
		Date   string `json:"date"`
		Staff1 string `json:"staff1"`
		Staff2 string `json:"staff2"`
	}

	if err := ParsePayload(payload, &req); err != nil {
		return fmt.Errorf("failed to parse swap payload: %w", err)
	}

	if req.Date == "" || req.Staff1 == "" || req.Staff2 == "" {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 请填写完整的换班信息（日期、员工 A、员工 B）。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		// 标记需要返回意图收集
		adjustCtx.ParsedIntent = nil
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 构建意图
	adjustCtx.ParsedIntent = &d_model.AdjustIntent{
		Type:    d_model.AdjustIntentSwap,
		Date:    req.Date,
		StaffA:  req.Staff1,
		StaffB:  req.Staff2,
		RawText: fmt.Sprintf("换班：%s 与 %s 在 %s 互换", req.Staff1, req.Staff2, req.Date),
	}

	adjustCtx.AddLog(fmt.Sprintf("快捷换班：%s 与 %s 在 %s 互换", req.Staff1, req.Staff2, req.Date))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterQuickSwap 快速换班后的流转
func actScheduleAdjustAfterQuickSwap(ctx context.Context, wctx engine.Context, payload any) error {
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

// actScheduleAdjustQuickReplace 快速替班操作
func actScheduleAdjustQuickReplace(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Processing quick replace")

	// 解析替班参数
	var req struct {
		Date     string `json:"date"`
		OldStaff string `json:"oldStaff"`
		NewStaff string `json:"newStaff"`
	}

	if err := ParsePayload(payload, &req); err != nil {
		return fmt.Errorf("failed to parse replace payload: %w", err)
	}

	if req.Date == "" || req.OldStaff == "" || req.NewStaff == "" {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 请填写完整的替班信息（日期、原员工、新员工）。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		// 标记需要返回意图收集
		adjustCtx.ParsedIntent = nil
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 构建意图
	adjustCtx.ParsedIntent = &d_model.AdjustIntent{
		Type:    d_model.AdjustIntentReplace,
		Date:    req.Date,
		StaffA:  req.OldStaff,
		StaffB:  req.NewStaff,
		RawText: fmt.Sprintf("替班：%s 替换 %s 在 %s 的班次", req.NewStaff, req.OldStaff, req.Date),
	}

	adjustCtx.AddLog(fmt.Sprintf("快捷替班：%s 替换 %s 在 %s", req.NewStaff, req.OldStaff, req.Date))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterQuickReplace 快速替班后的流转
func actScheduleAdjustAfterQuickReplace(ctx context.Context, wctx engine.Context, payload any) error {
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

// actScheduleAdjustQuickAdd 快速添加员工
func actScheduleAdjustQuickAdd(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Processing quick add")

	// 解析添加参数
	var req struct {
		Date  string `json:"date"`
		Staff string `json:"staff"`
	}

	if err := ParsePayload(payload, &req); err != nil {
		return fmt.Errorf("failed to parse add payload: %w", err)
	}

	if req.Date == "" || req.Staff == "" {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 请填写完整的添加信息（日期、员工）。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		// 标记需要返回意图收集
		adjustCtx.ParsedIntent = nil
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 构建意图
	adjustCtx.ParsedIntent = &d_model.AdjustIntent{
		Type:    d_model.AdjustIntentAdd,
		Date:    req.Date,
		StaffA:  req.Staff,
		RawText: fmt.Sprintf("添加员工：%s 在 %s", req.Staff, req.Date),
	}

	adjustCtx.AddLog(fmt.Sprintf("快捷添加：%s 在 %s", req.Staff, req.Date))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterQuickAdd 快速添加后的流转
func actScheduleAdjustAfterQuickAdd(ctx context.Context, wctx engine.Context, payload any) error {
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

// actScheduleAdjustQuickRemove 快速移除员工
func actScheduleAdjustQuickRemove(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	adjustCtx, err := GetScheduleAdjustContext(sess)
	if err != nil {
		return err
	}

	logger.Info("Processing quick remove")

	// 解析移除参数
	var req struct {
		Date  string `json:"date"`
		Staff string `json:"staff"`
	}

	if err := ParsePayload(payload, &req); err != nil {
		return fmt.Errorf("failed to parse remove payload: %w", err)
	}

	if req.Date == "" || req.Staff == "" {
		msg := session.Message{
			Role:    session.RoleAssistant,
			Content: "⚠️ 请填写完整的移除信息（日期、员工）。",
		}
		if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		// 标记需要返回意图收集
		adjustCtx.ParsedIntent = nil
		if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
			return fmt.Errorf("failed to save adjust context: %w", err)
		}
		return nil
	}

	// 构建意图
	adjustCtx.ParsedIntent = &d_model.AdjustIntent{
		Type:    d_model.AdjustIntentRemove,
		Date:    req.Date,
		StaffA:  req.Staff,
		RawText: fmt.Sprintf("移除员工：%s 在 %s", req.Staff, req.Date),
	}

	adjustCtx.AddLog(fmt.Sprintf("快捷移除：%s 在 %s", req.Staff, req.Date))

	// 保存上下文
	if _, err := wctx.SessionService().SetData(ctx, sess.ID, d_model.DataKeyScheduleAdjustContext, adjustCtx); err != nil {
		return fmt.Errorf("failed to save adjust context: %w", err)
	}

	return nil
}

// actScheduleAdjustAfterQuickRemove 快速移除后的流转
func actScheduleAdjustAfterQuickRemove(ctx context.Context, wctx engine.Context, payload any) error {
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

// actScheduleAdjustHandleCancel 处理取消操作
func actScheduleAdjustHandleCancel(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Info("Handling cancel")

	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: "❌ 已取消排班调整。",
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// 清除 WorkflowMeta - 使用 SetWorkflowMeta(nil) 来清除
	if _, err := wctx.SessionService().SetWorkflowMeta(ctx, sess.ID, nil); err != nil {
		logger.Warn("Failed to clear workflow meta", "error", err)
	}

	return nil
}

// actScheduleAdjustHandleError 处理异常
func actScheduleAdjustHandleError(ctx context.Context, wctx engine.Context, payload any) error {
	logger := wctx.Logger()
	sess := wctx.Session()

	logger.Error("Handling workflow error")

	var errorMessage string
	if payload != nil {
		var req struct {
			Error string `json:"error"`
		}
		if err := ParsePayload(payload, &req); err == nil {
			errorMessage = req.Error
		}
	}

	if errorMessage == "" {
		errorMessage = "工作流发生未知错误"
	}

	msg := session.Message{
		Role:    session.RoleAssistant,
		Content: fmt.Sprintf("⚠️ 处理过程中出现错误：%s\n\n请重试或联系管理员。", errorMessage),
		Actions: []session.WorkflowAction{
			{
				ID:    "retry",
				Event: session.WorkflowEvent(EventAdjustBackToIntent),
				Label: "🔄 重试",
				Type:  session.ActionTypeWorkflow,
				Style: session.ActionStylePrimary,
			},
			{
				ID:    "cancel",
				Event: session.WorkflowEvent(EventAdjustUserCancelled),
				Label: "❌ 取消",
				Type:  session.ActionTypeWorkflow,
				Style: session.ActionStyleSecondary,
			},
		},
	}

	if _, err := wctx.SessionService().AddMessage(ctx, sess.ID, msg); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}
