package discount

import (
	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/export"
	"github.com/proarash/ecommerce-server/internal/types"
)

// Export godoc
// @Summary Export discount codes as excel
// @Tags Discount
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param code query string false "Code"
// @Param type query string false "Type" Enums(percent, value)
// @Param status query bool false "Status"
// @Param owner_id query int false "Owner ID"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /discounts/export [get]
func (h *Handler) Export(c *gin.Context) {
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
	for _, d := range items {
		kind := "مبلغی"
		if d.Type == TypePercent {
			kind = "درصدی"
		}
		var maxPrice any = ""
		if d.MaxPrice != nil {
			maxPrice = *d.MaxPrice
		}
		state := "فعال"
		if !d.Status {
			state = "غیرفعال"
		}
		owner := ""
		if d.Owner != nil {
			owner = d.Owner.Mobile
		}
		rows = append(rows, []any{d.Code, kind, d.Value, maxPrice, d.UseCount, d.UsedCount, state, owner, export.FormatTime(d.CreatedAt)})
	}
	export.Send(c, export.Filename("discounts"), export.Sheet{
		Title:     "کدهای تخفیف",
		Headers:   []string{"کد", "نوع", "مقدار", "سقف تخفیف", "تعداد مجاز استفاده", "تعداد استفاده شده", "وضعیت", "موبایل صاحب کد", "تاریخ ایجاد"},
		Rows:      rows,
		MoneyCols: []int{2, 3},
	})
}
