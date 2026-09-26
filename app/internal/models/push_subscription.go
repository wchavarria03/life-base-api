package models

// PushSubscription is the stored shape from push_subscriptions — one row
// per browser/device the user has granted push permission on.
type PushSubscription struct {
	ID       string `json:"id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	AuthKey  string `json:"auth_key"`
}
