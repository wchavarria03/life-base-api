package models

import "time"

// ChildProfile is the stored shape from kids.child_profiles. Exactly one of
// Email (Google-login mode, matched live against the child's own JWT email
// claim) or a PIN (PIN-mode, verified server-side, never serialized back to
// the client) is expected to be set, though neither is enforced by the
// schema — a parent can set up a profile before deciding which mode to use.
type ChildProfile struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id,omitempty"`
	Name        string    `json:"name"`
	AvatarEmoji *string   `json:"avatar_emoji,omitempty"`
	Email       *string   `json:"email,omitempty"`
	PinHash     *string   `json:"-"`
	PenaltyFee  int       `json:"penalty_fee"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// ChildProfileInput is the write shape for create/update. PIN is a plain
// text PIN carried Go-side only (json:"-", never sent to PostgREST as-is);
// the service hashes it into PinHash before the repository call. Set PIN,
// never PinHash, when calling ChildProfileService.
type ChildProfileInput struct {
	OwnerUserID string  `json:"owner_user_id,omitempty"`
	Name        string  `json:"name,omitempty"`
	AvatarEmoji *string `json:"avatar_emoji,omitempty"`
	Email       *string `json:"email,omitempty"`
	PIN         *string `json:"-"`
	PinHash     *string `json:"pin_hash,omitempty"`
	PenaltyFee  *int    `json:"penalty_fee,omitempty"`
}

// WalletTransaction is one ledger entry from kids.wallet_transactions —
// balance is derived (sum of Amount), never stored, so the ledger itself is
// the only source of truth and stays auditable.
type WalletTransaction struct {
	ID          string    `json:"id"`
	ChildID     string    `json:"child_id"`
	Amount      int       `json:"amount"`
	Reason      string    `json:"reason"`
	TaskID      *string   `json:"task_id,omitempty"`
	ShopOrderID *string   `json:"shop_order_id,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// Wallet is the derived view handed to the frontend: current balance plus
// the transaction history that explains it.
type Wallet struct {
	Balance      int                  `json:"balance"`
	Transactions []*WalletTransaction `json:"transactions"`
}

// ShopItem is the stored shape from kids.shop_items — a parent-defined
// reward a child can spend coins on.
type ShopItem struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CoinCost    int       `json:"coin_cost"`
	Emoji       *string   `json:"emoji,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// ShopItemInput is the write shape for create/update.
type ShopItemInput struct {
	OwnerUserID string  `json:"owner_user_id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	CoinCost    *int    `json:"coin_cost,omitempty"`
	Emoji       *string `json:"emoji,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

// ShopOrder is the stored shape from kids.shop_orders — a purchase, pending
// until the parent fulfills it in real life.
type ShopOrder struct {
	ID                 string     `json:"id"`
	ChildID            string     `json:"child_id"`
	ShopItemID         string     `json:"shop_item_id"`
	CoinCostAtPurchase int        `json:"coin_cost_at_purchase"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at,omitempty"`
	FulfilledAt        *time.Time `json:"fulfilled_at,omitempty"`
}
