package services

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"life-base-api/app/internal/models"
)

type AccountRepository interface {
	FindAll(ctx context.Context) ([]*models.Account, error)
	FindByID(ctx context.Context, id string) (*models.Account, error)
	FindByAccountNumber(ctx context.Context, number string) (*models.Account, error)
	Upsert(ctx context.Context, a *models.Account) (*models.Account, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Account, error)
	Delete(ctx context.Context, id string) error
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	UpsertBatch(ctx context.Context, accountID string, sourceFile string, txs []models.Transaction) error
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*models.Transaction, error)
	ListFiltered(ctx context.Context, accountID string, filter models.TxFilter) ([]*models.Transaction, int, error)
	GetByAccountIDsInRange(ctx context.Context, accountIDs []string, from, to time.Time) ([]*models.Transaction, error)
	GetCurrentBalances(ctx context.Context, accountIDs []string) (map[string]float64, error)
	GetLastBalancePerAccount(ctx context.Context, accountIDs []string, before time.Time) (map[string]float64, error)
	FindMatchingPattern(ctx context.Context, pattern, accountID string) ([]*models.Transaction, error)
	Delete(ctx context.Context, id string) error
	SetTransferID(ctx context.Context, txID, transferID string) error
	ClearTransferID(ctx context.Context, txID string) error
	UpdateType(ctx context.Context, txID string, txType models.TransactionType) error
	UpdateNote(ctx context.Context, txID string, note string) error
}

// TransferRepository persists the link row connecting two transactions
// that represent the same money moving between accounts.
type TransferRepository interface {
	Create(ctx context.Context, fromTxID, toTxID string, exchangeRate *float64, exchangeSource string) (*models.Transfer, error)
	GetByID(ctx context.Context, id string) (*models.Transfer, error)
	Delete(ctx context.Context, id string) error
}

type ClassificationRuleRepository interface {
	FindAll(ctx context.Context) ([]models.ClassificationRule, error)
}

type CategoryRepository interface {
	FindAll(ctx context.Context) ([]*models.Category, error)
	FindByID(ctx context.Context, id string) (*models.Category, error)
	Create(ctx context.Context, c *models.Category) (*models.Category, error)
	Update(ctx context.Context, id string, fields map[string]string) (*models.Category, error)
	SoftDelete(ctx context.Context, id string) error
}

type CategoryRuleRepository interface {
	FindAll(ctx context.Context) ([]*models.CategoryRule, error)
	FindByID(ctx context.Context, id string) (*models.CategoryRule, error)
	FindByAccountID(ctx context.Context, accountID string) ([]*models.CategoryRule, error)
	Create(ctx context.Context, r *models.CategoryRule) (*models.CategoryRule, error)
	Delete(ctx context.Context, id string) error
}

type TransactionCategoryRepository interface {
	SetCategories(ctx context.Context, transactionID string, categoryIDs []string) error
	AddCategoryBatch(ctx context.Context, txIDs []string, categoryID string) error
}

type AccountRuleExceptionRepository interface {
	FindByAccount(ctx context.Context, accountID string) ([]string, error)
	Create(ctx context.Context, accountID, ruleID string) error
	Delete(ctx context.Context, accountID, ruleID string) error
}

type BudgetRepository interface {
	List(ctx context.Context) ([]*models.Budget, error)
	Create(ctx context.Context, input models.BudgetInput) (*models.Budget, error)
	FindByID(ctx context.Context, id string) (*models.Budget, error)
	Update(ctx context.Context, id string, amount decimal.Decimal) (*models.Budget, error)
	Delete(ctx context.Context, id string) error
	Acknowledge(ctx context.Context, budgetID, month, action string, transferID *string) (*models.BudgetAcknowledgment, error)
	ListAcknowledgments(ctx context.Context, budgetIDs []string, month string) ([]*models.BudgetAcknowledgment, error)
}

type EnvelopeRepository interface {
	List(ctx context.Context) ([]*models.Envelope, error)
	ListByAccountID(ctx context.Context, accountID string) ([]*models.Envelope, error)
	FindByID(ctx context.Context, id string) (*models.Envelope, error)
	Create(ctx context.Context, input models.EnvelopeInput) (*models.Envelope, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Envelope, error)
	Delete(ctx context.Context, id string) error
	Contribute(ctx context.Context, envelopeID string, input models.ContributionInput) (*models.EnvelopeContribution, error)
	GetBalances(ctx context.Context, envelopeIDs []string) (map[string]decimal.Decimal, error)
	SetNextContributionDate(ctx context.Context, id, date string) error
}

type ReminderRepository interface {
	List(ctx context.Context) ([]*models.Reminder, error)
	ListByAccountID(ctx context.Context, accountID string) ([]*models.Reminder, error)
	FindByID(ctx context.Context, id string) (*models.Reminder, error)
	Create(ctx context.Context, input models.ReminderInput) (*models.Reminder, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Reminder, error)
	Delete(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string) error
}

type SalaryProfileRepository interface {
	FindByUserID(ctx context.Context, userID string) (*models.SalaryProfile, error)
	Upsert(ctx context.Context, p *models.SalaryProfile) (*models.SalaryProfile, error)
}

type SocialPostRepository interface {
	List(ctx context.Context, limit, offset int, status *models.SocialPostStatus) ([]*models.SocialPost, error)
	FindByID(ctx context.Context, id string) (*models.SocialPost, error)
	FindRecentByFilename(ctx context.Context, filename string, since time.Time) ([]*models.SocialPost, error)
	Create(ctx context.Context, input models.SocialPostInput) (*models.SocialPost, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.SocialPost, error)
	Delete(ctx context.Context, id string) error
}

type BikeRepository interface {
	List(ctx context.Context) ([]*models.Bike, error)
	FindByID(ctx context.Context, id string) (*models.Bike, error)
	Create(ctx context.Context, input models.BikeInput) (*models.Bike, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Bike, error)
	Delete(ctx context.Context, id string) error
	IncrementMileage(ctx context.Context, id string, distanceKm float64) error
}

type BikeFitHistoryRepository interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.BikeFitHistory, error)
	Create(ctx context.Context, input models.BikeFitHistoryInput) (*models.BikeFitHistory, error)
	Delete(ctx context.Context, id string) error
}

type ComponentRepository interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.Component, error)
	FindByID(ctx context.Context, id string) (*models.Component, error)
	ListActiveByBikeID(ctx context.Context, bikeID string) ([]*models.Component, error)
	Create(ctx context.Context, input models.ComponentInput) (*models.Component, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Component, error)
	Delete(ctx context.Context, id string) error
}

type ComponentHistoryRepository interface {
	ListByComponentID(ctx context.Context, componentID string) ([]*models.ComponentHistory, error)
	Create(ctx context.Context, input models.ComponentHistoryInput) (*models.ComponentHistory, error)
}

type ServiceLogRepository interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.ServiceLog, error)
	Create(ctx context.Context, input models.ServiceLogInput) (*models.ServiceLog, error)
	Delete(ctx context.Context, id string) error
}

type MaintenanceTaskRepository interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.MaintenanceTask, error)
	Create(ctx context.Context, input models.MaintenanceTaskInput) (*models.MaintenanceTask, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.MaintenanceTask, error)
	Delete(ctx context.Context, id string) error
}

type GearRepository interface {
	List(ctx context.Context) ([]*models.Gear, error)
	FindByID(ctx context.Context, id string) (*models.Gear, error)
	ListActiveByBikeID(ctx context.Context, bikeID string) ([]*models.Gear, error)
	Create(ctx context.Context, input models.GearInput) (*models.Gear, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Gear, error)
	Delete(ctx context.Context, id string) error
}

type BottleRepository interface {
	List(ctx context.Context) ([]*models.Bottle, error)
	Create(ctx context.Context, input models.BottleInput) (*models.Bottle, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Bottle, error)
	Delete(ctx context.Context, id string) error
}

type SupplyRepository interface {
	List(ctx context.Context) ([]*models.Supply, error)
	FindByID(ctx context.Context, id string) (*models.Supply, error)
	Create(ctx context.Context, input models.SupplyInput) (*models.Supply, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Supply, error)
	Delete(ctx context.Context, id string) error
}

type SupplyHistoryRepository interface {
	List(ctx context.Context) ([]*models.SupplyHistory, error)
	Create(ctx context.Context, input models.SupplyHistoryInput) (*models.SupplyHistory, error)
}

type ActivityRepository interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.Activity, error)
	Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error)
	Delete(ctx context.Context, id string) error
}

type StravaRepository interface {
	EncryptToken(ctx context.Context, token, encryptionKey string) (string, error)
	DecryptToken(ctx context.Context, encryptedToken, encryptionKey string) (string, error)
	FindConnectionByUserID(ctx context.Context, userID string) (*models.StravaConnection, error)
	UpsertConnection(ctx context.Context, input models.StravaConnectionInput) (*models.StravaConnection, error)
	UpdateConnection(ctx context.Context, id string, fields map[string]any) (*models.StravaConnection, error)
	DeleteConnection(ctx context.Context, userID string) error
	CreateOAuthState(ctx context.Context, input models.OAuthStateInput) (*models.OAuthState, error)
	FindOAuthState(ctx context.Context, state, userID string) (*models.OAuthState, error)
	DeleteOAuthState(ctx context.Context, id string) error
}
