package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/auth"
	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

func NewAccountRuleExceptionRepository(client *databases.SupabaseClient) *AccountRuleExceptionRepository {
	return &AccountRuleExceptionRepository{client: client}
}

type ruleIDRow struct {
	RuleID string `json:"rule_id"`
}

// FindByAccount returns the IDs of global category rules that are disabled for the given account.
func (r *AccountRuleExceptionRepository) FindByAccount(ctx context.Context, accountID string) ([]string, error) {
	rows, err := databases.Get[[]*ruleIDRow](ctx, r.client, "/rest/v1/account_rule_exceptions", url.Values{
		"account_id": []string{"eq." + accountID},
		"select":     []string{"rule_id"},
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.RuleID
	}
	return ids, nil
}

func (r *AccountRuleExceptionRepository) Create(ctx context.Context, accountID, ruleID string) error {
	userID := auth.UserIDFromContext(ctx)
	exc := models.AccountRuleException{
		UserID:    userID,
		AccountID: accountID,
		RuleID:    ruleID,
	}
	_, err := databases.Post[struct{}](ctx, r.client,
		"/rest/v1/account_rule_exceptions?on_conflict=account_id,rule_id",
		exc, "resolution=ignore-duplicates")
	return err
}

func (r *AccountRuleExceptionRepository) Delete(ctx context.Context, accountID, ruleID string) error {
	return databases.Delete(ctx, r.client,
		"/rest/v1/account_rule_exceptions?account_id=eq."+accountID+"&rule_id=eq."+ruleID)
}
