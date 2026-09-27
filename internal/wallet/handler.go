package wallet

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(protected gin.IRouter, accountant gin.IRouter) {
	protected.GET("/wallet", h.Mine)
	protected.GET("/wallet/transactions", h.MyTransactions)

	g := accountant.Group("/wallets")
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/transactions", h.Transactions)
	g.POST("/:id/adjust", h.Adjust)
}

func OwnerOf(p *token.AuthPayload) (string, uint) {
	if p.UserType == token.UserTypeStaff {
		return OwnerStaff, p.UserID
	}
	return OwnerUser, p.UserID
}

func Fail(c *gin.Context, err error, notFound string) {
	switch {
	case errors.Is(err, ErrLocked):
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: err.Error()})
	case errors.Is(err, ErrInsufficientBalance), errors.Is(err, ErrInvalidAmount):
		types.BadRequest(c, err)
	default:
		types.HandleError(c, err, notFound)
	}
}

func (h *Handler) listTransactions(c *gin.Context, walletID uint) {
	var q TransactionQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.ListTransactions(c.Request.Context(), walletID, q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Mine godoc
// @Summary Get my wallet
// @Description Returns the wallet of the authenticated user or staff member
// @Tags Wallet
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Success 200 {object} types.ApiResponse{data=Wallet}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallet [get]
func (h *Handler) Mine(c *gin.Context) {
	ownerType, ownerID := OwnerOf(middleware.GetAuth(c))
	w, err := h.store.FindByOwner(c.Request.Context(), ownerType, ownerID)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	c.JSON(http.StatusOK, w)
}

// MyTransactions godoc
// @Summary List my wallet transactions
// @Tags Wallet
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param type query string false "Type" Enums(credit, debit)
// @Param source query string false "Source" Enums(gateway, purchase, manual)
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]WalletTransaction}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallet/transactions [get]
func (h *Handler) MyTransactions(c *gin.Context) {
	ownerType, ownerID := OwnerOf(middleware.GetAuth(c))
	w, err := h.store.FindByOwner(c.Request.Context(), ownerType, ownerID)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	h.listTransactions(c, w.ID)
}

// List godoc
// @Summary List wallets
// @Tags Wallet Management
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param owner_type query string false "Owner type" Enums(user, staff)
// @Param owner_id query int false "Owner ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Wallet}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallets [get]
func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Get godoc
// @Summary Get wallet
// @Tags Wallet Management
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Wallet ID"
// @Success 200 {object} types.ApiResponse{data=Wallet}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallets/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	w, err := h.store.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	c.JSON(http.StatusOK, w)
}

// Transactions godoc
// @Summary List wallet transactions
// @Tags Wallet Management
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Wallet ID"
// @Param type query string false "Type" Enums(credit, debit)
// @Param source query string false "Source" Enums(gateway, purchase, manual)
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]WalletTransaction}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallets/{id}/transactions [get]
func (h *Handler) Transactions(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	h.listTransactions(c, id)
}

// Adjust godoc
// @Summary Change wallet amount
// @Description Credits or debits a wallet (admin and accountant only). Admin wallets are locked and cannot be modified by anyone, and staff cannot change their own wallet.
// @Tags Wallet Management
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Wallet ID"
// @Param body body AdjustDto true "Adjustment"
// @Success 200 {object} types.ApiResponse{data=AdjustResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /wallets/{id}/adjust [post]
func (h *Handler) Adjust(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto AdjustDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	p := middleware.GetAuth(c)
	ctx := c.Request.Context()
	target, err := h.store.FindByID(ctx, id)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	if target.OwnerType == OwnerStaff && target.OwnerID == p.UserID {
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "cannot change your own wallet"})
		return
	}
	w, t, err := h.store.Adjust(ctx, id, Entry{Type: dto.Type, Source: SourceManual, Amount: dto.Amount, PerformedByID: &p.UserID, Description: dto.Description})
	if err != nil {
		Fail(c, err, "wallet not found")
		return
	}
	c.JSON(http.StatusOK, AdjustResponse{Wallet: w, Transaction: t})
}
