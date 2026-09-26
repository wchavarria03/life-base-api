package handlers

type ExtractHandler struct {
	importer Importer
}

type DumpHandler struct{}

type AccountHandler struct {
	svc AccountLister
}

type MeHandler struct {
	admin AdminManager
}

type TransactionHandler struct {
	svc TransactionLister
}

type CategoryHandler struct {
	svc CategoryManager
}

type ReportHandler struct {
	accounts   AccountLister
	summarizer ReportSummarizer
}

type UploadHandler struct {
	importer StatementImporter
}

type RuleExceptionHandler struct {
	exceptions RuleExceptionManager
	categories CategoryManager
}

type TransferHandler struct {
	svc TransferService
}

type BudgetHandler struct {
	budgets   BudgetManager
	transfers TransferService
}

type EnvelopeHandler struct {
	svc EnvelopeManager
}

type ReminderHandler struct {
	svc ReminderManager
}

type SalaryProfileHandler struct {
	svc SalaryProfileManager
}

type SocialHandler struct {
	svc      SocialPoster
	captions CaptionManager
}

type AdminHandler struct {
	svc AdminManager
}

// CaptionHandler serves the caption template library endpoints.
type CaptionHandler struct {
	svc CaptionManager
}

// PreferencesHandler serves the notification preferences endpoints.
type PreferencesHandler struct {
	svc PreferenceManager
}

// PushHandler serves the push subscription endpoints.
type PushHandler struct {
	svc PushManager
}

// ScheduledPostHandler serves the scheduled social post endpoints.
type ScheduledPostHandler struct {
	svc ScheduledPostManager
}

// AuditLogHandler serves the admin audit log endpoint.
type AuditLogHandler struct {
	svc AuditLogManager
}

// DogHandler serves the dogs/recipients/bulk-bags endpoints.
type DogHandler struct {
	svc DogManager
}

// ExportHandler serves a single JSON dump of everything the caller owns.
type ExportHandler struct {
	accounts     AccountLister
	transactions TransactionLister
	categories   CategoryManager
	budgets      BudgetManager
	envelopes    EnvelopeManager
	reminders    ReminderManager
	social       SocialPoster
}

type BikeHandler struct {
	svc BikeManager
}

type BikeFitHistoryHandler struct {
	svc BikeFitHistoryManager
}

type ComponentHandler struct {
	svc ComponentManager
}

type ServiceLogHandler struct {
	svc ServiceLogManager
}

type MaintenanceTaskHandler struct {
	svc MaintenanceTaskManager
}

type GearHandler struct {
	svc GearManager
}

type BottleHandler struct {
	svc BottleManager
}

type SupplyHandler struct {
	svc SupplyManager
}

type ActivityHandler struct {
	svc ActivityManager
}

type StravaHandler struct {
	svc StravaManager
}
