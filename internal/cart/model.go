package cart

import (
	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
)

type Cart struct {
	gorm.Model
	UserID uint       `json:"user_id" gorm:"uniqueIndex;not null"`
	Items  []CartItem `json:"items"`
}

type CartItem struct {
	gorm.Model
	CartID    uint             `json:"cart_id" gorm:"index;not null"`
	ProductID uint             `json:"product_id" gorm:"index;not null"`
	Product   *product.Product `json:"product,omitempty"`
	Quantity  int              `json:"quantity" gorm:"default:1"`
}
