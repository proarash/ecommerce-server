package finance

import (
	"time"

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
	OrderConfirmed  = "confirmed"
	OrderRejected   = "rejected"

	PreInvoiceIssued    = "issued"
	PreInvoicePaid      = "paid"
	PreInvoiceCancelled = "cancelled"

	IssuedBySystem = "system"
)

type UserBrief struct {
	ID     uint    `json:"id"`
	Name   *string `json:"name"`
	Mobile string  `json:"mobile"`
}

type Order struct {
	gorm.Model
	UserID         uint        `json:"user_id" gorm:"index;not null"`
	Status         string      `json:"status" gorm:"index;default:pending" enums:"pending,paid,failed,cancelled,processing,delivered,confirmed,rejected"`
	TotalAmount    float64     `json:"total_amount"`
	DiscountID     *uint       `json:"discount_id" gorm:"index"`
	DiscountAmount float64     `json:"discount_amount" gorm:"not null;default:0"`
	RejectReason   string      `json:"reject_reason" gorm:"not null;default:''"`
	ReviewedByID   *uint       `json:"reviewed_by_id"`
	ReviewedAt     *time.Time  `json:"reviewed_at"`
	User           *UserBrief  `json:"user,omitempty" gorm:"-"`
	Items          []OrderItem `json:"items,omitempty"`
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
	OrderID       uint       `json:"order_id" gorm:"index;not null"`
	UserID        uint       `json:"user_id" gorm:"index;not null"`
	InvoiceNumber string     `json:"invoice_number" gorm:"uniqueIndex;not null"`
	Amount        float64    `json:"amount"`
	IssuedBy      string     `json:"issued_by"`
	Status        string     `json:"status" gorm:"default:issued" enums:"issued,paid,cancelled"`
	User          *UserBrief `json:"user,omitempty" gorm:"-"`
}
