package repositories

import (
	"life-base-api/app/internal/databases"
	supabaserepo "life-base-api/app/internal/repositories/supabase"
)

type Registry struct {
	Accounts              *supabaserepo.AccountRepository
	Transactions          *supabaserepo.TransactionRepository
	Transfers             *supabaserepo.TransferRepository
	Classifications       *supabaserepo.ClassificationRepository
	Categories            *supabaserepo.CategoryRepository
	CategoryRules         *supabaserepo.CategoryRuleRepository
	TransactionCategories *supabaserepo.TransactionCategoryRepository
	RuleExceptions        *supabaserepo.AccountRuleExceptionRepository
	Budgets               *supabaserepo.BudgetRepository
	Envelopes             *supabaserepo.EnvelopeRepository
	Reminders             *supabaserepo.ReminderRepository
	SalaryProfiles        *supabaserepo.SalaryProfileRepository
	SocialPosts           *supabaserepo.SocialPostRepository
	Admin                 *supabaserepo.AdminRepository
	Caption               *supabaserepo.CaptionRepository
	Preferences           *supabaserepo.PreferencesRepository
	PushSubscriptions     *supabaserepo.PushSubscriptionRepository

	Bikes            *supabaserepo.BikeRepository
	BikeFitHistory   *supabaserepo.BikeFitHistoryRepository
	Components       *supabaserepo.ComponentRepository
	ComponentHistory *supabaserepo.ComponentHistoryRepository
	ServiceLogs      *supabaserepo.ServiceLogRepository
	MaintenanceTasks *supabaserepo.MaintenanceTaskRepository
	Gear             *supabaserepo.GearRepository
	Bottles          *supabaserepo.BottleRepository
	Supplies         *supabaserepo.SupplyRepository
	SupplyHistory    *supabaserepo.SupplyHistoryRepository
	Activities       *supabaserepo.ActivityRepository
	Strava           *supabaserepo.StravaRepository

	Tasks *supabaserepo.TaskRepository
	Notes *supabaserepo.NoteRepository
}

func NewRegistry(dbs *databases.Registry) *Registry {
	return &Registry{
		Accounts:              supabaserepo.NewAccountRepository(dbs.Supabase),
		Transactions:          supabaserepo.NewTransactionRepository(dbs.Supabase),
		Transfers:             supabaserepo.NewTransferRepository(dbs.Supabase),
		Classifications:       supabaserepo.NewClassificationRepository(dbs.Supabase),
		Categories:            supabaserepo.NewCategoryRepository(dbs.Supabase),
		CategoryRules:         supabaserepo.NewCategoryRuleRepository(dbs.Supabase),
		TransactionCategories: supabaserepo.NewTransactionCategoryRepository(dbs.Supabase),
		RuleExceptions:        supabaserepo.NewAccountRuleExceptionRepository(dbs.Supabase),
		Budgets:               supabaserepo.NewBudgetRepository(dbs.Supabase),
		Envelopes:             supabaserepo.NewEnvelopeRepository(dbs.Supabase),
		Reminders:             supabaserepo.NewReminderRepository(dbs.Supabase),
		SalaryProfiles:        supabaserepo.NewSalaryProfileRepository(dbs.Supabase),
		SocialPosts:           supabaserepo.NewSocialPostRepository(dbs.Supabase),
		Admin:                 supabaserepo.NewAdminRepository(dbs.Supabase),
		Caption:               supabaserepo.NewCaptionRepository(dbs.Supabase),
		Preferences:           supabaserepo.NewPreferencesRepository(dbs.Supabase),
		PushSubscriptions:     supabaserepo.NewPushSubscriptionRepository(dbs.Supabase),

		Bikes:            supabaserepo.NewBikeRepository(dbs.Supabase),
		BikeFitHistory:   supabaserepo.NewBikeFitHistoryRepository(dbs.Supabase),
		Components:       supabaserepo.NewComponentRepository(dbs.Supabase),
		ComponentHistory: supabaserepo.NewComponentHistoryRepository(dbs.Supabase),
		ServiceLogs:      supabaserepo.NewServiceLogRepository(dbs.Supabase),
		MaintenanceTasks: supabaserepo.NewMaintenanceTaskRepository(dbs.Supabase),
		Gear:             supabaserepo.NewGearRepository(dbs.Supabase),
		Bottles:          supabaserepo.NewBottleRepository(dbs.Supabase),
		Supplies:         supabaserepo.NewSupplyRepository(dbs.Supabase),
		SupplyHistory:    supabaserepo.NewSupplyHistoryRepository(dbs.Supabase),
		Activities:       supabaserepo.NewActivityRepository(dbs.Supabase),
		Strava:           supabaserepo.NewStravaRepository(dbs.Supabase),

		Tasks: supabaserepo.NewTaskRepository(dbs.Supabase),
		Notes: supabaserepo.NewNoteRepository(dbs.Supabase),
	}
}
