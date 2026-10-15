package inventory

import (
	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/export"
	"github.com/proarash/ecommerce-server/internal/types"
)

// Export godoc
// @Summary Export current stock levels as excel
// @Tags Inventory
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} file
// @Security BearerAuth
// @Router /inventory/export [get]
func (h *Handler) Export(c *gin.Context) {
	items, _, err := h.store.List(c.Request.Context(), 0, export.MaxRows)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, s := range items {
		title, sku := "", ""
		if s.Product != nil {
			title, sku = s.Product.Title, s.Product.SKU
		}
		location := ""
		if s.WarehouseLocation != nil {
			location = *s.WarehouseLocation
		}
		rows = append(rows, []any{s.ProductID, title, sku, s.Quantity, location})
	}
	export.Send(c, export.Filename("inventory"), export.Sheet{
		Title:   "موجودی انبار",
		Headers: []string{"شناسه محصول", "نام محصول", "SKU", "موجودی", "محل انبار"},
		Rows:    rows,
	})
}
