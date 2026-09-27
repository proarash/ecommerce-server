package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/notification"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/gorm"
)

type Handler struct {
	store     Store
	client    *Client
	finance   finance.Store
	wallets   wallet.Store
	notifier  *notification.Service
	clientURL string
}

func NewHandler(store Store, client *Client, financeStore finance.Store, walletStore wallet.Store, notifier *notification.Service, clientURL string) *Handler {
	return &Handler{store: store, client: client, finance: financeStore, wallets: walletStore, notifier: notifier, clientURL: clientURL}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, customer gin.IRouter, accountant gin.IRouter) {
	public.GET("/payment/callback", h.Callback)
	customer.POST("/payment/checkout/:orderId", h.Checkout)
	customer.POST("/payment/checkout/:orderId/wallet", h.PayWithWallet)
	customer.POST("/payment/wallet/charge", h.ChargeWallet)
	customer.GET("/payment/status/:trackId", h.Status)
	accountant.POST("/payment/inquiry/:trackId", h.Inquiry)
}

func trackParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("trackId"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid trackId"})
		return 0, false
	}
	return id, true
}

// Checkout godoc
// @Summary Start Zibal payment for an order
// @Description Requests a Zibal trackId for a pending/failed order and returns the gateway redirect URL. On success the paid amount is credited to the customer wallet and the order price is then debited from it.
// @Tags Payment
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param orderId path int true "Order ID"
// @Success 200 {object} types.ApiResponse{data=CheckoutResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 502 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /payment/checkout/{orderId} [post]
func (h *Handler) Checkout(c *gin.Context) {
	orderID, ok := types.ParamID(c, "orderId")
	if !ok {
		return
	}
	p := middleware.GetAuth(c)
	order, err := h.finance.FindUserOrder(c.Request.Context(), orderID, p.UserID)
	if err != nil {
		types.HandleError(c, err, "order not found")
		return
	}
	if order.Status != finance.OrderPending && order.Status != finance.OrderFailed {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "order is not payable in status " + order.Status})
		return
	}
	amount := int64(order.TotalAmount)
	description := fmt.Sprintf("Order #%d", order.ID)
	res, err := h.client.Request(c.Request.Context(), amount, strconv.FormatUint(uint64(order.ID), 10), description, p.Mobile)
	if err != nil {
		c.JSON(http.StatusBadGateway, types.ErrorResponse{Error: err.Error()})
		return
	}
	tx := PaymentTransaction{Purpose: PurposeOrder, OrderID: &order.ID, UserID: p.UserID, Amount: amount, TrackID: res.TrackID, Status: StatusPending, Result: res.Result, Description: description}
	if err := h.store.Create(c.Request.Context(), &tx); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, CheckoutResponse{TrackID: res.TrackID, PaymentURL: StartURL(res.TrackID)})
}

// ChargeWallet godoc
// @Summary Charge wallet via Zibal
// @Description Requests a Zibal trackId to top up the customer wallet. After a verified payment the wallet is credited automatically and the callback redirects to the client app with ?wallet=wlt-<TRACK-ID>
// @Tags Payment
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body WalletChargeDto true "Charge amount"
// @Success 200 {object} types.ApiResponse{data=CheckoutResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 502 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /payment/wallet/charge [post]
func (h *Handler) ChargeWallet(c *gin.Context) {
	var dto WalletChargeDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	p := middleware.GetAuth(c)
	ctx := c.Request.Context()
	description := "Wallet charge"
	res, err := h.client.Request(ctx, dto.Amount, fmt.Sprintf("wallet-%d-%d", p.UserID, time.Now().UnixNano()), description, p.Mobile)
	if err != nil {
		c.JSON(http.StatusBadGateway, types.ErrorResponse{Error: err.Error()})
		return
	}
	tx := PaymentTransaction{Purpose: PurposeWalletCharge, UserID: p.UserID, Amount: dto.Amount, TrackID: res.TrackID, Status: StatusPending, Result: res.Result, Description: description}
	if err := h.store.Create(ctx, &tx); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, CheckoutResponse{TrackID: res.TrackID, PaymentURL: StartURL(res.TrackID)})
}

// PayWithWallet godoc
// @Summary Pay order from wallet
// @Description Debits the order total from the customer wallet balance and marks the order as paid
// @Tags Payment
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param orderId path int true "Order ID"
// @Success 200 {object} types.ApiResponse{data=WalletPayResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /payment/checkout/{orderId}/wallet [post]
func (h *Handler) PayWithWallet(c *gin.Context) {
	orderID, ok := types.ParamID(c, "orderId")
	if !ok {
		return
	}
	p := middleware.GetAuth(c)
	ctx := c.Request.Context()
	err := h.wallets.Transaction(ctx, func(db *gorm.DB) error {
		order, err := h.finance.LockPayableOrderTx(db, orderID, p.UserID)
		if err != nil {
			return err
		}
		if order.TotalAmount > 0 {
			entry := wallet.Entry{Type: wallet.TxDebit, Source: wallet.SourcePurchase, Amount: order.TotalAmount, OrderID: &order.ID, Description: fmt.Sprintf("Order #%d", order.ID)}
			if _, _, err := h.wallets.ApplyTx(db, wallet.OwnerUser, p.UserID, entry); err != nil {
				return err
			}
		}
		return h.finance.MarkPaidTx(db, order.ID)
	})
	if errors.Is(err, finance.ErrOrderNotPayable) {
		types.BadRequest(c, err)
		return
	}
	if err != nil {
		wallet.Fail(c, err, "order not found")
		return
	}
	order, err := h.finance.FindUserOrder(ctx, orderID, p.UserID)
	if err != nil {
		types.HandleError(c, err, "order not found")
		return
	}
	w, err := h.wallets.FindByOwner(ctx, wallet.OwnerUser, p.UserID)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	go h.notify(PaymentTransaction{Purpose: PurposeOrder, OrderID: &order.ID, UserID: p.UserID}, true)
	c.JSON(http.StatusOK, WalletPayResponse{Order: order, Wallet: w})
}

func (h *Handler) redirect(c *gin.Context, tx PaymentTransaction) {
	u, err := url.Parse(h.clientURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "invalid client redirect url"})
		return
	}
	q := u.Query()
	if tx.OrderID != nil {
		q.Set("inv", fmt.Sprintf("inv-%d", *tx.OrderID))
	} else {
		q.Set("wallet", fmt.Sprintf("wlt-%d", tx.TrackID))
	}
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}

func (h *Handler) notify(tx PaymentTransaction, success bool) {
	p := notification.AutomatedMessagePayload{
		UserID:    tx.UserID,
		ExtraData: map[string]interface{}{"purpose": tx.Purpose, "track_id": tx.TrackID},
	}
	subject := fmt.Sprintf("wallet charge of %d", tx.Amount)
	if tx.OrderID != nil {
		subject = fmt.Sprintf("order #%d", *tx.OrderID)
		p.ExtraData["order_id"] = *tx.OrderID
	}
	if success {
		p.Title, p.Type = "Payment succeeded", notification.MsgPaymentSuccess
		p.Message = fmt.Sprintf("Payment for %s was successful. Track ID: %d", subject, tx.TrackID)
	} else {
		p.Title, p.Type = "Payment failed", notification.MsgPaymentFailed
		p.Message = fmt.Sprintf("Payment for %s failed. Track ID: %d", subject, tx.TrackID)
	}
	if err := h.notifier.SendAutomatedMessage(context.Background(), p); err != nil {
		log.Println("payment: notify:", err)
	}
}

func (h *Handler) settle(ctx context.Context, tx PaymentTransaction) error {
	return h.wallets.Transaction(ctx, func(db *gorm.DB) error {
		amount := float64(tx.Amount)
		credit := wallet.Entry{Type: wallet.TxCredit, Source: wallet.SourceGateway, Amount: amount, OrderID: tx.OrderID, TrackID: &tx.TrackID, Description: tx.Description}
		if _, _, err := h.wallets.ApplyTx(db, wallet.OwnerUser, tx.UserID, credit); err != nil {
			return err
		}
		if tx.OrderID == nil {
			return nil
		}
		debit := wallet.Entry{Type: wallet.TxDebit, Source: wallet.SourcePurchase, Amount: amount, OrderID: tx.OrderID, TrackID: &tx.TrackID, Description: tx.Description}
		if _, _, err := h.wallets.ApplyTx(db, wallet.OwnerUser, tx.UserID, debit); err != nil {
			return err
		}
		return h.finance.MarkPaidTx(db, *tx.OrderID)
	})
}

// Callback godoc
// @Summary Zibal payment callback
// @Description Called by the Zibal gateway. Verifies the payment and credits the paid amount to the customer wallet. For orders the order price is then debited from the wallet and order/transaction/pre-invoice are updated. Sends bot messages and redirects (302) to the client app with ?inv=inv-<ORDER-ID> for orders or ?wallet=wlt-<TRACK-ID> for wallet charges
// @Tags Payment
// @Param trackId query int true "Zibal track ID"
// @Param success query int false "1 when payment succeeded"
// @Param status query int false "Zibal status"
// @Param orderId query string false "Merchant order ID"
// @Success 302 "Redirect to client payment result page"
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /payment/callback [get]
func (h *Handler) Callback(c *gin.Context) {
	var q CallbackQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	tx, err := h.store.FindByTrackID(ctx, q.TrackID)
	if err != nil {
		types.HandleError(c, err, "transaction not found")
		return
	}
	if tx.Status != StatusPending {
		h.redirect(c, tx)
		return
	}

	success := false
	if q.Success == 1 {
		v, err := h.client.Verify(ctx, q.TrackID)
		if err != nil {
			log.Println("payment: verify:", err)
		} else {
			tx.Result = v.Result
			tx.Status = v.Status
			tx.RefNumber = v.RefNumber
			if v.CardNumber != "" {
				tx.CardNumber = &v.CardNumber
			}
			if t, err := time.Parse(time.RFC3339Nano, v.PaidAt); err == nil {
				tx.PaidAt = &t
			} else if v.PaidAt != "" {
				now := time.Now()
				tx.PaidAt = &now
			}
			success = (v.Result == ResultSuccess || v.Result == ResultAlreadyVerified) && v.Amount == tx.Amount
		}
	}
	if !success && tx.Status == StatusPending {
		tx.Status = q.Status
	}
	settled, err := h.store.Settle(ctx, &tx)
	if err != nil {
		log.Println("payment: settle:", err)
	}
	if !settled {
		h.redirect(c, tx)
		return
	}

	if success {
		if err := h.settle(ctx, tx); err != nil {
			log.Printf("payment: settle track %d: %v", tx.TrackID, err)
			if tx.OrderID != nil {
				if err := h.finance.MarkPaid(ctx, *tx.OrderID); err != nil {
					log.Println("payment: mark paid:", err)
				}
			}
		}
	} else if tx.OrderID != nil {
		if err := h.finance.MarkFailed(ctx, *tx.OrderID); err != nil {
			log.Println("payment: mark failed:", err)
		}
	}
	go h.notify(tx, success)

	h.redirect(c, tx)
}

// Status godoc
// @Summary Payment transaction status
// @Tags Payment
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param trackId path int true "Zibal track ID"
// @Success 200 {object} types.ApiResponse{data=PaymentTransaction}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /payment/status/{trackId} [get]
func (h *Handler) Status(c *gin.Context) {
	trackID, ok := trackParam(c)
	if !ok {
		return
	}
	tx, err := h.store.FindUserTransaction(c.Request.Context(), trackID, middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "transaction not found")
		return
	}
	c.JSON(http.StatusOK, tx)
}

// Inquiry godoc
// @Summary Manual gateway transaction inquiry
// @Description Queries Zibal /v1/inquiry for the track ID and returns it with the local transaction record
// @Tags Payment
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param trackId path int true "Zibal track ID"
// @Success 200 {object} types.ApiResponse{data=InquiryResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 502 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /payment/inquiry/{trackId} [post]
func (h *Handler) Inquiry(c *gin.Context) {
	trackID, ok := trackParam(c)
	if !ok {
		return
	}
	tx, err := h.store.FindByTrackID(c.Request.Context(), trackID)
	if err != nil {
		types.HandleError(c, err, "transaction not found")
		return
	}
	res, err := h.client.Inquiry(c.Request.Context(), trackID)
	if err != nil {
		c.JSON(http.StatusBadGateway, types.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, InquiryResponse{Transaction: tx, Gateway: res})
}
