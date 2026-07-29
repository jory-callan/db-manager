package routes

import (
	"czwlinux.cloud/go-friday-starter/features/audit"
	"czwlinux.cloud/go-friday-starter/features/auth"
	"czwlinux.cloud/go-friday-starter/features/datasource"
	"czwlinux.cloud/go-friday-starter/features/dsrule"
	"czwlinux.cloud/go-friday-starter/features/escalation"
	"czwlinux.cloud/go-friday-starter/features/exec"
	"czwlinux.cloud/go-friday-starter/features/permission"
	"czwlinux.cloud/go-friday-starter/features/poolstats"
	"czwlinux.cloud/go-friday-starter/features/project"
	"czwlinux.cloud/go-friday-starter/features/role"
	rolepermission "czwlinux.cloud/go-friday-starter/features/role_permission"
	"czwlinux.cloud/go-friday-starter/features/snippet"
	"czwlinux.cloud/go-friday-starter/features/stats"
	"czwlinux.cloud/go-friday-starter/features/ticket"
	"czwlinux.cloud/go-friday-starter/features/user"
	userrole "czwlinux.cloud/go-friday-starter/features/user_role"
	"czwlinux.cloud/go-friday-starter/features/webhook"
	"czwlinux.cloud/go-friday-starter/global"
	"czwlinux.cloud/go-friday-starter/pkg/httpx"
	"github.com/labstack/echo/v4"
)

func Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	// ====== 公开接口 ======
	auth.RegisterRoutes(v1.Group("/auth"))

	// ====== 受保护接口（需登录）=======
	protected := v1.Group("")
	protected.Use(httpx.JWTAuth(global.JWT))

	// 当前用户
	protected.GET("/auth/me", user.NewHandler().Me)
	protected.PUT("/auth/profile", user.NewHandler().UpdateProfile)

	// ====== 平台管理（platform:user:manage）=======
	admin := protected.Group("", auth.RequirePermission("platform:user:manage"))

	// 用户管理
	admin.GET("/users", user.NewHandler().List)
	admin.POST("/users", user.NewHandler().Create)
	admin.GET("/users/:id", user.NewHandler().Get)
	admin.PUT("/users/:id", user.NewHandler().Update)
	admin.DELETE("/users/:id", user.NewHandler().Delete)
	admin.DELETE("/users/batch", user.NewHandler().BatchDelete)

	// 用户角色管理
	admin.GET("/users/:userId/roles", userrole.NewHandler().ListUserRoles)
	admin.POST("/users/:userId/roles", userrole.NewHandler().Assign)
	admin.DELETE("/users/:userId/roles/:roleId", userrole.NewHandler().Unassign)

	// 角色管理
	admin.GET("/roles", role.NewHandler().List)
	admin.GET("/roles/:id", role.NewHandler().Get)
	admin.POST("/roles", role.NewHandler().Create)
	admin.PUT("/roles/:id", role.NewHandler().Update)
	admin.DELETE("/roles/:id", role.NewHandler().Delete)

	// 权限码管理
	admin.GET("/permissions", permission.NewHandler().List)
	admin.GET("/permissions/:id", permission.NewHandler().Get)
	admin.POST("/permissions", permission.NewHandler().Create)
	admin.PUT("/permissions/:id", permission.NewHandler().Update)
	admin.DELETE("/permissions/:id", permission.NewHandler().Delete)
	admin.GET("/permissions/:id/bindings", permission.NewHandler().GetBindings)

	// 角色-权限码关联
	admin.GET("/roles/:roleId/permissions", rolepermission.NewHandler().ListRolePermissions)
	admin.POST("/roles/:roleId/permissions", rolepermission.NewHandler().Assign)
	admin.DELETE("/roles/:roleId/permissions/:permId", rolepermission.NewHandler().Unassign)

	// ====== 数据源管理 ======
	ds := protected.Group("/datasources")
	ds.Use(auth.RequirePermission("db:datasource:view"))
	ds.GET("", datasource.NewHandler().List)
	ds.GET("/:id", datasource.NewHandler().Get)

	dsAdd := protected.Group("/datasources", auth.RequirePermission("db:datasource:add"))
	dsAdd.POST("", datasource.NewHandler().Create)

	dsEdit := protected.Group("/datasources", auth.RequirePermission("db:datasource:edit"))
	dsEdit.PUT("/:id", datasource.NewHandler().Update)
	dsEdit.POST("/:id/test", datasource.NewHandler().Test)

	dsDelete := protected.Group("/datasources", auth.RequirePermission("db:datasource:delete"))
	dsDelete.DELETE("/:id", datasource.NewHandler().Delete)
	dsDelete.POST("/batch-delete", datasource.NewHandler().BatchDelete)

	protected.GET("/datasource-types", datasource.NewHandler().ListTypes)

	// ====== 项目管理 ======
	proj := protected.Group("/projects")
	proj.Use(auth.RequirePermission("db:project:view"))
	proj.GET("", project.NewHandler().List)
	proj.GET("/:id", project.NewHandler().Get)
	proj.GET("/:id/members", project.NewHandler().ListMembers)

	projAdd := protected.Group("/projects", auth.RequirePermission("db:project:add"))
	projAdd.POST("", project.NewHandler().Create)

	projEdit := protected.Group("/projects", auth.RequirePermission("db:project:edit"))
	projEdit.PUT("/:id", project.NewHandler().Update)
	projEdit.PUT("/:id/members", project.NewHandler().UpdateMembers)

	projDelete := protected.Group("/projects", auth.RequirePermission("db:project:delete"))
	projDelete.DELETE("/:id", project.NewHandler().Delete)
	projDelete.POST("/batch-delete", project.NewHandler().BatchDelete)

	// ====== SQL 规则管理 ======
	rule := protected.Group("/datasource-rules")
	rule.Use(auth.RequirePermission("db:rule:view"))
	rule.GET("", dsrule.NewHandler().List)

	ruleAdd := protected.Group("/datasource-rules", auth.RequirePermission("db:rule:add"))
	ruleAdd.POST("", dsrule.NewHandler().Create)

	ruleEdit := protected.Group("/datasource-rules", auth.RequirePermission("db:rule:edit"))
	ruleEdit.PUT("/:id", dsrule.NewHandler().Update)

	ruleDelete := protected.Group("/datasource-rules", auth.RequirePermission("db:rule:delete"))
	ruleDelete.DELETE("/:id", dsrule.NewHandler().Delete)
	ruleDelete.POST("/batch-delete", dsrule.NewHandler().BatchDelete)

	// ====== SQL 执行（业务特判，无权限码中间件）=======
	exec.RegisterRoutes(protected)

	// ====== 审计日志 ======
	protected.GET("/audits", audit.NewHandler().List, auth.RequirePermission("db:audit:view"))

	// ====== 工单（业务特判）=======
	ticket.RegisterRoutes(protected.Group("/tickets"))

	// ====== 提权（业务特判）=======
	escalation.RegisterRoutes(protected.Group("/escalations"))

	// ====== Webhook 管理 ======
	webhookGroup := protected.Group("/webhooks")
	webhookGroup.Use(auth.RequirePermission("db:webhook:manage"))
	webhook.RegisterRoutes(webhookGroup)

	// ====== Dashboard 统计（仅登录）=======
	stats.RegisterRoutes(protected)

	// ====== 连接池监控（仅登录）=======
	poolstats.RegisterRoutes(protected)

	// ====== 代码片段收藏（仅登录）=======
	snippet.RegisterRoutes(protected)

	// ====== 健康检查 ======
	v1.GET("/health", func(c echo.Context) error {
		return c.String(200, "ok")
	})
}
