package finance

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/gorm"
)

func (h *Handler) refund(reviewerID uint) func(tx *gorm.DB, order Order) error {
	return func(tx *gorm.DB, order Order) error {
		if order.TotalAmount <= 0 {
			return nil
		}
		entry := wallet.Entry{
			Type:          wallet.TxCredit,
			Source:        wallet.SourceRefund,
			Amount:        order.TotalAmount,
			OrderID:       &order.ID,
			PerformedByID: &reviewerID,
			Description:   fmt.Sprintf("Refund of rejected order #%d", order.ID),
		}
		_, _, err := h.wallets.ApplyTx(tx, order.UserID, entry)
		return err
	}
}

func (h *Handler) review(c *gin.Context, accept bool, message string) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	reviewerID := middleware.GetAuth(c).UserID
	order, err := h.store.ReviewOrder(c.Request.Context(), id, reviewerID, accept, message, h.refund(reviewerID))
	if errors.Is(err, ErrOrderNotReviewable) {
		types.BadRequest(c, err)
		return
	}
	if err != nil {
		wallet.Fail(c, err, "order not found")
		return
	}
	h.store.NotifyOrderReviewed(c.Request.Context(), order)
	c.JSON(http.StatusOK, order)
}

// ConfirmOrder godoc
// @Summary Confirm a paid order
// @Description Support only. Moves a paid order to confirmed and notifies the customer. Only orders in the paid status can be reviewed.
// @Tags Support
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} types.ApiResponse{data=Order}
// @Security BearerAuth
// @Router /support/orders/{id}/confirm [post]
func (h *Handler) ConfirmOrder(c *gin.Context) {
	h.review(c, true, "")
}

// RejectOrder godoc
// @Summary Reject a paid order
// @Description Support only. Moves a paid order to rejected, stores the message, refunds the paid amount to the customer wallet (a wallet transaction with source refund) and notifies the customer with the message. Only orders in the paid status can be reviewed.
// @Tags Support
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Param body body RejectOrderDto true "Rejection message"
// @Success 200 {object} types.ApiResponse{data=Order}
// @Security BearerAuth
// @Router /support/orders/{id}/reject [post]
func (h *Handler) RejectOrder(c *gin.Context) {
	var dto RejectOrderDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.review(c, false, dto.Message)
}
