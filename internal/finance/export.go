package finance

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/export"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

var orderStatusLabels = map[string]string{
	OrderPending:    "در انتظار پرداخت",
	OrderPaid:       "پرداخت شده",
	OrderFailed:     "ناموفق",
	OrderCancelled:  "لغو شده",
	OrderProcessing: "در حال پردازش",
	OrderDelivered:  "تحویل شده",
	OrderConfirmed:  "پرداخت شده/تایید شده",
	OrderRejected:   "پرداخت شده/رد شده",
}

var preInvoiceStatusLabels = map[string]string{
	PreInvoiceIssued:    "صادر شده",
	PreInvoicePaid:      "پرداخت شده",
	PreInvoiceCancelled: "لغو شده",
}

func label(labels map[string]string, key string) string {
	if v, ok := labels[key]; ok {
		return v
	}
	return key
}

func exportQuery(c *gin.Context) (ListQuery, bool) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return q, false
	}
	q.Page, q.Limit = 1, export.MaxRows
	return q, true
}

func (h *Handler) exportOrders(c *gin.Context, q ListQuery, prefix string) {
	items, _, err := h.store.ListOrders(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, o := range items {
		user := fmt.Sprint(o.UserID)
		if o.User != nil {
			name := "-"
			if o.User.Name != nil && *o.User.Name != "" {
				name = *o.User.Name
			}
			user = name + "-" + o.User.Mobile
		}
		lines := make([]string, 0, len(o.Items))
		for _, it := range o.Items {
			title := fmt.Sprintf("#%d", it.ProductID)
			if it.Product != nil {
				title = it.Product.Title
			}
			lines = append(lines, fmt.Sprintf("%s x%d", title, it.Quantity))
		}
		rows = append(rows, []any{o.ID, user, strings.Join(lines, "\n"), label(orderStatusLabels, o.Status), o.DiscountAmount, o.TotalAmount, export.FormatJalali(o.CreatedAt)})
	}
	export.Send(c, export.Filename(prefix), export.Sheet{
		Title:     "سفارش‌ها",
		Headers:   []string{"کد سفارش", "کاربر", "محصولات", "وضعیت", "تخفیف", "مبلغ", "تاریخ"},
		Rows:      rows,
		MoneyCols: []int{4, 5},
	})
}

func (h *Handler) exportPreInvoices(c *gin.Context, q ListQuery, prefix string) {
	items, _, err := h.store.ListPreInvoices(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, p := range items {
		rows = append(rows, []any{p.InvoiceNumber, p.OrderID, p.UserID, label(preInvoiceStatusLabels, p.Status), p.Amount, p.IssuedBy, export.FormatTime(p.CreatedAt)})
	}
	export.Send(c, export.Filename(prefix), export.Sheet{
		Title:     "پیش‌فاکتورها",
		Headers:   []string{"شماره پیش‌فاکتور", "شماره سفارش", "شناسه کاربر", "وضعیت", "مبلغ", "صادرکننده", "تاریخ"},
		Rows:      rows,
		MoneyCols: []int{4},
	})
}

// ExportReports godoc
// @Summary Export sales report as excel
// @Description Same data and filters as GET /finance/reports, as an .xlsx file with a per-product sheet and a per-period sheet
// @Tags Finance
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param period query string false "Bucket size" Enums(daily, weekly, monthly, annually) default(daily)
// @Param from query string false "From date (YYYY-MM-DD)"
// @Param to query string false "To date inclusive (YYYY-MM-DD)"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /finance/reports/export [get]
func (h *Handler) ExportReports(c *gin.Context) {
	var q ReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.store.Report(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	productRows := make([][]any, 0, len(res.ProductRows))
	for _, r := range res.ProductRows {
		productRows = append(productRows, []any{export.FormatTime(r.PeriodStart), r.ProductTitle, r.ProductSKU, r.Quantity, r.OrdersCount, r.TotalAmount})
	}
	periodRows := make([][]any, 0, len(res.Rows))
	for _, r := range res.Rows {
		periodRows = append(periodRows, []any{export.FormatTime(r.PeriodStart), r.OrdersCount, r.TotalAmount})
	}
	export.Send(c, export.Filename("sales-report"),
		export.Sheet{
			Title:     "فروش محصولات",
			Headers:   []string{"شروع بازه", "نام محصول", "SKU", "تعداد فروش", "تعداد سفارش", "مبلغ فروش"},
			Rows:      productRows,
			MoneyCols: []int{5},
		},
		export.Sheet{
			Title:     "خلاصه بازه‌ها",
			Headers:   []string{"شروع بازه", "تعداد سفارش", "مبلغ فروش"},
			Rows:      periodRows,
			Notes:     [][]any{{"مجموع فروش", res.TotalAmount}, {"تعداد سفارش‌ها", res.OrdersCount}},
			MoneyCols: []int{2},
		},
	)
}

// ExportPreInvoices godoc
// @Summary Export pre-invoices as excel
// @Tags Finance
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param status query string false "Status"
// @Param user_id query int false "User ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /finance/preinvoices/export [get]
func (h *Handler) ExportPreInvoices(c *gin.Context) {
	if q, ok := exportQuery(c); ok {
		h.exportPreInvoices(c, q, "preinvoices")
	}
}

// ExportUserOrders godoc
// @Summary Export my orders as excel
// @Tags User
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param status query string false "Status"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /user/orders/export [get]
func (h *Handler) ExportUserOrders(c *gin.Context) {
	if q, ok := exportQuery(c); ok {
		q.UserID = middleware.GetAuth(c).UserID
		h.exportOrders(c, q, "my-orders")
	}
}

// ExportUserPreInvoices godoc
// @Summary Export my pre-invoices as excel
// @Tags User
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} file
// @Security BearerAuth
// @Router /user/preinvoices/export [get]
func (h *Handler) ExportUserPreInvoices(c *gin.Context) {
	if q, ok := exportQuery(c); ok {
		q.UserID = middleware.GetAuth(c).UserID
		h.exportPreInvoices(c, q, "my-preinvoices")
	}
}

// ExportSupportOrders godoc
// @Summary Export customer orders as excel
// @Tags Support
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param status query string false "Status"
// @Param user_id query int false "User ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /support/orders/export [get]
func (h *Handler) ExportSupportOrders(c *gin.Context) {
	if q, ok := exportQuery(c); ok {
		h.exportOrders(c, q, "orders")
	}
}

// ExportSupportPreInvoices godoc
// @Summary Export customer pre-invoices as excel
// @Tags Support
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param status query string false "Status"
// @Param user_id query int false "User ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /support/preinvoices/export [get]
func (h *Handler) ExportSupportPreInvoices(c *gin.Context) {
	if q, ok := exportQuery(c); ok {
		h.exportPreInvoices(c, q, "preinvoices")
	}
}

// ExportUserInvoice godoc
// @Summary Download an order invoice as excel
// @Description Invoice of one order of the logged in user with its items, discount and payable total. Available for every order status.
// @Tags User
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path int true "Order ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /user/orders/{id}/invoice [get]
func (h *Handler) ExportUserInvoice(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	o, err := h.store.FindUserOrder(c.Request.Context(), id, middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "order not found")
		return
	}
	rows := make([][]any, 0, len(o.Items))
	var subtotal float64
	for i, it := range o.Items {
		title, sku := "", ""
		if it.Product != nil {
			title, sku = it.Product.Title, it.Product.SKU
		}
		line := it.UnitPrice * float64(it.Quantity)
		subtotal += line
		rows = append(rows, []any{i + 1, title, sku, it.Quantity, it.UnitPrice, line})
	}
	footer := [][]any{
		{"", "", "", "", "جمع اقلام", subtotal},
		{"", "", "", "", "تخفیف", o.DiscountAmount},
		{"", "", "", "", "مبلغ قابل پرداخت", o.TotalAmount},
	}
	export.Send(c, export.Filename(fmt.Sprintf("invoice-%d", o.ID)), export.Sheet{
		Title:     "فاکتور",
		Headers:   []string{"ردیف", "محصول", "SKU", "تعداد", "قیمت واحد", "مبلغ"},
		Rows:      rows,
		Footer:    footer,
		MoneyCols: []int{4, 5},
		Notes: [][]any{
			{"شماره سفارش", o.ID},
			{"وضعیت", label(orderStatusLabels, o.Status)},
			{"تاریخ", export.FormatTime(o.CreatedAt)},
		},
	})
}
