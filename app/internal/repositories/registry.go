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
	}
}
