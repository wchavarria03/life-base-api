package handlers

import "life-base-api/app/internal/services"

// Registry holds all HTTP handlers.
type Registry struct {
	Account       *AccountHandler
	Budget        *BudgetHandler
	Envelope      *EnvelopeHandler
	Reminder      *ReminderHandler
	Category      *CategoryHandler
	Dump          *DumpHandler
	Extract       *ExtractHandler
	Me            *MeHandler
	Report        *ReportHandler
	RuleException *RuleExceptionHandler
	Transaction   *TransactionHandler
	Transfer      *TransferHandler
	Upload        *UploadHandler
	SalaryProfile *SalaryProfileHandler
	Social        *SocialHandler
	Admin         *AdminHandler
	Caption       *CaptionHandler

	Bike            *BikeHandler
	BikeFitHistory  *BikeFitHistoryHandler
	Component       *ComponentHandler
	ServiceLog      *ServiceLogHandler
	MaintenanceTask *MaintenanceTaskHandler
	Gear            *GearHandler
	Bottle          *BottleHandler
	Supply          *SupplyHandler
	Activity        *ActivityHandler
	Strava          *StravaHandler

	Task *TaskHandler
	Note *NoteHandler
}

func NewRegistry(svc *services.Registry) (*Registry, error) {
	return &Registry{
		Account:       NewAccountHandler(svc.Account),
		Budget:        NewBudgetHandler(svc.Budget, svc.Transfer),
		Envelope:      NewEnvelopeHandler(svc.Envelope),
		Reminder:      NewReminderHandler(svc.Reminder),
		Category:      NewCategoryHandler(svc.Category),
		Dump:          NewDumpHandler(),
		Extract:       NewExtractHandler(svc.Import),
		Me:            NewMeHandler(svc.Admin),
		Report:        NewReportHandler(svc.Account, svc.Report),
		RuleException: NewRuleExceptionHandler(svc.RuleExceptions, svc.Category),
		Transaction:   NewTransactionHandler(svc.Transaction),
		Transfer:      NewTransferHandler(svc.Transfer),
		Upload:        NewUploadHandler(svc.Import),
		SalaryProfile: NewSalaryProfileHandler(svc.SalaryProfile),
		Social:        NewSocialHandler(svc.Social, svc.Caption),
		Admin:         NewAdminHandler(svc.Admin),
		Caption:       NewCaptionHandler(svc.Caption),

		Bike:            NewBikeHandler(svc.Bike),
		BikeFitHistory:  NewBikeFitHistoryHandler(svc.BikeFitHistory),
		Component:       NewComponentHandler(svc.Component),
		ServiceLog:      NewServiceLogHandler(svc.ServiceLog),
		MaintenanceTask: NewMaintenanceTaskHandler(svc.MaintenanceTask),
		Gear:            NewGearHandler(svc.Gear),
		Bottle:          NewBottleHandler(svc.Bottle),
		Supply:          NewSupplyHandler(svc.Supply),
		Activity:        NewActivityHandler(svc.Activity),
		Strava:          NewStravaHandler(svc.Strava),

		Task: NewTaskHandler(svc.Task),
		Note: NewNoteHandler(svc.Note),
	}, nil
}

// Close releases any resources held by handlers.
//
//nolint:revive // receiver intentionally unused; method provided for consistency and future extensibility
func (r *Registry) Close() error {
	return nil
}
