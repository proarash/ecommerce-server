package finance

import "time"

type OrderLine struct {
	ProductID uint
	Quantity  int
	UnitPrice float64
}

type OrderDiscount struct {
	ID     uint
	Amount float64
}

type CreatePreInvoiceDto struct {
	OrderID uint     `json:"order_id" binding:"required,min=1"`
	Amount  *float64 `json:"amount" binding:"omitempty,gt=0"`
}

type RejectOrderDto struct {
	Message string `json:"message" binding:"required,max=500" example:"Product is out of stock"`
}

type ReportQuery struct {
	Period string     `form:"period,default=daily" binding:"oneof=daily weekly monthly annually"`
	From   *time.Time `form:"from" time_format:"2006-01-02"`
	To     *time.Time `form:"to" time_format:"2006-01-02"`
}

type ReportRow struct {
	PeriodStart time.Time `json:"period_start"`
	OrdersCount int64     `json:"orders_count"`
	TotalAmount float64   `json:"total_amount"`
}

type ProductSalesRow struct {
	PeriodStart  time.Time `json:"period_start"`
	ProductID    uint      `json:"product_id"`
	ProductTitle string    `json:"product_title"`
	ProductSKU   string    `json:"product_sku"`
	Quantity     int64     `json:"quantity"`
	OrdersCount  int64     `json:"orders_count"`
	TotalAmount  float64   `json:"total_amount"`
}

type ReportResponse struct {
	Period      string            `json:"period" enums:"daily,weekly,monthly,annually"`
	Rows        []ReportRow       `json:"rows"`
	ProductRows []ProductSalesRow `json:"product_rows"`
	OrdersCount int64             `json:"orders_count"`
	TotalAmount float64           `json:"total_amount"`
}

type ListQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Status string `form:"status"`
	UserID uint   `form:"user_id"`
	Search string `form:"search"`
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}
