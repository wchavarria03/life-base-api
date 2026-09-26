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

func NewRegistry(repos *repositories.Registry, userID string, social SocialConfig, strava StravaConfig) *Registry {
	classifier := NewClassificationService(repos.Classifications)
	transfer := NewTransferService(repos.Accounts, repos.Transactions, repos.Transfers)
	reminder := NewReminderService(repos.Reminders, repos.TransactionCategories)
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
		Social:         NewSocialService(repos.SocialPosts, social),
		Admin:          NewAdminService(repos.Admin),
		Caption:        NewCaptionService(repos.Caption),
		Preferences:    NewPreferencesService(repos.Preferences),

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
