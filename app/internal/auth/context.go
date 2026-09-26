package auth

import "context"

type contextKey string

const (
	userTokenKey contextKey = "supabase_user_token" //nolint:gosec // context key name, not a credential value
	userIDKey    contextKey = "supabase_user_id"
	userRoleKey  contextKey = "supabase_user_role"
)

func WithUser(ctx context.Context, token, userID, role string) context.Context {
	ctx = context.WithValue(ctx, userTokenKey, token)
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userRoleKey, role)
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

// RoleFromContext returns the caller's household role ("admin" or "member")
// as carried in the JWT's user_role claim. Empty if unset.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userRoleKey).(string)
	return v
}
