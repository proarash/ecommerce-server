package wallet

import (
	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/export"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

var transactionTypeLabels = map[string]string{
	TxCredit: "واریز",
	TxDebit:  "برداشت",
}

var transactionSourceLabels = map[string]string{
	SourceGateway:  "درگاه پرداخت",
	SourcePurchase: "خرید",
	SourceManual:   "دستی",
	SourceRefund:   "بازگشت وجه",
}

func label(labels map[string]string, key string) string {
	if v, ok := labels[key]; ok {
		return v
	}
	return key
}

func (h *Handler) exportTransactions(c *gin.Context, walletID uint, prefix string) {
	q := TransactionQuery{Page: 1, Limit: export.MaxRows}
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	q.Page, q.Limit = 1, export.MaxRows
	items, _, err := h.store.ListTransactions(c.Request.Context(), walletID, q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, t := range items {
		var order any = ""
		if t.OrderID != nil {
			order = *t.OrderID
		}
		var track any = ""
		if t.TrackID != nil {
			track = *t.TrackID
		}
		rows = append(rows, []any{t.ID, label(transactionTypeLabels, t.Type), label(transactionSourceLabels, t.Source), t.Amount, t.BalanceBefore, t.BalanceAfter, order, track, t.Description, export.FormatTime(t.CreatedAt)})
	}
	export.Send(c, export.Filename(prefix), export.Sheet{
		Title:     "تراکنش‌های کیف پول",
		Headers:   []string{"شناسه", "نوع", "منبع", "مبلغ", "موجودی قبل", "موجودی بعد", "شماره سفارش", "کد رهگیری", "توضیحات", "تاریخ"},
		Rows:      rows,
		MoneyCols: []int{3, 4, 5},
	})
}

// ExportMyTransactions godoc
// @Summary Export my wallet transactions as excel
// @Tags Wallet
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param type query string false "Type" Enums(credit, debit)
// @Param source query string false "Source" Enums(gateway, purchase, manual)
// @Success 200 {file} file
// @Security BearerAuth
// @Router /wallet/transactions/export [get]
func (h *Handler) ExportMyTransactions(c *gin.Context) {
	w, err := h.store.FindByOwner(c.Request.Context(), middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "wallet not found")
		return
	}
	h.exportTransactions(c, w.ID, "wallet-transactions")
}

// ExportWalletTransactions godoc
// @Summary Export wallet transactions as excel
// @Tags Wallet Management
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path int true "Wallet ID"
// @Param type query string false "Type" Enums(credit, debit)
// @Param source query string false "Source" Enums(gateway, purchase, manual)
// @Success 200 {file} file
// @Security BearerAuth
// @Router /wallets/{id}/transactions/export [get]
func (h *Handler) ExportWalletTransactions(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	h.exportTransactions(c, id, "wallet-transactions")
}

// ExportWallets godoc
// @Summary Export wallets as excel
// @Tags Wallet Management
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param owner_id query int false "Owner ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /wallets/export [get]
func (h *Handler) ExportWallets(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	q.Page, q.Limit = 1, export.MaxRows
	items, _, err := h.store.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, w := range items {
		state := "آزاد"
		if w.Locked {
			state = "قفل"
		}
		rows = append(rows, []any{w.ID, w.OwnerID, w.Amount, state, export.FormatTime(w.CreatedAt)})
	}
	export.Send(c, export.Filename("wallets"), export.Sheet{
		Title:     "کیف پول کاربران",
		Headers:   []string{"شناسه کیف پول", "شناسه کاربر", "موجودی", "وضعیت", "تاریخ ایجاد"},
		Rows:      rows,
		MoneyCols: []int{2},
	})
}
