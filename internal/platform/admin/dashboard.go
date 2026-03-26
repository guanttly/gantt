package admin

import (
	"context"
	"net/http"
	"time"

	"gantt-saas/internal/common/response"
	"gantt-saas/internal/tenant"

	"gorm.io/gorm"
)

// DashboardHandler 运营看板处理器。
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler 创建运营看板处理器。
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// DashboardData 运营看板响应数据。
type DashboardData struct {
	TotalOrgs             int64            `json:"total_orgs"`
	ActiveOrgs            int64            `json:"active_orgs"`
	TotalUsers            int64            `json:"total_users"`
	ActiveUsers30d        int64            `json:"active_users_30d"`
	SchedulesGenerated30d int64            `json:"schedules_generated_30d"`
	SubscriptionBreakdown map[string]int64 `json:"subscription_breakdown"`
}

// GetDashboard 获取运营看板数据。
// GET /api/v1/admin/dashboard
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := tenant.SkipTenantGuard(r.Context())

	data := DashboardData{
		SubscriptionBreakdown: make(map[string]int64),
	}

	h.db.WithContext(ctx).Table("org_nodes").Where("parent_id IS NULL AND code <> ?", "platform-root").Count(&data.TotalOrgs)
	h.db.WithContext(ctx).Table("org_nodes").Where("parent_id IS NULL AND code <> ? AND status = ?", "platform-root", "active").Count(&data.ActiveOrgs)
	h.db.WithContext(ctx).Table("platform_users").Count(&data.TotalUsers)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	h.db.WithContext(ctx).Table("platform_users").Where("updated_at >= ?", thirtyDaysAgo).Count(&data.ActiveUsers30d)
	h.countSchedules30d(ctx, thirtyDaysAgo, &data.SchedulesGenerated30d)
	h.countSubscriptionBreakdown(ctx, data.SubscriptionBreakdown)

	response.OK(w, data)
}

func (h *DashboardHandler) countSchedules30d(ctx context.Context, since time.Time, count *int64) {
	h.db.WithContext(ctx).Table("schedules").Where("created_at >= ?", since).Count(count)
}

func (h *DashboardHandler) countSubscriptionBreakdown(ctx context.Context, breakdown map[string]int64) {
	type planCount struct {
		Plan  string
		Count int64
	}
	var results []planCount
	h.db.WithContext(ctx).Table("subscriptions").Select("plan, COUNT(*) as count").Group("plan").Find(&results)
	for _, r := range results {
		breakdown[r.Plan] = r.Count
	}
	for _, plan := range []string{"free", "standard", "premium"} {
		if _, ok := breakdown[plan]; !ok {
			breakdown[plan] = 0
		}
	}
}
