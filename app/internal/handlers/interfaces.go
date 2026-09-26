package handlers

import (
	"context"
	"io"
	"time"

	"github.com/shopspring/decimal"

	"life-base-api/app/internal/models"
)

type Importer interface {
	Import(ctx context.Context, stmt *models.Statement, bankName string) error
}

type AccountLister interface {
	List(ctx context.Context) ([]*models.Account, error)
	GetByID(ctx context.Context, id string) (*models.Account, error)
	Create(ctx context.Context, a *models.Account) (*models.Account, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Account, error)
	Delete(ctx context.Context, id string) error
}

type TransactionLister interface {
	ListFiltered(ctx context.Context, accountID string, filter models.TxFilter) ([]*models.Transaction, int, error)
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	UpdateNote(ctx context.Context, id, note string) (*models.Transaction, error)
}

type StatementImporter interface {
	ImportWithSummary(ctx context.Context, stmt *models.Statement, bankName string, catOverrides map[int][]string) (*models.ImportSummary, error)
	CheckOverlap(ctx context.Context, stmt *models.Statement) (int, error)
	PreviewCategories(ctx context.Context, stmt *models.Statement) map[int]string
}

type ReportSummarizer interface {
	Summarize(ctx context.Context, accountIDs []string, from, to time.Time) (*models.ReportSummary, error)
}

type TransferService interface {
	MatchForPeriod(ctx context.Context, from, to time.Time, fxMin, fxMax *float64) ([]models.TransferMatch, error)
	CreateTransfer(ctx context.Context, input models.TransferInput) (*models.TransferResult, error)
	LinkTransactions(ctx context.Context, fromTxID, toTxID string) (*models.TransferResult, error)
	LinkExisting(ctx context.Context, existingTxID, counterpartAccountID string) (*models.TransferResult, error)
	UpdateTransactionType(ctx context.Context, txID string, newType models.TransactionType) (*models.Transaction, error)
	ReconcileForPeriod(ctx context.Context, from, to time.Time) (int, error)
}

type RuleExceptionManager interface {
	FindByAccount(ctx context.Context, accountID string) ([]string, error)
	Create(ctx context.Context, accountID, ruleID string) error
	Delete(ctx context.Context, accountID, ruleID string) error
}

type CategoryManager interface {
	List(ctx context.Context) ([]*models.Category, error)
	Create(ctx context.Context, c *models.Category) (*models.Category, error)
	Update(ctx context.Context, id string, fields map[string]string) (*models.Category, error)
	Delete(ctx context.Context, id string) error
	ListRules(ctx context.Context) ([]*models.CategoryRule, error)
	CreateRule(ctx context.Context, r *models.CategoryRule) (*models.CategoryRule, error)
	DeleteRule(ctx context.Context, id string) error
	SetTransactionCategories(ctx context.Context, transactionID string, categoryIDs []string) error
	PreviewRule(ctx context.Context, ruleID string) ([]*models.Transaction, error)
	ApplyRule(ctx context.Context, ruleID string) (int, error)
}

type BudgetManager interface {
	List(ctx context.Context, month string) ([]models.BudgetStatus, error)
	Create(ctx context.Context, input models.BudgetInput) (*models.Budget, error)
	Update(ctx context.Context, id string, amount decimal.Decimal) (*models.Budget, error)
	Delete(ctx context.Context, id string) error
	Acknowledge(ctx context.Context, budgetID, month, action string, transferID *string) (*models.BudgetAcknowledgment, error)
}

type EnvelopeManager interface {
	List(ctx context.Context) ([]models.EnvelopeStatus, error)
	ListByAccountID(ctx context.Context, accountID string) ([]models.EnvelopeStatus, error)
	Create(ctx context.Context, input models.EnvelopeInput) (*models.EnvelopeStatus, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.EnvelopeStatus, error)
	Delete(ctx context.Context, id string) error
	Contribute(ctx context.Context, id string, input models.ContributionInput) (*models.EnvelopeStatus, error)
}

type ReminderManager interface {
	List(ctx context.Context) ([]models.ReminderWithStatus, error)
	ListByAccountID(ctx context.Context, accountID string) ([]models.ReminderWithStatus, error)
	Create(ctx context.Context, input models.ReminderInput) (*models.ReminderWithStatus, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.ReminderWithStatus, error)
	Delete(ctx context.Context, id string) error
	Complete(ctx context.Context, id string) (*models.ReminderWithStatus, error)
	Link(ctx context.Context, id, transactionID, nextDueDate string) (*models.ReminderWithStatus, error)
}

type SalaryProfileManager interface {
	Get(ctx context.Context, userID string) (*models.SalaryProfile, error)
	Save(ctx context.Context, p *models.SalaryProfile) (*models.SalaryProfile, error)
	CheckPurchase(ctx context.Context, p *models.SalaryProfile, price float64) (*models.PurchaseCheck, error)
}

// AuditLogManager lists audit log entries for the admin UI.
type AuditLogManager interface {
	List(ctx context.Context, limit, offset int) ([]*models.AuditLogEntry, error)
}

// PushManager registers and removes web push subscriptions.
// ScheduledPostManager schedules social posts for later sending.
type ScheduledPostManager interface {
	Create(ctx context.Context, file io.Reader, filename, fbCaption string, igCaption *string, toFacebook, toInstagram bool, categoryIDs []string, scheduledAt time.Time) (*models.ScheduledPost, error)
	List(ctx context.Context) ([]*models.ScheduledPost, error)
	Cancel(ctx context.Context, id string) error
	ProcessDue(ctx context.Context, userID string) (sent int, failed int, err error)
}

type PushManager interface {
	Subscribe(ctx context.Context, userID, endpoint, p256dh, authKey string) error
	Unsubscribe(ctx context.Context, endpoint string) error
}

// PreferenceManager reads and writes per-user notification preferences.
type PreferenceManager interface {
	Get(ctx context.Context, userID string) (*models.UserPreferences, error)
	Set(ctx context.Context, userID string, pushEnabled, emailDigestEnabled bool) (*models.UserPreferences, error)
}

type AdminManager interface {
	ListMembers(ctx context.Context) ([]*models.HouseholdMember, error)
	SetMemberRole(ctx context.Context, userID, role string) error
	ListPageAccess(ctx context.Context) ([]*models.PageAccessEntry, error)
	SetPageAccess(ctx context.Context, entry *models.PageAccessEntry) error
	AllowedPageKeys(ctx context.Context, role string) ([]string, error)
}

// CaptionManager manages the caption template library.
type CaptionManager interface {
	ListCategories(ctx context.Context) ([]*models.CaptionCategory, error)
	CreateCategory(ctx context.Context, name string) (*models.CaptionCategory, error)
	UpdateCategory(ctx context.Context, id, name string) (*models.CaptionCategory, error)
	DeleteCategory(ctx context.Context, id string) error

	ListTemplates(ctx context.Context) ([]*models.CaptionTemplate, error)
	CreateTemplate(ctx context.Context, title, body string, categoryIDs []string) (*models.CaptionTemplate, error)
	UpdateTemplateMeta(ctx context.Context, id string, title *string, categoryIDs *[]string) error
	DeleteTemplate(ctx context.Context, id string) error

	ListVersions(ctx context.Context, templateID string) ([]*models.CaptionTemplateVersion, error)
	AddVersion(ctx context.Context, templateID, body string) (*models.CaptionTemplateVersion, error)
	RevertToVersion(ctx context.Context, templateID, versionID string) error

	SetPostCategories(ctx context.Context, postID string, categoryIDs []string) error
	ListPostCategoryIDs(ctx context.Context) (map[string][]string, error)
}

type SocialPoster interface {
	PostImage(ctx context.Context, file io.Reader, filename string, caption *string, force, toFacebook, toInstagram bool) (*models.SocialPost, error)
	PostImageWithCaptions(ctx context.Context, file io.Reader, filename string, fbCaption, igCaption *string, force, toFacebook, toInstagram bool) (*models.SocialPost, error)
	List(ctx context.Context, limit, offset int, status *models.SocialPostStatus) ([]*models.SocialPost, error)
	RetryInstagram(ctx context.Context, id string) (*models.SocialPost, error)
	Delete(ctx context.Context, id string) error
}

type BikeManager interface {
	List(ctx context.Context) ([]*models.Bike, error)
	FindByID(ctx context.Context, id string) (*models.Bike, error)
	Create(ctx context.Context, input models.BikeInput) (*models.Bike, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Bike, error)
	Delete(ctx context.Context, id string) error
}

type BikeFitHistoryManager interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.BikeFitHistory, error)
	Create(ctx context.Context, input models.BikeFitHistoryInput) (*models.BikeFitHistory, error)
	Delete(ctx context.Context, id string) error
}

type ComponentManager interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]models.ComponentWithStatus, error)
	Create(ctx context.Context, input models.ComponentInput) (*models.Component, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Component, error)
	Delete(ctx context.Context, id string) error
	Replace(ctx context.Context, componentID, replacedDate string, mileageAtReplacement *float64, notes *string) (*models.Component, error)
	ListHistory(ctx context.Context, componentID string) ([]*models.ComponentHistory, error)
}

type ServiceLogManager interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.ServiceLog, error)
	Create(ctx context.Context, input models.ServiceLogInput) (*models.ServiceLog, error)
	Delete(ctx context.Context, id string) error
}

type MaintenanceTaskManager interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]models.MaintenanceTaskWithStatus, error)
	Create(ctx context.Context, input models.MaintenanceTaskInput) (*models.MaintenanceTask, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.MaintenanceTask, error)
	Delete(ctx context.Context, id string) error
	Complete(ctx context.Context, id, bikeID string) (*models.MaintenanceTask, error)
}

type GearManager interface {
	List(ctx context.Context) ([]*models.Gear, error)
	Create(ctx context.Context, input models.GearInput) (*models.Gear, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Gear, error)
	Delete(ctx context.Context, id string) error
}

type BottleManager interface {
	List(ctx context.Context) ([]models.BottleWithStatus, error)
	Create(ctx context.Context, input models.BottleInput) (*models.Bottle, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Bottle, error)
	MarkCleaned(ctx context.Context, id string) (*models.Bottle, error)
	Delete(ctx context.Context, id string) error
}

type SupplyManager interface {
	List(ctx context.Context) ([]*models.Supply, error)
	Create(ctx context.Context, input models.SupplyInput) (*models.Supply, error)
	Update(ctx context.Context, id string, fields map[string]any) (*models.Supply, error)
	Delete(ctx context.Context, id string) error
	Deplete(ctx context.Context, id string) (*models.SupplyHistory, error)
	ListHistory(ctx context.Context) ([]*models.SupplyHistory, error)
}

type ActivityManager interface {
	ListByBikeID(ctx context.Context, bikeID string) ([]*models.Activity, error)
	Create(ctx context.Context, input models.ActivityInput) (*models.Activity, error)
	Delete(ctx context.Context, id string) error
}

type StravaManager interface {
	Authorize(ctx context.Context, redirectURI string) (string, error)
	Callback(ctx context.Context, code, state string) (*models.StravaConnection, error)
	Status(ctx context.Context) (*models.StravaConnection, error)
	Disconnect(ctx context.Context) error
	FetchActivities(ctx context.Context, after, before *time.Time, page, perPage int) ([]models.StravaActivityPreview, error)
}
