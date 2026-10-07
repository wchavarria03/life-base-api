package auth

import "context"

type contextKey string

const (
	userTokenKey contextKey = "supabase_user_token" //nolint:gosec // context key name, not a credential value
	userIDKey    contextKey = "supabase_user_id"
	userRoleKey  contextKey = "supabase_user_role"
	userEmailKey contextKey = "supabase_user_email"
)

func WithUser(ctx context.Context, token, userID, role, email string) context.Context {
	ctx = context.WithValue(ctx, userTokenKey, token)
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	ctx = context.WithValue(ctx, userEmailKey, email)
	return ctx
}

// EmailFromContext returns the caller's own email, as carried in the JWT's
// email claim. Empty if unset — used to look up a linked kids.child_profile
// (see services/kid.go's ChildProfileService.FindMyProfile).
func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userEmailKey).(string)
	return v
}

// WithServiceRole clears the user token carried in ctx, so any repository
// call made with the returned context falls through resolveKeys' service-key
// branch (databases/helpers.go) instead of forwarding the caller's own JWT —
// the same bypass CLI/cron paths get "for free" by never having a user
// token in context at all, just reachable mid-request for one specific call.
//
// This is a deliberate authorization-boundary exception, not a convenience:
// everywhere else in this app, RLS (keyed off the caller's own JWT) is the
// whole authorization boundary. The one case that doesn't fit is a kid
// completing a task assigned to them by a parent, or buying from a shop a
// parent stocked — those writes legitimately cross from the child's own
// auth.uid() into rows a different user (the parent) owns, which plain RLS
// can't express without also letting the child edit the row's other
// columns (Postgres RLS has no column-level grain). The caller MUST do its
// own authorization check in Go before using this — WithServiceRole itself
// enforces nothing. See services/kid.go for the only callers.
func WithServiceRole(ctx context.Context) context.Context {
	return context.WithValue(ctx, userTokenKey, "")
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
