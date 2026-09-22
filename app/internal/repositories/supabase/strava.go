package supabase

import (
	"context"
	"net/url"

	"life-base-api/app/internal/databases"
	"life-base-api/app/internal/models"
)

type StravaRepository struct {
	client *databases.SupabaseClient
}

func NewStravaRepository(client *databases.SupabaseClient) *StravaRepository {
	return &StravaRepository{client: client}
}

// EncryptToken calls the bikes.encrypt_token_pgp RPC (pgp_sym_encrypt under
// the hood) and returns the encrypted value as PostgREST's bytea-hex text
// representation — opaque, only meant to be stored and later passed back to
// DecryptToken.
func (r *StravaRepository) EncryptToken(ctx context.Context, token, encryptionKey string) (string, error) {
	return databases.Post[string](ctx, r.client, "/rest/v1/rpc/encrypt_token_pgp", map[string]string{
		"token":          token,
		"encryption_key": encryptionKey,
	}, "", bikesSchema)
}

// DecryptToken calls the bikes.decrypt_token_pgp RPC to recover the
// plaintext token from the bytea-hex text EncryptToken returned.
func (r *StravaRepository) DecryptToken(ctx context.Context, encryptedToken, encryptionKey string) (string, error) {
	return databases.Post[string](ctx, r.client, "/rest/v1/rpc/decrypt_token_pgp", map[string]string{
		"encrypted_token": encryptedToken,
		"encryption_key":  encryptionKey,
	}, "", bikesSchema)
}

func (r *StravaRepository) FindConnectionByUserID(ctx context.Context, userID string) (*models.StravaConnection, error) {
	rows, err := databases.Get[[]*models.StravaConnection](ctx, r.client, "/rest/v1/strava_connections", url.Values{
		"user_id": []string{"eq." + userID},
		"limit":   []string{"1"},
	}, bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// UpsertConnection creates or replaces the caller's Strava connection —
// strava_connections has a unique(user_id) constraint, so this is an upsert
// on conflict with that column.
func (r *StravaRepository) UpsertConnection(ctx context.Context, input models.StravaConnectionInput) (*models.StravaConnection, error) {
	rows, err := databases.Post[[]*models.StravaConnection](ctx, r.client,
		"/rest/v1/strava_connections?on_conflict=user_id", input,
		"resolution=merge-duplicates,return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *StravaRepository) UpdateConnection(ctx context.Context, id string, fields map[string]any) (*models.StravaConnection, error) {
	rows, err := databases.Patch[[]*models.StravaConnection](ctx, r.client,
		"/rest/v1/strava_connections?id=eq."+id, fields, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *StravaRepository) DeleteConnection(ctx context.Context, userID string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/strava_connections?user_id=eq."+userID, bikesSchema)
}

func (r *StravaRepository) CreateOAuthState(ctx context.Context, input models.OAuthStateInput) (*models.OAuthState, error) {
	rows, err := databases.Post[[]*models.OAuthState](ctx, r.client, "/rest/v1/oauth_states", input, "return=representation", bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *StravaRepository) FindOAuthState(ctx context.Context, state, userID string) (*models.OAuthState, error) {
	rows, err := databases.Get[[]*models.OAuthState](ctx, r.client, "/rest/v1/oauth_states", url.Values{
		"state":   []string{"eq." + state},
		"user_id": []string{"eq." + userID},
		"limit":   []string{"1"},
	}, bikesSchema)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

func (r *StravaRepository) DeleteOAuthState(ctx context.Context, id string) error {
	return databases.Delete(ctx, r.client, "/rest/v1/oauth_states?id=eq."+id, bikesSchema)
}
