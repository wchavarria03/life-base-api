package auth

import "context"

type contextKey string

const (
	userTokenKey contextKey = "supabase_user_token" //nolint:gosec // context key name, not a credential value
	userIDKey    contextKey = "supabase_user_id"
)

func WithUser(ctx context.Context, token, userID string) context.Context {
	ctx = context.WithValue(ctx, userTokenKey, token)
	ctx = context.WithValue(ctx, userIDKey, userID)
	return ctx
}

func UserTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userTokenKey).(string)
	return v
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}
