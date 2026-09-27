package payment

import (
	"context"
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
)

type Handler struct {
	store     Store
	client    *Client
	finance   finance.Store
	notifier  *notification.Service
	clientURL string
}

func NewHandler(store Store, client *Client, financeStore finance.Store, notifier *notification.Service, clientURL string) *Handler {
	return &Handler{store: store, client: client, finance: financeStore, notifier: notifier, clientURL: clientURL}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, customer gin.IRouter, accountant gin.IRouter) {
	public.GET("/payment/callback", h.Callback)
	customer.POST("/payment/checkout/:orderId", h.Checkout)
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
// @Description Requests a Zibal trackId for a pending/failed order and returns the gateway redirect URL
// @Tags Payment
// @Produce json
// @Security BearerAuth
// @Param orderId path int true "Order ID"
// @Success 200 {object} types.ApiResponse{data=CheckoutResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 502 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/payment/checkout/{orderId} [post]
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
	tx := PaymentTransaction{OrderID: order.ID, UserID: p.UserID, Amount: amount, TrackID: res.TrackID, Status: StatusPending, Result: res.Result, Description: description}
	if err := h.store.Create(c.Request.Context(), &tx); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, CheckoutResponse{TrackID: res.TrackID, PaymentURL: StartURL(res.TrackID)})
}

func (h *Handler) redirect(c *gin.Context, orderID uint) {
	u, err := url.Parse(h.clientURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "invalid client redirect url"})
		return
	}
	q := u.Query()
	q.Set("inv", fmt.Sprintf("inv-%d", orderID))
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}

func (h *Handler) notify(order uint, userID uint, trackID int64, success bool) {
	p := notification.AutomatedMessagePayload{
		UserID:    userID,
		ExtraData: map[string]interface{}{"order_id": order, "track_id": trackID},
	}
	if success {
		p.Title, p.Type = "Payment succeeded", notification.MsgPaymentSuccess
		p.Message = fmt.Sprintf("Payment for order #%d was successful. Track ID: %d", order, trackID)
	} else {
		p.Title, p.Type = "Payment failed", notification.MsgPaymentFailed
		p.Message = fmt.Sprintf("Payment for order #%d failed. Track ID: %d", order, trackID)
	}
	if err := h.notifier.SendAutomatedMessage(context.Background(), p); err != nil {
		log.Println("payment: notify:", err)
	}
}

// Callback godoc
// @Summary Zibal payment callback
// @Description Called by the Zibal gateway. Verifies the payment, updates order/transaction/pre-invoice, sends bot messages and redirects (302) to the client app with ?inv=inv-<ORDER-ID>
// @Tags Payment
// @Param trackId query int true "Zibal track ID"
// @Param success query int false "1 when payment succeeded"
// @Param status query int false "Zibal status"
// @Param orderId query string false "Merchant order ID"
// @Success 302 "Redirect to client payment result page"
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /api/payment/callback [get]
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
		h.redirect(c, tx.OrderID)
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
		h.redirect(c, tx.OrderID)
		return
	}

	if success {
		if err := h.finance.MarkPaid(ctx, tx.OrderID); err != nil {
			log.Println("payment: mark paid:", err)
		}
	} else if err := h.finance.MarkFailed(ctx, tx.OrderID); err != nil {
		log.Println("payment: mark failed:", err)
	}
	go h.notify(tx.OrderID, tx.UserID, tx.TrackID, success)

	h.redirect(c, tx.OrderID)
}

// Status godoc
// @Summary Payment transaction status
// @Tags Payment
// @Produce json
// @Security BearerAuth
// @Param trackId path int true "Zibal track ID"
// @Success 200 {object} types.ApiResponse{data=PaymentTransaction}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/payment/status/{trackId} [get]
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
// @Security BearerAuth
// @Param trackId path int true "Zibal track ID"
// @Success 200 {object} types.ApiResponse{data=InquiryResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 502 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/payment/inquiry/{trackId} [post]
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
