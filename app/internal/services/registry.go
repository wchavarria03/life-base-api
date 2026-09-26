package services

import "life-base-api/app/internal/repositories"

type Registry struct {
	Account        *AccountService
	Budget         *BudgetService
	Envelope       *EnvelopeService
	Reminder       *ReminderService
	Category       *CategoryService
	Classification *ClassificationService
	Import         *ImportService
	Report         *ReportService
	RuleExceptions AccountRuleExceptionRepository
	Transaction    *TransactionService
	Transfer       *TransferService
	SalaryProfile  *SalaryProfileService
	Social         *SocialService
	Admin          *AdminService
	Caption        *CaptionService
	Preferences    *PreferencesService
	Push           *PushService
	Digest         *DigestService
	AuditLog       *AuditLogService
	ScheduledPost  *ScheduledPostService

	Bike            *BikeService
	BikeFitHistory  *BikeFitHistoryService
	Component       *ComponentService
	ServiceLog      *ServiceLogService
	MaintenanceTask *MaintenanceTaskService
	Gear            *GearService
	Bottle          *BottleService
	Supply          *SupplyService
	Activity        *ActivityService
	Strava          *StravaService

	Task *TaskService
	Note *NoteService
}

// NewRegistry wires every service with its repository dependencies.
func NewRegistry(repos *repositories.Registry, userID string, social SocialConfig, strava StravaConfig, digest DigestConfig) *Registry {
	classifier := NewClassificationService(repos.Classifications)
	transfer := NewTransferService(repos.Accounts, repos.Transactions, repos.Transfers)
	reminder := NewReminderService(repos.Reminders, repos.TransactionCategories)
	preferences := NewPreferencesService(repos.Preferences)
	push := NewPushService(repos.PushSubscriptions)
	socialSvc := NewSocialService(repos.SocialPosts, social)
	captionSvc := NewCaptionService(repos.Caption)
	return &Registry{
		Account:        NewAccountService(repos.Accounts, repos.Transactions),
		Budget:         NewBudgetService(repos.Budgets, repos.Accounts, repos.Transactions),
		Envelope:       NewEnvelopeService(repos.Envelopes),
		Reminder:       reminder,
		Category:       NewCategoryService(repos.Categories, repos.CategoryRules, repos.TransactionCategories, repos.Transactions),
		Classification: classifier,
		Import:         NewImportService(repos.Accounts, repos.Transactions, classifier, repos.CategoryRules, repos.TransactionCategories, repos.RuleExceptions, transfer, reminder, userID),
		Report:         NewReportService(repos.Transactions, repos.Categories),
		RuleExceptions: repos.RuleExceptions,
		Transaction:    NewTransactionService(repos.Transactions),
		Transfer:       transfer,
		SalaryProfile:  NewSalaryProfileService(repos.SalaryProfiles),
		Social:         socialSvc,
		Admin:          NewAdminService(repos.Admin),
		Caption:        captionSvc,
		Preferences:    preferences,
		Push:           push,
		Digest:         NewDigestService(repos.Reminders, preferences, push, repos.Admin, digest),
		AuditLog:       NewAuditLogService(repos.AuditLog),
		ScheduledPost:  NewScheduledPostService(repos.ScheduledPosts, repos.Storage, socialSvc, captionSvc),

		Bike:            NewBikeService(repos.Bikes),
		BikeFitHistory:  NewBikeFitHistoryService(repos.BikeFitHistory),
		Component:       NewComponentService(repos.Components, repos.ComponentHistory),
		ServiceLog:      NewServiceLogService(repos.ServiceLogs),
		MaintenanceTask: NewMaintenanceTaskService(repos.MaintenanceTasks, repos.Bikes),
		Gear:            NewGearService(repos.Gear),
		Bottle:          NewBottleService(repos.Bottles),
		Supply:          NewSupplyService(repos.Supplies, repos.SupplyHistory),
		Activity:        NewActivityService(repos.Activities, repos.Bikes, repos.Components, repos.Gear),
		Strava:          NewStravaService(repos.Strava, strava),

		Task: NewTaskService(repos.Tasks),
		Note: NewNoteService(repos.Notes),
	}
}
