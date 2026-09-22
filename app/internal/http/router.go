package httpserver

import (
	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/handlers"
	"life-base-api/app/internal/http/middleware"
)

// NewRouter creates a new Router with all routes configured.
func NewRouter(hdlrs *handlers.Registry, jwksURL string, allowedOrigins []string) *Router {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	engine.Use(middleware.CORS(allowedOrigins))

	setupRoutes(engine, hdlrs, jwksURL)

	return &Router{engine: engine}
}

// setupRoutes configures all versioned routes for the application.
func setupRoutes(engine *gin.Engine, hdlrs *handlers.Registry, jwksURL string) {
	v1 := engine.Group("/v1")
	v1.Use(middleware.Auth(jwksURL))

	v1.GET("/me", hdlrs.Me.GetMe)
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
	v1.PATCH("/transactions/:id/type", hdlrs.Transfer.UpdateTransactionType)
	v1.PATCH("/transactions/:id/note", hdlrs.Transaction.UpdateNote)
}

func setupSocialRoutes(rg *gin.RouterGroup, hdlrs *handlers.Registry) {
	social := rg.Group("/social/posts")
	social.POST("", hdlrs.Social.Create)
	social.GET("", hdlrs.Social.List)
	social.DELETE("/:id", hdlrs.Social.Delete)
	social.POST("/:id/retry-instagram", hdlrs.Social.RetryInstagram)
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
