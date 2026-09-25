package inventory

import (
	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
)

const (
	LogInbound  = "inbound"
	LogOutbound = "outbound"
)

type InventoryStock struct {
	gorm.Model
	ProductID         uint             `json:"product_id" gorm:"uniqueIndex;not null"`
	Product           *product.Product `json:"product,omitempty"`
	Quantity          int              `json:"quantity" gorm:"default:0"`
	WarehouseLocation *string          `json:"warehouse_location"`
}

type InventoryLog struct {
	gorm.Model
	ProductID uint   `json:"product_id" gorm:"index;not null"`
	Type      string `json:"type" enums:"inbound,outbound"`
	Quantity  int    `json:"quantity"`
	Reason    string `json:"reason"`
	CreatedBy uint   `json:"created_by"`
}
