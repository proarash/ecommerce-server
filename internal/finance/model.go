package finance

import (
	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
)

const (
	OrderPending    = "pending"
	OrderPaid       = "paid"
	OrderFailed     = "failed"
	OrderCancelled  = "cancelled"
	OrderProcessing = "processing"
	OrderDelivered  = "delivered"

	PreInvoiceIssued    = "issued"
	PreInvoicePaid      = "paid"
	PreInvoiceCancelled = "cancelled"

	IssuedBySystem = "system"
)

type Order struct {
	gorm.Model
	UserID      uint        `json:"user_id" gorm:"index;not null"`
	Status      string      `json:"status" gorm:"index;default:pending" enums:"pending,paid,failed,cancelled,processing,delivered"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	gorm.Model
	OrderID   uint             `json:"order_id" gorm:"index;not null"`
	ProductID uint             `json:"product_id" gorm:"index;not null"`
	Product   *product.Product `json:"product,omitempty"`
	Quantity  int              `json:"quantity"`
	UnitPrice float64          `json:"unit_price"`
}

type PreInvoice struct {
	gorm.Model
	OrderID       uint    `json:"order_id" gorm:"index;not null"`
	UserID        uint    `json:"user_id" gorm:"index;not null"`
	InvoiceNumber string  `json:"invoice_number" gorm:"uniqueIndex;not null"`
	Amount        float64 `json:"amount"`
	IssuedBy      string  `json:"issued_by"`
	Status        string  `json:"status" gorm:"default:issued" enums:"issued,paid,cancelled"`
}
