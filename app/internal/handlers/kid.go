package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ── Child profiles ───────────────────────────────────────────────────────────

// NewChildProfileHandler constructs a ChildProfileHandler.
func NewChildProfileHandler(svc ChildProfileManager) *ChildProfileHandler {
	return &ChildProfileHandler{svc: svc}
}

func (h *ChildProfileHandler) List(c *gin.Context) {
	listHandler(h.svc.List)(c)
}

func (h *ChildProfileHandler) Create(c *gin.Context) {
	createHandler(h.svc.Create)(c)
}

func (h *ChildProfileHandler) Update(c *gin.Context) {
	updateHandler(h.svc.Update)(c)
}

func (h *ChildProfileHandler) Delete(c *gin.Context) {
	deleteHandler(h.svc.Delete)(c)
}

type verifyPINRequest struct {
	PIN string `json:"pin"`
}

// VerifyPIN handles POST /v1/kids/profiles/:id/verify-pin — the Kid Mode
// entry point. Runs under the parent's own session; RLS already scopes the
// profile lookup inside VerifyPIN to kids the caller owns.
func (h *ChildProfileHandler) VerifyPIN(c *gin.Context) {
	req, ok := bindJSON[verifyPINRequest](c)
	if !ok {
		return
	}
	ok, err := h.svc.VerifyPIN(c.Request.Context(), c.Param("id"), req.PIN)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": ok})
}

// ── Wallet ───────────────────────────────────────────────────────────────────

// NewWalletHandler constructs a WalletHandler.
func NewWalletHandler(svc WalletManager) *WalletHandler {
	return &WalletHandler{svc: svc}
}

// Get handles GET /v1/kids/wallet/:childId.
func (h *WalletHandler) Get(c *gin.Context) {
	wallet, err := h.svc.Wallet(c.Request.Context(), c.Param("childId"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, wallet)
}

// ── Shop ─────────────────────────────────────────────────────────────────────

// NewShopHandler constructs a ShopHandler.
func NewShopHandler(svc ShopManager) *ShopHandler {
	return &ShopHandler{svc: svc}
}

func (h *ShopHandler) ListItems(c *gin.Context) {
	listHandler(h.svc.ListItems)(c)
}

func (h *ShopHandler) CreateItem(c *gin.Context) {
	createHandler(h.svc.CreateItem)(c)
}

func (h *ShopHandler) UpdateItem(c *gin.Context) {
	updateHandler(h.svc.UpdateItem)(c)
}

func (h *ShopHandler) DeleteItem(c *gin.Context) {
	deleteHandler(h.svc.DeleteItem)(c)
}

// ListOrdersByChild handles GET /v1/kids/shop-orders?child_id=:id.
func (h *ShopHandler) ListOrdersByChild(c *gin.Context) {
	orders, err := h.svc.ListOrdersByChild(c.Request.Context(), c.Query("child_id"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, orders)
}

// ListPendingOrders handles GET /v1/kids/shop-orders/pending — the parent's
// review queue in Kids Settings.
func (h *ShopHandler) ListPendingOrders(c *gin.Context) {
	listHandler(h.svc.ListPendingOrders)(c)
}

type purchaseRequest struct {
	ChildID string `json:"child_id"`
	ItemID  string `json:"item_id"`
}

// Purchase handles POST /v1/kids/shop-orders/purchase.
func (h *ShopHandler) Purchase(c *gin.Context) {
	req, ok := bindJSON[purchaseRequest](c)
	if !ok {
		return
	}
	order, err := h.svc.Purchase(c.Request.Context(), req.ChildID, req.ItemID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, order)
}

// Fulfill handles POST /v1/kids/shop-orders/:id/fulfill.
func (h *ShopHandler) Fulfill(c *gin.Context) {
	order, err := h.svc.MarkFulfilled(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}
