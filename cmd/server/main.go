package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"gantt-saas/internal/ai"
	aiapi "gantt-saas/internal/ai/api"
	"gantt-saas/internal/ai/chat"
	"gantt-saas/internal/ai/intent"
	"gantt-saas/internal/ai/quota"
	"gantt-saas/internal/ai/ruleparse"
	"gantt-saas/internal/auth"
	"gantt-saas/internal/core/approle"
	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/group"
	"gantt-saas/internal/core/leave"
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/schedule"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/infra/cache"
	appconfig "gantt-saas/internal/infra/config"
	"gantt-saas/internal/infra/database"
	"gantt-saas/internal/infra/observability"
	appserver "gantt-saas/internal/infra/server"
	appws "gantt-saas/internal/infra/websocket"
	"gantt-saas/internal/platform/admin"
	"gantt-saas/internal/platform/audit"
	"gantt-saas/internal/platform/subscription"
	"gantt-saas/internal/tenant"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type employeeAppRoleReaderAdapter struct {
	svc *approle.Service
}

func (a employeeAppRoleReaderAdapter) ListEmployeeRolesBatch(ctx context.Context, employeeIDs []string) (map[string][]employee.EmployeeAppRoleInfo, error) {
	items, err := a.svc.ListEmployeeRolesBatch(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]employee.EmployeeAppRoleInfo, len(items))
	for employeeID, roles := range items {
		mapped := make([]employee.EmployeeAppRoleInfo, 0, len(roles))
		for _, role := range roles {
			var grantedAt string
			if !role.GrantedAt.IsZero() {
				grantedAt = role.GrantedAt.Format(time.RFC3339)
			}
			var expiresAt *string
			if role.ExpiresAt != nil {
				formatted := role.ExpiresAt.Format(time.RFC3339)
				expiresAt = &formatted
			}
			mapped = append(mapped, employee.EmployeeAppRoleInfo{
				ID:              role.ID,
				EmployeeID:      role.EmployeeID,
				OrgNodeID:       role.OrgNodeID,
				OrgNodeName:     role.OrgNodeName,
				AppRole:         role.AppRole,
				Source:          role.Source,
				SourceGroupID:   role.SourceGroupID,
				SourceGroupName: role.SourceGroupName,
				GrantedBy:       role.GrantedBy,
				GrantedAt:       grantedAt,
				ExpiresAt:       expiresAt,
			})
		}
		result[employeeID] = mapped
	}
	return result, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := appconfig.Load()
	if err != nil {
		panic(fmt.Errorf("加载配置失败: %w", err))
	}

	logger, err := observability.NewLogger(&cfg.Log)
	if err != nil {
		panic(fmt.Errorf("初始化日志失败: %w", err))
	}
	defer func() { _ = logger.Sync() }()

	db, err := database.NewDB(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Warn("关闭数据库失败", zap.Error(err))
		}
	}()

	rdb, err := cache.NewRedis(&cfg.Redis, logger)
	if err != nil {
		logger.Fatal("初始化 Redis 失败", zap.Error(err))
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.Warn("关闭 Redis 失败", zap.Error(err))
		}
	}()

	hub := appws.NewHub()
	deps, err := initDependencies(ctx, db, rdb, logger, cfg, hub)
	if err != nil {
		logger.Fatal("初始化依赖失败", zap.Error(err))
	}

	srv := appserver.New(cfg, logger, db, rdb)
	registerRoutes(srv, deps)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	select {
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅关闭")
	case err := <-errCh:
		if err != nil {
			logger.Fatal("HTTP 服务异常退出", zap.Error(err))
		}
	}

	if err := srv.Shutdown(); err != nil {
		logger.Fatal("关闭 HTTP 服务失败", zap.Error(err))
	}
}

type appDependencies struct {
	jwtManager          *auth.JWTManager
	appJWTManager       *auth.JWTManager
	appRoleService      *approle.Service
	tenantHandler       *tenant.Handler
	authHandler         *auth.Handler
	appAuthHandler      *auth.AppHandler
	appRoleHandler      *approle.Handler
	employeeHandler     *employee.Handler
	groupHandler        *group.Handler
	shiftHandler        *shift.Handler
	leaveHandler        *leave.Handler
	ruleHandler         *rule.Handler
	scheduleHandler     *schedule.Handler
	aiHandler           *aiapi.Handler
	dashboardHandler    *admin.DashboardHandler
	organizationHandler *admin.OrganizationHandler
	platformUserHandler *admin.PlatformUserHandler
	systemConfigHandler *admin.SystemConfigHandler
	auditLogger         *audit.Logger
	auditHandler        *audit.Handler
	subscriptionHandler *subscription.Handler
}

func initDependencies(
	ctx context.Context,
	db *gorm.DB,
	rdb *redis.Client,
	logger *zap.Logger,
	cfg *appconfig.Config,
	hub appws.Broadcaster,
) (*appDependencies, error) {
	tenantRepo := tenant.NewRepository(db)
	tenantSvc := tenant.NewService(tenantRepo)
	tenantHandler := tenant.NewHandler(tenantSvc)

	authRepo := auth.NewRepository(db)
	jwtManager := auth.NewJWTManager(auth.JWTConfig{
		Secret:          cfg.JWT.Secret,
		Issuer:          cfg.JWT.Issuer,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
	appJWTManager := auth.NewJWTManager(auth.JWTConfig{
		Secret:          cfg.JWT.Secret,
		Issuer:          cfg.JWT.Issuer + ":app",
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
	authSvc := auth.NewService(authRepo, tenantRepo, jwtManager, rdb)
	authHandler := auth.NewHandler(authSvc)
	appAuthSvc := auth.NewAppService(authRepo, appJWTManager)
	appAuthHandler := auth.NewAppHandler(appAuthSvc)
	appRoleRepo := approle.NewRepository(db)
	appRoleSvc := approle.NewService(appRoleRepo, tenantRepo)
	appRoleHandler := approle.NewHandler(appRoleSvc)

	employeeRepo := employee.NewRepository(db)
	employeeSvc := employee.NewService(employeeRepo)
	employeeSvc.SetAppRoleCleaner(appRoleSvc)
	employeeSvc.SetAppRoleReader(employeeAppRoleReaderAdapter{svc: appRoleSvc})
	employeeSvc.SetOrgNodeResolver(tenantSvc)
	employeeHandler := employee.NewHandler(employeeSvc)

	groupRepo := group.NewRepository(db)
	groupSvc := group.NewService(groupRepo)
	groupSvc.SetAppRoleSyncer(appRoleSvc)
	groupSvc.SetOrgNodeResolver(tenantSvc)
	groupHandler := group.NewHandler(groupSvc)

	employeeSvc.SetGroupCleaner(groupSvc)

	shiftRepo := shift.NewRepository(db)
	shiftSvc := shift.NewService(shiftRepo)
	shiftSvc.SetOrgNodeResolver(tenantSvc)
	shiftHandler := shift.NewHandler(shiftSvc)

	leaveRepo := leave.NewRepository(db)
	leaveSvc := leave.NewService(leaveRepo)
	leaveHandler := leave.NewHandler(leaveSvc)

	ruleRepo := rule.NewRepository(db)
	ruleSvc := rule.NewService(ruleRepo, tenantRepo)
	ruleSvc.SetOrgNodeResolver(tenantSvc)
	ruleHandler := rule.NewHandler(ruleSvc)

	scheduleRepo := schedule.NewRepository(db)
	scheduleSvc := schedule.NewService(scheduleRepo, ruleSvc, shiftSvc, employeeRepo, leaveRepo, logger)
	scheduleSvc.SetBroadcaster(hub)
	scheduleSvc.SetGroupMemberProvider(groupSvc)
	scheduleSvc.SetOrgNodeResolver(tenantSvc)
	scheduleHandler := schedule.NewHandler(scheduleSvc)

	quotaRepo := quota.NewRepository(db)
	quotaMgr := quota.NewManager(quotaRepo, cfg.AI.Quota.DefaultMonthlyTokens, logger)
	factory := ai.NewFactory(&cfg.AI, logger)
	var aiHandler *aiapi.Handler
	if factory.HasProvider() {
		provider, err := factory.Default()
		if err != nil {
			return nil, fmt.Errorf("获取默认 AI provider 失败: %w", err)
		}
		scheduleSvc.SetAIProvider(provider)
		intentParser := intent.NewParser(provider, logger)
		chatHandler := chat.NewHandler(intentParser, provider, logger)
		ruleParser := ruleparse.NewParser(provider, logger)
		aiHandler = aiapi.NewHandler(chatHandler, ruleParser, quotaMgr, factory, logger)
	} else {
		logger.Info("AI provider 未启用，跳过 AI HTTP 路由注册")
	}

	auditRepo := audit.NewRepository(db)
	auditLogger := audit.NewLogger(auditRepo, logger)
	auditHandler := audit.NewHandler(auditRepo)

	subRepo := subscription.NewRepository(db)
	subSvc := subscription.NewService(subRepo)
	subHandler := subscription.NewHandler(subSvc)

	dashboardHandler := admin.NewDashboardHandler(db)
	organizationHandler := admin.NewOrganizationHandler(admin.NewOrganizationService(db))
	platformUserHandler := admin.NewPlatformUserHandler(admin.NewPlatformUserService(db))
	systemConfigHandler := admin.NewSystemConfigHandler(db)

	if err := autoMigrate(
		tenantRepo,
		authRepo,
		appRoleRepo,
		employeeRepo,
		groupRepo,
		shiftRepo,
		leaveRepo,
		ruleRepo,
		scheduleRepo,
		quotaRepo,
		auditRepo,
		subRepo,
		systemConfigHandler,
	); err != nil {
		return nil, err
	}
	startBackgroundWorkers(ctx, appRoleSvc, logger)

	if err := authSvc.SeedSystemRoles(ctx); err != nil {
		return nil, fmt.Errorf("初始化系统角色失败: %w", err)
	}

	if err := authSvc.SeedDefaultAdmin(ctx, cfg.Admin); err != nil {
		return nil, fmt.Errorf("初始化默认管理员失败: %w", err)
	}

	return &appDependencies{
		jwtManager:          jwtManager,
		appJWTManager:       appJWTManager,
		appRoleService:      appRoleSvc,
		tenantHandler:       tenantHandler,
		authHandler:         authHandler,
		appAuthHandler:      appAuthHandler,
		appRoleHandler:      appRoleHandler,
		employeeHandler:     employeeHandler,
		groupHandler:        groupHandler,
		shiftHandler:        shiftHandler,
		leaveHandler:        leaveHandler,
		ruleHandler:         ruleHandler,
		scheduleHandler:     scheduleHandler,
		aiHandler:           aiHandler,
		dashboardHandler:    dashboardHandler,
		organizationHandler: organizationHandler,
		platformUserHandler: platformUserHandler,
		systemConfigHandler: systemConfigHandler,
		auditLogger:         auditLogger,
		auditHandler:        auditHandler,
		subscriptionHandler: subHandler,
	}, nil
}

func startBackgroundWorkers(ctx context.Context, appRoleService *approle.Service, logger *zap.Logger) {
	if appRoleService != nil {
		go runExpiredAppRoleCleanupWorker(ctx, appRoleService, logger)
	}
}

func runExpiredAppRoleCleanupWorker(ctx context.Context, svc *approle.Service, logger *zap.Logger) {
	runCleanup := func() {
		rowsAffected, err := svc.CleanExpiredRoles(context.Background())
		if err != nil {
			logger.Error("清理过期应用角色失败", zap.Error(err))
			return
		}
		if rowsAffected > 0 {
			logger.Info("已清理过期应用角色", zap.Int64("rows_affected", rowsAffected))
		}
	}

	runCleanup()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCleanup()
		}
	}
}

func autoMigrate(
	tenantRepo *tenant.Repository,
	authRepo *auth.Repository,
	appRoleRepo *approle.Repository,
	employeeRepo *employee.Repository,
	groupRepo *group.Repository,
	shiftRepo *shift.Repository,
	leaveRepo *leave.Repository,
	ruleRepo *rule.Repository,
	scheduleRepo *schedule.Repository,
	quotaRepo *quota.Repository,
	auditRepo *audit.Repository,
	subRepo *subscription.Repository,
	systemConfigHandler *admin.SystemConfigHandler,
) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{name: "tenant", fn: tenantRepo.AutoMigrate},
		{name: "auth", fn: authRepo.AutoMigrate},
		{name: "app_role", fn: appRoleRepo.AutoMigrate},
		{name: "employee", fn: employeeRepo.AutoMigrate},
		{name: "group", fn: groupRepo.AutoMigrate},
		{name: "shift", fn: shiftRepo.AutoMigrate},
		{name: "leave", fn: leaveRepo.AutoMigrate},
		{name: "rule", fn: ruleRepo.AutoMigrate},
		{name: "schedule", fn: scheduleRepo.AutoMigrate},
		{name: "ai_quota", fn: quotaRepo.AutoMigrate},
		{name: "audit", fn: auditRepo.AutoMigrate},
		{name: "subscription", fn: subRepo.AutoMigrate},
		{name: "system_config", fn: systemConfigHandler.AutoMigrate},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("迁移 %s 失败: %w", step.name, err)
		}
	}

	return nil
}

func registerRoutes(srv *appserver.Server, deps *appDependencies) {
	srv.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("gantt-saas server is running"))
	})

	srv.Router.Route("/api/v1", func(r chi.Router) {
		auth.RegisterPublicRoutes(r, deps.authHandler)
		auth.RegisterAdminPublicRoutes(r, deps.authHandler)
		auth.RegisterAppPublicRoutes(r, deps.appAuthHandler)

		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware(deps.jwtManager))
			r.Use(tenant.Middleware())
			r.Use(audit.Middleware(deps.auditLogger))

			auth.RegisterProtectedRoutes(r, deps.authHandler)
			approle.RegisterUserRoutes(r, deps.appRoleHandler)
			approle.RegisterManagementRoutes(r, deps.appRoleHandler, deps.appRoleService)
			employee.RegisterRoutes(r, deps.employeeHandler)
			group.RegisterRoutes(r, deps.groupHandler, deps.appRoleService)
			shift.RegisterRoutes(r, deps.shiftHandler, deps.appRoleService)
			leave.RegisterRoutes(r, deps.leaveHandler, deps.appRoleService)
			rule.RegisterRoutes(r, deps.ruleHandler, deps.appRoleService)
			schedule.RegisterRoutes(r, deps.scheduleHandler, deps.appRoleService)
			if deps.aiHandler != nil {
				aiapi.RegisterRoutes(r, deps.aiHandler)
			}

			r.Group(func(r chi.Router) {
				r.Use(auth.RequirePermission("org:write"))
				tenant.RegisterRoutes(r, deps.tenantHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequirePermission("platform:admin"))
				auth.RegisterAdminRoutes(r, deps.authHandler)
				admin.RegisterRoutes(r, deps.dashboardHandler, deps.systemConfigHandler, deps.organizationHandler)
				subscription.RegisterRoutes(r, deps.subscriptionHandler)
				audit.RegisterRoutes(r, deps.auditHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequirePermission("platform:user:manage"))
				admin.RegisterPlatformUserRoutes(r, deps.platformUserHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(auth.RequirePermission("platform:manage_scope"))
				approle.RegisterPlatformRoutes(r, deps.appRoleHandler)
				employee.RegisterPlatformRoutes(r, deps.employeeHandler)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware(deps.appJWTManager))
			r.Use(tenant.Middleware())
			r.Use(audit.Middleware(deps.auditLogger))

			auth.RegisterAppProtectedRoutes(r, deps.appAuthHandler)
			approle.RegisterAppManagementRoutes(r, deps.appRoleHandler, deps.appRoleService)
			group.RegisterAppRoutes(r, deps.groupHandler, deps.appRoleService)
			shift.RegisterAppRoutes(r, deps.shiftHandler, deps.appRoleService)
			leave.RegisterAppRoutes(r, deps.leaveHandler, deps.appRoleService)
			rule.RegisterAppRoutes(r, deps.ruleHandler, deps.appRoleService)
			schedule.RegisterAppRoutes(r, deps.scheduleHandler, deps.appRoleService)
			approle.RegisterAppRoutes(r, deps.appRoleHandler)
			employee.RegisterAppRefRoutes(r, deps.employeeHandler)
			group.RegisterAppRefRoutes(r, deps.groupHandler)
			shift.RegisterAppRefRoutes(r, deps.shiftHandler)
			rule.RegisterAppRefRoutes(r, deps.ruleHandler)
		})
	})
}
