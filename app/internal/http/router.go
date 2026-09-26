package httpserver

import (
	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/handlers"
	"life-base-api/app/internal/http/middleware"
)

// NewRouter creates a new Router with all routes configured.
func NewRouter(hdlrs *handlers.Registry, jwksURL, issuer string, allowedOrigins []string, auditWriter middleware.AuditWriter) *Router {
	engine := gin.New()
	// No trusted proxies: the app sits behind its hosting platform's own
	// proxy, and nothing here relies on ClientIP(), so don't trust a
	// client-supplied X-Forwarded-For.
	_ = engine.SetTrustedProxies(nil)
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(middleware.SecurityHeaders())
	engine.Use(middleware.BodyLimit())
	engine.Use(middleware.CORS(allowedOrigins))

	setupRoutes(engine, hdlrs, jwksURL, issuer, auditWriter)

	return &Router{engine: engine}
}

// setupRoutes configures all versioned routes for the application.
func setupRoutes(engine *gin.Engine, hdlrs *handlers.Registry, jwksURL, issuer string, auditWriter middleware.AuditWriter) {
	v1 := engine.Group("/v1")
	v1.Use(middleware.Auth(jwksURL, issuer))
	v1.Use(middleware.RateLimit())
	v1.Use(middleware.AuditLog(auditWriter))

	v1.GET("/me", hdlrs.Me.GetMe)
	v1.GET("/preferences", hdlrs.Preferences.Get)
	v1.PUT("/preferences", hdlrs.Preferences.Set)
	v1.POST("/push-subscriptions", hdlrs.Push.Subscribe)
	v1.DELETE("/push-subscriptions", hdlrs.Push.Unsubscribe)
	v1.GET("/export", hdlrs.Export.Handle)
	setupAdminRoutes(v1, hdlrs)
	setupCaptionRoutes(v1, hdlrs)
	setupDogRoutes(v1, hdlrs)
	setupSocialRoutes(v1, hdlrs)
	setupBikeRoutes(v1, hdlrs)
	setupTaskRoutes(v1, hdlrs)
	setupNoteRoutes(v1, hdlrs)
	setupAccountRoutes(v1, hdlrs)
	setupBudgetRoutes(v1, hdlrs)
	setupEnvelopeRoutes(v1, hdlrs)
	setupReminderRoutes(v1, hdlrs)
	setupCategoryRoutes(v1, hdlrs)
	setupReportRoutes(v1, hdlrs)
	setupSalaryProfileRoutes(v1, hdlrs)
	v1.POST("/import", hdlrs.Upload.Import)
	v1.POST("/transfers", hdlrs.Transfer.Create)
	v1.POST("/transfers/link", hdlrs.Transfer.Link)
	v1.POST("/transfers/link-existing", hdlrs.Transfer.LinkExisting)
	v1.GET("/transfers/matches", hdlrs.Transfer.GetMatches)
	v1.POST("/transfers/reconcile", hdlrs.Transfer.Reconcile)
	v1.PATCH("/transactions/:id/type", hdlrs.Transfer.UpdateTransactionType)
	v1.PATCH("/transactions/:id/note", hdlrs.Transaction.UpdateNote)
}

func setupAdminRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	admin := rg.Group("/admin")
	admin.Use(middleware.RequireRole("admin"))
	admin.GET("/users", hdlrs.Admin.ListMembers)
	admin.PATCH("/users/:id/role", hdlrs.Admin.SetMemberRole)
	admin.GET("/page-access", hdlrs.Admin.ListPageAccess)
	admin.PUT("/page-access", hdlrs.Admin.SetPageAccess)
	admin.GET("/audit-log", hdlrs.AuditLog.List)
}

func setupDogRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	dogs := rg.Group("/dogs")
	dogs.GET("", hdlrs.Dog.ListDogs)
	dogs.POST("", hdlrs.Dog.CreateDog)
	dogs.PATCH("/:id", hdlrs.Dog.UpdateDog)
	dogs.DELETE("/:id", hdlrs.Dog.DeleteDog)

	recipientTypes := rg.Group("/dog-recipient-types")
	recipientTypes.GET("", hdlrs.Dog.ListRecipientTypes)
	recipientTypes.POST("", hdlrs.Dog.CreateRecipientType)
	recipientTypes.PATCH("/:id", hdlrs.Dog.UpdateRecipientType)
	recipientTypes.DELETE("/:id", hdlrs.Dog.DeleteRecipientType)

	recipients := rg.Group("/dog-recipients")
	recipients.GET("", hdlrs.Dog.ListRecipients)
	recipients.POST("/portion", hdlrs.Dog.PortionBatch)
	recipients.POST("/:id/feed", hdlrs.Dog.MarkFed)

	bulkBags := rg.Group("/dog-bulk-bags")
	bulkBags.GET("", hdlrs.Dog.ListBulkBags)
	bulkBags.POST("", hdlrs.Dog.CreateBulkBag)
	bulkBags.DELETE("/:id", hdlrs.Dog.DeleteBulkBag)

	rg.GET("/dog-settings", hdlrs.Dog.GetSettings)
	rg.PUT("/dog-settings", hdlrs.Dog.SetSettings)
}

func setupCaptionRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	cats := rg.Group("/caption-categories")
	cats.GET("", hdlrs.Caption.ListCategories)
	cats.POST("", hdlrs.Caption.CreateCategory)
	cats.PATCH("/:id", hdlrs.Caption.UpdateCategory)
	cats.DELETE("/:id", hdlrs.Caption.DeleteCategory)

	templates := rg.Group("/caption-templates")
	templates.GET("", hdlrs.Caption.ListTemplates)
	templates.POST("", hdlrs.Caption.CreateTemplate)
	templates.PATCH("/:id", hdlrs.Caption.UpdateTemplate)
	templates.DELETE("/:id", hdlrs.Caption.DeleteTemplate)
	templates.GET("/:id/versions", hdlrs.Caption.ListVersions)
	templates.POST("/:id/versions", hdlrs.Caption.AddVersion)
	templates.POST("/:id/versions/:versionId/revert", hdlrs.Caption.RevertVersion)
}

func setupSocialRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	social := rg.Group("/social/posts")
	social.POST("", hdlrs.Social.Create)
	social.GET("", hdlrs.Social.List)
	social.DELETE("/:id", hdlrs.Social.Delete)
	social.POST("/:id/retry-instagram", hdlrs.Social.RetryInstagram)

	scheduled := rg.Group("/social/scheduled")
	scheduled.POST("", hdlrs.ScheduledPost.Create)
	scheduled.GET("", hdlrs.ScheduledPost.List)
	scheduled.DELETE("/:id", hdlrs.ScheduledPost.Delete)
	scheduled.POST("/check-now", hdlrs.ScheduledPost.CheckNow)
}

func setupBikeRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	bikes := rg.Group("/bikes")
	bikes.GET("", hdlrs.Bike.List)
	bikes.POST("", hdlrs.Bike.Create)
	bikes.GET("/:id", hdlrs.Bike.Get)
	bikes.PATCH("/:id", hdlrs.Bike.Update)
	bikes.DELETE("/:id", hdlrs.Bike.Delete)

	bikes.GET("/:id/fit-history", hdlrs.BikeFitHistory.ListByBike)
	bikes.POST("/:id/fit-history", hdlrs.BikeFitHistory.Create)
	bikes.DELETE("/:id/fit-history/:fitId", hdlrs.BikeFitHistory.Delete)

	bikes.GET("/:id/components", hdlrs.Component.ListByBike)
	bikes.POST("/:id/components", hdlrs.Component.Create)
	bikes.PATCH("/:id/components/:componentId", hdlrs.Component.Update)
	bikes.DELETE("/:id/components/:componentId", hdlrs.Component.Delete)
	bikes.POST("/:id/components/:componentId/replace", hdlrs.Component.Replace)
	bikes.GET("/:id/components/:componentId/history", hdlrs.Component.ListHistory)

	bikes.GET("/:id/service-logs", hdlrs.ServiceLog.ListByBike)
	bikes.POST("/:id/service-logs", hdlrs.ServiceLog.Create)
	bikes.DELETE("/:id/service-logs/:logId", hdlrs.ServiceLog.Delete)

	bikes.GET("/:id/maintenance-tasks", hdlrs.MaintenanceTask.ListByBike)
	bikes.POST("/:id/maintenance-tasks", hdlrs.MaintenanceTask.Create)
	bikes.PATCH("/:id/maintenance-tasks/:taskId", hdlrs.MaintenanceTask.Update)
	bikes.DELETE("/:id/maintenance-tasks/:taskId", hdlrs.MaintenanceTask.Delete)
	bikes.POST("/:id/maintenance-tasks/:taskId/complete", hdlrs.MaintenanceTask.Complete)

	bikes.GET("/:id/activities", hdlrs.Activity.ListByBike)
	bikes.POST("/:id/activities", hdlrs.Activity.Create)
	bikes.DELETE("/:id/activities/:activityId", hdlrs.Activity.Delete)

	bikes.GET("/strava/authorize", hdlrs.Strava.Authorize)
	bikes.GET("/strava/callback", hdlrs.Strava.Callback)
	bikes.GET("/strava/status", hdlrs.Strava.Status)
	bikes.DELETE("/strava", hdlrs.Strava.Disconnect)
	bikes.GET("/strava/activities", hdlrs.Strava.Activities)

	gear := rg.Group("/gear")
	gear.GET("", hdlrs.Gear.List)
	gear.POST("", hdlrs.Gear.Create)
	gear.PATCH("/:id", hdlrs.Gear.Update)
	gear.DELETE("/:id", hdlrs.Gear.Delete)

	bottles := rg.Group("/bottles")
	bottles.GET("", hdlrs.Bottle.List)
	bottles.POST("", hdlrs.Bottle.Create)
	bottles.PATCH("/:id", hdlrs.Bottle.Update)
	bottles.POST("/:id/clean", hdlrs.Bottle.MarkCleaned)
	bottles.DELETE("/:id", hdlrs.Bottle.Delete)

	supplies := rg.Group("/supplies")
	supplies.GET("", hdlrs.Supply.List)
	supplies.POST("", hdlrs.Supply.Create)
	supplies.PATCH("/:id", hdlrs.Supply.Update)
	supplies.POST("/:id/deplete", hdlrs.Supply.Deplete)
	supplies.DELETE("/:id", hdlrs.Supply.Delete)
	rg.GET("/supply-history", hdlrs.Supply.ListHistory)
}

func setupTaskRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	tasks := rg.Group("/tasks")
	tasks.GET("", hdlrs.Task.List)
	tasks.POST("", hdlrs.Task.Create)
	tasks.PATCH("/:id", hdlrs.Task.Update)
	tasks.DELETE("/:id", hdlrs.Task.Delete)
	tasks.POST("/:id/complete", hdlrs.Task.Complete)
}

func setupNoteRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	notes := rg.Group("/notes")
	notes.GET("", hdlrs.Note.List)
	notes.POST("", hdlrs.Note.Create)
	notes.GET("/:id", hdlrs.Note.Get)
	notes.PATCH("/:id", hdlrs.Note.Update)
	notes.DELETE("/:id", hdlrs.Note.Delete)
}

func setupEnvelopeRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	env := rg.Group("/envelopes")
	env.GET("", hdlrs.Envelope.List)
	env.POST("", hdlrs.Envelope.Create)
	env.PATCH("/:id", hdlrs.Envelope.Update)
	env.DELETE("/:id", hdlrs.Envelope.Delete)
	env.POST("/:id/contribute", hdlrs.Envelope.Contribute)
}

func setupBudgetRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	budgets := rg.Group("/budgets")
	budgets.GET("", hdlrs.Budget.List)
	budgets.POST("", hdlrs.Budget.Create)
	budgets.PATCH("/:id", hdlrs.Budget.Update)
	budgets.DELETE("/:id", hdlrs.Budget.Delete)
	budgets.POST("/:id/acknowledge", hdlrs.Budget.Acknowledge)
}

func setupReminderRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	reminders := rg.Group("/reminders")
	reminders.GET("", hdlrs.Reminder.List)
	reminders.POST("", hdlrs.Reminder.Create)
	reminders.PATCH("/:id", hdlrs.Reminder.Update)
	reminders.DELETE("/:id", hdlrs.Reminder.Delete)
	reminders.POST("/:id/complete", hdlrs.Reminder.Complete)
	reminders.POST("/:id/link", hdlrs.Reminder.Link)
}

func setupAccountRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	accounts := rg.Group("/accounts")
	accounts.GET("", hdlrs.Account.List)
	accounts.POST("", hdlrs.Account.Create)
	accounts.GET("/:id", hdlrs.Account.Get)
	accounts.PATCH("/:id", hdlrs.Account.Update)
	accounts.DELETE("/:id", hdlrs.Account.Delete)
	accounts.GET("/:id/transactions", hdlrs.Transaction.ListByAccount)
	accounts.POST("/:id/transactions", hdlrs.Transaction.Create)
	accounts.GET("/:id/envelopes", hdlrs.Envelope.ListByAccount)
	accounts.GET("/:id/reminders", hdlrs.Reminder.ListByAccount)
	accounts.GET("/:id/rule-exceptions", hdlrs.RuleException.ListByAccount)
	accounts.POST("/:id/rule-exceptions", hdlrs.RuleException.Disable)
	accounts.DELETE("/:id/rule-exceptions/:rule_id", hdlrs.RuleException.Enable)
}

func setupReportRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	reports := rg.Group("/reports")
	reports.GET("/summary", hdlrs.Report.GetSummary)
	reports.GET("/net-worth", hdlrs.Report.NetWorth)
}

func setupSalaryProfileRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	rg.GET("/salary-profile", hdlrs.SalaryProfile.Get)
	rg.PUT("/salary-profile", hdlrs.SalaryProfile.Save)
	rg.POST("/purchase-check", hdlrs.SalaryProfile.CheckPurchase)
}

func setupCategoryRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	cats := rg.Group("/categories")
	cats.GET("", hdlrs.Category.List)
	cats.POST("", hdlrs.Category.Create)
	cats.PATCH("/:id", hdlrs.Category.Update)
	cats.DELETE("/:id", hdlrs.Category.Delete)

	rules := rg.Group("/category-rules")
	rules.GET("", hdlrs.Category.ListRules)
	rules.POST("", hdlrs.Category.CreateRule)
	rules.DELETE("/:id", hdlrs.Category.DeleteRule)
	rules.GET("/:id/preview", hdlrs.Category.PreviewRule)
	rules.POST("/:id/apply", hdlrs.Category.ApplyRule)

	txs := rg.Group("/transactions")
	txs.PATCH("/:id/categories", hdlrs.Category.SetTransactionCategories)
}
